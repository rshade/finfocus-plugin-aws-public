package carbon

// DynamoDBEstimator estimates carbon footprint for DynamoDB tables.
type DynamoDBEstimator struct{}

// NewDynamoDBEstimator creates a new DynamoDB carbon estimator.
func NewDynamoDBEstimator() *DynamoDBEstimator {
	return &DynamoDBEstimator{}
}

// EstimateCarbonGrams calculates the carbon footprint for DynamoDB storage.
//
// DynamoDB carbon estimation is based on storage only (compute is managed/opaque):
//  1. Uses SSD storage technology (1.2 Wh/TB-hour)
//  2. Applies 3× replication factor (cross-AZ durability)
//  3. Follows the standard CCF storage methodology
//
// Parameters:
//   - config: DynamoDB table configuration with storage size, region, and hours
//
// Returns the carbon footprint in grams CO2e and whether the calculation succeeded.
func (e *DynamoDBEstimator) EstimateCarbonGrams(config DynamoDBTableConfig) (float64, bool) {
	// Validate inputs
	if config.SizeGB < 0 || config.Hours < 0 {
		return 0, false
	}

	// Try standard lookup first
	_, ok := GetDynamoDBStorageSpec()
	if ok {
		// Use shared calculation logic
		carbonGrams := CalculateStorageCarbonGrams(
			config.SizeGB,
			config.Hours,
			"dynamodb",
			"DYNAMODB",
			config.Region,
		)
		return carbonGrams, true
	}

	// Fallback to DynamoDB defaults: SSD with 3× replication for durability
	// This ensures we always return a value even if specs are missing
	spec := StorageSpec{
		Technology:        "SSD",
		ReplicationFactor: 3,
		PowerCoefficient:  SSDPowerCoefficient,
	}

	// Convert GB to TB
	sizeTB := config.SizeGB / 1024.0

	// Calculate energy (kWh)
	energyWh := sizeTB * config.Hours * spec.PowerCoefficient * float64(spec.ReplicationFactor)
	energyKWh := energyWh / 1000.0

	// Apply PUE
	energyWithPUE := energyKWh * AWSPUE

	// Get grid factor for region
	gridFactor := GetGridFactor(config.Region)

	// Calculate carbon (gCO2e)
	carbonGrams := energyWithPUE * gridFactor * 1_000_000

	return carbonGrams, true
}

// EstimateCarbonGramsSimple is a convenience method that takes individual parameters.
func (e *DynamoDBEstimator) EstimateCarbonGramsSimple(sizeGB float64, region string, hours float64) (float64, bool) {
	return e.EstimateCarbonGrams(DynamoDBTableConfig{
		SizeGB: sizeGB,
		Region: region,
		Hours:  hours,
	})
}

// GetBillingDetail returns a human-readable description of the carbon estimation.
func (e *DynamoDBEstimator) GetBillingDetail(config DynamoDBTableConfig) string {
	return "DynamoDB table, " +
		formatFloat(config.SizeGB) + " GB storage (SSD, 3× replication), " +
		formatFloat(config.Hours) + " hrs"
}
