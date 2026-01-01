package carbon

// CarbonEstimator provides carbon emission estimation for resources.
type CarbonEstimator interface {
	// EstimateCarbonGrams calculates carbon emissions for an EC2 instance.
	// Returns carbon in grams CO2e and whether the calculation succeeded.
	// Returns (0, false) if the instance type is unknown.
	EstimateCarbonGrams(instanceType, region string, utilization, hours float64) (float64, bool)
}

// Estimator implements CarbonEstimator using CCF methodology.
type Estimator struct {
	// IncludeGPU controls whether GPU power consumption is included.
	// Default is true.
	IncludeGPU bool
}

// NewEstimator creates a new carbon estimator with GPU power included by default.
func NewEstimator() *Estimator {
	return &Estimator{
		IncludeGPU: true,
	}
}

// EstimateCarbonGrams calculates carbon emissions for an EC2 instance.
//
// The calculation follows the Cloud Carbon Footprint methodology:
//  1. Average CPU watts = MinWatts + (utilization × (MaxWatts - MinWatts))
//  2. GPU watts = GPU TDP × GPU count × utilization (if GPU instance and IncludeGPU)
//  3. Total watts = CPU watts × vCPU count + GPU watts
//  4. Energy (kWh) = Total watts × hours / 1000
//  5. Energy with PUE = Energy × AWS_PUE (1.135)
//  6. Carbon (gCO2e) = Energy with PUE × grid intensity × 1,000,000
//
// Returns (0, false) if the instance type is not found in CCF data.
func (e *Estimator) EstimateCarbonGrams(instanceType, region string, utilization, hours float64) (float64, bool) {
	cpuCarbon, gpuCarbon, ok := e.EstimateCarbonGramsWithBreakdown(instanceType, region, utilization, hours)
	if !ok {
		return 0, false
	}
	return cpuCarbon + gpuCarbon, true
}

// EstimateCarbonGramsWithBreakdown calculates carbon emissions for an EC2 instance
// and returns a breakdown of CPU and GPU contributions.
//
// Returns:
//   - cpuCarbon: Carbon from CPU power consumption (gCO2e)
//   - gpuCarbon: Carbon from GPU power consumption (gCO2e)
//   - ok: Whether the calculation succeeded
func (e *Estimator) EstimateCarbonGramsWithBreakdown(instanceType, region string, utilization, hours float64) (cpuCarbon, gpuCarbon float64, ok bool) {
	spec, found := GetInstanceSpec(instanceType)
	if !found {
		return 0, 0, false
	}

	gridFactor := GetGridFactor(region)

	// Calculate CPU carbon
	cpuCarbon = CalculateCarbonGrams(
		spec.MinWatts,
		spec.MaxWatts,
		spec.VCPUCount,
		utilization,
		gridFactor,
		hours,
	)

	// Calculate GPU carbon if enabled and instance has GPUs
	if e.IncludeGPU {
		gpuPowerWatts := CalculateGPUPowerWatts(instanceType, utilization)
		if gpuPowerWatts > 0 {
			gpuEnergyKWh := (gpuPowerWatts * hours) / 1000.0
			gpuEnergyWithPUE := gpuEnergyKWh * AWSPUE
			gpuCarbon = gpuEnergyWithPUE * gridFactor * 1_000_000
		}
	}

	return cpuCarbon, gpuCarbon, true
}

// CalculateCarbonGrams applies the CCF formula to calculate carbon emissions.
//
// Parameters:
//   - minWatts: Power consumption at idle (watts per vCPU)
//   - maxWatts: Power consumption at 100% utilization (watts per vCPU)
//   - vCPUCount: Number of virtual CPUs
//   - utilization: CPU utilization (0.0 to 1.0)
//   - gridIntensity: Grid carbon intensity (metric tons CO2eq/kWh)
//   - hours: Operating hours
//
// Returns the estimated carbon emissions in grams of CO2e.
func CalculateCarbonGrams(minWatts, maxWatts float64, vCPUCount int, utilization, gridIntensity, hours float64) float64 {
	// Step 1: Average watts based on utilization (linear interpolation)
	avgWatts := minWatts + (utilization * (maxWatts - minWatts))

	// Step 2: Energy consumption (kWh)
	energyKWh := (avgWatts * float64(vCPUCount) * hours) / 1000.0

	// Step 3: Apply Power Usage Effectiveness (PUE) overhead
	energyWithPUE := energyKWh * AWSPUE

	// Step 4: Carbon emissions (gCO2e)
	// gridIntensity is metric tons/kWh, multiply by 1,000,000 for grams
	carbonGrams := energyWithPUE * gridIntensity * 1_000_000

	return carbonGrams
}
