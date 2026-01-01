package carbon

import (
	"fmt"
	"strings"
)

// EmbodiedCarbonEstimator calculates the embodied carbon footprint of AWS resources.
// Embodied carbon represents the carbon emissions from manufacturing the hardware.
// Uses CCF methodology: 1000 kgCO2e per server, amortized over 48 months (4 years).
type EmbodiedCarbonEstimator struct {
	// EmbodiedCarbonPerServerKg is the total embodied carbon per server in kgCO2e.
	// Default is 1000 kg from CCF methodology.
	EmbodiedCarbonPerServerKg float64

	// ServerLifespanMonths is the expected lifespan for amortization.
	// Default is 48 months (4 years).
	ServerLifespanMonths float64
}

// NewEmbodiedCarbonEstimator creates a new estimator with default CCF values.
func NewEmbodiedCarbonEstimator() *EmbodiedCarbonEstimator {
	return &EmbodiedCarbonEstimator{
		EmbodiedCarbonPerServerKg: EmbodiedCarbonPerServerKg,
		ServerLifespanMonths:      ServerLifespanMonths,
	}
}

// EstimateEmbodiedCarbonKg calculates the embodied carbon for an EC2 instance.
// It uses the CCF formula: (EmbodiedPerServer / LifespanMonths) × (vCPUs / MaxFamilyVCPUs) × months
//
// Parameters:
//   - instanceType: EC2 instance type (e.g., "m5.large")
//   - months: Duration in months to calculate embodied carbon for
//
// Returns:
//   - embodiedCarbonKg: Embodied carbon in kilograms CO2e
//   - ok: Whether the calculation was successful (instance type found)
func (e *EmbodiedCarbonEstimator) EstimateEmbodiedCarbonKg(instanceType string, months float64) (float64, bool) {
	// Validate inputs to prevent division by zero or negative results
	if months <= 0 {
		return 0, false
	}
	if e.ServerLifespanMonths <= 0 {
		return 0, false
	}

	spec, ok := GetInstanceSpec(instanceType)
	if !ok {
		return 0, false
	}

	// Validate spec has valid vCPU count
	if spec.VCPUCount <= 0 {
		return 0, false
	}

	// Get max vCPUs for the instance family
	maxVCPUs := GetMaxFamilyVCPUs(instanceType)
	if maxVCPUs <= 0 {
		// Fallback: use instance's own vCPUs as max (full server allocation)
		maxVCPUs = float64(spec.VCPUCount)
	}

	// CCF Formula: (EmbodiedPerServer / LifespanMonths) × (vCPUs / MaxFamilyVCPUs) × months
	monthlyAmortizedPerServer := e.EmbodiedCarbonPerServerKg / e.ServerLifespanMonths
	vCPURatio := float64(spec.VCPUCount) / maxVCPUs
	embodiedCarbonKg := monthlyAmortizedPerServer * vCPURatio * months

	return embodiedCarbonKg, true
}

// EstimateEmbodiedCarbonGrams returns embodied carbon in grams CO2e for convenience.
func (e *EmbodiedCarbonEstimator) EstimateEmbodiedCarbonGrams(instanceType string, months float64) (float64, bool) {
	carbonKg, ok := e.EstimateEmbodiedCarbonKg(instanceType, months)
	if !ok {
		return 0, false
	}
	return carbonKg * 1000, ok // Convert kg to grams
}

// GetTotalCarbonGrams returns the combined operational + embodied carbon footprint.
//
// Parameters:
//   - instanceType: EC2 instance type (e.g., "m5.large")
//   - region: AWS region (e.g., "us-east-1")
//   - utilization: CPU utilization as a decimal (0.0 to 1.0)
//   - hours: Duration in hours
//
// Returns:
//   - operationalCarbon: Operational carbon in grams CO2e
//   - embodiedCarbon: Embodied carbon in grams CO2e
//   - totalCarbon: Sum of operational and embodied carbon
//   - ok: Whether the calculation was successful
func (e *EmbodiedCarbonEstimator) GetTotalCarbonGrams(
	instanceType string,
	region string,
	utilization float64,
	hours float64,
) (operationalCarbon, embodiedCarbon, totalCarbon float64, ok bool) {
	// Use the Estimator for operational carbon (includes GPU)
	estimator := NewEstimator()
	operationalCarbon, ok = estimator.EstimateCarbonGrams(instanceType, region, utilization, hours)
	if !ok {
		return 0, 0, 0, false
	}

	// Convert hours to months for embodied carbon calculation
	months := hours / HoursPerMonth

	// Calculate embodied carbon
	embodiedCarbon, ok = e.EstimateEmbodiedCarbonGrams(instanceType, months)
	if !ok {
		return 0, 0, 0, false
	}

	totalCarbon = operationalCarbon + embodiedCarbon
	return operationalCarbon, embodiedCarbon, totalCarbon, true
}

// GetBillingDetail returns a human-readable explanation of the embodied carbon calculation.
func (e *EmbodiedCarbonEstimator) GetBillingDetail(instanceType string, months float64) string {
	spec, ok := GetInstanceSpec(instanceType)
	if !ok {
		return "Unknown instance type for embodied carbon calculation"
	}

	maxVCPUs := GetMaxFamilyVCPUs(instanceType)
	if maxVCPUs == 0 {
		maxVCPUs = float64(spec.VCPUCount)
	}

	monthlyAmortized := e.EmbodiedCarbonPerServerKg / e.ServerLifespanMonths

	return fmt.Sprintf("Embodied carbon: %s (%d/%.0f vCPUs of server), %.2f kgCO2e/month amortized over %.0f months for %.1f months",
		instanceType, spec.VCPUCount, maxVCPUs, monthlyAmortized, e.ServerLifespanMonths, months)
}

// GetMaxFamilyVCPUs returns the maximum vCPU count for the instance family.
// This is used to calculate the proportional share of embodied carbon.
//
// Instance families with their max vCPUs (from AWS documentation):
//   - t3: 8 (t3.2xlarge)
//   - m5: 96 (m5.24xlarge)
//   - m6i: 128 (m6i.32xlarge)
//   - c5: 96 (c5.24xlarge)
//   - r5: 96 (r5.24xlarge)
//   - p4d: 96 (p4d.24xlarge)
//   - g4dn: 96 (g4dn.metal)
func GetMaxFamilyVCPUs(instanceType string) float64 {
	// Parse the family from instance type (e.g., "m5" from "m5.large")
	family := parseInstanceFamily(instanceType)

	// Max vCPUs per family based on AWS documentation.
	// Reference: https://aws.amazon.com/ec2/instance-types/
	// Last updated: 2025-01 (update when AWS adds new instance families)
	maxVCPUs := map[string]float64{
		// General purpose
		"t2":  8,   // t2.2xlarge
		"t3":  8,   // t3.2xlarge
		"t3a": 8,   // t3a.2xlarge
		"m4":  40,  // m4.10xlarge
		"m5":  96,  // m5.24xlarge
		"m5a": 96,  // m5a.24xlarge
		"m5n": 96,  // m5n.24xlarge
		"m6i": 128, // m6i.32xlarge
		"m6a": 192, // m6a.48xlarge

		// Compute optimized
		"c4":  36,  // c4.8xlarge
		"c5":  96,  // c5.24xlarge
		"c5a": 96,  // c5a.24xlarge
		"c5n": 72,  // c5n.18xlarge
		"c6i": 128, // c6i.32xlarge

		// Memory optimized
		"r4":  64,  // r4.16xlarge
		"r5":  96,  // r5.24xlarge
		"r5a": 96,  // r5a.24xlarge
		"r5n": 96,  // r5n.24xlarge
		"r6i": 128, // r6i.32xlarge

		// Storage optimized
		"i3":   64, // i3.16xlarge
		"i3en": 96, // i3en.24xlarge
		"d2":   36, // d2.8xlarge
		"d3":   96, // d3.8xlarge

		// GPU instances
		"p3":   64, // p3.16xlarge
		"p4d":  96, // p4d.24xlarge
		"p5":   96, // Estimated
		"g4dn": 96, // g4dn.metal
		"g5":   96, // g5.48xlarge

		// Inference instances
		"inf1": 96,  // inf1.24xlarge
		"inf2": 192, // inf2.48xlarge

		// Training instances
		"trn1": 128, // trn1.32xlarge
	}

	if max, ok := maxVCPUs[family]; ok {
		return max
	}

	// Default: return 0 to signal that caller should use instance's own vCPUs
	return 0
}

// parseInstanceFamily extracts the family from an instance type.
// e.g., "m5.large" -> "m5", "p4d.24xlarge" -> "p4d"
func parseInstanceFamily(instanceType string) string {
	family, _, _ := strings.Cut(instanceType, ".")
	return family
}
