package carbon

// S3Estimator estimates carbon footprint for S3 storage.
type S3Estimator struct{}

// NewS3Estimator creates a new S3 carbon estimator.
func NewS3Estimator() *S3Estimator {
	return &S3Estimator{}
}

// EstimateCarbonGrams calculates the carbon footprint for S3 storage.
//
// The calculation follows the CCF storage methodology:
//  1. Convert size from GB to TB
//  2. Energy (kWh) = (Size_TB × Hours × Power_Coefficient × Replication) / 1000
//  3. Energy with PUE = Energy × AWS_PUE (1.135)
//  4. Carbon (gCO2e) = Energy with PUE × Grid Factor × 1,000,000
//
// Parameters:
//   - config: S3 storage configuration with class, size, region, and hours
//
// Returns the carbon footprint in grams CO2e and whether the calculation succeeded.
// Returns (0, false) if the storage class is unknown.
func (e *S3Estimator) EstimateCarbonGrams(config S3StorageConfig) (float64, bool) {
	// Validate inputs
	if config.SizeGB < 0 || config.Hours < 0 {
		return 0, false
	}

	// Check if spec exists first
	_, ok := GetS3StorageSpec(config.StorageClass)
	if !ok {
		return 0, false
	}

	// Use shared calculation logic
	carbonGrams := CalculateStorageCarbonGrams(
		config.SizeGB,
		config.Hours,
		"s3",
		config.StorageClass,
		config.Region,
	)

	return carbonGrams, true
}

// EstimateCarbonGramsSimple is a convenience method that takes individual parameters.
func (e *S3Estimator) EstimateCarbonGramsSimple(storageClass string, sizeGB float64, region string, hours float64) (float64, bool) {
	return e.EstimateCarbonGrams(S3StorageConfig{
		StorageClass: storageClass,
		SizeGB:       sizeGB,
		Region:       region,
		Hours:        hours,
	})
}

// GetBillingDetail returns a human-readable description of the carbon estimation.
func (e *S3Estimator) GetBillingDetail(config S3StorageConfig) string {
	spec, ok := GetS3StorageSpec(config.StorageClass)
	if !ok {
		return "Unknown S3 storage class"
	}

	return "S3 " + config.StorageClass + " (" + spec.Technology + "), " +
		formatFloat(config.SizeGB) + " GB, " +
		formatFloat(config.Hours) + " hrs, " +
		"replication " + formatInt(spec.ReplicationFactor) + "×"
}
