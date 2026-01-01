package carbon

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLambdaEstimator_EstimateCarbonGrams verifies Lambda carbon estimation.
func TestLambdaEstimator_EstimateCarbonGrams(t *testing.T) {
	tests := []struct {
		name           string
		config         LambdaFunctionConfig
		minCarbonGrams float64
		maxCarbonGrams float64
	}{
		{
			name: "1792MB memory (1 vCPU equiv), 500ms, 1M invocations - x86_64",
			config: LambdaFunctionConfig{
				MemoryMB:     1792,
				DurationMs:   500,
				Invocations:  1_000_000,
				Architecture: "x86_64",
				Region:       "us-east-1",
			},
			// Running time: 500 × 1M / 3.6M = 138.89 hours
			// Average watts: 2.12 + 0.5 × (4.5 - 2.12) = 3.31 W
			// Energy: 3.31 × 1 × 138.89 / 1000 = 0.46 kWh × 1.135 = 0.522 kWh
			// Carbon: 0.522 × 0.000379 × 1e6 = 197.8 gCO2e
			minCarbonGrams: 100,
			maxCarbonGrams: 400,
		},
		{
			name: "1792MB memory - arm64 (20% more efficient)",
			config: LambdaFunctionConfig{
				MemoryMB:     1792,
				DurationMs:   500,
				Invocations:  1_000_000,
				Architecture: "arm64",
				Region:       "us-east-1",
			},
			// Same as above but × 0.80 for ARM efficiency
			minCarbonGrams: 80,
			maxCarbonGrams: 320,
		},
		{
			name: "512MB memory (0.29 vCPU equiv), 100ms, 10M invocations",
			config: LambdaFunctionConfig{
				MemoryMB:     512,
				DurationMs:   100,
				Invocations:  10_000_000,
				Architecture: "x86_64",
				Region:       "us-east-1",
			},
			// vCPU equiv: 512/1792 = 0.286
			// Running time: 100 × 10M / 3.6M = 277.78 hours
			// Energy: 3.31 × 0.286 × 277.78 / 1000 = 0.263 kWh × 1.135 = 0.298 kWh
			minCarbonGrams: 50,
			maxCarbonGrams: 250,
		},
		{
			name: "low-carbon region (eu-north-1)",
			config: LambdaFunctionConfig{
				MemoryMB:     1792,
				DurationMs:   500,
				Invocations:  1_000_000,
				Architecture: "x86_64",
				Region:       "eu-north-1",
			},
			// Much lower due to clean grid
			minCarbonGrams: 1,
			maxCarbonGrams: 20,
		},
		{
			name: "zero invocations",
			config: LambdaFunctionConfig{
				MemoryMB:     1792,
				DurationMs:   500,
				Invocations:  0,
				Architecture: "x86_64",
				Region:       "us-east-1",
			},
			minCarbonGrams: 0,
			maxCarbonGrams: 0.001,
		},
	}

	e := NewLambdaEstimator()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			carbon, ok := e.EstimateCarbonGrams(tt.config)

			require.True(t, ok, "EstimateCarbonGrams should always succeed for Lambda")
			assert.GreaterOrEqual(t, carbon, tt.minCarbonGrams)
			assert.LessOrEqual(t, carbon, tt.maxCarbonGrams)
		})
	}
}

// TestLambdaEstimator_ARM64Efficiency verifies ARM64 is more efficient.
func TestLambdaEstimator_ARM64Efficiency(t *testing.T) {
	e := NewLambdaEstimator()

	carbonX86, _ := e.EstimateCarbonGramsSimple(1792, 500, 1_000_000, "x86_64", "us-east-1")
	carbonARM, _ := e.EstimateCarbonGramsSimple(1792, 500, 1_000_000, "arm64", "us-east-1")

	// ARM64 should be 80% of x86_64 (20% efficiency improvement)
	ratio := carbonARM / carbonX86
	assert.InDelta(t, 0.80, ratio, 0.01, "ARM64/x86_64 ratio should be 0.80")
}

// TestLambdaEstimator_MemoryScaling verifies carbon scales with memory.
func TestLambdaEstimator_MemoryScaling(t *testing.T) {
	e := NewLambdaEstimator()

	carbon1024, _ := e.EstimateCarbonGramsSimple(1024, 500, 1_000_000, "x86_64", "us-east-1")
	carbon2048, _ := e.EstimateCarbonGramsSimple(2048, 500, 1_000_000, "x86_64", "us-east-1")

	// 2048MB should be ~2× carbon of 1024MB
	ratio := carbon2048 / carbon1024
	assert.InDelta(t, 2.0, ratio, 0.1, "Carbon should scale linearly with memory")
}

// TestLambdaEstimator_InputValidation verifies that invalid inputs are rejected.
func TestLambdaEstimator_InputValidation(t *testing.T) {
	e := NewLambdaEstimator()

	tests := []struct {
		name   string
		config LambdaFunctionConfig
	}{
		{
			name: "zero memory",
			config: LambdaFunctionConfig{
				MemoryMB:     0,
				DurationMs:   500,
				Invocations:  1_000_000,
				Architecture: "x86_64",
				Region:       "us-east-1",
			},
		},
		{
			name: "negative memory",
			config: LambdaFunctionConfig{
				MemoryMB:     -1024,
				DurationMs:   500,
				Invocations:  1_000_000,
				Architecture: "x86_64",
				Region:       "us-east-1",
			},
		},
		{
			name: "negative duration",
			config: LambdaFunctionConfig{
				MemoryMB:     1792,
				DurationMs:   -500,
				Invocations:  1_000_000,
				Architecture: "x86_64",
				Region:       "us-east-1",
			},
		},
		{
			name: "negative invocations",
			config: LambdaFunctionConfig{
				MemoryMB:     1792,
				DurationMs:   500,
				Invocations:  -1000,
				Architecture: "x86_64",
				Region:       "us-east-1",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			carbon, ok := e.EstimateCarbonGrams(tt.config)
			assert.False(t, ok, "EstimateCarbonGrams should return false for invalid inputs")
			assert.Equal(t, 0.0, carbon, "Carbon should be 0 for invalid inputs")
		})
	}
}
