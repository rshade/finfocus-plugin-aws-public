# Tasks: GetRecommendations RPC Implementation

**Feature**: 012-recommendations
**Branch**: `012-recommendations`
**Generated**: 2025-12-15
**Spec**: [spec.md](spec.md) | **Plan**: [plan.md](plan.md)

## Task Overview

| Phase | User Story | Tasks | Priority |
|-------|------------|-------|----------|
| 1 | Setup | 3 | - |
| 2 | US1: EC2 Generation Upgrades | 4 | P1 |
| 3 | US2: Graviton Migrations | 2 | P2 |
| 4 | US3: EBS gp2→gp3 | 2 | P3 |
| 5 | US4: Graceful Error Handling | 3 | P4 |
| 6 | Polish | 2 | - |

---

## Phase 1: Setup

### T1.1: Create instance_type.go with parsing helper

**File**: `internal/plugin/instance_type.go`
**Depends on**: None
**Acceptance**: `go build ./...` passes

Create the instance type parsing utility and static mappings.

```go
// internal/plugin/instance_type.go
package plugin

import "strings"

// parseInstanceType splits an EC2 instance type into family and size.
// Example: "t2.medium" → ("t2", "medium")
// Returns empty strings if the format is invalid.
func parseInstanceType(instanceType string) (family, size string) {
    parts := strings.SplitN(instanceType, ".", 2)
    if len(parts) != 2 {
        return "", ""
    }
    return parts[0], parts[1]
}

// generationUpgradeMap maps old instance families to newer generations.
var generationUpgradeMap = map[string]string{
    "t2":  "t3",
    "t3":  "t3a",
    "m4":  "m5",
    "m5":  "m6i",
    "m5a": "m6a",
    "c4":  "c5",
    "c5":  "c6i",
    "c5a": "c6a",
    "r4":  "r5",
    "r5":  "r6i",
    "r5a": "r6a",
    "i3":  "i3en",
    "d2":  "d3",
}

// gravitonMap maps x86 families to Graviton equivalents.
var gravitonMap = map[string]string{
    "m5":  "m6g",
    "m5a": "m6g",
    "m5n": "m6g",
    "m6i": "m6g",
    "m6a": "m6g",
    "c5":  "c6g",
    "c5a": "c6g",
    "c5n": "c6gn",
    "c6i": "c6g",
    "c6a": "c6g",
    "r5":  "r6g",
    "r5a": "r6g",
    "r5n": "r6g",
    "r6i": "r6g",
    "r6a": "r6g",
    "t3":  "t4g",
    "t3a": "t4g",
}
```

**Test**: Add `instance_type_test.go` with table-driven tests for `parseInstanceType`.

---

### T1.2: Create recommendations.go skeleton

**File**: `internal/plugin/recommendations.go`
**Depends on**: T1.1
**Acceptance**: `go build ./...` passes, interface compile check

Create the GetRecommendations method skeleton implementing `pluginsdk.RecommendationsProvider`.

```go
// internal/plugin/recommendations.go
package plugin

import (
    "context"
    "time"

    "github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
    pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
    "google.golang.org/grpc/codes"
)

const (
    hoursPerMonth        = 730.0
    confidenceHigh       = 0.9
    confidenceMedium     = 0.7
    sourceAWSPublic      = "aws-public"
    modTypeGenUpgrade    = "generation_upgrade"
    modTypeGraviton      = "graviton_migration"
    modTypeVolumeUpgrade = "volume_type_upgrade"
)

// Ensure AWSPublicPlugin implements RecommendationsProvider
var _ pluginsdk.RecommendationsProvider = (*AWSPublicPlugin)(nil)

// GetRecommendations returns cost optimization recommendations.
func (p *AWSPublicPlugin) GetRecommendations(
    ctx context.Context,
    req *pbc.GetRecommendationsRequest,
) (*pbc.GetRecommendationsResponse, error) {
    start := time.Now()
    traceID := p.getTraceID(ctx)

    if req == nil {
        err := p.newErrorWithID(traceID, codes.InvalidArgument,
            "missing request", pbc.ErrorCode_ERROR_CODE_INVALID_RESOURCE)
        p.logErrorWithID(traceID, "GetRecommendations", err, pbc.ErrorCode_ERROR_CODE_INVALID_RESOURCE)
        return nil, err
    }

    // Placeholder - returns empty recommendations
    recommendations := []*pbc.Recommendation{}

    p.logger.Info().
        Str(pluginsdk.FieldTraceID, traceID).
        Str(pluginsdk.FieldOperation, "GetRecommendations").
        Int("recommendation_count", len(recommendations)).
        Int64(pluginsdk.FieldDurationMs, time.Since(start).Milliseconds()).
        Msg("recommendations generated")

    return &pbc.GetRecommendationsResponse{
        Recommendations: recommendations,
        Summary:         pluginsdk.CalculateRecommendationSummary(recommendations, "monthly"),
    }, nil
}
```

**Verify**: Interface compliance via `var _ pluginsdk.RecommendationsProvider = (*AWSPublicPlugin)(nil)`

---

### T1.3: Wire resource dispatchers to GetRecommendations

**File**: `internal/plugin/recommendations.go`
**Depends on**: T1.2
**Acceptance**: GetRecommendations dispatches to EC2/EBS handlers based on resource_type

Update the GetRecommendations method to dispatch based on resource type:

```go
// Inside GetRecommendations, after nil check:
var recommendations []*pbc.Recommendation

// Extract resource info from filter or request context
// For now, plugin returns empty (no resource inventory)
// Future: Extract from req.Filter.Metadata if provided

// Dispatch based on detected service type
// service := detectService(resource.ResourceType)
// switch service {
// case "ec2":
//     recs := p.generateEC2Recommendations(resource.Sku, resource.Region)
//     recommendations = append(recommendations, recs...)
// case "ebs":
//     recs := p.getEBSRecommendations(resource.Sku, resource.Region, resource.Tags)
//     recommendations = append(recommendations, recs...)
// }
```

**Note**: Full dispatcher implementation depends on how FinFocus core passes
resource context. Current skeleton returns empty list per research.md R7.

---

## Phase 2: US1 - EC2 Instance Generation Upgrades (P1)

**User Story**: As an infrastructure engineer, I want to receive recommendations when my EC2 instances use older generation types so that I can modernize to newer, more cost-effective instances.

### T2.1: Implement generateEC2Recommendations dispatcher

**File**: `internal/plugin/recommendations.go`
**Depends on**: T1.2
**Acceptance**: Method compiles, dispatches to generation and graviton helpers

```go
// generateEC2Recommendations creates recommendations for an EC2 instance.
func (p *AWSPublicPlugin) generateEC2Recommendations(
    instanceType, region string,
) []*pbc.Recommendation {
    var recommendations []*pbc.Recommendation

    // Generation upgrade
    if rec := p.getGenerationUpgradeRecommendation(instanceType, region); rec != nil {
        recommendations = append(recommendations, rec)
    }

    // Graviton migration (added in Phase 3)
    if rec := p.getGravitonRecommendation(instanceType, region); rec != nil {
        recommendations = append(recommendations, rec)
    }

    return recommendations
}
```

---

### T2.2: Implement getGenerationUpgradeRecommendation

**File**: `internal/plugin/recommendations.go`
**Depends on**: T2.1
**Acceptance**: Returns valid `*pbc.Recommendation` or nil

Implements FR-002, FR-005, FR-006, FR-011 from spec.md.

```go
func (p *AWSPublicPlugin) getGenerationUpgradeRecommendation(
    instanceType, region string,
) *pbc.Recommendation {
    family, size := parseInstanceType(instanceType)
    if family == "" {
        return nil
    }

    newFamily, exists := generationUpgradeMap[family]
    if !exists {
        return nil
    }

    newType := newFamily + "." + size

    currentPrice, found := p.pricing.EC2OnDemandPricePerHour(instanceType, "Linux", "Shared")
    if !found {
        return nil
    }

    newPrice, found := p.pricing.EC2OnDemandPricePerHour(newType, "Linux", "Shared")
    if !found || newPrice > currentPrice {
        return nil  // FR-011: Only recommend when new price <= current
    }

    currentMonthly := currentPrice * hoursPerMonth
    newMonthly := newPrice * hoursPerMonth
    savings := currentMonthly - newMonthly
    savingsPercent := (savings / currentMonthly) * 100

    confidence := confidenceHigh
    return &pbc.Recommendation{
        Id:         uuid.New().String(),
        Category:   pbc.RecommendationCategory_RECOMMENDATION_CATEGORY_COST_OPTIMIZATION,
        ActionType: pbc.RecommendationActionType_RECOMMENDATION_ACTION_TYPE_MODIFY,
        Resource: &pbc.ResourceRecommendationInfo{
            Provider:     "aws",
            ResourceType: "ec2",
            Region:       region,
            Sku:          instanceType,
        },
        Modify: &pbc.ModifyAction{
            ModificationType:  modTypeGenUpgrade,
            CurrentConfig:     map[string]string{"instance_type": instanceType},
            RecommendedConfig: map[string]string{"instance_type": newType},
        },
        Impact: &pbc.RecommendationImpact{
            EstimatedSavings:  savings,
            Currency:          "USD",
            ProjectionPeriod:  "monthly",
            CurrentCost:       currentMonthly,
            ProjectedCost:     newMonthly,
            SavingsPercentage: savingsPercent,
        },
        Priority:        pbc.RecommendationPriority_RECOMMENDATION_PRIORITY_MEDIUM,
        ConfidenceScore: &confidence,
        Description:     fmt.Sprintf("Upgrade from %s to %s for better performance at same or lower cost", instanceType, newType),
        Reasoning: []string{
            fmt.Sprintf("Newer %s instances offer better performance", newFamily),
            "Drop-in replacement with no architecture changes required",
        },
        Source: sourceAWSPublic,
    }
}
```

**Note**: Add `"github.com/google/uuid"` and `"fmt"` imports.

---

### T2.3: Add unit tests for generation upgrade

**File**: `internal/plugin/recommendations_test.go`
**Depends on**: T2.2
**Acceptance**: `make test` passes

Test scenarios from spec.md acceptance criteria:

1. t2.medium → t3.medium recommendation returned
2. Latest generation (t3a.micro) → no recommendation
3. New generation more expensive → no recommendation (FR-011)

```go
func TestGetGenerationUpgradeRecommendation(t *testing.T) {
    tests := []struct {
        name         string
        instanceType string
        wantUpgrade  bool
        wantNewType  string
    }{
        {"t2.medium upgrades to t3.medium", "t2.medium", true, "t3.medium"},
        {"t3a.micro has no upgrade", "t3a.micro", false, ""},
        {"m5.large upgrades to m6i.large", "m5.large", true, "m6i.large"},
        {"invalid type returns nil", "invalid", false, ""},
    }
    // ... test implementation
}
```

---

### T2.4: Add instance_type_test.go

**File**: `internal/plugin/instance_type_test.go`
**Depends on**: T1.1
**Acceptance**: `make test` passes

```go
func TestParseInstanceType(t *testing.T) {
    tests := []struct {
        input      string
        wantFamily string
        wantSize   string
    }{
        {"t2.medium", "t2", "medium"},
        {"m5.xlarge", "m5", "xlarge"},
        {"c6i.2xlarge", "c6i", "2xlarge"},
        {"invalid", "", ""},
        {"", "", ""},
    }

    for _, tt := range tests {
        t.Run(tt.input, func(t *testing.T) {
            family, size := parseInstanceType(tt.input)
            if family != tt.wantFamily || size != tt.wantSize {
                t.Errorf("parseInstanceType(%q) = (%q, %q), want (%q, %q)",
                    tt.input, family, size, tt.wantFamily, tt.wantSize)
            }
        })
    }
}
```

---

## Phase 3: US2 - Graviton/ARM Migration Suggestions (P2)

**User Story**: As an infrastructure engineer, I want to receive recommendations for migrating x86 instances to ARM-based Graviton instances so that I can evaluate potential cost savings.

### T3.1: Implement getGravitonRecommendation

**File**: `internal/plugin/recommendations.go`
**Depends on**: T2.1
**Acceptance**: Returns recommendation with 0.7 confidence, includes architecture warning

Implements FR-003, FR-007, FR-012 from spec.md.

```go
func (p *AWSPublicPlugin) getGravitonRecommendation(
    instanceType, region string,
) *pbc.Recommendation {
    family, size := parseInstanceType(instanceType)
    if family == "" {
        return nil
    }

    gravitonFamily, exists := gravitonMap[family]
    if !exists {
        return nil
    }

    gravitonType := gravitonFamily + "." + size

    currentPrice, found := p.pricing.EC2OnDemandPricePerHour(instanceType, "Linux", "Shared")
    if !found {
        return nil
    }

    gravitonPrice, found := p.pricing.EC2OnDemandPricePerHour(gravitonType, "Linux", "Shared")
    if !found || gravitonPrice > currentPrice {
        return nil
    }

    currentMonthly := currentPrice * hoursPerMonth
    gravitonMonthly := gravitonPrice * hoursPerMonth
    savings := currentMonthly - gravitonMonthly
    savingsPercent := (savings / currentMonthly) * 100

    confidence := confidenceMedium  // 0.7 per FR-007
    return &pbc.Recommendation{
        Id:         uuid.New().String(),
        Category:   pbc.RecommendationCategory_RECOMMENDATION_CATEGORY_COST_OPTIMIZATION,
        ActionType: pbc.RecommendationActionType_RECOMMENDATION_ACTION_TYPE_MODIFY,
        Resource: &pbc.ResourceRecommendationInfo{
            Provider:     "aws",
            ResourceType: "ec2",
            Region:       region,
            Sku:          instanceType,
        },
        Modify: &pbc.ModifyAction{
            ModificationType:  modTypeGraviton,
            CurrentConfig:     map[string]string{"instance_type": instanceType, "architecture": "x86_64"},
            RecommendedConfig: map[string]string{"instance_type": gravitonType, "architecture": "arm64"},
        },
        Impact: &pbc.RecommendationImpact{
            EstimatedSavings:  savings,
            Currency:          "USD",
            ProjectionPeriod:  "monthly",
            CurrentCost:       currentMonthly,
            ProjectedCost:     gravitonMonthly,
            SavingsPercentage: savingsPercent,
        },
        Priority:        pbc.RecommendationPriority_RECOMMENDATION_PRIORITY_LOW,
        ConfidenceScore: &confidence,
        Description:     fmt.Sprintf("Migrate from %s to %s (Graviton) for ~%.0f%% cost savings", instanceType, gravitonType, savingsPercent),
        Reasoning: []string{
            "Graviton instances are typically ~20% cheaper with comparable performance",
            "Requires validation that application supports ARM architecture",
        },
        Metadata: map[string]string{
            "architecture_change":  "x86_64 -> arm64",
            "requires_validation": "Application must support ARM architecture",
        },
        Source: sourceAWSPublic,
    }
}
```

---

### T3.2: Add unit tests for Graviton recommendation

**File**: `internal/plugin/recommendations_test.go`
**Depends on**: T3.1
**Acceptance**: `make test` passes, metadata verified

Implements SC-003 from spec.md.

Test scenarios:

1. m5.large → m6g.large with 0.7 confidence
2. Architecture warning in metadata (`Metadata["architecture_change"]` = "x86_64 -> arm64")
3. Validation warning in metadata (`Metadata["requires_validation"]` present)
4. No Graviton equivalent (mac1.metal) → no recommendation

```go
func TestGetGravitonRecommendation(t *testing.T) {
    tests := []struct {
        name             string
        instanceType     string
        wantRecommend    bool
        wantGravitonType string
        wantConfidence   float64
    }{
        {"m5.large to m6g.large", "m5.large", true, "m6g.large", 0.7},
        {"t3.micro to t4g.micro", "t3.micro", true, "t4g.micro", 0.7},
        {"no graviton for mac1.metal", "mac1.metal", false, "", 0},
    }
    // ... test implementation

    // For successful recommendations, verify metadata:
    // assert.Equal(t, "x86_64 -> arm64", rec.Metadata["architecture_change"])
    // assert.Contains(t, rec.Metadata["requires_validation"], "ARM")
}
```

---

## Phase 4: US3 - EBS Volume Type Upgrade (P3)

**User Story**: As an infrastructure engineer, I want to receive recommendations to migrate gp2 volumes to gp3 so that I can reduce storage costs while improving baseline performance.

### T4.1: Implement getEBSRecommendations

**File**: `internal/plugin/recommendations.go`
**Depends on**: T1.2
**Acceptance**: Returns gp2→gp3 recommendation with size handling

Implements FR-004, FR-006 from spec.md.

```go
func (p *AWSPublicPlugin) getEBSRecommendations(
    volumeType, region string,
    tags map[string]string,
) []*pbc.Recommendation {
    // Only recommend for gp2 volumes
    if volumeType != "gp2" {
        return nil
    }

    // Extract size from tags, default 100GB
    sizeGB := 100
    if sizeStr, ok := tags["size"]; ok {
        if parsed, err := strconv.Atoi(sizeStr); err == nil && parsed > 0 {
            sizeGB = parsed
        }
    } else if sizeStr, ok := tags["volume_size"]; ok {
        if parsed, err := strconv.Atoi(sizeStr); err == nil && parsed > 0 {
            sizeGB = parsed
        }
    }

    gp2Price, found := p.pricing.EBSPricePerGBMonth("gp2")
    if !found {
        return nil
    }

    gp3Price, found := p.pricing.EBSPricePerGBMonth("gp3")
    if !found || gp3Price > gp2Price {
        return nil
    }

    currentMonthly := gp2Price * float64(sizeGB)
    gp3Monthly := gp3Price * float64(sizeGB)
    savings := currentMonthly - gp3Monthly
    savingsPercent := (savings / currentMonthly) * 100

    confidence := confidenceHigh
    return []*pbc.Recommendation{{
        Id:         uuid.New().String(),
        Category:   pbc.RecommendationCategory_RECOMMENDATION_CATEGORY_COST_OPTIMIZATION,
        ActionType: pbc.RecommendationActionType_RECOMMENDATION_ACTION_TYPE_MODIFY,
        Resource: &pbc.ResourceRecommendationInfo{
            Provider:     "aws",
            ResourceType: "ebs",
            Region:       region,
            Sku:          volumeType,
        },
        Modify: &pbc.ModifyAction{
            ModificationType:  modTypeVolumeUpgrade,
            CurrentConfig:     map[string]string{"volume_type": "gp2", "size_gb": strconv.Itoa(sizeGB)},
            RecommendedConfig: map[string]string{"volume_type": "gp3", "size_gb": strconv.Itoa(sizeGB)},
        },
        Impact: &pbc.RecommendationImpact{
            EstimatedSavings:  savings,
            Currency:          "USD",
            ProjectionPeriod:  "monthly",
            CurrentCost:       currentMonthly,
            ProjectedCost:     gp3Monthly,
            SavingsPercentage: savingsPercent,
        },
        Priority:        pbc.RecommendationPriority_RECOMMENDATION_PRIORITY_MEDIUM,
        ConfidenceScore: &confidence,
        Description:     fmt.Sprintf("Upgrade %dGB gp2 volume to gp3 for ~%.0f%% cost savings", sizeGB, savingsPercent),
        Reasoning: []string{
            "gp3 volumes are ~20% cheaper than gp2",
            "gp3 provides better baseline performance (3000 IOPS, 125 MB/s)",
            "API-compatible change with no data migration required",
        },
        Metadata: map[string]string{
            "baseline_iops":       "gp2: 100 IOPS/GB, gp3: 3000 IOPS (included)",
            "baseline_throughput": "gp2: 128-250 MB/s, gp3: 125 MB/s (included)",
        },
        Source: sourceAWSPublic,
    }}
}
```

**Note**: Add `"strconv"` import.

---

### T4.2: Add unit tests for EBS recommendations

**File**: `internal/plugin/recommendations_test.go`
**Depends on**: T4.1
**Acceptance**: `make test` passes, metadata verified

Test scenarios:

1. gp2 100GB → gp3 recommendation with savings
2. gp3 volume → no recommendation
3. io1/io2 volume → no recommendation
4. Missing size tag → uses 100GB default
5. Performance metadata present (`Metadata["baseline_iops"]`, `Metadata["baseline_throughput"]`)

```go
func TestGetEBSRecommendations(t *testing.T) {
    // ... test implementation

    // For gp2 recommendations, verify metadata:
    // assert.Contains(t, rec.Metadata["baseline_iops"], "gp3: 3000 IOPS")
    // assert.Contains(t, rec.Metadata["baseline_throughput"], "gp3: 125 MB/s")
}
```

---

## Phase 5: US4 - Graceful Error Handling (P4)

**User Story**: As a FinFocus user, I want GetRecommendations to return an empty list for unsupported services so that my automation doesn't break.

### T5.1: Verify empty list for unsupported services

**File**: `internal/plugin/recommendations_test.go`
**Depends on**: T1.2
**Acceptance**: `make test` passes

Implements FR-008 from spec.md.

Test scenarios:

1. S3 resource → empty list, no error
2. Lambda resource → empty list, no error
3. RDS resource → empty list, no error
4. DynamoDB resource → empty list, no error

```go
func TestGetRecommendations_UnsupportedServices(t *testing.T) {
    unsupportedTypes := []string{"s3", "lambda", "rds", "dynamodb"}
    for _, resourceType := range unsupportedTypes {
        t.Run(resourceType, func(t *testing.T) {
            // Call GetRecommendations
            // Assert empty list returned, no error
        })
    }
}
```

---

### T5.2: Verify ERROR_CODE_INVALID_RESOURCE for nil request

**File**: `internal/plugin/recommendations_test.go`
**Depends on**: T1.2
**Acceptance**: `make test` passes

Implements FR-009 from spec.md.

```go
func TestGetRecommendations_NilRequest(t *testing.T) {
    plugin := NewAWSPublicPlugin(/* ... */)
    resp, err := plugin.GetRecommendations(context.Background(), nil)

    // Assert err is not nil
    // Assert error contains ERROR_CODE_INVALID_RESOURCE
    // Assert resp is nil
}
```

---

### T5.3: Verify trace_id propagation in logs

**File**: `internal/plugin/recommendations_test.go`
**Depends on**: T1.2
**Acceptance**: `make test` passes

Implements FR-010, SC-007 from spec.md.

```go
func TestGetRecommendations_TraceIDLogging(t *testing.T) {
    // Capture log output
    var logBuf bytes.Buffer
    logger := zerolog.New(&logBuf)

    plugin := NewAWSPublicPluginWithLogger(/* ... */, logger)

    // Create context with trace_id metadata
    md := metadata.New(map[string]string{
        pluginsdk.TraceIDMetadataKey: "test-trace-123",
    })
    ctx := metadata.NewOutgoingContext(context.Background(), md)

    _, _ = plugin.GetRecommendations(ctx, &pbc.GetRecommendationsRequest{})

    // Verify trace_id appears in log output
    logOutput := logBuf.String()
    assert.Contains(t, logOutput, "test-trace-123")
    assert.Contains(t, logOutput, "trace_id")
}
```

**Note**: May require exposing logger injection in plugin constructor for testability.

---

## Phase 6: Polish

### T6.1: Run make lint and fix issues

**Depends on**: All previous tasks
**Acceptance**: `make lint` passes with zero errors

Common issues to check:

- gofmt formatting
- golangci-lint rules
- Exported function documentation

---

### T6.2: Run make test and verify coverage

**Depends on**: T6.1
**Acceptance**: `make test` passes, coverage report shows new code covered

Verify all acceptance scenarios from spec.md pass:

- [ ] SC-001: GetRecommendations returns valid proto response within 100ms
- [ ] SC-002: 100% of generation upgrades result in equal or lower cost
- [ ] SC-003: 100% of Graviton recommendations include architecture warning
- [ ] SC-004: All recommendations include accurate monthly savings
- [ ] SC-005: Zero false positives (no recs where new > current)
- [ ] SC-006: Empty list for unsupported services
- [ ] SC-007: All log entries include trace_id

---

## Checklist

### Before Starting

- [ ] Read spec.md user stories and acceptance criteria
- [ ] Read data-model.md for mapping definitions
- [ ] Understand existing plugin.go structure
- [ ] Verify finfocus-spec v0.4.7 dependency

### After Completion

- [ ] `make lint` passes
- [ ] `make test` passes
- [ ] `go build ./...` passes
- [ ] All acceptance scenarios verified
- [ ] PR_MESSAGE.md generated with conventional commit
