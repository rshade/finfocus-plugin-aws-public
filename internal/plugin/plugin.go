package plugin

import (
	"context"

	pbc "github.com/rshade/pulumicost-spec/sdk/go/proto/pulumicost/v1"
	"github.com/rshade/pulumicost-plugin-aws-public/internal/pricing"
)

// AWSPublicPlugin implements the pluginsdk.Plugin interface for AWS public pricing.
type AWSPublicPlugin struct {
	region  string
	pricing pricing.PricingClient
}

// NewAWSPublicPlugin creates a new AWSPublicPlugin instance.
// The region should match the region for which pricing data is embedded.
func NewAWSPublicPlugin(region string, pricingClient pricing.PricingClient) *AWSPublicPlugin {
	return &AWSPublicPlugin{
		region:  region,
		pricing: pricingClient,
	}
}

// Name returns the plugin name identifier.
func (p *AWSPublicPlugin) Name() string {
	return "pulumicost-plugin-aws-public"
}

// GetActualCost retrieves actual cost for a resource.
// This plugin does not support actual cost retrieval - it only provides projected costs based on public pricing.
func (p *AWSPublicPlugin) GetActualCost(ctx context.Context, req *pbc.GetActualCostRequest) (*pbc.GetActualCostResponse, error) {
	// Not implemented - return nil to indicate unsupported
	return nil, nil
}
