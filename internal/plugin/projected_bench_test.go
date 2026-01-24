package plugin

import (
	"context"
	"io"
	"testing"

	"github.com/rs/zerolog"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

// BenchmarkDetectService_Baseline measures the baseline performance of
// detectService() without caching. This establishes the cost of redundant
// calls that the serviceResolver optimization eliminates.
//
// The benchmark simulates the pre-optimization behavior where detectService()
// was called multiple times per request: once during validation, once for
// support checking, and once for cost routing.
//
// Benchmark workflow:
//  1. Defines a set of common resource types (simple and Pulumi formats)
//  2. Resets the benchmark timer to exclude setup overhead
//  3. For each iteration, loops through all resource types
//  4. For each type, calls normalizeResourceType() then detectService() 3 times
//  5. Reports ns/op and allocations for the redundant call pattern
//
// Prerequisites:
//   - Build tag: region_use1 (or other valid region tag)
//   - No external services or environment variables required
//
// Run with: go test -tags=region_use1 -bench=BenchmarkDetectService_Baseline -benchmem ./internal/plugin/...
func BenchmarkDetectService_Baseline(b *testing.B) {
	// Test cases representing common resource types
	testCases := []string{
		"ec2",
		"aws:ec2/instance:Instance",
		"aws:eks/cluster:Cluster",
		"aws:rds/instance:Instance",
		"aws:s3/bucket:Bucket",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, resourceType := range testCases {
			// Simulate the current behavior: multiple calls per request
			normalized := normalizeResourceType(resourceType)
			_ = detectService(normalized)           // call 1: validation
			_ = detectService(normalized)           // call 2: supports check
			_ = detectService(normalized)           // call 3: cost routing
		}
	}
}

// BenchmarkServiceResolver_Cached measures the performance when using
// serviceResolver with lazy initialization. This demonstrates the benefit
// of computing the service type exactly once per resource.
//
// Run with: go test -tags=region_use1 -bench=BenchmarkServiceResolver_Cached -benchmem ./internal/plugin/...
func BenchmarkServiceResolver_Cached(b *testing.B) {
	// Test cases representing common resource types
	testCases := []string{
		"ec2",
		"aws:ec2/instance:Instance",
		"aws:eks/cluster:Cluster",
		"aws:rds/instance:Instance",
		"aws:s3/bucket:Bucket",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, resourceType := range testCases {
			// New behavior: resolver caches the computation
			resolver := newServiceResolver(resourceType)
			_ = resolver.ServiceType() // first call: computes and caches
			_ = resolver.ServiceType() // second call: returns cached
			_ = resolver.ServiceType() // third call: returns cached
		}
	}
}

// BenchmarkServiceResolver_SingleAccess measures the overhead of creating
// a serviceResolver and accessing the service type once. This is the
// typical pattern for requests that succeed validation and proceed to
// cost calculation.
//
// Run with: go test -tags=region_use1 -bench=BenchmarkServiceResolver_SingleAccess -benchmem ./internal/plugin/...
func BenchmarkServiceResolver_SingleAccess(b *testing.B) {
	testCases := []string{
		"ec2",
		"aws:ec2/instance:Instance",
		"aws:eks/cluster:Cluster",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, resourceType := range testCases {
			resolver := newServiceResolver(resourceType)
			_ = resolver.ServiceType()
		}
	}
}

// BenchmarkNormalizeResourceType_Direct measures the cost of normalizeResourceType()
// alone, for comparison with the full resolver overhead.
//
// Run with: go test -tags=region_use1 -bench=BenchmarkNormalizeResourceType_Direct -benchmem ./internal/plugin/...
func BenchmarkNormalizeResourceType_Direct(b *testing.B) {
	testCases := []string{
		"ec2",
		"aws:ec2/instance:Instance",
		"aws:eks/cluster:Cluster",
		"aws:rds/instance:Instance",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, resourceType := range testCases {
			_ = normalizeResourceType(resourceType)
		}
	}
}

// BenchmarkDetectService_Direct measures the cost of detectService() alone,
// for comparison with the full resolver overhead.
//
// Run with: go test -tags=region_use1 -bench=BenchmarkDetectService_Direct -benchmem ./internal/plugin/...
func BenchmarkDetectService_Direct(b *testing.B) {
	// Pre-normalized types to isolate detectService cost
	testCases := []string{
		"ec2",
		"eks",
		"rds",
		"s3",
		"ebs",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, normalizedType := range testCases {
			_ = detectService(normalizedType)
		}
	}
}

// BenchmarkGetProjectedCost_SingleEC2 measures the end-to-end performance of
// GetProjectedCost for a single EC2 instance request.
//
// This benchmark validates SC-002: single request shows reduction from 2-3
// detectService calls to 1 call with the serviceResolver optimization.
//
// Run with: go test -tags=region_use1 -bench=BenchmarkGetProjectedCost_SingleEC2 -benchmem ./internal/plugin/...
func BenchmarkGetProjectedCost_SingleEC2(b *testing.B) {
	mock := newMockPricingClient("us-east-1", "USD")
	logger := zerolog.New(io.Discard).Level(zerolog.Disabled)
	plugin := NewAWSPublicPlugin("us-east-1", "test-version", mock, logger)

	req := &pbc.GetProjectedCostRequest{
		Resource: &pbc.ResourceDescriptor{
			Provider:     "aws",
			ResourceType: "aws:ec2/instance:Instance", // Pulumi format to exercise normalization
			Sku:          "t3.micro",
			Region:       "us-east-1",
		},
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := plugin.GetProjectedCost(ctx, req)
		if err != nil {
			b.Fatalf("GetProjectedCost failed: %v", err)
		}
	}
}

// BenchmarkGetProjectedCost_SingleEBS measures GetProjectedCost for EBS volumes.
//
// Run with: go test -tags=region_use1 -bench=BenchmarkGetProjectedCost_SingleEBS -benchmem ./internal/plugin/...
func BenchmarkGetProjectedCost_SingleEBS(b *testing.B) {
	mock := newMockPricingClient("us-east-1", "USD")
	logger := zerolog.New(io.Discard).Level(zerolog.Disabled)
	plugin := NewAWSPublicPlugin("us-east-1", "test-version", mock, logger)

	req := &pbc.GetProjectedCostRequest{
		Resource: &pbc.ResourceDescriptor{
			Provider:     "aws",
			ResourceType: "aws:ec2/volume:Volume", // Pulumi format (maps to EBS)
			Sku:          "gp3",
			Region:       "us-east-1",
			Tags: map[string]string{
				"size": "100",
			},
		},
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := plugin.GetProjectedCost(ctx, req)
		if err != nil {
			b.Fatalf("GetProjectedCost failed: %v", err)
		}
	}
}
