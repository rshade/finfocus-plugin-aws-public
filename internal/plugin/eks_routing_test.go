package plugin

import (
	"context"
	"testing"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
)

// TestEKSTypeRouting_OnlyClusterPaysControlPlaneFee walks the nine aws:eks/
// resource tokens. Only the cluster pays the control-plane fee.
func TestEKSTypeRouting_OnlyClusterPaysControlPlaneFee(t *testing.T) {
	ctx := context.Background()
	plugin := newTestPluginForActual()

	descriptor := func(rt, sku string, attrs map[string]any) *pbc.ResourceDescriptor {
		res := &pbc.ResourceDescriptor{Provider: "aws", ResourceType: rt, Sku: sku, Region: "us-east-1"}
		if attrs != nil {
			st, err := structpb.NewStruct(attrs)
			require.NoError(t, err)
			res.Attributes = st
		}
		return res
	}

	t.Run("cluster keeps the control-plane fee", func(t *testing.T) {
		resp, err := plugin.GetProjectedCost(ctx, &pbc.GetProjectedCostRequest{
			Resource: descriptor("aws:eks/cluster:Cluster", "cluster", nil),
		})
		require.NoError(t, err)
		assert.InDelta(t, 73.0, resp.GetCostPerMonth(), 0.001)
	})

	zeroCost := []struct{ resourceType, sku string }{
		{"aws:eks/addon:Addon", "addon"},
		{"aws:eks/accessEntry:AccessEntry", "STANDARD"},
		{"aws:eks/accessPolicyAssociation:AccessPolicyAssociation", ""},
		{"aws:eks/podIdentityAssociation:PodIdentityAssociation", ""},
		{"aws:eks/identityProviderConfig:IdentityProviderConfig", ""},
		{"aws:eks/fargateProfile:FargateProfile", ""},
	}
	for _, tt := range zeroCost {
		t.Run(tt.resourceType, func(t *testing.T) {
			assert.NotEqual(t, serviceEKS, detectService(normalizeResourceType(tt.resourceType)))
			res := descriptor(tt.resourceType, tt.sku, nil)

			sup, err := plugin.Supports(ctx, &pbc.SupportsRequest{Resource: res})
			require.NoError(t, err)
			assert.True(t, sup.GetSupported())

			resp, err := plugin.GetProjectedCost(ctx, &pbc.GetProjectedCostRequest{Resource: res})
			require.NoError(t, err)
			assert.InDelta(t, 0.0, resp.GetCostPerMonth(), 0)
			assert.NotContains(t, resp.GetBillingDetail(), "EKS cluster")
		})
	}

	t.Run("capability is billed, so it is neither the cluster fee nor zero-cost", func(t *testing.T) {
		rt := "aws:eks/capability:Capability"
		svc := detectService(normalizeResourceType(rt))
		assert.NotEqual(t, serviceEKS, svc)
		assert.False(t, IsZeroCostService(svc))

		sup, err := plugin.Supports(ctx, &pbc.SupportsRequest{Resource: descriptor(rt, "ARGOCD", nil)})
		require.NoError(t, err)
		assert.False(t, sup.GetSupported())
		assert.Contains(t, sup.GetReason(), "Capabilities")
	})

	t.Run("managed node group is priced from its instances", func(t *testing.T) {
		resp, err := plugin.GetProjectedCost(ctx, &pbc.GetProjectedCostRequest{
			Resource: descriptor("aws:eks/nodeGroup:NodeGroup", "", map[string]any{
				"instanceTypes": []any{"m5.large"},
				"scalingConfig": map[string]any{"desiredSize": 3, "minSize": 1, "maxSize": 5},
			}),
		})
		require.NoError(t, err)
		assert.InDelta(t, 3*0.096*730, resp.GetCostPerMonth(), 0.001)
		assert.NotContains(t, resp.GetBillingDetail(), "EKS cluster")
	})
}
