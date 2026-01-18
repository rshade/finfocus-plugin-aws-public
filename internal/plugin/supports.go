package plugin

import (
	"context"
	"fmt"
	"time"

	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

// Supports checks if this plugin can estimate costs for the given resource.
func (p *AWSPublicPlugin) Supports(ctx context.Context, req *pbc.SupportsRequest) (*pbc.SupportsResponse, error) {
	start := time.Now()
	traceID := p.getTraceID(ctx)

	if req == nil || req.Resource == nil {
		p.traceLogger(traceID, "Supports").Info().
			Str(pluginsdk.FieldErrorCode, pbc.ErrorCode_ERROR_CODE_INVALID_RESOURCE.String()).
			Int64(pluginsdk.FieldDurationMs, time.Since(start).Milliseconds()).
			Msg("resource support check")

		return &pbc.SupportsResponse{
			Supported: false,
			Reason:    "Invalid request: missing resource descriptor",
		}, nil
	}

	resource := req.Resource

	// Normalize resource type (handles Pulumi formats like aws:eks/cluster:Cluster)
	// Note: detectService() is called multiple times across validation and support checks.
	// For optimization opportunity: consider caching normalized service types per resource_type
	// to avoid repeated string parsing if high-frequency batches of identical resource types occur.
	normalizedResourceType := normalizeResourceType(resource.ResourceType)
	normalizedType := detectService(normalizedResourceType)

	// Check provider
	if resource.Provider != providerAWS {
		p.traceLogger(traceID, "Supports").Info().
			Str(pluginsdk.FieldResourceType, resource.ResourceType).
			Str("aws_region", resource.Region).
			Bool("supported", false).
			Int64(pluginsdk.FieldDurationMs, time.Since(start).Milliseconds()).
			Msg("resource support check")

		return &pbc.SupportsResponse{
			Supported: false,
			Reason:    fmt.Sprintf("Provider %q not supported (only %q is supported)", resource.Provider, providerAWS),
		}, nil
	}

	// Check region match
	// For global services (S3, IAM) and zero-cost resources (VPC, SecurityGroup, Subnet),
	// allow empty region and default to plugin region.
	effectiveRegion := resource.Region
	if effectiveRegion == "" && (normalizedType == "s3" || normalizedType == "iam" || IsZeroCostService(normalizedType)) {
		effectiveRegion = p.region
	}

	if effectiveRegion != p.region {
		p.traceLogger(traceID, "Supports").Info().
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

	// Check resource type
	switch normalizedType {
	case "ec2", "rds", "lambda", "s3", "ebs", "eks", "dynamodb", "elasticache":
		// These services support cost estimation
		// EC2 also supports carbon footprint estimation
		supportedMetrics := getSupportedMetrics(normalizedType)
		p.traceLogger(traceID, "Supports").Info().
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

	case "elb", "natgw", "cloudwatch":
		// Supported but no carbon estimation yet
		p.traceLogger(traceID, "Supports").Info().
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

	default:
		// Check for zero-cost resources using centralized ZeroCostServices map
		if IsZeroCostService(normalizedType) {
			p.traceLogger(traceID, "Supports").Info().
				Str(pluginsdk.FieldResourceType, resource.ResourceType).
				Str("aws_region", effectiveRegion).
				Bool("supported", true).
				Str("cost_type", "zero-cost").
				Int64(pluginsdk.FieldDurationMs, time.Since(start).Milliseconds()).
				Msg("resource support check")

			return &pbc.SupportsResponse{
				Supported:        true,
				Reason:           "",
				SupportedMetrics: nil, // No metrics for zero-cost resources
			}, nil
		}

		// Unknown resource type
		p.traceLogger(traceID, "Supports").Info().
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
// Services with carbon footprint estimation return METRIC_KIND_CARBON_FOOTPRINT.
// resourceType is the normalized resource type (e.g., "ec2", "rds", "lambda", "s3", "ebs", "eks", "dynamodb", "elasticache").
func getSupportedMetrics(resourceType string) []pbc.MetricKind {
	switch resourceType {
	case "ec2":
		// EC2 instances: CPU/GPU power × utilization × grid factor + optional embodied carbon
		return []pbc.MetricKind{pbc.MetricKind_METRIC_KIND_CARBON_FOOTPRINT}
	case "ebs":
		// EBS volumes: Storage energy × replication factor × grid factor
		return []pbc.MetricKind{pbc.MetricKind_METRIC_KIND_CARBON_FOOTPRINT}
	case "s3":
		// S3 storage: Storage energy × replication factor × grid factor (by storage class)
		return []pbc.MetricKind{pbc.MetricKind_METRIC_KIND_CARBON_FOOTPRINT}
	case "lambda":
		// Lambda functions: vCPU-equivalent × duration × grid factor (ARM64 efficiency adjusted)
		return []pbc.MetricKind{pbc.MetricKind_METRIC_KIND_CARBON_FOOTPRINT}
	case "rds":
		// RDS instances: Compute carbon + storage carbon (Multi-AZ 2× multiplier)
		return []pbc.MetricKind{pbc.MetricKind_METRIC_KIND_CARBON_FOOTPRINT}
	case "dynamodb":
		// DynamoDB tables: Storage-based carbon (SSD × 3× replication factor)
		return []pbc.MetricKind{pbc.MetricKind_METRIC_KIND_CARBON_FOOTPRINT}
	case "eks":
		// EKS clusters: Control plane returns 0 (shared infrastructure); worker nodes estimated as EC2
		return []pbc.MetricKind{pbc.MetricKind_METRIC_KIND_CARBON_FOOTPRINT}
	case "elasticache":
		// ElastiCache clusters: EC2-equivalent node carbon × cluster size
		return []pbc.MetricKind{pbc.MetricKind_METRIC_KIND_CARBON_FOOTPRINT}
	default:
		// ELB, NAT Gateway, CloudWatch: No carbon estimation yet
		return nil
	}
}