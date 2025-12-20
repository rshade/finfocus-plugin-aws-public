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
