package carbon

// GridEmissionFactors maps AWS region codes to grid carbon intensity.
// Values are in metric tons CO2eq per kWh.
// Source: Cloud Carbon Footprint methodology.
var GridEmissionFactors = map[string]float64{
	"us-east-1":      0.000379,    // Virginia (SERC)
	"us-east-2":      0.000411,    // Ohio (RFC)
	"us-west-1":      0.000322,    // N. California (WECC)
	"us-west-2":      0.000322,    // Oregon (WECC)
	"ca-central-1":   0.00012,     // Canada
	"eu-west-1":      0.0002786,   // Ireland
	"eu-north-1":     0.0000088,   // Sweden (very low carbon)
	"ap-southeast-1": 0.000408,    // Singapore
	"ap-southeast-2": 0.00079,     // Sydney
	"ap-northeast-1": 0.000506,    // Tokyo
	"ap-south-1":     0.000708,    // Mumbai
	"sa-east-1":      0.0000617,   // São Paulo (very low carbon)
}

// DefaultGridFactor is used when a region doesn't have a specific factor.
// This is the global average from CCF.
const DefaultGridFactor = 0.00039278

// GetGridFactor returns the grid emission factor for a region.
// Falls back to DefaultGridFactor if the region is not found.
func GetGridFactor(region string) float64 {
	if factor, ok := GridEmissionFactors[region]; ok {
		return factor
	}
	return DefaultGridFactor
}
