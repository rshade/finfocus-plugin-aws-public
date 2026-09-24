package router

import (
	"context"
	"testing"

	"github.com/rs/zerolog"
	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPlugin_ImplementsResolveResourceTypesProvider verifies that the SDK
// advertises PLUGIN_CAPABILITY_RESOLVE_RESOURCE_TYPES for the router. The
// compile-time interface assertion lives in resolve_types.go.
func TestPlugin_ImplementsResolveResourceTypesProvider(t *testing.T) {
	plugin := NewPlugin("1.0.0", zerolog.Nop(), t.TempDir(), true, nil)
	server := pluginsdk.NewServer(plugin)

	assert.Contains(t, server.GetGlobalCapabilities(),
		pbc.PluginCapability_PLUGIN_CAPABILITY_RESOLVE_RESOURCE_TYPES)
}

// TestPlugin_GetPluginInfo_AdvertisesCapabilities guards the PluginInfoProvider
// path: the SDK serves the router's own GetPluginInfo response verbatim, so the
// capabilities must be set there — FinFocus Core gates ResolveResourceTypes on
// them.
func TestPlugin_GetPluginInfo_AdvertisesCapabilities(t *testing.T) {
	plugin := NewPlugin("1.0.0", zerolog.Nop(), t.TempDir(), true, nil)

	resp, err := plugin.GetPluginInfo(context.Background(), &pbc.GetPluginInfoRequest{})
	require.NoError(t, err)

	assert.Contains(t, resp.GetCapabilities(),
		pbc.PluginCapability_PLUGIN_CAPABILITY_RESOLVE_RESOURCE_TYPES)
	assert.Contains(t, resp.GetCapabilities(),
		pbc.PluginCapability_PLUGIN_CAPABILITY_DRY_RUN)
}

// TestPlugin_ResolveResourceTypes_ServedLocally verifies that the router answers
// type resolution from its own registry without launching or delegating to a
// region child; the test registry is empty, so any delegation would fail.
func TestPlugin_ResolveResourceTypes_ServedLocally(t *testing.T) {
	plugin := NewPlugin("1.0.0", zerolog.Nop(), t.TempDir(), true, nil)

	resp, err := plugin.ResolveResourceTypes(context.Background(), &pbc.ResolveResourceTypesRequest{
		SourceFormat: pbc.SourceFormat_SOURCE_FORMAT_TERRAFORM,
		SourceTypes:  []string{"aws_instance", "aws_lambda_function", "aws_unknown_thing"},
	})
	require.NoError(t, err)

	require.Len(t, resp.GetMappings(), 2)
	assert.Equal(t, "aws:ec2/instance:Instance", resp.GetMappings()["aws_instance"].GetPulumiToken())
	assert.Equal(t, "aws:lambda/function:Function", resp.GetMappings()["aws_lambda_function"].GetPulumiToken())
	assert.NotContains(t, resp.GetMappings(), "aws_unknown_thing")
}

// TestPlugin_ResolveResourceTypes_UnspecifiedFormat verifies that the router
// returns no mappings for a request with SOURCE_FORMAT_UNSPECIFIED.
func TestPlugin_ResolveResourceTypes_UnspecifiedFormat(t *testing.T) {
	plugin := NewPlugin("1.0.0", zerolog.Nop(), t.TempDir(), true, nil)

	resp, err := plugin.ResolveResourceTypes(context.Background(), &pbc.ResolveResourceTypesRequest{
		SourceFormat: pbc.SourceFormat_SOURCE_FORMAT_UNSPECIFIED,
		SourceTypes:  []string{"aws_instance"},
	})
	require.NoError(t, err)

	assert.Empty(t, resp.GetMappings())
}
