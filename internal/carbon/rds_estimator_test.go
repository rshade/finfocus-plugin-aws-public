package carbon

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRDSEstimator_EstimateCarbonGrams verifies RDS carbon estimation.
func TestRDSEstimator_EstimateCarbonGrams(t *testing.T) {
	tests := []struct {
		name           string
		config         RDSInstanceConfig
		wantOK         bool
		minCarbonGrams float64
		maxCarbonGrams float64
	}{
		{
			name: "db.m5.large Single-AZ with 100GB storage",
			config: RDSInstanceConfig{
				InstanceType:  "db.m5.large",
				Region:        "us-east-1",
				MultiAZ:       false,
				StorageType:   "gp3",
				StorageSizeGB: 100,
				Utilization:   0.5,
				Hours: HoursPerMonth,
			},
			wantOK:         true,
			minCarbonGrams: 3000,  // CCF: m5.large ~3.5kg CO2e/month
			maxCarbonGrams: 5000,
		},
		{
			name: "db.m5.large Multi-AZ with 100GB storage (2× multiplier)",
			config: RDSInstanceConfig{
				InstanceType:  "db.m5.large",
				Region:        "us-east-1",
				MultiAZ:       true,
				StorageType:   "gp3",
				StorageSizeGB: 100,
				Utilization:   0.5,
				Hours: HoursPerMonth,
			},
			wantOK:         true,
			minCarbonGrams: 6000,  // 2× Single-AZ ~7kg CO2e/month
			maxCarbonGrams: 10000,
		},
		{
			name: "db.r5.xlarge with 500GB io2 storage",
			config: RDSInstanceConfig{
				InstanceType:  "db.r5.xlarge",
				Region:        "us-east-1",
				MultiAZ:       false,
				StorageType:   "io2",
				StorageSizeGB: 500,
				Utilization:   0.5,
				Hours: HoursPerMonth,
			},
			wantOK:         true,
			minCarbonGrams: 10000, // r5.xlarge 4vCPU + 500GB storage
			maxCarbonGrams: 20000,
		},
		{
			name: "unknown RDS instance type",
			config: RDSInstanceConfig{
				InstanceType:  "db.unknown.size",
				Region:        "us-east-1",
				MultiAZ:       false,
				StorageType:   "gp3",
				StorageSizeGB: 100,
				Utilization:   0.5,
				Hours: HoursPerMonth,
			},
			wantOK:         false,
			minCarbonGrams: 0,
			maxCarbonGrams: 0,
		},
	}

	e := NewRDSEstimator()

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

// TestRDSEstimator_MultiAZDoubles verifies Multi-AZ doubles carbon.
func TestRDSEstimator_MultiAZDoubles(t *testing.T) {
	e := NewRDSEstimator()

	configSingleAZ := RDSInstanceConfig{
		InstanceType:  "db.m5.large",
		Region:        "us-east-1",
		MultiAZ:       false,
		StorageType:   "gp3",
		StorageSizeGB: 100,
		Utilization:   0.5,
		Hours: HoursPerMonth,
	}

	configMultiAZ := configSingleAZ
	configMultiAZ.MultiAZ = true

	carbonSingle, ok1 := e.EstimateCarbonGrams(configSingleAZ)
	carbonMulti, ok2 := e.EstimateCarbonGrams(configMultiAZ)

	require.True(t, ok1)
	require.True(t, ok2)

	// Multi-AZ should be exactly 2× Single-AZ
	ratio := carbonMulti / carbonSingle
	assert.InDelta(t, 2.0, ratio, 0.01, "Multi-AZ should be 2× Single-AZ")
}

// TestRDSEstimator_Breakdown verifies compute and storage breakdown.
func TestRDSEstimator_Breakdown(t *testing.T) {
	e := NewRDSEstimator()

	config := RDSInstanceConfig{
		InstanceType:  "db.m5.large",
		Region:        "us-east-1",
		MultiAZ:       false,
		StorageType:   "gp3",
		StorageSizeGB: 100,
		Utilization:   0.5,
		Hours: HoursPerMonth,
	}

	computeCarbon, storageCarbon, ok := e.EstimateCarbonGramsWithBreakdown(config)

	require.True(t, ok)
	assert.Greater(t, computeCarbon, 0.0, "Compute carbon should be positive")
	assert.Greater(t, storageCarbon, 0.0, "Storage carbon should be positive")

	// Total should match the standard method
	totalCarbon, _ := e.EstimateCarbonGrams(config)
	assert.InDelta(t, totalCarbon, computeCarbon+storageCarbon, 0.01)
}

// TestRDSToEC2InstanceType verifies RDS to EC2 type conversion.
func TestRDSToEC2InstanceType(t *testing.T) {
	tests := []struct {
		rdsType  string
		wantEC2  string
	}{
		{"db.m5.large", "m5.large"},
		{"db.r5.xlarge", "r5.xlarge"},
		{"db.t3.micro", "t3.micro"},
		{"m5.large", "m5.large"}, // Already EC2 format
	}

	for _, tt := range tests {
		t.Run(tt.rdsType, func(t *testing.T) {
			got := rdsToEC2InstanceType(tt.rdsType)
			assert.Equal(t, tt.wantEC2, got)
		})
	}
}
