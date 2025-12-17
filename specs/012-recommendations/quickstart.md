# Quickstart: GetRecommendations Implementation

**Date**: 2025-12-15
**Feature**: 012-recommendations

## Overview

This guide provides a quick implementation path for the GetRecommendations RPC.

## Step 1: Create instance_type.go

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
    "t2": "t3", "t3": "t3a",
    "m4": "m5", "m5": "m6i", "m5a": "m6a",
    "c4": "c5", "c5": "c6i", "c5a": "c6a",
    "r4": "r5", "r5": "r6i", "r5a": "r6a",
    "i3": "i3en", "d2": "d3",
}

// gravitonMap maps x86 families to Graviton equivalents.
var gravitonMap = map[string]string{
    "m5": "m6g", "m5a": "m6g", "m6i": "m6g", "m6a": "m6g",
    "c5": "c6g", "c5a": "c6g", "c6i": "c6g", "c6a": "c6g",
    "r5": "r6g", "r5a": "r6g", "r6i": "r6g", "r6a": "r6g",
    "t3": "t4g", "t3a": "t4g",
}
```

## Step 2: Create recommendations.go

```go
// internal/plugin/recommendations.go
package plugin

import (
    "context"
    "fmt"
    "strconv"
    "time"

    "github.com/google/uuid"
    "github.com/rshade/pulumicost-spec/sdk/go/pluginsdk"
    pbc "github.com/rshade/pulumicost-spec/sdk/go/proto/pulumicost/v1"
    "google.golang.org/grpc/codes"
)

const (
    hoursPerMonth         = 730.0
    confidenceHigh        = 0.9
    confidenceMedium      = 0.7
    sourceAWSPublic       = "aws-public"
    modTypeGenUpgrade     = "generation_upgrade"
    modTypeGraviton       = "graviton_migration"
    modTypeVolumeUpgrade  = "volume_type_upgrade"
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

    // For now, return empty recommendations (no resource inventory)
    // Future: Extract resource from filter metadata
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

// generateEC2Recommendations creates recommendations for an EC2 instance.
func (p *AWSPublicPlugin) generateEC2Recommendations(
    instanceType, region string,
) []*pbc.Recommendation {
    var recommendations []*pbc.Recommendation

    // Generation upgrade
    if rec := p.getGenerationUpgradeRecommendation(instanceType, region); rec != nil {
        recommendations = append(recommendations, rec)
    }

    // Graviton migration
    if rec := p.getGravitonRecommendation(instanceType, region); rec != nil {
        recommendations = append(recommendations, rec)
    }

    return recommendations
}

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
        return nil
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
            ModificationType: modTypeGenUpgrade,
            CurrentConfig:    map[string]string{"instance_type": instanceType},
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

// Similar pattern for getGravitonRecommendation and getEBSRecommendations...
```

## Step 3: Add Tests

```go
// internal/plugin/instance_type_test.go
package plugin

import "testing"

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
        family, size := parseInstanceType(tt.input)
        if family != tt.wantFamily || size != tt.wantSize {
            t.Errorf("parseInstanceType(%q) = (%q, %q), want (%q, %q)",
                tt.input, family, size, tt.wantFamily, tt.wantSize)
        }
    }
}
```

## Step 4: Verify Interface Compliance

```bash
# Build to verify interface implementation
go build ./...

# Run tests
make test
```

## Key Implementation Notes

1. **Thread Safety**: All methods are stateless (use `p.pricing` which is thread-safe)
2. **Trace ID**: Always extract via `p.getTraceID(ctx)` for log correlation
3. **Validation**: Use `pluginsdk.ValidateRecommendation()` before returning
4. **Pagination**: Use `pluginsdk.PaginateRecommendations()` if list > pageSize
5. **Summary**: Always include via `pluginsdk.CalculateRecommendationSummary()`

## Testing Checklist

- [ ] `parseInstanceType` handles edge cases
- [ ] Generation upgrade returns nil when new type not found
- [ ] Generation upgrade returns nil when new price > current
- [ ] Graviton recommendation has 0.7 confidence
- [ ] EBS recommendation only for gp2 volumes
- [ ] Empty list for unsupported services (no error)
- [ ] ERROR_CODE_INVALID_RESOURCE for nil request
