package plugin

import (
	"context"
	"testing"

	"github.com/rs/zerolog"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetProjectedCost_ASGFromLaunchTemplateRef prices an ASG from the
// launch template core already sends on ref.launchTemplate.*.
func TestGetProjectedCost_ASGFromLaunchTemplateRef(t *testing.T) {
	mock := newMockPricingClient("us-east-1", "USD")
	mock.ec2Prices["m5.large/Linux/Shared"] = 0.096
	plugin := NewAWSPublicPlugin("us-east-1", "test-version", mock, zerolog.Nop())

	resp, err := plugin.GetProjectedCost(context.Background(), &pbc.GetProjectedCostRequest{
		Resource: &pbc.ResourceDescriptor{
			Provider:     "aws",
			ResourceType: "aws:autoscaling/group:Group",
			Region:       "us-east-1",
			Tags: map[string]string{
				"ref.launchTemplate.type": "aws:ec2/launchTemplate:LaunchTemplate",
				"ref.launchTemplate.sku":  "m5.large",
			},
		},
	})
	require.NoError(t, err)
	assert.InDelta(t, 0.096*730, resp.GetCostPerMonth(), 0.001)
	assert.Contains(t, resp.GetBillingDetail(), "instance type from launch template")
	assert.NotContains(t, resp.GetMetadata()["defaults_applied"], "m5.large")

	wrong, err := plugin.GetProjectedCost(context.Background(), &pbc.GetProjectedCostRequest{
		Resource: &pbc.ResourceDescriptor{
			Provider:     "aws",
			ResourceType: "aws:autoscaling/group:Group",
			Region:       "us-east-1",
			Tags: map[string]string{
				"ref.launchTemplate.type": "aws:ec2/instance:Instance",
				"ref.launchTemplate.sku":  "m5.large",
			},
		},
	})
	require.NoError(t, err)
	assert.InDelta(t, 0.0, wrong.GetCostPerMonth(), 0)
	assert.Contains(t, wrong.GetBillingDetail(), "ref.launchTemplate.sku")
}
