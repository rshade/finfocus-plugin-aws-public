package carbon

// EBSEstimator estimates carbon footprint for EBS volumes.
type EBSEstimator struct{}

// NewEBSEstimator creates a new EBS carbon estimator.
func NewEBSEstimator() *EBSEstimator {
	return &EBSEstimator{}
}

// EstimateCarbonGrams calculates the carbon footprint for an EBS volume.
//
// The calculation follows the CCF storage methodology:
//  1. Convert size from GB to TB
//  2. Energy (kWh) = (Size_TB × Hours × Power_Coefficient × Replication) / 1000
//  3. Energy with PUE = Energy × AWS_PUE (1.135)
//  4. Carbon (gCO2e) = Energy with PUE × Grid Factor × 1,000,000
//
// Parameters:
//   - config: EBS volume configuration with type, size, region, and hours
//
// Returns the carbon footprint in grams CO2e and whether the calculation succeeded.
// Returns (0, false) if the volume type is unknown or inputs are invalid.
func (e *EBSEstimator) EstimateCarbonGrams(config EBSVolumeConfig) (float64, bool) {
	// Validate inputs: negative values would produce incorrect estimates
	if config.SizeGB < 0 || config.Hours < 0 {
		return 0, false
	}

	// Check if spec exists first to match contract of returning (0, false) for unknown types
	_, ok := GetEBSStorageSpec(config.VolumeType)
	if !ok {
		return 0, false
	}

	// Use shared calculation logic
	carbonGrams := CalculateStorageCarbonGrams(
		config.SizeGB,
		config.Hours,
		"ebs",
		config.VolumeType,
		config.Region,
	)

	return carbonGrams, true
}

// EstimateCarbonGramsSimple is a convenience method that takes individual parameters.
func (e *EBSEstimator) EstimateCarbonGramsSimple(volumeType string, sizeGB float64, region string, hours float64) (float64, bool) {
	return e.EstimateCarbonGrams(EBSVolumeConfig{
		VolumeType: volumeType,
		SizeGB:     sizeGB,
		Region:     region,
		Hours:      hours,
	})
}

// GetBillingDetail returns a human-readable description of the carbon estimation.
func (e *EBSEstimator) GetBillingDetail(config EBSVolumeConfig) string {
	spec, ok := GetEBSStorageSpec(config.VolumeType)
	if !ok {
		return "Unknown EBS volume type"
	}

	return "EBS " + config.VolumeType + " (" + spec.Technology + "), " +
		formatFloat(config.SizeGB) + " GB, " +
		formatFloat(config.Hours) + " hrs, " +
		"replication " + formatInt(spec.ReplicationFactor) + "×"
}
