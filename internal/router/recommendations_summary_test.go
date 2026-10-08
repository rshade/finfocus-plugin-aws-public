package router

import (
	"context"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/rs/zerolog"
	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	"github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1/pbcconnect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recommendationChild struct {
	pbcconnect.UnimplementedCostSourceServiceHandler
}

func (recommendationChild) GetRecommendations(
	_ context.Context, req *connect.Request[pbc.GetRecommendationsRequest],
) (*connect.Response[pbc.GetRecommendationsResponse], error) {
	return connect.NewResponse(&pbc.GetRecommendationsResponse{
		Recommendations: []*pbc.Recommendation{{
			Id:     req.Msg.GetTargetResources()[0].GetRegion(),
			Impact: &pbc.RecommendationImpact{EstimatedSavings: 10, Currency: "USD"},
		}},
		Summary: &pbc.RecommendationSummary{TotalRecommendations: 1, TotalEstimatedSavings: 10, Currency: "USD"},
	}), nil
}

func TestRecommendationsMultiRegionSummary(t *testing.T) {
	t.Parallel()
	_, handler := pbcconnect.NewCostSourceServiceHandler(recommendationChild{})
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	logger := zerolog.New(zerolog.NewTestWriter(t))
	router := NewPlugin("test", logger, t.TempDir(), true, nil)
	client := pluginsdk.NewClient(pluginsdk.ClientConfig{BaseURL: server.URL, HTTPClient: server.Client()})
	for _, region := range []string{"us-east-1", "us-west-2"} {
		router.registry.children[region] = &ChildProcess{region: region, state: ChildStateReady, client: client}
	}
	response, err := router.GetRecommendations(context.Background(), &pbc.GetRecommendationsRequest{
		ProjectionPeriod: "monthly",
		TargetResources:  []*pbc.ResourceDescriptor{{Region: "us-east-1"}, {Region: "us-west-2"}},
	})
	require.NoError(t, err)
	require.NotNil(t, response.GetSummary(), "SDK requires a summary on merged responses")
	assert.Len(t, response.GetRecommendations(), 2)
	assert.Equal(t, int32(2), response.GetSummary().GetTotalRecommendations())
	assert.Equal(t, 20.0, response.GetSummary().GetTotalEstimatedSavings())
	assert.Equal(t, "USD", response.GetSummary().GetCurrency())
	assert.Equal(t, "monthly", response.GetSummary().GetProjectionPeriod())
}
