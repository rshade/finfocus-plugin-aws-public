package plugin

import (
	"context"
	"fmt"
	"maps"

	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
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
		return p.newErrorWithID(
			traceID,
			codes.InvalidArgument,
			"provider is required",
			pbc.ErrorCode_ERROR_CODE_INVALID_RESOURCE,
		)
	}
	if provider != providerAWS {
		return p.newErrorWithID(
			traceID,
			codes.InvalidArgument,
			fmt.Sprintf("only %q provider is supported", providerAWS),
			pbc.ErrorCode_ERROR_CODE_INVALID_RESOURCE,
		)
	}
	return nil
}

// allowsEmptySKU returns true for services that resolve their primary identifier from tags
// rather than requiring it in the SKU field. These services bypass SDK SKU validation.
func allowsEmptySKU(service string) bool {
	return service == serviceASG
}

// ValidateProjectedCostRequest validates the request using SDK helpers and custom region checks.
// Returns the extracted resource descriptor if valid.
func (p *AWSPublicPlugin) ValidateProjectedCostRequest(
	ctx context.Context,
	req *pbc.GetProjectedCostRequest,
) (*pbc.ResourceDescriptor, error) {
	traceID := p.getTraceID(ctx)

	// Basic nil checks before any validation
	if req == nil || req.GetResource() == nil {
		return nil, p.newErrorWithID(
			traceID,
			codes.InvalidArgument,
			"request and resource are required",
			pbc.ErrorCode_ERROR_CODE_INVALID_RESOURCE,
		)
	}

	resource := req.GetResource()

	// Check if this is a zero-cost resource or tag-resolved service BEFORE SDK validation.
	// Zero-cost resources (VPC, Security Groups, Subnets) don't require a SKU
	// since they always return $0 cost estimates.
	// Tag-resolved services (ASG) resolve their primary identifier from tags.
	normalizedRT := normalizeResourceType(resource.GetResourceType())
	detectedSvc := detectService(normalizedRT)
	if IsZeroCostService(detectedSvc) || allowsEmptySKU(detectedSvc) {
		// Validate provider and region manually (skip SDK's SKU requirement)
		if err := p.validateProvider(traceID, resource.GetProvider()); err != nil {
			return nil, err
		}

		// Region check: tag-resolved services must provide an explicit region
		// to prevent silently defaulting to the plugin's region.
		effectiveRegion := resource.GetRegion()
		if effectiveRegion == "" && allowsEmptySKU(detectedSvc) {
			return nil, p.newErrorWithID(
				traceID,
				codes.InvalidArgument,
				"region is required for tag-resolved services",
				pbc.ErrorCode_ERROR_CODE_INVALID_RESOURCE,
			)
		}
		if effectiveRegion == "" {
			effectiveRegion = p.region
		}
		if effectiveRegion != p.region {
			return nil, p.RegionMismatchError(traceID, effectiveRegion)
		}

		return resource, nil
	}

	// SDK validation (checks nil request, required fields including SKU)
	if err := pluginsdk.ValidateProjectedCostRequest(req); err != nil {
		// Map SDK error to gRPC status with ErrorDetail
		return nil, p.newErrorWithID(
			traceID,
			codes.InvalidArgument,
			err.Error(),
			pbc.ErrorCode_ERROR_CODE_INVALID_RESOURCE,
		)
	}

	// Comprehensive field validation (T011)
	if err := p.validateProvider(traceID, resource.GetProvider()); err != nil {
		return nil, err
	}
	if resource.GetResourceType() == "" {
		return nil, p.newErrorWithID(
			traceID,
			codes.InvalidArgument,
			"resource_type is required",
			pbc.ErrorCode_ERROR_CODE_INVALID_RESOURCE,
		)
	}

	// Custom region check
	effectiveRegion := resource.GetRegion()
	normalizedResourceType := normalizeResourceType(resource.GetResourceType())
	service := detectService(normalizedResourceType)

	// For global services with empty region, use the plugin's region (T012)
	if effectiveRegion == "" && (service == serviceS3 || service == serviceIAM) {
		effectiveRegion = p.region
		// Note: We do not mutate the incoming request. The effective region is used
		// only for validation, not returned to the caller.
	}

	if effectiveRegion != p.region {
		return nil, p.RegionMismatchError(traceID, effectiveRegion)
	}

	return resource, nil
}

// validateProjectedCostRequestWithResolver validates the request using a pre-computed serviceResolver.
// This variant avoids redundant normalizeResourceType() and detectService() calls when the caller
// has already created a resolver for other purposes (e.g., cost routing).
//
// The resolver parameter must be created from resource.ResourceType before calling this function.
// This is an optimization for SC-002: single detectService call per request.
func (p *AWSPublicPlugin) validateProjectedCostRequestWithResolver(
	ctx context.Context,
	req *pbc.GetProjectedCostRequest,
	resolver *serviceResolver,
) (*pbc.ResourceDescriptor, error) {
	traceID := p.getTraceID(ctx)

	// Basic nil checks before any validation
	// Note: The caller should have already validated req and req.Resource before creating the resolver,
	// but we keep this check for safety.
	if req == nil || req.GetResource() == nil {
		return nil, p.newErrorWithID(
			traceID,
			codes.InvalidArgument,
			"request and resource are required",
			pbc.ErrorCode_ERROR_CODE_INVALID_RESOURCE,
		)
	}

	resource := req.GetResource()

	// Check if this is a zero-cost resource or tag-resolved service BEFORE SDK validation.
	// Use resolver to avoid redundant detectService() calls.
	svcType := resolver.ServiceType()
	if IsZeroCostService(svcType) || allowsEmptySKU(svcType) {
		// Validate provider and region manually (skip SDK's SKU requirement)
		if err := p.validateProvider(traceID, resource.GetProvider()); err != nil {
			return nil, err
		}

		// Region check: tag-resolved services must provide an explicit region
		// to prevent silently defaulting to the plugin's region.
		effectiveRegion := resource.GetRegion()
		if effectiveRegion == "" && allowsEmptySKU(svcType) {
			return nil, p.newErrorWithID(
				traceID,
				codes.InvalidArgument,
				"region is required for tag-resolved services",
				pbc.ErrorCode_ERROR_CODE_INVALID_RESOURCE,
			)
		}
		if effectiveRegion == "" {
			effectiveRegion = p.region
		}
		if effectiveRegion != p.region {
			return nil, p.RegionMismatchError(traceID, effectiveRegion)
		}

		return resource, nil
	}

	// SDK validation (checks nil request, required fields including SKU)
	if err := pluginsdk.ValidateProjectedCostRequest(req); err != nil {
		return nil, p.newErrorWithID(
			traceID,
			codes.InvalidArgument,
			err.Error(),
			pbc.ErrorCode_ERROR_CODE_INVALID_RESOURCE,
		)
	}

	// Comprehensive field validation
	if err := p.validateProvider(traceID, resource.GetProvider()); err != nil {
		return nil, err
	}
	if resource.GetResourceType() == "" {
		return nil, p.newErrorWithID(
			traceID,
			codes.InvalidArgument,
			"resource_type is required",
			pbc.ErrorCode_ERROR_CODE_INVALID_RESOURCE,
		)
	}

	// Custom region check using cached service type from resolver
	effectiveRegion := resource.GetRegion()
	service := resolver.ServiceType()

	// For global services with empty region, use the plugin's region
	if effectiveRegion == "" && (service == serviceS3 || service == serviceIAM) {
		effectiveRegion = p.region
	}

	if effectiveRegion != p.region {
		return nil, p.RegionMismatchError(traceID, effectiveRegion)
	}

	return resource, nil
}

// ValidateActualCostRequest validates the request using SDK helpers and custom region checks.
// Returns the extracted resource descriptor and timestamp resolution if valid.
//
// Timestamp Resolution (Feature 016):
// This function first resolves timestamps from explicit request fields OR pulumi:created tag,
// then populates req.Start/End before validation. This enables automatic runtime detection
// from Pulumi state metadata while maintaining backward compatibility.
//
// Side Effect: For global services (S3, IAM) with empty region, this function sets the
// returned resource's Region field to the plugin's region. This allows downstream cost
// estimation to work correctly without requiring explicit region specification.
//
// Side Effect: req.Start and req.End may be populated from resolution if originally nil.
//
// Fallback chain (FR-018, FR-019):
//  1. req.Arn - Parse AWS ARN and extract region/service (SKU must come from tags)
//  2. req.ResourceId as JSON - JSON-encoded ResourceDescriptor
//  3. req.Tags - Extract provider, resource_type, sku, region from tags
func (p *AWSPublicPlugin) ValidateActualCostRequest( //nolint:gocognit,funlen
	ctx context.Context,
	req *pbc.GetActualCostRequest,
) (*pbc.ResourceDescriptor, *TimestampResolution, error) {
	traceID := p.getTraceID(ctx)

	// Basic nil check
	if req == nil {
		return nil, nil, p.newErrorWithID(
			traceID,
			codes.InvalidArgument,
			"request is required",
			pbc.ErrorCode_ERROR_CODE_INVALID_RESOURCE,
		)
	}

	// Resolve timestamps BEFORE validation (Feature 016)
	// This populates req.Start/End from tags if not explicitly provided
	resolution, err := resolveTimestamps(req)
	if err != nil {
		return nil, nil, p.newErrorWithID(
			traceID,
			codes.InvalidArgument,
			err.Error(),
			pbc.ErrorCode_ERROR_CODE_INVALID_RESOURCE,
		)
	}

	// Populate request timestamps from resolution for downstream validation
	// Note: This mutates the request but is safe since we're within validation
	if req.GetStart() == nil {
		req.Start = timestamppb.New(resolution.Start)
	}
	if req.GetEnd() == nil {
		req.End = timestamppb.New(resolution.End)
	}

	// Log timestamp resolution source for debugging
	p.logger.Debug().
		Str(pluginsdk.FieldTraceID, traceID).
		Str("resolution_source", resolution.Source).
		Bool("is_imported", resolution.IsImported).
		Time("resolved_start", resolution.Start).
		Time("resolved_end", resolution.End).
		Msg("timestamps resolved")

	// Validate timestamps (now guaranteed non-nil after resolution)
	if tsErr := validateTimestamps(req); tsErr != nil {
		return nil, nil, p.newErrorWithID(
			traceID,
			codes.InvalidArgument,
			tsErr.Error(),
			pbc.ErrorCode_ERROR_CODE_INVALID_RESOURCE,
		)
	}

	// FR-018: Check ARN first (highest priority)
	if req.GetArn() != "" {
		resource, arnErr := p.parseResourceFromARN(req)
		if arnErr != nil {
			msg := fmt.Sprintf("failed to parse ARN %q: %v", req.GetArn(), arnErr)
			return nil, nil, p.newErrorWithID(
				traceID,
				codes.InvalidArgument,
				msg,
				pbc.ErrorCode_ERROR_CODE_INVALID_RESOURCE,
			)
		}

		// Custom region check (ARN region vs plugin binary region)
		// Note: Global services (like S3) may have empty region in ARN
		effectiveRegion := resource.GetRegion()
		normalizedResourceType := normalizeResourceType(resource.GetResourceType())
		service := detectService(normalizedResourceType)
		if effectiveRegion == "" && (service == serviceS3 || service == serviceIAM) {
			effectiveRegion = p.region
			// Set resource region so caller knows the effective region
			resource.Region = p.region
			p.logger.Debug().
				Str("resource_type", resource.GetResourceType()).
				Str("assigned_region", p.region).
				Msg("assigned plugin region to global service with empty ARN region")
		}

		if effectiveRegion != "" && effectiveRegion != p.region {
			return nil, nil, p.RegionMismatchError(traceID, effectiveRegion)
		}

		return resource, resolution, nil
	}

	// For non-ARN requests, use SDK validation (requires ResourceId)
	if sdkErr := pluginsdk.ValidateActualCostRequest(req); sdkErr != nil {
		return nil, nil, p.newErrorWithID(
			traceID,
			codes.InvalidArgument,
			sdkErr.Error(),
			pbc.ErrorCode_ERROR_CODE_INVALID_RESOURCE,
		)
	}

	// FR-019: Fallback to JSON ResourceId or Tags extraction
	resource, parseErr := p.parseResourceFromRequest(req)
	if parseErr != nil {
		return nil, nil, p.newErrorWithID(
			traceID,
			codes.InvalidArgument,
			parseErr.Error(),
			pbc.ErrorCode_ERROR_CODE_INVALID_RESOURCE,
		)
	}

	// Custom region check (consistent with ValidateProjectedCostRequest)
	effectiveRegion := resource.GetRegion()
	normalizedResourceType := normalizeResourceType(resource.GetResourceType())
	service := detectService(normalizedResourceType)

	// For global services with empty region, use the plugin's region
	if effectiveRegion == "" && (service == serviceS3 || service == serviceIAM) {
		effectiveRegion = p.region
		// Set resource region so caller knows the effective region
		resource.Region = p.region
		p.logger.Debug().
			Str("resource_type", resource.GetResourceType()).
			Str("assigned_region", p.region).
			Msg("assigned plugin region to global service with empty region")
	}

	if effectiveRegion != p.region {
		return nil, nil, p.RegionMismatchError(traceID, effectiveRegion)
	}

	return resource, resolution, nil
}

// validateTimestamps checks that start/end timestamps are present and valid.
func validateTimestamps(req *pbc.GetActualCostRequest) error {
	if req.GetStart() == nil {
		return status.Error(codes.InvalidArgument, "start_time is required")
	}
	if req.GetEnd() == nil {
		return status.Error(codes.InvalidArgument, "end_time is required")
	}
	if !req.GetEnd().AsTime().After(req.GetStart().AsTime()) {
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
//
// Tag values are extracted from user input and should be treated as untrusted.
func (p *AWSPublicPlugin) parseResourceFromARN(req *pbc.GetActualCostRequest) (*pbc.ResourceDescriptor, error) {
	arn, err := ParseARN(req.GetArn())
	if err != nil {
		return nil, err
	}

	// Map ARN service to Pulumi resource type
	resourceType := arn.ToPulumiResourceType()
	canonicalService := detectService(normalizeResourceType(resourceType))

	// Zero-cost resources (VPC, Subnet, SecurityGroup, IAM, LaunchTemplate, etc.)
	// don't need a SKU - skip extraction and return immediately with empty SKU.
	if IsZeroCostService(canonicalService) {
		tags := make(map[string]string, len(req.GetTags()))
		maps.Copy(tags, req.GetTags())
		return &pbc.ResourceDescriptor{
			Provider:     providerAWS,
			ResourceType: canonicalService,
			Sku:          "",
			Region:       arn.Region,
			Tags:         tags,
		}, nil
	}

	// Tag-resolved services (ASG) resolve their SKU from tags during estimation,
	// not during validation. Pass all tags through with empty SKU so the estimator
	// (e.g., ExtractASGAttributes) can resolve the instance type from tags.
	if allowsEmptySKU(canonicalService) {
		tags := make(map[string]string, len(req.GetTags()))
		maps.Copy(tags, req.GetTags())
		return &pbc.ResourceDescriptor{
			Provider:     providerAWS,
			ResourceType: canonicalService,
			Sku:          "",
			Region:       arn.Region,
			Tags:         tags,
		}, nil
	}

	// Extract SKU from tags (ARN doesn't contain instance type/SKU)
	sku := ""
	if tags := req.GetTags(); tags != nil {
		sku = tags["sku"]
		if sku == "" {
			sku = extractAWSSKU(tags)
		}
	}
	if sku == "" {
		// Return simple error - caller wraps with newErrorWithID for trace correlation
		return nil, fmt.Errorf(
			"ARN provided (%s) but tags missing 'sku' (instance type, volume type, etc.)",
			req.GetArn(),
		)
	}

	// Copy remaining tags (excluding fields we've extracted)
	tags := make(map[string]string)
	for k, v := range req.GetTags() {
		switch k {
		case "sku", "instanceType", "instance_class", "type", "volumeType", "volume_type":
			// Skip - already extracted for SKU
		default:
			tags[k] = v
		}
	}

	return &pbc.ResourceDescriptor{
		Provider:     providerAWS,
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
