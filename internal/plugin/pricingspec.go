package plugin

import (
	"context"
	"fmt"
	"time"

	"github.com/rshade/pulumicost-spec/sdk/go/pluginsdk"
	pbc "github.com/rshade/pulumicost-spec/sdk/go/proto/pulumicost/v1"
	"google.golang.org/grpc/codes"
)

// GetPricingSpec returns detailed pricing specification for a resource type.
// This provides information about how a resource is billed without calculating the actual cost.
func (p *AWSPublicPlugin) GetPricingSpec(ctx context.Context, req *pbc.GetPricingSpecRequest) (*pbc.GetPricingSpecResponse, error) {
	start := time.Now()
	traceID := p.getTraceID(ctx)

	if req == nil || req.Resource == nil {
		err := p.newErrorWithID(traceID, codes.InvalidArgument, "missing resource descriptor", pbc.ErrorCode_ERROR_CODE_INVALID_RESOURCE)
		p.logErrorWithID(traceID, "GetPricingSpec", err, pbc.ErrorCode_ERROR_CODE_INVALID_RESOURCE)
		return nil, err
	}

	resource := req.Resource

	// Validate required fields
	if resource.Provider == "" || resource.ResourceType == "" || resource.Sku == "" || resource.Region == "" {
		err := p.newErrorWithID(traceID, codes.InvalidArgument, "resource descriptor missing required fields (provider, resource_type, sku, region)", pbc.ErrorCode_ERROR_CODE_INVALID_RESOURCE)
		p.logErrorWithID(traceID, "GetPricingSpec", err, pbc.ErrorCode_ERROR_CODE_INVALID_RESOURCE)
		return nil, err
	}

	// Check region match
	if resource.Region != p.region {
		details := map[string]string{
			"trace_id":       traceID,
			"pluginRegion":   p.region,
			"requiredRegion": resource.Region,
		}
		errDetail := &pbc.ErrorDetail{
			Code:    pbc.ErrorCode_ERROR_CODE_UNSUPPORTED_REGION,
			Message: fmt.Sprintf("Resource region %q does not match plugin region %q", resource.Region, p.region),
			Details: details,
		}
		err := p.newErrorWithID(traceID, codes.FailedPrecondition, errDetail.Message, pbc.ErrorCode_ERROR_CODE_UNSUPPORTED_REGION)
		p.logErrorWithID(traceID, "GetPricingSpec", err, pbc.ErrorCode_ERROR_CODE_UNSUPPORTED_REGION)
		return nil, err
	}

	// Normalize resource type (handles Pulumi formats like aws:ec2/instance:Instance)
	serviceType := detectService(resource.ResourceType)

	var spec *pbc.PricingSpec

	switch serviceType {
	case "ec2":
		spec = p.ec2PricingSpec(resource)
	case "ebs":
		spec = p.ebsPricingSpec(resource)
	case "s3", "lambda", "rds", "dynamodb":
		spec = p.stubPricingSpec(resource)
	default:
		spec = &pbc.PricingSpec{
			Provider:     resource.Provider,
			ResourceType: resource.ResourceType,
			Sku:          resource.Sku,
			Region:       resource.Region,
			BillingMode:  "unknown",
			RatePerUnit:  0,
			Currency:     "USD",
			Description:  fmt.Sprintf("Resource type %q not supported for pricing specification", resource.ResourceType),
			Source:       "aws-public",
		}
	}

	p.logger.Info().
		Str(pluginsdk.FieldTraceID, traceID).
		Str(pluginsdk.FieldOperation, "GetPricingSpec").
		Str(pluginsdk.FieldResourceType, resource.ResourceType).
		Str("aws_region", resource.Region).
		Int64(pluginsdk.FieldDurationMs, time.Since(start).Milliseconds()).
		Msg("pricing spec retrieved")

	return &pbc.GetPricingSpecResponse{
		Spec: spec,
	}, nil
}

// ec2PricingSpec returns the pricing specification for an EC2 instance.
func (p *AWSPublicPlugin) ec2PricingSpec(resource *pbc.ResourceDescriptor) *pbc.PricingSpec {
	instanceType := resource.Sku
	os := "Linux"
	tenancy := "Shared"

	hourlyRate, found := p.pricing.EC2OnDemandPricePerHour(instanceType, os, tenancy)
	if !found {
		return &pbc.PricingSpec{
			Provider:     resource.Provider,
			ResourceType: resource.ResourceType,
			Sku:          resource.Sku,
			Region:       resource.Region,
			BillingMode:  "per_hour",
			RatePerUnit:  0,
			Currency:     "USD",
			Unit:         "hour",
			Description:  fmt.Sprintf("EC2 instance type %q not found in pricing data", instanceType),
			Source:       "aws-public",
			Assumptions:  []string{"Instance type not found in embedded pricing data"},
		}
	}

	return &pbc.PricingSpec{
		Provider:     resource.Provider,
		ResourceType: resource.ResourceType,
		Sku:          resource.Sku,
		Region:       resource.Region,
		BillingMode:  "per_hour",
		RatePerUnit:  hourlyRate,
		Currency:     "USD",
		Unit:         "hour",
		Description:  fmt.Sprintf("On-demand %s EC2 instance with %s tenancy", os, tenancy),
		Source:       "aws-public",
		Assumptions: []string{
			fmt.Sprintf("Operating System: %s", os),
			fmt.Sprintf("Tenancy: %s", tenancy),
			"Pre-installed software: None",
			"Capacity Status: Used",
		},
	}
}

// ebsPricingSpec returns the pricing specification for an EBS volume.
func (p *AWSPublicPlugin) ebsPricingSpec(resource *pbc.ResourceDescriptor) *pbc.PricingSpec {
	volumeType := resource.Sku

	ratePerGBMonth, found := p.pricing.EBSPricePerGBMonth(volumeType)
	if !found {
		return &pbc.PricingSpec{
			Provider:     resource.Provider,
			ResourceType: resource.ResourceType,
			Sku:          resource.Sku,
			Region:       resource.Region,
			BillingMode:  "per_gb_month",
			RatePerUnit:  0,
			Currency:     "USD",
			Unit:         "GB-month",
			Description:  fmt.Sprintf("EBS volume type %q not found in pricing data", volumeType),
			Source:       "aws-public",
			Assumptions:  []string{"Volume type not found in embedded pricing data"},
		}
	}

	return &pbc.PricingSpec{
		Provider:     resource.Provider,
		ResourceType: resource.ResourceType,
		Sku:          resource.Sku,
		Region:       resource.Region,
		BillingMode:  "per_gb_month",
		RatePerUnit:  ratePerGBMonth,
		Currency:     "USD",
		Unit:         "GB-month",
		Description:  fmt.Sprintf("EBS %s storage", volumeType),
		Source:       "aws-public",
		Assumptions: []string{
			"Storage only (IOPS/throughput not included)",
			"Standard provisioned capacity",
		},
	}
}

// stubPricingSpec returns a placeholder pricing specification for unsupported services.
func (p *AWSPublicPlugin) stubPricingSpec(resource *pbc.ResourceDescriptor) *pbc.PricingSpec {
	return &pbc.PricingSpec{
		Provider:     resource.Provider,
		ResourceType: resource.ResourceType,
		Sku:          resource.Sku,
		Region:       resource.Region,
		BillingMode:  "unknown",
		RatePerUnit:  0,
		Currency:     "USD",
		Description:  fmt.Sprintf("%s pricing specification not fully implemented", resource.ResourceType),
		Source:       "aws-public",
		Assumptions:  []string{"Service not fully supported - returns placeholder pricing"},
	}
}
