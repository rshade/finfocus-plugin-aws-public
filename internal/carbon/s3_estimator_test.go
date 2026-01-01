package carbon

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestS3Estimator_EstimateCarbonGrams verifies S3 carbon estimation.
func TestS3Estimator_EstimateCarbonGrams(t *testing.T) {
	tests := []struct {
		name           string
		config         S3StorageConfig
		wantOK         bool
		minCarbonGrams float64
		maxCarbonGrams float64
	}{
		{
			name: "100GB S3 STANDARD for 1 month",
			config: S3StorageConfig{
				StorageClass: "STANDARD",
				SizeGB:       100,
				Region:       "us-east-1",
				Hours:        730,
			},
			wantOK:         true,
			minCarbonGrams: 80,   // ~110 gCO2e expected (higher than EBS due to 3× replication)
			maxCarbonGrams: 200,
		},
		{
			name: "1TB S3 STANDARD for 1 month",
			config: S3StorageConfig{
				StorageClass: "STANDARD",
				SizeGB:       1024,
				Region:       "us-east-1",
				Hours:        730,
			},
			wantOK:         true,
			minCarbonGrams: 800,  // ~1131 gCO2e expected
			maxCarbonGrams: 2000,
		},
		{
			name: "100GB S3 ONEZONE_IA (single replication)",
			config: S3StorageConfig{
				StorageClass: "ONEZONE_IA",
				SizeGB:       100,
				Region:       "us-east-1",
				Hours:        730,
			},
			wantOK:         true,
			minCarbonGrams: 20,  // Lower due to 1× replication
			maxCarbonGrams: 80,
		},
		{
			name: "100GB S3 GLACIER (HDD)",
			config: S3StorageConfig{
				StorageClass: "GLACIER",
				SizeGB:       100,
				Region:       "us-east-1",
				Hours:        730,
			},
			wantOK:         true,
			minCarbonGrams: 30, // Lower due to HDD coefficient (0.65 vs 1.2)
			maxCarbonGrams: 120,
		},
		{
			name: "unknown storage class",
			config: S3StorageConfig{
				StorageClass: "UNKNOWN_CLASS",
				SizeGB:       100,
				Region:       "us-east-1",
				Hours:        730,
			},
			wantOK:         false,
			minCarbonGrams: 0,
			maxCarbonGrams: 0,
		},
	}

	e := NewS3Estimator()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			carbon, ok := e.EstimateCarbonGrams(tt.config)

			if tt.wantOK {
				require.True(t, ok, "EstimateCarbonGrams should succeed")
				assert.GreaterOrEqual(t, carbon, tt.minCarbonGrams)
				assert.LessOrEqual(t, carbon, tt.maxCarbonGrams)
			} else {
				assert.False(t, ok)
				assert.Equal(t, 0.0, carbon)
			}
		})
	}
}

// TestS3Estimator_ReplicationImpact verifies replication affects carbon.
func TestS3Estimator_ReplicationImpact(t *testing.T) {
	e := NewS3Estimator()

	// STANDARD has 3× replication, ONEZONE_IA has 1×
	carbonStandard, ok1 := e.EstimateCarbonGramsSimple("STANDARD", 100, "us-east-1", 730)
	carbonOneZone, ok2 := e.EstimateCarbonGramsSimple("ONEZONE_IA", 100, "us-east-1", 730)

	require.True(t, ok1)
	require.True(t, ok2)

	// STANDARD should be 3× higher than ONEZONE_IA
	ratio := carbonStandard / carbonOneZone
	assert.InDelta(t, 3.0, ratio, 0.1, "STANDARD/ONEZONE_IA ratio should be ~3.0")
}
