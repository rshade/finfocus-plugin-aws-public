//go:build integration

// Package integration provides integration tests for the carbon estimation feature.
//
// These tests verify the complete flow from gRPC request to response including
// carbon footprint metrics (ImpactMetrics) for all supported AWS services.
//
// Run with: go test -tags=integration ./test/integration/... -v
package integration

import (
	"testing"

	"github.com/rshade/pulumicost-plugin-aws-public/internal/carbon"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCarbonEstimation_EC2_WithGPU verifies end-to-end carbon estimation
// for GPU instances including both CPU and GPU power consumption.
func TestCarbonEstimation_EC2_WithGPU(t *testing.T) {
	estimator := carbon.NewEstimator()
	estimator.IncludeGPU = true

	tests := []struct {
		name           string
		instanceType   string
		region         string
		utilization    float64
		hours          float64
		minCarbonGrams float64
		maxCarbonGrams float64
		hasGPU         bool
	}{
		{
			name:           "p4d.24xlarge with 8x A100 GPUs in us-east-1",
			instanceType:   "p4d.24xlarge",
			region:         "us-east-1",
			utilization:    0.5,
			hours:          730,
			minCarbonGrams: 500000, // High due to 96 vCPUs + 8x A100
			maxCarbonGrams: 20000000,
			hasGPU:         true,
		},
		{
			name:           "g4dn.xlarge with 1x T4 GPU in us-east-1",
			instanceType:   "g4dn.xlarge",
			region:         "us-east-1",
			utilization:    0.5,
			hours:          730,
			minCarbonGrams: 10000,
			maxCarbonGrams: 500000,
			hasGPU:         true,
		},
		{
			name:           "t3.micro (no GPU) in us-east-1",
			instanceType:   "t3.micro",
			region:         "us-east-1",
			utilization:    0.5,
			hours:          730,
			minCarbonGrams: 100,
			maxCarbonGrams: 5000,
			hasGPU:         false,
		},
		{
			name:           "m5.large in eu-north-1 (clean grid)",
			instanceType:   "m5.large",
			region:         "eu-north-1",
			utilization:    0.5,
			hours:          730,
			minCarbonGrams: 10, // Very low due to Sweden's clean grid
			maxCarbonGrams: 1000,
			hasGPU:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			carbonGrams, ok := estimator.EstimateCarbonGrams(
				tt.instanceType, tt.region, tt.utilization, tt.hours)

			require.True(t, ok, "Carbon estimation should succeed for %s", tt.instanceType)
			assert.GreaterOrEqual(t, carbonGrams, tt.minCarbonGrams,
				"Carbon should be >= %f grams", tt.minCarbonGrams)
			assert.LessOrEqual(t, carbonGrams, tt.maxCarbonGrams,
				"Carbon should be <= %f grams", tt.maxCarbonGrams)

			// Verify GPU breakdown if applicable
			if tt.hasGPU {
				cpuCarbon, gpuCarbon, breakdownOK := estimator.EstimateCarbonGramsWithBreakdown(
					tt.instanceType, tt.region, tt.utilization, tt.hours)
				require.True(t, breakdownOK)
				assert.Greater(t, gpuCarbon, 0.0, "GPU carbon should be positive for GPU instance")
				assert.Greater(t, cpuCarbon, 0.0, "CPU carbon should be positive")
				assert.InDelta(t, carbonGrams, cpuCarbon+gpuCarbon, 1.0,
					"Total should equal CPU + GPU")
			}
		})
	}
}

// TestCarbonEstimation_EBS verifies carbon estimation for EBS volumes.
func TestCarbonEstimation_EBS(t *testing.T) {
	estimator := carbon.NewEBSEstimator()

	tests := []struct {
		name           string
		volumeType     string
		sizeGB         float64
		region         string
		hours          float64
		minCarbonGrams float64
		maxCarbonGrams float64
	}{
		{
			name:           "100GB gp3 in us-east-1",
			volumeType:     "gp3",
			sizeGB:         100,
			region:         "us-east-1",
			hours:          730,
			minCarbonGrams: 10,
			maxCarbonGrams: 200,
		},
		{
			name:           "1000GB io2 in us-east-1",
			volumeType:     "io2",
			sizeGB:         1000,
			region:         "us-east-1",
			hours:          730,
			minCarbonGrams: 100,
			maxCarbonGrams: 2000,
		},
		{
			name:           "500GB st1 HDD in us-east-1",
			volumeType:     "st1",
			sizeGB:         500,
			region:         "us-east-1",
			hours:          730,
			minCarbonGrams: 50,
			maxCarbonGrams: 500,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			carbonGrams, ok := estimator.EstimateCarbonGrams(carbon.EBSVolumeConfig{
				VolumeType: tt.volumeType,
				SizeGB:     tt.sizeGB,
				Region:     tt.region,
				Hours:      tt.hours,
			})

			require.True(t, ok, "EBS carbon estimation should succeed")
			assert.GreaterOrEqual(t, carbonGrams, tt.minCarbonGrams)
			assert.LessOrEqual(t, carbonGrams, tt.maxCarbonGrams)
		})
	}
}

// TestCarbonEstimation_S3 verifies carbon estimation for S3 storage.
func TestCarbonEstimation_S3(t *testing.T) {
	estimator := carbon.NewS3Estimator()

	tests := []struct {
		name           string
		storageClass   string
		sizeGB         float64
		region         string
		hours          float64
		minCarbonGrams float64
		maxCarbonGrams float64
	}{
		{
			name:           "100GB STANDARD in us-east-1",
			storageClass:   "STANDARD",
			sizeGB:         100,
			region:         "us-east-1",
			hours:          730,
			minCarbonGrams: 10,
			maxCarbonGrams: 500,
		},
		{
			name:           "1TB GLACIER in eu-north-1",
			storageClass:   "GLACIER",
			sizeGB:         1000,
			region:         "eu-north-1",
			hours:          730,
			minCarbonGrams: 1,
			maxCarbonGrams: 100, // Low due to clean grid
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			carbonGrams, ok := estimator.EstimateCarbonGrams(carbon.S3StorageConfig{
				StorageClass: tt.storageClass,
				SizeGB:       tt.sizeGB,
				Region:       tt.region,
				Hours:        tt.hours,
			})

			require.True(t, ok, "S3 carbon estimation should succeed")
			assert.GreaterOrEqual(t, carbonGrams, tt.minCarbonGrams)
			assert.LessOrEqual(t, carbonGrams, tt.maxCarbonGrams)
		})
	}
}

// TestCarbonEstimation_Lambda verifies carbon estimation for Lambda functions.
func TestCarbonEstimation_Lambda(t *testing.T) {
	estimator := carbon.NewLambdaEstimator()

	tests := []struct {
		name           string
		memoryMB       int
		durationMs     int
		invocations    int64
		architecture   string
		region         string
		minCarbonGrams float64
		maxCarbonGrams float64
	}{
		{
			name:           "1792MB x86_64 with 1M invocations",
			memoryMB:       1792,
			durationMs:     500,
			invocations:    1000000,
			architecture:   "x86_64",
			region:         "us-east-1",
			minCarbonGrams: 1,
			maxCarbonGrams: 1000,
		},
		{
			name:           "1792MB arm64 with 1M invocations (more efficient)",
			memoryMB:       1792,
			durationMs:     500,
			invocations:    1000000,
			architecture:   "arm64",
			region:         "us-east-1",
			minCarbonGrams: 0.5,
			maxCarbonGrams: 800, // 20% more efficient
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			carbonGrams, ok := estimator.EstimateCarbonGrams(carbon.LambdaFunctionConfig{
				MemoryMB:     tt.memoryMB,
				DurationMs:   tt.durationMs,
				Invocations:  tt.invocations,
				Architecture: tt.architecture,
				Region:       tt.region,
			})

			require.True(t, ok, "Lambda carbon estimation should succeed")
			assert.GreaterOrEqual(t, carbonGrams, tt.minCarbonGrams)
			assert.LessOrEqual(t, carbonGrams, tt.maxCarbonGrams)
		})
	}
}

// TestCarbonEstimation_RDS verifies carbon estimation for RDS instances.
func TestCarbonEstimation_RDS(t *testing.T) {
	estimator := carbon.NewRDSEstimator()

	tests := []struct {
		name           string
		instanceType   string
		region         string
		multiAZ        bool
		storageType    string
		storageSizeGB  float64
		utilization    float64
		hours          float64
		minCarbonGrams float64
		maxCarbonGrams float64
	}{
		{
			name:           "db.m5.large single-AZ in us-east-1",
			instanceType:   "db.m5.large",
			region:         "us-east-1",
			multiAZ:        false,
			storageType:    "gp3",
			storageSizeGB:  100,
			utilization:    0.5,
			hours:          730,
			minCarbonGrams: 1000,
			maxCarbonGrams: 50000,
		},
		{
			name:           "db.m5.large Multi-AZ in us-east-1 (2x carbon)",
			instanceType:   "db.m5.large",
			region:         "us-east-1",
			multiAZ:        true,
			storageType:    "gp3",
			storageSizeGB:  100,
			utilization:    0.5,
			hours:          730,
			minCarbonGrams: 2000, // Should be ~2x single-AZ
			maxCarbonGrams: 100000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			carbonGrams, ok := estimator.EstimateCarbonGrams(carbon.RDSInstanceConfig{
				InstanceType:  tt.instanceType,
				Region:        tt.region,
				MultiAZ:       tt.multiAZ,
				StorageType:   tt.storageType,
				StorageSizeGB: tt.storageSizeGB,
				Utilization:   tt.utilization,
				Hours:         tt.hours,
			})

			require.True(t, ok, "RDS carbon estimation should succeed")
			assert.GreaterOrEqual(t, carbonGrams, tt.minCarbonGrams)
			assert.LessOrEqual(t, carbonGrams, tt.maxCarbonGrams)
		})
	}

	// Verify Multi-AZ doubles the carbon
	t.Run("Multi-AZ should approximately double carbon", func(t *testing.T) {
		singleAZ, _ := estimator.EstimateCarbonGrams(carbon.RDSInstanceConfig{
			InstanceType:  "db.m5.large",
			Region:        "us-east-1",
			MultiAZ:       false,
			StorageType:   "gp3",
			StorageSizeGB: 100,
			Utilization:   0.5,
			Hours:         730,
		})

		multiAZ, _ := estimator.EstimateCarbonGrams(carbon.RDSInstanceConfig{
			InstanceType:  "db.m5.large",
			Region:        "us-east-1",
			MultiAZ:       true,
			StorageType:   "gp3",
			StorageSizeGB: 100,
			Utilization:   0.5,
			Hours:         730,
		})

		ratio := multiAZ / singleAZ
		assert.InDelta(t, 2.0, ratio, 0.1,
			"Multi-AZ should be ~2x single-AZ (got %.2f)", ratio)
	})
}

// TestCarbonEstimation_DynamoDB verifies carbon estimation for DynamoDB tables.
func TestCarbonEstimation_DynamoDB(t *testing.T) {
	estimator := carbon.NewDynamoDBEstimator()

	tests := []struct {
		name           string
		sizeGB         float64
		region         string
		hours          float64
		minCarbonGrams float64
		maxCarbonGrams float64
	}{
		{
			name:           "50GB table in us-east-1",
			sizeGB:         50,
			region:         "us-east-1",
			hours:          730,
			minCarbonGrams: 10,
			maxCarbonGrams: 200,
		},
		{
			name:           "500GB table in ap-south-1 (higher carbon grid)",
			sizeGB:         500,
			region:         "ap-south-1",
			hours:          730,
			minCarbonGrams: 100,
			maxCarbonGrams: 2000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			carbonGrams, ok := estimator.EstimateCarbonGrams(carbon.DynamoDBTableConfig{
				SizeGB: tt.sizeGB,
				Region: tt.region,
				Hours:  tt.hours,
			})

			require.True(t, ok, "DynamoDB carbon estimation should succeed")
			assert.GreaterOrEqual(t, carbonGrams, tt.minCarbonGrams)
			assert.LessOrEqual(t, carbonGrams, tt.maxCarbonGrams)
		})
	}
}

// TestCarbonEstimation_EKS verifies EKS returns zero carbon for control plane.
func TestCarbonEstimation_EKS(t *testing.T) {
	estimator := carbon.NewEKSEstimator()

	carbonGrams, ok := estimator.EstimateCarbonGrams(carbon.EKSClusterConfig{
		Region: "us-east-1",
	})

	require.True(t, ok, "EKS carbon estimation should succeed")
	assert.Equal(t, 0.0, carbonGrams,
		"EKS control plane should return 0 carbon (shared infrastructure)")
}

// TestCarbonEstimation_RegionalVariation verifies that different regions
// produce significantly different carbon estimates due to grid factors.
func TestCarbonEstimation_RegionalVariation(t *testing.T) {
	estimator := carbon.NewEstimator()

	// Same instance type, different regions
	instanceType := "m5.large"
	utilization := 0.5
	hours := 730.0

	carbonUSEast, _ := estimator.EstimateCarbonGrams(instanceType, "us-east-1", utilization, hours)
	carbonEUNorth, _ := estimator.EstimateCarbonGrams(instanceType, "eu-north-1", utilization, hours)
	carbonAPSouth, _ := estimator.EstimateCarbonGrams(instanceType, "ap-south-1", utilization, hours)

	// EU North (Sweden) should be much lower than US East (Virginia)
	assert.Greater(t, carbonUSEast, carbonEUNorth*10,
		"US East should have >10x more carbon than EU North")

	// AP South (Mumbai) should be higher than US East (coal-heavy grid)
	assert.Greater(t, carbonAPSouth, carbonUSEast,
		"AP South (Mumbai) should have more carbon than US East")
}

// TestCarbonEstimation_EmbodiedCarbon verifies embodied carbon calculation.
func TestCarbonEstimation_EmbodiedCarbon(t *testing.T) {
	estimator := carbon.NewEmbodiedCarbonEstimator()

	tests := []struct {
		name              string
		instanceType      string
		months            float64
		minEmbodiedCarbKg float64
		maxEmbodiedCarbKg float64
	}{
		{
			name:              "m5.large for 1 month",
			instanceType:      "m5.large",
			months:            1,
			minEmbodiedCarbKg: 0.3,
			maxEmbodiedCarbKg: 1.0,
		},
		{
			name:              "m5.24xlarge for 1 month (full server)",
			instanceType:      "m5.24xlarge",
			months:            1,
			minEmbodiedCarbKg: 15,
			maxEmbodiedCarbKg: 30,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			carbonKg, ok := estimator.EstimateEmbodiedCarbonKg(tt.instanceType, tt.months)

			require.True(t, ok, "Embodied carbon estimation should succeed")
			assert.GreaterOrEqual(t, carbonKg, tt.minEmbodiedCarbKg)
			assert.LessOrEqual(t, carbonKg, tt.maxEmbodiedCarbKg)
		})
	}
}
