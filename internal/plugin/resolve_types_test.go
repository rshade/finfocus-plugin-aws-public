package plugin

import (
	"context"
	"testing"

	"github.com/rs/zerolog"
	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rshade/finfocus-plugin-aws-public/internal/typeregistry"
)

func newResolveTestPlugin() *AWSPublicPlugin {
	return NewAWSPublicPlugin("us-east-1", "test-version", newMockPricingClient("us-east-1", "USD"), zerolog.Nop())
}

// TestAWSPublicPlugin_ImplementsResolveResourceTypesProvider verifies that the
// SDK advertises PLUGIN_CAPABILITY_RESOLVE_RESOURCE_TYPES for the plugin. The
// compile-time interface assertion lives in resolve_types.go.
func TestAWSPublicPlugin_ImplementsResolveResourceTypesProvider(t *testing.T) {
	plugin := newResolveTestPlugin()
	server := pluginsdk.NewServer(plugin)

	assert.Contains(t, server.GetGlobalCapabilities(),
		pbc.PluginCapability_PLUGIN_CAPABILITY_RESOLVE_RESOURCE_TYPES)
}

// TestGetPluginInfo_AdvertisesCapabilities guards the PluginInfoProvider path:
// the SDK serves the plugin's own GetPluginInfo response verbatim, so the
// capabilities (and legacy metadata keys) must be set there — FinFocus Core
// gates ResolveResourceTypes on them.
func TestGetPluginInfo_AdvertisesCapabilities(t *testing.T) {
	plugin := newResolveTestPlugin()

	resp, err := plugin.GetPluginInfo(context.Background(), &pbc.GetPluginInfoRequest{})
	require.NoError(t, err)

	assert.Contains(t, resp.GetCapabilities(),
		pbc.PluginCapability_PLUGIN_CAPABILITY_RESOLVE_RESOURCE_TYPES)
	assert.Contains(t, resp.GetCapabilities(),
		pbc.PluginCapability_PLUGIN_CAPABILITY_RECOMMENDATIONS)
	assert.Equal(t, "true", resp.GetMetadata()["supports_resolve_resource_types"])
}

// TestResolveResourceTypes_ResolvesTerraformTypes verifies that the plugin maps
// Terraform resource types to their Pulumi tokens and marks them supported.
func TestResolveResourceTypes_ResolvesTerraformTypes(t *testing.T) {
	plugin := newResolveTestPlugin()

	resp, err := plugin.ResolveResourceTypes(context.Background(), &pbc.ResolveResourceTypesRequest{
		SourceFormat: pbc.SourceFormat_SOURCE_FORMAT_TERRAFORM,
		SourceTypes:  []string{"aws_instance", "aws_s3_bucket"},
	})
	require.NoError(t, err)

	require.Len(t, resp.GetMappings(), 2)
	assert.Equal(t, "aws:ec2/instance:Instance", resp.GetMappings()["aws_instance"].GetPulumiToken())
	assert.True(t, resp.GetMappings()["aws_instance"].GetSupported())
	assert.Equal(t, "aws:s3/bucket:Bucket", resp.GetMappings()["aws_s3_bucket"].GetPulumiToken())
}

// TestResolveResourceTypes_OmitsUnknownTypes verifies that non-AWS or unknown
// Terraform types are omitted from the plugin's response.
func TestResolveResourceTypes_OmitsUnknownTypes(t *testing.T) {
	plugin := newResolveTestPlugin()

	resp, err := plugin.ResolveResourceTypes(context.Background(), &pbc.ResolveResourceTypesRequest{
		SourceFormat: pbc.SourceFormat_SOURCE_FORMAT_TERRAFORM,
		SourceTypes:  []string{"aws_instance", "google_compute_instance"},
	})
	require.NoError(t, err)

	require.Len(t, resp.GetMappings(), 1)
	assert.Contains(t, resp.GetMappings(), "aws_instance")
	assert.NotContains(t, resp.GetMappings(), "google_compute_instance")
}

// TestResolveResourceTypes_SetsExpiresAt verifies that the plugin's response
// includes an expires_at timestamp so Core can cache the mappings.
func TestResolveResourceTypes_SetsExpiresAt(t *testing.T) {
	plugin := newResolveTestPlugin()

	resp, err := plugin.ResolveResourceTypes(context.Background(), &pbc.ResolveResourceTypesRequest{
		SourceFormat: pbc.SourceFormat_SOURCE_FORMAT_TERRAFORM,
		SourceTypes:  []string{"aws_instance"},
	})
	require.NoError(t, err)

	assert.NotNil(t, resp.GetExpiresAt())
}

// TestTerraformMappings_AllTokensRouteToSupportedServices guards against drift
// between the Terraform mapping table and the plugin's resource type routing:
// every Pulumi token we hand out must normalize to a service this plugin
// actually estimates (or a known zero-cost service).
func TestTerraformMappings_AllTokensRouteToSupportedServices(t *testing.T) {
	supported := map[string]bool{
		serviceEC2: true, serviceEBS: true, serviceS3: true, serviceRDS: true,
		serviceEKS: true, serviceLambda: true, serviceDynamoDB: true,
		serviceELB: true, serviceNATGW: true, serviceCloudWatch: true,
		serviceElastiCache: true, serviceASG: true,
		serviceVPC: true, serviceSecurityGroup: true, serviceSubnet: true,
		serviceIAM: true, serviceLaunchTmpl: true, serviceLaunchConfig: true,
	}

	for tfType, pulumiToken := range typeregistry.TerraformMappings {
		t.Run(tfType, func(t *testing.T) {
			service := detectService(normalizeResourceType(pulumiToken))
			assert.True(t, supported[service],
				"token %q (from %q) routes to unsupported service %q", pulumiToken, tfType, service)
		})
	}
}
