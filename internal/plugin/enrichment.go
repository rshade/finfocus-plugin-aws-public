package plugin

import (
	"github.com/rs/zerolog"
	pbc "github.com/rshade/pulumicost-spec/sdk/go/proto/pulumicost/v1"
)

// hasUsageProfile checks if UsageProfile field exists in request (feature detection)
func hasUsageProfile(req *pbc.ResourceDescriptor) bool { //nolint:unused
	// UsageProfile does not exist yet in current pulumicost-spec version
	return false
}

// hasGrowthHint checks if GrowthType field exists in response (feature detection)
func hasGrowthHint(resp *pbc.GetProjectedCostResponse) bool {
	// GrowthType exists in pulumicost-spec v0.4.12+
	// This function always returns true because the GrowthType field is available
	// in the current pulumicost-spec version used by this plugin.
	return true
}

// hasLineage checks if Lineage field exists in response (feature detection)
func hasLineage(resp *pbc.GetProjectedCostResponse) bool { //nolint:unused
	// CostAllocationLineage does not exist yet in current pulumicost-spec version
	return false
}

// setGrowthHint sets the growth_type field based on service classification
func setGrowthHint(logger zerolog.Logger, serviceType string, resp *pbc.GetProjectedCostResponse) {
	if resp == nil {
		return
	}

	if !hasGrowthHint(resp) {
		return // Field not available in this spec version
	}

	classification, ok := serviceClassifications[serviceType]
	if ok {
		resp.GrowthType = classification.GrowthType
		logger.Debug().
			Str("service_type", serviceType).
			Str("growth_type", classification.GrowthType.String()).
			Msg("applied growth hint")
	}
}
