// Package benchmark provides performance benchmarks for the carbon estimation feature.
//
// These benchmarks verify that all carbon estimators meet the <100ms latency target
// as specified in the task requirements.
//
// Run with: go test ./test/benchmark/... -bench=. -benchmem
package benchmark

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/rshade/finfocus-plugin-aws-public/internal/carbon"
)

const (
	// maxLatencyMs is the maximum acceptable latency in milliseconds.
	// All estimators must complete within this threshold.
	maxLatencyMs = 100
)

// BenchmarkEC2Estimator measures the performance of EC2 carbon estimation.
func BenchmarkEC2Estimator(b *testing.B) {
	estimator := carbon.NewEstimator()
	estimator.IncludeGPU = true

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		estimator.EstimateCarbonGrams("m5.large", "us-east-1", 0.5, 730)
	}
}

// BenchmarkEC2Estimator_GPU measures the performance of GPU instance carbon estimation.
func BenchmarkEC2Estimator_GPU(b *testing.B) {
	estimator := carbon.NewEstimator()
	estimator.IncludeGPU = true

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		estimator.EstimateCarbonGrams("p4d.24xlarge", "us-east-1", 0.5, 730)
	}
}

// BenchmarkEC2Estimator_WithBreakdown measures performance of breakdown calculation.
func BenchmarkEC2Estimator_WithBreakdown(b *testing.B) {
	estimator := carbon.NewEstimator()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		estimator.EstimateCarbonGramsWithBreakdown("p4d.24xlarge", "us-east-1", 0.5, 730)
	}
}

// BenchmarkEBSEstimator measures the performance of EBS carbon estimation.
func BenchmarkEBSEstimator(b *testing.B) {
	estimator := carbon.NewEBSEstimator()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		estimator.EstimateCarbonGrams(carbon.EBSVolumeConfig{
			VolumeType: "gp3",
			SizeGB:     100,
			Region:     "us-east-1",
			Hours:      730,
		})
	}
}

// BenchmarkS3Estimator measures the performance of S3 carbon estimation.
func BenchmarkS3Estimator(b *testing.B) {
	estimator := carbon.NewS3Estimator()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		estimator.EstimateCarbonGrams(carbon.S3StorageConfig{
			StorageClass: "STANDARD",
			SizeGB:       100,
			Region:       "us-east-1",
			Hours:        730,
		})
	}
}

// BenchmarkLambdaEstimator measures the performance of Lambda carbon estimation.
func BenchmarkLambdaEstimator(b *testing.B) {
	estimator := carbon.NewLambdaEstimator()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		estimator.EstimateCarbonGrams(carbon.LambdaFunctionConfig{
			MemoryMB:     1792,
			DurationMs:   500,
			Invocations:  1000000,
			Architecture: "x86_64",
			Region:       "us-east-1",
		})
	}
}

// BenchmarkRDSEstimator measures the performance of RDS carbon estimation.
func BenchmarkRDSEstimator(b *testing.B) {
	estimator := carbon.NewRDSEstimator()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		estimator.EstimateCarbonGrams(carbon.RDSInstanceConfig{
			InstanceType:  "db.m5.large",
			Region:        "us-east-1",
			MultiAZ:       false,
			StorageType:   "gp3",
			StorageSizeGB: 100,
			Utilization:   0.5,
			Hours:         730,
		})
	}
}

// BenchmarkRDSEstimator_MultiAZ measures the performance of Multi-AZ RDS estimation.
func BenchmarkRDSEstimator_MultiAZ(b *testing.B) {
	estimator := carbon.NewRDSEstimator()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		estimator.EstimateCarbonGrams(carbon.RDSInstanceConfig{
			InstanceType:  "db.m5.large",
			Region:        "us-east-1",
			MultiAZ:       true,
			StorageType:   "gp3",
			StorageSizeGB: 100,
			Utilization:   0.5,
			Hours:         730,
		})
	}
}

// BenchmarkDynamoDBEstimator measures the performance of DynamoDB carbon estimation.
func BenchmarkDynamoDBEstimator(b *testing.B) {
	estimator := carbon.NewDynamoDBEstimator()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		estimator.EstimateCarbonGrams(carbon.DynamoDBTableConfig{
			SizeGB: 50,
			Region: "us-east-1",
			Hours:  730,
		})
	}
}

// BenchmarkEKSEstimator measures the performance of EKS carbon estimation.
func BenchmarkEKSEstimator(b *testing.B) {
	estimator := carbon.NewEKSEstimator()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		estimator.EstimateCarbonGrams(carbon.EKSClusterConfig{
			Region: "us-east-1",
		})
	}
}

// BenchmarkEmbodiedCarbonEstimator measures the performance of embodied carbon estimation.
func BenchmarkEmbodiedCarbonEstimator(b *testing.B) {
	estimator := carbon.NewEmbodiedCarbonEstimator()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		estimator.EstimateEmbodiedCarbonKg("m5.large", 1.0)
	}
}

// BenchmarkGetGridFactor measures the performance of grid factor lookup.
func BenchmarkGetGridFactor(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		carbon.GetGridFactor("us-east-1")
	}
}

// BenchmarkGetGridFactor_Unknown measures grid factor lookup for unknown region.
func BenchmarkGetGridFactor_Unknown(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		carbon.GetGridFactor("unknown-region")
	}
}

// TestLatencyRequirement_EC2 verifies EC2 estimator meets <100ms latency.
func TestLatencyRequirement_EC2(t *testing.T) {
	estimator := carbon.NewEstimator()
	estimator.IncludeGPU = true

	start := time.Now()
	estimator.EstimateCarbonGrams("p4d.24xlarge", "us-east-1", 0.5, 730)
	elapsed := time.Since(start)

	if elapsed.Milliseconds() > maxLatencyMs {
		t.Errorf("EC2 estimation took %v, exceeds %dms limit", elapsed, maxLatencyMs)
	}
}

// TestLatencyRequirement_EBS verifies EBS estimator meets <100ms latency.
func TestLatencyRequirement_EBS(t *testing.T) {
	estimator := carbon.NewEBSEstimator()

	start := time.Now()
	estimator.EstimateCarbonGrams(carbon.EBSVolumeConfig{
		VolumeType: "gp3",
		SizeGB:     100,
		Region:     "us-east-1",
		Hours:      730,
	})
	elapsed := time.Since(start)

	if elapsed.Milliseconds() > maxLatencyMs {
		t.Errorf("EBS estimation took %v, exceeds %dms limit", elapsed, maxLatencyMs)
	}
}

// TestLatencyRequirement_S3 verifies S3 estimator meets <100ms latency.
func TestLatencyRequirement_S3(t *testing.T) {
	estimator := carbon.NewS3Estimator()

	start := time.Now()
	estimator.EstimateCarbonGrams(carbon.S3StorageConfig{
		StorageClass: "STANDARD",
		SizeGB:       100,
		Region:       "us-east-1",
		Hours:        730,
	})
	elapsed := time.Since(start)

	if elapsed.Milliseconds() > maxLatencyMs {
		t.Errorf("S3 estimation took %v, exceeds %dms limit", elapsed, maxLatencyMs)
	}
}

// TestLatencyRequirement_Lambda verifies Lambda estimator meets <100ms latency.
func TestLatencyRequirement_Lambda(t *testing.T) {
	estimator := carbon.NewLambdaEstimator()

	start := time.Now()
	estimator.EstimateCarbonGrams(carbon.LambdaFunctionConfig{
		MemoryMB:     1792,
		DurationMs:   500,
		Invocations:  1000000,
		Architecture: "x86_64",
		Region:       "us-east-1",
	})
	elapsed := time.Since(start)

	if elapsed.Milliseconds() > maxLatencyMs {
		t.Errorf("Lambda estimation took %v, exceeds %dms limit", elapsed, maxLatencyMs)
	}
}

// TestLatencyRequirement_RDS verifies RDS estimator meets <100ms latency.
func TestLatencyRequirement_RDS(t *testing.T) {
	estimator := carbon.NewRDSEstimator()

	start := time.Now()
	estimator.EstimateCarbonGrams(carbon.RDSInstanceConfig{
		InstanceType:  "db.m5.large",
		Region:        "us-east-1",
		MultiAZ:       true,
		StorageType:   "gp3",
		StorageSizeGB: 100,
		Utilization:   0.5,
		Hours:         730,
	})
	elapsed := time.Since(start)

	if elapsed.Milliseconds() > maxLatencyMs {
		t.Errorf("RDS estimation took %v, exceeds %dms limit", elapsed, maxLatencyMs)
	}
}

// TestLatencyRequirement_DynamoDB verifies DynamoDB estimator meets <100ms latency.
func TestLatencyRequirement_DynamoDB(t *testing.T) {
	estimator := carbon.NewDynamoDBEstimator()

	start := time.Now()
	estimator.EstimateCarbonGrams(carbon.DynamoDBTableConfig{
		SizeGB: 50,
		Region: "us-east-1",
		Hours:  730,
	})
	elapsed := time.Since(start)

	if elapsed.Milliseconds() > maxLatencyMs {
		t.Errorf("DynamoDB estimation took %v, exceeds %dms limit", elapsed, maxLatencyMs)
	}
}

// TestLatencyRequirement_EKS verifies EKS estimator meets <100ms latency.
func TestLatencyRequirement_EKS(t *testing.T) {
	estimator := carbon.NewEKSEstimator()

	start := time.Now()
	estimator.EstimateCarbonGrams(carbon.EKSClusterConfig{
		Region: "us-east-1",
	})
	elapsed := time.Since(start)

	if elapsed.Milliseconds() > maxLatencyMs {
		t.Errorf("EKS estimation took %v, exceeds %dms limit", elapsed, maxLatencyMs)
	}
}

// TestLatencyRequirement_EmbodiedCarbon verifies embodied carbon estimator meets <100ms latency.
func TestLatencyRequirement_EmbodiedCarbon(t *testing.T) {
	estimator := carbon.NewEmbodiedCarbonEstimator()

	start := time.Now()
	estimator.EstimateEmbodiedCarbonKg("m5.large", 1.0)
	elapsed := time.Since(start)

	if elapsed.Milliseconds() > maxLatencyMs {
		t.Errorf("Embodied carbon estimation took %v, exceeds %dms limit", elapsed, maxLatencyMs)
	}
}

// TestLatencyRequirement_AllEstimators verifies all estimators in sequence.
func TestLatencyRequirement_AllEstimators(t *testing.T) {
	tests := []struct {
		name string
		fn   func()
	}{
		{"EC2", func() {
			e := carbon.NewEstimator()
			e.IncludeGPU = true
			e.EstimateCarbonGrams("p4d.24xlarge", "us-east-1", 0.5, 730)
		}},
		{"EBS", func() {
			e := carbon.NewEBSEstimator()
			e.EstimateCarbonGrams(carbon.EBSVolumeConfig{
				VolumeType: "gp3", SizeGB: 100, Region: "us-east-1", Hours: 730,
			})
		}},
		{"S3", func() {
			e := carbon.NewS3Estimator()
			e.EstimateCarbonGrams(carbon.S3StorageConfig{
				StorageClass: "STANDARD", SizeGB: 100, Region: "us-east-1", Hours: 730,
			})
		}},
		{"Lambda", func() {
			e := carbon.NewLambdaEstimator()
			e.EstimateCarbonGrams(carbon.LambdaFunctionConfig{
				MemoryMB: 1792, DurationMs: 500, Invocations: 1000000,
				Architecture: "x86_64", Region: "us-east-1",
			})
		}},
		{"RDS", func() {
			e := carbon.NewRDSEstimator()
			e.EstimateCarbonGrams(carbon.RDSInstanceConfig{
				InstanceType: "db.m5.large", Region: "us-east-1", MultiAZ: true,
				StorageType: "gp3", StorageSizeGB: 100, Utilization: 0.5, Hours: 730,
			})
		}},
		{"DynamoDB", func() {
			e := carbon.NewDynamoDBEstimator()
			e.EstimateCarbonGrams(carbon.DynamoDBTableConfig{
				SizeGB: 50, Region: "us-east-1", Hours: 730,
			})
		}},
		{"EKS", func() {
			e := carbon.NewEKSEstimator()
			e.EstimateCarbonGrams(carbon.EKSClusterConfig{Region: "us-east-1"})
		}},
		{"EmbodiedCarbon", func() {
			e := carbon.NewEmbodiedCarbonEstimator()
			e.EstimateEmbodiedCarbonKg("m5.large", 1.0)
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start := time.Now()
			tt.fn()
			elapsed := time.Since(start)

			if elapsed.Milliseconds() > maxLatencyMs {
				t.Errorf("%s estimation took %v, exceeds %dms limit", tt.name, elapsed, maxLatencyMs)
			} else {
				t.Logf("%s estimation completed in %v", tt.name, elapsed)
			}
		})
	}
}

// TestConcurrentLatency verifies estimators work correctly under concurrent load.
func TestConcurrentLatency(t *testing.T) {
	const goroutines = 150
	var wg sync.WaitGroup
	errors := make(chan error, goroutines)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// Run a representative estimator call
			start := time.Now()
			e := carbon.NewEstimator()
			e.EstimateCarbonGrams("m5.large", "us-east-1", 0.5, 730)
			if time.Since(start).Milliseconds() > maxLatencyMs {
				errors <- fmt.Errorf("exceeded latency under concurrent load")
			}
		}()
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Error(err)
	}
}
