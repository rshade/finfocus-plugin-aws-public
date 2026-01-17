package plugin

import (
	"github.com/rs/zerolog"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

// Note: The following feature detection functions are placeholders for future capabilities.
// They are intentionally kept for documentation purposes and will be activated when
// the corresponding fields become available in the finfocus-spec proto definitions.

// hasUsageProfile checks if UsageProfile field exists in request (future feature placeholder)
func hasUsageProfile(req *pbc.ResourceDescriptor) bool { //nolint:unused
	// UsageProfile does not exist yet in current finfocus-spec version
	return false
}

// hasLineage checks if Lineage field exists in response (future feature placeholder)
func hasLineage(resp *pbc.GetProjectedCostResponse) bool { //nolint:unused
	// CostAllocationLineage does not exist yet in current finfocus-spec version
	return false
}

// setGrowthHint sets the growth_type field based on service classification.
// GrowthType field is available in finfocus-spec v0.4.12+, so no feature detection is needed.
func setGrowthHint(logger zerolog.Logger, serviceType string, resp *pbc.GetProjectedCostResponse) {
	if resp == nil {
		return
	}

	classification, ok := GetServiceClassification(serviceType)
	if ok {
		resp.GrowthType = classification.GrowthType
		logger.Debug().
			Str("service_type", serviceType).
			Str("growth_type", classification.GrowthType.String()).
			Msg("applied growth hint")
	}
}
