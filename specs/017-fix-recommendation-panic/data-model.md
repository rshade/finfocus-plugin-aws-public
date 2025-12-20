# Data Model: Bug Fix and Documentation Sprint - Dec 2025

**Feature**: 017-fix-recommendation-panic

## Key Entities

### Carbon Data (internal/carbon)

Represents the embedded power consumption characteristics for EC2 instance types.

| Field | Type | Description |
|-------|------|-------------|
| InstanceType | string | Canonical AWS instance type (e.g., "t3.micro") |
| VCPUCount | int | Number of vCPUs |
| MinWatts | float64 | Power consumption at idle (watts per vCPU) |
| MaxWatts | float64 | Power consumption at 100% utilization (watts per vCPU) |

**Validation Rules**:
- `VCPUCount` must be >= 1.
- `MinWatts` must be >= 0.
- `MaxWatts` must be >= `MinWatts`.

### S3 Resource (internal/plugin)

Represents an S3 bucket for cost estimation.

| Field | Type | Description |
|-------|------|-------------|
| BucketName | string | Unique name of the S3 bucket |
| Region | string | AWS region where bucket resides (falls back to plugin region if global ARN) |
| StorageClass | string | S3 storage class (STANDARD, GLACIER, etc.) |

## State Transitions

### Recommendation Processing Lifecycle

1. **Request Received**: `GetRecommendations` called with `target_resources`.
2. **Analysis**: Iterate through resources, generate potential optimizations.
3. **Impact Calculation**: Calculate potential savings.
4. **Validation (NEW)**: Verify `Impact` is non-nil.
5. **Aggregation**: Add savings to `BatchStats.TotalSavings` ONLY if `Impact` is valid.
6. **Response**: Return list of recommendations with unique UUIDs.

## Persistence

All pricing data and carbon specifications are embedded in the binary using `//go:embed` and loaded into thread-safe maps at startup using `sync.Once`.
