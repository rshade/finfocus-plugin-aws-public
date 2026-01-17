package carbon

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEBSEstimator_EstimateCarbonGrams verifies EBS carbon estimation.
func TestEBSEstimator_EstimateCarbonGrams(t *testing.T) {
	tests := []struct {
		name           string
		config         EBSVolumeConfig
		wantOK         bool
		minCarbonGrams float64
		maxCarbonGrams float64
	}{
		{
			name: "100GB gp3 SSD for 1 month in us-east-1",
			config: EBSVolumeConfig{
				VolumeType: "gp3",
				SizeGB:     100,
				Region:     "us-east-1",
				Hours:      HoursPerMonth,
			},
			wantOK:         true,
			minCarbonGrams: 50,  // ~73 gCO2e expected
			maxCarbonGrams: 150,
		},
		{
			name: "1TB gp3 SSD for 1 month in us-east-1",
			config: EBSVolumeConfig{
				VolumeType: "gp3",
				SizeGB:     1024,
				Region:     "us-east-1",
				Hours:      HoursPerMonth,
			},
			wantOK:         true,
			minCarbonGrams: 500,  // ~754 gCO2e expected
			maxCarbonGrams: 1500,
		},
		{
			name: "100GB io2 provisioned IOPS SSD",
			config: EBSVolumeConfig{
				VolumeType: "io2",
				SizeGB:     100,
				Region:     "us-east-1",
				Hours:      HoursPerMonth,
			},
			wantOK:         true,
			minCarbonGrams: 50,
			maxCarbonGrams: 150,
		},
		{
			name: "500GB st1 throughput HDD",
			config: EBSVolumeConfig{
				VolumeType: "st1",
				SizeGB:     500,
				Region:     "us-east-1",
				Hours:      HoursPerMonth,
			},
			wantOK:         true,
			minCarbonGrams: 100, // Lower due to HDD coefficient (0.65 vs 1.2)
			maxCarbonGrams: 400,
		},
		{
			name: "100GB in low-carbon region (eu-north-1)",
			config: EBSVolumeConfig{
				VolumeType: "gp3",
				SizeGB:     100,
				Region:     "eu-north-1",
				Hours:      HoursPerMonth,
			},
			wantOK:         true,
			minCarbonGrams: 0.1, // Very low due to clean grid
			maxCarbonGrams: 5,
		},
		{
			name: "100GB in high-carbon region (ap-south-1)",
			config: EBSVolumeConfig{
				VolumeType: "gp3",
				SizeGB:     100,
				Region:     "ap-south-1",
				Hours:      HoursPerMonth,
			},
			wantOK:         true,
			minCarbonGrams: 100, // Higher due to coal-heavy grid
			maxCarbonGrams: 300,
		},
		{
			name: "unknown volume type",
			config: EBSVolumeConfig{
				VolumeType: "unknown",
				SizeGB:     100,
				Region:     "us-east-1",
				Hours:      HoursPerMonth,
			},
			wantOK:         false,
			minCarbonGrams: 0,
			maxCarbonGrams: 0,
		},
		{
			name: "zero size returns zero carbon",
			config: EBSVolumeConfig{
				VolumeType: "gp3",
				SizeGB:     0,
				Region:     "us-east-1",
				Hours:      HoursPerMonth,
			},
			wantOK:         true,
			minCarbonGrams: 0,
			maxCarbonGrams: 0.001,
		},
		{
			name: "zero hours returns zero carbon",
			config: EBSVolumeConfig{
				VolumeType: "gp3",
				SizeGB:     100,
				Region:     "us-east-1",
				Hours:      0,
			},
			wantOK:         true,
			minCarbonGrams: 0,
			maxCarbonGrams: 0.001,
		},
	}

	e := NewEBSEstimator()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			carbon, ok := e.EstimateCarbonGrams(tt.config)

			if tt.wantOK {
				require.True(t, ok, "EstimateCarbonGrams should succeed")
				assert.GreaterOrEqual(t, carbon, tt.minCarbonGrams)
				assert.LessOrEqual(t, carbon, tt.maxCarbonGrams)
			} else {
				assert.False(t, ok, "EstimateCarbonGrams should fail for unknown type")
				assert.Equal(t, 0.0, carbon)
			}
		})
	}
}

// TestEBSEstimator_EstimateCarbonGramsSimple verifies the convenience method.
func TestEBSEstimator_EstimateCarbonGramsSimple(t *testing.T) {
	e := NewEBSEstimator()

	carbon, ok := e.EstimateCarbonGramsSimple("gp3", 100, "us-east-1", 730)

	require.True(t, ok)
	assert.Greater(t, carbon, 0.0)
}

// TestEBSEstimator_VolumeTypeComparison verifies HDD uses less energy than SSD.
func TestEBSEstimator_VolumeTypeComparison(t *testing.T) {
	e := NewEBSEstimator()

	// Same size, same region, same hours - but different volume types
	carbonSSD, ok1 := e.EstimateCarbonGramsSimple("gp3", 100, "us-east-1", 730)
	carbonHDD, ok2 := e.EstimateCarbonGramsSimple("st1", 100, "us-east-1", 730)

	require.True(t, ok1)
	require.True(t, ok2)

	// SSD has higher power coefficient (1.2 vs 0.65), so higher carbon
	assert.Greater(t, carbonSSD, carbonHDD,
		"SSD (gp3) should have higher carbon than HDD (st1)")

	// The ratio should be approximately 1.2/0.65 ≈ 1.85
	ratio := carbonSSD / carbonHDD
	assert.Greater(t, ratio, 1.5, "SSD/HDD ratio should be > 1.5")
	assert.Less(t, ratio, 2.2, "SSD/HDD ratio should be < 2.2")
}

// TestEBSEstimator_RegionImpact verifies regional grid factors affect carbon.
func TestEBSEstimator_RegionImpact(t *testing.T) {
	e := NewEBSEstimator()

	carbonUSEast, ok1 := e.EstimateCarbonGramsSimple("gp3", 100, "us-east-1", 730)
	carbonEUNorth, ok2 := e.EstimateCarbonGramsSimple("gp3", 100, "eu-north-1", 730)

	require.True(t, ok1)
	require.True(t, ok2)

	// US East has ~43× higher grid factor than EU North
	// 0.000379 / 0.0000088 ≈ 43
	assert.Greater(t, carbonUSEast, carbonEUNorth*10,
		"US East should have at least 10× more carbon than EU North")
}

// TestEBSEstimator_GetBillingDetail verifies billing detail generation.
func TestEBSEstimator_GetBillingDetail(t *testing.T) {
	e := NewEBSEstimator()

	tests := []struct {
		config      EBSVolumeConfig
		wantContains []string
	}{
		{
			config: EBSVolumeConfig{
				VolumeType: "gp3",
				SizeGB:     100,
				Region:     "us-east-1",
				Hours:      HoursPerMonth,
			},
			wantContains: []string{"EBS", "gp3", "SSD", "100", "730"},
		},
		{
			config: EBSVolumeConfig{
				VolumeType: "st1",
				SizeGB:     500,
				Region:     "us-east-1",
				Hours:      HoursPerMonth,
			},
			wantContains: []string{"EBS", "st1", "HDD", "500"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.config.VolumeType, func(t *testing.T) {
			detail := e.GetBillingDetail(tt.config)
			for _, want := range tt.wantContains {
				assert.Contains(t, detail, want)
			}
		})
	}
}
