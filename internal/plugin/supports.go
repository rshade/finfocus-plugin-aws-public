package plugin

import (
	"context"
	"fmt"
	"time"

	"github.com/rshade/pulumicost-spec/sdk/go/pluginsdk"
	pbc "github.com/rshade/pulumicost-spec/sdk/go/proto/pulumicost/v1"
)

// Supports checks if this plugin can estimate costs for the given resource.
func (p *AWSPublicPlugin) Supports(ctx context.Context, req *pbc.SupportsRequest) (*pbc.SupportsResponse, error) {
	start := time.Now()
	traceID := p.getTraceID(ctx)

	if req == nil || req.Resource == nil {
		p.logger.Info().
			Str(pluginsdk.FieldTraceID, traceID).
			Str(pluginsdk.FieldOperation, "Supports").
			Str(pluginsdk.FieldErrorCode, pbc.ErrorCode_ERROR_CODE_INVALID_RESOURCE.String()).
			Int64(pluginsdk.FieldDurationMs, time.Since(start).Milliseconds()).
			Msg("resource support check")

		return &pbc.SupportsResponse{
			Supported: false,
			Reason:    "Invalid request: missing resource descriptor",
		}, nil
	}

	resource := req.Resource

	// Check provider
	if resource.Provider != "aws" {
		p.logger.Info().
			Str(pluginsdk.FieldTraceID, traceID).
			Str(pluginsdk.FieldOperation, "Supports").
			Str(pluginsdk.FieldResourceType, resource.ResourceType).
			Str("aws_region", resource.Region).
			Bool("supported", false).
			Int64(pluginsdk.FieldDurationMs, time.Since(start).Milliseconds()).
			Msg("resource support check")

		return &pbc.SupportsResponse{
			Supported: false,
			Reason:    fmt.Sprintf("Provider %q not supported (only 'aws' is supported)", resource.Provider),
		}, nil
	}

	// Check region match
	if resource.Region != p.region {
		p.logger.Info().
			Str(pluginsdk.FieldTraceID, traceID).
			Str(pluginsdk.FieldOperation, "Supports").
			Str(pluginsdk.FieldResourceType, resource.ResourceType).
			Str("aws_region", resource.Region).
			Bool("supported", false).
			Int64(pluginsdk.FieldDurationMs, time.Since(start).Milliseconds()).
			Msg("resource support check")

		return &pbc.SupportsResponse{
			Supported: false,
			Reason:    fmt.Sprintf("Region not supported by this binary (plugin region: %s, resource region: %s)", p.region, resource.Region),
		}, nil
	}

	// Normalize resource type (handles Pulumi formats like aws:eks/cluster:Cluster)
	normalizedType := detectService(resource.ResourceType)

	// Check resource type
	switch normalizedType {
	case "ec2":
		// EC2 fully supported with carbon estimation
		supportedMetrics := getSupportedMetrics(normalizedType)
		p.logger.Info().
			Str(pluginsdk.FieldTraceID, traceID).
			Str(pluginsdk.FieldOperation, "Supports").
			Str(pluginsdk.FieldResourceType, resource.ResourceType).
			Str("aws_region", resource.Region).
			Bool("supported", true).
			Int("supported_metrics_count", len(supportedMetrics)).
			Int64(pluginsdk.FieldDurationMs, time.Since(start).Milliseconds()).
			Msg("resource support check")

		return &pbc.SupportsResponse{
			Supported:        true,
			Reason:           "",
			SupportedMetrics: supportedMetrics,
		}, nil

	case "ebs", "rds", "eks", "s3", "lambda":
		// Supported but no carbon estimation yet
		p.logger.Info().
			Str(pluginsdk.FieldTraceID, traceID).
			Str(pluginsdk.FieldOperation, "Supports").
			Str(pluginsdk.FieldResourceType, resource.ResourceType).
			Str("aws_region", resource.Region).
			Bool("supported", true).
			Int64(pluginsdk.FieldDurationMs, time.Since(start).Milliseconds()).
			Msg("resource support check")

		return &pbc.SupportsResponse{
			Supported:        true,
			Reason:           "",
			SupportedMetrics: nil, // No additional metrics for these types yet
		}, nil

	case "dynamodb":
		// Stub support - returns $0 estimates, no carbon
		p.logger.Info().
			Str(pluginsdk.FieldTraceID, traceID).
			Str(pluginsdk.FieldOperation, "Supports").
			Str(pluginsdk.FieldResourceType, resource.ResourceType).
			Str("aws_region", resource.Region).
			Bool("supported", true).
			Int64(pluginsdk.FieldDurationMs, time.Since(start).Milliseconds()).
			Msg("resource support check")

		return &pbc.SupportsResponse{
			Supported:        true,
			Reason:           fmt.Sprintf("Limited support - %s cost estimation not fully implemented, returns $0 estimate", resource.ResourceType),
			SupportedMetrics: nil, // No additional metrics for stub types
		}, nil

	default:
		// Unknown resource type
		p.logger.Info().
			Str(pluginsdk.FieldTraceID, traceID).
			Str(pluginsdk.FieldOperation, "Supports").
			Str(pluginsdk.FieldResourceType, resource.ResourceType).
			Str("aws_region", resource.Region).
			Bool("supported", false).
			Int64(pluginsdk.FieldDurationMs, time.Since(start).Milliseconds()).
			Msg("resource support check")

		return &pbc.SupportsResponse{
			Supported:        false,
			Reason:           fmt.Sprintf("Resource type %q not supported", resource.ResourceType),
			SupportedMetrics: nil,
		}, nil
	}
}

// getSupportedMetrics returns the list of supported metric kinds for a given resource type.
// Currently, only EC2 supports carbon footprint estimation.
// This function enables PulumiCost core to discover which plugins provide which metrics.
func getSupportedMetrics(resourceType string) []pbc.MetricKind {
	switch resourceType {
	case "ec2":
		// EC2 supports carbon footprint estimation via CCF data
		return []pbc.MetricKind{pbc.MetricKind_METRIC_KIND_CARBON_FOOTPRINT}
	default:
		// Other resource types don't have additional metrics yet
		return nil
	}
}
