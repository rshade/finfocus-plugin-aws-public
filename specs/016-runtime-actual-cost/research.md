# Research: Runtime-Based Actual Cost Estimation

**Feature Branch**: `016-runtime-actual-cost`
**Date**: 2025-12-31
**Status**: Complete

## Summary

This document resolves the technical clarifications identified in the implementation
plan for feature #196. All NEEDS CLARIFICATION items have been researched and decisions
documented.

---

## R1: Confidence Field Representation

### Question

How to represent confidence levels (HIGH/MEDIUM/LOW) in `ActualCostResult` without
modifying the finfocus-spec proto schema?

### Research Findings

The `ActualCostResult` message has these available fields:

```protobuf
message ActualCostResult {
  google.protobuf.Timestamp timestamp = 1;
  double cost = 2;
  double usage_amount = 3;
  string usage_unit = 4;
  string source = 5;           // <-- Best option
  FocusCostRecord focus_record = 6;
  repeated ImpactMetric impact_metrics = 7;
}
```

**Options Evaluated:**

| Approach | Proto Change | Complexity | Clarity | Recommended |
|----------|--------------|-----------|---------|-------------|
| Semantic encoding in `source` | None | Low | Medium | **YES** |
| FOCUS extended_columns | None | Medium | High | Alternative |
| Custom ImpactMetric abuse | None | Low | Poor | No |
| Proto amendment | Yes | Medium | High | Future |

### Decision: Semantic Encoding in `source` Field

**Format Specification:**

```text
source_format = provider_name "[confidence:" level "]" [ notes ]
provider_name = "aws-public-fallback"
level         = "HIGH" | "MEDIUM" | "LOW"
notes         = " " detail_string

Examples:
  "aws-public-fallback[confidence:HIGH]"
  "aws-public-fallback[confidence:MEDIUM] imported resource"
  "aws-public-fallback[confidence:LOW] unsupported resource"
```

**Confidence Level Guidelines:**

| Scenario | Level | Rationale |
|----------|-------|-----------|
| Native resource with valid `pulumi:created` | HIGH | Precise runtime calculation |
| Explicit request timestamps provided | HIGH | User-specified time range |
| Imported resource (`pulumi:external=true`) | MEDIUM | Import time, not actual creation |
| Missing `pulumi:created`, no explicit times | LOW | Cannot determine runtime |
| Unsupported resource type | LOW | $0 estimate with explanation |

### Implementation Pattern

```go
// internal/plugin/actual.go

// ConfidenceLevel represents the estimation confidence for actual cost calculations.
type ConfidenceLevel string

const (
    ConfidenceHigh   ConfidenceLevel = "HIGH"
    ConfidenceMedium ConfidenceLevel = "MEDIUM"
    ConfidenceLow    ConfidenceLevel = "LOW"
)

// formatSourceWithConfidence creates the source string with embedded confidence level.
func formatSourceWithConfidence(confidence ConfidenceLevel, note string) string {
    if note != "" {
        return fmt.Sprintf("aws-public-fallback[confidence:%s] %s", confidence, note)
    }
    return fmt.Sprintf("aws-public-fallback[confidence:%s]", confidence)
}
```

### Alternatives Considered

1. **FOCUS `extended_columns`**: Good for detailed metadata but requires populating
   the entire FocusCostRecord, adding complexity for simple confidence indication.

2. **Proto amendment**: Ideal long-term but blocks implementation on external spec
   changes. Can be pursued in parallel if there's consensus in finfocus-spec.

---

## R2: Pulumi Metadata Injection

### Question

How does finfocus-core inject `pulumi:created`, `pulumi:modified`, and
`pulumi:external` into `GetActualCostRequest`? What are the exact key names?

### Research Findings

Per spec assumption A-001:
> finfocus-core injects `pulumi:created`, `pulumi:modified`, and `pulumi:external`
> into resource properties before calling the plugin.

The injection point is `GetActualCostRequest.tags` map. This follows the existing
pattern where resource metadata is passed via tags.

**Key Names (from spec):**

| Key | Type | Format | Description |
|-----|------|--------|-------------|
| `pulumi:created` | string | RFC3339 | Resource creation timestamp |
| `pulumi:modified` | string | RFC3339 | Last modification timestamp |
| `pulumi:external` | string | "true" | Present when resource was imported |

### Decision: Read from `req.Tags` Map

The plugin will read Pulumi metadata from the existing tags mechanism:

```go
// internal/plugin/actual.go

// PulumiMetadataKeys defines the standard keys for Pulumi state metadata.
const (
    TagPulumiCreated  = "pulumi:created"
    TagPulumiModified = "pulumi:modified"
    TagPulumiExternal = "pulumi:external"
)

// extractPulumiCreated parses the pulumi:created timestamp from tags.
// Returns (timestamp, true) if valid, or (zero, false) if missing/invalid.
func extractPulumiCreated(tags map[string]string) (time.Time, bool) {
    if tags == nil {
        return time.Time{}, false
    }
    createdStr, ok := tags[TagPulumiCreated]
    if !ok || createdStr == "" {
        return time.Time{}, false
    }
    t, err := time.Parse(time.RFC3339, createdStr)
    if err != nil {
        return time.Time{}, false
    }
    return t, true
}

// isImportedResource checks if the resource was imported (pulumi:external=true).
func isImportedResource(tags map[string]string) bool {
    if tags == nil {
        return false
    }
    return tags[TagPulumiExternal] == "true"
}
```

### Verification

- The current `parseResourceFromRequest()` function in `plugin.go` already copies
  tags from the request to the ResourceDescriptor.
- The tags map is accessible via both `req.Tags` directly and through the parsed
  `resource.Tags` after validation.

---

## R3: Timestamp Priority Semantics

### Question

How should explicit `req.Start`/`req.End` interact with `pulumi:created`? What
happens when both are present? When neither is present?

### Research Findings

Current validation (`validateTimestamps` in `validation.go:158-170`) **requires**
both `req.Start` and `req.End` to be non-nil:

```go
func validateTimestamps(req *pbc.GetActualCostRequest) error {
    if req.Start == nil {
        return status.Error(codes.InvalidArgument, "start_time is required")
    }
    if req.End == nil {
        return status.Error(codes.InvalidArgument, "end_time is required")
    }
    // ...
}
```

This means the feature cannot make timestamps "optional" without modifying the
validation flow.

### Decision: Two-Phase Timestamp Resolution

**Phase 1: Pre-Validation Resolution (NEW)**

Before SDK validation, resolve timestamps using the priority order:

1. **Explicit Request Timestamps** (highest priority)
   - If `req.Start` and `req.End` are both set, use them
   - This enables "last 7 days" queries regardless of resource age

2. **Pulumi Metadata** (fallback)
   - If `req.Start` is nil but `tags["pulumi:created"]` exists, use it
   - If `req.End` is nil, default to `time.Now()`

3. **Error** (no timestamps available)
   - If neither explicit times nor `pulumi:created` is available, return error

**Phase 2: Validation**

After resolution, the existing `validateTimestamps()` runs with guaranteed
non-nil timestamps.

### Implementation Pattern

```go
// internal/plugin/actual.go

// TimestampResolution captures the resolved timestamps and their source.
type TimestampResolution struct {
    Start      time.Time
    End        time.Time
    Source     string       // "explicit", "pulumi:created", "mixed"
    IsImported bool         // true if pulumi:external=true
}

// resolveTimestamps applies the priority-based timestamp resolution.
// Priority: (1) explicit req.Start/End, (2) pulumi:created from tags, (3) error
func resolveTimestamps(req *pbc.GetActualCostRequest) (*TimestampResolution, error) {
    res := &TimestampResolution{}

    // Check explicit timestamps first
    hasExplicitStart := req.Start != nil && req.Start.IsValid()
    hasExplicitEnd := req.End != nil && req.End.IsValid()

    if hasExplicitStart && hasExplicitEnd {
        res.Start = req.Start.AsTime()
        res.End = req.End.AsTime()
        res.Source = "explicit"
        res.IsImported = isImportedResource(req.Tags)
        return res, nil
    }

    // Try pulumi:created for start time
    pulumiCreated, hasCreated := extractPulumiCreated(req.Tags)

    if !hasExplicitStart {
        if hasCreated {
            res.Start = pulumiCreated
            res.Source = "pulumi:created"
        } else {
            return nil, fmt.Errorf("start_time required: no explicit timestamp and no pulumi:created in tags")
        }
    } else {
        res.Start = req.Start.AsTime()
        res.Source = "explicit"
    }

    if !hasExplicitEnd {
        // Default end time to now
        res.End = time.Now()
        if res.Source == "explicit" {
            res.Source = "mixed"
        }
    } else {
        res.End = req.End.AsTime()
        if res.Source == "pulumi:created" {
            res.Source = "mixed"
        }
    }

    res.IsImported = isImportedResource(req.Tags)
    return res, nil
}
```

### Confidence Mapping

| Source | IsImported | Confidence |
|--------|------------|------------|
| explicit | false | HIGH |
| explicit | true | HIGH |
| pulumi:created | false | HIGH |
| pulumi:created | true | MEDIUM |
| mixed | false | HIGH |
| mixed | true | MEDIUM |
| error | - | (no response) |

### Edge Cases

1. **End before Start**: Return zero cost with explanation (FR-009)
2. **Invalid RFC3339**: Fall back to requiring explicit timestamps
3. **pulumi:modified without pulumi:created**: Do NOT use modified as creation time

---

## Design Decisions Summary

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Confidence encoding | `source` field semantic | No proto changes, backward compatible |
| Metadata source | `req.Tags` map | Matches existing pattern |
| Timestamp priority | Explicit > pulumi:created > error | Per FR-003 |
| Imported resource detection | `pulumi:external=true` tag | Per FR-004 |
| Default end time | `time.Now()` | Intuitive "cost since creation" |

---

## References

- **Proto Source**: `finfocus-spec/proto/finfocus/v1/costsource.proto`
  - ActualCostResult: lines 369-386
  - GetActualCostRequest: lines 140-152

- **Plugin Source**: `internal/plugin/`
  - validation.go: Request validation
  - plugin.go: GetActualCost handler
  - actual.go: Calculation helpers

- **Spec**: `/specs/016-runtime-actual-cost/spec.md`
  - FR-001 through FR-009
  - Assumptions A-001 through A-005
