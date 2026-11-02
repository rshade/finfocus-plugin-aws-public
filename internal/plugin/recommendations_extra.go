package plugin

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"

	"github.com/rshade/finfocus-plugin-aws-public/internal/carbon"
)

const (
	modTypeEngineMigration = "engine_migration"
	modTypeGatewayEndpoint = "vpc_gateway_endpoint"
	recFieldNodeType       = "node_type"
	recFieldRoute          = "route"
	metaRequiresValidation = "requires_validation"
	tagNumNodes            = "num_nodes"
	tagNumCacheNodes       = "num_cache_nodes"
	tagS3DynamoDBDataGB    = "s3_dynamodb_data_gb"
	engineRedis            = "redis"
	engineValkey           = "valkey"
	mbPerGB                = 1024.0
	msPerSecond            = 1000.0
	defaultLambdaMemoryMB  = 128
	defaultLambdaDuration  = 100
	maxElastiCacheNodes    = 1000
)

// elasticacheGenerationUpgradeMap maps previous-generation ElastiCache node
// families to their current-generation x86 equivalents.
var elasticacheGenerationUpgradeMap = map[string]string{
	"cache.t2": "cache.t3",
	"cache.m4": "cache.m5",
	"cache.r4": "cache.r5",
}

// elasticacheGravitonMap maps x86 ElastiCache node families to Graviton
// equivalents. m7g and r7g are deliberately absent: they price above m5 and r5.
var elasticacheGravitonMap = map[string]string{
	"cache.t3": "cache.t4g",
	"cache.m5": "cache.m6g",
	"cache.r5": "cache.r6g",
}

// parseCacheInstanceType splits an ElastiCache node type into family and size.
// Example: "cache.m5.large" returns ("cache.m5", "large").
// Returns empty strings if the format is invalid.
func parseCacheInstanceType(nodeType string) (string, string) {
	trimmed, ok := strings.CutPrefix(nodeType, "cache.")
	if !ok || strings.Count(trimmed, ".") != 1 {
		return "", ""
	}
	family, size, _ := strings.Cut(trimmed, ".")
	if family == "" || size == "" {
		return "", ""
	}
	return "cache." + family, size
}

// buildModifyRecommendation assembles a cost recommendation around a ModifyAction.
func buildModifyRecommendation(
	actionType pbc.RecommendationActionType,
	info *pbc.ResourceRecommendationInfo,
	modify *pbc.ModifyAction,
	currentMonthly, newMonthly float64,
	priority pbc.RecommendationPriority,
	confidence float64,
	description string,
	reasoning []string,
	metadata map[string]string,
) *pbc.Recommendation {
	savings := currentMonthly - newMonthly
	savingsPercent := 0.0
	if currentMonthly > 0 {
		savingsPercent = (savings / currentMonthly) * 100
	}
	return &pbc.Recommendation{
		Id:           uuid.New().String(),
		Category:     pbc.RecommendationCategory_RECOMMENDATION_CATEGORY_COST,
		ActionType:   actionType,
		Resource:     info,
		ActionDetail: &pbc.Recommendation_Modify{Modify: modify},
		Impact: &pbc.RecommendationImpact{
			EstimatedSavings:  savings,
			Currency:          currencyUSD,
			ProjectionPeriod:  projectionPeriodMonthly,
			CurrentCost:       currentMonthly,
			ProjectedCost:     newMonthly,
			SavingsPercentage: savingsPercent,
		},
		Priority:        priority,
		ConfidenceScore: &confidence,
		Description:     description,
		Reasoning:       reasoning,
		Metadata:        metadata,
		Source:          sourceAWSPublic,
	}
}

// generateStaticServiceRecommendations routes ElastiCache, Lambda, and NAT
// Gateway resources to their generators after canonicalizing Terraform and
// Pulumi names, so every resource-type spelling yields the same advice.
func (p *AWSPublicPlugin) generateStaticServiceRecommendations(
	service string,
	resource *pbc.ResourceDescriptor,
	region string,
) []*pbc.Recommendation {
	canon := canonicalizeResource(service, resource)
	tags := pricingTags(canon)
	switch service {
	case serviceElastiCache:
		nodeType := canon.GetSku()
		if nodeType == "" {
			nodeType = tags[recFieldNodeType]
		}
		return p.generateElastiCacheRecommendations(nodeType, region, tags)
	case serviceLambda:
		return p.generateLambdaRecommendations(canon.GetSku(), region, tags)
	case serviceNATGW:
		return p.generateNATGatewayRecommendations(region, tags)
	}
	return nil
}

// elastiCacheNodeCount reads the node count from tags, defaulting to 1.
// Invalid or out-of-range values also yield 1.
func elastiCacheNodeCount(tags map[string]string) int {
	for _, key := range []string{tagNumNodes, tagNumCacheNodes} {
		if val := tags[key]; val != "" {
			if n, err := strconv.Atoi(val); err == nil && n >= 1 && n <= maxElastiCacheNodes {
				return n
			}
			return 1
		}
	}
	return 1
}

// generateElastiCacheRecommendations returns previous-generation upgrade,
// Graviton, and Redis-to-Valkey recommendations for a cache node type.
// Savings come only from embedded rates multiplied by node count and 730 hours.
func (p *AWSPublicPlugin) generateElastiCacheRecommendations(
	nodeType, region string,
	tags map[string]string,
) []*pbc.Recommendation {
	family, size := parseCacheInstanceType(nodeType)
	if family == "" {
		return nil
	}
	engine := engineRedis
	if val := strings.ToLower(tags[recFieldEngine]); val != "" {
		engine = val
	}
	nodes := elastiCacheNodeCount(tags)

	var recs []*pbc.Recommendation
	if newFamily, ok := elasticacheGenerationUpgradeMap[family]; ok {
		if rec := p.elastiCacheNodeTypeRecommendation(
			nodeType, newFamily+"."+size, engine, region, nodes, modTypeGenUpgrade); rec != nil {
			recs = append(recs, rec)
		}
	}
	if newFamily, ok := elasticacheGravitonMap[family]; ok {
		if rec := p.elastiCacheNodeTypeRecommendation(
			nodeType, newFamily+"."+size, engine, region, nodes, modTypeGraviton); rec != nil {
			recs = append(recs, rec)
		}
	}
	if engine == engineRedis {
		if rec := p.elastiCacheValkeyRecommendation(nodeType, region, nodes); rec != nil {
			recs = append(recs, rec)
		}
	}
	return recs
}

func (p *AWSPublicPlugin) elastiCacheNodeTypeRecommendation(
	nodeType, newType, engine, region string,
	nodes int,
	modType string,
) *pbc.Recommendation {
	currentRate, found := p.pricing.ElastiCacheOnDemandPricePerHour(nodeType, engine)
	if !found {
		return nil
	}
	newRate, found := p.pricing.ElastiCacheOnDemandPricePerHour(newType, engine)
	if !found || newRate > currentRate {
		return nil
	}
	currentMonthly := currentRate * float64(nodes) * carbon.HoursPerMonth
	newMonthly := newRate * float64(nodes) * carbon.HoursPerMonth

	confidence := confidenceHigh
	priority := pbc.RecommendationPriority_RECOMMENDATION_PRIORITY_MEDIUM
	description := fmt.Sprintf("Upgrade ElastiCache %s to %s", nodeType, newType)
	reasoning := []string{
		fmt.Sprintf("%s nodes are a newer generation than %s for the %s engine", newType, nodeType, engine),
		"The change is priced at or below the current on-demand rate",
	}
	metadata := map[string]string{"node_count": strconv.Itoa(nodes)}
	if modType == modTypeGraviton {
		confidence = confidenceMedium
		priority = pbc.RecommendationPriority_RECOMMENDATION_PRIORITY_LOW
		description = fmt.Sprintf("Migrate ElastiCache %s to Graviton %s", nodeType, newType)
		reasoning = []string{
			fmt.Sprintf("%s runs on AWS Graviton at a lower on-demand rate than %s", newType, nodeType),
			"The cache engine is architecture-transparent, but a node type change triggers a cluster modification",
		}
		metadata[metaRequiresValidation] = "true"
	}

	return buildModifyRecommendation(
		pbc.RecommendationActionType_RECOMMENDATION_ACTION_TYPE_MODIFY,
		&pbc.ResourceRecommendationInfo{
			Provider: providerAWS, ResourceType: serviceElastiCache, Region: region, Sku: nodeType,
		},
		&pbc.ModifyAction{
			ModificationType:  modType,
			CurrentConfig:     map[string]string{recFieldNodeType: nodeType, recFieldEngine: engine},
			RecommendedConfig: map[string]string{recFieldNodeType: newType, recFieldEngine: engine},
		},
		currentMonthly, newMonthly, priority, confidence, description, reasoning, metadata,
	)
}

func (p *AWSPublicPlugin) elastiCacheValkeyRecommendation(
	nodeType, region string,
	nodes int,
) *pbc.Recommendation {
	redisRate, found := p.pricing.ElastiCacheOnDemandPricePerHour(nodeType, engineRedis)
	if !found {
		return nil
	}
	valkeyRate, found := p.pricing.ElastiCacheOnDemandPricePerHour(nodeType, engineValkey)
	if !found || valkeyRate >= redisRate {
		return nil
	}
	currentMonthly := redisRate * float64(nodes) * carbon.HoursPerMonth
	newMonthly := valkeyRate * float64(nodes) * carbon.HoursPerMonth
	return buildModifyRecommendation(
		pbc.RecommendationActionType_RECOMMENDATION_ACTION_TYPE_MODIFY,
		&pbc.ResourceRecommendationInfo{
			Provider: providerAWS, ResourceType: serviceElastiCache, Region: region, Sku: nodeType,
		},
		&pbc.ModifyAction{
			ModificationType:  modTypeEngineMigration,
			CurrentConfig:     map[string]string{recFieldNodeType: nodeType, recFieldEngine: engineRedis},
			RecommendedConfig: map[string]string{recFieldNodeType: nodeType, recFieldEngine: engineValkey},
		},
		currentMonthly, newMonthly,
		pbc.RecommendationPriority_RECOMMENDATION_PRIORITY_MEDIUM, confidenceMedium,
		fmt.Sprintf("Switch ElastiCache %s from Redis OSS to Valkey", nodeType),
		[]string{
			"Valkey node-based pricing is lower than Redis OSS for the same node type",
			"Valkey is API-compatible with Redis OSS 7.2",
		},
		map[string]string{"node_count": strconv.Itoa(nodes), metaRequiresValidation: "true"},
	)
}

// generateLambdaRecommendations recommends moving an x86_64 function to arm64.
// It needs requests_per_month > 0 and requires the arm64 rate to be strictly
// below the x86 rate, since the pricing client falls back to x86 when ARM
// pricing is absent.
func (p *AWSPublicPlugin) generateLambdaRecommendations(
	sku, region string,
	tags map[string]string,
) []*pbc.Recommendation {
	arch := tags["arch"]
	if arch == "" {
		arch = tags["architecture"]
	}
	if a := strings.ToLower(arch); a == archARM64 || a == archARM {
		return nil
	}

	requests, err := strconv.ParseInt(tags["requests_per_month"], 10, 64)
	if err != nil || requests <= 0 {
		return nil
	}
	memoryMB := defaultLambdaMemoryMB
	if mem, memErr := strconv.Atoi(sku); memErr == nil && mem > 0 {
		memoryMB = mem
	}
	durationMs := defaultLambdaDuration
	if dur, durErr := strconv.Atoi(tags["avg_duration_ms"]); durErr == nil && dur > 0 {
		durationMs = dur
	}

	x86Rate, found := p.pricing.LambdaPricePerGBSecond(archX86)
	if !found {
		return nil
	}
	armRate, found := p.pricing.LambdaPricePerGBSecond(archARM64)
	if !found || armRate >= x86Rate {
		return nil
	}

	gbSeconds := float64(memoryMB) / mbPerGB * float64(durationMs) / msPerSecond * float64(requests)

	return []*pbc.Recommendation{buildModifyRecommendation(
		pbc.RecommendationActionType_RECOMMENDATION_ACTION_TYPE_MODIFY,
		&pbc.ResourceRecommendationInfo{
			Provider: providerAWS, ResourceType: serviceLambda, Region: region, Sku: sku,
		},
		&pbc.ModifyAction{
			ModificationType:  modTypeGraviton,
			CurrentConfig:     map[string]string{recFieldArchitecture: archX86},
			RecommendedConfig: map[string]string{recFieldArchitecture: archARM64},
		},
		gbSeconds*x86Rate, gbSeconds*armRate,
		pbc.RecommendationPriority_RECOMMENDATION_PRIORITY_MEDIUM, confidenceMedium,
		"Move Lambda function from x86_64 to arm64 for lower GB-second pricing",
		[]string{
			"arm64 GB-second pricing is lower than x86_64; request charges are identical",
			"Savings cover compute duration only, at the first pricing tier",
		},
		map[string]string{
			metaRequiresValidation: "true",
			"gb_seconds_per_month": strconv.FormatFloat(gbSeconds, 'f', 0, 64),
		},
	)}
}

// generateNATGatewayRecommendations recommends S3 and DynamoDB gateway VPC
// endpoints. The share of traffic bound for those services cannot be inferred
// from a descriptor, so it fires only when the caller supplies the
// s3_dynamodb_data_gb tag, capped at data_processed_gb when that tag is set.
// AWS charges nothing for gateway endpoints, so the avoided cost is the NAT
// Gateway data processing rate times that volume.
func (p *AWSPublicPlugin) generateNATGatewayRecommendations(
	region string,
	tags map[string]string,
) []*pbc.Recommendation {
	endpointGB, err := strconv.ParseFloat(tags[tagS3DynamoDBDataGB], 64)
	if err != nil || endpointGB <= 0 {
		return nil
	}
	totalGB := endpointGB
	if total, totalErr := strconv.ParseFloat(tags["data_processed_gb"], 64); totalErr == nil && total >= 0 {
		totalGB = total
		endpointGB = min(endpointGB, total)
	}
	if endpointGB <= 0 {
		return nil
	}

	price, found := p.pricing.NATGatewayPrice()
	if !found || price.DataProcessingRate <= 0 {
		return nil
	}

	return []*pbc.Recommendation{buildModifyRecommendation(
		pbc.RecommendationActionType_RECOMMENDATION_ACTION_TYPE_REFACTOR,
		&pbc.ResourceRecommendationInfo{
			Provider: providerAWS, ResourceType: serviceNATGW, Region: region,
		},
		&pbc.ModifyAction{
			ModificationType:  modTypeGatewayEndpoint,
			CurrentConfig:     map[string]string{recFieldRoute: "nat_gateway"},
			RecommendedConfig: map[string]string{recFieldRoute: "gateway_endpoint", "services": "s3,dynamodb"},
		},
		totalGB*price.DataProcessingRate, (totalGB-endpointGB)*price.DataProcessingRate,
		pbc.RecommendationPriority_RECOMMENDATION_PRIORITY_MEDIUM, confidenceMedium,
		fmt.Sprintf("Route %.0f GB/month of S3 and DynamoDB traffic through gateway VPC endpoints", endpointGB),
		[]string{
			"Gateway VPC endpoints for S3 and DynamoDB have no hourly or data processing charge",
			"The NAT Gateway hourly charge remains because other traffic still needs it",
			"Savings use the caller-supplied s3_dynamodb_data_gb tag, not an inferred traffic share",
		},
		map[string]string{
			metaRequiresValidation: "true",
			"endpoint_data_gb":     strconv.FormatFloat(endpointGB, 'f', -1, 64),
		},
	)}
}
