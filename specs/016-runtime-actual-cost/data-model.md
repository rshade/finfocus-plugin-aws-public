# Data Model: Runtime-Based Actual Cost Estimation

**Feature Branch**: `016-runtime-actual-cost`
**Date**: 2025-12-31

## Overview

This document defines the data structures and their relationships for the runtime-based
actual cost estimation feature. The design extends existing proto messages without
modification, using semantic encoding for new metadata.

---

## Entity Definitions

### 1. TimestampResolution

**Purpose**: Captures the resolved start/end timestamps and their provenance.

**Location**: `internal/plugin/actual.go` (new type)

```go
// TimestampResolution captures the resolved timestamps and their source.
// This struct is used internally to track how timestamps were determined
// and whether the resource was imported.
type TimestampResolution struct {
    // Start is the resolved start timestamp for cost calculation.
    Start time.Time

    // End is the resolved end timestamp for cost calculation.
    End time.Time

    // Source indicates how timestamps were resolved.
    // Values: "explicit", "pulumi:created", "mixed"
    Source string

    // IsImported is true when the resource has pulumi:external=true.
    // This affects confidence level (MEDIUM vs HIGH).
    IsImported bool
}
```

**Validation Rules**:
- `Start` must be before or equal to `End`
- If `End` equals `Start`, runtime is zero (valid case)
- `Source` must be one of the defined values

**State Transitions**: N/A (immutable value object)

---

### 2. ConfidenceLevel

**Purpose**: Represents the estimation confidence for actual cost calculations.

**Location**: `internal/plugin/actual.go` (new type)

```go
// ConfidenceLevel represents the estimation confidence for actual cost calculations.
// The level affects how the cost data should be interpreted by consumers.
type ConfidenceLevel string

const (
    // ConfidenceHigh indicates precise calculation with known timestamps.
    // Used when: explicit request timestamps OR native resource with pulumi:created.
    ConfidenceHigh ConfidenceLevel = "HIGH"

    // ConfidenceMedium indicates reasonable estimate with caveats.
    // Used when: imported resource (pulumi:external=true) - timestamp is import time.
    ConfidenceMedium ConfidenceLevel = "MEDIUM"

    // ConfidenceLow indicates rough estimate or fallback.
    // Used when: unsupported resource, missing data, or significant assumptions.
    ConfidenceLow ConfidenceLevel = "LOW"
)
```

**Confidence Determination Logic**:

| Scenario | IsImported | Confidence |
|----------|------------|------------|
| Explicit timestamps provided | false | HIGH |
| Explicit timestamps provided | true | HIGH |
| Using pulumi:created | false | HIGH |
| Using pulumi:created | true | MEDIUM |
| Unsupported resource type | - | LOW |
| Zero cost (no runtime) | - | HIGH |

---

### 3. Pulumi Metadata Tags

**Purpose**: Standard tag keys for Pulumi state metadata injected by pulumicost-core.

**Location**: `internal/plugin/actual.go` (constants)

```go
// Pulumi metadata tag keys.
// These keys are injected by pulumicost-core from Pulumi state.
const (
    // TagPulumiCreated is the RFC3339 timestamp of resource creation in Pulumi state.
    // For imported resources, this is the import time, not actual cloud creation.
    TagPulumiCreated = "pulumi:created"

    // TagPulumiModified is the RFC3339 timestamp of last modification.
    // NOTE: This feature does NOT use modified time as a fallback for created.
    TagPulumiModified = "pulumi:modified"

    // TagPulumiExternal indicates the resource was imported (not created by Pulumi).
    // Value is "true" when present; absence means native Pulumi resource.
    TagPulumiExternal = "pulumi:external"
)
```

**Input Validation**:
- Timestamps must be valid RFC3339 format
- Invalid timestamps are treated as missing (fallback to explicit times)
- `pulumi:external` value comparison is case-sensitive ("true" only)

---

## Proto Message Extensions

### ActualCostResult.source Field Encoding

**Existing Proto Field** (no modification needed):

```protobuf
message ActualCostResult {
    // ... other fields ...
    string source = 5;  // Identifies the data source
}
```

**Semantic Encoding Format**:

```text
source = base_source "[confidence:" level "]" [ " " note ]

base_source = "aws-public-fallback"
level       = "HIGH" | "MEDIUM" | "LOW"
note        = arbitrary_string

Examples:
  "aws-public-fallback[confidence:HIGH]"
  "aws-public-fallback[confidence:MEDIUM] imported resource"
  "aws-public-fallback[confidence:LOW] unsupported resource type"
```

**Parser Pattern** (for consumers):

```go
// ParseSourceConfidence extracts the confidence level from an ActualCostResult.source string.
// Returns ("HIGH"|"MEDIUM"|"LOW", true) if found, or ("", false) if not encoded.
func ParseSourceConfidence(source string) (string, bool) {
    const prefix = "[confidence:"
    start := strings.Index(source, prefix)
    if start < 0 {
        return "", false
    }
    start += len(prefix)
    end := strings.Index(source[start:], "]")
    if end < 0 {
        return "", false
    }
    return source[start : start+end], true
}
```

---

## Relationships

```text
┌─────────────────────────┐
│  GetActualCostRequest   │
├─────────────────────────┤
│ start: Timestamp (opt)  │──┐
│ end: Timestamp (opt)    │  │
│ tags: map[string]string │──┼──► resolveTimestamps()
└─────────────────────────┘  │
                             │
  Tags contain:              │
  - pulumi:created ─────────►│
  - pulumi:external ────────►│
                             │
                             ▼
                  ┌─────────────────────────┐
                  │   TimestampResolution   │
                  ├─────────────────────────┤
                  │ Start: time.Time        │
                  │ End: time.Time          │
                  │ Source: string          │──► determineConfidence()
                  │ IsImported: bool        │         │
                  └─────────────────────────┘         │
                             │                        │
                             │                        ▼
                             │              ┌─────────────────┐
                             │              │ ConfidenceLevel │
                             │              └─────────────────┘
                             │                        │
                             ▼                        │
                  ┌─────────────────────────┐         │
                  │   GetActualCostResponse │         │
                  ├─────────────────────────┤         │
                  │ results: []ActualCost   │◄────────┘
                  │   └─ source: string     │  (confidence encoded)
                  └─────────────────────────┘
```

---

## Function Signatures

### New Functions (internal/plugin/actual.go)

```go
// resolveTimestamps applies priority-based timestamp resolution.
// Priority: (1) explicit req.Start/End, (2) pulumi:created, (3) error
//
// Returns TimestampResolution with source tracking for confidence determination.
// Returns error if no valid timestamps can be resolved.
func resolveTimestamps(req *pbc.GetActualCostRequest) (*TimestampResolution, error)

// extractPulumiCreated parses the pulumi:created timestamp from tags.
// Returns (timestamp, true) if valid RFC3339, or (zero, false) if missing/invalid.
func extractPulumiCreated(tags map[string]string) (time.Time, bool)

// isImportedResource checks if the resource has pulumi:external=true.
func isImportedResource(tags map[string]string) bool

// determineConfidence maps resolution source and import status to confidence level.
func determineConfidence(resolution *TimestampResolution) ConfidenceLevel

// formatSourceWithConfidence creates the source string with embedded confidence.
func formatSourceWithConfidence(confidence ConfidenceLevel, note string) string
```

### Modified Functions (internal/plugin/plugin.go)

```go
// GetActualCost - MODIFIED
// Changes:
// 1. Call resolveTimestamps() before validation
// 2. Update req.Start/End with resolved values if needed
// 3. Determine confidence from resolution
// 4. Include confidence in response source field
func (p *AWSPublicPlugin) GetActualCost(ctx context.Context, req *pbc.GetActualCostRequest) (*pbc.GetActualCostResponse, error)
```

---

## Test Scenarios Mapping

| Spec Scenario | Data Model Element | Validation |
|---------------|-------------------|------------|
| US1-1: pulumi:created 7 days ago | extractPulumiCreated | Parse RFC3339 |
| US1-2: Explicit start after creation | resolveTimestamps | Priority check |
| US1-3: Missing pulumi:created | resolveTimestamps | Error path |
| US2-1: pulumi:external=true | isImportedResource | MEDIUM confidence |
| US2-2: No pulumi:external | isImportedResource | HIGH confidence |
| US3-1: Override with explicit times | resolveTimestamps | Explicit wins |
| Edge: Invalid RFC3339 | extractPulumiCreated | Returns false |
| Edge: End before start | resolveTimestamps | Zero cost path |

---

## Constants and Limits

```go
const (
    // hoursPerMonth is the standard hours-per-month constant for cost calculations.
    // Existing constant, documented here for reference.
    hoursPerMonth = 730.0

    // RFC3339 is the expected timestamp format.
    // Go's time.RFC3339 = "2006-01-02T15:04:05Z07:00"
)
```

---

## Migration Notes

**Backward Compatibility**:

1. **Existing callers**: Continue to work unchanged (explicit timestamps required)
2. **New callers**: Can omit timestamps if `pulumi:created` is in tags
3. **Response parsing**: Existing consumers see `source` as before; new consumers
   can parse confidence suffix

**No database migrations**: Feature is stateless, no persistent storage changes.
