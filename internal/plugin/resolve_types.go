package plugin

import (
	"context"
	"time"

	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

// Compile-time check that AWSPublicPlugin implements the optional
// ResolveResourceTypesProvider interface. This also causes the SDK to
// advertise PLUGIN_CAPABILITY_RESOLVE_RESOURCE_TYPES in GetPluginInfo,
// which FinFocus Core requires before sending ResolveResourceTypes requests.
var _ pluginsdk.ResolveResourceTypesProvider = (*AWSPublicPlugin)(nil)

// ResolveResourceTypes translates IaC resource type strings (e.g., Terraform's
// "aws_instance") into the Pulumi type tokens used by this plugin
// (e.g., "aws:ec2/instance:Instance"). The mappings are static and
// region-independent; lookups are served from the embedded type registry.
//
// Unknown types are omitted from the response (not an error), allowing the
// core to fall back to heuristic type conversion.
func (p *AWSPublicPlugin) ResolveResourceTypes(
	ctx context.Context,
	req *pbc.ResolveResourceTypesRequest,
) (*pbc.ResolveResourceTypesResponse, error) {
	start := time.Now()
	traceID := p.getTraceID(ctx)

	resp := p.typeRegistry.Resolve(req)

	p.traceLogger(traceID, "ResolveResourceTypes").Info().
		Str("source_format", req.GetSourceFormat().String()).
		Int("source_types", len(req.GetSourceTypes())).
		Int("resolved", len(resp.GetMappings())).
		Int64(pluginsdk.FieldDurationMs, time.Since(start).Milliseconds()).
		Msg("resource types resolved")

	return resp, nil
}
