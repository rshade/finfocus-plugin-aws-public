package router

import (
	"context"

	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

// Compile-time check that Plugin implements the optional
// ResolveResourceTypesProvider interface. This also causes the SDK to
// advertise PLUGIN_CAPABILITY_RESOLVE_RESOURCE_TYPES in GetPluginInfo,
// which FinFocus Core requires before sending ResolveResourceTypes requests.
var _ pluginsdk.ResolveResourceTypesProvider = (*Plugin)(nil)

// ResolveResourceTypes translates IaC resource type strings (e.g., Terraform's
// "aws_instance") into Pulumi type tokens. The mappings are static and
// region-independent, so the router answers directly from its embedded type
// registry instead of delegating to a region child. This lets FinFocus Core
// resolve types up front, before any region-specific routing happens.
//
// Unknown types are omitted from the response (not an error), allowing the
// core to fall back to heuristic type conversion.
func (r *Plugin) ResolveResourceTypes(
	_ context.Context,
	req *pbc.ResolveResourceTypesRequest,
) (*pbc.ResolveResourceTypesResponse, error) {
	return r.typeRegistry.Resolve(req), nil
}
