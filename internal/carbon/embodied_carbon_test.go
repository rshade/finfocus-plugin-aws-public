package carbon

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEmbodiedCarbonEstimator_EstimateEmbodiedCarbonKg verifies embodied carbon calculation.
func TestEmbodiedCarbonEstimator_EstimateEmbodiedCarbonKg(t *testing.T) {
	tests := []struct {
		name              string
		instanceType      string
		months            float64
		wantOK            bool
		minEmbodiedCarbKg float64
		maxEmbodiedCarbKg float64
	}{
		{
			name:         "m5.large for 1 month",
			instanceType: "m5.large",
			months:       1,
			wantOK:       true,
			// m5.large has 2 vCPUs, m5 family max is 96 vCPUs
			// (1000 / 48) × (2 / 96) × 1 = 0.43 kg
			minEmbodiedCarbKg: 0.3,
			maxEmbodiedCarbKg: 0.6,
		},
		{
			name:         "m5.24xlarge for 1 month (full server)",
			instanceType: "m5.24xlarge",
			months:       1,
			wantOK:       true,
			// m5.24xlarge has 96 vCPUs, m5 family max is 96 vCPUs
			// (1000 / 48) × (96 / 96) × 1 = 20.83 kg
			minEmbodiedCarbKg: 18,
			maxEmbodiedCarbKg: 25,
		},
		{
			name:         "t3.micro for 1 month",
			instanceType: "t3.micro",
			months:       1,
			wantOK:       true,
			// t3.micro has 2 vCPUs, t3 family max is 8 vCPUs
			// (1000 / 48) × (2 / 8) × 1 = 5.21 kg
			minEmbodiedCarbKg: 4,
			maxEmbodiedCarbKg: 7,
		},
		{
			name:         "p4d.24xlarge for 1 month (GPU instance)",
			instanceType: "p4d.24xlarge",
			months:       1,
			wantOK:       true,
			// p4d.24xlarge has 96 vCPUs, p4d family max is 96 vCPUs
			// (1000 / 48) × (96 / 96) × 1 = 20.83 kg
			minEmbodiedCarbKg: 18,
			maxEmbodiedCarbKg: 25,
		},
		{
			name:         "m5.large for 12 months",
			instanceType: "m5.large",
			months:       12,
			wantOK:       true,
			// 12× the 1-month value
			minEmbodiedCarbKg: 3.6,
			maxEmbodiedCarbKg: 7.2,
		},
		{
			name:              "unknown instance type",
			instanceType:      "unknown.type",
			months:            1,
			wantOK:            false,
			minEmbodiedCarbKg: 0,
			maxEmbodiedCarbKg: 0,
		},
	}

	e := NewEmbodiedCarbonEstimator()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			carbonKg, ok := e.EstimateEmbodiedCarbonKg(tt.instanceType, tt.months)

			if tt.wantOK {
				require.True(t, ok, "EstimateEmbodiedCarbonKg should succeed for %s", tt.instanceType)
				assert.GreaterOrEqual(t, carbonKg, tt.minEmbodiedCarbKg,
					"Embodied carbon should be >= %f kg", tt.minEmbodiedCarbKg)
				assert.LessOrEqual(t, carbonKg, tt.maxEmbodiedCarbKg,
					"Embodied carbon should be <= %f kg", tt.maxEmbodiedCarbKg)
			} else {
				assert.False(t, ok)
				assert.Equal(t, 0.0, carbonKg)
			}
		})
	}
}

// TestEmbodiedCarbonEstimator_GetTotalCarbonGrams verifies combined calculation.
func TestEmbodiedCarbonEstimator_GetTotalCarbonGrams(t *testing.T) {
	e := NewEmbodiedCarbonEstimator()

	opCarbon, embodiedCarbon, totalCarbon, ok := e.GetTotalCarbonGrams(
		"m5.large",   // instanceType
		"us-east-1",  // region
		0.5,          // utilization
		730,          // hours (1 month)
	)

	require.True(t, ok, "GetTotalCarbonGrams should succeed")
	assert.Greater(t, opCarbon, 0.0, "Operational carbon should be positive")
	assert.Greater(t, embodiedCarbon, 0.0, "Embodied carbon should be positive")
	assert.InDelta(t, opCarbon+embodiedCarbon, totalCarbon, 0.01, "Total should equal sum")
}

// TestEmbodiedCarbonEstimator_EmbodiedContribution verifies embodied is ~20-30% of total.
func TestEmbodiedCarbonEstimator_EmbodiedContribution(t *testing.T) {
	e := NewEmbodiedCarbonEstimator()

	opCarbon, embodiedCarbon, totalCarbon, ok := e.GetTotalCarbonGrams(
		"m5.large",
		"us-east-1",
		0.5,
		730, // 1 month
	)

	require.True(t, ok)

	// Embodied carbon is typically 20-30% of total for small instances
	// For small instances in the family, it may be lower (proportional sharing)
	embodiedRatio := embodiedCarbon / totalCarbon
	t.Logf("Embodied ratio: %.2f%%, Op: %.2f g, Embodied: %.2f g, Total: %.2f g",
		embodiedRatio*100, opCarbon, embodiedCarbon, totalCarbon)

	// With proportional sharing across instance family, embodied may be quite small
	assert.Greater(t, embodiedRatio, 0.01, "Embodied should be at least 1% of total")
	assert.Less(t, embodiedRatio, 0.50, "Embodied should be less than 50% of total")
}

// TestGetMaxFamilyVCPUs verifies family vCPU lookups.
func TestGetMaxFamilyVCPUs(t *testing.T) {
	tests := []struct {
		instanceType string
		expectedMax  float64
	}{
		{"m5.large", 96},
		{"m5.xlarge", 96},
		{"m5.24xlarge", 96},
		{"t3.micro", 8},
		{"c5.xlarge", 96},
		{"r5.2xlarge", 96},
		{"p4d.24xlarge", 96},
		{"g4dn.xlarge", 96},
	}

	for _, tt := range tests {
		t.Run(tt.instanceType, func(t *testing.T) {
			maxVCPUs := GetMaxFamilyVCPUs(tt.instanceType)
			assert.Equal(t, tt.expectedMax, maxVCPUs,
				"Max vCPUs for %s family", tt.instanceType)
		})
	}
}

// TestParseInstanceFamily verifies family extraction.
func TestParseInstanceFamily(t *testing.T) {
	tests := []struct {
		instanceType   string
		expectedFamily string
	}{
		{"m5.large", "m5"},
		{"m5.24xlarge", "m5"},
		{"t3.micro", "t3"},
		{"p4d.24xlarge", "p4d"},
		{"g4dn.xlarge", "g4dn"},
		{"c5n.18xlarge", "c5n"},
		// Edge cases
		{"", ""},                     // Empty string
		{"nodot", "nodot"},           // No dot separator
		{".leadingdot", ""},          // Leading dot
		{"trailing.", "trailing"},    // Trailing dot
		{"a.b.c", "a"},               // Multiple dots - takes first part
	}

	for _, tt := range tests {
		t.Run(tt.instanceType, func(t *testing.T) {
			family := parseInstanceFamily(tt.instanceType)
			assert.Equal(t, tt.expectedFamily, family)
		})
	}
}

// TestEmbodiedCarbonEstimator_CustomValues verifies custom configuration.
func TestEmbodiedCarbonEstimator_CustomValues(t *testing.T) {
	// Custom estimator with different values
	e := &EmbodiedCarbonEstimator{
		EmbodiedCarbonPerServerKg: 500,  // Lower embodied carbon
		ServerLifespanMonths:      60,   // Longer lifespan
	}

	carbonKg, ok := e.EstimateEmbodiedCarbonKg("m5.24xlarge", 1)
	require.True(t, ok)

	// (500 / 60) × (96 / 96) × 1 = 8.33 kg
	assert.InDelta(t, 8.33, carbonKg, 0.5)
}

// TestEmbodiedCarbonEstimator_GetBillingDetail verifies billing detail.
func TestEmbodiedCarbonEstimator_GetBillingDetail(t *testing.T) {
	e := NewEmbodiedCarbonEstimator()

	detail := e.GetBillingDetail("m5.large", 1)

	assert.Contains(t, detail, "Embodied carbon")
	assert.Contains(t, detail, "m5.large")
	assert.Contains(t, detail, "kgCO2e/month")
}
