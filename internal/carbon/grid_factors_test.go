package carbon

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestGridEmissionFactors_AllWithinValidRange validates that all grid emission factors
// fall within the physically reasonable range of 0.0 to 2.0 metric tons CO2e per kWh.
//
// This test ensures that:
// 1. No grid factor is negative (impossible)
// 2. No grid factor exceeds 2.0 metric tons CO2e/kWh (no grid is this carbon-intensive)
// 3. Grid factors are in the expected unit (metric tons, not kg or grams)
//
// The upper bound of 2.0 is based on the most carbon-intensive grids globally,
// which are coal-heavy grids in regions like China, India, and parts of Australia.
// Even these rarely exceed 1.0 metric tons CO2e/kWh.
func TestGridEmissionFactors_AllWithinValidRange(t *testing.T) {
	const minValidFactor = 0.0
	const maxValidFactor = 2.0

	for region, factor := range GridEmissionFactors {
		t.Run(region, func(t *testing.T) {
			assert.GreaterOrEqual(t, factor, minValidFactor,
				"Grid factor for %s should be >= 0 (got %f)", region, factor)
			assert.LessOrEqual(t, factor, maxValidFactor,
				"Grid factor for %s should be <= 2.0 metric tons CO2e/kWh (got %f)", region, factor)
		})
	}
}

// TestGridEmissionFactors_DefaultWithinValidRange validates that the default grid factor
// (used for unknown regions) is within a reasonable range.
func TestGridEmissionFactors_DefaultWithinValidRange(t *testing.T) {
	const minValidFactor = 0.0
	const maxValidFactor = 2.0

	assert.GreaterOrEqual(t, DefaultGridFactor, minValidFactor,
		"Default grid factor should be >= 0")
	assert.LessOrEqual(t, DefaultGridFactor, maxValidFactor,
		"Default grid factor should be <= 2.0 metric tons CO2e/kWh")

	// Default should also be a reasonable global average (roughly 0.0003-0.0006)
	assert.GreaterOrEqual(t, DefaultGridFactor, 0.0001,
		"Default grid factor should be at least 0.0001 (100 gCO2e/kWh)")
	assert.LessOrEqual(t, DefaultGridFactor, 0.001,
		"Default grid factor should be at most 0.001 (1000 gCO2e/kWh)")
}

// TestGridEmissionFactors_ExpectedRegionsPresent validates that all major AWS regions
// have grid emission factors defined.
func TestGridEmissionFactors_ExpectedRegionsPresent(t *testing.T) {
	expectedRegions := []struct {
		region      string
		description string
	}{
		{"us-east-1", "Virginia"},
		{"us-east-2", "Ohio"},
		{"us-west-1", "N. California"},
		{"us-west-2", "Oregon"},
		{"ca-central-1", "Canada"},
		{"eu-west-1", "Ireland"},
		{"eu-north-1", "Sweden"},
		{"ap-southeast-1", "Singapore"},
		{"ap-southeast-2", "Sydney"},
		{"ap-northeast-1", "Tokyo"},
		{"ap-south-1", "Mumbai"},
		{"sa-east-1", "São Paulo"},
	}

	for _, expected := range expectedRegions {
		t.Run(expected.region, func(t *testing.T) {
			_, exists := GridEmissionFactors[expected.region]
			assert.True(t, exists,
				"Grid factor should exist for %s (%s)", expected.region, expected.description)
		})
	}
}

// TestGridEmissionFactors_RegionalVariation validates that grid factors reflect
// real-world differences between clean and dirty grids.
//
// This test ensures the data hasn't been accidentally corrupted or normalized.
func TestGridEmissionFactors_RegionalVariation(t *testing.T) {
	// Sweden (eu-north-1) should have very low carbon grid (hydroelectric)
	swedenFactor := GridEmissionFactors["eu-north-1"]
	assert.Less(t, swedenFactor, 0.0001,
		"Sweden (eu-north-1) should have very low carbon grid")

	// Brazil (sa-east-1) should also have low carbon grid (hydroelectric)
	brazilFactor := GridEmissionFactors["sa-east-1"]
	assert.Less(t, brazilFactor, 0.0002,
		"Brazil (sa-east-1) should have low carbon grid")

	// India (ap-south-1) should have higher carbon grid (coal-heavy)
	indiaFactor := GridEmissionFactors["ap-south-1"]
	assert.Greater(t, indiaFactor, 0.0005,
		"India (ap-south-1) should have higher carbon grid")

	// Verify the expected ratio between clean and dirty grids
	// India should be at least 10x more carbon-intensive than Sweden
	ratio := indiaFactor / swedenFactor
	assert.Greater(t, ratio, 10.0,
		"India should be at least 10x more carbon-intensive than Sweden (ratio: %.1f)", ratio)
}

// TestGridEmissionFactors_UnitsAreMetricTons validates that grid factors are in
// metric tons CO2e per kWh (not kg or grams).
//
// This is important because the CCF methodology uses metric tons, and mixing
// units would cause 1000x or 1000000x errors in carbon estimation.
func TestGridEmissionFactors_UnitsAreMetricTons(t *testing.T) {
	// If factors were in kg instead of metric tons, they would be 1000x larger
	// If factors were in grams instead of metric tons, they would be 1,000,000x larger
	// Check that values are in the expected range for metric tons

	for region, factor := range GridEmissionFactors {
		t.Run(region, func(t *testing.T) {
			// Metric ton factors should be in range 0.00001 to 0.002
			// kg factors would be 0.01 to 2.0
			// gram factors would be 10 to 2000
			assert.Less(t, factor, 0.01,
				"Grid factor for %s appears to be in kg, not metric tons", region)
			assert.Greater(t, factor, 0.000001,
				"Grid factor for %s appears unrealistically low", region)
		})
	}
}

// TestGetGridFactor_USWest1 validates that us-west-1 returns the WECC grid factor.
// FR-004: System MUST use the correct grid emission factor (CAISO/WECC) for carbon estimation in us-west-1.
// The WECC (Western Electricity Coordinating Council) factor of 0.000322 metric tons CO2e/kWh
// is used for both us-west-1 (N. California) and us-west-2 (Oregon) as they share the same grid.
func TestGetGridFactor_USWest1(t *testing.T) {
	expectedWECC := 0.000322

	usWest1Factor := GetGridFactor("us-west-1")
	assert.Equal(t, expectedWECC, usWest1Factor,
		"us-west-1 should use WECC grid factor")

	usWest2Factor := GetGridFactor("us-west-2")
	assert.Equal(t, expectedWECC, usWest2Factor,
		"us-west-2 should use WECC grid factor")

	// Both western US regions should have the same factor (same WECC grid)
	assert.Equal(t, usWest1Factor, usWest2Factor,
		"us-west-1 and us-west-2 should have the same WECC grid factor")
}

// TestGetGridFactor_KnownRegions validates that GetGridFactor returns expected values
// for known regions.
func TestGetGridFactor_KnownRegions(t *testing.T) {
	tests := []struct {
		region         string
		expectedFactor float64
	}{
		{"us-east-1", 0.000379},
		{"us-west-1", 0.000322}, // WECC grid
		{"us-west-2", 0.000322}, // WECC grid
		{"eu-north-1", 0.0000088},
		{"ap-south-1", 0.000708},
		{"ca-central-1", 0.00012},
		{"sa-east-1", 0.0000617},
	}

	for _, tt := range tests {
		t.Run(tt.region, func(t *testing.T) {
			got := GetGridFactor(tt.region)
			assert.Equal(t, tt.expectedFactor, got,
				"GetGridFactor(%s) should return expected factor", tt.region)
		})
	}
}

// TestGetGridFactor_UnknownRegion validates that GetGridFactor returns the default
// factor for unknown regions.
func TestGetGridFactor_UnknownRegion(t *testing.T) {
	unknownRegions := []string{
		"unknown-region",
		"us-gov-west-1", // GovCloud not in our list
		"me-central-1",  // Newer region not yet added
		"",              // Empty string
	}

	for _, region := range unknownRegions {
		t.Run(region, func(t *testing.T) {
			got := GetGridFactor(region)
			assert.Equal(t, DefaultGridFactor, got,
				"GetGridFactor(%q) should return DefaultGridFactor", region)
		})
	}
}

// TestGridEmissionFactors_NoDuplicates validates that no region appears twice
// with different values (would indicate a map initialization bug).
func TestGridEmissionFactors_NoDuplicates(t *testing.T) {
	// This test is implicit in Go maps, but we validate the count matches
	// what we expect to ensure no regions were accidentally removed
	const minimumExpectedRegions = 10

	assert.GreaterOrEqual(t, len(GridEmissionFactors), minimumExpectedRegions,
		"Should have at least %d regions defined", minimumExpectedRegions)
}

// TestGridEmissionFactors_Precision validates that grid factors have sufficient
// precision for accurate carbon calculations.
//
// Grid factors should have at least 5 decimal places to avoid rounding errors
// that would compound in carbon calculations.
func TestGridEmissionFactors_Precision(t *testing.T) {
	for region, factor := range GridEmissionFactors {
		t.Run(region, func(t *testing.T) {
			// Format to 10 decimal places and count significant decimal places
			// (digits after decimal point, excluding trailing zeros)
			formatted := fmt.Sprintf("%.10f", factor)
			parts := strings.Split(formatted, ".")
			assert.Len(t, parts, 2, "Factor should be a decimal number")

			decimalPart := strings.TrimRight(parts[1], "0")
			assert.GreaterOrEqual(t, len(decimalPart), 5,
				"Grid factor for %s should have at least 5 decimal places (got %.10f)", region, factor)
		})
	}
}
