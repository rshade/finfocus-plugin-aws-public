package plugin

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/rshade/pulumicost-plugin-aws-public/internal/pricing"
	pbc "github.com/rshade/pulumicost-spec/sdk/go/proto/pulumicost/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// AWSPublicPlugin implements the pluginsdk.Plugin interface for AWS public pricing.
type AWSPublicPlugin struct {
	region  string
	pricing pricing.PricingClient
}

// NewAWSPublicPlugin creates a new AWSPublicPlugin instance.
// The region should match the region for which pricing data is embedded.
func NewAWSPublicPlugin(region string, pricingClient pricing.PricingClient) *AWSPublicPlugin {
	return &AWSPublicPlugin{
		region:  region,
		pricing: pricingClient,
	}
}

// Name returns the plugin name identifier.
func (p *AWSPublicPlugin) Name() string {
	return "pulumicost-plugin-aws-public"
}

// GetActualCost retrieves actual cost for a resource based on runtime.
// Uses fallback formula: actual_cost = projected_monthly_cost × (runtime_hours / 730)
//
// The proto API uses ResourceId (string) which we expect to be a JSON-encoded
// ResourceDescriptor. If ResourceId is empty, we fall back to extracting
// resource info from the Tags map.
func (p *AWSPublicPlugin) GetActualCost(ctx context.Context, req *pbc.GetActualCostRequest) (*pbc.GetActualCostResponse, error) {
	// Validate request
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "missing request")
	}

	// Validate timestamps (proto uses Start/End)
	if req.Start == nil {
		return nil, status.Error(codes.InvalidArgument, "missing Start timestamp")
	}
	if req.End == nil {
		return nil, status.Error(codes.InvalidArgument, "missing End timestamp")
	}

	// Parse timestamps
	fromTime := req.Start.AsTime()
	toTime := req.End.AsTime()

	// Calculate runtime hours
	runtimeHours, err := calculateRuntimeHours(fromTime, toTime)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid time range: %v", err))
	}

	// Parse ResourceDescriptor from ResourceId (JSON) or Tags
	resource, err := p.parseResourceFromRequest(req)
	if err != nil {
		return nil, err
	}

	// Handle zero duration - return $0 with single result
	if runtimeHours == 0 {
		return &pbc.GetActualCostResponse{
			Results: []*pbc.ActualCostResult{{
				Timestamp: req.Start,
				Cost:      0,
				Source:    "aws-public-fallback",
			}},
		}, nil
	}

	// Get projected monthly cost using helper
	projectedResp, err := p.getProjectedForResource(ctx, resource)
	if err != nil {
		return nil, err
	}

	// Apply formula: actual_cost = projected_monthly_cost × (runtime_hours / 730)
	actualCost := projectedResp.CostPerMonth * (runtimeHours / hoursPerMonth)

	return &pbc.GetActualCostResponse{
		Results: []*pbc.ActualCostResult{{
			Timestamp:   req.Start,
			Cost:        actualCost,
			UsageAmount: runtimeHours,
			UsageUnit:   "hours",
			Source:      formatActualBillingDetail(projectedResp.BillingDetail, runtimeHours, actualCost),
		}},
	}, nil
}

// parseResourceFromRequest extracts a ResourceDescriptor from the request.
// It first tries to parse ResourceId as JSON, then falls back to Tags.
func (p *AWSPublicPlugin) parseResourceFromRequest(req *pbc.GetActualCostRequest) (*pbc.ResourceDescriptor, error) {
	// Try parsing ResourceId as JSON-encoded ResourceDescriptor
	if req.ResourceId != "" {
		var resource pbc.ResourceDescriptor
		if err := json.Unmarshal([]byte(req.ResourceId), &resource); err == nil {
			return &resource, nil
		}
		// If JSON parsing fails, treat ResourceId as a simple ID and use Tags
	}

	// Fall back to extracting from Tags
	tags := req.Tags
	if tags == nil {
		return nil, status.Error(codes.InvalidArgument, "missing resource information: provide ResourceId as JSON or use Tags")
	}

	// Extract resource info from tags
	resource := &pbc.ResourceDescriptor{
		Provider:     tags["provider"],
		ResourceType: tags["resource_type"],
		Sku:          tags["sku"],
		Region:       tags["region"],
		Tags:         make(map[string]string),
	}

	// Copy remaining tags (excluding the resource descriptor fields)
	for k, v := range tags {
		switch k {
		case "provider", "resource_type", "sku", "region":
			// Skip - already extracted
		default:
			resource.Tags[k] = v
		}
	}

	// Validate required fields
	if resource.Provider == "" || resource.ResourceType == "" || resource.Sku == "" || resource.Region == "" {
		return nil, status.Error(codes.InvalidArgument, "resource information incomplete: need provider, resource_type, sku, region in ResourceId or Tags")
	}

	return resource, nil
}
