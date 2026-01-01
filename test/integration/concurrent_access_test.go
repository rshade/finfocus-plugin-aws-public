// Package integration provides integration tests for the carbon estimation feature.
//
// This file contains concurrent access tests verifying thread safety of all
// carbon estimators under high concurrency (100+ goroutines).
//
// Run with: go test ./test/integration/... -v -run Concurrent
package integration

import (
	"sync"
	"testing"

	"github.com/rshade/pulumicost-plugin-aws-public/internal/carbon"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	// numGoroutines is the number of concurrent goroutines for stress testing.
	// This exceeds the 100+ requirement from the task specification.
	numGoroutines = 150

	// numIterations is the number of iterations per goroutine.
	numIterations = 10
)

// TestConcurrentAccess_EC2Estimator verifies thread safety of the EC2 carbon estimator.
//
// This test spawns 150 goroutines, each making 10 calls to the estimator,
// verifying that all calls succeed and return consistent results.
func TestConcurrentAccess_EC2Estimator(t *testing.T) {
	estimator := carbon.NewEstimator()
	estimator.IncludeGPU = true

	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines*numIterations)
	results := make(chan float64, numGoroutines*numIterations)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < numIterations; j++ {
				carbonGrams, ok := estimator.EstimateCarbonGrams(
					"m5.large", "us-east-1", 0.5, 730)
				if !ok {
					errors <- assert.AnError
					return
				}
				results <- carbonGrams
			}
		}()
	}

	wg.Wait()
	close(errors)
	close(results)

	// Verify no errors occurred
	require.Empty(t, errors, "No errors should occur during concurrent access")

	// Verify all results are consistent (same input should produce same output)
	var firstResult float64
	count := 0
	for result := range results {
		if count == 0 {
			firstResult = result
		} else {
			assert.InDelta(t, firstResult, result, 0.001,
				"All results should be identical for same input")
		}
		count++
	}

	// Verify we got all expected results
	assert.Equal(t, numGoroutines*numIterations, count,
		"Should have received all expected results")
}

// TestConcurrentAccess_EBSEstimator verifies thread safety of the EBS carbon estimator.
func TestConcurrentAccess_EBSEstimator(t *testing.T) {
	estimator := carbon.NewEBSEstimator()

	var wg sync.WaitGroup
	successCount := make(chan int, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			success := 0
			for j := 0; j < numIterations; j++ {
				_, ok := estimator.EstimateCarbonGrams(carbon.EBSVolumeConfig{
					VolumeType: "gp3",
					SizeGB:     100,
					Region:     "us-east-1",
					Hours:      730,
				})
				if ok {
					success++
				}
			}
			successCount <- success
		}()
	}

	wg.Wait()
	close(successCount)

	totalSuccess := 0
	for count := range successCount {
		totalSuccess += count
	}

	assert.Equal(t, numGoroutines*numIterations, totalSuccess,
		"All concurrent EBS estimations should succeed")
}

// TestConcurrentAccess_S3Estimator verifies thread safety of the S3 carbon estimator.
func TestConcurrentAccess_S3Estimator(t *testing.T) {
	estimator := carbon.NewS3Estimator()

	var wg sync.WaitGroup
	successCount := make(chan int, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			success := 0
			for j := 0; j < numIterations; j++ {
				_, ok := estimator.EstimateCarbonGrams(carbon.S3StorageConfig{
					StorageClass: "STANDARD",
					SizeGB:       100,
					Region:       "us-east-1",
					Hours:        730,
				})
				if ok {
					success++
				}
			}
			successCount <- success
		}()
	}

	wg.Wait()
	close(successCount)

	totalSuccess := 0
	for count := range successCount {
		totalSuccess += count
	}

	assert.Equal(t, numGoroutines*numIterations, totalSuccess,
		"All concurrent S3 estimations should succeed")
}

// TestConcurrentAccess_LambdaEstimator verifies thread safety of the Lambda carbon estimator.
func TestConcurrentAccess_LambdaEstimator(t *testing.T) {
	estimator := carbon.NewLambdaEstimator()

	var wg sync.WaitGroup
	successCount := make(chan int, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			success := 0
			for j := 0; j < numIterations; j++ {
				_, ok := estimator.EstimateCarbonGrams(carbon.LambdaFunctionConfig{
					MemoryMB:     1792,
					DurationMs:   500,
					Invocations:  1000000,
					Architecture: "x86_64",
					Region:       "us-east-1",
				})
				if ok {
					success++
				}
			}
			successCount <- success
		}()
	}

	wg.Wait()
	close(successCount)

	totalSuccess := 0
	for count := range successCount {
		totalSuccess += count
	}

	assert.Equal(t, numGoroutines*numIterations, totalSuccess,
		"All concurrent Lambda estimations should succeed")
}

// TestConcurrentAccess_RDSEstimator verifies thread safety of the RDS carbon estimator.
func TestConcurrentAccess_RDSEstimator(t *testing.T) {
	estimator := carbon.NewRDSEstimator()

	var wg sync.WaitGroup
	successCount := make(chan int, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			success := 0
			for j := 0; j < numIterations; j++ {
				_, ok := estimator.EstimateCarbonGrams(carbon.RDSInstanceConfig{
					InstanceType:  "db.m5.large",
					Region:        "us-east-1",
					MultiAZ:       false,
					StorageType:   "gp3",
					StorageSizeGB: 100,
					Utilization:   0.5,
					Hours:         730,
				})
				if ok {
					success++
				}
			}
			successCount <- success
		}()
	}

	wg.Wait()
	close(successCount)

	totalSuccess := 0
	for count := range successCount {
		totalSuccess += count
	}

	assert.Equal(t, numGoroutines*numIterations, totalSuccess,
		"All concurrent RDS estimations should succeed")
}

// TestConcurrentAccess_DynamoDBEstimator verifies thread safety of the DynamoDB carbon estimator.
func TestConcurrentAccess_DynamoDBEstimator(t *testing.T) {
	estimator := carbon.NewDynamoDBEstimator()

	var wg sync.WaitGroup
	successCount := make(chan int, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			success := 0
			for j := 0; j < numIterations; j++ {
				_, ok := estimator.EstimateCarbonGrams(carbon.DynamoDBTableConfig{
					SizeGB: 50,
					Region: "us-east-1",
					Hours:  730,
				})
				if ok {
					success++
				}
			}
			successCount <- success
		}()
	}

	wg.Wait()
	close(successCount)

	totalSuccess := 0
	for count := range successCount {
		totalSuccess += count
	}

	assert.Equal(t, numGoroutines*numIterations, totalSuccess,
		"All concurrent DynamoDB estimations should succeed")
}

// TestConcurrentAccess_EKSEstimator verifies thread safety of the EKS carbon estimator.
func TestConcurrentAccess_EKSEstimator(t *testing.T) {
	estimator := carbon.NewEKSEstimator()

	var wg sync.WaitGroup
	successCount := make(chan int, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			success := 0
			for j := 0; j < numIterations; j++ {
				_, ok := estimator.EstimateCarbonGrams(carbon.EKSClusterConfig{
					Region: "us-east-1",
				})
				if ok {
					success++
				}
			}
			successCount <- success
		}()
	}

	wg.Wait()
	close(successCount)

	totalSuccess := 0
	for count := range successCount {
		totalSuccess += count
	}

	assert.Equal(t, numGoroutines*numIterations, totalSuccess,
		"All concurrent EKS estimations should succeed")
}

// TestConcurrentAccess_EmbodiedCarbonEstimator verifies thread safety of the embodied carbon estimator.
func TestConcurrentAccess_EmbodiedCarbonEstimator(t *testing.T) {
	estimator := carbon.NewEmbodiedCarbonEstimator()

	var wg sync.WaitGroup
	successCount := make(chan int, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			success := 0
			for j := 0; j < numIterations; j++ {
				_, ok := estimator.EstimateEmbodiedCarbonKg("m5.large", 1.0)
				if ok {
					success++
				}
			}
			successCount <- success
		}()
	}

	wg.Wait()
	close(successCount)

	totalSuccess := 0
	for count := range successCount {
		totalSuccess += count
	}

	assert.Equal(t, numGoroutines*numIterations, totalSuccess,
		"All concurrent embodied carbon estimations should succeed")
}

// TestConcurrentAccess_MixedEstimators verifies thread safety when multiple
// estimator types are accessed concurrently.
func TestConcurrentAccess_MixedEstimators(t *testing.T) {
	ec2Estimator := carbon.NewEstimator()
	ebsEstimator := carbon.NewEBSEstimator()
	s3Estimator := carbon.NewS3Estimator()
	lambdaEstimator := carbon.NewLambdaEstimator()
	rdsEstimator := carbon.NewRDSEstimator()
	dynamoEstimator := carbon.NewDynamoDBEstimator()
	eksEstimator := carbon.NewEKSEstimator()
	embodiedEstimator := carbon.NewEmbodiedCarbonEstimator()

	var wg sync.WaitGroup
	successCount := make(chan int, numGoroutines*8)

	// Launch goroutines for each estimator type
	for i := 0; i < numGoroutines; i++ {
		// EC2
		wg.Add(1)
		go func() {
			defer wg.Done()
			success := 0
			for j := 0; j < numIterations; j++ {
				if _, ok := ec2Estimator.EstimateCarbonGrams("m5.large", "us-east-1", 0.5, 730); ok {
					success++
				}
			}
			successCount <- success
		}()

		// EBS
		wg.Add(1)
		go func() {
			defer wg.Done()
			success := 0
			for j := 0; j < numIterations; j++ {
				if _, ok := ebsEstimator.EstimateCarbonGrams(carbon.EBSVolumeConfig{
					VolumeType: "gp3", SizeGB: 100, Region: "us-east-1", Hours: 730,
				}); ok {
					success++
				}
			}
			successCount <- success
		}()

		// S3
		wg.Add(1)
		go func() {
			defer wg.Done()
			success := 0
			for j := 0; j < numIterations; j++ {
				if _, ok := s3Estimator.EstimateCarbonGrams(carbon.S3StorageConfig{
					StorageClass: "STANDARD", SizeGB: 100, Region: "us-east-1", Hours: 730,
				}); ok {
					success++
				}
			}
			successCount <- success
		}()

		// Lambda
		wg.Add(1)
		go func() {
			defer wg.Done()
			success := 0
			for j := 0; j < numIterations; j++ {
				if _, ok := lambdaEstimator.EstimateCarbonGrams(carbon.LambdaFunctionConfig{
					MemoryMB: 1792, DurationMs: 500, Invocations: 1000000, Architecture: "x86_64", Region: "us-east-1",
				}); ok {
					success++
				}
			}
			successCount <- success
		}()

		// RDS
		wg.Add(1)
		go func() {
			defer wg.Done()
			success := 0
			for j := 0; j < numIterations; j++ {
				if _, ok := rdsEstimator.EstimateCarbonGrams(carbon.RDSInstanceConfig{
					InstanceType: "db.m5.large", Region: "us-east-1", MultiAZ: false,
					StorageType: "gp3", StorageSizeGB: 100, Utilization: 0.5, Hours: 730,
				}); ok {
					success++
				}
			}
			successCount <- success
		}()

		// DynamoDB
		wg.Add(1)
		go func() {
			defer wg.Done()
			success := 0
			for j := 0; j < numIterations; j++ {
				if _, ok := dynamoEstimator.EstimateCarbonGrams(carbon.DynamoDBTableConfig{
					SizeGB: 50, Region: "us-east-1", Hours: 730,
				}); ok {
					success++
				}
			}
			successCount <- success
		}()

		// EKS
		wg.Add(1)
		go func() {
			defer wg.Done()
			success := 0
			for j := 0; j < numIterations; j++ {
				if _, ok := eksEstimator.EstimateCarbonGrams(carbon.EKSClusterConfig{
					Region: "us-east-1",
				}); ok {
					success++
				}
			}
			successCount <- success
		}()

		// Embodied
		wg.Add(1)
		go func() {
			defer wg.Done()
			success := 0
			for j := 0; j < numIterations; j++ {
				if _, ok := embodiedEstimator.EstimateEmbodiedCarbonKg("m5.large", 1.0); ok {
					success++
				}
			}
			successCount <- success
		}()
	}

	wg.Wait()
	close(successCount)

	totalSuccess := 0
	for count := range successCount {
		totalSuccess += count
	}

	expectedTotal := numGoroutines * numIterations * 8 // 8 estimator types
	assert.Equal(t, expectedTotal, totalSuccess,
		"All concurrent mixed estimations should succeed")
}

// TestConcurrentAccess_GridFactorLookup verifies thread safety of grid factor lookups.
func TestConcurrentAccess_GridFactorLookup(t *testing.T) {
	regions := []string{
		"us-east-1", "us-west-2", "eu-west-1", "eu-north-1",
		"ap-southeast-1", "ap-south-1", "sa-east-1", "unknown-region",
	}

	var wg sync.WaitGroup
	results := make(chan float64, numGoroutines*numIterations*len(regions))

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < numIterations; j++ {
				for _, region := range regions {
					factor := carbon.GetGridFactor(region)
					results <- factor
				}
			}
		}()
	}

	wg.Wait()
	close(results)

	count := 0
	for range results {
		count++
	}

	expectedCount := numGoroutines * numIterations * len(regions)
	assert.Equal(t, expectedCount, count,
		"All concurrent grid factor lookups should complete")
}
