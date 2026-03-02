# Research: Correctness Fixes Batch

**Branch**: `001-correctness-fixes`
**Date**: 2026-02-16

## R1: Proto Response Metadata Constraint

**Decision**: Use structured `BillingDetail` suffix for confidence signal
(adapted from clarification Option A).

**Rationale**: The `GetProjectedCostResponse` proto (finfocus-spec v0.5.6)
does **not** have a `map<string, string> metadata` field. Available fields:
`unit_price`, `currency`, `cost_per_month`, `billing_detail`,
`impact_metrics`, `growth_type`, `dry_run_result`, `pricing_category`,
`spot_interruption_risk_score`, `prediction_interval_lower/upper`,
`confidence_level`, `expires_at`. None serve as a generic metadata
carrier.

The `BillingDetail` field already contains default-indication text
(e.g., "EBS gp3 storage, 8GB (defaulted)"). This pattern will be
extended with a structured suffix that is both human-readable and
machine-parseable by the cost diff engine.

**Format**:

```text
EBS gp3 storage, 8GB [defaults:size=8] [confidence:low]
```

The `[defaults:key1=val1,key2=val2]` and `[confidence:high|medium|low]`
suffixes use bracket notation that is:

- Distinguishable from natural language (brackets are not used elsewhere
  in billing detail strings)
- Parseable with a simple regex: `\[defaults:([^\]]+)\]` and
  `\[confidence:(high|medium|low)\]`
- Human-readable as a fallback
- Backward compatible (existing consumers ignore unknown suffixes)

**Alternatives considered**:

- Proto schema change to add metadata map: Requires upstream
  finfocus-spec change, delays this batch. Proposed as follow-up.
- Abuse `PredictionIntervalLower/Upper`: Semantically incorrect, would
  confuse Core's forecasting logic.
- Separate RPC for confidence: Over-engineered for v1.

**Follow-up**: Propose `map<string, string> metadata` field addition to
`GetProjectedCostResponse` in finfocus-spec for v2 of confidence signal.

## R2: Zero-Cost Resource Detection in GetActualCost

**Decision**: Add `isZeroCostResource()` check to
`ValidateActualCostRequest()` and `parseResourceFromARN()`.

**Rationale**: `ValidateProjectedCostRequest()` (validation.go:60-86)
already checks `isZeroCostResource()` before SDK validation, skipping
the SKU requirement. The same pattern is missing from the actual cost
path. The existing `ZeroCostServices` map and `ZeroCostPulumiPatterns`
map in constants.go are the authoritative sources.

**Implementation pattern**: Mirror the projected cost validation:

1. In `ValidateActualCostRequest()`: Check `isZeroCostResource()` after
   ARN parsing but before SKU extraction
2. In `parseResourceFromARN()`: Return a valid ResourceDescriptor with
   empty SKU for zero-cost resources
3. In `GetActualCost()`: Route zero-cost resources to return empty
   `ActualCostResult` with $0 cost (not error)

**Alternatives considered**:

- Returning a `PricingUnavailableError`: Would cause fallback to
  recorder plugin, which is the current broken behavior.
- Adding zero-cost check only in `parseResourceFromARN()`: Insufficient;
  ResourceId-based requests also need the check.

## R3: LaunchTemplate / LaunchConfiguration Zero-Cost Patterns

**Decision**: Add `ec2/launchtemplate` and `ec2/launchconfiguration`
to `ZeroCostPulumiPatterns` in constants.go.

**Rationale**: LaunchTemplates are configuration-only AWS resources with
zero runtime cost. They define instance parameters but do not launch
instances. The actual compute cost is incurred by the ASG or EC2 Fleet
that references the template.

**Existing patterns** (constants.go:46-54):

```go
var ZeroCostPulumiPatterns = map[string]string{
    "ec2/vpc":           "vpc",
    "ec2/securitygroup": "securitygroup",
    "ec2/subnet":        "subnet",
}
```

New entries follow the same convention:

```go
"ec2/launchtemplate":      "launchtemplate",
"ec2/launchconfiguration": "launchconfiguration",
```

The `ZeroCostServices` map also needs the new service names.

**Alternatives considered**:

- Handling in `estimateEC2()`: Would require special-casing inside the
  EC2 estimator, violating SRP. The zero-cost pattern is cleaner.

## R4: Proto Getter Migration Scope

**Decision**: Fix all `protogetter` lint violations across the codebase
as part of the lint compliance work.

**Rationale**: The codebase is partially migrated. Recent changes (the
router branch) use getters consistently, but older code in projected.go,
enrichment.go, and other files still uses direct field access. The
`protogetter` linter reports ~50 violations.

**Pattern**: Replace `resource.ResourceType` with
`resource.GetResourceType()`, `resource.Sku` with `resource.GetSku()`,
etc. This is a mechanical transformation with no behavioral change.

## R5: Confidence Threshold per Service

**Decision**: Each service defines its own "applicable properties" count
for the ratio calculation.

| Service      | Applicable Properties                                     | Count |
|--------------|-----------------------------------------------------------|-------|
| EC2          | (none - SKU is required, not defaulted)                   | 0     |
| EBS          | size                                                      | 1     |
| S3           | size                                                      | 1     |
| RDS          | engine, storageType, allocatedStorage                     | 3     |
| Lambda       | memory, requests, duration, architecture                  | 4     |
| DynamoDB     | mode, RCU/WCU or requests, storage                        | 4     |
| ElastiCache  | engine, numNodes                                          | 2     |
| ELB          | type, capacityUnits                                       | 2     |
| NAT Gateway  | dataProcessedGb                                           | 1     |
| CloudWatch   | logIngestionGb, logStorageGb, customMetrics               | 3     |

**Threshold formula** (from clarification):

- `high`: 0 defaults (0% ratio)
- `medium`: 1 to floor(count/2) defaults (1-49%)
- `low`: ceil(count/2) or more defaults (50%+)

For services with 1 applicable property (EBS, S3, NAT Gateway):
0 defaults = high, 1 default = low (no medium possible).
