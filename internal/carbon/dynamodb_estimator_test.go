package carbon

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDynamoDBEstimator_EstimateCarbonGrams verifies DynamoDB carbon estimation.
func TestDynamoDBEstimator_EstimateCarbonGrams(t *testing.T) {
	tests := []struct {
		name           string
		config         DynamoDBTableConfig
		minCarbonGrams float64
		maxCarbonGrams float64
	}{
		{
			name: "50GB table for 1 month in us-east-1",
			config: DynamoDBTableConfig{
				SizeGB: 50,
				Region: "us-east-1",
				Hours:  HoursPerMonth,
			},
			// Same as S3 Standard methodology (SSD, 3× replication)
			minCarbonGrams: 40,
			maxCarbonGrams: 120,
		},
		{
			name: "100GB table for 1 month in us-east-1",
			config: DynamoDBTableConfig{
				SizeGB: 100,
				Region: "us-east-1",
				Hours:  HoursPerMonth,
			},
			// Should be ~2× of 50GB
			minCarbonGrams: 80,
			maxCarbonGrams: 240,
		},
		{
			name: "100GB table in low-carbon region",
			config: DynamoDBTableConfig{
				SizeGB: 100,
				Region: "eu-north-1",
				Hours:  HoursPerMonth,
			},
			// Much lower due to clean grid
			minCarbonGrams: 0.1,
			maxCarbonGrams: 10,
		},
		{
			name: "zero storage",
			config: DynamoDBTableConfig{
				SizeGB: 0,
				Region: "us-east-1",
				Hours:  HoursPerMonth,
			},
			minCarbonGrams: 0,
			maxCarbonGrams: 0.001,
		},
	}

	e := NewDynamoDBEstimator()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			carbon, ok := e.EstimateCarbonGrams(tt.config)

			require.True(t, ok, "EstimateCarbonGrams should always succeed for DynamoDB")
			assert.GreaterOrEqual(t, carbon, tt.minCarbonGrams)
			assert.LessOrEqual(t, carbon, tt.maxCarbonGrams)
		})
	}
}

// TestDynamoDBEstimator_MatchesS3Standard verifies DynamoDB matches S3 Standard methodology.
func TestDynamoDBEstimator_MatchesS3Standard(t *testing.T) {
	dynamo := NewDynamoDBEstimator()
	s3 := NewS3Estimator()

	carbonDynamo, ok1 := dynamo.EstimateCarbonGramsSimple(100, "us-east-1", 730)
	carbonS3, ok2 := s3.EstimateCarbonGramsSimple("STANDARD", 100, "us-east-1", 730)

	require.True(t, ok1)
	require.True(t, ok2)

	// DynamoDB and S3 Standard should have same carbon (same SSD, 3× replication)
	assert.InDelta(t, carbonS3, carbonDynamo, 1.0, "DynamoDB should match S3 Standard methodology")
}

// TestDynamoDBEstimator_StorageScaling verifies carbon scales with storage.
func TestDynamoDBEstimator_StorageScaling(t *testing.T) {
	e := NewDynamoDBEstimator()

	carbon50, _ := e.EstimateCarbonGramsSimple(50, "us-east-1", 730)
	carbon100, _ := e.EstimateCarbonGramsSimple(100, "us-east-1", 730)

	// 100GB should be exactly 2× of 50GB
	ratio := carbon100 / carbon50
	assert.InDelta(t, 2.0, ratio, 0.01, "Carbon should scale linearly with storage")
}

// TestDynamoDBEstimator_GetBillingDetail verifies billing detail.
func TestDynamoDBEstimator_GetBillingDetail(t *testing.T) {
	e := NewDynamoDBEstimator()

	detail := e.GetBillingDetail(DynamoDBTableConfig{
		SizeGB: 100,
		Region: "us-east-1",
		Hours:  HoursPerMonth,
	})

	assert.Contains(t, detail, "DynamoDB")
	assert.Contains(t, detail, "100")
	assert.Contains(t, detail, "SSD")
	assert.Contains(t, detail, "3×")
}
