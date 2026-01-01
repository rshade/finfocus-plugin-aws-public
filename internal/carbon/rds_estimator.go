package carbon

import "strings"

// RDSEstimator estimates carbon footprint for RDS instances.
type RDSEstimator struct{}

// NewRDSEstimator creates a new RDS carbon estimator.
func NewRDSEstimator() *RDSEstimator {
	return &RDSEstimator{}
}

// EstimateCarbonGrams calculates the carbon footprint for an RDS instance.
//
// RDS carbon is a composite of:
//  1. Compute carbon: EC2-equivalent carbon for the db instance class
//  2. Storage carbon: EBS-equivalent carbon for the allocated storage
//  3. Multi-AZ multiplier: 2× for both compute and storage if Multi-AZ enabled
//
// Parameters:
//   - config: RDS instance configuration
//
// Returns the carbon footprint in grams CO2e and whether the calculation succeeded.
// Returns (0, false) if the instance type is unknown.
//
// This method is thread-safe and can be called concurrently.
func (e *RDSEstimator) EstimateCarbonGrams(config RDSInstanceConfig) (float64, bool) {
	// Convert RDS instance type to EC2 equivalent (db.m5.large -> m5.large)
	ec2InstanceType := rdsToEC2InstanceType(config.InstanceType)

	// Calculate compute carbon using a fresh EC2 estimator with GPU disabled.
	// RDS instances don't have GPUs, so we always disable GPU power calculation.
	ec2Estimator := NewEstimator()
	ec2Estimator.IncludeGPU = false
	computeCarbon, ok := ec2Estimator.EstimateCarbonGrams(
		ec2InstanceType,
		config.Region,
		config.Utilization,
		config.Hours,
	)
	if !ok {
		return 0, false
	}

	// Calculate storage carbon using EBS estimator
	ebsEstimator := NewEBSEstimator()
	storageCarbon, _ := ebsEstimator.EstimateCarbonGrams(EBSVolumeConfig{
		VolumeType: config.StorageType,
		SizeGB:     config.StorageSizeGB,
		Region:     config.Region,
		Hours:      config.Hours,
	})

	// Apply Multi-AZ multiplier (2× for both compute and storage)
	totalCarbon := computeCarbon + storageCarbon
	if config.MultiAZ {
		totalCarbon *= 2
	}

	return totalCarbon, true
}

// EstimateCarbonGramsWithBreakdown returns compute and storage carbon separately.
//
// This method is thread-safe and can be called concurrently.
func (e *RDSEstimator) EstimateCarbonGramsWithBreakdown(config RDSInstanceConfig) (computeCarbon, storageCarbon float64, ok bool) {
	ec2InstanceType := rdsToEC2InstanceType(config.InstanceType)

	// Create a fresh EC2 estimator per call for thread-safety (see EstimateCarbonGrams)
	ec2Estimator := NewEstimator()
	ec2Estimator.IncludeGPU = false
	computeCarbon, ok = ec2Estimator.EstimateCarbonGrams(
		ec2InstanceType,
		config.Region,
		config.Utilization,
		config.Hours,
	)
	if !ok {
		return 0, 0, false
	}

	ebsEstimator := NewEBSEstimator()
	storageCarbon, _ = ebsEstimator.EstimateCarbonGrams(EBSVolumeConfig{
		VolumeType: config.StorageType,
		SizeGB:     config.StorageSizeGB,
		Region:     config.Region,
		Hours:      config.Hours,
	})

	// Apply Multi-AZ multiplier
	if config.MultiAZ {
		computeCarbon *= 2
		storageCarbon *= 2
	}

	return computeCarbon, storageCarbon, true
}

// GetBillingDetail returns a human-readable description of the carbon estimation.
func (e *RDSEstimator) GetBillingDetail(config RDSInstanceConfig) string {
	multiAZ := ""
	if config.MultiAZ {
		multiAZ = " Multi-AZ (2×)"
	}

	return "RDS " + config.InstanceType + multiAZ + ", " +
		formatFloat(config.StorageSizeGB) + " GB " + config.StorageType + " storage, " +
		formatFloat(config.Hours) + " hrs, " +
		formatInt(int(config.Utilization*100)) + "% utilization"
}

// rdsToEC2InstanceType converts an RDS instance type to its EC2 equivalent.
// Example: db.m5.large -> m5.large
func rdsToEC2InstanceType(rdsType string) string {
	// Remove "db." prefix if present
	if strings.HasPrefix(rdsType, "db.") {
		return rdsType[3:]
	}
	return rdsType
}
