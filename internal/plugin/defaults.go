package plugin

import "strings"

// DefaultKind classifies the impact of a defaulted value on estimate quality.
type DefaultKind int

const (
	// KindConfig indicates a configuration default that still produces a non-zero
	// cost component (e.g., EBS size=8GB, RDS engine=mysql). The estimate is
	// usable but may not match the user's actual configuration.
	KindConfig DefaultKind = iota

	// KindUsageZero indicates a usage default that produces a $0 cost component
	// (e.g., Lambda requests=0, DynamoDB RCU=0). The estimate is likely inaccurate
	// because the user's actual usage is unknown.
	KindUsageZero
)

const (
	// metadataKeyDefaultsApplied is the canonical metadata key listing defaulted fields.
	metadataKeyDefaultsApplied = "defaults_applied"

	// metadataKeyEstimateQuality is the canonical metadata key for estimate quality.
	metadataKeyEstimateQuality = "estimate_quality"

	// qualityHigh indicates all values were explicit — no defaults applied.
	qualityHigh = "high"

	// qualityMedium indicates only configuration defaults (non-zero cost).
	qualityMedium = "medium"

	// qualityLow indicates usage-zero defaults ($0 cost component).
	qualityLow = "low"
)

// trackedDefault records a single defaulted field.
type trackedDefault struct {
	name  string
	value string
}

// DefaultsTracker accumulates information about which fields were defaulted
// during cost estimation. It produces metadata for the GetProjectedCostResponse
// enabling downstream consumers to distinguish "real $0" from "defaulted $0".
type DefaultsTracker struct {
	defaults     []trackedDefault
	hasUsageZero bool
}

// Add records a defaulted field with its default value and impact kind.
// KindUsageZero sets a flag that downgrades quality to "low".
func (dt *DefaultsTracker) Add(name, value string, kind DefaultKind) {
	dt.defaults = append(dt.defaults, trackedDefault{name: name, value: value})
	if kind == KindUsageZero {
		dt.hasUsageZero = true
	}
}

// Quality returns the estimate quality classification:
//   - "high": No defaults applied (all values explicit)
//   - "medium": Only KindConfig defaults (non-zero cost, wrong config possible)
//   - "low": Any KindUsageZero default (produces $0 cost component)
func (dt *DefaultsTracker) Quality() string {
	if len(dt.defaults) == 0 {
		return qualityHigh
	}
	if dt.hasUsageZero {
		return qualityLow
	}
	return qualityMedium
}

// Metadata returns the metadata map for GetProjectedCostResponse, or nil
// if no defaults were applied (backward compatible — no metadata overhead
// when all values are explicit).
func (dt *DefaultsTracker) Metadata() map[string]string {
	if len(dt.defaults) == 0 {
		return nil
	}

	pairs := make([]string, 0, len(dt.defaults))
	for _, d := range dt.defaults {
		pairs = append(pairs, d.name+"="+d.value)
	}

	return map[string]string{
		metadataKeyDefaultsApplied: strings.Join(pairs, ","),
		metadataKeyEstimateQuality: dt.Quality(),
	}
}
