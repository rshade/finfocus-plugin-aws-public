package carbon

import (
	"fmt"
	"strings"
)

// LambdaEstimator estimates carbon footprint for Lambda functions.
type LambdaEstimator struct{}

// NewLambdaEstimator creates a new Lambda carbon estimator.
func NewLambdaEstimator() *LambdaEstimator {
	return &LambdaEstimator{}
}

// EstimateCarbonGrams calculates the carbon footprint for Lambda function invocations.
//
// The calculation follows the CCF methodology for serverless:
//  1. vCPU Equivalent = Memory (MB) / 1792 MB (AWS Lambda: 1792 MB = 1 vCPU)
//  2. Running Time (Hours) = Duration (ms) × Invocations / 3,600,000
//  3. Average Watts = MinWatts + 0.50 × (MaxWatts - MinWatts) [50% utilization assumption]
//  4. Energy (kWh) = (Average Watts × vCPU Equivalent × Running Time) / 1000
//  5. Energy with PUE = Energy × AWS_PUE (1.135)
//  6. Carbon (gCO2e) = Energy with PUE × Grid Factor × 1,000,000
//  7. ARM64 Adjustment: Multiply by 0.80 for arm64 architecture (20% efficiency)
//
// Returns the carbon footprint in grams CO2e and whether the calculation succeeded.
func (e *LambdaEstimator) EstimateCarbonGrams(config LambdaFunctionConfig) (float64, bool) {
	// Validate inputs: negative or zero memory would produce incorrect estimates
	if config.MemoryMB <= 0 || config.DurationMs < 0 || config.Invocations < 0 {
		return 0, false
	}

	// Calculate vCPU equivalent based on memory allocation
	vCPUEquivalent := float64(config.MemoryMB) / VCPUPer1792MB

	// Calculate running time in hours
	runningTimeHours := float64(config.DurationMs) * float64(config.Invocations) / 3_600_000.0

	// Calculate average watts at 50% utilization (CCF assumption for Lambda)
	avgWatts := LambdaMinWattsPerVCPU + DefaultUtilization*(LambdaMaxWattsPerVCPU-LambdaMinWattsPerVCPU)

	// Calculate energy (kWh)
	energyKWh := (avgWatts * vCPUEquivalent * runningTimeHours) / 1000.0

	// Apply PUE
	energyWithPUE := energyKWh * AWSPUE

	// Get grid factor for region
	gridFactor := GetGridFactor(config.Region)

	// Calculate carbon (gCO2e)
	carbonGrams := energyWithPUE * gridFactor * 1_000_000

	// Apply ARM64 efficiency factor if applicable
	if strings.ToLower(config.Architecture) == "arm64" {
		carbonGrams *= ARM64EfficiencyFactor
	}

	return carbonGrams, true
}

// EstimateCarbonGramsSimple is a convenience method that takes individual parameters.
func (e *LambdaEstimator) EstimateCarbonGramsSimple(memoryMB, durationMs int, invocations int64, architecture, region string) (float64, bool) {
	return e.EstimateCarbonGrams(LambdaFunctionConfig{
		MemoryMB:     memoryMB,
		DurationMs:   durationMs,
		Invocations:  invocations,
		Architecture: architecture,
		Region:       region,
	})
}

// GetBillingDetail returns a human-readable description of the carbon estimation.
func (e *LambdaEstimator) GetBillingDetail(config LambdaFunctionConfig) string {
	arch := config.Architecture
	if arch == "" {
		arch = "x86_64"
	}

	vCPUEquivalent := float64(config.MemoryMB) / VCPUPer1792MB
	runningTimeHours := float64(config.DurationMs) * float64(config.Invocations) / 3_600_000.0

	return fmt.Sprintf("Lambda %s, %d MB memory (%.2f vCPU equiv), %d invocations × %dms = %.2f compute hours",
		arch, config.MemoryMB, vCPUEquivalent, config.Invocations, config.DurationMs, runningTimeHours)
}
