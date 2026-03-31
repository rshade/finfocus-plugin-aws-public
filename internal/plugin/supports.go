package plugin

import (
	"context"
	"fmt"
	"time"

	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

// Supports checks if this plugin can estimate costs for the given resource.
func (p *AWSPublicPlugin) Supports( //nolint:funlen
	ctx context.Context,
	req *pbc.SupportsRequest,
) (*pbc.SupportsResponse, error) {
	start := time.Now()
	traceID := p.getTraceID(ctx)

	if req == nil || req.GetResource() == nil {
		p.traceLogger(traceID, "Supports").Info().
			Str(pluginsdk.FieldErrorCode, pbc.ErrorCode_ERROR_CODE_INVALID_RESOURCE.String()).
			Int64(pluginsdk.FieldDurationMs, time.Since(start).Milliseconds()).
			Msg("resource support check")

		return &pbc.SupportsResponse{
			Supported: false,
			Reason:    "Invalid request: missing resource descriptor",
		}, nil
	}

	resource := req.GetResource()

	// Use serviceResolver for consistent normalization and service detection.
	// This caches the computation within this request (optimization implemented per T019).
	resolver := newServiceResolver(resource.GetResourceType())
	serviceType := resolver.ServiceType()

	// Check provider
	if resource.GetProvider() != providerAWS {
		p.traceLogger(traceID, "Supports").Info().
			Str(pluginsdk.FieldResourceType, resource.GetResourceType()).
			Str("aws_region", resource.GetRegion()).
			Bool("supported", false).
			Int64(pluginsdk.FieldDurationMs, time.Since(start).Milliseconds()).
			Msg("resource support check")

		return &pbc.SupportsResponse{
			Supported: false,
			Reason: fmt.Sprintf(
				"Provider %q not supported (only %q is supported)",
				resource.GetProvider(),
				providerAWS,
			),
		}, nil
	}

	// Check region match
	// For global services (S3, IAM) and zero-cost resources (VPC, SecurityGroup, Subnet),
	// allow empty region and default to plugin region.
	effectiveRegion := resource.GetRegion()
	if effectiveRegion == "" &&
		(serviceType == serviceS3 || serviceType == serviceIAM || IsZeroCostService(serviceType)) {
		effectiveRegion = p.region
	}

	if effectiveRegion != p.region {
		p.traceLogger(traceID, "Supports").Info().
			Str(pluginsdk.FieldResourceType, resource.GetResourceType()).
			Str("aws_region", resource.GetRegion()).
			Bool("supported", false).
			Int64(pluginsdk.FieldDurationMs, time.Since(start).Milliseconds()).
			Msg("resource support check")

		return &pbc.SupportsResponse{
			Supported: false,
			Reason: fmt.Sprintf(
				"Region not supported by this binary (plugin region: %s, resource region: %s)",
				p.region,
				resource.GetRegion(),
			),
		}, nil
	}

	// Check resource type
	switch serviceType {
	case serviceEC2, serviceRDS, serviceLambda, serviceS3, serviceEBS,
		serviceEKS, serviceDynamoDB, serviceElastiCache, serviceASG:
		// These services support cost estimation
		// EC2 also supports carbon footprint estimation
		supportedMetrics := getSupportedMetrics(serviceType)
		p.traceLogger(traceID, "Supports").Info().
			Str(pluginsdk.FieldResourceType, resource.GetResourceType()).
			Str("aws_region", resource.GetRegion()).
			Bool("supported", true).
			Int("supported_metrics_count", len(supportedMetrics)).
			Int64(pluginsdk.FieldDurationMs, time.Since(start).Milliseconds()).
			Msg("resource support check")

		return &pbc.SupportsResponse{
			Supported:        true,
			Reason:           "",
			SupportedMetrics: supportedMetrics,
		}, nil

	case serviceELB, serviceNATGW, serviceCloudWatch:
		// Supported but no carbon estimation yet
		p.traceLogger(traceID, "Supports").Info().
			Str(pluginsdk.FieldResourceType, resource.GetResourceType()).
			Str("aws_region", resource.GetRegion()).
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
		if IsZeroCostService(serviceType) {
			p.traceLogger(traceID, "Supports").Info().
				Str(pluginsdk.FieldResourceType, resource.GetResourceType()).
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
			Str(pluginsdk.FieldResourceType, resource.GetResourceType()).
			Str("aws_region", resource.GetRegion()).
			Bool("supported", false).
			Int64(pluginsdk.FieldDurationMs, time.Since(start).Milliseconds()).
			Msg("resource support check")

		return &pbc.SupportsResponse{
			Supported:        false,
			Reason:           fmt.Sprintf("Resource type %q not supported", resource.GetResourceType()),
			SupportedMetrics: nil,
		}, nil
	}
}

// getSupportedMetrics returns the list of supported metric kinds for a given resource type.
// Services with carbon footprint estimation return METRIC_KIND_CARBON_FOOTPRINT.
// resourceType is the normalized resource type (e.g., "ec2", "rds", "lambda", "s3", "ebs", "eks", "dynamodb", "elasticache").
func getSupportedMetrics(resourceType string) []pbc.MetricKind {
	switch resourceType {
	case serviceEC2:
		// EC2 instances: CPU/GPU power × utilization × grid factor + optional embodied carbon
		return []pbc.MetricKind{pbc.MetricKind_METRIC_KIND_CARBON_FOOTPRINT}
	case serviceEBS:
		// EBS volumes: Storage energy × replication factor × grid factor
		return []pbc.MetricKind{pbc.MetricKind_METRIC_KIND_CARBON_FOOTPRINT}
	case serviceS3:
		// S3 storage: Storage energy × replication factor × grid factor (by storage class)
		return []pbc.MetricKind{pbc.MetricKind_METRIC_KIND_CARBON_FOOTPRINT}
	case serviceLambda:
		// Lambda functions: vCPU-equivalent × duration × grid factor (ARM64 efficiency adjusted)
		return []pbc.MetricKind{pbc.MetricKind_METRIC_KIND_CARBON_FOOTPRINT}
	case serviceRDS:
		// RDS instances: Compute carbon + storage carbon (Multi-AZ 2× multiplier)
		return []pbc.MetricKind{pbc.MetricKind_METRIC_KIND_CARBON_FOOTPRINT}
	case serviceDynamoDB:
		// DynamoDB tables: Storage-based carbon (SSD × 3× replication factor)
		return []pbc.MetricKind{pbc.MetricKind_METRIC_KIND_CARBON_FOOTPRINT}
	case serviceEKS:
		// EKS clusters: Control plane returns 0 (shared infrastructure); worker nodes estimated as EC2
		return []pbc.MetricKind{pbc.MetricKind_METRIC_KIND_CARBON_FOOTPRINT}
	case serviceElastiCache:
		// ElastiCache clusters: EC2-equivalent node carbon × cluster size
		return []pbc.MetricKind{pbc.MetricKind_METRIC_KIND_CARBON_FOOTPRINT}
	case serviceASG:
		// ASG: EC2-equivalent carbon × desired capacity
		return []pbc.MetricKind{pbc.MetricKind_METRIC_KIND_CARBON_FOOTPRINT}
	default:
		// ELB, NAT Gateway, CloudWatch: No carbon estimation yet
		return nil
	}
}
