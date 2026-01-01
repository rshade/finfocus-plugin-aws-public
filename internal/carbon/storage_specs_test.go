package carbon

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetStorageSpec_KnownTypes verifies storage spec lookup for known types.
func TestGetStorageSpec_KnownTypes(t *testing.T) {
	tests := []struct {
		name            string
		serviceType     string
		storageClass    string
		wantTechnology  string
		wantReplication int
		wantPowerCoeff  float64
	}{
		{
			name:            "EBS gp3 (SSD)",
			serviceType:     "ebs",
			storageClass:    "gp3",
			wantTechnology:  "SSD",
			wantReplication: 2,
			wantPowerCoeff:  1.2,
		},
		{
			name:            "EBS gp2 (SSD)",
			serviceType:     "ebs",
			storageClass:    "gp2",
			wantTechnology:  "SSD",
			wantReplication: 2,
			wantPowerCoeff:  1.2,
		},
		{
			name:            "EBS io1 (SSD)",
			serviceType:     "ebs",
			storageClass:    "io1",
			wantTechnology:  "SSD",
			wantReplication: 2,
			wantPowerCoeff:  1.2,
		},
		{
			name:            "EBS st1 (HDD)",
			serviceType:     "ebs",
			storageClass:    "st1",
			wantTechnology:  "HDD",
			wantReplication: 2,
			wantPowerCoeff:  0.65,
		},
		{
			name:            "S3 STANDARD",
			serviceType:     "s3",
			storageClass:    "STANDARD",
			wantTechnology:  "SSD",
			wantReplication: 3,
			wantPowerCoeff:  1.2,
		},
		{
			name:            "S3 ONEZONE_IA (single AZ)",
			serviceType:     "s3",
			storageClass:    "ONEZONE_IA",
			wantTechnology:  "SSD",
			wantReplication: 1,
			wantPowerCoeff:  1.2,
		},
		{
			name:            "S3 GLACIER (HDD)",
			serviceType:     "s3",
			storageClass:    "GLACIER",
			wantTechnology:  "HDD",
			wantReplication: 3,
			wantPowerCoeff:  0.65,
		},
		{
			name:            "DynamoDB default",
			serviceType:     "dynamodb",
			storageClass:    "DYNAMODB",
			wantTechnology:  "SSD",
			wantReplication: 3,
			wantPowerCoeff:  1.2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec, ok := GetStorageSpec(tt.serviceType, tt.storageClass)
			require.True(t, ok, "GetStorageSpec should find %s:%s", tt.serviceType, tt.storageClass)

			assert.Equal(t, tt.wantTechnology, spec.Technology)
			assert.Equal(t, tt.wantReplication, spec.ReplicationFactor)
			assert.Equal(t, tt.wantPowerCoeff, spec.PowerCoefficient)
		})
	}
}

// TestGetStorageSpec_UnknownType verifies unknown storage types return false.
func TestGetStorageSpec_UnknownType(t *testing.T) {
	_, ok := GetStorageSpec("ebs", "unknown")
	assert.False(t, ok)

	_, ok = GetStorageSpec("unknown", "gp3")
	assert.False(t, ok)
}

// TestGetStorageSpec_CaseInsensitive verifies case-insensitive lookup for service type.
func TestGetStorageSpec_CaseInsensitive(t *testing.T) {
	spec1, ok1 := GetStorageSpec("ebs", "gp3")
	spec2, ok2 := GetStorageSpec("EBS", "gp3")

	require.True(t, ok1)
	require.True(t, ok2)
	assert.Equal(t, spec1.Technology, spec2.Technology)
}

// TestGetEBSStorageSpec verifies the EBS-specific convenience function.
func TestGetEBSStorageSpec(t *testing.T) {
	spec, ok := GetEBSStorageSpec("gp3")
	require.True(t, ok)
	assert.Equal(t, "SSD", spec.Technology)
	assert.Equal(t, 2, spec.ReplicationFactor)
}

// TestGetS3StorageSpec verifies the S3-specific convenience function.
func TestGetS3StorageSpec(t *testing.T) {
	spec, ok := GetS3StorageSpec("STANDARD")
	require.True(t, ok)
	assert.Equal(t, "SSD", spec.Technology)
	assert.Equal(t, 3, spec.ReplicationFactor)
}

// TestGetDynamoDBStorageSpec verifies the DynamoDB-specific convenience function.
func TestGetDynamoDBStorageSpec(t *testing.T) {
	spec, ok := GetDynamoDBStorageSpec()
	require.True(t, ok)
	assert.Equal(t, "SSD", spec.Technology)
	assert.Equal(t, 3, spec.ReplicationFactor)
}

// TestStorageSpecCount verifies storage specs are loaded.
func TestStorageSpecCount(t *testing.T) {
	count := StorageSpecCount()
	// We have EBS (7 types) + S3 (7 types) + DynamoDB (1) = 15 total
	assert.GreaterOrEqual(t, count, 10, "StorageSpecCount should have at least 10 entries")
}

// TestCalculateStorageEnergyKWh verifies storage energy calculation.
func TestCalculateStorageEnergyKWh(t *testing.T) {
	tests := []struct {
		name          string
		sizeGB        float64
		hours         float64
		serviceType   string
		storageClass  string
		wantEnergyKWh float64
	}{
		{
			name:         "100GB gp3 for 1 month",
			sizeGB:       100,
			hours:        730,
			serviceType:  "ebs",
			storageClass: "gp3",
			// Energy = (100/1024 TB) × 730h × 1.2 Wh/TB × 2 replication / 1000
			// = 0.09765625 × 730 × 1.2 × 2 / 1000 = 0.171 kWh
			wantEnergyKWh: 0.171,
		},
		{
			name:         "1TB S3 STANDARD for 1 month",
			sizeGB:       1024,
			hours:        730,
			serviceType:  "s3",
			storageClass: "STANDARD",
			// Energy = 1 TB × 730h × 1.2 Wh/TB × 3 replication / 1000
			// = 730 × 1.2 × 3 / 1000 = 2.628 kWh
			wantEnergyKWh: 2.628,
		},
		{
			name:          "unknown storage type",
			sizeGB:        100,
			hours:         730,
			serviceType:   "unknown",
			storageClass:  "unknown",
			wantEnergyKWh: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateStorageEnergyKWh(tt.sizeGB, tt.hours, tt.serviceType, tt.storageClass)
			assert.InDelta(t, tt.wantEnergyKWh, got, 0.01, "Energy should be approximately %f kWh", tt.wantEnergyKWh)
		})
	}
}

// TestCalculateStorageCarbonGrams verifies storage carbon calculation.
func TestCalculateStorageCarbonGrams(t *testing.T) {
	tests := []struct {
		name           string
		sizeGB         float64
		hours          float64
		serviceType    string
		storageClass   string
		region         string
		minCarbonGrams float64
		maxCarbonGrams float64
	}{
		{
			name:           "100GB gp3 EBS for 1 month in us-east-1",
			sizeGB:         100,
			hours:          730,
			serviceType:    "ebs",
			storageClass:   "gp3",
			region:         "us-east-1",
			minCarbonGrams: 50, // ~73 gCO2e expected
			maxCarbonGrams: 150,
		},
		{
			name:           "1TB S3 STANDARD for 1 month in us-east-1",
			sizeGB:         1024,
			hours:          730,
			serviceType:    "s3",
			storageClass:   "STANDARD",
			region:         "us-east-1",
			minCarbonGrams: 500, // ~1131 gCO2e expected
			maxCarbonGrams: 2000,
		},
		{
			name:           "100GB EBS in low-carbon region (eu-north-1)",
			sizeGB:         100,
			hours:          730,
			serviceType:    "ebs",
			storageClass:   "gp3",
			region:         "eu-north-1",
			minCarbonGrams: 0.1, // Very low due to clean grid
			maxCarbonGrams: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateStorageCarbonGrams(tt.sizeGB, tt.hours, tt.serviceType, tt.storageClass, tt.region)
			assert.GreaterOrEqual(t, got, tt.minCarbonGrams)
			assert.LessOrEqual(t, got, tt.maxCarbonGrams)
		})
	}
}
