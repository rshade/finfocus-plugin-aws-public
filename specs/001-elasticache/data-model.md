# Data Model: ElastiCache Cost Estimation

**Feature**: 001-elasticache
**Date**: 2025-12-31

## Entity Definitions

### ElastiCache Instance Price

**Purpose**: Represents the hourly pricing for a specific ElastiCache node type and engine combination.

**Location**: `internal/pricing/types.go`

```go
// elasticacheInstancePrice represents the hourly cost for an ElastiCache cache node.
// This is the primary pricing unit for ElastiCache - all cost calculations multiply
// this rate by node count and hours.
type elasticacheInstancePrice struct {
    // Unit is the billing unit, expected to be "Hrs" for hourly pricing
    Unit string

    // HourlyRate is the on-demand cost per hour in USD
    HourlyRate float64

    // Currency is the pricing currency, always "USD" for this plugin
    Currency string
}
```

**Key Properties**:
- Immutable after initialization (populated during pricing parse)
- Thread-safe reads (no mutations after init)
- Currency is always "USD" (hardcoded in plugin)

---

### ElastiCache Pricing Index

**Purpose**: Fast lookup of pricing by instance type and engine.

**Location**: `internal/pricing/client.go` (field on Client struct)

```go
// elasticacheIndex maps composite keys to pricing data.
// Key format: "{instanceType}:{engine}" (e.g., "cache.m5.large:Redis")
// Thread-safe after initialization via sync.Once
elasticacheIndex map[string]elasticacheInstancePrice
```

**Index Key Format**: `{instanceType}:{engine}`

**Examples**:

| Key | Instance Type | Engine |
|-----|---------------|--------|
| `cache.t3.micro:Redis` | cache.t3.micro | Redis |
| `cache.m5.large:Memcached` | cache.m5.large | Memcached |
| `cache.r6g.xlarge:Valkey` | cache.r6g.xlarge | Valkey |

**Validation Rules**:
- Instance type must start with `cache.`
- Engine must be one of: Redis, Memcached, Valkey
- HourlyRate must be positive

---

### Engine Normalization Map

**Purpose**: Normalize user input to AWS canonical engine names.

**Location**: `internal/pricing/client.go` (package-level variable)

```go
// elasticacheEngineNormalization maps lowercase engine names to AWS canonical form.
var elasticacheEngineNormalization = map[string]string{
    "redis":     "Redis",
    "memcached": "Memcached",
    "valkey":    "Valkey",
}
```

**Usage**: Called at lookup time to normalize user-provided engine names.

---

## Input/Output Structures

### Input: ResourceDescriptor

**Source**: Proto definition from pulumicost-spec

**Relevant Fields for ElastiCache**:

```go
type ResourceDescriptor struct {
    Provider     string            // "aws"
    ResourceType string            // "elasticache", "aws:elasticache/cluster:Cluster", etc.
    Sku          string            // Instance type: "cache.m5.large"
    Region       string            // "us-east-1"
    Tags         map[string]string // Contains "engine", "num_nodes", etc.
}
```

**Tag Extraction**:

| Tag Key | Purpose | Default |
|---------|---------|---------|
| `engine` | Cache engine type | "redis" |
| `num_cache_clusters` | Node count (Pulumi replication group) | 1 |
| `num_nodes` | Node count (generic) | 1 |
| `nodes` | Node count (short form) | 1 |

---

### Output: GetProjectedCostResponse

**Source**: Proto definition from pulumicost-spec

```go
type GetProjectedCostResponse struct {
    CostPerMonth  float64  // Total monthly cost (hourly × nodes × 730)
    UnitPrice     float64  // Hourly rate per node
    Currency      string   // "USD"
    BillingDetail string   // Human-readable explanation
}
```

**BillingDetail Format**:

```text
ElastiCache {instanceType} ({engine}), {numNodes} node(s), 730 hrs/month
```

**With Defaults**:

```text
ElastiCache cache.m5.large (redis), 1 node(s), 730 hrs/month, engine defaulted to redis, node count defaulted to 1
```

---

## State Transitions

### Pricing Client Lifecycle

```text
┌─────────────────┐
│  Uninitialized  │
│  (zero values)  │
└────────┬────────┘
         │
         │ First access triggers sync.Once
         ▼
┌─────────────────┐
│   Initializing  │
│  (parsing JSON) │
└────────┬────────┘
         │
         │ Parallel parse completes
         ▼
┌─────────────────┐
│   Initialized   │
│ (ready for use) │
└─────────────────┘
```

**States**:
1. **Uninitialized**: `elasticacheIndex` is nil
2. **Initializing**: `sync.Once` in progress, parsing JSON in goroutine
3. **Initialized**: Map populated, ready for concurrent reads

**Thread Safety**: After initialization, all map access is read-only. No mutations occur during normal operation.

---

## Relationships

```text
┌─────────────────────────────────────────────────────────────────┐
│                        Client (pricing)                          │
│                                                                  │
│  ┌──────────────────┐    ┌──────────────────┐                   │
│  │ elasticacheIndex │    │ ec2Index         │                   │
│  │ (map)            │    │ ebsIndex         │ ... other indexes │
│  └────────┬─────────┘    │ rdsInstanceIndex │                   │
│           │              └──────────────────┘                   │
└───────────┼─────────────────────────────────────────────────────┘
            │
            │ Contains
            ▼
┌───────────────────────────────────────────────────────────────┐
│              elasticacheInstancePrice                          │
│                                                                │
│  ┌──────────┐    ┌────────────┐    ┌──────────┐               │
│  │ Unit     │    │ HourlyRate │    │ Currency │               │
│  │ "Hrs"    │    │ 0.156      │    │ "USD"    │               │
│  └──────────┘    └────────────┘    └──────────┘               │
└───────────────────────────────────────────────────────────────┘
```

---

## Validation Rules

### Instance Type Validation

**Rule**: Must be non-empty
**Enforcement**: Return ERROR_CODE_INVALID_RESOURCE if missing

```go
if instanceType == "" {
    return nil, pluginsdk.NewPulumiCostError(
        pbc.ErrorCode_ERROR_CODE_INVALID_RESOURCE,
        "ElastiCache requires instance type in Sku or tags",
        nil,
    )
}
```

### Engine Validation

**Rule**: Normalize and attempt lookup
**Enforcement**: Return $0 if engine/instance combo not found

```go
engine := strings.ToLower(engineTag)
normalizedEngine, ok := elasticacheEngineNormalization[engine]
if !ok {
    // Unknown engine, try lookup anyway
    normalizedEngine = strings.Title(engine)
}
```

### Node Count Validation

**Rule**: Must be positive integer
**Enforcement**: Default to 1 if invalid

```go
numNodes := 1
if n, err := strconv.Atoi(nodesTag); err == nil && n > 0 {
    numNodes = n
}
```

---

## Data Volume Estimates

| Metric | Estimate | Notes |
|--------|----------|-------|
| Instance types | ~50 | cache.t3.*, m5.*, m6g.*, r5.*, r6g.*, etc. |
| Engines | 3 | Redis, Memcached, Valkey |
| Total SKUs per region | ~150 | 50 types × 3 engines |
| Index entries | ~150 | One per SKU |
| Memory per entry | ~100 bytes | Struct + map overhead |
| Total index memory | ~15KB | Negligible |
| JSON file size | 5-10MB | Per region |
