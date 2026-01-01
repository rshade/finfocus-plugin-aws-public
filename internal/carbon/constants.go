// Package carbon provides carbon emission estimation for AWS resources
// using Cloud Carbon Footprint (CCF) methodology.
package carbon

const (
	// AWSPUE is the Power Usage Effectiveness for AWS datacenters.
	// Source: Cloud Carbon Footprint methodology.
	AWSPUE = 1.135

	// DefaultUtilization is the default CPU utilization assumption (50%)
	// when no utilization_percentage is provided.
	// Source: CCF hyperscale datacenter assumption.
	DefaultUtilization = 0.50

	// HoursPerMonth is the standard hours per month for cost calculations.
	HoursPerMonth = 730.0

	// VCPUPer1792MB is the vCPU equivalent for Lambda memory allocation.
	// 1792 MB of Lambda memory = 1 vCPU.
	// Source: AWS Lambda documentation.
	VCPUPer1792MB = 1792.0

	// SSDPowerCoefficient is the power coefficient for SSD storage in Wh/TB-hour.
	// Source: Cloud Carbon Footprint methodology.
	SSDPowerCoefficient = 1.2

	// HDDPowerCoefficient is the power coefficient for HDD storage in Wh/TB-hour.
	// Source: Cloud Carbon Footprint methodology.
	HDDPowerCoefficient = 0.65

	// ARM64EfficiencyFactor is the efficiency improvement for ARM64 architecture.
	// ARM64 is approximately 20% more efficient than x86_64.
	// Source: Cloud Carbon Footprint methodology.
	ARM64EfficiencyFactor = 0.80

	// EmbodiedCarbonPerServerKg is the default embodied carbon per server in kgCO2e.
	// Source: Cloud Carbon Footprint methodology.
	EmbodiedCarbonPerServerKg = 1000.0

	// ServerLifespanMonths is the default server lifespan for amortization in months.
	// Source: Cloud Carbon Footprint methodology (4 years).
	ServerLifespanMonths = 48

	// LambdaMinWattsPerVCPU is the baseline power for Lambda functions at idle.
	// Source: CCF methodology typical values for modern x86 processors.
	LambdaMinWattsPerVCPU = 2.12

	// LambdaMaxWattsPerVCPU is the peak power for Lambda functions at 100% utilization.
	// Source: CCF methodology typical values for modern x86 processors.
	LambdaMaxWattsPerVCPU = 4.5
)
