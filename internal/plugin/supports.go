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
	case "ec2", "ebs", "rds", "eks", "s3", "lambda":
		// Fully supported
		p.logger.Info().
			Str(pluginsdk.FieldTraceID, traceID).
			Str(pluginsdk.FieldOperation, "Supports").
			Str(pluginsdk.FieldResourceType, resource.ResourceType).
			Str("aws_region", resource.Region).
			Bool("supported", true).
			Int64(pluginsdk.FieldDurationMs, time.Since(start).Milliseconds()).
			Msg("resource support check")

		return &pbc.SupportsResponse{
			Supported: true,
			Reason:    "",
		}, nil

	case "dynamodb":
		// Stub support - returns $0 estimates
		p.logger.Info().
			Str(pluginsdk.FieldTraceID, traceID).
			Str(pluginsdk.FieldOperation, "Supports").
			Str(pluginsdk.FieldResourceType, resource.ResourceType).
			Str("aws_region", resource.Region).
			Bool("supported", true).
			Int64(pluginsdk.FieldDurationMs, time.Since(start).Milliseconds()).
			Msg("resource support check")

		return &pbc.SupportsResponse{
			Supported: true,
			Reason:    fmt.Sprintf("Limited support - %s cost estimation not fully implemented, returns $0 estimate", resource.ResourceType),
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
			Supported: false,
			Reason:    fmt.Sprintf("Resource type %q not supported", resource.ResourceType),
		}, nil
	}
}
