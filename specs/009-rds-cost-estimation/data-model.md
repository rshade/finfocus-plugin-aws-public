# Data Model: RDS Instance Cost Estimation

**Feature**: 009-rds-cost-estimation
**Date**: 2025-12-02

## Entity Definitions

### RDS Instance Price

Represents the hourly compute cost for a specific RDS instance type and database engine.

| Field | Type | Description |
|-------|------|-------------|
| Unit | string | Billing unit, always "Hrs" |
| HourlyRate | float64 | Cost per hour in USD |
| Currency | string | Currency code, always "USD" |

**Lookup Key**: `{instanceType}/{engine}` (e.g., `db.t3.medium/MySQL`)

**Cardinality**: ~500-1000 entries per region (varies by engine availability)

### RDS Storage Price

Represents the per-GB-month cost for a specific storage volume type.

| Field | Type | Description |
|-------|------|-------------|
| Unit | string | Billing unit, always "GB-Mo" |
| RatePerGBMonth | float64 | Cost per GB per month in USD |
| Currency | string | Currency code, always "USD" |

**Lookup Key**: `{volumeType}` (e.g., `gp2`, `gp3`, `io1`)

**Cardinality**: 4-5 entries per region (gp2, gp3, io1, io2, standard)

## Index Structures

### Client Struct Extensions

```go
type Client struct {
    // Existing fields
    region   string
    currency string
    once     sync.Once
    err      error
    ec2Index map[string]ec2Price
    ebsIndex map[string]ebsPrice

    // New RDS fields
    rdsInstanceIndex map[string]rdsInstancePrice  // key: "instanceType/engine"
    rdsStorageIndex  map[string]rdsStoragePrice   // key: "volumeType"
}
```

### Index Key Examples

**Instance Index**:

| Key | Value (HourlyRate) |
|-----|-------------------|
| `db.t3.micro/MySQL` | 0.017 |
| `db.t3.medium/MySQL` | 0.068 |
| `db.t3.medium/PostgreSQL` | 0.068 |
| `db.m5.large/MySQL` | 0.171 |
| `db.m5.large/Oracle` | 0.350 |

**Storage Index**:

| Key | Value (RatePerGBMonth) |
|-----|------------------------|
| `gp2` | 0.115 |
| `gp3` | 0.10 |
| `io1` | 0.125 |
| `standard` | 0.10 |

## Input/Output Mapping

### ResourceDescriptor (Input)

```go
ResourceDescriptor{
    Provider:     "aws",
    ResourceType: "rds",
    Sku:          "db.t3.medium",     // Instance type
    Region:       "us-east-1",
    Tags: map[string]string{
        "engine":       "mysql",      // Database engine (optional, default: mysql)
        "storage_type": "gp3",        // Volume type (optional, default: gp2)
        "storage_size": "100",        // GB (optional, default: 20)
    },
}
```

### GetProjectedCostResponse (Output)

```go
GetProjectedCostResponse{
    UnitPrice:     0.068,             // Hourly instance rate
    Currency:      "USD",
    CostPerMonth:  59.64,             // Instance (49.64) + Storage (10.00)
    BillingDetail: "RDS db.t3.medium MySQL, 730 hrs/month + 100GB gp3 storage",
}
```

## Validation Rules

### Instance Type

- Must start with `db.` prefix (e.g., `db.t3.medium`, `db.m5.large`)
- Unknown types return $0 with explanatory message

### Engine

- Valid values: `mysql`, `postgres`, `mariadb`, `oracle-se2`, `sqlserver-ex`
- Unknown engines default to `mysql` with note in billing_detail

### Storage Type

- Valid values: `gp2`, `gp3`, `io1`, `io2`, `standard` (magnetic)
- Unknown types default to `gp2` with note in billing_detail

### Storage Size

- Must be positive integer
- Invalid values (non-numeric, negative, zero) default to 20 GB
- No maximum enforced (AWS limits vary by instance type)

## Engine Normalization Map

```go
var engineNormalization = map[string]string{
    "mysql":        "MySQL",
    "postgres":     "PostgreSQL",
    "postgresql":   "PostgreSQL",  // Alias
    "mariadb":      "MariaDB",
    "oracle":       "Oracle",
    "oracle-se2":   "Oracle",      // Standard Edition 2 (license-included)
    "sqlserver":    "SQL Server",
    "sqlserver-ex": "SQL Server",  // Express Edition
    "sql-server":   "SQL Server",  // Alias
}
```

## State Transitions

N/A - Pricing data is immutable at runtime (embedded at build time).

## Relationships

```text
ResourceDescriptor
    │
    ├── Sku (instance type) ──────┐
    │                             ├──► rdsInstanceIndex ──► rdsInstancePrice
    └── Tags["engine"] ───────────┘

    └── Tags["storage_type"] ─────────► rdsStorageIndex ──► rdsStoragePrice
```

## Thread Safety

All index maps are populated once during `init()` via `sync.Once`. After initialization,
maps are read-only, making concurrent access safe without additional locking.
