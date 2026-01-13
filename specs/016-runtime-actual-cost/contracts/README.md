# API Contracts: Runtime-Based Actual Cost Estimation

**Feature Branch**: `016-runtime-actual-cost`
**Date**: 2025-12-31

## Overview

This feature does NOT require proto schema modifications. All new functionality is
implemented using existing proto message fields with semantic encoding.

## Existing Contract (Unchanged)

The `GetActualCost` RPC uses the existing `finfocus.v1.CostSourceService` contract:

```protobuf
service CostSourceService {
  rpc GetActualCost(GetActualCostRequest) returns (GetActualCostResponse);
}

message GetActualCostRequest {
  string resource_id = 1;
  google.protobuf.Timestamp start = 2;  // Now OPTIONAL (resolved from tags)
  google.protobuf.Timestamp end = 3;    // Now OPTIONAL (defaults to now)
  map<string, string> tags = 4;         // Pulumi metadata injected here
  string arn = 5;
}

message GetActualCostResponse {
  repeated ActualCostResult results = 1;
  FallbackHint fallback_hint = 2;
}

message ActualCostResult {
  google.protobuf.Timestamp timestamp = 1;
  double cost = 2;
  double usage_amount = 3;
  string usage_unit = 4;
  string source = 5;  // Confidence encoded here
  FocusCostRecord focus_record = 6;
  repeated ImpactMetric impact_metrics = 7;
}
```

## Extended Semantics (This Feature)

### Request Tags Extension

The `tags` map accepts the following Pulumi metadata keys (injected by finfocus-core):

| Key | Required | Format | Description |
|-----|----------|--------|-------------|
| `pulumi:created` | No | RFC3339 | Resource creation timestamp |
| `pulumi:modified` | No | RFC3339 | Last modification timestamp |
| `pulumi:external` | No | "true" | Indicates imported resource |

### Response Source Field Extension

The `source` field in `ActualCostResult` uses semantic encoding to communicate
confidence level:

**Format**: `provider[confidence:LEVEL] optional_note`

**Examples**:

```text
aws-public-fallback[confidence:HIGH]
aws-public-fallback[confidence:MEDIUM] imported resource
aws-public-fallback[confidence:LOW] unsupported resource type
```

**Confidence Levels**:

| Level | Meaning |
|-------|---------|
| HIGH | Precise calculation with known timestamps |
| MEDIUM | Reasonable estimate with caveats (e.g., imported resource) |
| LOW | Rough estimate or significant assumptions |

## No Proto Changes Required

This design intentionally avoids proto modifications:

1. **Backward Compatible**: Existing clients continue to work unchanged
2. **No Coordination**: No finfocus-spec PR or version bump needed
3. **Self-Documenting**: Source field carries its own metadata
4. **Easy Parsing**: Simple string pattern matching for consumers

## Future Proto Enhancement (Optional)

If the community desires explicit confidence fields, a future proto version could add:

```protobuf
message ActualCostResult {
  // ... existing fields ...

  // Proposed: Explicit confidence enum
  ConfidenceLevel confidence = 8;
}

enum ConfidenceLevel {
  CONFIDENCE_LEVEL_UNSPECIFIED = 0;
  CONFIDENCE_LEVEL_HIGH = 1;
  CONFIDENCE_LEVEL_MEDIUM = 2;
  CONFIDENCE_LEVEL_LOW = 3;
}
```

This would be a MINOR version increment to finfocus-spec and is not required
for this feature implementation.
