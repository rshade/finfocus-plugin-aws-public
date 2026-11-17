package router

import (
	"testing"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	"github.com/stretchr/testify/assert"
)

// TestRegionRecommendationsRequest_ForwardsVisibility verifies the child
// request keeps excluded ids, include_dismissed, and projection_period.
func TestRegionRecommendationsRequest_ForwardsVisibility(t *testing.T) {
	t.Parallel()
	resources := []*pbc.ResourceDescriptor{{
		Provider: "aws",
		Region:   "us-east-1",
		Sku:      "t3.micro",
	}}
	req := &pbc.GetRecommendationsRequest{
		Filter:                    &pbc.RecommendationFilter{Region: "us-east-1"},
		TargetResources:           resources,
		ExcludedRecommendationIds: []string{"11111111-1111-1111-1111-111111111111"},
		IncludeDismissed:          true,
		ProjectionPeriod:          "monthly",
	}

	got := regionRecommendationsRequest(req, resources)
	assert.Equal(t, req.GetFilter(), got.GetFilter())
	assert.Equal(t, resources, got.GetTargetResources())
	assert.Equal(t, req.GetExcludedRecommendationIds(), got.GetExcludedRecommendationIds())
	assert.True(t, got.GetIncludeDismissed())
	assert.Equal(t, "monthly", got.GetProjectionPeriod())
}
