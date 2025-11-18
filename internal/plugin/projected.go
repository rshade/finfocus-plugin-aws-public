package plugin

import (
	"context"
	"fmt"
	"strconv"

	pbc "github.com/rshade/pulumicost-spec/sdk/go/proto/pulumicost/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	hoursPerMonth = 730.0
	defaultEBSGB  = 8
)

// GetProjectedCost estimates the monthly cost for the given resource.
func (p *AWSPublicPlugin) GetProjectedCost(ctx context.Context, req *pbc.GetProjectedCostRequest) (*pbc.GetProjectedCostResponse, error) {
	if req == nil || req.Resource == nil {
		return nil, status.Error(codes.InvalidArgument, "missing resource descriptor")
	}

	resource := req.Resource

	// FR-029: Validate required fields
	if resource.Provider == "" || resource.ResourceType == "" || resource.Sku == "" || resource.Region == "" {
		return nil, status.Error(codes.InvalidArgument, "resource descriptor missing required fields (provider, resource_type, sku, region)")
	}

	// FR-027 & FR-028: Check region match
	if resource.Region != p.region {
		// Create error details map
		details := map[string]string{
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
		return nil, st.Err()
	}

	// Route to appropriate estimator based on resource type
	switch resource.ResourceType {
	case "ec2":
		return p.estimateEC2(ctx, resource)
	case "ebs":
		return p.estimateEBS(ctx, resource)
	case "s3", "lambda", "rds", "dynamodb":
		return p.estimateStub(ctx, resource)
	default:
		// Unknown resource type - return $0 with explanation
		return &pbc.GetProjectedCostResponse{
			CostPerMonth: 0,
			UnitPrice:    0,
			Currency:     "USD",
			BillingDetail: fmt.Sprintf("Resource type %q not supported for cost estimation", resource.ResourceType),
		}, nil
	}
}

// estimateEC2 calculates the projected monthly cost for an EC2 instance.
func (p *AWSPublicPlugin) estimateEC2(ctx context.Context, resource *pbc.ResourceDescriptor) (*pbc.GetProjectedCostResponse, error) {
	instanceType := resource.Sku

	// Hardcoded assumptions for v1
	os := "Linux"
	tenancy := "Shared"

	// FR-020: Lookup pricing using embedded data
	hourlyRate, found := p.pricing.EC2OnDemandPricePerHour(instanceType, os, tenancy)
	if !found {
		// FR-035: Unknown instance types return $0 with explanation
		return &pbc.GetProjectedCostResponse{
			CostPerMonth: 0,
			UnitPrice:    0,
			Currency:     "USD",
			BillingDetail: fmt.Sprintf("EC2 instance type %q not found in pricing data for %s/%s", instanceType, os, tenancy),
		}, nil
	}

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
func (p *AWSPublicPlugin) estimateEBS(ctx context.Context, resource *pbc.ResourceDescriptor) (*pbc.GetProjectedCostResponse, error) {
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
		return &pbc.GetProjectedCostResponse{
			CostPerMonth:  0,
			UnitPrice:     0,
			Currency:      "USD",
			BillingDetail: fmt.Sprintf("EBS volume type %q not found in pricing data", volumeType),
		}, nil
	}

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
func (p *AWSPublicPlugin) estimateStub(ctx context.Context, resource *pbc.ResourceDescriptor) (*pbc.GetProjectedCostResponse, error) {
	// FR-025 & FR-026: Return $0 with explanation
	return &pbc.GetProjectedCostResponse{
		CostPerMonth:  0,
		UnitPrice:     0,
		Currency:      "USD",
		BillingDetail: fmt.Sprintf("%s cost estimation not fully implemented - returns $0 estimate", resource.ResourceType),
	}, nil
}
