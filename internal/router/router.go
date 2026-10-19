package router

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/goccy/go-json"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/rshade/finfocus-plugin-aws-public/internal/pluginsettings"
	"github.com/rshade/finfocus-plugin-aws-public/internal/typeregistry"
)

// awsProvider is the only cloud provider this router can price.
const awsProvider = "aws"

// Plugin implements pluginsdk.Plugin as a multi-region router that delegates RPCs
// to region-specific child processes. It contains no embedded pricing data and
// acts as a thin routing layer.
type Plugin struct {
	version      string
	logger       zerolog.Logger
	registry     *ChildRegistry
	downloader   *Downloader
	offline      bool
	binaryDir    string
	typeRegistry *pluginsdk.TypeRegistry
}

// NewPlugin creates a new router Plugin with the given dependencies.
// It runs binary discovery and pre-populates the registry with idle children.
func NewPlugin(
	version string,
	logger zerolog.Logger,
	binaryDir string,
	offline bool,
	downloader *Downloader,
) *Plugin {
	discovered := Discover(binaryDir, logger)

	registry := NewChildRegistry(discovered, downloader, offline, logger)

	return &Plugin{
		version:      version,
		logger:       logger.With().Str("component", "router").Logger(),
		registry:     registry,
		downloader:   downloader,
		offline:      offline,
		binaryDir:    binaryDir,
		typeRegistry: typeregistry.New(),
	}
}

// Name returns the plugin name identifier.
func (r *Plugin) Name() string {
	return "finfocus-plugin-aws-public"
}

// GetPluginInfo returns the router name, version, and explicit capability
// list. Legacy supports_* metadata is left unset so finfocus-spec v0.7.3 can
// backfill it from that list.
func (r *Plugin) GetPluginInfo(_ context.Context, _ *pbc.GetPluginInfoRequest) (*pbc.GetPluginInfoResponse, error) {
	return &pbc.GetPluginInfoResponse{
		Name:        r.Name(),
		Version:     r.version,
		SpecVersion: pluginsdk.SpecVersion,
		Providers:   []string{awsProvider},
		Metadata: map[string]string{
			"type": "multi-region-router",
		},
		Capabilities: r.capabilities(),
		Settings:     pluginsettings.Router(),
	}, nil
}

// capabilities returns the capability set advertised through GetPluginInfo.
// The list is explicit and non-empty, so the SDK returns it unchanged instead
// of inferring capabilities from implemented interfaces. Inference would
// advertise BUDGETS and DISMISS_RECOMMENDATIONS, because GetBudgets and
// DismissRecommendation exist only to return Unimplemented. DRY_RUN stays in
// the list because HandleDryRun routes through Supports. The SDK backfills
// the legacy supports_* metadata keys from this list. Optional entries come
// from interface assertions so the advertised set cannot drift from the
// implementation.
func (r *Plugin) capabilities() []pbc.PluginCapability {
	capabilities := []pbc.PluginCapability{
		pbc.PluginCapability_PLUGIN_CAPABILITY_PROJECTED_COSTS,
		pbc.PluginCapability_PLUGIN_CAPABILITY_ACTUAL_COSTS,
		pbc.PluginCapability_PLUGIN_CAPABILITY_PRICING_SPEC,
		pbc.PluginCapability_PLUGIN_CAPABILITY_ESTIMATE_COST,
	}
	if _, ok := any(r).(pluginsdk.RecommendationsProvider); ok {
		capabilities = append(capabilities, pbc.PluginCapability_PLUGIN_CAPABILITY_RECOMMENDATIONS)
	}
	if _, ok := any(r).(pluginsdk.DryRunHandler); ok {
		capabilities = append(capabilities, pbc.PluginCapability_PLUGIN_CAPABILITY_DRY_RUN)
	}
	if _, ok := any(r).(pluginsdk.ResolveResourceTypesProvider); ok {
		capabilities = append(capabilities, pbc.PluginCapability_PLUGIN_CAPABILITY_RESOLVE_RESOURCE_TYPES)
	}
	return capabilities
}

// Supports checks the provider before it resolves a region or launches a child.
// A non-AWS provider is declined with supported=false. AWS resources are
// delegated to the region-specific child.
func (r *Plugin) Supports(ctx context.Context, req *pbc.SupportsRequest) (*pbc.SupportsResponse, error) {
	if provider := req.GetResource().GetProvider(); foreignProvider(provider) {
		return &pbc.SupportsResponse{
			Supported: false,
			Reason:    unsupportedProviderReason(provider),
		}, nil
	}

	region := extractRegionFromResource(req.GetResource())
	if region == "" {
		return nil, newInvalidRegionError("Supports")
	}

	traceID := r.getTraceID(ctx)
	client, err := r.registry.GetOrLaunch(ctx, region)
	if err != nil {
		r.logger.Error().
			Str("trace_id", traceID).
			Str("region", region).
			Err(err).
			Msg("failed to get child for Supports")
		return nil, wrapRegionError(region, err)
	}

	return client.Supports(propagateTraceID(ctx, traceID), req.GetResource())
}

// GetProjectedCost delegates to the region-specific child process.
// A non-AWS provider is rejected before region resolution.
func (r *Plugin) GetProjectedCost(
	ctx context.Context,
	req *pbc.GetProjectedCostRequest,
) (*pbc.GetProjectedCostResponse, error) {
	if err := rejectForeignProvider(req.GetResource().GetProvider()); err != nil {
		return nil, err
	}

	region := extractRegionFromResource(req.GetResource())
	if region == "" {
		return nil, newInvalidRegionError("GetProjectedCost")
	}

	traceID := r.getTraceID(ctx)
	client, err := r.registry.GetOrLaunch(ctx, region)
	if err != nil {
		r.logger.Error().
			Str("trace_id", traceID).
			Str("region", region).
			Err(err).
			Msg("failed to get child for GetProjectedCost")
		return nil, wrapRegionError(region, err)
	}

	return client.GetProjectedCost(propagateTraceID(ctx, traceID), req)
}

// GetActualCost delegates to the region-specific child process.
// A descriptor that names a non-AWS provider is rejected before region resolution.
func (r *Plugin) GetActualCost(ctx context.Context, req *pbc.GetActualCostRequest) (*pbc.GetActualCostResponse, error) {
	if err := rejectForeignProvider(req.GetResource().GetProvider()); err != nil {
		return nil, err
	}

	region := extractRegionFromActualCostRequest(req)
	if region == "" {
		return nil, newInvalidRegionError("GetActualCost")
	}

	traceID := r.getTraceID(ctx)
	client, err := r.registry.GetOrLaunch(ctx, region)
	if err != nil {
		r.logger.Error().
			Str("trace_id", traceID).
			Str("region", region).
			Err(err).
			Msg("failed to get child for GetActualCost")
		return nil, wrapRegionError(region, err)
	}

	return client.GetActualCost(propagateTraceID(ctx, traceID), req)
}

// EstimateCost delegates to the region-specific child process.
// A non-AWS Pulumi type returns a zero-cost response, matching the region
// binary, and does not resolve a region or launch a child.
func (r *Plugin) EstimateCost(ctx context.Context, req *pbc.EstimateCostRequest) (*pbc.EstimateCostResponse, error) {
	if foreignTypeProvider(req.GetResourceType()) {
		return &pbc.EstimateCostResponse{
			Currency:    "USD",
			CostMonthly: 0,
		}, nil
	}

	region := extractRegionFromEstimateCostRequest(req)
	if region == "" {
		return nil, newInvalidRegionError("EstimateCost")
	}

	traceID := r.getTraceID(ctx)
	client, err := r.registry.GetOrLaunch(ctx, region)
	if err != nil {
		r.logger.Error().
			Str("trace_id", traceID).
			Str("region", region).
			Err(err).
			Msg("failed to get child for EstimateCost")
		return nil, wrapRegionError(region, err)
	}

	return client.EstimateCost(propagateTraceID(ctx, traceID), req)
}

// GetPricingSpec delegates to the region-specific child process.
// A non-AWS provider is rejected before region resolution.
func (r *Plugin) GetPricingSpec(
	ctx context.Context,
	req *pbc.GetPricingSpecRequest,
) (*pbc.GetPricingSpecResponse, error) {
	if err := rejectForeignProvider(req.GetResource().GetProvider()); err != nil {
		return nil, err
	}

	region := extractRegionFromResource(req.GetResource())
	if region == "" {
		return nil, newInvalidRegionError("GetPricingSpec")
	}

	traceID := r.getTraceID(ctx)
	client, err := r.registry.GetOrLaunch(ctx, region)
	if err != nil {
		r.logger.Error().
			Str("trace_id", traceID).
			Str("region", region).
			Err(err).
			Msg("failed to get child for GetPricingSpec")
		return nil, wrapRegionError(region, err)
	}

	return client.GetPricingSpec(propagateTraceID(ctx, traceID), req)
}

// GetRecommendations groups AWS resources by region, delegates in parallel, and merges results.
// Resources that name a non-AWS provider are skipped before a child is launched.
// A request made up only of those resources returns an empty result.
func (r *Plugin) GetRecommendations(
	ctx context.Context,
	req *pbc.GetRecommendationsRequest,
) (*pbc.GetRecommendationsResponse, error) {
	traceID := r.getTraceID(ctx)

	regionResources, onlyForeign := resourcesByRegion(req.GetTargetResources())
	if len(regionResources) == 0 {
		if onlyForeign {
			return &pbc.GetRecommendationsResponse{
				Summary: pluginsdk.CalculateRecommendationSummary(nil, recommendationProjectionPeriod(req)),
			}, nil
		}
		return nil, newInvalidRegionError("GetRecommendations")
	}

	// Single region — simple delegation
	if len(regionResources) == 1 {
		for region, resources := range regionResources {
			client, err := r.registry.GetOrLaunch(ctx, region)
			if err != nil {
				return nil, wrapRegionError(region, err)
			}

			regionReq := &pbc.GetRecommendationsRequest{
				Filter:          req.GetFilter(),
				TargetResources: resources,
			}
			return client.GetRecommendations(propagateTraceID(ctx, traceID), regionReq)
		}
	}

	// Multi-region fan-out
	type regionResult struct {
		resp *pbc.GetRecommendationsResponse
		err  error
	}

	var (
		mu      sync.Mutex
		results []regionResult
		wg      sync.WaitGroup
	)

	for region, resources := range regionResources {
		wg.Add(1)
		go func(reg string, res []*pbc.ResourceDescriptor) {
			defer wg.Done()

			client, err := r.registry.GetOrLaunch(ctx, reg)
			if err != nil {
				r.logger.Warn().
					Str("trace_id", traceID).
					Str("region", reg).
					Err(err).
					Msg("region child unavailable for recommendations fan-out")
				mu.Lock()
				results = append(results, regionResult{err: err})
				mu.Unlock()
				return
			}

			// Create a region-scoped request with only this region's resources
			regionReq := &pbc.GetRecommendationsRequest{
				Filter:          req.GetFilter(),
				TargetResources: res,
			}

			resp, err := client.GetRecommendations(propagateTraceID(ctx, traceID), regionReq)
			mu.Lock()
			results = append(results, regionResult{resp: resp, err: err})
			mu.Unlock()
		}(region, resources)
	}

	wg.Wait()

	// Merge results
	merged := &pbc.GetRecommendationsResponse{}
	var anySuccess bool
	for _, result := range results {
		if result.err != nil {
			r.logger.Warn().Err(result.err).Msg("partial failure in recommendations fan-out")
			continue
		}
		if result.resp != nil {
			merged.Recommendations = append(merged.Recommendations, result.resp.GetRecommendations()...)
			anySuccess = true
		}
	}

	if !anySuccess {
		return nil, status.Error(codes.Unavailable, "all region children failed during recommendations fan-out")
	}

	merged.Summary = pluginsdk.CalculateRecommendationSummary(
		merged.GetRecommendations(), recommendationProjectionPeriod(req),
	)

	return merged, nil
}

// recommendationProjectionPeriod applies the protocol default for merged summaries.
func recommendationProjectionPeriod(req *pbc.GetRecommendationsRequest) string {
	if period := req.GetProjectionPeriod(); period != "" {
		return period
	}
	return "monthly"
}

// DismissRecommendation returns Unimplemented (stateless plugin, no recommendation state).
func (r *Plugin) DismissRecommendation(
	_ context.Context,
	_ *pbc.DismissRecommendationRequest,
) (*pbc.DismissRecommendationResponse, error) {
	return nil, status.Error(codes.Unimplemented, "DismissRecommendation is not supported by the router plugin")
}

// GetBudgets returns Unimplemented (no budget support in this plugin).
func (r *Plugin) GetBudgets(_ context.Context, _ *pbc.GetBudgetsRequest) (*pbc.GetBudgetsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "GetBudgets is not supported by the router plugin")
}

// HandleDryRun reports whether the requested resource type is supported by
// routing the descriptor through Supports, so dry-run results follow the same
// region routing and per-resource support rules as real cost requests.
//
// A descriptor missing provider, resource_type, or region is rejected with
// InvalidArgument. A non-AWS provider is declined by Supports before a child
// launches. When the region's child cannot be launched, the response marks
// the configuration invalid and carries the launch error.
func (r *Plugin) HandleDryRun(ctx context.Context, req *pbc.DryRunRequest) (*pbc.DryRunResponse, error) {
	resource := req.GetResource()
	if resource.GetProvider() == "" || resource.GetResourceType() == "" {
		return nil, status.Error(codes.InvalidArgument, "provider and resource_type are required for DryRun")
	}

	supportsResp, err := r.Supports(ctx, &pbc.SupportsRequest{Resource: resource})
	if err != nil {
		if status.Code(err) == codes.InvalidArgument {
			return nil, err
		}
		return pluginsdk.NewDryRunResponse(
			pluginsdk.WithResourceTypeSupported(false),
			pluginsdk.WithConfigurationErrors([]string{status.Convert(err).Message()}),
		), nil
	}

	return pluginsdk.NewDryRunResponse(
		pluginsdk.WithResourceTypeSupported(supportsResp.GetSupported()),
		pluginsdk.WithConfigurationValid(true),
	), nil
}

// WarmUp eagerly launches all discovered children in parallel.
// It returns immediately (fire-and-forget goroutines).
func (r *Plugin) WarmUp(ctx context.Context) {
	r.registry.WarmUp(ctx)
}

// ShutdownAll gracefully terminates all child processes.
func (r *Plugin) ShutdownAll(ctx context.Context) {
	r.registry.ShutdownAll(ctx)
}

// foreignProvider reports a provider this router must decline before it
// resolves a region or launches a child. An empty provider is not foreign:
// an AWS descriptor that omitted it still fails the region check.
func foreignProvider(provider string) bool {
	return provider != "" && provider != awsProvider
}

// foreignTypeProvider reports whether a Pulumi resource type names a provider
// other than AWS, such as "civo:compute/instance:Instance". Types without a
// provider prefix stay on the AWS path.
func foreignTypeProvider(resourceType string) bool {
	provider, _, ok := strings.Cut(resourceType, ":")
	return ok && foreignProvider(provider)
}

// unsupportedProviderReason is the decline text shared with the region binary.
func unsupportedProviderReason(provider string) string {
	return fmt.Sprintf("Provider %q not supported (only %q is supported)", provider, awsProvider)
}

// rejectForeignProvider returns InvalidArgument when provider is a foreign cloud.
func rejectForeignProvider(provider string) error {
	if !foreignProvider(provider) {
		return nil
	}
	message := unsupportedProviderReason(provider)
	errDetail := &pbc.ErrorDetail{
		Code:    pbc.ErrorCode_ERROR_CODE_INVALID_RESOURCE,
		Message: message,
	}
	st := status.New(codes.InvalidArgument, message)
	stWithDetails, err := st.WithDetails(errDetail)
	if err != nil {
		return st.Err()
	}
	return stWithDetails.Err()
}

// resourcesByRegion groups resources that this router can price by region.
// Foreign providers are omitted. onlyForeign is true when the request named
// resources and every one of them was a foreign provider.
func resourcesByRegion(resources []*pbc.ResourceDescriptor) (map[string][]*pbc.ResourceDescriptor, bool) {
	grouped := make(map[string][]*pbc.ResourceDescriptor)
	sawRoutable := false
	for _, res := range resources {
		if foreignProvider(res.GetProvider()) {
			continue
		}
		sawRoutable = true
		if region := res.GetRegion(); region != "" {
			grouped[region] = append(grouped[region], res)
		}
	}
	return grouped, len(resources) > 0 && !sawRoutable
}

// extractRegionFromResource extracts the region from a ResourceDescriptor.
func extractRegionFromResource(resource *pbc.ResourceDescriptor) string {
	if resource == nil {
		return ""
	}
	return resource.GetRegion()
}

// extractRegionFromActualCostRequest extracts the region from a GetActualCostRequest.
// A set resource descriptor supplies the region, including when it is empty.
// Otherwise it tries Tags, then ResourceId.
func extractRegionFromActualCostRequest(req *pbc.GetActualCostRequest) string {
	if req == nil {
		return ""
	}
	if req.GetResource() != nil {
		return req.GetResource().GetRegion()
	}
	if tags := req.GetTags(); tags != nil {
		if region := tags["region"]; region != "" {
			return region
		}
	}
	if req.GetResourceId() == "" {
		return ""
	}

	resourceID := req.GetResourceId()

	var resource pbc.ResourceDescriptor
	if err := json.Unmarshal([]byte(resourceID), &resource); err == nil {
		if region := resource.GetRegion(); region != "" {
			return region
		}
		if tags := resource.GetTags(); tags != nil {
			if region := tags["region"]; region != "" {
				return region
			}
		}
	}

	// ARN format: arn:partition:service:region:account:resource
	if parsed, err := arn.Parse(resourceID); err == nil && parsed.Region != "" {
		return parsed.Region
	}

	return ""
}

// extractRegionFromEstimateCostRequest extracts the region from an EstimateCostRequest.
// It parses the Attributes protobuf Struct to find the "region" field.
func extractRegionFromEstimateCostRequest(req *pbc.EstimateCostRequest) string {
	attrs := req.GetAttributes()
	if attrs == nil {
		return ""
	}
	fields := attrs.GetFields()
	if fields == nil {
		return ""
	}
	if regionVal, ok := fields["region"]; ok {
		return regionVal.GetStringValue()
	}
	return ""
}

// getTraceID extracts the trace_id from context or generates a UUID.
func (r *Plugin) getTraceID(ctx context.Context) string {
	traceID := pluginsdk.TraceIDFromContext(ctx)
	if traceID != "" {
		return traceID
	}

	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if values := md.Get(pluginsdk.TraceIDMetadataKey); len(values) > 0 {
			return values[0]
		}
	}

	return uuid.New().String()
}

// propagateTraceID creates a new context with the trace_id set as outgoing gRPC metadata.
func propagateTraceID(ctx context.Context, traceID string) context.Context {
	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok || md == nil {
		md = metadata.MD{}
	} else {
		md = md.Copy()
	}
	md.Set(pluginsdk.TraceIDMetadataKey, traceID)
	return metadata.NewOutgoingContext(ctx, md)
}

// newInvalidRegionError creates a gRPC InvalidArgument error for missing region.
func newInvalidRegionError(operation string) error {
	errDetail := &pbc.ErrorDetail{
		Code:    pbc.ErrorCode_ERROR_CODE_INVALID_RESOURCE,
		Message: fmt.Sprintf("region is required for %s", operation),
	}
	st := status.New(codes.InvalidArgument, fmt.Sprintf("region is required for %s", operation))
	stWithDetails, err := st.WithDetails(errDetail)
	if err != nil {
		return st.Err()
	}
	return stWithDetails.Err()
}

// wrapRegionError wraps a region-related error as a gRPC Unavailable error.
func wrapRegionError(region string, err error) error {
	errDetail := &pbc.ErrorDetail{
		Code:    pbc.ErrorCode_ERROR_CODE_UNSUPPORTED_REGION,
		Message: err.Error(),
		Details: map[string]string{
			"requiredRegion": region,
		},
	}
	st := status.New(codes.Unavailable, fmt.Sprintf("region %s: %s", region, err.Error()))
	stWithDetails, detailErr := st.WithDetails(errDetail)
	if detailErr != nil {
		return st.Err()
	}
	return stWithDetails.Err()
}
