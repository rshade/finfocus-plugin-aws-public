# Data Model: Correctness Fixes Batch

**Branch**: `001-correctness-fixes`
**Date**: 2026-02-16

## Entities

### Zero-Cost Resource Registry

Extends the existing `ZeroCostServices` and `ZeroCostPulumiPatterns`
maps in `internal/plugin/constants.go`.

**Current state**:

| Service Key       | Pulumi Pattern         |
|-------------------|------------------------|
| vpc               | ec2/vpc                |
| securitygroup     | ec2/securitygroup      |
| subnet            | ec2/subnet             |
| iam               | (no pattern)           |

**New entries**:

| Service Key          | Pulumi Pattern            |
|----------------------|---------------------------|
| launchtemplate       | ec2/launchtemplate        |
| launchconfiguration  | ec2/launchconfiguration   |

**Lookup flow**: `normalizeResourceType()` matches Pulumi patterns
token-aware (requires ":" boundary). `detectService()` then resolves
to the canonical service name. `IsZeroCostService()` checks the
`ZeroCostServices` map.

### Property Confidence Signal

Embedded in the `BillingDetail` string field of
`GetProjectedCostResponse`. Not a separate data structure at the proto
level.

**Format**:

```text
{human-readable detail} [defaults:{key}={value},...] [confidence:{level}]
```

**Fields**:

| Field       | Type   | Values                          |
|-------------|--------|---------------------------------|
| defaults    | string | Comma-separated key=value pairs |
| confidence  | enum   | "high", "medium", "low"         |

**Confidence computation**:

```text
ratio = count(defaulted_properties) / count(applicable_properties)

high:   ratio == 0.0  (no defaults)
medium: 0.0 < ratio < 0.5
low:    ratio >= 0.5
```

**Per-service applicable properties**:

| Service      | Properties                                        | Count |
|--------------|---------------------------------------------------|-------|
| EC2          | (none defaulted; SKU required)                    | 0     |
| EBS          | size                                              | 1     |
| S3           | size                                              | 1     |
| RDS          | engine, storageType, allocatedStorage              | 3     |
| Lambda       | memory, requests, duration, architecture           | 4     |
| DynamoDB     | mode, RCU/WCU or requests, storage                | 4     |
| ElastiCache  | engine, numNodes                                  | 2     |
| ELB          | type, capacityUnits                               | 2     |
| NAT Gateway  | dataProcessedGb                                   | 1     |
| CloudWatch   | logIngestionGb, logStorageGb, customMetrics        | 3     |

### Validation Path Comparison

| Path                        | Zero-Cost Check | SKU Required | Affected Code            |
|-----------------------------|-----------------|--------------|--------------------------|
| ValidateProjectedCostRequest | Yes             | No (skipped) | validation.go:60-86      |
| ValidateActualCostRequest    | **No (bug)**    | Yes (fails)  | validation.go:206-306    |
| parseResourceFromARN         | **No (bug)**    | Yes (fails)  | validation.go:331-371    |

After fix, all three paths will check `isZeroCostResource()` before
requiring SKU.

## State Transitions

No new state transitions. Zero-cost resources are stateless lookups.

## Relationships

- `ZeroCostServices` map is consumed by `IsZeroCostService()` function
- `ZeroCostPulumiPatterns` map is consumed by
  `normalizeResourceType()` function
- Both maps must be kept in sync (a pattern implies its service key
  exists in the services map)
- Property confidence is computed per-request, not stored
