// Package typeregistry provides the Terraform-to-Pulumi resource type mappings
// served via the finfocus-spec ResolveResourceTypes RPC (finfocus-spec >= v0.6.1).
//
// FinFocus Core v0.3.7+ sends raw Terraform resource type strings (e.g.,
// "aws_instance") from --terraform-state ingestion and asks plugins to translate
// them into Pulumi type tokens (e.g., "aws:ec2/instance:Instance"). The mappings
// are static and region-independent, so both the region-specific plugin binary
// and the multi-region router serve the same registry.
package typeregistry

import (
	"time"

	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

// defaultTTL is the caching hint applied to every Resolve() response.
// Type mappings are near-static (they change only with plugin releases), so a
// long TTL is appropriate; callers may apply their own maximum TTL policy.
const defaultTTL = 24 * time.Hour

// TerraformMappings maps Terraform resource type strings to the Pulumi type
// tokens understood by this plugin's resource type normalization
// (see internal/plugin normalizeResourceType/detectService). Only resource
// types the plugin can price (including zero-cost networking/IAM resources)
// are included; unknown types are omitted from resolution responses so the
// core can fall back to heuristic conversion.
//
// Read-only after package initialization. Do not modify at runtime.
var TerraformMappings = map[string]string{
	// Compute
	"aws_instance":             "aws:ec2/instance:Instance",
	"aws_launch_template":      "aws:ec2/launchTemplate:LaunchTemplate",
	"aws_launch_configuration": "aws:ec2/launchConfiguration:LaunchConfiguration",
	"aws_autoscaling_group":    "aws:autoscaling/group:Group",

	// Storage
	"aws_ebs_volume": "aws:ebs/volume:Volume",
	"aws_s3_bucket":  "aws:s3/bucket:Bucket",

	// Database and caching
	"aws_db_instance":         "aws:rds/instance:Instance",
	"aws_dynamodb_table":      "aws:dynamodb/table:Table",
	"aws_elasticache_cluster": "aws:elasticache/cluster:Cluster",

	// Networking
	"aws_lb":             "aws:lb/loadBalancer:LoadBalancer",
	"aws_alb":            "aws:lb/loadBalancer:LoadBalancer",
	"aws_nat_gateway":    "aws:ec2/natGateway:NatGateway",
	"aws_vpc":            "aws:ec2/vpc:Vpc",
	"aws_subnet":         "aws:ec2/subnet:Subnet",
	"aws_security_group": "aws:ec2/securityGroup:SecurityGroup",

	// Serverless and containers
	"aws_lambda_function": "aws:lambda/function:Function",
	"aws_eks_cluster":     "aws:eks/cluster:Cluster",

	// Observability
	"aws_cloudwatch_log_group":    "aws:cloudwatch/logGroup:LogGroup",
	"aws_cloudwatch_metric_alarm": "aws:cloudwatch/metricAlarm:MetricAlarm",

	// IAM (zero-cost)
	"aws_iam_role":             "aws:iam/role:Role",
	"aws_iam_policy":           "aws:iam/policy:Policy",
	"aws_iam_instance_profile": "aws:iam/instanceProfile:InstanceProfile",
	"aws_iam_user":             "aws:iam/user:User",
}

// New creates a TypeRegistry pre-populated with the Terraform mappings and a
// 24-hour default expires_at caching hint. The returned registry is safe for
// concurrent reads after construction.
func New() *pluginsdk.TypeRegistry {
	registry := pluginsdk.NewTypeRegistry(pluginsdk.WithDefaultTTL(defaultTTL))
	registry.RegisterMappings(pbc.SourceFormat_SOURCE_FORMAT_TERRAFORM, TerraformMappings)
	return registry
}
