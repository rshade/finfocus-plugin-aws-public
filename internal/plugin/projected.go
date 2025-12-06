package plugin

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/rshade/pulumicost-spec/sdk/go/pluginsdk"
	pbc "github.com/rshade/pulumicost-spec/sdk/go/proto/pulumicost/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	hoursPerMonth     = 730.0
	defaultEBSGB      = 8
	defaultRDSEngine  = "mysql"
	defaultRDSStorage = "gp2"
	defaultRDSSizeGB  = 20
)

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

	if req == nil || req.Resource == nil {
		err := p.newErrorWithID(traceID, codes.InvalidArgument, "missing resource descriptor", pbc.ErrorCode_ERROR_CODE_INVALID_RESOURCE)
		p.logErrorWithID(traceID, "GetProjectedCost", err, pbc.ErrorCode_ERROR_CODE_INVALID_RESOURCE)
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

	// FR-029: Validate required fields
	if resource.Provider == "" || resource.ResourceType == "" || resource.Sku == "" || resource.Region == "" {
		err := p.newErrorWithID(traceID, codes.InvalidArgument, "resource descriptor missing required fields (provider, resource_type, sku, region)", pbc.ErrorCode_ERROR_CODE_INVALID_RESOURCE)
		p.logErrorWithID(traceID, "GetProjectedCost", err, pbc.ErrorCode_ERROR_CODE_INVALID_RESOURCE)
		return nil, err
	}

	// FR-027 & FR-028: Check region match
	if resource.Region != p.region {
		// Create error details map with trace_id
		details := map[string]string{
			"trace_id":       traceID,
			"pluginRegion":   p.region,
			"requiredRegion": resource.Region,
		}

		// Create ErrorDetail
		errDetail := &pbc.ErrorDetail{
			Code:    pbc.ErrorCode_ERROR_CODE_UNSUPPORTED_REGION,
			Message: fmt.Sprintf("Resource region %q does not match plugin region %q", resource.Region, p.region),
			Details: details,
		}

		// Return error with details
		st := status.New(codes.FailedPrecondition, errDetail.Message)
		st, _ = st.WithDetails(errDetail)
		p.logErrorWithID(traceID, "GetProjectedCost", st.Err(), pbc.ErrorCode_ERROR_CODE_UNSUPPORTED_REGION)
		return nil, st.Err()
	}

	// Route to appropriate estimator based on resource type
	var resp *pbc.GetProjectedCostResponse
	var err error

	switch resource.ResourceType {
	case "ec2":
		resp, err = p.estimateEC2(traceID, resource)
	case "ebs":
		resp, err = p.estimateEBS(traceID, resource)
	case "rds":
		resp, err = p.estimateRDS(traceID, resource)
	case "s3", "lambda", "dynamodb":
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
	instanceType := resource.Sku

	// Extract OS and tenancy from resource tags, with fallbacks
	os := "Linux"       // Default fallback
	tenancy := "Shared" // Default fallback

	// Check for OS information in tags
	if resource.Tags != nil {
		// Common tags that might indicate OS
		if platform, ok := resource.Tags["platform"]; ok && platform != "" {
			// AWS uses "windows" for Windows platforms, otherwise assume Linux-based
			if strings.ToLower(platform) == "windows" {
				os = "Windows"
			} else {
				os = "Linux" // Treat other platforms as Linux-based
			}
		}

		// Check for tenancy information in tags
		if tenancyTag, ok := resource.Tags["tenancy"]; ok && tenancyTag != "" {
			// Validate tenancy values (AWS supports Shared, Dedicated, Host)
			switch strings.ToLower(tenancyTag) {
			case "dedicated":
				tenancy = "Dedicated"
			case "host":
				tenancy = "Host"
			default:
				tenancy = "Shared" // Default to Shared for any other value
			}
		}
	}

	// FR-020: Lookup pricing using embedded data
	hourlyRate, found := p.pricing.EC2OnDemandPricePerHour(instanceType, os, tenancy)
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
			BillingDetail: fmt.Sprintf("EC2 instance type %q not found in pricing data for %s/%s", instanceType, os, tenancy),
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
		BillingDetail: fmt.Sprintf("On-demand %s, %s tenancy, 730 hrs/month", os, tenancy),
	}, nil
}

// estimateEBS calculates the projected monthly cost for an EBS volume.
// traceID is passed from the parent handler to ensure consistent trace correlation.
func (p *AWSPublicPlugin) estimateEBS(traceID string, resource *pbc.ResourceDescriptor) (*pbc.GetProjectedCostResponse, error) {
	volumeType := resource.Sku

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
	instanceType := resource.Sku

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
