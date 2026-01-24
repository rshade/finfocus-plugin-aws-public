package plugin

import (
	"context"
	"io"
	"testing"

	"github.com/rs/zerolog"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

// BenchmarkGetRecommendations_100Resources measures the performance of
// GetRecommendations processing a batch of 100 EC2 resources.
//
// This benchmark validates SC-001: detectService call count reduction from
// ~300 calls (pre-optimization) to ~100 calls (with serviceResolver).
//
// The optimization creates one serviceResolver per resource in the loop,
// caching the normalized type and service type for reuse within that
// resource's processing lifecycle.
//
// Run with: go test -tags=region_use1 -bench=BenchmarkGetRecommendations_100Resources -benchmem ./internal/plugin/...
func BenchmarkGetRecommendations_100Resources(b *testing.B) {
	mock := newMockPricingClient("us-east-1", "USD")
	logger := zerolog.New(io.Discard).Level(zerolog.Disabled)
	plugin := NewAWSPublicPlugin("us-east-1", "test-version", mock, logger)

	// Create 100 EC2 resources for the batch
	resources := make([]*pbc.ResourceDescriptor, 100)
	for i := 0; i < 100; i++ {
		resources[i] = &pbc.ResourceDescriptor{
			Provider:     "aws",
			ResourceType: "aws:ec2/instance:Instance", // Pulumi format to exercise normalization
			Sku:          "t3.micro",
			Region:       "us-east-1",
		}
	}

	req := &pbc.GetRecommendationsRequest{
		TargetResources: resources,
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := plugin.GetRecommendations(ctx, req)
		if err != nil {
			b.Fatalf("GetRecommendations failed: %v", err)
		}
	}
}

// BenchmarkGetRecommendations_MixedResourceTypes measures performance with
// a heterogeneous batch of different AWS resource types.
//
// This validates that the serviceResolver correctly handles diverse
// resource type formats in a single batch request.
//
// Run with: go test -tags=region_use1 -bench=BenchmarkGetRecommendations_MixedResourceTypes -benchmem ./internal/plugin/...
func BenchmarkGetRecommendations_MixedResourceTypes(b *testing.B) {
	mock := newMockPricingClient("us-east-1", "USD")
	logger := zerolog.New(io.Discard).Level(zerolog.Disabled)
	plugin := NewAWSPublicPlugin("us-east-1", "test-version", mock, logger)

	// Create a mix of resource types (EC2, EBS, RDS)
	resourceTypes := []struct {
		resourceType string
		sku          string
	}{
		{"aws:ec2/instance:Instance", "t3.micro"},
		{"aws:ec2/instance:Instance", "m5.large"},
		{"aws:ec2/volume:Volume", "gp3"},
		{"aws:rds/instance:Instance", "db.t3.micro"},
		{"ec2", "c5.xlarge"},
		{"ebs", "io1"},
		{"rds", "db.m5.large"},
	}

	// Create 98 resources (14 of each type to get close to 100)
	resources := make([]*pbc.ResourceDescriptor, 0, 98)
	for i := 0; i < 14; i++ {
		for _, rt := range resourceTypes {
			resources = append(resources, &pbc.ResourceDescriptor{
				Provider:     "aws",
				ResourceType: rt.resourceType,
				Sku:          rt.sku,
				Region:       "us-east-1",
			})
		}
	}

	req := &pbc.GetRecommendationsRequest{
		TargetResources: resources,
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := plugin.GetRecommendations(ctx, req)
		if err != nil {
			b.Fatalf("GetRecommendations failed: %v", err)
		}
	}
}

// BenchmarkGetRecommendations_SingleResource measures the baseline overhead
// of GetRecommendations for a single resource request.
//
// Run with: go test -tags=region_use1 -bench=BenchmarkGetRecommendations_SingleResource -benchmem ./internal/plugin/...
func BenchmarkGetRecommendations_SingleResource(b *testing.B) {
	mock := newMockPricingClient("us-east-1", "USD")
	logger := zerolog.New(io.Discard).Level(zerolog.Disabled)
	plugin := NewAWSPublicPlugin("us-east-1", "test-version", mock, logger)

	req := &pbc.GetRecommendationsRequest{
		TargetResources: []*pbc.ResourceDescriptor{
			{
				Provider:     "aws",
				ResourceType: "aws:ec2/instance:Instance",
				Sku:          "t3.micro",
				Region:       "us-east-1",
			},
		},
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := plugin.GetRecommendations(ctx, req)
		if err != nil {
			b.Fatalf("GetRecommendations failed: %v", err)
		}
	}
}
