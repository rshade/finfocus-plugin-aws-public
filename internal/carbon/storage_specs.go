package carbon

import (
	_ "embed"
	"encoding/csv"
	"io"
	"strconv"
	"strings"
	"sync"
)

// CSV column indices for storage specs.
const (
	colStorageServiceType       = 0 // service_type (ebs, s3, dynamodb)
	colStorageClass             = 1 // storage_class (gp2, STANDARD, etc.)
	colStorageTechnology        = 2 // technology (SSD, HDD)
	colStorageReplicationFactor = 3 // replication_factor
	colStoragePowerCoefficient  = 4 // power_coefficient_wh_per_tbh
)

//go:embed data/storage_specs.csv
var storageSpecsCSV string

// StorageSpec contains storage specifications for a volume or bucket type.
type StorageSpec struct {
	// ServiceType is the AWS service (ebs, s3, dynamodb).
	ServiceType string

	// StorageClass is the storage class or volume type (e.g., "gp3", "STANDARD").
	StorageClass string

	// Technology is the storage technology (SSD or HDD).
	Technology string

	// ReplicationFactor is the data replication factor for durability (1×, 2×, 3×).
	ReplicationFactor int

	// PowerCoefficient is the power coefficient in Watt-Hours per TB-Hour.
	PowerCoefficient float64
}

var (
	storageSpecs     map[string]StorageSpec
	storageSpecsOnce sync.Once
)

// storageKey generates a unique key for storage spec lookup.
func storageKey(serviceType, storageClass string) string {
	return strings.ToLower(serviceType) + ":" + strings.ToUpper(storageClass)
}

// parseStorageSpecs initializes the package-level storageSpecs map by parsing
// the embedded CSV of storage specifications.
func parseStorageSpecs() {
	storageSpecs = make(map[string]StorageSpec)

	reader := csv.NewReader(strings.NewReader(storageSpecsCSV))

	// Skip header row
	_, err := reader.Read()
	if err != nil {
		logger.Error().Err(err).Msg("failed to read storage specs CSV header")
		return
	}

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			logger.Warn().Err(err).Msg("skipping malformed storage specs CSV row")
			continue
		}

		// Ensure we have enough columns
		if len(record) <= colStoragePowerCoefficient {
			continue
		}

		serviceType := strings.TrimSpace(record[colStorageServiceType])
		storageClass := strings.TrimSpace(record[colStorageClass])
		technology := strings.TrimSpace(record[colStorageTechnology])

		if serviceType == "" || storageClass == "" {
			continue
		}

		// Parse replication factor
		replicationFactor, err := strconv.Atoi(strings.TrimSpace(record[colStorageReplicationFactor]))
		if err != nil || replicationFactor < 1 {
			continue
		}

		// Parse power coefficient
		powerCoefficient, err := strconv.ParseFloat(strings.TrimSpace(record[colStoragePowerCoefficient]), 64)
		if err != nil || powerCoefficient <= 0 {
			continue
		}

		key := storageKey(serviceType, storageClass)
		storageSpecs[key] = StorageSpec{
			ServiceType:       serviceType,
			StorageClass:      storageClass,
			Technology:        technology,
			ReplicationFactor: replicationFactor,
			PowerCoefficient:  powerCoefficient,
		}
	}
}

// GetStorageSpec retrieves the StorageSpec for the given service type and storage class.
// Returns the StorageSpec and true if found, or an empty StorageSpec and false otherwise.
func GetStorageSpec(serviceType, storageClass string) (StorageSpec, bool) {
	storageSpecsOnce.Do(parseStorageSpecs)
	key := storageKey(serviceType, storageClass)
	spec, ok := storageSpecs[key]
	return spec, ok
}

// GetEBSStorageSpec retrieves the StorageSpec for an EBS volume type.
func GetEBSStorageSpec(volumeType string) (StorageSpec, bool) {
	return GetStorageSpec("ebs", volumeType)
}

// GetS3StorageSpec retrieves the StorageSpec for an S3 storage class.
func GetS3StorageSpec(storageClass string) (StorageSpec, bool) {
	return GetStorageSpec("s3", storageClass)
}

// GetDynamoDBStorageSpec retrieves the StorageSpec for DynamoDB.
func GetDynamoDBStorageSpec() (StorageSpec, bool) {
	return GetStorageSpec("dynamodb", "DYNAMODB")
}

// StorageSpecCount reports the number of loaded storage specifications.
func StorageSpecCount() int {
	storageSpecsOnce.Do(parseStorageSpecs)
	return len(storageSpecs)
}

// CalculateStorageEnergyKWh calculates the energy consumption for storage.
// Parameters:
//   - sizeGB: Storage size in gigabytes
//   - hours: Duration in hours
//   - serviceType: AWS service (ebs, s3, dynamodb)
//   - storageClass: Storage class or volume type
//
// Returns the energy consumption in kWh, or 0 if the storage type is unknown.
func CalculateStorageEnergyKWh(sizeGB, hours float64, serviceType, storageClass string) float64 {
	spec, ok := GetStorageSpec(serviceType, storageClass)
	if !ok {
		return 0
	}

	// Convert GB to TB
	sizeTB := sizeGB / 1024.0

	// Energy (kWh) = (Size in TB × Hours × Power Coefficient × Replication Factor) / 1000
	energyWh := sizeTB * hours * spec.PowerCoefficient * float64(spec.ReplicationFactor)
	return energyWh / 1000.0
}

// CalculateStorageCarbonGrams calculates the carbon emissions for storage.
// Parameters:
//   - sizeGB: Storage size in gigabytes
//   - hours: Duration in hours
//   - serviceType: AWS service (ebs, s3, dynamodb)
//   - storageClass: Storage class or volume type
//   - region: AWS region for grid factor lookup
//
// Returns the carbon emissions in gCO2e, or 0 if the storage type is unknown.
func CalculateStorageCarbonGrams(sizeGB, hours float64, serviceType, storageClass, region string) float64 {
	energyKWh := CalculateStorageEnergyKWh(sizeGB, hours, serviceType, storageClass)
	if energyKWh == 0 {
		return 0
	}

	gridFactor := GetGridFactor(region)

	// Apply PUE and convert to grams
	energyWithPUE := energyKWh * AWSPUE
	carbonGrams := energyWithPUE * gridFactor * 1_000_000

	return carbonGrams
}
