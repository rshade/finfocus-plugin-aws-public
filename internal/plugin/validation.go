package plugin

import (
	"context"
	"fmt"

	"github.com/rshade/pulumicost-spec/sdk/go/pluginsdk"
	pbc "github.com/rshade/pulumicost-spec/sdk/go/proto/pulumicost/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// validateProvider checks that the provider is "aws".
// Returns an error if the provider is empty or set to a non-AWS value.
//
// Design Note: Validation functions (GetProjectedCost, GetActualCost) are stricter than
// recommendation generation (GetRecommendations), which tolerates empty provider as implicit "aws".
// This is intentional: users must explicitly specify "aws" for cost estimation, but recommendations
// can be lenient since they're informational. This prevents accidental silent filtering of cost estimates.
func (p *AWSPublicPlugin) validateProvider(traceID string, provider string) error {
	if provider == "" {
		return p.newErrorWithID(traceID, codes.InvalidArgument, "provider is required", pbc.ErrorCode_ERROR_CODE_INVALID_RESOURCE)
	}
	if provider != "aws" {
		return p.newErrorWithID(traceID, codes.InvalidArgument, "only 'aws' provider is supported", pbc.ErrorCode_ERROR_CODE_INVALID_RESOURCE)
	}
	return nil
}

// ValidateProjectedCostRequest validates the request using SDK helpers and custom region checks.
// Returns the extracted resource descriptor if valid.
func (p *AWSPublicPlugin) ValidateProjectedCostRequest(ctx context.Context, req *pbc.GetProjectedCostRequest) (*pbc.ResourceDescriptor, error) {
	traceID := p.getTraceID(ctx)

	// SDK validation (checks nil request, required fields)
	if err := pluginsdk.ValidateProjectedCostRequest(req); err != nil {
		// Map SDK error to gRPC status with ErrorDetail
		return nil, p.newErrorWithID(traceID, codes.InvalidArgument, err.Error(), pbc.ErrorCode_ERROR_CODE_INVALID_RESOURCE)
	}

	resource := req.Resource

	// Comprehensive field validation (T011)
	if err := p.validateProvider(traceID, resource.Provider); err != nil {
		return nil, err
	}
	if resource.ResourceType == "" {
		return nil, p.newErrorWithID(traceID, codes.InvalidArgument, "resource_type is required", pbc.ErrorCode_ERROR_CODE_INVALID_RESOURCE)
	}

	// Custom region check
	effectiveRegion := resource.Region
	service := detectService(resource.ResourceType)

	// For global services with empty region, use the plugin's region (T012)
	if effectiveRegion == "" && (service == "s3" || service == "iam") {
		effectiveRegion = p.region
		// Note: We do not mutate the incoming request. The effective region is used
		// only for validation, not returned to the caller.
	}

	if effectiveRegion != p.region {
		return nil, p.RegionMismatchError(traceID, effectiveRegion)
	}

	return resource, nil
}

// ValidateActualCostRequest validates the request using SDK helpers and custom region checks.
// Returns the extracted resource descriptor if valid.
//
// Side Effect: For global services (S3, IAM) with empty region, this function sets the
// returned resource's Region field to the plugin's region. This allows downstream cost
// estimation to work correctly without requiring explicit region specification.
//
// Fallback chain (FR-018, FR-019):
//  1. req.Arn - Parse AWS ARN and extract region/service (SKU must come from tags)
//  2. req.ResourceId as JSON - JSON-encoded ResourceDescriptor
//  3. req.Tags - Extract provider, resource_type, sku, region from tags
func (p *AWSPublicPlugin) ValidateActualCostRequest(ctx context.Context, req *pbc.GetActualCostRequest) (*pbc.ResourceDescriptor, error) {
	traceID := p.getTraceID(ctx)

	// Basic nil check
	if req == nil {
		return nil, p.newErrorWithID(traceID, codes.InvalidArgument, "request is required", pbc.ErrorCode_ERROR_CODE_INVALID_RESOURCE)
	}

	// Validate timestamps (required regardless of resource identification method)
	if err := validateTimestamps(req); err != nil {
		return nil, p.newErrorWithID(traceID, codes.InvalidArgument, err.Error(), pbc.ErrorCode_ERROR_CODE_INVALID_RESOURCE)
	}

	// FR-018: Check ARN first (highest priority)
	if req.Arn != "" {
		resource, err := p.parseResourceFromARN(req)
		if err != nil {
			return nil, p.newErrorWithID(traceID, codes.InvalidArgument, err.Error(), pbc.ErrorCode_ERROR_CODE_INVALID_RESOURCE)
		}

		// Custom region check (ARN region vs plugin binary region)
		// Note: Global services (like S3) may have empty region in ARN
		effectiveRegion := resource.Region
		service := detectService(resource.ResourceType)
		if effectiveRegion == "" && (service == "s3" || service == "iam") {
			effectiveRegion = p.region
			// Set resource region so caller knows the effective region
			resource.Region = p.region
			p.logger.Debug().
				Str("resource_type", resource.ResourceType).
				Str("assigned_region", p.region).
				Msg("assigned plugin region to global service with empty ARN region")
		}

		if effectiveRegion != "" && effectiveRegion != p.region {
			return nil, p.RegionMismatchError(traceID, effectiveRegion)
		}

		return resource, nil
	}

	// For non-ARN requests, use SDK validation (requires ResourceId)
	if err := pluginsdk.ValidateActualCostRequest(req); err != nil {
		return nil, p.newErrorWithID(traceID, codes.InvalidArgument, err.Error(), pbc.ErrorCode_ERROR_CODE_INVALID_RESOURCE)
	}

	// FR-019: Fallback to JSON ResourceId or Tags extraction
	resource, err := p.parseResourceFromRequest(req)
	if err != nil {
		return nil, p.newErrorWithID(traceID, codes.InvalidArgument, err.Error(), pbc.ErrorCode_ERROR_CODE_INVALID_RESOURCE)
	}

	// Custom region check (consistent with ValidateProjectedCostRequest)
	effectiveRegion := resource.Region
	service := detectService(resource.ResourceType)

	// For global services with empty region, use the plugin's region
	if effectiveRegion == "" && (service == "s3" || service == "iam") {
		effectiveRegion = p.region
		// Set resource region so caller knows the effective region
		resource.Region = p.region
		p.logger.Debug().
			Str("resource_type", resource.ResourceType).
			Str("assigned_region", p.region).
			Msg("assigned plugin region to global service with empty region")
	}

	if effectiveRegion != p.region {
		return nil, p.RegionMismatchError(traceID, effectiveRegion)
	}

	return resource, nil
}

// validateTimestamps checks that start/end timestamps are present and valid.
func validateTimestamps(req *pbc.GetActualCostRequest) error {
	if req.Start == nil {
		return status.Error(codes.InvalidArgument, "start_time is required")
	}
	if req.End == nil {
		return status.Error(codes.InvalidArgument, "end_time is required")
	}
	if !req.End.AsTime().After(req.Start.AsTime()) {
		return status.Error(codes.InvalidArgument, "end_time must be after start_time")
	}
	return nil
}

// parseResourceFromARN extracts a ResourceDescriptor from the ARN + tags combination.
// ARN provides: provider, region, resource_type (via service mapping)
// Tags must provide: sku (instance type, volume type, etc.)
//
// Security Note: ARN validation is delegated to ParseARN(), which must:
//   - Validate ARN format strictly (prevent malformed ARN injection)
//   - Enforce reasonable length limits (prevent DoS via huge ARNs)
//   - Reject path traversal attempts or special sequences
// Tag values are extracted from user input and should be treated as untrusted.
func (p *AWSPublicPlugin) parseResourceFromARN(req *pbc.GetActualCostRequest) (*pbc.ResourceDescriptor, error) {
	arn, err := ParseARN(req.Arn)
	if err != nil {
		return nil, err
	}

	// Extract SKU from tags (ARN doesn't contain instance type/SKU)
	sku := ""
	if req.Tags != nil {
		sku = req.Tags["sku"]
		if sku == "" {
			sku = extractAWSSKU(req.Tags)
		}
	}
	if sku == "" {
		// Return simple error - caller wraps with newErrorWithID for trace correlation
		return nil, fmt.Errorf("ARN provided but tags missing 'sku' (instance type, volume type, etc.)")
	}

	// Map ARN service to Pulumi resource type
	resourceType := arn.ToPulumiResourceType()

	// Copy remaining tags (excluding fields we've extracted)
	tags := make(map[string]string)
	for k, v := range req.Tags {
		switch k {
		case "sku", "instanceType", "instance_class", "type", "volumeType", "volume_type":
			// Skip - already extracted for SKU
		default:
			tags[k] = v
		}
	}

	return &pbc.ResourceDescriptor{
		Provider:     "aws",
		ResourceType: resourceType,
		Sku:          sku,
		Region:       arn.Region,
		Tags:         tags,
	}, nil
}

// RegionMismatchError creates a standardized UNSUPPORTED_REGION error with details.
func (p *AWSPublicPlugin) RegionMismatchError(traceID, resourceRegion string) error {
	msg := "region mismatch"
	errDetail := &pbc.ErrorDetail{
		Code:    pbc.ErrorCode_ERROR_CODE_UNSUPPORTED_REGION,
		Message: msg,
		Details: map[string]string{
			"trace_id":        traceID,
			"plugin_region":   p.region,
			"resource_region": resourceRegion,
			"required_region": p.region,
		},
	}
	st := status.New(codes.FailedPrecondition, msg)
	stWithDetails, err := st.WithDetails(errDetail)
	if err != nil {
		// Fallback if details cannot be attached (unlikely)
		return status.Error(codes.FailedPrecondition, msg)
	}
	return stWithDetails.Err()
}