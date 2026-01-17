package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/rshade/finfocus-plugin-aws-public/internal/carbon"
	"github.com/rshade/finfocus-plugin-aws-public/internal/pricing"
	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// AWSPublicPlugin implements the pluginsdk.Plugin interface for AWS public pricing.
type AWSPublicPlugin struct {
	region           string
	version          string
	pricing          pricing.PricingClient
	carbonEstimator  carbon.CarbonEstimator
	logger           zerolog.Logger // logger is immutable (copy-on-write)
	testMode         bool           // true when FINFOCUS_TEST_MODE=true
	maxBatchSize     int            // configured max batch size for recommendations (read-only after init)
	strictValidation bool           // fail-fast on invalid resources in recommendations (read-only after init)
}

// NewAWSPublicPlugin creates and returns a configured AWSPublicPlugin for the given AWS region.
// It initializes the pricing client, a carbon estimator, and copies the provided logger.
// Test mode is determined from the FINFOCUS_TEST_MODE environment variable and, if enabled, will be logged.
//
// Parameters:
//   - region: AWS region used for pricing and lookups.
//   - version: Plugin version string (semver).
//   - pricingClient: client used to retrieve AWS pricing data.
//   - logger: logger used by the plugin for structured logs.
//
// Returns:
//
//	A pointer to an initialized AWSPublicPlugin.
func NewAWSPublicPlugin(region string, version string, pricingClient pricing.PricingClient, logger zerolog.Logger) *AWSPublicPlugin {
	testMode := IsTestMode()

	if testMode {
		logger.Info().Msg("Test mode enabled")
	}

	// Inject logger into carbon package for CSV parsing error logging (T004)
	// Issue #159: carbon.SetLogger() is called here, ensure it happens before NewEstimator()
	// and any carbon functionality is used.
	carbon.SetLogger(logger)

	// Initialize configuration
	maxBatchSize := defaultMaxBatchSize
	// Check for batch size (new variable takes precedence over deprecated)
	val, varName, found := getEnvWithDeprecation(logger, EnvMaxBatchSize, EnvMaxBatchSizeDeprecated, EnvMaxBatchSizeLegacy)

	if found {
		if n, err := strconv.Atoi(val); err == nil && n > 0 {
			if n > maxMaxBatchSize {
				logger.Warn().
					Str("variable", varName).
					Int("requested", n).
					Int("max_allowed", maxMaxBatchSize).
					Msg("requested batch size exceeds maximum, capping")
				maxBatchSize = maxMaxBatchSize
			} else {
				maxBatchSize = n
			}
		} else {
			logger.Warn().
				Str("variable", varName).
				Str("value", val).
				Msg("invalid batch size value, using default")
		}
	}

	// Check for strict validation (new variable takes precedence over deprecated)
	var strictValidation bool
	val, _, found = getEnvWithDeprecation(logger, EnvStrictValidation, EnvStrictValidationDeprecated, EnvStrictValidationLegacy)
	if found {
		strictValidation = parseBoolVal(val)
	}

	return &AWSPublicPlugin{
		region:           region,
		version:          version,
		pricing:          pricingClient,
		carbonEstimator:  carbon.NewEstimator(),
		logger:           logger,
		testMode:         testMode,
		maxBatchSize:     maxBatchSize,
		strictValidation: strictValidation,
	}
}

// parseBoolVal returns true if the string value is truthy.
// Accepted values: "true", "1", "yes", "on" (case-insensitive).
func parseBoolVal(val string) bool {
	val = strings.ToLower(strings.TrimSpace(val))
	return val == "true" || val == "1" || val == "yes" || val == "on"
}

// getEnvWithDeprecation checks for an environment variable with fallback to deprecated names.
// It logs warnings when deprecated variables are used.
// Returns (value, variableName, found).
func getEnvWithDeprecation(logger zerolog.Logger, current, deprecated, legacy string) (string, string, bool) {
	if val := os.Getenv(current); val != "" {
		return val, current, true
	}

	if val := os.Getenv(deprecated); val != "" {
		logger.Warn().
			Str("env_var", deprecated).
			Str("replacement", current).
			Str("deprecated_since", "v0.0.18").
			Str("removal_version", "v1.0.0").
			Msgf("%s is deprecated, use %s instead", deprecated, current)
		return val, deprecated, true
	}

	if val := os.Getenv(legacy); val != "" {
		logger.Warn().
			Str("env_var", legacy).
			Str("replacement", current).
			Str("deprecated_since", "v0.0.18").
			Str("removal_version", "v1.0.0").
			Msgf("%s is deprecated, use %s instead", legacy, current)
		return val, legacy, true
	}

	return "", "", false
}

// getTraceID extracts the trace_id from context or generates a UUID if not present.
// This implements the workaround for missing interceptor support in ServeConfig.
// See research.md U1 Remediation for details.
//
// Extraction order:
//  1. SDK helper (if interceptor registered)
//  2. Direct gRPC metadata lookup
//  3. UUID generation (FR-003 fallback)
//
// Returns a non-empty trace_id string suitable for log correlation.
func (p *AWSPublicPlugin) getTraceID(ctx context.Context) string {
	// First try SDK helper (works if interceptor was somehow registered)
	traceID := pluginsdk.TraceIDFromContext(ctx)
	if traceID != "" {
		return traceID
	}

	// Fallback: read directly from gRPC metadata
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if values := md.Get(pluginsdk.TraceIDMetadataKey); len(values) > 0 {
			return values[0]
		}
	}

	// Generate UUID if not present (per FR-003)
	return uuid.New().String()
}

const maxTagsToLog = 5

// sanitizeTagsForLogging returns a sanitized subset of the input tags suitable for logging.
// It returns nil if tags is nil. The result contains at most maxTagsToLog entries,
// excludes any entries whose key (case-insensitive) contains "secret", "password", or "token",
// and preserves the original key casing for included entries.
func sanitizeTagsForLogging(tags map[string]string) map[string]string {
	if tags == nil {
		return nil
	}
	// Pre-allocate with bounded capacity
	capacity := len(tags)
	if capacity > maxTagsToLog {
		capacity = maxTagsToLog
	}
	sanitized := make(map[string]string, capacity)
	count := 0
	for k, v := range tags {
		// Issue #115: Use explicit count tracking
		if count >= maxTagsToLog {
			break
		}
		kLower := strings.ToLower(k)
		// Skip known sensitive keys
		if strings.Contains(kLower, "secret") ||
			strings.Contains(kLower, "password") ||
			strings.Contains(kLower, "token") {
			continue
		}
		sanitized[k] = v
		count++
	}
	return sanitized
}

// logErrorWithID logs an error using a pre-captured trace ID.
// Use this when you've already extracted the trace ID to ensure consistency
// between error objects and log entries.
func (p *AWSPublicPlugin) logErrorWithID(traceID, operation string, err error, code pbc.ErrorCode) {
	p.traceLogger(traceID, operation).Error().
		Str(pluginsdk.FieldErrorCode, code.String()).
		Err(err).
		Msg("request failed")
}

// newErrorWithID creates a gRPC error with trace_id in the error details using a pre-captured trace ID.
// Use this when you've already extracted the trace ID to ensure consistency
// between error objects and log entries.
func (p *AWSPublicPlugin) newErrorWithID(traceID string, grpcCode codes.Code, msg string, errorCode pbc.ErrorCode) error {
	errDetail := &pbc.ErrorDetail{
		Code:    errorCode,
		Message: msg,
		Details: map[string]string{
			"trace_id": traceID,
		},
	}

	st := status.New(grpcCode, msg)
	stWithDetails, err := st.WithDetails(errDetail)
	if err != nil {
		// Log a warning if details could not be attached
		p.logger.Warn().
			Str(pluginsdk.FieldTraceID, traceID).
			Str("grpc_code", grpcCode.String()).
			Str("message", msg).
			Str("error_code", errorCode.String()).
			Err(err). // Log the error returned by WithDetails
			Msg("failed to attach error details to gRPC status")
		return st.Err() // Return original status without details
	}
	return stWithDetails.Err()
}

// traceLogger returns a logger with traceID and operation pre-filled.
// This reduces code duplication for repeated logging patterns throughout the plugin.
// Usage: p.traceLogger(traceID, "GetProjectedCost").Debug().Msg("message")
func (p *AWSPublicPlugin) traceLogger(traceID string, operation string) *zerolog.Logger {
	logger := p.logger.With().
		Str(pluginsdk.FieldTraceID, traceID).
		Str(pluginsdk.FieldOperation, operation).
		Logger()
	return &logger
}

// Name returns the plugin name identifier.
func (p *AWSPublicPlugin) Name() string {
	return "finfocus-plugin-aws-public"
}

// GetPluginInfo returns metadata about the plugin.
func (p *AWSPublicPlugin) GetPluginInfo(ctx context.Context, _ *pbc.GetPluginInfoRequest) (*pbc.GetPluginInfoResponse, error) {
	traceID := p.getTraceID(ctx)

	p.traceLogger(traceID, "GetPluginInfo").Info().
		Msg("providing plugin info")

	return &pbc.GetPluginInfoResponse{
		Name:        p.Name(),
		Version:     p.version,
		SpecVersion: pluginsdk.SpecVersion,
		Providers:   []string{"aws"},
		Metadata: map[string]string{
			"region": p.region,
			"type":   "public-pricing-fallback",
		},
	}, nil
}

// GetActualCost retrieves actual cost for a resource based on runtime.
// Uses fallback formula: actual_cost = projected_monthly_cost × (runtime_hours / 730)
//
// The proto API uses ResourceId (string) which we expect to be a JSON-encoded
// ResourceDescriptor. If ResourceId is empty, we fall back to extracting
// resource info from the Tags map.
func (p *AWSPublicPlugin) GetActualCost(ctx context.Context, req *pbc.GetActualCostRequest) (*pbc.GetActualCostResponse, error) {
	start := time.Now()
	traceID := p.getTraceID(ctx)

	// Validate request, resolve timestamps, and extract resource
	// Note: ValidateActualCostRequest now returns TimestampResolution for confidence tracking (Feature 016)
	resource, resolution, err := p.ValidateActualCostRequest(ctx, req)
	if err != nil {
		// Error already formatted with trace_id and code in ValidateActualCostRequest
		p.logErrorWithID(traceID, "GetActualCost", err, pbc.ErrorCode_ERROR_CODE_INVALID_RESOURCE)
		return nil, err
	}

	// Determine confidence level from resolution (Feature 016)
	confidence := determineConfidence(resolution)

	// Parse timestamps and calculate runtime hours
	// Note: req.Start and req.End are guaranteed non-nil by ValidateActualCostRequest
	fromTime := req.Start.AsTime()
	toTime := req.End.AsTime()

	runtimeHours, err := calculateRuntimeHours(fromTime, toTime)
	if err != nil {
		statusErr := p.newErrorWithID(traceID, codes.InvalidArgument, fmt.Sprintf("invalid time range: %v", err), pbc.ErrorCode_ERROR_CODE_INVALID_RESOURCE)
		p.logErrorWithID(traceID, "GetActualCost", statusErr, pbc.ErrorCode_ERROR_CODE_INVALID_RESOURCE)
		return nil, statusErr
	}

	// Test mode: Enhanced logging for request details (US3)
	if p.testMode {
		p.logger.Debug().
			Str(pluginsdk.FieldTraceID, traceID).
			Str("resource_type", resource.ResourceType).
			Str("sku", resource.Sku).
			Str("region", resource.Region).
			Float64("runtime_hours", runtimeHours).
			Msg("Test mode: GetActualCost request details")
	}

	// Determine service type once for FOCUS record (used in both branches)
	normalizedType := normalizeResourceType(resource.ResourceType)
	serviceType := detectService(normalizedType)

	// Handle zero duration - return $0 with single result
	if runtimeHours == 0 {
		// Build source with confidence (Feature 016)
		note := ""
		if resolution != nil && resolution.IsImported {
			note = "imported resource"
		}
		source := formatSourceWithConfidence(confidence, note)

		p.traceLogger(traceID, "GetActualCost").Info().
			Float64("cost_monthly", 0).
			Float64("usage_amount", runtimeHours).
			Str("usage_unit", "hours").
			Str("confidence", string(confidence)).
			Int64(pluginsdk.FieldDurationMs, time.Since(start).Milliseconds()).
			Msg("cost calculated")

		return &pbc.GetActualCostResponse{
			Results: []*pbc.ActualCostResult{{
				Timestamp: req.Start,
				Cost:      0,
				Source:    source,
				// FOCUS 1.2 record for FinOps reporting
				FocusRecord: buildFocusRecord(
					serviceType,
					resource.ResourceType,
					resource.Region,
					0, 0, // cost and unit price are 0 for zero duration
					getPricingUnitForService(serviceType),
					fromTime, toTime,
					resource.Sku,
				),
			}},
		}, nil
	}

	// Get projected monthly cost using helper
	projectedResp, err := p.getProjectedForResource(traceID, resource)
	if err != nil {
		// Extract error code from gRPC status to preserve context
		errCode := extractErrorCode(err)
		p.logErrorWithID(traceID, "GetActualCost", err, errCode)
		return nil, err
	}

	// Apply formula: actual_cost = projected_monthly_cost × (runtime_hours / 730)
	actualCost := projectedResp.CostPerMonth * (runtimeHours / carbon.HoursPerMonth)

	// Test mode: Enhanced logging for calculation result (US3)
	if p.testMode {
		p.logger.Debug().
			Str(pluginsdk.FieldTraceID, traceID).
			Float64("projected_monthly", projectedResp.CostPerMonth).
			Float64("runtime_hours", runtimeHours).
			Float64("actual_cost", actualCost).
			Str("formula", "projected_monthly × (runtime_hours / 730)").
			Msg("Test mode: GetActualCost calculation result")
	}

	// Build source with confidence and billing detail (Feature 016)
	note := ""
	if resolution != nil && resolution.IsImported {
		note = "imported resource"
	}
	sourceWithConfidence := formatSourceWithConfidence(confidence, note)
	billingDetail := formatActualBillingDetail(projectedResp.BillingDetail, runtimeHours, actualCost)
	// Combine: confidence prefix + billing detail
	fullSource := sourceWithConfidence + " | " + billingDetail

	p.traceLogger(traceID, "GetActualCost").Info().
		Str(pluginsdk.FieldResourceType, resource.ResourceType).
		Str("aws_service", resource.ResourceType).
		Str("aws_region", resource.Region).
		Interface("tags", sanitizeTagsForLogging(resource.Tags)).
		Float64("cost_monthly", actualCost).
		Float64("usage_amount", runtimeHours).
		Str("usage_unit", "hours").
		Str("confidence", string(confidence)).
		Str("resolution_source", resolution.Source).
		Int64(pluginsdk.FieldDurationMs, time.Since(start).Milliseconds()).
		Msg("cost calculated")

	return &pbc.GetActualCostResponse{
		Results: []*pbc.ActualCostResult{{
			Timestamp:   req.Start,
			Cost:        actualCost,
			UsageAmount: runtimeHours,
			UsageUnit:   "hours",
			Source:      fullSource,
			// FOCUS 1.2 record for FinOps reporting
			FocusRecord: buildFocusRecord(
				serviceType,
				resource.ResourceType,
				resource.Region,
				actualCost,
				projectedResp.UnitPrice, // Hourly rate from projected cost
				"Hours",
				fromTime, toTime,
				resource.Sku,
			),
		}},
	}, nil
}

// extractErrorCode retrieves the pbc.ErrorCode from a gRPC error status.
// It inspects ErrorDetail in the status details.
func extractErrorCode(err error) pbc.ErrorCode {
	if st, ok := status.FromError(err); ok {
		for _, detail := range st.Details() {
			if errDetail, ok := detail.(*pbc.ErrorDetail); ok {
				return errDetail.Code
			}
		}
	}
	return pbc.ErrorCode_ERROR_CODE_UNSPECIFIED
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
	// FR-012: Try direct "sku" tag first, then use SDK mapping for AWS-specific keys
	sku := tags["sku"]
	if sku == "" {
		sku = extractAWSSKU(tags)
	}
	resource := &pbc.ResourceDescriptor{
		Provider:     tags["provider"],
		ResourceType: tags["resource_type"],
		Sku:          sku,
		Region:       extractAWSRegion(tags), // FR-013: SDK mapping checks "region" then "availabilityZone"
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
