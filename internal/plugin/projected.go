package plugin

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/rshade/pulumicost-spec/sdk/go/pluginsdk"
	pbc "github.com/rshade/pulumicost-spec/sdk/go/proto/pulumicost/v1"
)

const (
	hoursPerMonth     = 730.0
	defaultEBSGB      = 8
	defaultRDSEngine  = "mysql"
	defaultRDSStorage = "gp2"
	defaultRDSSizeGB  = 20
)

const (
	resTypeEC2Instance          = "aws:ec2/instance:Instance"
	resTypeEC2InstanceLegacy    = "ec2/instance" // Legacy format without aws: prefix
	resTypeEBSVolume            = "aws:ebs/volume:Volume"
	resTypeEBSVolumeLegacy      = "ec2/volume" // Legacy format
	resTypeRDSInstance          = "aws:rds/instance:Instance"
	resTypeRDSInstanceLegacy    = "rds/instance" // Legacy format
	resTypeS3Bucket             = "aws:s3/bucket:Bucket"
	resTypeS3BucketLegacy       = "s3/bucket" // Legacy format
	resTypeLambdaFunction       = "aws:lambda/function:Function"
	resTypeLambdaFunctionLegacy = "lambda/function" // Legacy format
	resTypeDynamoDBTable        = "aws:dynamodb/table:Table"
	resTypeDynamoDBTableLegacy  = "dynamodb/table" // Legacy format
	resTypeEKSCluster           = "aws:eks/cluster:Cluster"
	resTypeEKSClusterLegacy     = "eks/cluster" // Legacy format
)

// extractAWSSKU extracts AWS SKU from tags with priority: instanceType > instance_class > type > volumeType > volume_type
// This implements the same logic as SDK mapping.ExtractAWSSKU() until it's available
func extractAWSSKU(tags map[string]string) string {
	if tags == nil {
		return ""
	}

	// Priority order: instanceType > instance_class > type > volumeType > volume_type
	keys := []string{"instanceType", "instance_class", "type", "volumeType", "volume_type"}
	for _, key := range keys {
		if value, ok := tags[key]; ok && value != "" {
			return value
		}
	}

	return ""
}

// extractAWSRegion extracts AWS region from tags with priority: region > availabilityZone (with AZ parsing)
// This implements the same logic as SDK mapping.ExtractAWSRegion()
func extractAWSRegion(tags map[string]string) string {
	if tags == nil {
		return ""
	}

	// Check explicit region first (SDK priority)
	if region, ok := tags["region"]; ok && region != "" {
		return region
	}

	// Try to derive from availability zone
	if az, ok := tags["availabilityZone"]; ok && az != "" {
		return extractAWSRegionFromAZ(az)
	}

	return ""
}

// extractAWSRegionFromAZ derives the AWS region from an availability zone string.
// Matches SDK behavior: removes trailing lowercase letter, returns input as-is if no trailing letter.
func extractAWSRegionFromAZ(az string) string {
	if az == "" {
		return ""
	}

	length := len(az)
	// Single character cannot be a valid AZ (e.g., just "a")
	// Valid AZ format is at minimum "region" + "az-letter" (e.g., "us-east-1a")
	if length == 1 {
		return ""
	}

	// If the last character is a lowercase letter, remove it
	lastChar := az[length-1]
	if lastChar >= 'a' && lastChar <= 'z' {
		return az[:length-1]
	}

	// Return as-is if no trailing letter (might already be a region)
	return az
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

	serviceType := detectService(resource.ResourceType)
	switch serviceType {
	case "ec2":
		resp, err = p.estimateEC2(traceID, resource)
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
		resp, err = p.estimateStub(resource)
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
	p.logger.Info().
		Str(pluginsdk.FieldTraceID, traceID).
		Str(pluginsdk.FieldOperation, "GetProjectedCost").
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
func (p *AWSPublicPlugin) estimateEC2(traceID string, resource *pbc.ResourceDescriptor) (*pbc.GetProjectedCostResponse, error) {
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
		p.logger.Debug().
			Str(pluginsdk.FieldTraceID, traceID).
			Str(pluginsdk.FieldOperation, "GetProjectedCost").
			Str("instance_type", instanceType).
			Str("aws_region", p.region).
			Str("pricing_source", "embedded").
			Msg("EC2 instance type not found in pricing data")

		return &pbc.GetProjectedCostResponse{
			CostPerMonth:  0,
			UnitPrice:     0,
			Currency:      "USD",
			BillingDetail: fmt.Sprintf("EC2 instance type %q not found in pricing data for %s/%s", instanceType, ec2Attrs.OS, ec2Attrs.Tenancy),
		}, nil
	}

	// Debug log successful lookup
	p.logger.Debug().
		Str(pluginsdk.FieldTraceID, traceID).
		Str(pluginsdk.FieldOperation, "GetProjectedCost").
		Str("instance_type", instanceType).
		Str("aws_region", p.region).
		Str("pricing_source", "embedded").
		Float64("unit_price", hourlyRate).
		Msg("EC2 pricing lookup successful")

	// FR-021: Calculate monthly cost (730 hours/month)
	costPerMonth := hourlyRate * hoursPerMonth

	// FR-022, FR-023, FR-024: Return response with all required fields
	return &pbc.GetProjectedCostResponse{
		CostPerMonth:  costPerMonth,
		UnitPrice:     hourlyRate,
		Currency:      "USD",
		BillingDetail: fmt.Sprintf("On-demand %s, %s tenancy, 730 hrs/month", ec2Attrs.OS, ec2Attrs.Tenancy),
	}, nil
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
		p.logger.Debug().
			Str(pluginsdk.FieldTraceID, traceID).
			Str(pluginsdk.FieldOperation, "GetProjectedCost").
			Str("storage_type", volumeType).
			Str("aws_region", p.region).
			Str("pricing_source", "embedded").
			Msg("EBS volume type not found in pricing data")

		return &pbc.GetProjectedCostResponse{
			CostPerMonth:  0,
			UnitPrice:     0,
			Currency:      "USD",
			BillingDetail: fmt.Sprintf("EBS volume type %q not found in pricing data", volumeType),
		}, nil
	}

	// Debug log successful lookup
	p.logger.Debug().
		Str(pluginsdk.FieldTraceID, traceID).
		Str(pluginsdk.FieldOperation, "GetProjectedCost").
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

	// FR-022, FR-023, FR-024: Return response
	return &pbc.GetProjectedCostResponse{
		CostPerMonth:  costPerMonth,
		UnitPrice:     ratePerGBMonth,
		Currency:      "USD",
		BillingDetail: billingDetail,
	}, nil
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
		p.logger.Debug().
			Str(pluginsdk.FieldTraceID, traceID).
			Str(pluginsdk.FieldOperation, "GetProjectedCost").
			Str("storage_class", storageClass).
			Str("aws_region", p.region).
			Str("pricing_source", "embedded").
			Msg("S3 storage class not found in pricing data")

		return &pbc.GetProjectedCostResponse{
			CostPerMonth:  0,
			UnitPrice:     0,
			Currency:      "USD",
			BillingDetail: fmt.Sprintf("S3 storage class %q not found in pricing data", storageClass),
		}, nil
	}

	// Debug log successful lookup
	p.logger.Debug().
		Str(pluginsdk.FieldTraceID, traceID).
		Str(pluginsdk.FieldOperation, "GetProjectedCost").
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

	return &pbc.GetProjectedCostResponse{
		CostPerMonth:  costPerMonth,
		UnitPrice:     ratePerGBMonth,
		Currency:      "USD",
		BillingDetail: billingDetail,
	}, nil
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

	// Lookup instance hourly rate
	hourlyRate, found := p.pricing.RDSOnDemandPricePerHour(instanceType, normalizedEngine)
	if !found {
		// Unknown instance type - return $0 with explanation
		p.logger.Debug().
			Str(pluginsdk.FieldTraceID, traceID).
			Str(pluginsdk.FieldOperation, "GetProjectedCost").
			Str("instance_type", instanceType).
			Str("engine", normalizedEngine).
			Str("aws_region", p.region).
			Str("pricing_source", "embedded").
			Msg("RDS instance type not found in pricing data")

		return &pbc.GetProjectedCostResponse{
			CostPerMonth:  0,
			UnitPrice:     0,
			Currency:      "USD",
			BillingDetail: fmt.Sprintf("RDS instance type %q not found in pricing data for %s", instanceType, normalizedEngine),
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
		Str(pluginsdk.FieldTraceID, traceID).
		Str(pluginsdk.FieldOperation, "GetProjectedCost").
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
	instanceCostPerMonth := hourlyRate * hoursPerMonth
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

	return &pbc.GetProjectedCostResponse{
		CostPerMonth:  totalCostPerMonth,
		UnitPrice:     hourlyRate,
		Currency:      "USD",
		BillingDetail: billingDetail,
	}, nil
}

// detectService maps a provider resource type string to a normalized service identifier.
// It returns one of "ec2", "ebs", "rds", "s3", "lambda", or "dynamodb" when a known mapping or pattern is found;
// otherwise it returns the original resourceType unchanged.
func detectService(resourceType string) string {
	resourceTypeLower := strings.ToLower(resourceType)

	switch resourceTypeLower {
	case "ec2", resTypeEC2Instance, resTypeEC2InstanceLegacy:
		return "ec2"
	case "ebs", resTypeEBSVolume, resTypeEBSVolumeLegacy:
		return "ebs"
	case "rds", resTypeRDSInstance, resTypeRDSInstanceLegacy:
		return "rds"
	case "s3", resTypeS3Bucket, resTypeS3BucketLegacy:
		return "s3"
	case "lambda", resTypeLambdaFunction, resTypeLambdaFunctionLegacy:
		return "lambda"
	case "dynamodb", resTypeDynamoDBTable, resTypeDynamoDBTableLegacy:
		return "dynamodb"
	case "eks", resTypeEKSCluster, resTypeEKSClusterLegacy:
		return "eks"
	}

	// Fallback: simple containment check for common patterns
	if strings.Contains(resourceTypeLower, "ec2/instance") || strings.Contains(resourceTypeLower, "aws:ec2:instance") {
		return "ec2"
	}
	if strings.Contains(resourceTypeLower, "ebs/volume") || strings.Contains(resourceTypeLower, "ec2/volume") || strings.Contains(resourceTypeLower, "aws:ebs:volume") {
		return "ebs"
	}
	if strings.Contains(resourceTypeLower, "rds/instance") {
		return "rds"
	}
	if strings.Contains(resourceTypeLower, "eks/cluster") {
		return "eks"
	}
	if strings.Contains(resourceTypeLower, "s3/bucket") || strings.Contains(resourceTypeLower, "aws:s3:bucket") {
		return "s3"
	}
	if strings.Contains(resourceTypeLower, "lambda/function") || strings.Contains(resourceTypeLower, "aws:lambda:function") {
		return "lambda"
	}
	if strings.Contains(resourceTypeLower, "dynamodb/table") || strings.Contains(resourceTypeLower, "aws:dynamodb:table") {
		return "dynamodb"
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
		p.logger.Debug().
			Str(pluginsdk.FieldTraceID, traceID).
			Str(pluginsdk.FieldOperation, "GetProjectedCost").
			Str("aws_region", p.region).
			Bool("extended_support", extendedSupport).
			Msg("EKS pricing data not found")

		return &pbc.GetProjectedCostResponse{
			CostPerMonth:  0,
			UnitPrice:     0,
			Currency:      "USD",
			BillingDetail: "EKS cluster pricing data not available for this region",
		}, nil
	}

	// Debug log successful lookup
	p.logger.Debug().
		Str(pluginsdk.FieldTraceID, traceID).
		Str(pluginsdk.FieldOperation, "GetProjectedCost").
		Str("aws_region", p.region).
		Bool("extended_support", extendedSupport).
		Float64("hourly_rate", hourlyRate).
		Msg("EKS pricing lookup successful")

	// Calculate monthly cost (730 hours/month)
	costPerMonth := hourlyRate * hoursPerMonth

	// Determine support type description
	supportType := "standard support"
	if extendedSupport {
		supportType = "extended support"
	}

	// Return response with billing details
	return &pbc.GetProjectedCostResponse{
		CostPerMonth:  costPerMonth,
		UnitPrice:     hourlyRate,
		Currency:      "USD",
		BillingDetail: fmt.Sprintf("EKS cluster (%s), 730 hrs/month (control plane only, excludes worker nodes)", supportType),
	}, nil
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
		p.logger.Debug().
			Str(pluginsdk.FieldTraceID, traceID).
			Str(pluginsdk.FieldOperation, "GetProjectedCost").
			Str("aws_region", p.region).
			Str("architecture", architecture).
			Msg("Lambda pricing data not found")

		return &pbc.GetProjectedCostResponse{
			CostPerMonth:  0,
			UnitPrice:     0,
			Currency:      "USD",
			BillingDetail: "Lambda pricing data not available for this region",
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
		Str(pluginsdk.FieldTraceID, traceID).
		Str(pluginsdk.FieldOperation, "GetProjectedCost").
		Int("memory_mb", memoryMB).
		Str("architecture", archDisplay).
		Int64("requests", requestsPerMonth).
		Int("duration_ms", avgDurationMs).
		Float64("gb_seconds", totalGBSec).
		Float64("total_cost", totalCost).
		Msg("Lambda cost estimated")

	return &pbc.GetProjectedCostResponse{
		CostPerMonth:  totalCost,
		UnitPrice:     gbSecPrice, // Using GB-second price as unit price
		Currency:      "USD",
		BillingDetail: detail,
	}, nil
}

