package carbon

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestEFSEstimator_EstimateCarbonGrams verifies the per-class sum, the One Zone
// replication change, and input validation.
func TestEFSEstimator_EstimateCarbonGrams(t *testing.T) {
	e := NewEFSEstimator()
	const region = "us-east-1"
	const hours = HoursPerMonth

	std := CalculateStorageCarbonGrams(100, hours, "efs", "STANDARD", region)
	ia := CalculateStorageCarbonGrams(400, hours, "efs", "IA", region)
	archive := CalculateStorageCarbonGrams(50, hours, "efs", "ARCHIVE", region)
	oneZone := CalculateStorageCarbonGrams(100, hours, "efs", "ONEZONE", region)

	tests := []struct {
		name   string
		config EFSStorageConfig
		want   float64
		wantOK bool
	}{
		{
			name:   "regional sums every class",
			config: EFSStorageConfig{StandardGB: 100, IAGB: 400, ArchiveGB: 50, Region: region, Hours: hours},
			want:   std + ia + archive,
			wantOK: true,
		},
		{
			name:   "one zone ignores archive and uses lower replication",
			config: EFSStorageConfig{StandardGB: 100, ArchiveGB: 50, OneZone: true, Region: region, Hours: hours},
			want:   oneZone,
			wantOK: true,
		},
		{
			name:   "negative size is rejected",
			config: EFSStorageConfig{StandardGB: -1, Region: region, Hours: hours},
			wantOK: false,
		},
		{
			name:   "negative hours are rejected",
			config: EFSStorageConfig{StandardGB: 1, Region: region, Hours: -1},
			wantOK: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := e.EstimateCarbonGrams(tt.config)
			assert.Equal(t, tt.wantOK, ok)
			if tt.wantOK {
				assert.InDelta(t, tt.want, got, 1e-9)
			}
		})
	}

	assert.InDelta(t, std*2/3, oneZone, 1e-9, "One Zone replication (2x) is two thirds of regional (3x)")
}

// TestGetEFSStorageSpec verifies the EFS storage classes loaded from the CSV.
func TestGetEFSStorageSpec(t *testing.T) {
	for class, replication := range map[string]int{
		"STANDARD": 3, "IA": 3, "ARCHIVE": 3, "ONEZONE": 2, "ONEZONE_IA": 2,
	} {
		spec, ok := GetEFSStorageSpec(class)
		assert.True(t, ok, class)
		assert.Equal(t, "HDD", spec.Technology, class)
		assert.Equal(t, replication, spec.ReplicationFactor, class)
	}
	_, ok := GetEFSStorageSpec("GLACIER")
	assert.False(t, ok)
}
