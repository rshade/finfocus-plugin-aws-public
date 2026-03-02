package router

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// Plugin implements pluginsdk.Plugin as a multi-region router that delegates RPCs
// to region-specific child processes. It contains no embedded pricing data and
// acts as a thin routing layer.
type Plugin struct {
	version    string
	logger     zerolog.Logger
	registry   *ChildRegistry
	downloader *Downloader
	offline    bool
	binaryDir  string
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
		version:    version,
		logger:     logger.With().Str("component", "router").Logger(),
		registry:   registry,
		downloader: downloader,
		offline:    offline,
		binaryDir:  binaryDir,
	}
}

// Name returns the plugin name identifier.
func (r *Plugin) Name() string {
	return "finfocus-plugin-aws-public"
}

// GetPluginInfo returns metadata about the router plugin.
func (r *Plugin) GetPluginInfo(_ context.Context, _ *pbc.GetPluginInfoRequest) (*pbc.GetPluginInfoResponse, error) {
	return &pbc.GetPluginInfoResponse{
		Name:        r.Name(),
		Version:     r.version,
		SpecVersion: pluginsdk.SpecVersion,
		Providers:   []string{"aws"},
		Metadata: map[string]string{
			"type": "multi-region-router",
		},
	}, nil
}

// Supports delegates to the region-specific child process to check resource support.
func (r *Plugin) Supports(ctx context.Context, req *pbc.SupportsRequest) (*pbc.SupportsResponse, error) {
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
func (r *Plugin) GetProjectedCost(
	ctx context.Context,
	req *pbc.GetProjectedCostRequest,
) (*pbc.GetProjectedCostResponse, error) {
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
func (r *Plugin) GetActualCost(ctx context.Context, req *pbc.GetActualCostRequest) (*pbc.GetActualCostResponse, error) {
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
func (r *Plugin) EstimateCost(ctx context.Context, req *pbc.EstimateCostRequest) (*pbc.EstimateCostResponse, error) {
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
func (r *Plugin) GetPricingSpec(
	ctx context.Context,
	req *pbc.GetPricingSpecRequest,
) (*pbc.GetPricingSpecResponse, error) {
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

// GetRecommendations groups resources by region, delegates in parallel, and merges results.
func (r *Plugin) GetRecommendations(
	ctx context.Context,
	req *pbc.GetRecommendationsRequest,
) (*pbc.GetRecommendationsResponse, error) {
	traceID := r.getTraceID(ctx)

	// Group target resources by region
	regionResources := make(map[string][]*pbc.ResourceDescriptor)
	for _, res := range req.GetTargetResources() {
		region := res.GetRegion()
		if region == "" {
			continue
		}
		regionResources[region] = append(regionResources[region], res)
	}

	if len(regionResources) == 0 {
		return nil, newInvalidRegionError("GetRecommendations")
	}

	// Single region — simple delegation
	if len(regionResources) == 1 {
		for region := range regionResources {
			client, err := r.registry.GetOrLaunch(ctx, region)
			if err != nil {
				return nil, wrapRegionError(region, err)
			}
			return client.GetRecommendations(propagateTraceID(ctx, traceID), req)
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

	return merged, nil
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

// HandleDryRun delegates to the first available child for region-agnostic introspection.
func (r *Plugin) HandleDryRun(_ *pbc.DryRunRequest) (*pbc.DryRunResponse, error) {
	// DryRun is region-agnostic; delegate to any ready child
	r.registry.mu.RLock()
	var firstChild *ChildProcess
	for _, child := range r.registry.children {
		if child.State() == ChildStateReady {
			firstChild = child
			break
		}
	}
	r.registry.mu.RUnlock()

	if firstChild == nil {
		return pluginsdk.NewDryRunResponse(
			pluginsdk.WithResourceTypeSupported(false),
			pluginsdk.WithConfigurationErrors([]string{"no region children available for DryRun"}),
		), nil
	}

	// Use the child's Inner() connect client for DryRun
	client := firstChild.Client()
	if client == nil {
		return pluginsdk.NewDryRunResponse(
			pluginsdk.WithResourceTypeSupported(false),
			pluginsdk.WithConfigurationErrors([]string{"child client unavailable"}),
		), nil
	}

	// Return a basic response since DryRun is region-agnostic
	return pluginsdk.NewDryRunResponse(
		pluginsdk.WithResourceTypeSupported(true),
		pluginsdk.WithConfigurationValid(true),
	), nil
}

// ShutdownAll gracefully terminates all child processes.
func (r *Plugin) ShutdownAll(ctx context.Context) {
	r.registry.ShutdownAll(ctx)
}

// extractRegionFromResource extracts the region from a ResourceDescriptor.
func extractRegionFromResource(resource *pbc.ResourceDescriptor) string {
	if resource == nil {
		return ""
	}
	return resource.GetRegion()
}

// extractRegionFromActualCostRequest extracts the region from a GetActualCostRequest.
// It first tries Tags, then falls back to parsing ResourceId.
func extractRegionFromActualCostRequest(req *pbc.GetActualCostRequest) string {
	if tags := req.GetTags(); tags != nil {
		if region := tags["region"]; region != "" {
			return region
		}
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
	md := metadata.New(map[string]string{
		pluginsdk.TraceIDMetadataKey: traceID,
	})
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
