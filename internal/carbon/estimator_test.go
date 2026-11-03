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
		want          float64
	}{
		{
			name:          "typical EC2 instance at 50% utilization",
			minWatts:      0.47,
			maxWatts:      1.69,
			vCPUCount:     2,
			utilization:   0.50,
			gridIntensity: 0.000379, // us-east-1
			hours:         730,      // one month
			want:          678.284172,
		},
		{
			name:          "zero utilization (idle)",
			minWatts:      0.47,
			maxWatts:      1.69,
			vCPUCount:     2,
			utilization:   0.0,
			gridIntensity: 0.000379,
			hours:         730,
			want:          295.179223,
		},
		{
			name:          "100% utilization",
			minWatts:      0.47,
			maxWatts:      1.69,
			vCPUCount:     2,
			utilization:   1.0,
			gridIntensity: 0.000379,
			hours:         730,
			want:          1061.389121,
		},
		{
			name:          "low carbon region (eu-north-1)",
			minWatts:      0.47,
			maxWatts:      1.69,
			vCPUCount:     2,
			utilization:   0.50,
			gridIntensity: 0.0000088, // eu-north-1 (Sweden)
			hours:         730,
			want:          15.7490784,
		},
		{
			name:          "high carbon region (ap-south-1)",
			minWatts:      0.47,
			maxWatts:      1.69,
			vCPUCount:     2,
			utilization:   0.50,
			gridIntensity: 0.000708, // ap-south-1 (Mumbai)
			hours:         730,
			want:          1267.084944,
		},
		{
			name:          "zero hours returns zero",
			minWatts:      0.47,
			maxWatts:      1.69,
			vCPUCount:     2,
			utilization:   0.50,
			gridIntensity: 0.000379,
			hours:         0,
			want:          0,
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

			// want is the CLAUDE.md formula evaluated for these inputs:
			// avg = min + util*(max-min); kWh = avg*vCPU*hours/1000;
			// gCO2e = kWh * 1.135 * grid * 1e6.
			assert.InDelta(t, tt.want, got, 1e-4)
		})
	}
}

func TestEstimator_EstimateCarbonGrams_KnownInstance(t *testing.T) {
	e := NewEstimator()

	carbon, ok := e.EstimateCarbonGrams("t3.micro", "us-east-1", 0.5, 730)

	require.True(t, ok, "t3.micro should be found")
	spec, found := GetInstanceSpec("t3.micro")
	require.True(t, found)
	// CCF PkgWatt columns for t3.micro: idle 1.21, 100% 9.96, 2 vCPU.
	assert.Equal(t, 2, spec.VCPUCount)
	assert.InDelta(t, 1.21, spec.MinWatts, 1e-9)
	assert.InDelta(t, 9.96, spec.MaxWatts, 1e-9)
	want := cpuCarbonGrams(spec.MinWatts, spec.MaxWatts, spec.VCPUCount, 0.5, 0.000379, 730)
	assert.InDelta(t, want, carbon, 1e-4)
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
		name          string
		instanceType  string
		wantGPUCarbon bool
		wantVCPU      int
		wantMinWatts  float64
		wantMaxWatts  float64
		wantGPUCount  int
		wantTDP       float64
	}{
		{
			name:          "GPU instance has both CPU and GPU carbon",
			instanceType:  "p4d.24xlarge",
			wantGPUCarbon: true,
			wantVCPU:      96,
			wantMinWatts:  57.93,
			wantMaxWatts:  626.76,
			wantGPUCount:  8,
			wantTDP:       400,
		},
		{
			name:          "non-GPU instance has only CPU carbon",
			instanceType:  "t3.micro",
			wantGPUCarbon: false,
			wantVCPU:      2,
			wantMinWatts:  1.21,
			wantMaxWatts:  9.96,
		},
		{
			name:          "g4dn.xlarge has moderate GPU carbon (1x T4)",
			instanceType:  "g4dn.xlarge",
			wantGPUCarbon: true,
			wantVCPU:      4,
			wantMinWatts:  2.28,
			wantMaxWatts:  18.56,
			wantGPUCount:  1,
			wantTDP:       70,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := NewEstimator()
			cpuCarbon, gpuCarbon, ok := e.EstimateCarbonGramsWithBreakdown(
				tt.instanceType, "us-east-1", 0.5, 730)

			require.True(t, ok, "instance should be found")

			spec, found := GetInstanceSpec(tt.instanceType)
			require.True(t, found)
			assert.Equal(t, tt.wantVCPU, spec.VCPUCount)
			assert.InDelta(t, tt.wantMinWatts, spec.MinWatts, 1e-9)
			assert.InDelta(t, tt.wantMaxWatts, spec.MaxWatts, 1e-9)

			const utilization = 0.5
			const hours = 730.0
			const grid = 0.000379
			wantCPU := cpuCarbonGrams(spec.MinWatts, spec.MaxWatts, spec.VCPUCount, utilization, grid, hours)
			assert.InDelta(t, wantCPU, cpuCarbon, 1e-3)

			if tt.wantGPUCarbon {
				gpu, gpuOK := GetGPUSpec(tt.instanceType)
				require.True(t, gpuOK)
				assert.Equal(t, tt.wantGPUCount, gpu.GPUCount)
				assert.InDelta(t, tt.wantTDP, gpu.TDPPerGPU, 1e-9)
				wantGPU := gpuCarbonGrams(gpu.TDPPerGPU, gpu.GPUCount, utilization, grid, hours)
				assert.InDelta(t, wantGPU, gpuCarbon, 1e-3)
			} else {
				assert.Equal(t, 0.0, gpuCarbon, "non-GPU instance should have zero GPU carbon")
			}
		})
	}
}

// cpuCarbonGrams evaluates the EC2 CPU formula from CLAUDE.md.
// It is copied here so the estimator cannot satisfy the test by calling itself.
func cpuCarbonGrams(minWatts, maxWatts float64, vCPU int, utilization, gridIntensity, hours float64) float64 {
	avgWatts := minWatts + (utilization * (maxWatts - minWatts))
	energyKWh := (avgWatts * float64(vCPU) * hours) / 1000
	return energyKWh * AWSPUE * gridIntensity * 1_000_000
}

// gpuCarbonGrams evaluates GPU watts = TDP * count * utilization, then the
// same PUE and grid conversion as the CPU formula.
func gpuCarbonGrams(tdp float64, count int, utilization, gridIntensity, hours float64) float64 {
	watts := tdp * float64(count) * utilization
	energyKWh := (watts * hours) / 1000
	return energyKWh * AWSPUE * gridIntensity * 1_000_000
}
