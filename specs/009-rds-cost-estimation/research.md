# Research: RDS Instance Cost Estimation

**Feature**: 009-rds-cost-estimation
**Date**: 2025-12-02

## Research Summary

No critical unknowns requiring clarification. Technical context is well-defined based on
existing EC2/EBS implementation patterns and AWS Pricing API documentation.

## Decision Log

### 1. AWS Pricing API Structure for RDS

**Decision**: Use AWS Price List API with service code `AmazonRDS`

**Rationale**: The existing `tools/generate-pricing/main.go` already supports parameterized
service codes. RDS pricing follows the same JSON structure as EC2 with `products` and
`terms.OnDemand` sections.

**Alternatives Considered**:

- AWS Cost Explorer API: Rejected - requires credentials and provides historical data, not
  public pricing
- Web scraping AWS pricing page: Rejected - fragile, against ToS

**URL Pattern**:

```text
https://pricing.us-east-1.amazonaws.com/offers/v1.0/aws/AmazonRDS/current/{region}/index.json
```

### 2. RDS Product Family Filters

**Decision**: Filter by `ProductFamily` values for instance and storage pricing

**Rationale**: AWS Price List JSON uses consistent product family naming. Testing against
actual API responses confirms these values.

**Filters**:

| Component | ProductFamily | Key Attributes |
|-----------|---------------|----------------|
| Instances | `Database Instance` | `dbInstanceClass`, `databaseEngine`, `deploymentOption` |
| Storage | `Database Storage` | `volumeType`, `storageMedia` |

**Additional Filters for Single-AZ**:

- `deploymentOption == "Single-AZ"` (excludes Multi-AZ pricing)
- Exclude `Reserved` term types (use only `OnDemand`)

### 3. Database Engine Mapping

**Decision**: Map AWS API engine names to user-friendly tag values

**Rationale**: AWS uses capitalized engine names in the API, but users typically pass
lowercase values. Normalize input to match AWS naming.

| AWS API Value | User Tag Value | Notes |
|---------------|----------------|-------|
| `MySQL` | `mysql` | Community Edition |
| `PostgreSQL` | `postgres` | Standard PostgreSQL |
| `MariaDB` | `mariadb` | MariaDB Community |
| `Oracle` | `oracle-se2` | Standard Edition 2 (license-included) |
| `SQL Server` | `sqlserver-ex` | Express Edition (license-included) |

**Engine Normalization**: Convert user input to title case for lookup, with special handling
for Oracle and SQL Server editions.

### 4. Pricing Index Key Structure

**Decision**: Use composite key `{instanceType}/{engine}` for instance pricing

**Rationale**: RDS pricing varies by both instance type AND database engine (unlike EC2
which varies by OS). The composite key ensures accurate lookups.

**Index Keys**:

- **Instance**: `db.t3.medium/MySQL` → hourly rate
- **Storage**: `gp2` → rate per GB-month (storage pricing is engine-agnostic for most types)

### 5. Default Values

**Decision**: Use sensible defaults matching typical RDS configurations

| Parameter | Default | Rationale |
|-----------|---------|-----------|
| engine | `mysql` | Most popular RDS engine |
| storage_type | `gp2` | Default RDS storage type |
| storage_size | `20` GB | AWS minimum for most engines |

### 6. Cost Calculation Formula

**Decision**: Combine instance and storage costs

```text
monthly_cost = (hourly_rate × 730) + (storage_rate × storage_gb)
unit_price = hourly_rate (primary cost driver)
```

**Rationale**: Follows EC2/EBS pattern where `unit_price` represents the atomic billing
unit (hourly for compute) and `cost_per_month` is the total projected cost.

### 7. Billing Detail Format

**Decision**: Include all assumptions in human-readable format

**Format**:

```text
RDS {instance_type} {engine}, 730 hrs/month + {size}GB {storage_type} storage
```

**Examples**:

- `RDS db.t3.medium MySQL, 730 hrs/month + 100GB gp3 storage`
- `RDS db.m5.large PostgreSQL, 730 hrs/month + 20GB gp2 storage (defaulted)`

### 8. Generate-Pricing Tool Update

**Decision**: Extend existing tool to support multiple services

**Approach**: The tool already accepts `--service` flag. Need to:

1. Call tool twice in GoReleaser (once for AmazonEC2, once for AmazonRDS)
2. OR modify to accept comma-separated services
3. Combine both EC2 and RDS data into single embedded JSON per region

**Chosen**: Option 3 - Combine data into single JSON file per region for simpler embedding.

## Best Practices Applied

### From Existing Codebase

1. **Thread-safe initialization**: Use `sync.Once` for parsing embedded data
2. **Indexed lookups**: Build maps during init, O(1) lookups at runtime
3. **Performance logging**: Log warning if lookup exceeds 50ms
4. **Graceful degradation**: Return $0 with explanation for unknown types
5. **Trace ID propagation**: Pass traceID through all method calls

### From Constitution

1. **KISS**: No new abstractions; extend existing `PricingClient` interface
2. **Single Responsibility**: `estimateRDS()` handles only RDS, delegates to pricing client
3. **Proto-defined types**: Use existing `GetProjectedCostResponse` structure
4. **Table-driven tests**: Test multiple engines/instance types in single test function

## Open Items

None. All technical decisions resolved.
