package carbon

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCalculateCarbonGrams(t *testing.T) {
	tests := []struct {
		name          string
		minWatts      float64
		maxWatts      float64
		vCPUCount     int
		utilization   float64
		gridIntensity float64
		hours         float64
		wantMin       float64
		wantMax       float64
	}{
		{
			name:          "typical EC2 instance at 50% utilization",
			minWatts:      0.47,  // t3.micro idle
			maxWatts:      1.69,  // t3.micro 100%
			vCPUCount:     2,
			utilization:   0.50,
			gridIntensity: 0.000379, // us-east-1
			hours:         730,      // one month
			wantMin:       100,      // expect ~831 gCO2e
			wantMax:       2000,
		},
		{
			name:          "zero utilization (idle)",
			minWatts:      0.47,
			maxWatts:      1.69,
			vCPUCount:     2,
			utilization:   0.0,
			gridIntensity: 0.000379,
			hours:         730,
			wantMin:       50,
			wantMax:       500,
		},
		{
			name:          "100% utilization",
			minWatts:      0.47,
			maxWatts:      1.69,
			vCPUCount:     2,
			utilization:   1.0,
			gridIntensity: 0.000379,
			hours:         730,
			wantMin:       500,
			wantMax:       3000,
		},
		{
			name:          "low carbon region (eu-north-1)",
			minWatts:      0.47,
			maxWatts:      1.69,
			vCPUCount:     2,
			utilization:   0.50,
			gridIntensity: 0.0000088, // eu-north-1 (Sweden)
			hours:         730,
			wantMin:       1,
			wantMax:       100, // Much lower due to clean grid
		},
		{
			name:          "high carbon region (ap-south-1)",
			minWatts:      0.47,
			maxWatts:      1.69,
			vCPUCount:     2,
			utilization:   0.50,
			gridIntensity: 0.000708, // ap-south-1 (Mumbai)
			hours:         730,
			wantMin:       500,
			wantMax:       3000, // Higher due to coal-heavy grid
		},
		{
			name:          "zero hours returns zero",
			minWatts:      0.47,
			maxWatts:      1.69,
			vCPUCount:     2,
			utilization:   0.50,
			gridIntensity: 0.000379,
			hours:         0,
			wantMin:       0,
			wantMax:       0.001,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateCarbonGrams(
				tt.minWatts,
				tt.maxWatts,
				tt.vCPUCount,
				tt.utilization,
				tt.gridIntensity,
				tt.hours,
			)

			assert.GreaterOrEqual(t, got, tt.wantMin, "carbon should be >= %f", tt.wantMin)
			assert.LessOrEqual(t, got, tt.wantMax, "carbon should be <= %f", tt.wantMax)
		})
	}
}

func TestEstimator_EstimateCarbonGrams_KnownInstance(t *testing.T) {
	e := NewEstimator()

	carbon, ok := e.EstimateCarbonGrams("t3.micro", "us-east-1", 0.5, 730)

	require.True(t, ok, "t3.micro should be found")
	assert.Greater(t, carbon, 0.0, "carbon should be positive")
	// Expect approximately 500-5000 gCO2e for t3.micro monthly
	// (based on actual CCF data power values)
	assert.Greater(t, carbon, 500.0)
	assert.Less(t, carbon, 5000.0)
}

func TestEstimator_EstimateCarbonGrams_UnknownInstance(t *testing.T) {
	e := NewEstimator()

	carbon, ok := e.EstimateCarbonGrams("nonexistent.type", "us-east-1", 0.5, 730)

	assert.False(t, ok)
	assert.Equal(t, 0.0, carbon)
}

func TestEstimator_EstimateCarbonGrams_RegionAffectsResult(t *testing.T) {
	e := NewEstimator()

	// Same instance, different regions
	carbonUSEast, ok1 := e.EstimateCarbonGrams("t3.micro", "us-east-1", 0.5, 730)
	carbonEUNorth, ok2 := e.EstimateCarbonGrams("t3.micro", "eu-north-1", 0.5, 730)

	require.True(t, ok1)
	require.True(t, ok2)

	// EU North (Sweden) should have much lower carbon than US East (Virginia)
	// EU North grid factor is 0.0000088, US East is 0.000379 (~43x difference)
	assert.Greater(t, carbonUSEast, carbonEUNorth*10,
		"us-east-1 should have at least 10x more carbon than eu-north-1")
}

func TestEstimator_EstimateCarbonGrams_UtilizationAffectsResult(t *testing.T) {
	e := NewEstimator()

	carbonLow, ok1 := e.EstimateCarbonGrams("t3.micro", "us-east-1", 0.2, 730)
	carbonHigh, ok2 := e.EstimateCarbonGrams("t3.micro", "us-east-1", 0.8, 730)

	require.True(t, ok1)
	require.True(t, ok2)

	// Higher utilization should result in higher carbon
	assert.Greater(t, carbonHigh, carbonLow,
		"80%% utilization should have more carbon than 20%%")
}

func TestEstimator_EstimateCarbonGrams_UnknownRegion(t *testing.T) {
	e := NewEstimator()

	// Unknown region should use default grid factor
	carbon, ok := e.EstimateCarbonGrams("t3.micro", "unknown-region", 0.5, 730)

	require.True(t, ok, "calculation should succeed even for unknown region")
	assert.Greater(t, carbon, 0.0, "carbon should be positive")
}

func TestGetGridFactor(t *testing.T) {
	tests := []struct {
		region     string
		wantFactor float64
	}{
		{"us-east-1", 0.000379},
		{"eu-north-1", 0.0000088},
		{"ap-south-1", 0.000708},
		{"unknown-region", DefaultGridFactor},
	}

	for _, tt := range tests {
		t.Run(tt.region, func(t *testing.T) {
			got := GetGridFactor(tt.region)
			assert.Equal(t, tt.wantFactor, got)
		})
	}
}

// TestEstimator_GPU_IncludesGPUPower verifies that GPU instances have higher carbon
// due to GPU power consumption.
func TestEstimator_GPU_IncludesGPUPower(t *testing.T) {
	e := NewEstimator()
	e.IncludeGPU = true

	// p4d.24xlarge has 8x A100 GPUs (400W each = 3200W total)
	// This should significantly increase carbon compared to CPU alone
	carbon, ok := e.EstimateCarbonGrams("p4d.24xlarge", "us-east-1", 0.5, 730)

	require.True(t, ok, "p4d.24xlarge should be found")

	// GPU power alone: 8 × 400W × 0.5 = 1600W
	// Over 730 hours: 1600 × 730 / 1000 = 1168 kWh × 1.135 PUE = 1325.68 kWh
	// Carbon: 1325.68 × 0.000379 × 1,000,000 = ~502,394 gCO2e from GPU alone
	// Total should be CPU + GPU, expect > 500,000 gCO2e
	assert.Greater(t, carbon, 100000.0, "GPU instance should have high carbon due to GPU power")
}

// TestEstimator_GPU_CanBeDisabled verifies that GPU power can be excluded.
func TestEstimator_GPU_CanBeDisabled(t *testing.T) {
	eWithGPU := NewEstimator()
	eWithGPU.IncludeGPU = true

	eWithoutGPU := NewEstimator()
	eWithoutGPU.IncludeGPU = false

	carbonWithGPU, ok1 := eWithGPU.EstimateCarbonGrams("p4d.24xlarge", "us-east-1", 0.5, 730)
	carbonWithoutGPU, ok2 := eWithoutGPU.EstimateCarbonGrams("p4d.24xlarge", "us-east-1", 0.5, 730)

	require.True(t, ok1)
	require.True(t, ok2)

	// With GPU should be higher than without (GPU adds power consumption)
	// For p4d.24xlarge: 96 vCPUs have high CPU power (~10.3M gCO2e)
	// 8x A100 @ 400W adds 3200W, which at 50% util = 1600W × 730h = ~502k gCO2e (about 5% of total)
	assert.Greater(t, carbonWithGPU, carbonWithoutGPU,
		"Carbon with GPU should be higher than without")

	// Verify the difference is meaningful (GPU adds at least 3% more carbon)
	gpuContribution := carbonWithGPU - carbonWithoutGPU
	assert.Greater(t, gpuContribution, carbonWithoutGPU*0.03,
		"GPU should contribute at least 3%% of total carbon for GPU instance")
}

// TestEstimator_GPU_NonGPUInstanceUnaffected verifies that non-GPU instances
// are unaffected by the IncludeGPU setting.
func TestEstimator_GPU_NonGPUInstanceUnaffected(t *testing.T) {
	eWithGPU := NewEstimator()
	eWithGPU.IncludeGPU = true

	eWithoutGPU := NewEstimator()
	eWithoutGPU.IncludeGPU = false

	carbonWithGPU, ok1 := eWithGPU.EstimateCarbonGrams("t3.micro", "us-east-1", 0.5, 730)
	carbonWithoutGPU, ok2 := eWithoutGPU.EstimateCarbonGrams("t3.micro", "us-east-1", 0.5, 730)

	require.True(t, ok1)
	require.True(t, ok2)

	// Non-GPU instance should have same carbon regardless of IncludeGPU setting
	assert.Equal(t, carbonWithGPU, carbonWithoutGPU,
		"Non-GPU instance should have same carbon regardless of IncludeGPU")
}

// TestEstimator_EstimateCarbonGramsWithBreakdown verifies the breakdown method.
func TestEstimator_EstimateCarbonGramsWithBreakdown(t *testing.T) {
	tests := []struct {
		name            string
		instanceType    string
		wantGPUCarbon   bool
		minTotalCarbon  float64
		maxTotalCarbon  float64
	}{
		{
			name:            "GPU instance has both CPU and GPU carbon",
			instanceType:    "p4d.24xlarge",
			wantGPUCarbon:   true,
			minTotalCarbon:  1000000.0,  // p4d.24xlarge has 96 vCPUs + 8x A100 GPUs
			maxTotalCarbon:  20000000.0, // High due to 96 vCPUs + 3200W GPU power
		},
		{
			name:            "non-GPU instance has only CPU carbon",
			instanceType:    "t3.micro",
			wantGPUCarbon:   false,
			minTotalCarbon:  100.0,
			maxTotalCarbon:  5000.0,
		},
		{
			name:            "g4dn.xlarge has moderate GPU carbon (1x T4)",
			instanceType:    "g4dn.xlarge",
			wantGPUCarbon:   true,
			minTotalCarbon:  10000.0,
			maxTotalCarbon:  500000.0, // g4dn.xlarge has 4 vCPUs + 1 T4 GPU (70W)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := NewEstimator()
			cpuCarbon, gpuCarbon, ok := e.EstimateCarbonGramsWithBreakdown(
				tt.instanceType, "us-east-1", 0.5, 730)

			require.True(t, ok, "instance should be found")

			// Verify CPU carbon is always positive
			assert.Greater(t, cpuCarbon, 0.0, "CPU carbon should be positive")

			// Verify GPU carbon expectation
			if tt.wantGPUCarbon {
				assert.Greater(t, gpuCarbon, 0.0, "GPU carbon should be positive")
			} else {
				assert.Equal(t, 0.0, gpuCarbon, "non-GPU instance should have zero GPU carbon")
			}

			// Verify total range
			totalCarbon := cpuCarbon + gpuCarbon
			assert.GreaterOrEqual(t, totalCarbon, tt.minTotalCarbon)
			assert.LessOrEqual(t, totalCarbon, tt.maxTotalCarbon)
		})
	}
}
