# Research: GetRecommendations RPC Implementation

**Date**: 2025-12-15
**Feature**: 012-recommendations

## R1: Proto API Structure Analysis

### Decision

Use the official `pulumicost.v1` proto types for recommendations, which differ from the simplified proposal in the GitHub issue.

### Findings

The pulumicost-spec v0.4.7 defines a rich `Recommendation` type:

```go
type Recommendation struct {
    Id               string                      // Unique ID (generate UUID)
    Category         RecommendationCategory      // Enum: COST_OPTIMIZATION, etc.
    ActionType       RecommendationActionType    // Enum: MODIFY, RIGHTSIZE, etc.
    Resource         *ResourceRecommendationInfo // Resource context
    ActionDetail     isRecommendation_ActionDetail // oneof: Modify, Rightsize, etc.
    Impact           *RecommendationImpact       // Financial impact
    Priority         RecommendationPriority      // Enum: LOW, MEDIUM, HIGH, CRITICAL
    ConfidenceScore  *float64                    // 0.0-1.0 (not string)
    Description      string                      // Human-readable summary
    Reasoning        []string                    // List of reasons
    Source           string                      // "aws-public"
    Metadata         map[string]string           // Additional details
}
```

### Mapping Spec Requirements to Proto

| Spec Requirement | Proto Mapping |
|------------------|---------------|
| Type: "generation_upgrade" | `Category: RECOMMENDATION_CATEGORY_COST_OPTIMIZATION`, `ActionType: RECOMMENDATION_ACTION_TYPE_MODIFY`, `Modify.ModificationType: "generation_upgrade"` |
| Type: "graviton" | Same as above with `ModificationType: "graviton_migration"` |
| Type: "volume_type_upgrade" | Same as above with `ModificationType: "volume_type_upgrade"` |
| Confidence "high" | `ConfidenceScore: 0.9` |
| Confidence "medium" | `ConfidenceScore: 0.7` |
| Monthly savings | `Impact.EstimatedSavings` with `ProjectionPeriod: "monthly"` |
| Details map | `Metadata` map |

### Rationale

Using official proto types ensures:
1. Compatibility with PulumiCost core aggregation
2. Proper validation via `pluginsdk.ValidateRecommendation()`
3. Pagination support via `pluginsdk.PaginateRecommendations()`
4. Summary calculation via `pluginsdk.CalculateRecommendationSummary()`

### Alternatives Rejected

- Custom JSON types: Would break protocol compatibility
- Simplified types: Would require custom serialization

---

## R2: RecommendationsProvider Interface

### Decision

Implement `pluginsdk.RecommendationsProvider` interface to enable GetRecommendations.

### Findings

The SDK provides an optional interface:

```go
type RecommendationsProvider interface {
    GetRecommendations(ctx context.Context, req *pbc.GetRecommendationsRequest) (
        *pbc.GetRecommendationsResponse, error)
}
```

Plugins that implement this interface will have `GetRecommendations` available via gRPC.

### Implementation Pattern

```go
// Ensure AWSPublicPlugin implements RecommendationsProvider
var _ pluginsdk.RecommendationsProvider = (*AWSPublicPlugin)(nil)

func (p *AWSPublicPlugin) GetRecommendations(
    ctx context.Context,
    req *pbc.GetRecommendationsRequest,
) (*pbc.GetRecommendationsResponse, error) {
    // Implementation
}
```

### SDK Helpers Available

- `pluginsdk.ValidateRecommendation(rec)` - Validate recommendation structure
- `pluginsdk.ValidateRecommendationImpact(impact)` - Validate impact fields
- `pluginsdk.ApplyRecommendationFilter(recs, filter)` - Apply request filters
- `pluginsdk.PaginateRecommendations(recs, pageSize, pageToken)` - Handle pagination
- `pluginsdk.CalculateRecommendationSummary(recs, period)` - Generate summary

---

## R3: Instance Type Parsing

### Decision

Implement `parseInstanceType(instanceType string) (family, size string)` using simple string split.

### Findings

AWS instance types follow the pattern: `<family><generation>[<modifier>].<size>`

Examples:
- `t2.medium` → family: `t2`, size: `medium`
- `t3a.large` → family: `t3a`, size: `large`
- `m5d.xlarge` → family: `m5d`, size: `xlarge`
- `c6i.2xlarge` → family: `c6i`, size: `2xlarge`

### Implementation

```go
func parseInstanceType(instanceType string) (family, size string) {
    parts := strings.SplitN(instanceType, ".", 2)
    if len(parts) != 2 {
        return "", ""
    }
    return parts[0], parts[1]
}
```

### Edge Cases

- Empty string: returns `"", ""`
- No dot (e.g., `metal`): returns `"", ""`
- Multiple dots (shouldn't exist): only splits on first

---

## R4: Generation Upgrade Mappings

### Decision

Create static mapping of instance family generations.

### Findings

Based on AWS documentation and common upgrade paths:

```go
var generationUpgradeMap = map[string]string{
    // T-series (burstable)
    "t2":  "t3",
    "t3":  "t3a",  // AMD variant, often cheaper

    // M-series (general purpose)
    "m4":  "m5",
    "m5":  "m6i",
    "m5a": "m6a",

    // C-series (compute optimized)
    "c4":  "c5",
    "c5":  "c6i",
    "c5a": "c6a",

    // R-series (memory optimized)
    "r4":  "r5",
    "r5":  "r6i",
    "r5a": "r6a",

    // I-series (storage optimized)
    "i3":  "i3en",

    // D-series (dense storage)
    "d2":  "d3",
}
```

### Price Validation Required

Before recommending, must verify:
1. New instance type exists in pricing data
2. New price ≤ current price (avoid false positives)

---

## R5: Graviton Migration Mappings

### Decision

Create mapping from x86 families to Graviton equivalents.

### Findings

Graviton instances use `g` suffix (m6g, c6g, r6g, t4g):

```go
var gravitonMap = map[string]string{
    // From Intel/AMD to Graviton
    "m5":   "m6g",
    "m5a":  "m6g",
    "m5n":  "m6g",
    "m6i":  "m6g",
    "m6a":  "m6g",

    "c5":   "c6g",
    "c5a":  "c6g",
    "c5n":  "c6gn",
    "c6i":  "c6g",
    "c6a":  "c6g",

    "r5":   "r6g",
    "r5a":  "r6g",
    "r5n":  "r6g",
    "r6i":  "r6g",
    "r6a":  "r6g",

    "t3":   "t4g",
    "t3a":  "t4g",
}
```

### Confidence Level

Set `ConfidenceScore: 0.7` (medium) because:
- Requires ARM architecture compatibility
- Application must be tested on Graviton
- Some software doesn't support ARM

### Required Metadata

Include warning in `Metadata`:
```go
"architecture_change": "x86_64 -> arm64"
"requires_validation": "Application must support ARM architecture"
```

---

## R6: EBS gp2 to gp3 Migration

### Decision

Recommend gp2→gp3 unconditionally (gp3 is always cheaper and faster).

### Findings

gp3 advantages over gp2:
- **Price**: ~20% cheaper per GB-month
- **Baseline IOPS**: 3000 (vs 100 IOPS/GB for gp2)
- **Baseline Throughput**: 125 MB/s (vs scaling with size for gp2)
- **API compatible**: Same attach/detach process

### Implementation

Only recommend for `volumeType == "gp2"`:

```go
func (p *AWSPublicPlugin) getEBSRecommendations(resource *pbc.ResourceDescriptor) []*pbc.Recommendation {
    if resource.Sku != "gp2" {
        return nil
    }
    // Generate recommendation
}
```

### Size Handling

If volume size not in tags, use 100GB default for cost calculation.
Include actual volume size in metadata if available.

---

## R7: Request Handling

### Decision

The `GetRecommendationsRequest` uses filters, not a direct resource descriptor. Generate recommendations for the plugin's embedded region.

### Findings

The request structure:
```go
type GetRecommendationsRequest struct {
    Filter           *RecommendationFilter
    ProjectionPeriod string  // "daily", "monthly", "annual"
    PageSize         int32
    PageToken        string
}
```

### Implementation Approach

Since this plugin is region-specific and doesn't have access to actual resource inventory:

1. **Option A**: Return empty list (plugin doesn't track resources)
2. **Option B**: Accept resource descriptor via filter metadata
3. **Option C**: Return static recommendations based on common patterns

**Selected**: Option A with enhancement - Plugin returns empty unless called with specific resource context via the filter's `ResourceId` field.

For practical use, PulumiCost core would call GetRecommendations for specific resources it knows about, passing context via the filter.

---

## R8: Error Handling

### Decision

Return empty recommendations list for unsupported services (not error).

### Findings

Per spec FR-008: "System MUST return empty recommendations list (not error) for unsupported resource types"

Per spec FR-009: "System MUST return ERROR_CODE_INVALID_RESOURCE when request or resource descriptor is nil"

### Implementation

```go
func (p *AWSPublicPlugin) GetRecommendations(ctx context.Context, req *pbc.GetRecommendationsRequest) (
    *pbc.GetRecommendationsResponse, error) {

    if req == nil {
        return nil, p.newErrorWithID(traceID, codes.InvalidArgument,
            "missing request", pbc.ErrorCode_ERROR_CODE_INVALID_RESOURCE)
    }

    // For unsupported services, return empty list
    return &pbc.GetRecommendationsResponse{
        Recommendations: []*pbc.Recommendation{},
        Summary:         pluginsdk.CalculateRecommendationSummary(nil, "monthly"),
    }, nil
}
```

---

## Summary

All technical decisions resolved. No NEEDS CLARIFICATION items remaining.

| Research Item | Decision | Confidence |
|---------------|----------|------------|
| R1: Proto API | Use official types | High |
| R2: Interface | Implement RecommendationsProvider | High |
| R3: Instance parsing | Simple string split | High |
| R4: Generation map | Static mapping with price check | High |
| R5: Graviton map | Static mapping, 0.7 confidence | High |
| R6: EBS gp2→gp3 | Unconditional recommend | High |
| R7: Request handling | Filter-based or empty | Medium |
| R8: Error handling | Empty list for unsupported | High |
