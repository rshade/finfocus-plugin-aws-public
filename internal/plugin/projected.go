package plugin

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/rshade/finfocus-plugin-aws-public/internal/carbon"
	"github.com/rshade/finfocus-plugin-aws-public/internal/pricing"
	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk/mapping"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	"google.golang.org/grpc/codes"
)

const (
	defaultEBSGB      = 8
	defaultRDSEngine  = "mysql"
	defaultRDSStorage = "gp2"
	defaultRDSSizeGB  = 20
)

// normalizeResourceType converts various resource type formats to a canonical form.
// Examples:
//   - "aws:ec2/instance:Instance" -> "ec2"
//   - "aws:ebs:Volume" -> "ebs"
//   - "aws:ec2/vpc:Vpc" -> "vpc"
//   - "ec2" -> "ec2"
func normalizeResourceType(resourceType string) string {
	rt := strings.ToLower(resourceType)

	// Pattern: aws:<service>/...:... or aws:<service>:...
	if strings.HasPrefix(rt, "aws:") {
		// Special case: aws:ec2/volume is EBS
		if strings.Contains(rt, "ec2/volume") {
			return "ebs"
		}

		// IAM resources (prefix match)
		if strings.HasPrefix(rt, "aws:iam/") {
			return "iam"
		}

		// Zero-cost EC2 networking resources (centralized in ZeroCostPulumiPatterns)
		// Use token-aware matching to avoid false positives (e.g., "ec2/vpc" matching "ec2/vpcEndpoint")
		awsSuffix := rt[4:] // Remove "aws:" prefix (already verified above)
		for pattern, service := range ZeroCostPulumiPatterns {
			// Pattern must be a complete path segment, followed by ":" (Pulumi type separator) or end of string
			if strings.HasPrefix(awsSuffix, pattern) {
				remaining := awsSuffix[len(pattern):]
				if remaining == "" || remaining[0] == ':' {
					return service
				}
			}
		}

		parts := strings.Split(rt[4:], "/")
		if len(parts) > 0 {
			// Extract service from aws:<service>/...
			svcParts := strings.Split(parts[0], ":")
			svc := svcParts[0]
			switch svc {
			case "ec2", "ebs", "rds", "s3", "lambda", "dynamodb", "eks", "natgw", "cloudwatch", "elasticache":
				return svc
			case "lb", "alb", "nlb":
				return "elb"
			case "natgateway":
				return "natgw"
			}
		}
		// If it's an AWS resource but we don't recognize the service canonical form,
		// return the original string to preserve information for detectService fallback.
		return resourceType
	}

	return rt
}

// extractAWSSKU extracts AWS SKU from tags with priority matching SDK behavior.
// Uses SDK mapping.ExtractSKU with extended key list for backwards compatibility
// with both camelCase (SDK standard) and snake_case (legacy) property names.
//
// Priority order: instanceType > instance_class > instanceClass > type > volumeType > volume_type
func extractAWSSKU(tags map[string]string) string {
	// Use SDK's generic ExtractSKU with extended key list for backwards compatibility.
	// This includes both SDK canonical keys (camelCase) and legacy snake_case variants.
	return mapping.ExtractSKU(tags,
		mapping.AWSKeyInstanceType,  // "instanceType" - EC2
		"instance_class",            // legacy snake_case for RDS
		mapping.AWSKeyInstanceClass, // "instanceClass" - RDS (SDK canonical)
		mapping.AWSKeyType,          // "type" - generic fallback
		mapping.AWSKeyVolumeType,    // "volumeType" - EBS (SDK canonical)
		"volume_type",               // legacy snake_case for EBS
	)
}

// extractAWSRegion extracts AWS region from tags with priority: region > availabilityZone.
// Delegates to SDK mapping.ExtractAWSRegion which handles AZ-to-region conversion.
func extractAWSRegion(tags map[string]string) string {
	return mapping.ExtractAWSRegion(tags)
}

// engineNormalization maps user-friendly engine names to AWS pricing API identifiers.
// Multiple aliases (e.g., "postgres" and "postgresql") map to the same canonical name.
var engineNormalization = map[string]string{
	"mysql":        "MySQL",
	"postgres":     "PostgreSQL",
	"postgresql":   "PostgreSQL",
	"mariadb":      "MariaDB",
	"oracle":       "Oracle",
	"oracle-se2":   "Oracle",
	"sqlserver":    "SQL Server",
	"sqlserver-ex": "SQL Server",
	"sql-server":   "SQL Server",
}

// validRDSStorageTypes contains the supported RDS storage volume types.
var validRDSStorageTypes = map[string]bool{
	"gp2":      true,
	"gp3":      true,
	"io1":      true,
	"io2":      true,
	"standard": true,
}

// GetProjectedCost estimates the monthly cost for the given resource.
func (p *AWSPublicPlugin) GetProjectedCost(ctx context.Context, req *pbc.GetProjectedCostRequest) (*pbc.GetProjectedCostResponse, error) {
	start := time.Now()
	traceID := p.getTraceID(ctx)

	// FR-009, FR-010: Use SDK validation + custom region check (US2)
	if _, err := p.ValidateProjectedCostRequest(ctx, req); err != nil {
		// Extract error code from error details for proper logging
		errCode := extractErrorCode(err)
		p.logErrorWithID(traceID, "GetProjectedCost", err, errCode)
		return nil, err
	}

	resource := req.Resource

	// Test mode: Enhanced logging for request details (US3)
	if p.testMode {
		p.logger.Debug().
			Str(pluginsdk.FieldTraceID, traceID).
			Str("resource_type", resource.ResourceType).
			Str("sku", resource.Sku).
			Str("region", resource.Region).
			Str("provider", resource.Provider).
			Msg("Test mode: request details")
	}

	// Route to appropriate estimator based on resource type
	var resp *pbc.GetProjectedCostResponse
	var err error

	// Normalize resource type first (T006, Issue #124)
	normalizedType := normalizeResourceType(resource.ResourceType)
	serviceType := detectService(normalizedType)
	switch serviceType {
	case "ec2":
		resp, err = p.estimateEC2(traceID, resource, req)
	case "ebs":
		resp, err = p.estimateEBS(traceID, resource)
	case "rds":
		resp, err = p.estimateRDS(traceID, resource)
	case "eks":
		resp, err = p.estimateEKS(traceID, resource)
	case "s3":
		resp, err = p.estimateS3(traceID, resource)
	case "lambda":
		resp, err = p.estimateLambda(traceID, resource)
	case "dynamodb":
		resp, err = p.estimateDynamoDB(traceID, resource)
	case "elb":
		resp, err = p.estimateELB(traceID, resource)
	case "natgw":
		resp, err = p.estimateNATGateway(traceID, resource)
	case "cloudwatch":
		resp, err = p.estimateCloudWatch(traceID, resource)
	case "elasticache":
		resp, err = p.estimateElastiCache(traceID, resource)
	case "vpc", "securitygroup", "subnet", "iam":
		// Zero-cost AWS networking and IAM resources - no direct charges
		resp = p.estimateZeroCostResource(traceID, resource, serviceType)
	default:
		// Unknown resource type - return $0 with explanation
		resp = &pbc.GetProjectedCostResponse{
			CostPerMonth:  0,
			UnitPrice:     0,
			Currency:      "USD",
			BillingDetail: fmt.Sprintf("Resource type %q not supported for cost estimation", resource.ResourceType),
		}
	}

	if err != nil {
		p.logErrorWithID(traceID, "GetProjectedCost", err, pbc.ErrorCode_ERROR_CODE_UNSPECIFIED)
		return nil, err
	}

	// Test mode: Enhanced logging for calculation result (US3)
	if p.testMode {
		p.logger.Debug().
			Str(pluginsdk.FieldTraceID, traceID).
			Float64("unit_price", resp.UnitPrice).
			Float64("cost_per_month", resp.CostPerMonth).
			Str("currency", resp.Currency).
			Str("billing_detail", resp.BillingDetail).
			Msg("Test mode: calculation result")
	}

	// Log successful completion with all required fields
	p.traceLogger(traceID, "GetProjectedCost").Info().
		Str(pluginsdk.FieldResourceType, resource.ResourceType).
		Str("aws_service", resource.ResourceType).
		Str("aws_region", resource.Region).
		Interface("tags", sanitizeTagsForLogging(resource.Tags)).
		Float64(pluginsdk.FieldCostMonthly, resp.CostPerMonth).
		Int64(pluginsdk.FieldDurationMs, time.Since(start).Milliseconds()).
		Msg("cost calculated")

	return resp, nil
}

// estimateEC2 calculates the projected monthly cost for an EC2 instance.
// traceID is passed from the parent handler to ensure consistent trace correlation.
func (p *AWSPublicPlugin) estimateEC2(traceID string, resource *pbc.ResourceDescriptor, req *pbc.GetProjectedCostRequest) (*pbc.GetProjectedCostResponse, error) {
	// FR-012: Use resource.Sku first, fallback to tags extraction
	instanceType := resource.Sku
	if instanceType == "" {
		instanceType = extractAWSSKU(resource.Tags)
	}

	// Extract OS and tenancy using shared helper (FR-001, FR-002)
	ec2Attrs := ExtractEC2AttributesFromTags(resource.Tags)

	// FR-020: Lookup pricing using embedded data
	hourlyRate, found := p.pricing.EC2OnDemandPricePerHour(instanceType, ec2Attrs.OS, ec2Attrs.Tenancy)
	if !found {
		// FR-035: Unknown instance types return $0 with explanation
		p.traceLogger(traceID, "GetProjectedCost").Debug().
			Str("instance_type", instanceType).
			Str("aws_region", p.region).
			Str("pricing_source", "embedded").
			Msg("EC2 instance type not found in pricing data")

		return &pbc.GetProjectedCostResponse{
			CostPerMonth:  0,
			UnitPrice:     0,
			Currency:      "USD",
			BillingDetail: fmt.Sprintf(PricingNotFoundTemplate, "EC2 instance type", instanceType),
		}, nil
	}

	// Debug log successful lookup
	p.logger.Debug().
		Str("instance_type", instanceType).
		Str("aws_region", p.region).
		Str("pricing_source", "embedded").
		Float64("unit_price", hourlyRate).
		Msg("EC2 pricing lookup successful")

	// FR-021: Calculate monthly cost (730 hours/month)
	costPerMonth := hourlyRate * carbon.HoursPerMonth

	// FR-022, FR-023, FR-024: Return response with all required fields
	resp := &pbc.GetProjectedCostResponse{
		CostPerMonth:  costPerMonth,
		UnitPrice:     hourlyRate,
		Currency:      "USD",
		BillingDetail: fmt.Sprintf("On-demand %s, %s tenancy, 730 hrs/month", ec2Attrs.OS, ec2Attrs.Tenancy),
	}

	// Carbon estimation: Calculate carbon footprint for EC2 instance
	utilization := carbon.GetUtilization(req.UtilizationPercentage, resource.UtilizationPercentage)
	carbonGrams, carbonOK := p.carbonEstimator.EstimateCarbonGrams(
		instanceType, resource.Region, utilization, carbon.HoursPerMonth,
	)

	if carbonOK {
		resp.ImpactMetrics = []*pbc.ImpactMetric{
			{
				Kind:  pbc.MetricKind_METRIC_KIND_CARBON_FOOTPRINT,
				Value: carbonGrams,
				Unit:  "gCO2e",
			},
		}

		p.traceLogger(traceID, "GetProjectedCost").Debug().
			Str("instance_type", instanceType).
			Str("aws_region", resource.Region).
			Float64("utilization", utilization).
			Float64("carbon_grams", carbonGrams).
			Msg("Carbon estimation successful")
	} else {
		// Unknown instance type for carbon - log warning but continue with financial cost
		p.traceLogger(traceID, "GetProjectedCost").Debug().
			Str("instance_type", instanceType).
			Msg("Carbon estimation skipped - instance type not in CCF data")
	}

	// Apply growth hint enrichment
	setGrowthHint(p.logger.With().Str(pluginsdk.FieldTraceID, traceID).Logger(), "aws:ec2:instance", resp)

	return resp, nil
}

// estimateEBS calculates the projected monthly cost for an EBS volume.
// traceID is passed from the parent handler to ensure consistent trace correlation.
func (p *AWSPublicPlugin) estimateEBS(traceID string, resource *pbc.ResourceDescriptor) (*pbc.GetProjectedCostResponse, error) {
	// FR-012: Use resource.Sku first, fallback to tags extraction
	volumeType := resource.Sku
	if volumeType == "" {
		volumeType = extractAWSSKU(resource.Tags)
	}

	// FR-041 & FR-042: Extract size from tags, default to 8GB
	sizeGB := defaultEBSGB
	sizeAssumed := true

	if resource.Tags != nil {
		if sizeStr, ok := resource.Tags["size"]; ok {
			if size, err := strconv.Atoi(sizeStr); err == nil && size > 0 {
				sizeGB = size
				sizeAssumed = false
			}
		} else if sizeStr, ok := resource.Tags["volume_size"]; ok {
			if size, err := strconv.Atoi(sizeStr); err == nil && size > 0 {
				sizeGB = size
				sizeAssumed = false
			}
		}
	}

	// FR-020: Lookup pricing using embedded data
	ratePerGBMonth, found := p.pricing.EBSPricePerGBMonth(volumeType)
	if !found {
		// Unknown volume type - return $0 with explanation
		p.traceLogger(traceID, "GetProjectedCost").Debug().
			Str("storage_type", volumeType).
			Str("aws_region", p.region).
			Str("pricing_source", "embedded").
			Msg("EBS volume type not found in pricing data")

		return &pbc.GetProjectedCostResponse{
			CostPerMonth:  0,
			UnitPrice:     0,
			Currency:      "USD",
			BillingDetail: fmt.Sprintf(PricingNotFoundTemplate, "EBS volume type", volumeType),
		}, nil
	}

	// Debug log successful lookup
	p.logger.Debug().
		Str("storage_type", volumeType).
		Str("aws_region", p.region).
		Str("pricing_source", "embedded").
		Float64("unit_price", ratePerGBMonth).
		Msg("EBS pricing lookup successful")

	// Calculate monthly cost
	costPerMonth := ratePerGBMonth * float64(sizeGB)

	// FR-043: Include assumption in billing_detail if size was defaulted
	var billingDetail string
	if sizeAssumed {
		billingDetail = fmt.Sprintf("%s volume, %d GB (defaulted), $%.4f/GB-month", volumeType, sizeGB, ratePerGBMonth)
	} else {
		billingDetail = fmt.Sprintf("%s volume, %d GB, $%.4f/GB-month", volumeType, sizeGB, ratePerGBMonth)
	}

	// FR-022, FR-023, FR-024: Build response
	resp := &pbc.GetProjectedCostResponse{
		CostPerMonth:  costPerMonth,
		UnitPrice:     ratePerGBMonth,
		Currency:      "USD",
		BillingDetail: billingDetail,
	}

	// Carbon estimation for EBS volume
	ebsEstimator := carbon.NewEBSEstimator()
	carbonGrams, carbonOK := ebsEstimator.EstimateCarbonGrams(carbon.EBSVolumeConfig{
		VolumeType: volumeType,
		SizeGB:     float64(sizeGB),
		Region:     resource.Region,
		Hours:      HoursPerMonthProd,
	})

	if carbonOK {
		resp.ImpactMetrics = []*pbc.ImpactMetric{
			{
				Kind:  pbc.MetricKind_METRIC_KIND_CARBON_FOOTPRINT,
				Value: carbonGrams,
				Unit:  "gCO2e",
			},
		}

		p.traceLogger(traceID, "GetProjectedCost").Debug().
			Str("volume_type", volumeType).
			Int("size_gb", sizeGB).
			Str("aws_region", resource.Region).
			Float64("carbon_grams", carbonGrams).
			Msg("EBS carbon estimation successful")
	}

	// Apply growth hint enrichment
	setGrowthHint(p.logger.With().Str(pluginsdk.FieldTraceID, traceID).Logger(), "aws:ebs:volume", resp)

	return resp, nil
}

// estimateS3 calculates projected monthly cost for S3 storage.
func (p *AWSPublicPlugin) estimateS3(traceID string, resource *pbc.ResourceDescriptor) (*pbc.GetProjectedCostResponse, error) {
	storageClass := resource.Sku

	// Extract size from tags, default to 1GB
	sizeGB := 1.0
	sizeAssumed := true

	if resource.Tags != nil {
		if sizeStr, ok := resource.Tags["size"]; ok {
			if size, err := strconv.ParseFloat(sizeStr, 64); err == nil && size > 0 {
				sizeGB = size
				sizeAssumed = false
			}
		}
	}

	// Lookup pricing using embedded data
	ratePerGBMonth, found := p.pricing.S3PricePerGBMonth(storageClass)
	if !found {
		// Unknown storage class - return $0 with explanation
		p.traceLogger(traceID, "GetProjectedCost").Debug().
			Str("storage_class", storageClass).
			Str("aws_region", p.region).
			Str("pricing_source", "embedded").
			Msg("S3 storage class not found in pricing data")

		return &pbc.GetProjectedCostResponse{
			CostPerMonth:  0,
			UnitPrice:     0,
			Currency:      "USD",
			BillingDetail: fmt.Sprintf(PricingNotFoundTemplate, "S3 storage class", storageClass),
		}, nil
	}

	// Debug log successful lookup
	p.logger.Debug().
		Str("storage_class", storageClass).
		Str("aws_region", p.region).
		Str("pricing_source", "embedded").
		Float64("unit_price", ratePerGBMonth).
		Msg("S3 pricing lookup successful")

	// Calculate monthly cost
	costPerMonth := ratePerGBMonth * sizeGB

	// Include assumption in billing_detail if size was defaulted
	var billingDetail string
	if sizeAssumed {
		billingDetail = fmt.Sprintf("S3 %s storage, %.0f GB (defaulted), $%.4f/GB-month", storageClass, sizeGB, ratePerGBMonth)
	} else {
		billingDetail = fmt.Sprintf("S3 %s storage, %.0f GB, $%.4f/GB-month", storageClass, sizeGB, ratePerGBMonth)
	}

	resp := &pbc.GetProjectedCostResponse{
		CostPerMonth:  costPerMonth,
		UnitPrice:     ratePerGBMonth,
		Currency:      "USD",
		BillingDetail: billingDetail,
	}

	// Carbon estimation for S3 storage
	s3Estimator := carbon.NewS3Estimator()
	carbonGrams, carbonOK := s3Estimator.EstimateCarbonGrams(carbon.S3StorageConfig{
		StorageClass: storageClass,
		SizeGB:       sizeGB,
		Region:       resource.Region,
		Hours:        HoursPerMonthProd,
	})

	if carbonOK {
		resp.ImpactMetrics = []*pbc.ImpactMetric{
			{
				Kind:  pbc.MetricKind_METRIC_KIND_CARBON_FOOTPRINT,
				Value: carbonGrams,
				Unit:  "gCO2e",
			},
		}

		p.traceLogger(traceID, "GetProjectedCost").Debug().
			Str("storage_class", storageClass).
			Float64("size_gb", sizeGB).
			Str("aws_region", resource.Region).
			Float64("carbon_grams", carbonGrams).
			Msg("S3 carbon estimation successful")
	}

	// Apply growth hint enrichment
	setGrowthHint(p.logger.With().Str(pluginsdk.FieldTraceID, traceID).Logger(), "aws:s3:bucket", resp)

	return resp, nil
}

// validateNonNegativeInt64 validates and parses an int64 tag value.
// Returns the parsed value (defaulting to 0 if negative) and logs a warning if invalid.
func (p *AWSPublicPlugin) validateNonNegativeInt64(traceID, tagName, value string) int64 {
	v, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		p.logger.Warn().
			Str(pluginsdk.FieldTraceID, traceID).
			Str("tag", tagName).
			Str("value", value).
			Msg("invalid integer value, defaulting to 0")
		return 0
	}
	if v < 0 {
		p.logger.Warn().
			Str(pluginsdk.FieldTraceID, traceID).
			Str("tag", tagName).
			Int64("value", v).
			Msg("negative value, defaulting to 0")
		return 0
	}
	return v
}

// validateNonNegativeFloat64 validates and parses a float64 tag value.
func (p *AWSPublicPlugin) validateNonNegativeFloat64(traceID, tagName, value string) float64 {
	v, err := strconv.ParseFloat(value, 64)
	if err != nil {
		p.logger.Warn().
			Str(pluginsdk.FieldTraceID, traceID).
			Str("tag", tagName).
			Str("value", value).
			Msg("invalid float value, defaulting to 0")
		return 0
	}
	if v < 0 {
		p.logger.Warn().
			Str(pluginsdk.FieldTraceID, traceID).
			Str("tag", tagName).
			Float64("value", v).
			Msg("negative value, defaulting to 0")
		return 0
	}
	return v
}

// estimateDynamoDB calculates projected monthly cost for DynamoDB tables.
func (p *AWSPublicPlugin) estimateDynamoDB(traceID string, resource *pbc.ResourceDescriptor) (*pbc.GetProjectedCostResponse, error) {
	capacityMode := strings.ToLower(resource.Sku)
	if capacityMode == "" {
		capacityMode = "on-demand"
	}

	var readUnits, writeUnits int64
	var storageGB float64
	var billingDetail string
	var unitPrice float64

	// Extract common storage
	if resource.Tags != nil {
		if s, ok := resource.Tags["storage_gb"]; ok {
			storageGB = p.validateNonNegativeFloat64(traceID, "storage_gb", s)
		}
	}

	storagePrice, storageFound := p.pricing.DynamoDBStoragePricePerGBMonth()
	storageCost := storageGB * storagePrice
	var unavailable []string
	if !storageFound {
		p.logger.Warn().
			Str(pluginsdk.FieldTraceID, traceID).
			Str("component", "Storage").
			Msg("DynamoDB storage pricing unavailable")
		unavailable = append(unavailable, "Storage")
	}

	if capacityMode == "provisioned" {
		// Provisioned Mode
		if resource.Tags != nil {
			if s, ok := resource.Tags["read_capacity_units"]; ok {
				readUnits = p.validateNonNegativeInt64(traceID, "read_capacity_units", s)
			}
			if s, ok := resource.Tags["write_capacity_units"]; ok {
				writeUnits = p.validateNonNegativeInt64(traceID, "write_capacity_units", s)
			}
		}

		rcuPrice, rcuFound := p.pricing.DynamoDBProvisionedRCUPrice()
		wcuPrice, wcuFound := p.pricing.DynamoDBProvisionedWCUPrice()
		unitPrice = rcuPrice // Use RCU as primary unit price

		if !rcuFound {
			p.logger.Warn().
				Str(pluginsdk.FieldTraceID, traceID).
				Str("component", "RCU").
				Msg("DynamoDB provisioned RCU pricing unavailable")
			unavailable = append(unavailable, "RCU")
		}
		if !wcuFound {
			p.logger.Warn().
				Str(pluginsdk.FieldTraceID, traceID).
				Str("component", "WCU").
				Msg("DynamoDB provisioned WCU pricing unavailable")
			unavailable = append(unavailable, "WCU")
		}

		// Monthly cost = (RCU * 730 * price) + (WCU * 730 * price) + (Storage * price)
		rcuCost := float64(readUnits) * 730 * rcuPrice
		wcuCost := float64(writeUnits) * 730 * wcuPrice
		totalCost := rcuCost + wcuCost + storageCost

		billingDetail = fmt.Sprintf("DynamoDB provisioned, %d RCUs, %d WCUs, 730 hrs/month, %.0fGB storage",
			readUnits, writeUnits, storageGB)

		if len(unavailable) > 0 {
			billingDetail += fmt.Sprintf(" (pricing unavailable: %s)", strings.Join(unavailable, ", "))
		}

		// FR-007 & US3: Explicitly mention zero/missing inputs if total cost is 0
		if totalCost == 0 {
			billingDetail += " (missing or zero usage inputs)"
		}

		p.traceLogger(traceID, "GetProjectedCost").Debug().
			Str("capacity_mode", "provisioned").
			Int64("rcu", readUnits).
			Int64("wcu", writeUnits).
			Float64("storage_gb", storageGB).
			Msg("DynamoDB provisioned lookup successful")

		resp := &pbc.GetProjectedCostResponse{
			CostPerMonth:  totalCost,
			UnitPrice:     unitPrice,
			Currency:      "USD",
			BillingDetail: billingDetail,
		}

		// Carbon estimation for DynamoDB (storage-based)
		dynamoEstimator := carbon.NewDynamoDBEstimator()
		carbonGrams, carbonOK := dynamoEstimator.EstimateCarbonGrams(carbon.DynamoDBTableConfig{
			SizeGB: storageGB,
			Region: resource.Region,
			Hours:  HoursPerMonthProd,
		})

		if carbonOK && storageGB > 0 {
			resp.ImpactMetrics = []*pbc.ImpactMetric{
				{
					Kind:  pbc.MetricKind_METRIC_KIND_CARBON_FOOTPRINT,
					Value: carbonGrams,
					Unit:  "gCO2e",
				},
			}

			p.traceLogger(traceID, "GetProjectedCost").Debug().
				Float64("storage_gb", storageGB).
				Str("aws_region", resource.Region).
				Float64("carbon_grams", carbonGrams).
				Msg("DynamoDB carbon estimation successful")
		}

		// Apply growth hint enrichment
		setGrowthHint(p.logger.With().Str(pluginsdk.FieldTraceID, traceID).Logger(), "aws:dynamodb:table", resp)

		return resp, nil

	}

	// Default to On-Demand Mode
	if resource.Tags != nil {
		if s, ok := resource.Tags["read_requests_per_month"]; ok {
			readUnits = p.validateNonNegativeInt64(traceID, "read_requests_per_month", s)
		}
		if s, ok := resource.Tags["write_requests_per_month"]; ok {
			writeUnits = p.validateNonNegativeInt64(traceID, "write_requests_per_month", s)
		}
	}

	readPrice, readFound := p.pricing.DynamoDBOnDemandReadPrice()
	writePrice, writeFound := p.pricing.DynamoDBOnDemandWritePrice()
	unitPrice = storagePrice // Use storage as primary unit price for on-demand

	if !readFound {
		p.logger.Warn().
			Str(pluginsdk.FieldTraceID, traceID).
			Str("component", "Read").
			Msg("DynamoDB on-demand read pricing unavailable")
		unavailable = append(unavailable, "Read")
	}
	if !writeFound {
		p.logger.Warn().
			Str(pluginsdk.FieldTraceID, traceID).
			Str("component", "Write").
			Msg("DynamoDB on-demand write pricing unavailable")
		unavailable = append(unavailable, "Write")
	}

	// Monthly cost = (Reads * readPrice) + (Writes * writePrice) + (Storage * storagePrice)
	// Prices are per request unit
	readCost := float64(readUnits) * readPrice
	writeCost := float64(writeUnits) * writePrice
	totalCost := readCost + writeCost + storageCost

	billingDetail = fmt.Sprintf("DynamoDB on-demand, %d reads, %d writes, %.0fGB storage",
		readUnits, writeUnits, storageGB)

	if len(unavailable) > 0 {
		billingDetail += fmt.Sprintf(" (pricing unavailable: %s)", strings.Join(unavailable, ", "))
	}

	// FR-007 & US3: Explicitly mention zero/missing inputs if total cost is 0
	if totalCost == 0 {
		billingDetail += " (missing or zero usage inputs)"
	}

	p.logger.Debug().
		Str("capacity_mode", "on-demand").
		Int64("read_units", readUnits).
		Int64("write_units", writeUnits).
		Float64("storage_gb", storageGB).
		Msg("DynamoDB on-demand lookup successful")

	resp := &pbc.GetProjectedCostResponse{
		CostPerMonth:  totalCost,
		UnitPrice:     unitPrice,
		Currency:      "USD",
		BillingDetail: billingDetail,
	}

	// Carbon estimation for DynamoDB (storage-based)
	dynamoEstimator := carbon.NewDynamoDBEstimator()
	carbonGrams, carbonOK := dynamoEstimator.EstimateCarbonGrams(carbon.DynamoDBTableConfig{
		SizeGB: storageGB,
		Region: resource.Region,
		Hours:  HoursPerMonthProd,
	})

	if carbonOK && storageGB > 0 {
		resp.ImpactMetrics = []*pbc.ImpactMetric{
			{
				Kind:  pbc.MetricKind_METRIC_KIND_CARBON_FOOTPRINT,
				Value: carbonGrams,
				Unit:  "gCO2e",
			},
		}

		p.traceLogger(traceID, "GetProjectedCost").Debug().
			Float64("storage_gb", storageGB).
			Str("aws_region", resource.Region).
			Float64("carbon_grams", carbonGrams).
			Msg("DynamoDB carbon estimation successful")
	}

	// Apply growth hint enrichment
	setGrowthHint(p.logger.With().Str(pluginsdk.FieldTraceID, traceID).Logger(), "aws:dynamodb:table", resp)

	return resp, nil
}

// estimateELB calculates projected monthly cost for load balancers.
func (p *AWSPublicPlugin) estimateELB(traceID string, resource *pbc.ResourceDescriptor) (*pbc.GetProjectedCostResponse, error) {
	// 1. Identify Load Balancer Type (ALB vs NLB)
	// Default to ALB per clarification
	lbType := "alb"
	skuLower := strings.ToLower(resource.Sku)
	if strings.Contains(skuLower, "nlb") || strings.Contains(skuLower, "network") {
		lbType = "nlb"
	} else if strings.Contains(skuLower, "alb") || strings.Contains(skuLower, "application") {
		lbType = "alb"
	}

	// 2. Extract Capacity Units from Tags
	capacityUnits := 0.0
	tagFound := false
	if resource.Tags != nil {
		// Specific tags take precedence
		if lbType == "alb" {
			if s, ok := resource.Tags["lcu_per_hour"]; ok {
				if v, err := strconv.ParseFloat(s, 64); err == nil && v >= 0 {
					capacityUnits = v
					tagFound = true
				}
			}
		} else {
			if s, ok := resource.Tags["nlcu_per_hour"]; ok {
				if v, err := strconv.ParseFloat(s, 64); err == nil && v >= 0 {
					capacityUnits = v
					tagFound = true
				}
			}
		}

		// Generic fallback if specific tag not found or invalid
		if !tagFound {
			if s, ok := resource.Tags["capacity_units"]; ok {
				if v, err := strconv.ParseFloat(s, 64); err == nil && v >= 0 {
					capacityUnits = v
				}
			}
		}
	}

	// Warn if capacity units are unusually high (#165)
	const warnCapacityUnitThreshold = 1000.0
	if capacityUnits > warnCapacityUnitThreshold {
		p.logger.Warn().
			Str(pluginsdk.FieldTraceID, traceID).
			Float64("capacity_units", capacityUnits).
			Float64("threshold", warnCapacityUnitThreshold).
			Msg("Capacity units unusually high - verify this is intentional")
	}

	// 3. Lookup Pricing
	var fixedRate, cuRate float64
	var fixedFound, cuFound bool
	var cuMetricName string

	if lbType == "alb" {
		fixedRate, fixedFound = p.pricing.ALBPricePerHour()
		cuRate, cuFound = p.pricing.ALBPricePerLCU()
		cuMetricName = "LCU"
	} else {
		fixedRate, fixedFound = p.pricing.NLBPricePerHour()
		cuRate, cuFound = p.pricing.NLBPricePerNLCU()
		cuMetricName = "NLCU"
	}

	if !fixedFound || !cuFound {
		p.traceLogger(traceID, "GetProjectedCost").Debug().
			Str("lb_type", lbType).
			Str("aws_region", p.region).
			Msg("ELB pricing data not found")

		return &pbc.GetProjectedCostResponse{
			CostPerMonth:  0,
			UnitPrice:     0,
			Currency:      "USD",
			BillingDetail: fmt.Sprintf(PricingUnavailableTemplate, strings.ToUpper(lbType), p.region),
		}, nil
	}

	// 4. Calculate Costs
	fixedMonthly := carbon.HoursPerMonth * fixedRate
	cuMonthly := carbon.HoursPerMonth * capacityUnits * cuRate
	totalMonthly := fixedMonthly + cuMonthly

	// 5. Build Billing Detail
	billingDetail := fmt.Sprintf("%s, 730 hrs/month, %.1f %s avg/hr",
		strings.ToUpper(lbType), capacityUnits, cuMetricName)

	p.logger.Debug().
		Str("lb_type", lbType).
		Float64("capacity_units", capacityUnits).
		Str("aws_region", p.region).
		Float64("fixed_rate", fixedRate).
		Float64("cu_rate", cuRate).
		Float64("total_cost", totalMonthly).
		Msg("ELB cost estimated")

	resp := &pbc.GetProjectedCostResponse{
		CostPerMonth:  totalMonthly,
		UnitPrice:     fixedRate, // Using fixed hourly as primary unit price
		Currency:      "USD",
		BillingDetail: billingDetail,
	}

	// Apply growth hint enrichment
	setGrowthHint(p.logger.With().Str(pluginsdk.FieldTraceID, traceID).Logger(), "aws:elasticloadbalancing:loadbalancer", resp)

	return resp, nil
}

// estimateStub returns $0 cost for services not yet implemented.
func (p *AWSPublicPlugin) estimateStub(resource *pbc.ResourceDescriptor) (*pbc.GetProjectedCostResponse, error) {
	// FR-025 & FR-026: Return $0 with explanation
	return &pbc.GetProjectedCostResponse{
		CostPerMonth:  0,
		UnitPrice:     0,
		Currency:      "USD",
		BillingDetail: fmt.Sprintf("%s cost estimation not fully implemented - returns $0 estimate", resource.ResourceType),
	}, nil
}

// estimateRDS calculates the projected monthly cost for an RDS instance.
// traceID is passed from the parent handler to ensure consistent trace correlation.
func (p *AWSPublicPlugin) estimateRDS(traceID string, resource *pbc.ResourceDescriptor) (*pbc.GetProjectedCostResponse, error) {
	// FR-012: Use resource.Sku first, fallback to tags extraction
	instanceType := resource.Sku
	if instanceType == "" {
		instanceType = extractAWSSKU(resource.Tags)
	}

	// Extract engine from tags, default to MySQL
	engine := defaultRDSEngine
	engineDefaulted := true
	if resource.Tags != nil {
		if engineTag, ok := resource.Tags["engine"]; ok && engineTag != "" {
			engine = strings.ToLower(engineTag)
			engineDefaulted = false
		}
	}

	// Normalize engine name for AWS pricing lookup
	normalizedEngine, engineKnown := engineNormalization[engine]
	if !engineKnown {
		// Unknown engine - default to MySQL with note
		normalizedEngine = "MySQL"
		engineDefaulted = true
	}

	// Extract storage info from tags
	storageType := defaultRDSStorage
	storageDefaulted := true
	if resource.Tags != nil {
		if st, ok := resource.Tags["storage_type"]; ok && st != "" {
			storageType = strings.ToLower(st)
			storageDefaulted = false
		}
	}

	// Validate storage type
	if !validRDSStorageTypes[storageType] {
		storageType = defaultRDSStorage
		storageDefaulted = true
	}

	// Extract storage size from tags
	storageSizeGB := defaultRDSSizeGB
	sizeDefaulted := true
	if resource.Tags != nil {
		if sizeStr, ok := resource.Tags["storage_size"]; ok {
			if size, err := strconv.Atoi(sizeStr); err == nil && size > 0 {
				storageSizeGB = size
				sizeDefaulted = false
			}
		}
	}

	// Extract Multi-AZ from tags (for carbon estimation)
	multiAZ := false
	if resource.Tags != nil {
		if multiAZStr, ok := resource.Tags["multi_az"]; ok {
			multiAZ = strings.EqualFold(multiAZStr, "true")
		}
	}

	// Lookup instance hourly rate
	hourlyRate, found := p.pricing.RDSOnDemandPricePerHour(instanceType, normalizedEngine)
	if !found {
		// Unknown instance type - return $0 with explanation
		p.traceLogger(traceID, "GetProjectedCost").Debug().
			Str("instance_type", instanceType).
			Str("engine", normalizedEngine).
			Str("aws_region", p.region).
			Str("pricing_source", "embedded").
			Msg("RDS instance type not found in pricing data")

		return &pbc.GetProjectedCostResponse{
			CostPerMonth:  0,
			UnitPrice:     0,
			Currency:      "USD",
			BillingDetail: fmt.Sprintf(PricingNotFoundTemplate, "RDS instance type", instanceType),
		}, nil
	}

	// Lookup storage rate
	storageRate, storageFound := p.pricing.RDSStoragePricePerGBMonth(storageType)
	if !storageFound {
		// Storage type not found, use 0 for storage cost
		storageRate = 0
	}

	// Debug log successful lookup
	p.logger.Debug().
		Str("instance_type", instanceType).
		Str("engine", normalizedEngine).
		Str("storage_type", storageType).
		Int("storage_size_gb", storageSizeGB).
		Str("aws_region", p.region).
		Str("pricing_source", "embedded").
		Float64("unit_price", hourlyRate).
		Float64("storage_rate", storageRate).
		Msg("RDS pricing lookup successful")

	// Calculate monthly costs
	instanceCostPerMonth := hourlyRate * carbon.HoursPerMonth
	storageCostPerMonth := storageRate * float64(storageSizeGB)
	totalCostPerMonth := instanceCostPerMonth + storageCostPerMonth

	// Build billing detail message
	var billingDetail string
	defaultNotes := []string{}
	if engineDefaulted {
		defaultNotes = append(defaultNotes, "engine defaulted to MySQL")
	}
	if storageDefaulted {
		defaultNotes = append(defaultNotes, "storage type defaulted")
	}
	if sizeDefaulted {
		defaultNotes = append(defaultNotes, "size defaulted to 20GB")
	}

	if len(defaultNotes) > 0 {
		billingDetail = fmt.Sprintf("RDS %s %s, 730 hrs/month + %dGB %s storage (%s)",
			instanceType, normalizedEngine, storageSizeGB, storageType, strings.Join(defaultNotes, ", "))
	} else {
		billingDetail = fmt.Sprintf("RDS %s %s, 730 hrs/month + %dGB %s storage",
			instanceType, normalizedEngine, storageSizeGB, storageType)
	}

	resp := &pbc.GetProjectedCostResponse{
		CostPerMonth:  totalCostPerMonth,
		UnitPrice:     hourlyRate,
		Currency:      "USD",
		BillingDetail: billingDetail,
	}

	// Carbon estimation for RDS instance (compute + storage)
	rdsEstimator := carbon.NewRDSEstimator()
	carbonGrams, carbonOK := rdsEstimator.EstimateCarbonGrams(carbon.RDSInstanceConfig{
		InstanceType:  instanceType,
		Region:        resource.Region,
		MultiAZ:       multiAZ,
		StorageType:   storageType,
		StorageSizeGB: float64(storageSizeGB),
		Utilization:   carbon.DefaultUtilization, // Use CCF default (50%)
		Hours:         HoursPerMonthProd,
	})

	if carbonOK {
		resp.ImpactMetrics = []*pbc.ImpactMetric{
			{
				Kind:  pbc.MetricKind_METRIC_KIND_CARBON_FOOTPRINT,
				Value: carbonGrams,
				Unit:  "gCO2e",
			},
		}

		p.traceLogger(traceID, "GetProjectedCost").Debug().
			Str("instance_type", instanceType).
			Int("storage_size_gb", storageSizeGB).
			Bool("multi_az", multiAZ).
			Str("aws_region", resource.Region).
			Float64("carbon_grams", carbonGrams).
			Msg("RDS carbon estimation successful")
	}

	// Apply growth hint enrichment
	setGrowthHint(p.logger.With().Str(pluginsdk.FieldTraceID, traceID).Logger(), "aws:rds:instance", resp)

	return resp, nil
}

// detectService maps a provider resource type string to a normalized service identifier.
// The input resourceType is expected to be normalized by normalizeResourceType().
func detectService(resourceType string) string {
	// Fast path for canonical forms
	switch resourceType {
	case "ec2", "ebs", "rds", "s3", "lambda", "dynamodb", "eks", "elb", "natgw", "cloudwatch", "elasticache":
		return resourceType
	case "alb", "nlb":
		return "elb"
	}

	// Zero-cost networking resources (no direct AWS charges)
	// Defined centrally in ZeroCostServices (constants.go)
	if IsZeroCostService(resourceType) {
		return resourceType
	}

	// Fallback for legacy patterns if normalization didn't catch them
	resourceTypeLower := strings.ToLower(resourceType)

	if strings.Contains(resourceTypeLower, "ec2/instance") {
		return "ec2"
	}
	if strings.Contains(resourceTypeLower, "ebs/volume") || strings.Contains(resourceTypeLower, "ec2/volume") {
		return "ebs"
	}
	if strings.Contains(resourceTypeLower, "rds/instance") {
		return "rds"
	}
	if strings.Contains(resourceTypeLower, "eks/cluster") {
		return "eks"
	}
	if strings.Contains(resourceTypeLower, "s3/bucket") {
		return "s3"
	}
	if strings.Contains(resourceTypeLower, "lambda/function") {
		return "lambda"
	}
	if strings.Contains(resourceTypeLower, "dynamodb/table") {
		return "dynamodb"
	}
	if strings.Contains(resourceTypeLower, "lb/loadbalancer") || strings.Contains(resourceTypeLower, "alb/loadbalancer") || strings.Contains(resourceTypeLower, "nlb/loadbalancer") {
		return "elb"
	}
	if strings.Contains(resourceTypeLower, "ec2/natgateway") {
		return "natgw"
	}
	if strings.Contains(resourceTypeLower, "cloudwatch/loggroup") || strings.Contains(resourceTypeLower, "cloudwatch/logstream") ||
		strings.Contains(resourceTypeLower, "cloudwatch/metricalarm") {
		return "cloudwatch"
	}
	if strings.Contains(resourceTypeLower, "elasticache/") {
		return "elasticache"
	}
	if strings.Contains(resourceTypeLower, "iam/") {
		return "iam"
	}

	return resourceType
}

// estimateEKS calculates projected monthly cost for EKS clusters.
// EKS has a simple fixed hourly rate per cluster (standard or extended support).
func (p *AWSPublicPlugin) estimateEKS(traceID string, resource *pbc.ResourceDescriptor) (*pbc.GetProjectedCostResponse, error) {
	// Determine support type from resource.Sku or tags
	// resource.Sku = "cluster" (standard) or "cluster-extended" (extended support)
	// OR use tags: tags["support_type"] == "extended" (case-insensitive)
	extendedSupport := resource.Sku == "cluster-extended" ||
		(resource.Tags != nil && strings.EqualFold(resource.Tags["support_type"], "extended"))

	// Look up EKS pricing based on support type
	hourlyRate, found := p.pricing.EKSClusterPricePerHour(extendedSupport)
	if !found {
		p.traceLogger(traceID, "GetProjectedCost").Debug().
			Str("aws_region", p.region).
			Bool("extended_support", extendedSupport).
			Msg("EKS pricing data not found")

		return &pbc.GetProjectedCostResponse{
			CostPerMonth:  0,
			UnitPrice:     0,
			Currency:      "USD",
			BillingDetail: fmt.Sprintf(PricingUnavailableTemplate, "EKS", p.region),
		}, nil
	}

	// Debug log successful lookup
	p.logger.Debug().
		Str("aws_region", p.region).
		Bool("extended_support", extendedSupport).
		Float64("hourly_rate", hourlyRate).
		Msg("EKS pricing lookup successful")

	// Calculate monthly cost (730 hours/month)
	costPerMonth := hourlyRate * carbon.HoursPerMonth

	// Determine support type description
	supportType := "standard support"
	if extendedSupport {
		supportType = "extended support"
	}

	resp := &pbc.GetProjectedCostResponse{
		CostPerMonth:  costPerMonth,
		UnitPrice:     hourlyRate,
		Currency:      "USD",
		BillingDetail: fmt.Sprintf("EKS cluster (%s), 730 hrs/month (control plane only, excludes worker nodes)", supportType),
	}

	// Carbon estimation for EKS (control plane is shared, returns 0)
	eksEstimator := carbon.NewEKSEstimator()
	carbonGrams, _ := eksEstimator.EstimateCarbonGrams(carbon.EKSClusterConfig{
		Region: resource.Region,
	})

	// EKS control plane carbon is shared infrastructure, so we always include
	// the metric with value 0 and explain in billing detail that worker nodes
	// should be estimated as EC2 instances.
	resp.ImpactMetrics = []*pbc.ImpactMetric{
		{
			Kind:  pbc.MetricKind_METRIC_KIND_CARBON_FOOTPRINT,
			Value: carbonGrams, // 0 for control plane
			Unit:  "gCO2e",
		},
	}

	p.logger.Debug().
		Str("aws_region", resource.Region).
		Float64("carbon_grams", carbonGrams).
		Msg("EKS carbon estimation: control plane is shared infrastructure (0 gCO2e)")

	// Apply growth hint enrichment
	setGrowthHint(p.logger.With().Str(pluginsdk.FieldTraceID, traceID).Logger(), "aws:eks:cluster", resp)

	return resp, nil
}

// estimateLambda calculates projected monthly cost for Lambda functions.
// Uses request count and GB-seconds from resource tags.
func (p *AWSPublicPlugin) estimateLambda(traceID string, resource *pbc.ResourceDescriptor) (*pbc.GetProjectedCostResponse, error) {
	// 1. Determine Memory (SKU -> MB)
	memoryMB := 128
	memoryDefaulted := false
	if resource.Sku != "" {
		if mem, err := strconv.Atoi(resource.Sku); err == nil && mem > 0 {
			memoryMB = mem
		} else {
			memoryDefaulted = true
		}
	} else {
		memoryDefaulted = true
	}

	// 2. Extract Usage Tags (Requests, Duration, Architecture)
	requestsPerMonth := int64(0)
	avgDurationMs := 100
	architecture := "x86_64" // Default to x86_64 per FR-011
	requestsDefaulted := true
	durationDefaulted := true
	archDefaulted := true

	if resource.Tags != nil {
		if reqStr, ok := resource.Tags["requests_per_month"]; ok {
			if reqs, err := strconv.ParseInt(reqStr, 10, 64); err == nil && reqs >= 0 {
				requestsPerMonth = reqs
				requestsDefaulted = false
			}
		}
		if durStr, ok := resource.Tags["avg_duration_ms"]; ok {
			if dur, err := strconv.Atoi(durStr); err == nil && dur > 0 {
				avgDurationMs = dur
				durationDefaulted = false
			}
		}
		// FR-011: Read architecture from tags
		if archStr, ok := resource.Tags["arch"]; ok && archStr != "" {
			architecture = archStr
			archDefaulted = false
		} else if archStr, ok := resource.Tags["architecture"]; ok && archStr != "" {
			architecture = archStr
			archDefaulted = false
		}
	}

	// 3. Lookup Pricing (with architecture)
	reqPrice, reqFound := p.pricing.LambdaPricePerRequest()
	gbSecPrice, gbSecFound := p.pricing.LambdaPricePerGBSecond(architecture)

	if !reqFound || !gbSecFound {
		p.traceLogger(traceID, "GetProjectedCost").Debug().
			Str("aws_region", p.region).
			Str("architecture", architecture).
			Msg("Lambda pricing data not found")

		return &pbc.GetProjectedCostResponse{
			CostPerMonth:  0,
			UnitPrice:     0,
			Currency:      "USD",
			BillingDetail: fmt.Sprintf(PricingUnavailableTemplate, "Lambda", p.region),
		}, nil
	}

	// 4. Calculate Costs
	// Memory (GB) = Memory (MB) / 1024
	memoryGB := float64(memoryMB) / 1024.0
	// Duration (Seconds) = Duration (ms) / 1000
	durationSeconds := float64(avgDurationMs) / 1000.0
	// Total GB-Seconds = Memory (GB) * Duration (Seconds) * Request Count
	totalGBSec := memoryGB * durationSeconds * float64(requestsPerMonth)

	requestCost := float64(requestsPerMonth) * reqPrice
	computeCost := totalGBSec * gbSecPrice
	totalCost := requestCost + computeCost

	// 5. Build Billing Detail
	var notes []string
	if memoryDefaulted {
		notes = append(notes, "memory defaulted")
	}
	if requestsDefaulted {
		notes = append(notes, "requests defaulted")
	}
	if durationDefaulted {
		notes = append(notes, "duration defaulted")
	}
	if archDefaulted {
		notes = append(notes, "arch defaulted to x86_64")
	}

	// Normalize architecture display name for consistent billing details.
	// User input "arm" is normalized to "arm64" to match AWS Lambda's official
	// architecture naming (x86_64, arm64). This ensures billing details are
	// consistent regardless of whether the user specifies "arm" or "arm64".
	archDisplay := "x86_64"
	if strings.ToLower(architecture) == "arm64" || strings.ToLower(architecture) == "arm" {
		archDisplay = "arm64"
	}

	detail := fmt.Sprintf("Lambda %dMB (%s), %d requests/month, %dms avg duration",
		memoryMB, archDisplay, requestsPerMonth, avgDurationMs)
	if len(notes) > 0 {
		detail += fmt.Sprintf(" (%s)", strings.Join(notes, ", "))
	}
	detail += fmt.Sprintf(", %.0f GB-seconds", totalGBSec)

	p.logger.Debug().
		Int("memory_mb", memoryMB).
		Str("architecture", archDisplay).
		Int64("requests", requestsPerMonth).
		Int("duration_ms", avgDurationMs).
		Float64("gb_seconds", totalGBSec).
		Float64("total_cost", totalCost).
		Msg("Lambda cost estimated")

	resp := &pbc.GetProjectedCostResponse{
		CostPerMonth:  totalCost,
		UnitPrice:     gbSecPrice, // Using GB-second price as unit price
		Currency:      "USD",
		BillingDetail: detail,
	}

	// Carbon estimation for Lambda function
	lambdaEstimator := carbon.NewLambdaEstimator()
	carbonGrams, carbonOK := lambdaEstimator.EstimateCarbonGrams(carbon.LambdaFunctionConfig{
		MemoryMB:     memoryMB,
		DurationMs:   avgDurationMs,
		Invocations:  requestsPerMonth,
		Architecture: archDisplay,
		Region:       resource.Region,
	})

	if carbonOK {
		resp.ImpactMetrics = []*pbc.ImpactMetric{
			{
				Kind:  pbc.MetricKind_METRIC_KIND_CARBON_FOOTPRINT,
				Value: carbonGrams,
				Unit:  "gCO2e",
			},
		}

		p.traceLogger(traceID, "GetProjectedCost").Debug().
			Int("memory_mb", memoryMB).
			Str("architecture", archDisplay).
			Int64("invocations", requestsPerMonth).
			Str("aws_region", resource.Region).
			Float64("carbon_grams", carbonGrams).
			Msg("Lambda carbon estimation successful")
	}

	// Apply growth hint enrichment
	setGrowthHint(p.logger.With().Str(pluginsdk.FieldTraceID, traceID).Logger(), "aws:lambda:function", resp)

	return resp, nil
}

// estimateNATGateway calculates projected monthly cost for VPC NAT Gateways.
// Combines fixed hourly cost and variable data processing cost.
func (p *AWSPublicPlugin) estimateNATGateway(traceID string, resource *pbc.ResourceDescriptor) (*pbc.GetProjectedCostResponse, error) {
	// 1. Lookup Pricing
	pricing, found := p.pricing.NATGatewayPrice()
	if !found {
		p.traceLogger(traceID, "GetProjectedCost").Debug().
			Str("aws_region", p.region).
			Msg("NAT Gateway pricing data not found")

		return &pbc.GetProjectedCostResponse{
			CostPerMonth:  0,
			UnitPrice:     0,
			Currency:      "USD",
			BillingDetail: fmt.Sprintf(PricingUnavailableTemplate, "NAT Gateway", p.region),
		}, nil
	}

	// 2. Extract and Validate Data Processed Tag
	dataProcessedGB := 0.0
	tagPresent := false
	if resource.Tags != nil {
		if val, ok := resource.Tags["data_processed_gb"]; ok {
			tagPresent = true
			if val == "" {
				return nil, p.newErrorWithID(traceID, codes.InvalidArgument, "tag 'data_processed_gb' is present but empty", pbc.ErrorCode_ERROR_CODE_INVALID_RESOURCE)
			}
			parsed, err := strconv.ParseFloat(val, 64)
			if err != nil {
				return nil, p.newErrorWithID(traceID, codes.InvalidArgument, fmt.Sprintf("invalid value for 'data_processed_gb': %q is not a valid number", val), pbc.ErrorCode_ERROR_CODE_INVALID_RESOURCE)
			}
			if parsed < 0 {
				return nil, p.newErrorWithID(traceID, codes.InvalidArgument, fmt.Sprintf("invalid value for 'data_processed_gb': %.2f cannot be negative", parsed), pbc.ErrorCode_ERROR_CODE_INVALID_RESOURCE)
			}
			dataProcessedGB = parsed
		}
	}

	// 3. Calculate Costs
	hourlyCost := pricing.HourlyRate * carbon.HoursPerMonth
	processingCost := dataProcessedGB * pricing.DataProcessingRate
	totalCost := hourlyCost + processingCost

	// 4. Build Billing Detail
	detail := fmt.Sprintf("NAT Gateway, %d hrs/month ($%.3f/hr)", int(carbon.HoursPerMonth), pricing.HourlyRate)
	if tagPresent && dataProcessedGB > 0 {
		detail += fmt.Sprintf(" + %.2f GB data processed ($%.3f/GB)", dataProcessedGB, pricing.DataProcessingRate)
	} else if tagPresent && dataProcessedGB == 0 {
		detail += " (0 GB data processed)"
	} else {
		detail += " (data processing cost not included; use 'data_processed_gb' tag to estimate)"
	}

	p.logger.Debug().
		Float64("hourly_rate", pricing.HourlyRate).
		Float64("data_rate", pricing.DataProcessingRate).
		Float64("data_gb", dataProcessedGB).
		Float64("total_cost", totalCost).
		Msg("NAT Gateway cost estimated")

	resp := &pbc.GetProjectedCostResponse{
		CostPerMonth:  totalCost,
		UnitPrice:     pricing.HourlyRate, // Using hourly rate as primary unit price
		Currency:      "USD",
		BillingDetail: detail,
	}

	// Apply growth hint enrichment
	setGrowthHint(p.logger.With().Str(pluginsdk.FieldTraceID, traceID).Logger(), "aws:ec2:nat-gateway", resp)

	return resp, nil
}

// calculateTieredCost calculates the total cost for a quantity using tiered pricing.
// Tiers are processed in order of their upper bounds (already sorted by parsing).
// For each tier, we calculate the portion that falls within that tier's range.
//
// Example for CloudWatch metrics with 50,000 metrics:
//   - Tier 1: First 10,000 @ $0.30 = $3,000
//   - Tier 2: Next 40,000 @ $0.10 = $4,000
//   - Total: $7,000
func calculateTieredCost(quantity float64, tiers []pricing.TierRate) float64 {
	if len(tiers) == 0 || quantity <= 0 {
		return 0
	}

	totalCost := 0.0
	previousUpperBound := 0.0

	for _, tier := range tiers {
		// This check handles the case where we've already processed all the quantity
		// in earlier tiers. This can occur when:
		// 1. The quantity falls entirely within the first tier (quantity < tier1.UpTo)
		// 2. We've iterated past the tier containing the quantity
		// Without this guard, we'd incorrectly add $0 for subsequent tiers (since
		// tierQuantity would be 0 or negative after clamping).
		if quantity <= previousUpperBound {
			// Already processed all quantity
			break
		}

		// Calculate quantity in this tier
		tierLowerBound := previousUpperBound
		tierUpperBound := tier.UpTo
		if tierUpperBound > quantity {
			tierUpperBound = quantity
		}

		tierQuantity := tierUpperBound - tierLowerBound
		if tierQuantity > 0 {
			totalCost += tierQuantity * tier.Rate
		}

		previousUpperBound = tier.UpTo
	}

	return totalCost
}

// estimateCloudWatch calculates projected monthly cost for CloudWatch resources.
// Supports log ingestion, log storage, and custom metrics.
//
// SKU values:
//   - "logs" or empty: Logs only (ingestion + storage)
//   - "metrics": Custom metrics only
//   - "combined": Both logs and metrics
//
// Tags:
//   - log_ingestion_gb: GB of logs ingested per month
//   - log_storage_gb: GB of logs stored
//   - custom_metrics: Number of custom metrics
func (p *AWSPublicPlugin) estimateCloudWatch(traceID string, resource *pbc.ResourceDescriptor) (*pbc.GetProjectedCostResponse, error) {
	sku := strings.ToLower(resource.Sku)
	if sku == "" {
		sku = "logs" // Default to logs estimation
	}

	// Extract tag values with safe defaults
	logIngestionGB := 0.0
	logStorageGB := 0.0
	customMetrics := 0.0

	if resource.Tags != nil {
		// Parse log_ingestion_gb
		if val, ok := resource.Tags["log_ingestion_gb"]; ok && val != "" {
			parsed, err := strconv.ParseFloat(val, 64)
			if err != nil {
				return nil, p.newErrorWithID(traceID, codes.InvalidArgument,
					fmt.Sprintf("invalid value for 'log_ingestion_gb': %q is not a valid number", val),
					pbc.ErrorCode_ERROR_CODE_INVALID_RESOURCE)
			}
			if parsed < 0 {
				return nil, p.newErrorWithID(traceID, codes.InvalidArgument,
					fmt.Sprintf("invalid value for 'log_ingestion_gb': %.2f cannot be negative", parsed),
					pbc.ErrorCode_ERROR_CODE_INVALID_RESOURCE)
			}
			logIngestionGB = parsed
		}

		// Parse log_storage_gb
		if val, ok := resource.Tags["log_storage_gb"]; ok && val != "" {
			parsed, err := strconv.ParseFloat(val, 64)
			if err != nil {
				return nil, p.newErrorWithID(traceID, codes.InvalidArgument,
					fmt.Sprintf("invalid value for 'log_storage_gb': %q is not a valid number", val),
					pbc.ErrorCode_ERROR_CODE_INVALID_RESOURCE)
			}
			if parsed < 0 {
				return nil, p.newErrorWithID(traceID, codes.InvalidArgument,
					fmt.Sprintf("invalid value for 'log_storage_gb': %.2f cannot be negative", parsed),
					pbc.ErrorCode_ERROR_CODE_INVALID_RESOURCE)
			}
			logStorageGB = parsed
		}

		// Parse custom_metrics
		if val, ok := resource.Tags["custom_metrics"]; ok && val != "" {
			parsed, err := strconv.ParseFloat(val, 64)
			if err != nil {
				return nil, p.newErrorWithID(traceID, codes.InvalidArgument,
					fmt.Sprintf("invalid value for 'custom_metrics': %q is not a valid number", val),
					pbc.ErrorCode_ERROR_CODE_INVALID_RESOURCE)
			}
			if parsed < 0 || parsed > 1e6 {
				return nil, p.newErrorWithID(traceID, codes.InvalidArgument,
					fmt.Sprintf("invalid value for 'custom_metrics': %g must be between 0 and 1000000", parsed),
					pbc.ErrorCode_ERROR_CODE_INVALID_RESOURCE)
			}
			customMetrics = parsed
		}
	}

	// Calculate costs based on SKU
	var totalCost float64
	var details []string

	// Logs cost calculation
	if sku == "logs" || sku == "combined" {
		ingestionCost := 0.0
		storageCost := 0.0

		// Log ingestion (tiered pricing)
		if logIngestionGB > 0 {
			tiers, found := p.pricing.CloudWatchLogsIngestionTiers()
			if found {
				ingestionCost = calculateTieredCost(logIngestionGB, tiers)
				details = append(details, fmt.Sprintf("%.2f GB logs ingested ($%.2f)", logIngestionGB, ingestionCost))
			} else {
				details = append(details, fmt.Sprintf(PricingUnavailableTemplate, "CloudWatch Logs ingestion", p.region))
			}
		}

		// Log storage (flat rate)
		if logStorageGB > 0 {
			storageRate, found := p.pricing.CloudWatchLogsStoragePrice()
			if found {
				storageCost = logStorageGB * storageRate
				details = append(details, fmt.Sprintf("%.2f GB logs stored @ $%.4f/GB-mo ($%.2f)", logStorageGB, storageRate, storageCost))
			} else {
				details = append(details, fmt.Sprintf(PricingUnavailableTemplate, "CloudWatch Logs storage", p.region))
			}
		}

		totalCost += ingestionCost + storageCost
	}

	// Metrics cost calculation
	if sku == "metrics" || sku == "combined" {
		metricsCost := 0.0

		if customMetrics > 0 {
			tiers, found := p.pricing.CloudWatchMetricsTiers()
			if found {
				metricsCost = calculateTieredCost(customMetrics, tiers)
				details = append(details, fmt.Sprintf("%.0f custom metrics ($%.2f)", customMetrics, metricsCost))
			} else {
				details = append(details, fmt.Sprintf(PricingUnavailableTemplate, "CloudWatch Metrics", p.region))
			}
		}

		totalCost += metricsCost
	}

	// Build billing detail
	billingDetail := ""
	if len(details) > 0 {
		billingDetail = "CloudWatch: " + strings.Join(details, ", ")
	} else {
		// No usage provided
		billingDetail = "CloudWatch: No usage specified (use tags: log_ingestion_gb, log_storage_gb, custom_metrics)"
	}

	p.logger.Debug().
		Str("sku", sku).
		Float64("log_ingestion_gb", logIngestionGB).
		Float64("log_storage_gb", logStorageGB).
		Float64("custom_metrics", customMetrics).
		Float64("total_cost", totalCost).
		Msg("CloudWatch cost estimated")

	resp := &pbc.GetProjectedCostResponse{
		CostPerMonth:  totalCost,
		UnitPrice:     0, // No single unit price for CloudWatch (multi-component)
		Currency:      "USD",
		BillingDetail: billingDetail,
	}

	// Apply growth hint enrichment
	setGrowthHint(p.logger.With().Str(pluginsdk.FieldTraceID, traceID).Logger(), "aws:cloudwatch:metric", resp)

	return resp, nil
}

// estimateElastiCache calculates projected monthly cost for ElastiCache clusters.
//
// ElastiCache pricing is based on:
//   - Node type (e.g., cache.m5.large, cache.t3.micro)
//   - Cache engine (Redis, Memcached, Valkey)
//   - Number of nodes (default: 1)
//
// Cost formula: hourly_rate × num_nodes × 730 hours/month
//
// Required fields:
//   - resource.Sku: The cache node type (e.g., "cache.m5.large")
//
// Optional tags:
//   - "engine": Cache engine - "redis" (default), "memcached", or "valkey" (open-source Redis fork)
//   - "num_nodes" or "num_cache_nodes": Number of cache nodes (default: 1)
func (p *AWSPublicPlugin) estimateElastiCache(traceID string, resource *pbc.ResourceDescriptor) (*pbc.GetProjectedCostResponse, error) {
	// Extract node type from SKU
	nodeType := resource.Sku
	if nodeType == "" {
		// Try extracting from tags using extractAWSSKU pattern
		nodeType = extractAWSSKU(resource.Tags)
	}
	if nodeType == "" {
		return nil, p.newErrorWithID(traceID, codes.InvalidArgument,
			"ElastiCache node type not specified: use 'sku' field or 'instanceType' tag",
			pbc.ErrorCode_ERROR_CODE_INVALID_RESOURCE)
	}

	// Extract engine (default: redis)
	engine := "redis"
	if resource.Tags != nil {
		if val, ok := resource.Tags["engine"]; ok && val != "" {
			engine = strings.ToLower(val)
		}
	}

	// Extract number of nodes (default: 1)
	numNodes := 1
	if resource.Tags != nil {
		// Try num_nodes first, then num_cache_nodes
		nodeCountStr := ""
		if val, ok := resource.Tags["num_nodes"]; ok && val != "" {
			nodeCountStr = val
		} else if val, ok := resource.Tags["num_cache_nodes"]; ok && val != "" {
			nodeCountStr = val
		}
		if nodeCountStr != "" {
			parsed, err := strconv.Atoi(nodeCountStr)
			if err != nil {
				return nil, p.newErrorWithID(traceID, codes.InvalidArgument,
					fmt.Sprintf("invalid value for node count: %q is not a valid integer", nodeCountStr),
					pbc.ErrorCode_ERROR_CODE_INVALID_RESOURCE)
			}
			if parsed < 1 || parsed > 1000 {
				return nil, p.newErrorWithID(traceID, codes.InvalidArgument,
					fmt.Sprintf("invalid value for node count: %d must be between 1 and 1000", parsed),
					pbc.ErrorCode_ERROR_CODE_INVALID_RESOURCE)
			}
			numNodes = parsed
		}
	}

	// Look up hourly rate from pricing client
	hourlyRate, found := p.pricing.ElastiCacheOnDemandPricePerHour(nodeType, engine)
	if !found {
		// Unknown node type or engine - return $0 with explanation
		return &pbc.GetProjectedCostResponse{
			CostPerMonth:  0,
			UnitPrice:     0,
			Currency:      "USD",
			BillingDetail: fmt.Sprintf(PricingNotFoundTemplate, fmt.Sprintf("ElastiCache %s node", engine), nodeType),
		}, nil
	}

	// Calculate monthly cost: hourly_rate × num_nodes × hours_per_month
	monthlyCost := hourlyRate * float64(numNodes) * carbon.HoursPerMonth

	// Build billing detail
	var billingDetail string
	if numNodes == 1 {
		billingDetail = fmt.Sprintf("ElastiCache %s (%s), 1 node, 730 hrs/month", nodeType, engine)
	} else {
		billingDetail = fmt.Sprintf("ElastiCache %s (%s), %d nodes, 730 hrs/month", nodeType, engine, numNodes)
	}

	p.logger.Debug().
		Str("node_type", nodeType).
		Str("engine", engine).
		Int("num_nodes", numNodes).
		Float64("hourly_rate", hourlyRate).
		Float64("monthly_cost", monthlyCost).
		Msg("ElastiCache cost estimated")

	resp := &pbc.GetProjectedCostResponse{
		CostPerMonth:  monthlyCost,
		UnitPrice:     hourlyRate,
		Currency:      "USD",
		BillingDetail: billingDetail,
	}

	// Carbon estimation for ElastiCache cluster
	elasticacheEstimator := carbon.NewElastiCacheEstimator()
	carbonGrams, carbonOK := elasticacheEstimator.EstimateCarbonGrams(carbon.ElastiCacheConfig{
		NodeType:    nodeType,
		Engine:      engine,
		Nodes:       numNodes,
		Region:      resource.Region,
		Utilization: carbon.DefaultUtilization, // Use CCF default (50%)
		Hours:       carbon.HoursPerMonth,
	})

	if carbonOK {
		resp.ImpactMetrics = []*pbc.ImpactMetric{
			{
				Kind:  pbc.MetricKind_METRIC_KIND_CARBON_FOOTPRINT,
				Value: carbonGrams,
				Unit:  "gCO2e",
			},
		}

		p.traceLogger(traceID, "GetProjectedCost").Debug().
			Str("node_type", nodeType).
			Int("num_nodes", numNodes).
			Str("aws_region", resource.Region).
			Float64("carbon_grams", carbonGrams).
			Msg("ElastiCache carbon estimation successful")
	}

	// Apply growth hint enrichment
	setGrowthHint(p.logger.With().Str(pluginsdk.FieldTraceID, traceID).Logger(), "aws:elasticache:cluster", resp)

	return resp, nil
}

// zeroCostResourceDescriptions provides billing detail messages for resources with no direct AWS charges.
var zeroCostResourceDescriptions = map[string]string{
	"vpc":           "VPC has no direct hourly or monthly charge. Costs may apply for associated resources (NAT Gateway, VPN, etc.)",
	"securitygroup": "Security Groups have no direct charge. They are a free networking feature.",
	"subnet":        "Subnets have no direct charge. Costs may apply for data transfer between AZs.",
	"iam":           "IAM resources (users, roles, policies) have no direct charge. They are a free AWS feature.",
}

// estimateZeroCostResource returns a $0 cost estimate for AWS resources that have no direct charges.
// These include VPC, Security Groups, and Subnets - fundamental networking resources that are
// free to create but may have associated costs from other resources.
func (p *AWSPublicPlugin) estimateZeroCostResource(traceID string, resource *pbc.ResourceDescriptor, serviceType string) *pbc.GetProjectedCostResponse {
	description, ok := zeroCostResourceDescriptions[serviceType]
	if !ok {
		description = fmt.Sprintf("%s has no direct AWS charge", serviceType)
	}

	p.logger.Debug().
		Str(pluginsdk.FieldTraceID, traceID).
		Str("service_type", serviceType).
		Str("original_resource_type", resource.ResourceType).
		Msg("Zero-cost resource estimated")

	resp := &pbc.GetProjectedCostResponse{
		CostPerMonth:  0,
		UnitPrice:     0,
		Currency:      "USD",
		BillingDetail: description,
	}

	// Apply growth hint enrichment (static for networking resources)
	setGrowthHint(p.logger.With().Str(pluginsdk.FieldTraceID, traceID).Logger(), resource.ResourceType, resp)

	return resp
}
