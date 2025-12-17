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
	// confidenceHigh is used for generation upgrades and EBS changes (FR-006).
	confidenceHigh = 0.9
	// confidenceMedium is used for Graviton recommendations (FR-007).
	confidenceMedium = 0.7
	// sourceAWSPublic identifies recommendations from this plugin.
	sourceAWSPublic = "aws-public"
	// modTypeGenUpgrade is the modification type for generation upgrades.
	modTypeGenUpgrade = "generation_upgrade"
	// modTypeGraviton is the modification type for Graviton migrations.
	modTypeGraviton = "graviton_migration"
	// modTypeVolumeUpgrade is the modification type for EBS volume upgrades.
	modTypeVolumeUpgrade = "volume_type_upgrade"
	// defaultEBSVolumeGB is the default volume size when not specified in tags.
	defaultEBSVolumeGB = 100
)

// Ensure AWSPublicPlugin implements RecommendationsProvider.
var _ pluginsdk.RecommendationsProvider = (*AWSPublicPlugin)(nil)

// GetRecommendations returns cost optimization recommendations.
// Implements FR-001 from spec.md.
func (p *AWSPublicPlugin) GetRecommendations(
	ctx context.Context,
	req *pbc.GetRecommendationsRequest,
) (*pbc.GetRecommendationsResponse, error) {
	start := time.Now()
	traceID := p.getTraceID(ctx)

	// FR-009: Return ERROR_CODE_INVALID_RESOURCE when request is nil
	if req == nil {
		err := p.newErrorWithID(traceID, codes.InvalidArgument,
			"missing request", pbc.ErrorCode_ERROR_CODE_INVALID_RESOURCE)
		p.logErrorWithID(traceID, "GetRecommendations", err, pbc.ErrorCode_ERROR_CODE_INVALID_RESOURCE)
		return nil, err
	}

	// Generate recommendations based on filter (pulumicost-spec v0.4.8+)
	// FR-008: Return empty list when no filter context provided
	var recommendations []*pbc.Recommendation

	if req.Filter != nil && req.Filter.Sku != "" {
		// Determine resource type from filter
		resourceType := req.Filter.ResourceType
		region := req.Filter.Region
		if region == "" {
			region = p.region // Default to plugin's region
		}

		// Normalize resource type using detectService
		service := detectService(resourceType)

		switch service {
		case "ec2":
			// FR-002, FR-003: Generate EC2 generation upgrade and Graviton recommendations
			recommendations = append(recommendations, p.generateEC2Recommendations(req.Filter.Sku, region)...)
		case "ebs":
			// FR-004: Generate EBS volume type recommendations
			recommendations = append(recommendations, p.getEBSRecommendations(req.Filter.Sku, region, req.Filter.Tags)...)
		}
	}

	// FR-010: Include trace_id in all log entries
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
// Returns up to 2 recommendations: generation upgrade and/or Graviton migration.
func (p *AWSPublicPlugin) generateEC2Recommendations(
	instanceType, region string,
) []*pbc.Recommendation {
	var recommendations []*pbc.Recommendation

	// Generation upgrade (FR-002)
	if rec := p.getGenerationUpgradeRecommendation(instanceType, region); rec != nil {
		recommendations = append(recommendations, rec)
	}

	// Graviton migration (FR-003)
	if rec := p.getGravitonRecommendation(instanceType, region); rec != nil {
		recommendations = append(recommendations, rec)
	}

	return recommendations
}

// getGenerationUpgradeRecommendation returns a recommendation to upgrade to a newer
// EC2 instance generation if available and cost-effective.
// Implements FR-002, FR-005, FR-006, FR-011 from spec.md.
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
	// FR-011: Only recommend when new price <= current price
	if !found || newPrice > currentPrice {
		return nil
	}

	// FR-005: Calculate monthly savings based on 730 hours/month
	currentMonthly := currentPrice * hoursPerMonth
	newMonthly := newPrice * hoursPerMonth
	savings := currentMonthly - newMonthly
	savingsPercent := 0.0
	if currentMonthly > 0 {
		savingsPercent = (savings / currentMonthly) * 100
	}

	// FR-006: Set confidence level to 0.9 (high) for generation upgrades
	confidence := confidenceHigh

	// Build reasoning with optional Graviton alternative note
	reasoning := []string{
		fmt.Sprintf("Newer %s instances offer better performance", newFamily),
		"Drop-in replacement with no architecture changes required",
	}

	// Check if there's a Graviton alternative for the recommended family
	if gravitonFamily, hasGraviton := gravitonMap[newFamily]; hasGraviton {
		gravitonType := gravitonFamily + "." + size
		reasoning = append(reasoning,
			fmt.Sprintf("Alternative: consider %s for ARM compatibility (~20%% additional savings)", gravitonType))
	}

	return &pbc.Recommendation{
		Id:         uuid.New().String(),
		Category:   pbc.RecommendationCategory_RECOMMENDATION_CATEGORY_COST,
		ActionType: pbc.RecommendationActionType_RECOMMENDATION_ACTION_TYPE_MODIFY,
		Resource: &pbc.ResourceRecommendationInfo{
			Provider:     "aws",
			ResourceType: "ec2",
			Region:       region,
			Sku:          instanceType,
		},
		ActionDetail: &pbc.Recommendation_Modify{
			Modify: &pbc.ModifyAction{
				ModificationType:  modTypeGenUpgrade,
				CurrentConfig:     map[string]string{"instance_type": instanceType},
				RecommendedConfig: map[string]string{"instance_type": newType},
			},
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
		Description: fmt.Sprintf("Upgrade from %s to %s for better performance at same or lower cost",
			instanceType, newType),
		Reasoning: reasoning,
		Source:    sourceAWSPublic,
	}
}

// getGravitonRecommendation returns a recommendation to migrate to ARM-based
// Graviton instances if available and cost-effective.
// Implements FR-003, FR-007, FR-012 from spec.md.
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
	// FR-011: Only recommend when new price <= current price
	if !found || gravitonPrice > currentPrice {
		return nil
	}

	// FR-005: Calculate monthly savings based on 730 hours/month
	currentMonthly := currentPrice * hoursPerMonth
	gravitonMonthly := gravitonPrice * hoursPerMonth
	savings := currentMonthly - gravitonMonthly
	savingsPercent := 0.0
	if currentMonthly > 0 {
		savingsPercent = (savings / currentMonthly) * 100
	}

	// FR-007: Set confidence level to 0.7 (medium) for Graviton recommendations
	confidence := confidenceMedium
	return &pbc.Recommendation{
		Id:         uuid.New().String(),
		Category:   pbc.RecommendationCategory_RECOMMENDATION_CATEGORY_COST,
		ActionType: pbc.RecommendationActionType_RECOMMENDATION_ACTION_TYPE_MODIFY,
		Resource: &pbc.ResourceRecommendationInfo{
			Provider:     "aws",
			ResourceType: "ec2",
			Region:       region,
			Sku:          instanceType,
		},
		ActionDetail: &pbc.Recommendation_Modify{
			Modify: &pbc.ModifyAction{
				ModificationType:  modTypeGraviton,
				CurrentConfig:     map[string]string{"instance_type": instanceType, "architecture": "x86_64"},
				RecommendedConfig: map[string]string{"instance_type": gravitonType, "architecture": "arm64"},
			},
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
		Description: fmt.Sprintf("Migrate from %s to %s (Graviton) for ~%.0f%% cost savings",
			instanceType, gravitonType, savingsPercent),
		Reasoning: []string{
			"Graviton instances are typically ~20% cheaper with comparable performance",
			"Requires validation that application supports ARM architecture",
		},
		// FR-012: Include relevant metadata (architecture warnings)
		Metadata: map[string]string{
			"architecture_change":  "x86_64 -> arm64",
			"requires_validation": "Application must support ARM architecture",
		},
		Source: sourceAWSPublic,
	}
}

// getEBSRecommendations returns recommendations for EBS volume optimization.
// Currently supports gp2 to gp3 migration.
// Implements FR-004, FR-006 from spec.md.
func (p *AWSPublicPlugin) getEBSRecommendations(
	volumeType, region string,
	tags map[string]string,
) []*pbc.Recommendation {
	// Only recommend for gp2 volumes
	if volumeType != "gp2" {
		return nil
	}

	// Extract size from tags, default to defaultEBSVolumeGB per edge case spec
	sizeGB := defaultEBSVolumeGB
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
	// FR-011: Only recommend when new price <= current price
	if !found || gp3Price > gp2Price {
		return nil
	}

	currentMonthly := gp2Price * float64(sizeGB)
	gp3Monthly := gp3Price * float64(sizeGB)
	savings := currentMonthly - gp3Monthly
	savingsPercent := 0.0
	if currentMonthly > 0 {
		savingsPercent = (savings / currentMonthly) * 100
	}

	// FR-006: Set confidence level to 0.9 (high) for EBS volume changes
	confidence := confidenceHigh
	return []*pbc.Recommendation{{
		Id:         uuid.New().String(),
		Category:   pbc.RecommendationCategory_RECOMMENDATION_CATEGORY_COST,
		ActionType: pbc.RecommendationActionType_RECOMMENDATION_ACTION_TYPE_MODIFY,
		Resource: &pbc.ResourceRecommendationInfo{
			Provider:     "aws",
			ResourceType: "ebs",
			Region:       region,
			Sku:          volumeType,
		},
		ActionDetail: &pbc.Recommendation_Modify{
			Modify: &pbc.ModifyAction{
				ModificationType:  modTypeVolumeUpgrade,
				CurrentConfig:     map[string]string{"volume_type": "gp2", "size_gb": strconv.Itoa(sizeGB)},
				RecommendedConfig: map[string]string{"volume_type": "gp3", "size_gb": strconv.Itoa(sizeGB)},
			},
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
		// FR-012: Include relevant metadata (performance info)
		Metadata: map[string]string{
			"baseline_iops":       "gp2: 100 IOPS/GB, gp3: 3000 IOPS (included)",
			"baseline_throughput": "gp2: 128-250 MB/s, gp3: 125 MB/s (included)",
		},
		Source: sourceAWSPublic,
	}}
}
