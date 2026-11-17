package plugin

import (
	"context"
	"maps"
	"testing"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func recommendationPlugin(prices map[string]float64) *AWSPublicPlugin {
	mock := newMockPricingClient("us-east-1", "USD")
	maps.Copy(mock.ec2Prices, prices)
	return NewAWSPublicPlugin("us-east-1", "test-version", mock, zerolog.Nop())
}

func ec2Prices() map[string]float64 {
	return map[string]float64{
		"t2.micro/Linux/Shared":  0.0116,
		"t3.micro/Linux/Shared":  0.0104,
		"m5.large/Linux/Shared":  0.096,
		"m6i.large/Linux/Shared": 0.096,
	}
}

func recRequest(resources ...*pbc.ResourceDescriptor) *pbc.GetRecommendationsRequest {
	return &pbc.GetRecommendationsRequest{TargetResources: resources}
}

func ec2Resource(id, sku string) *pbc.ResourceDescriptor {
	return &pbc.ResourceDescriptor{
		Id:           id,
		Provider:     "aws",
		ResourceType: "ec2",
		Sku:          sku,
		Region:       "us-east-1",
	}
}

func idsOf(recs []*pbc.Recommendation) []string {
	out := make([]string, len(recs))
	for i, rec := range recs {
		out[i] = rec.GetId()
	}
	return out
}

// TestGetRecommendations_StableIDs verifies that the same resources produce the
// same recommendation ids, that a price change does not mint a new id, and
// that a different target or a different resource identity does.
func TestGetRecommendations_StableIDs(t *testing.T) {
	plugin := recommendationPlugin(ec2Prices())
	resource := ec2Resource("urn:pulumi:dev::app::aws:ec2/instance:Instance::web", "t2.micro")

	first, err := plugin.GetRecommendations(context.Background(), recRequest(resource))
	require.NoError(t, err)
	require.NotEmpty(t, first.GetRecommendations())

	second, err := plugin.GetRecommendations(context.Background(), recRequest(resource))
	require.NoError(t, err)
	assert.Equal(t, idsOf(first.GetRecommendations()), idsOf(second.GetRecommendations()))
	for _, id := range idsOf(first.GetRecommendations()) {
		_, parseErr := uuid.Parse(id)
		assert.NoError(t, parseErr, "id %q", id)
	}

	plugin.pricing.(*mockPricingClient).ec2Prices["t2.micro/Linux/Shared"] = 0.02
	plugin.pricing.(*mockPricingClient).ec2Prices["t3.micro/Linux/Shared"] = 0.015
	repriced, err := plugin.GetRecommendations(context.Background(), recRequest(resource))
	require.NoError(t, err)
	assert.Equal(t, idsOf(first.GetRecommendations()), idsOf(repriced.GetRecommendations()))

	other := ec2Resource("urn:pulumi:dev::app::aws:ec2/instance:Instance::api", "t2.micro")
	differentResource, err := plugin.GetRecommendations(context.Background(), recRequest(other))
	require.NoError(t, err)
	assert.NotEqual(t, first.GetRecommendations()[0].GetId(), differentResource.GetRecommendations()[0].GetId())

	differentSKU, err := plugin.GetRecommendations(
		context.Background(), recRequest(ec2Resource(resource.GetId(), "m5.large")),
	)
	require.NoError(t, err)
	assert.NotEqual(t, first.GetRecommendations()[0].GetId(), differentSKU.GetRecommendations()[0].GetId())
}

// TestGetRecommendations_IDsUniquePerIdentity verifies two resources with the
// same SKU and region keep distinct ids, and two anonymous resources do not collide.
func TestGetRecommendations_IDsUniquePerIdentity(t *testing.T) {
	plugin := recommendationPlugin(ec2Prices())
	resp, err := plugin.GetRecommendations(context.Background(), recRequest(
		ec2Resource("web", "t2.micro"),
		ec2Resource("api", "t2.micro"),
		ec2Resource("", "t2.micro"),
		ec2Resource("", "t2.micro"),
	))
	require.NoError(t, err)
	seen := map[string]struct{}{}
	for _, rec := range resp.GetRecommendations() {
		_, dup := seen[rec.GetId()]
		assert.False(t, dup, "duplicate id %s", rec.GetId())
		seen[rec.GetId()] = struct{}{}
	}
	assert.GreaterOrEqual(t, len(seen), 4)
}

// TestGetRecommendations_ExcludedIDs verifies an excluded id is omitted even
// when include_dismissed is set.
func TestGetRecommendations_ExcludedIDs(t *testing.T) {
	plugin := recommendationPlugin(ec2Prices())
	resource := ec2Resource("web", "t2.micro")
	first, err := plugin.GetRecommendations(context.Background(), recRequest(resource))
	require.NoError(t, err)
	require.NotEmpty(t, first.GetRecommendations())
	excluded := first.GetRecommendations()[0].GetId()

	hidden, err := plugin.GetRecommendations(context.Background(), &pbc.GetRecommendationsRequest{
		TargetResources:           []*pbc.ResourceDescriptor{resource},
		ExcludedRecommendationIds: []string{excluded},
		IncludeDismissed:          true,
	})
	require.NoError(t, err)
	for _, rec := range hidden.GetRecommendations() {
		assert.NotEqual(t, excluded, rec.GetId())
	}
	assert.Less(t, len(hidden.GetRecommendations()), len(first.GetRecommendations()))
}
