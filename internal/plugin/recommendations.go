package plugin

import (
	"context"
	"fmt"
	"maps"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rshade/finfocus-plugin-aws-public/internal/carbon"
	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/proto"
)

const (
	// confidenceHigh is used for generation upgrades and EBS changes (FR-006).
	confidenceHigh = 0.9
	// confidenceMedium is used for Graviton migrations (FR-007).
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
	// defaultMaxBatchSize is the default maximum number of resources to process in GetRecommendations
	defaultMaxBatchSize = 100
	// maxMaxBatchSize is the absolute maximum allowed batch size to prevent OOM/abuse
	maxMaxBatchSize = 500
	// EnvMaxBatchSize is the environment variable to override defaultMaxBatchSize
	EnvMaxBatchSize = "FINFOCUS_MAX_BATCH_SIZE"
	// EnvMaxBatchSizeDeprecated is the deprecated environment variable for backward compatibility
	EnvMaxBatchSizeDeprecated = "PULUMICOST_MAX_BATCH_SIZE"
	// EnvMaxBatchSizeLegacy is the legacy environment variable for additional backward compatibility
	EnvMaxBatchSizeLegacy = "MAX_BATCH_SIZE"
	// EnvStrictValidation is the environment variable to enable fail-fast validation
	EnvStrictValidation = "FINFOCUS_STRICT_VALIDATION"
	// EnvStrictValidationDeprecated is the deprecated environment variable for backward compatibility
	EnvStrictValidationDeprecated = "PULUMICOST_STRICT_VALIDATION"
	// EnvStrictValidationLegacy is the legacy environment variable for additional backward compatibility
	EnvStrictValidationLegacy = "STRICT_VALIDATION"
)

// Ensure AWSPublicPlugin implements RecommendationsProvider.
var _ pluginsdk.RecommendationsProvider = (*AWSPublicPlugin)(nil)

// ProcessingContext holds state during batch request processing.
type ProcessingContext struct {
	Scope      []*pbc.ResourceDescriptor
	Filter     *pbc.RecommendationFilter
	BatchStats BatchStats
}

// BatchStats tracks aggregation metrics for logging.
type BatchStats struct {
	TotalResources   int
	MatchedResources int
	TotalSavings     float64
}

// GetRecommendations generates cost optimization recommendations for the requested resources.
// It supports batch processing of resources provided in the target_resources field.
// For each matching resource, it populates correlation info (Id and Name) in the recommendation
// object by extracting the "resource_id" and "name" tags from the input ResourceDescriptor.
// This allows the caller to correlate recommendations back to their infrastructure definitions.
func (p *AWSPublicPlugin) GetRecommendations(ctx context.Context, req *pbc.GetRecommendationsRequest) (*pbc.GetRecommendationsResponse, error) {
	start := time.Now()
	traceID := p.getTraceID(ctx)

	// FR-009: Return ERROR_CODE_INVALID_RESOURCE when request is nil
	if req == nil {
		err := p.newErrorWithID(traceID, codes.InvalidArgument,
			"missing request", pbc.ErrorCode_ERROR_CODE_INVALID_RESOURCE)
		p.logErrorWithID(traceID, "GetRecommendations", err, pbc.ErrorCode_ERROR_CODE_INVALID_RESOURCE)
		return nil, err
	}

	// Validate batch size (max 100 resources per request)
	if len(req.TargetResources) > p.maxBatchSize {
		err := p.newErrorWithID(traceID, codes.InvalidArgument,
			fmt.Sprintf("batch size %d exceeds maximum of %d", len(req.TargetResources), p.maxBatchSize),
			pbc.ErrorCode_ERROR_CODE_INVALID_RESOURCE)
		p.logErrorWithID(traceID, "GetRecommendations", err, pbc.ErrorCode_ERROR_CODE_INVALID_RESOURCE)
		return nil, err
	}

	// Normalize input into ProcessingContext (T006)
	pctx := p.normalizeInput(req)

	// Generate recommendations by iterating over scope (T007)
	var recommendations []*pbc.Recommendation
	var skippedCount int
	for _, resource := range pctx.Scope {
		// Provider check: only process AWS resources (T011)
		if resource.Provider != "" && resource.Provider != providerAWS {
			skippedCount++
			p.logger.Debug().
				Str("trace_id", traceID).
				Str("provider", resource.Provider).
				Str("resource_type", resource.ResourceType).
				Str("reason", "non-AWS provider").
				Msg("skipping resource in recommendations batch")
			if p.strictValidation {
				err := p.newErrorWithID(traceID, codes.InvalidArgument,
					fmt.Sprintf("strict validation: unsupported provider %q (only %q supported)",
						resource.Provider, providerAWS),
					pbc.ErrorCode_ERROR_CODE_INVALID_RESOURCE)
				return nil, err
			}
			continue
		}

		// Apply filter criteria using AND logic (T010)
		if !p.matchesFilter(resource, pctx.Filter) {
			skippedCount++
			p.logger.Debug().
				Str("trace_id", traceID).
				Str("resource_type", resource.ResourceType).
				Str("sku", resource.Sku).
				Str("reason", "filter mismatch").
				Msg("skipping resource in recommendations batch")
			continue
		}

		pctx.BatchStats.MatchedResources++

		// Determine region (default to plugin's region if not specified)
		region := resource.Region
		if region == "" {
			region = p.region
		}

		// Generate recommendations based on resource type
		service := detectService(resource.ResourceType)
		var recs []*pbc.Recommendation

		switch service {
		case "ec2":
			recs = p.generateEC2Recommendations(resource.Sku, region)
		case "ebs":
			recs = p.getEBSRecommendations(resource.Sku, region, resource.Tags)
		case "rds":
			engine := extractRDSEngine(resource.Tags)
			recs = p.generateRDSRecommendations(resource.Sku, engine, region)
		default:
			// Log unsupported service types at debug level
			p.logger.Debug().
				Str("trace_id", traceID).
				Str("resource_type", resource.ResourceType).
				Str("detected_service", service).
				Str("reason", "unsupported service for recommendations").
				Msg("no recommendations generated for resource")
			if p.strictValidation {
				err := p.newErrorWithID(traceID, codes.InvalidArgument,
					fmt.Sprintf("strict validation: service %q does not support recommendations (resource_type: %s)",
						service, resource.ResourceType),
					pbc.ErrorCode_ERROR_CODE_INVALID_RESOURCE)
				return nil, err
			}
		}

		// Populate correlation info: Native Id takes priority over tag (FR-001, FR-002, FR-003)
		for _, rec := range recs {
			if rec.Resource != nil {
				// Priority 1: Use native Id field from ResourceDescriptor (FR-001, FR-002)
				if id := strings.TrimSpace(resource.Id); id != "" {
					rec.Resource.Id = id
					p.logger.Trace().
						Str(pluginsdk.FieldTraceID, traceID).
						Str("id_source", "native").
						Str("id", id).
						Msg("using native ID for recommendation correlation")
				} else if resourceID := resource.Tags["resource_id"]; resourceID != "" {
					// Priority 2: Fall back to resource_id tag for backward compat (FR-003)
					rec.Resource.Id = resourceID
					p.logger.Trace().
						Str(pluginsdk.FieldTraceID, traceID).
						Str("id_source", "tag").
						Str("id", resourceID).
						Msg("using tag ID for recommendation correlation")
				}
				// Use name tag if available (FR-004 - unchanged)
				if name := resource.Tags["name"]; name != "" {
					rec.Resource.Name = name
				}
			} else { // Handle missing resource impact logging (rec.Resource is nil here)
				p.logger.Warn().
					Str("recommendation_id", rec.Id).
					Msg("recommendation missing resource data")
			}

			if rec.Impact != nil {
				pctx.BatchStats.TotalSavings += rec.Impact.GetEstimatedSavings()
			} else {
				resourceSKU := ""
				if rec.Resource != nil {
					resourceSKU = rec.Resource.Sku
				}
				p.logger.Warn().
					Str("recommendation_id", rec.Id).
					Str("resource_sku", resourceSKU).
					Msg("recommendation missing impact data, skipping savings aggregation")
			}
		}

		recommendations = append(recommendations, recs...)
	}

	// FR-010: Summary logging (one line per batch, not per resource)
	p.traceLogger(traceID, "GetRecommendations").Info().
		Int("total_resources", pctx.BatchStats.TotalResources).
		Int("matched_resources", pctx.BatchStats.MatchedResources).
		Int("recommendation_count", len(recommendations)).
		Int("skipped_resources", skippedCount).
		Float64("total_savings", pctx.BatchStats.TotalSavings).
		Int64(pluginsdk.FieldDurationMs, time.Since(start).Milliseconds()).
		Msg("batch recommendations generated")

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
	currentMonthly := currentPrice * carbon.HoursPerMonth
	newMonthly := newPrice * carbon.HoursPerMonth
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
			Provider:     providerAWS,
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
	currentMonthly := currentPrice * carbon.HoursPerMonth
	gravitonMonthly := gravitonPrice * carbon.HoursPerMonth
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
			Provider:     providerAWS,
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
			"architecture_change": "x86_64 -> arm64",
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
			Provider:     providerAWS,
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

// extractRDSEngine gets the database engine from resource tags.
// Falls back to "mysql" if not specified (most common RDS engine).
// Normalizes engine names for consistent pricing lookup.
func extractRDSEngine(tags map[string]string) string {
	if tags == nil {
		return "mysql"
	}
	engine := tags["engine"]
	if engine == "" {
		engine = tags["Engine"] // Handle capitalization variants
	}
	if engine == "" {
		return "mysql"
	}
	return normalizeRDSEngine(engine)
}

// normalizeRDSEngine converts engine names to the format used in pricing lookups.
// Handles common aliases and capitalization variants.
func normalizeRDSEngine(engine string) string {
	switch strings.ToLower(strings.TrimSpace(engine)) {
	case "mysql", "mysql8", "mysql-8.0":
		return "mysql"
	case "postgres", "postgresql", "postgres13", "postgres14", "postgres15":
		return "postgresql"
	case "mariadb", "maria":
		return "mariadb"
	case "oracle", "oracle-ee", "oracle-se", "oracle-se1", "oracle-se2":
		return "oracle"
	case "sqlserver", "sql-server", "sqlserver-ee", "sqlserver-se", "sqlserver-ex", "sqlserver-web":
		return "sqlserver"
	case "aurora", "aurora-mysql":
		return "aurora-mysql"
	case "aurora-postgresql":
		return "aurora-postgresql"
	default:
		return strings.ToLower(engine)
	}
}

// generateRDSRecommendations creates recommendations for an RDS instance.
// Returns up to 2 recommendations: generation upgrade and/or Graviton migration.
// Graviton is only recommended for engines that support it (MySQL, PostgreSQL, MariaDB).
func (p *AWSPublicPlugin) generateRDSRecommendations(
	instanceType, engine, region string,
) []*pbc.Recommendation {
	var recommendations []*pbc.Recommendation

	// Generation upgrade
	if rec := p.getRDSGenerationUpgradeRecommendation(instanceType, engine, region); rec != nil {
		recommendations = append(recommendations, rec)
	}

	// Graviton migration (only for supported engines)
	if rdsGravitonSupportedEngines[strings.ToLower(engine)] {
		if rec := p.getRDSGravitonRecommendation(instanceType, engine, region); rec != nil {
			recommendations = append(recommendations, rec)
		}
	}

	return recommendations
}

// getRDSGenerationUpgradeRecommendation returns a recommendation to upgrade to a newer
// RDS instance generation if available and cost-effective.
func (p *AWSPublicPlugin) getRDSGenerationUpgradeRecommendation(
	instanceType, engine, region string,
) *pbc.Recommendation {
	family, size := parseRDSInstanceType(instanceType)
	if family == "" {
		return nil
	}

	newFamily, exists := rdsGenerationUpgradeMap[family]
	if !exists {
		return nil
	}

	newType := newFamily + "." + size

	currentPrice, found := p.pricing.RDSOnDemandPricePerHour(instanceType, engine)
	if !found {
		return nil
	}

	newPrice, found := p.pricing.RDSOnDemandPricePerHour(newType, engine)
	if !found || newPrice > currentPrice {
		return nil
	}

	currentMonthly := currentPrice * carbon.HoursPerMonth
	newMonthly := newPrice * carbon.HoursPerMonth
	savings := currentMonthly - newMonthly
	savingsPercent := 0.0
	if currentMonthly > 0 {
		savingsPercent = (savings / currentMonthly) * 100
	}

	confidence := confidenceHigh

	reasoning := []string{
		fmt.Sprintf("Newer %s instances offer better performance for %s", newFamily, engine),
		"Drop-in replacement with no architecture changes required",
	}

	// Check if there's a Graviton alternative for the recommended family
	if gravitonFamily, hasGraviton := rdsGravitonMap[newFamily]; hasGraviton && rdsGravitonSupportedEngines[engine] {
		gravitonType := gravitonFamily + "." + size
		reasoning = append(reasoning,
			fmt.Sprintf("Alternative: consider %s for ARM compatibility (~20%% additional savings)", gravitonType))
	}

	return &pbc.Recommendation{
		Id:         uuid.New().String(),
		Category:   pbc.RecommendationCategory_RECOMMENDATION_CATEGORY_COST,
		ActionType: pbc.RecommendationActionType_RECOMMENDATION_ACTION_TYPE_MODIFY,
		Resource: &pbc.ResourceRecommendationInfo{
			Provider:     providerAWS,
			ResourceType: "rds",
			Region:       region,
			Sku:          instanceType,
		},
		ActionDetail: &pbc.Recommendation_Modify{
			Modify: &pbc.ModifyAction{
				ModificationType:  modTypeGenUpgrade,
				CurrentConfig:     map[string]string{"instance_type": instanceType, "engine": engine},
				RecommendedConfig: map[string]string{"instance_type": newType, "engine": engine},
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
		Description: fmt.Sprintf("Upgrade RDS %s from %s to %s for better performance at same or lower cost",
			engine, instanceType, newType),
		Reasoning: reasoning,
		Source:    sourceAWSPublic,
	}
}

// getRDSGravitonRecommendation returns a recommendation to migrate to ARM-based
// Graviton RDS instances if available and cost-effective.
// Only called for engines that support Graviton (MySQL, PostgreSQL, MariaDB).
func (p *AWSPublicPlugin) getRDSGravitonRecommendation(
	instanceType, engine, region string,
) *pbc.Recommendation {
	family, size := parseRDSInstanceType(instanceType)
	if family == "" {
		return nil
	}

	gravitonFamily, exists := rdsGravitonMap[family]
	if !exists {
		return nil
	}

	gravitonType := gravitonFamily + "." + size

	currentPrice, found := p.pricing.RDSOnDemandPricePerHour(instanceType, engine)
	if !found {
		return nil
	}

	gravitonPrice, found := p.pricing.RDSOnDemandPricePerHour(gravitonType, engine)
	if !found || gravitonPrice > currentPrice {
		return nil
	}

	currentMonthly := currentPrice * carbon.HoursPerMonth
	gravitonMonthly := gravitonPrice * carbon.HoursPerMonth
	savings := currentMonthly - gravitonMonthly
	savingsPercent := 0.0
	if currentMonthly > 0 {
		savingsPercent = (savings / currentMonthly) * 100
	}

	confidence := confidenceMedium
	return &pbc.Recommendation{
		Id:         uuid.New().String(),
		Category:   pbc.RecommendationCategory_RECOMMENDATION_CATEGORY_COST,
		ActionType: pbc.RecommendationActionType_RECOMMENDATION_ACTION_TYPE_MODIFY,
		Resource: &pbc.ResourceRecommendationInfo{
			Provider:     providerAWS,
			ResourceType: "rds",
			Region:       region,
			Sku:          instanceType,
		},
		ActionDetail: &pbc.Recommendation_Modify{
			Modify: &pbc.ModifyAction{
				ModificationType:  modTypeGraviton,
				CurrentConfig:     map[string]string{"instance_type": instanceType, "engine": engine, "architecture": "x86_64"},
				RecommendedConfig: map[string]string{"instance_type": gravitonType, "engine": engine, "architecture": "arm64"},
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
		Description: fmt.Sprintf("Migrate RDS %s from %s to %s (Graviton) for ~%.0f%% cost savings",
			engine, instanceType, gravitonType, savingsPercent),
		Reasoning: []string{
			"Graviton RDS instances are typically ~20% cheaper with comparable performance",
			fmt.Sprintf("Validated: %s engine supports Graviton architecture", engine),
		},
		Metadata: map[string]string{
			"architecture_change": "x86_64 -> arm64",
			"engine":              engine,
		},
		Source: sourceAWSPublic,
	}
}

// matchesFilter checks if a resource matches the given filter criteria.
// Implements FR-005 (AND operation).
func (p *AWSPublicPlugin) matchesFilter(resource *pbc.ResourceDescriptor, filter *pbc.RecommendationFilter) bool {
	if filter == nil {
		return true
	}

	// Check Region
	if filter.Region != "" && filter.Region != resource.Region {
		return false
	}

	// Check ResourceType
	if filter.ResourceType != "" && filter.ResourceType != resource.ResourceType {
		return false
	}

	// Check Sku
	if filter.Sku != "" && filter.Sku != resource.Sku {
		return false
	}

	// Check Tags (if filter has tags, resource must have all of them with matching values)
	if len(filter.Tags) > 0 {
		for k, v := range filter.Tags {
			if resVal, ok := resource.Tags[k]; !ok || resVal != v {
				return false
			}
		}
	}

	return true
}

// normalizeInput converts a GetRecommendationsRequest into a ProcessingContext.
// If TargetResources is populated, uses it as the scope.
// Otherwise, constructs a single-item scope from Filter fields (legacy mode).
//
// IMPORTANT: This function creates defensive copies of Filter and TargetResources
// before normalizing resource types. This prevents mutation of caller-owned objects
// and ensures thread-safety for concurrent gRPC calls.
func (p *AWSPublicPlugin) normalizeInput(req *pbc.GetRecommendationsRequest) *ProcessingContext {
	pctx := &ProcessingContext{}

	// Deep copy Filter to avoid mutating caller's object (thread-safety for concurrent gRPC calls)
	if req.Filter != nil {
		pctx.Filter = proto.Clone(req.Filter).(*pbc.RecommendationFilter)
	}

	// Issue #124: Normalize filter resource type if present
	if pctx.Filter != nil && pctx.Filter.ResourceType != "" {
		pctx.Filter.ResourceType = normalizeResourceType(pctx.Filter.ResourceType)
	}

	if len(req.TargetResources) > 0 {
		// Batch mode: deep copy each ResourceDescriptor to avoid mutating caller's objects
		pctx.Scope = make([]*pbc.ResourceDescriptor, len(req.TargetResources))
		for i, res := range req.TargetResources {
			pctx.Scope[i] = proto.Clone(res).(*pbc.ResourceDescriptor)
		}
		// Normalize resource types (Issue #124) - now safe to mutate our copies
		for _, res := range pctx.Scope {
			res.ResourceType = normalizeResourceType(res.ResourceType)
		}
	} else if req.Filter != nil && req.Filter.Sku != "" {
		// Legacy mode: construct single-item scope from Filter (already cloned above)
		pctx.Scope = []*pbc.ResourceDescriptor{{
			ResourceType: pctx.Filter.ResourceType, // Use normalized copy
			Sku:          pctx.Filter.Sku,
			Region:       pctx.Filter.Region,
			Tags:         copyTags(pctx.Filter.Tags), // Deep copy to avoid sharing map
			Provider:     providerAWS,                // Implicit for this plugin
		}}
	}

	pctx.BatchStats.TotalResources = len(pctx.Scope)
	return pctx
}

// copyTags creates a shallow copy of a tags map.
// Returns nil if input is nil, preserving the distinction between nil and empty maps.
func copyTags(tags map[string]string) map[string]string {
	if tags == nil {
		return nil
	}
	result := make(map[string]string, len(tags))
	maps.Copy(result, tags)
	return result
}
