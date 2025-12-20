package plugin

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog"
	pbc "github.com/rshade/pulumicost-spec/sdk/go/proto/pulumicost/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TestGetProjectedCost_EC2 tests EC2 cost estimation (T040)
func TestGetProjectedCost_EC2(t *testing.T) {
	mock := newMockPricingClient("us-east-1", "USD")
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	mock.ec2Prices["t3.micro/Linux/Shared"] = 0.0104
	plugin := NewAWSPublicPlugin("us-east-1", mock, logger)

	resp, err := plugin.GetProjectedCost(context.Background(), &pbc.GetProjectedCostRequest{
		Resource: &pbc.ResourceDescriptor{
			Provider:     "aws",
			ResourceType: "ec2",
			Sku:          "t3.micro",
			Region:       "us-east-1",
		},
	})

	if err != nil {
		t.Fatalf("GetProjectedCost() returned error: %v", err)
	}

	// Verify cost calculation: 0.0104 * 730 = 7.592
	expectedCost := 0.0104 * 730.0
	if resp.CostPerMonth != expectedCost {
		t.Errorf("CostPerMonth = %v, want %v", resp.CostPerMonth, expectedCost)
	}

	if resp.UnitPrice != 0.0104 {
		t.Errorf("UnitPrice = %v, want 0.0104", resp.UnitPrice)
	}

	if resp.Currency != "USD" {
		t.Errorf("Currency = %q, want %q", resp.Currency, "USD")
	}

	if resp.BillingDetail == "" {
		t.Error("BillingDetail should not be empty")
	}

	// Verify pricing client was called
	if mock.ec2OnDemandCalled != 1 {
		t.Errorf("EC2OnDemandPricePerHour called %d times, want 1", mock.ec2OnDemandCalled)
	}
}

// TestGetProjectedCost_EC2_PulumiFormat tests EC2 cost estimation with Pulumi resource type format (T042)
func TestGetProjectedCost_EC2_PulumiFormat(t *testing.T) {
	mock := newMockPricingClient("us-east-1", "USD")
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	mock.ec2Prices["t3.micro/Linux/Shared"] = 0.0104
	plugin := NewAWSPublicPlugin("us-east-1", mock, logger)

	// Test with Pulumi format resource type
	resp, err := plugin.GetProjectedCost(context.Background(), &pbc.GetProjectedCostRequest{
		Resource: &pbc.ResourceDescriptor{
			Provider:     "aws",
			ResourceType: "aws:ec2/instance:Instance", // Pulumi format
			Sku:          "t3.micro",
			Region:       "us-east-1",
		},
	})

	if err != nil {
		t.Fatalf("GetProjectedCost() with Pulumi format failed: %v", err)
	}

	expectedCost := 0.0104 * 730.0
	if resp.CostPerMonth != expectedCost {
		t.Errorf("CostPerMonth = %v, want %v", resp.CostPerMonth, expectedCost)
	}
}

// TestGetProjectedCost_EBS_WithSize tests EBS cost estimation with explicit size (T041)
func TestGetProjectedCost_EBS_WithSize(t *testing.T) {
	mock := newMockPricingClient("us-east-1", "USD")
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	mock.ebsPrices["gp3"] = 0.08
	plugin := NewAWSPublicPlugin("us-east-1", mock, logger)

	resp, err := plugin.GetProjectedCost(context.Background(), &pbc.GetProjectedCostRequest{
		Resource: &pbc.ResourceDescriptor{
			Provider:     "aws",
			ResourceType: "ebs",
			Sku:          "gp3",
			Region:       "us-east-1",
			Tags: map[string]string{
				"size": "100",
			},
		},
	})

	if err != nil {
		t.Fatalf("GetProjectedCost() returned error: %v", err)
	}

	// Verify cost calculation: 0.08 * 100 = 8.0
	expectedCost := 0.08 * 100.0
	if resp.CostPerMonth != expectedCost {
		t.Errorf("CostPerMonth = %v, want %v", resp.CostPerMonth, expectedCost)
	}

	if resp.UnitPrice != 0.08 {
		t.Errorf("UnitPrice = %v, want 0.08", resp.UnitPrice)
	}

	if resp.Currency != "USD" {
		t.Errorf("Currency = %q, want %q", resp.Currency, "USD")
	}

	// Verify billing detail exists
	if resp.BillingDetail == "" {
		t.Error("BillingDetail should not be empty")
	}

	// Verify pricing client was called
	if mock.ebsPriceCalled != 1 {
		t.Errorf("EBSPricePerGBMonth called %d times, want 1", mock.ebsPriceCalled)
	}
}

// TestGetProjectedCost_EBS_DefaultSize tests EBS with defaulted 8GB size (T042)
func TestGetProjectedCost_EBS_DefaultSize(t *testing.T) {
	mock := newMockPricingClient("us-east-1", "USD")
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	mock.ebsPrices["gp2"] = 0.10
	plugin := NewAWSPublicPlugin("us-east-1", mock, logger)

	resp, err := plugin.GetProjectedCost(context.Background(), &pbc.GetProjectedCostRequest{
		Resource: &pbc.ResourceDescriptor{
			Provider:     "aws",
			ResourceType: "ebs",
			Sku:          "gp2",
			Region:       "us-east-1",
			// No tags - size should default to 8GB
		},
	})

	if err != nil {
		t.Fatalf("GetProjectedCost() returned error: %v", err)
	}

	// Verify cost calculation: 0.10 * 8 = 0.80
	expectedCost := 0.10 * 8.0
	if resp.CostPerMonth != expectedCost {
		t.Errorf("CostPerMonth = %v, want %v", resp.CostPerMonth, expectedCost)
	}

	if resp.Currency != "USD" {
		t.Errorf("Currency = %q, want %q", resp.Currency, "USD")
	}

	// Should mention "defaulted" in billing detail
	if resp.BillingDetail == "" {
		t.Error("BillingDetail should not be empty")
	}
}

// TestGetProjectedCost_RegionMismatch tests region mismatch error handling (T043)
func TestGetProjectedCost_RegionMismatch(t *testing.T) {
	mock := newMockPricingClient("us-east-1", "USD")
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	plugin := NewAWSPublicPlugin("us-east-1", mock, logger)

	_, err := plugin.GetProjectedCost(context.Background(), &pbc.GetProjectedCostRequest{
		Resource: &pbc.ResourceDescriptor{
			Provider:     "aws",
			ResourceType: "ec2",
			Sku:          "t3.micro",
			Region:       "eu-west-1", // Wrong region
		},
	})

	if err == nil {
		t.Fatal("GetProjectedCost() should return error for region mismatch")
	}

	// Check error code
	st, ok := status.FromError(err)
	if !ok {
		t.Fatal("Error should be a gRPC status error")
	}

	if st.Code() != codes.FailedPrecondition {
		t.Errorf("Error code = %v, want %v", st.Code(), codes.FailedPrecondition)
	}

	// Check error details contain pluginRegion and requiredRegion
	details := st.Details()
	if len(details) == 0 {
		t.Error("Error should contain details")
	}
}

// TestGetProjectedCost_MissingRequiredField tests validation error (T044)
func TestGetProjectedCost_MissingRequiredField(t *testing.T) {
	mock := newMockPricingClient("us-east-1", "USD")
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	plugin := NewAWSPublicPlugin("us-east-1", mock, logger)

	testCases := []struct {
		name     string
		resource *pbc.ResourceDescriptor
	}{
		{
			name: "Missing SKU",
			resource: &pbc.ResourceDescriptor{
				Provider:     "aws",
				ResourceType: "ec2",
				Sku:          "", // Missing
				Region:       "us-east-1",
			},
		},
		{
			name: "Missing Provider",
			resource: &pbc.ResourceDescriptor{
				Provider:     "", // Missing
				ResourceType: "ec2",
				Sku:          "t3.micro",
				Region:       "us-east-1",
			},
		},
		{
			name: "Missing ResourceType",
			resource: &pbc.ResourceDescriptor{
				Provider:     "aws",
				ResourceType: "", // Missing
				Sku:          "t3.micro",
				Region:       "us-east-1",
			},
		},
		{
			name: "Missing Region",
			resource: &pbc.ResourceDescriptor{
				Provider:     "aws",
				ResourceType: "ec2",
				Sku:          "t3.micro",
				Region:       "", // Missing
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := plugin.GetProjectedCost(context.Background(), &pbc.GetProjectedCostRequest{
				Resource: tc.resource,
			})

			if err == nil {
				t.Fatal("GetProjectedCost() should return error for missing required field")
			}

			st, ok := status.FromError(err)
			if !ok {
				t.Fatal("Error should be a gRPC status error")
			}

			if st.Code() != codes.InvalidArgument {
				t.Errorf("Error code = %v, want %v", st.Code(), codes.InvalidArgument)
			}
		})
	}
}

// TestGetProjectedCost_UnknownInstanceType tests unknown instance type handling
func TestGetProjectedCost_UnknownInstanceType(t *testing.T) {
	mock := newMockPricingClient("us-east-1", "USD")
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	// Don't add any pricing data
	plugin := NewAWSPublicPlugin("us-east-1", mock, logger)

	resp, err := plugin.GetProjectedCost(context.Background(), &pbc.GetProjectedCostRequest{
		Resource: &pbc.ResourceDescriptor{
			Provider:     "aws",
			ResourceType: "ec2",
			Sku:          "unknown.large",
			Region:       "us-east-1",
		},
	})

	if err != nil {
		t.Fatalf("GetProjectedCost() returned error: %v", err)
	}

	// Should return $0 with explanation
	if resp.CostPerMonth != 0 {
		t.Errorf("CostPerMonth = %v, want 0 for unknown instance type", resp.CostPerMonth)
	}

	if resp.BillingDetail == "" {
		t.Error("BillingDetail should explain why cost is $0")
	}
}

// TestGetProjectedCost_StubServices tests stub service handling
func TestGetProjectedCost_StubServices(t *testing.T) {
	mock := newMockPricingClient("us-east-1", "USD")
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	plugin := NewAWSPublicPlugin("us-east-1", mock, logger)

	testCases := []string{"s3", "lambda", "dynamodb"} // RDS is now fully supported

	for _, resourceType := range testCases {
		t.Run(resourceType, func(t *testing.T) {
			resp, err := plugin.GetProjectedCost(context.Background(), &pbc.GetProjectedCostRequest{
				Resource: &pbc.ResourceDescriptor{
					Provider:     "aws",
					ResourceType: resourceType,
					Sku:          "test-sku",
					Region:       "us-east-1",
				},
			})

			if err != nil {
				t.Fatalf("GetProjectedCost() returned error: %v", err)
			}

			// Should return $0 with explanation
			if resp.CostPerMonth != 0 {
				t.Errorf("CostPerMonth = %v, want 0 for stub service", resp.CostPerMonth)
			}

			if resp.Currency != "USD" {
				t.Errorf("Currency = %q, want %q", resp.Currency, "USD")
			}

			if resp.BillingDetail == "" {
				t.Error("BillingDetail should explain stub implementation")
			}
		})
	}
}

// TestGetProjectedCost_APSoutheast1_EC2 tests EC2 pricing for ap-southeast-1 (T011)
func TestGetProjectedCost_APSoutheast1_EC2(t *testing.T) {
	mock := newMockPricingClient("ap-southeast-1", "USD")
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	mock.ec2Prices["t3.micro/Linux/Shared"] = 0.0116 // Singapore pricing (+12% vs us-east-1)
	mock.ec2Prices["m5.large/Linux/Shared"] = 0.112
	plugin := NewAWSPublicPlugin("ap-southeast-1", mock, logger)

	tests := []struct {
		name         string
		instanceType string
		wantPrice    float64
	}{
		{
			name:         "t3.micro in Singapore",
			instanceType: "t3.micro",
			wantPrice:    0.0116,
		},
		{
			name:         "m5.large in Singapore",
			instanceType: "m5.large",
			wantPrice:    0.112,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := plugin.GetProjectedCost(context.Background(), &pbc.GetProjectedCostRequest{
				Resource: &pbc.ResourceDescriptor{
					Provider:     "aws",
					ResourceType: "ec2",
					Sku:          tt.instanceType,
					Region:       "ap-southeast-1",
				},
			})

			if err != nil {
				t.Fatalf("GetProjectedCost() returned error: %v", err)
			}

			expectedCost := tt.wantPrice * 730.0
			if resp.CostPerMonth != expectedCost {
				t.Errorf("CostPerMonth = %v, want %v", resp.CostPerMonth, expectedCost)
			}

			if resp.UnitPrice != tt.wantPrice {
				t.Errorf("UnitPrice = %v, want %v", resp.UnitPrice, tt.wantPrice)
			}

			if resp.Currency != "USD" {
				t.Errorf("Currency = %q, want %q", resp.Currency, "USD")
			}
		})
	}
}

// TestGetProjectedCost_APSoutheast1_EBS tests EBS pricing for ap-southeast-1 (T011)
func TestGetProjectedCost_APSoutheast1_EBS(t *testing.T) {
	mock := newMockPricingClient("ap-southeast-1", "USD")
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	mock.ebsPrices["gp3"] = 0.0896 // Singapore pricing
	mock.ebsPrices["io2"] = 0.1456
	plugin := NewAWSPublicPlugin("ap-southeast-1", mock, logger)

	tests := []struct {
		name       string
		volumeType string
		size       string
		wantPrice  float64
	}{
		{
			name:       "gp3 100GB in Singapore",
			volumeType: "gp3",
			size:       "100",
			wantPrice:  0.0896,
		},
		{
			name:       "io2 50GB in Singapore",
			volumeType: "io2",
			size:       "50",
			wantPrice:  0.1456,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := plugin.GetProjectedCost(context.Background(), &pbc.GetProjectedCostRequest{
				Resource: &pbc.ResourceDescriptor{
					Provider:     "aws",
					ResourceType: "ebs",
					Sku:          tt.volumeType,
					Region:       "ap-southeast-1",
					Tags: map[string]string{
						"size": tt.size,
					},
				},
			})

			if err != nil {
				t.Fatalf("GetProjectedCost() returned error: %v", err)
			}

			sizeGB := 100.0
			if tt.size == "50" {
				sizeGB = 50.0
			}
			expectedCost := tt.wantPrice * sizeGB

			if resp.CostPerMonth != expectedCost {
				t.Errorf("CostPerMonth = %v, want %v", resp.CostPerMonth, expectedCost)
			}

			if resp.UnitPrice != tt.wantPrice {
				t.Errorf("UnitPrice = %v, want %v", resp.UnitPrice, tt.wantPrice)
			}
		})
	}
}

// TestGetProjectedCost_APSoutheast1_RegionMismatch tests region mismatch for ap-southeast-1 binary (T011)
func TestGetProjectedCost_APSoutheast1_RegionMismatch(t *testing.T) {
	mock := newMockPricingClient("ap-southeast-1", "USD")
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	plugin := NewAWSPublicPlugin("ap-southeast-1", mock, logger)

	wrongRegions := []string{"us-east-1", "eu-west-1", "ap-southeast-2", "ap-northeast-1"}

	for _, region := range wrongRegions {
		t.Run("Request from "+region, func(t *testing.T) {
			_, err := plugin.GetProjectedCost(context.Background(), &pbc.GetProjectedCostRequest{
				Resource: &pbc.ResourceDescriptor{
					Provider:     "aws",
					ResourceType: "ec2",
					Sku:          "t3.micro",
					Region:       region,
				},
			})

			if err == nil {
				t.Fatal("GetProjectedCost() should return error for region mismatch")
			}

			st, ok := status.FromError(err)
			if !ok {
				t.Fatal("Error should be a gRPC status error")
			}

			if st.Code() != codes.FailedPrecondition {
				t.Errorf("Error code = %v, want %v", st.Code(), codes.FailedPrecondition)
			}
		})
	}
}

// TestGetProjectedCost_ConcurrentCalls tests thread safety with 10+ parallel gRPC calls (T040, SC-006)
func TestGetProjectedCost_ConcurrentCalls(t *testing.T) {
	mock := newMockPricingClient("ap-southeast-1", "USD")
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	mock.ec2Prices["t3.micro/Linux/Shared"] = 0.0116
	mock.ec2Prices["m5.large/Linux/Shared"] = 0.112
	mock.ebsPrices["gp3"] = 0.0896
	plugin := NewAWSPublicPlugin("ap-southeast-1", mock, logger)

	const numGoroutines = 20
	const callsPerGoroutine = 10
	totalCalls := numGoroutines * callsPerGoroutine

	errors := make(chan error, totalCalls)
	done := make(chan bool, totalCalls)

	// Launch concurrent goroutines making gRPC calls
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			for j := 0; j < callsPerGoroutine; j++ {
				// Alternate between EC2 and EBS requests
				var resp interface{}
				var err error

				if (id+j)%2 == 0 {
					// EC2 request
					resp, err = plugin.GetProjectedCost(context.Background(), &pbc.GetProjectedCostRequest{
						Resource: &pbc.ResourceDescriptor{
							Provider:     "aws",
							ResourceType: "ec2",
							Sku:          "t3.micro",
							Region:       "ap-southeast-1",
						},
					})
				} else {
					// EBS request
					resp, err = plugin.GetProjectedCost(context.Background(), &pbc.GetProjectedCostRequest{
						Resource: &pbc.ResourceDescriptor{
							Provider:     "aws",
							ResourceType: "ebs",
							Sku:          "gp3",
							Region:       "ap-southeast-1",
							Tags: map[string]string{
								"size": "100",
							},
						},
					})
				}

				if err != nil {
					errors <- err
				} else if resp == nil {
					errors <- nil // Signal completion but no error
				}
				done <- true
			}
		}(i)
	}

	// Wait for all calls to complete
	errorCount := 0
	for i := 0; i < totalCalls; i++ {
		<-done
	}

	// Check if any errors occurred
	close(errors)
	for err := range errors {
		if err != nil {
			errorCount++
			t.Errorf("Concurrent call failed: %v", err)
		}
	}

	if errorCount > 0 {
		t.Errorf("Failed %d out of %d concurrent calls", errorCount, totalCalls)
	}

	t.Logf("Successfully completed %d concurrent gRPC calls across %d goroutines", totalCalls, numGoroutines)
}

// BenchmarkGetProjectedCost_RegionMismatch benchmarks region mismatch error response time (T041, SC-005)
// Success criteria: Response time < 100ms
func BenchmarkGetProjectedCost_RegionMismatch(b *testing.B) {
	mock := newMockPricingClient("ap-southeast-1", "USD")
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	plugin := NewAWSPublicPlugin("ap-southeast-1", mock, logger)

	req := &pbc.GetProjectedCostRequest{
		Resource: &pbc.ResourceDescriptor{
			Provider:     "aws",
			ResourceType: "ec2",
			Sku:          "t3.micro",
			Region:       "us-east-1", // Wrong region
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = plugin.GetProjectedCost(context.Background(), req)
	}
}

// TestGetProjectedCost_RegionMismatchLatency tests that region mismatch errors return < 100ms (T041, SC-005)
func TestGetProjectedCost_RegionMismatchLatency(t *testing.T) {
	mock := newMockPricingClient("ap-southeast-1", "USD")
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	plugin := NewAWSPublicPlugin("ap-southeast-1", mock, logger)

	req := &pbc.GetProjectedCostRequest{
		Resource: &pbc.ResourceDescriptor{
			Provider:     "aws",
			ResourceType: "ec2",
			Sku:          "t3.micro",
			Region:       "us-east-1", // Wrong region
		},
	}

	// Run 100 samples to get average latency
	const samples = 100
	var totalDuration int64

	for i := 0; i < samples; i++ {
		start := time.Now()
		_, err := plugin.GetProjectedCost(context.Background(), req)
		duration := time.Since(start)
		totalDuration += duration.Nanoseconds()

		if err == nil {
			t.Fatal("Expected error for region mismatch")
		}
	}

	avgLatencyMs := float64(totalDuration) / float64(samples) / 1000000.0
	t.Logf("Average region mismatch latency: %.2f ms", avgLatencyMs)

	// Success criteria: < 100ms
	if avgLatencyMs >= 100.0 {
		t.Errorf("Region mismatch latency %.2f ms exceeds 100ms threshold (SC-005)", avgLatencyMs)
	}
}

// TestGetProjectedCost_CrossRegionPricingDifference tests that pricing differs across AP regions (T042, SC-003)
func TestGetProjectedCost_CrossRegionPricingDifference(t *testing.T) {
	// Create plugins for different AP regions with realistic pricing variations
	regions := map[string]struct {
		region   string
		ec2Price float64
		ebsPrice float64
	}{
		"Singapore": {"ap-southeast-1", 0.0116, 0.0896}, // +12%
		"Sydney":    {"ap-southeast-2", 0.0120, 0.0920}, // +15%
		"Tokyo":     {"ap-northeast-1", 0.0123, 0.0944}, // +18%
		"Mumbai":    {"ap-south-1", 0.0112, 0.0864},     // +8%
	}

	costs := make(map[string]float64)

	for name, data := range regions {
		mock := newMockPricingClient(data.region, "USD")
		logger := zerolog.New(nil).Level(zerolog.InfoLevel)
		mock.ec2Prices["t3.micro/Linux/Shared"] = data.ec2Price
		plugin := NewAWSPublicPlugin(data.region, mock, logger)

		resp, err := plugin.GetProjectedCost(context.Background(), &pbc.GetProjectedCostRequest{
			Resource: &pbc.ResourceDescriptor{
				Provider:     "aws",
				ResourceType: "ec2",
				Sku:          "t3.micro",
				Region:       data.region,
			},
		})

		if err != nil {
			t.Fatalf("%s: GetProjectedCost() error: %v", name, err)
		}

		costs[name] = resp.CostPerMonth
		t.Logf("%s (t3.micro): $%.2f/month (hourly: $%.4f)", name, resp.CostPerMonth, resp.UnitPrice)
	}

	// Verify that costs differ between regions
	singaporeCost := costs["Singapore"]
	for name, cost := range costs {
		if name == "Singapore" {
			continue
		}
		if cost == singaporeCost {
			t.Errorf("Cost for %s ($%.2f) equals Singapore cost ($%.2f), expected different pricing (SC-003)", name, cost, singaporeCost)
		}
	}

	// Verify we have at least 4 different costs
	uniqueCosts := make(map[float64]bool)
	for _, cost := range costs {
		uniqueCosts[cost] = true
	}
	if len(uniqueCosts) < 4 {
		t.Errorf("Expected 4 unique costs across regions, got %d (SC-003)", len(uniqueCosts))
	}

	t.Logf("Successfully verified pricing varies across %d AP regions", len(regions))
}

// TestSupports_RegionRejection tests that each region binary rejects other regions 100% of the time (T043, SC-008)
func TestSupports_RegionRejection(t *testing.T) {
	testRegions := []string{"ap-southeast-1", "ap-southeast-2", "ap-northeast-1", "ap-south-1"}

	for _, pluginRegion := range testRegions {
		t.Run("Binary_"+pluginRegion, func(t *testing.T) {
			mock := newMockPricingClient(pluginRegion, "USD")
			logger := zerolog.New(nil).Level(zerolog.InfoLevel)
			plugin := NewAWSPublicPlugin(pluginRegion, mock, logger)

			totalTests := 0
			successfulRejections := 0

			// Test all regions except the plugin's own region
			for _, requestRegion := range testRegions {
				if requestRegion == pluginRegion {
					continue // Skip same region
				}

				totalTests++

				// Test EC2
				resp, err := plugin.Supports(context.Background(), &pbc.SupportsRequest{
					Resource: &pbc.ResourceDescriptor{
						Provider:     "aws",
						ResourceType: "ec2",
						Region:       requestRegion,
					},
				})

				if err != nil {
					t.Errorf("Supports() returned error: %v", err)
					continue
				}

				if resp.Supported {
					t.Errorf("Plugin %s incorrectly supported EC2 request from %s", pluginRegion, requestRegion)
				} else {
					successfulRejections++
				}

				// Test EBS
				resp, err = plugin.Supports(context.Background(), &pbc.SupportsRequest{
					Resource: &pbc.ResourceDescriptor{
						Provider:     "aws",
						ResourceType: "ebs",
						Region:       requestRegion,
					},
				})

				if err != nil {
					t.Errorf("Supports() returned error: %v", err)
					continue
				}

				if resp.Supported {
					t.Errorf("Plugin %s incorrectly supported EBS request from %s", pluginRegion, requestRegion)
				} else {
					successfulRejections++
				}

				totalTests++ // Increment for EBS test
			}

			rejectionRate := float64(successfulRejections) / float64(totalTests) * 100.0
			t.Logf("Plugin %s: Rejected %d/%d wrong-region requests (%.1f%%)", pluginRegion, successfulRejections, totalTests, rejectionRate)

			// Success criteria: 100% rejection rate (SC-008)
			if rejectionRate < 100.0 {
				t.Errorf("Plugin %s rejection rate %.1f%% is below 100%% requirement (SC-008)", pluginRegion, rejectionRate)
			}
		})
	}
}

// T027: Test GetProjectedCost logs contain required structured fields
func TestGetProjectedCostLogsContainRequiredFields(t *testing.T) {
	var logBuf bytes.Buffer
	mock := newMockPricingClient("us-east-1", "USD")
	mock.ec2Prices["t3.micro/Linux/Shared"] = 0.0104
	logger := zerolog.New(&logBuf).Level(zerolog.InfoLevel)
	plugin := NewAWSPublicPlugin("us-east-1", mock, logger)

	req := &pbc.GetProjectedCostRequest{
		Resource: &pbc.ResourceDescriptor{
			Provider:     "aws",
			ResourceType: "ec2",
			Sku:          "t3.micro",
			Region:       "us-east-1",
		},
	}

	_, err := plugin.GetProjectedCost(context.Background(), req)
	if err != nil {
		t.Fatalf("GetProjectedCost() error: %v", err)
	}

	// Parse log output and verify required fields
	var logEntry map[string]interface{}
	if err := json.Unmarshal(logBuf.Bytes(), &logEntry); err != nil {
		t.Fatalf("Failed to parse log output as JSON: %v", err)
	}

	// Required fields per data-model.md and tasks.md T023
	requiredFields := []string{
		"trace_id",
		"operation",
		"resource_type",
		"aws_service",
		"aws_region",
		"cost_monthly",
		"duration_ms",
		"message",
	}

	for _, field := range requiredFields {
		if _, ok := logEntry[field]; !ok {
			t.Errorf("GetProjectedCost log missing required field: %s", field)
		}
	}

	// Verify specific values
	if op, ok := logEntry["operation"].(string); ok {
		if op != "GetProjectedCost" {
			t.Errorf("operation = %q, want %q", op, "GetProjectedCost")
		}
	}

	if rt, ok := logEntry["resource_type"].(string); ok {
		if rt != "ec2" {
			t.Errorf("resource_type = %q, want %q", rt, "ec2")
		}
	}

	if region, ok := logEntry["aws_region"].(string); ok {
		if region != "us-east-1" {
			t.Errorf("aws_region = %q, want %q", region, "us-east-1")
		}
	}

	// cost_monthly should be the expected value
	if cost, ok := logEntry["cost_monthly"].(float64); ok {
		expectedCost := 0.0104 * 730.0
		if cost != expectedCost {
			t.Errorf("cost_monthly = %v, want %v", cost, expectedCost)
		}
	}

	// duration_ms should be non-negative
	if durationMs, ok := logEntry["duration_ms"].(float64); ok {
		if durationMs < 0 {
			t.Errorf("duration_ms = %v, should be non-negative", durationMs)
		}
	}
}

// T038: Test debug logs contain instance_type for EC2
func TestDebugLogsContainInstanceTypeForEC2(t *testing.T) {
	var logBuf bytes.Buffer
	mock := newMockPricingClient("us-east-1", "USD")
	mock.ec2Prices["t3.micro/Linux/Shared"] = 0.0104
	logger := zerolog.New(&logBuf).Level(zerolog.DebugLevel)
	plugin := NewAWSPublicPlugin("us-east-1", mock, logger)

	req := &pbc.GetProjectedCostRequest{
		Resource: &pbc.ResourceDescriptor{
			Provider:     "aws",
			ResourceType: "ec2",
			Sku:          "t3.micro",
			Region:       "us-east-1",
		},
	}

	_, err := plugin.GetProjectedCost(context.Background(), req)
	if err != nil {
		t.Fatalf("GetProjectedCost() error: %v", err)
	}

	// Parse all log lines (there may be multiple)
	logLines := bytes.Split(logBuf.Bytes(), []byte("\n"))
	foundInstanceType := false

	for _, line := range logLines {
		if len(line) == 0 {
			continue
		}
		var logEntry map[string]interface{}
		if err := json.Unmarshal(line, &logEntry); err != nil {
			continue // Skip invalid lines
		}

		// Look for debug log with instance_type
		if instanceType, ok := logEntry["instance_type"].(string); ok {
			if instanceType == "t3.micro" {
				foundInstanceType = true
				break
			}
		}
	}

	if !foundInstanceType {
		t.Error("Debug log should contain instance_type field for EC2 requests")
	}
}

// T039: Test debug logs contain storage_type for EBS
func TestDebugLogsContainStorageTypeForEBS(t *testing.T) {
	var logBuf bytes.Buffer
	mock := newMockPricingClient("us-east-1", "USD")
	mock.ebsPrices["gp3"] = 0.08
	logger := zerolog.New(&logBuf).Level(zerolog.DebugLevel)
	plugin := NewAWSPublicPlugin("us-east-1", mock, logger)

	req := &pbc.GetProjectedCostRequest{
		Resource: &pbc.ResourceDescriptor{
			Provider:     "aws",
			ResourceType: "ebs",
			Sku:          "gp3",
			Region:       "us-east-1",
			Tags: map[string]string{
				"size": "100",
			},
		},
	}

	_, err := plugin.GetProjectedCost(context.Background(), req)
	if err != nil {
		t.Fatalf("GetProjectedCost() error: %v", err)
	}

	// Parse all log lines (there may be multiple)
	logLines := bytes.Split(logBuf.Bytes(), []byte("\n"))
	foundStorageType := false

	for _, line := range logLines {
		if len(line) == 0 {
			continue
		}
		var logEntry map[string]interface{}
		if err := json.Unmarshal(line, &logEntry); err != nil {
			continue // Skip invalid lines
		}

		// Look for debug log with storage_type
		if storageType, ok := logEntry["storage_type"].(string); ok {
			if storageType == "gp3" {
				foundStorageType = true
				break
			}
		}
	}

	if !foundStorageType {
		t.Error("Debug log should contain storage_type field for EBS requests")
	}
}

// TestGetProjectedCost_RDS_MySQL tests RDS cost estimation with MySQL engine
func TestGetProjectedCost_RDS_MySQL(t *testing.T) {
	mock := newMockPricingClient("us-east-1", "USD")
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	mock.rdsInstancePrices["db.t3.medium/MySQL"] = 0.068
	mock.rdsStoragePrices["gp3"] = 0.10
	plugin := NewAWSPublicPlugin("us-east-1", mock, logger)

	resp, err := plugin.GetProjectedCost(context.Background(), &pbc.GetProjectedCostRequest{
		Resource: &pbc.ResourceDescriptor{
			Provider:     "aws",
			ResourceType: "rds",
			Sku:          "db.t3.medium",
			Region:       "us-east-1",
			Tags: map[string]string{
				"engine":       "mysql",
				"storage_type": "gp3",
				"storage_size": "100",
			},
		},
	})

	if err != nil {
		t.Fatalf("GetProjectedCost() returned error: %v", err)
	}

	// Verify instance cost: 0.068 * 730 = 49.64
	// Storage cost: 0.10 * 100 = 10.00
	// Total: 59.64
	expectedInstanceCost := 0.068 * 730.0
	expectedStorageCost := 0.10 * 100.0
	expectedTotal := expectedInstanceCost + expectedStorageCost

	if resp.CostPerMonth != expectedTotal {
		t.Errorf("CostPerMonth = %v, want %v", resp.CostPerMonth, expectedTotal)
	}

	if resp.UnitPrice != 0.068 {
		t.Errorf("UnitPrice = %v, want 0.068", resp.UnitPrice)
	}

	if resp.Currency != "USD" {
		t.Errorf("Currency = %q, want %q", resp.Currency, "USD")
	}

	if resp.BillingDetail == "" {
		t.Error("BillingDetail should not be empty")
	}

	// Verify pricing client was called
	if mock.rdsOnDemandCalled != 1 {
		t.Errorf("RDSOnDemandPricePerHour called %d times, want 1", mock.rdsOnDemandCalled)
	}
}

// TestGetProjectedCost_RDS_DefaultValues tests RDS with default values
func TestGetProjectedCost_RDS_DefaultValues(t *testing.T) {
	mock := newMockPricingClient("us-east-1", "USD")
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	mock.rdsInstancePrices["db.m5.large/MySQL"] = 0.171
	mock.rdsStoragePrices["gp2"] = 0.115
	plugin := NewAWSPublicPlugin("us-east-1", mock, logger)

	resp, err := plugin.GetProjectedCost(context.Background(), &pbc.GetProjectedCostRequest{
		Resource: &pbc.ResourceDescriptor{
			Provider:     "aws",
			ResourceType: "rds",
			Sku:          "db.m5.large",
			Region:       "us-east-1",
			// No tags - should default to mysql, gp2, 20GB
		},
	})

	if err != nil {
		t.Fatalf("GetProjectedCost() returned error: %v", err)
	}

	// Instance cost: 0.171 * 730 = 124.83
	// Storage cost: 0.115 * 20 = 2.30
	// Total: 127.13
	expectedInstanceCost := 0.171 * 730.0
	expectedStorageCost := 0.115 * 20.0
	expectedTotal := expectedInstanceCost + expectedStorageCost

	// Use tolerance for floating-point comparison
	tolerance := 0.0001
	if diff := resp.CostPerMonth - expectedTotal; diff < -tolerance || diff > tolerance {
		t.Errorf("CostPerMonth = %v, want %v (within tolerance %v)", resp.CostPerMonth, expectedTotal, tolerance)
	}

	// BillingDetail should mention defaults
	if resp.BillingDetail == "" {
		t.Error("BillingDetail should not be empty")
	}
	if !strings.Contains(resp.BillingDetail, "defaulted") {
		t.Errorf("BillingDetail should mention defaults, got: %s", resp.BillingDetail)
	}
}

// TestGetProjectedCost_RDS_PostgreSQL tests RDS with PostgreSQL engine
func TestGetProjectedCost_RDS_PostgreSQL(t *testing.T) {
	mock := newMockPricingClient("us-east-1", "USD")
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	mock.rdsInstancePrices["db.t3.medium/PostgreSQL"] = 0.068
	mock.rdsStoragePrices["gp3"] = 0.10
	plugin := NewAWSPublicPlugin("us-east-1", mock, logger)

	resp, err := plugin.GetProjectedCost(context.Background(), &pbc.GetProjectedCostRequest{
		Resource: &pbc.ResourceDescriptor{
			Provider:     "aws",
			ResourceType: "rds",
			Sku:          "db.t3.medium",
			Region:       "us-east-1",
			Tags: map[string]string{
				"engine":       "postgres",
				"storage_type": "gp3",
				"storage_size": "50",
			},
		},
	})

	if err != nil {
		t.Fatalf("GetProjectedCost() returned error: %v", err)
	}

	// Verify PostgreSQL was normalized correctly
	if resp.UnitPrice != 0.068 {
		t.Errorf("UnitPrice = %v, want 0.068", resp.UnitPrice)
	}

	// BillingDetail should show PostgreSQL
	if !strings.Contains(resp.BillingDetail, "PostgreSQL") {
		t.Errorf("BillingDetail should contain PostgreSQL, got: %s", resp.BillingDetail)
	}
}

// TestGetProjectedCost_RDS_UnknownEngine tests defaulting to MySQL for unknown engine
func TestGetProjectedCost_RDS_UnknownEngine(t *testing.T) {
	mock := newMockPricingClient("us-east-1", "USD")
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	mock.rdsInstancePrices["db.t3.micro/MySQL"] = 0.017
	mock.rdsStoragePrices["gp2"] = 0.115
	plugin := NewAWSPublicPlugin("us-east-1", mock, logger)

	resp, err := plugin.GetProjectedCost(context.Background(), &pbc.GetProjectedCostRequest{
		Resource: &pbc.ResourceDescriptor{
			Provider:     "aws",
			ResourceType: "rds",
			Sku:          "db.t3.micro",
			Region:       "us-east-1",
			Tags: map[string]string{
				"engine": "unknown-engine",
			},
		},
	})

	if err != nil {
		t.Fatalf("GetProjectedCost() returned error: %v", err)
	}

	// Should default to MySQL pricing
	if resp.UnitPrice != 0.017 {
		t.Errorf("UnitPrice = %v, want 0.017 (MySQL default)", resp.UnitPrice)
	}

	// BillingDetail should mention it defaulted
	if !strings.Contains(resp.BillingDetail, "defaulted") {
		t.Errorf("BillingDetail should mention defaulted engine, got: %s", resp.BillingDetail)
	}
}

// TestGetProjectedCost_RDS_UnknownInstance tests $0 return for unknown instance type
func TestGetProjectedCost_RDS_UnknownInstance(t *testing.T) {
	mock := newMockPricingClient("us-east-1", "USD")
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	// Don't add any RDS pricing data
	plugin := NewAWSPublicPlugin("us-east-1", mock, logger)

	resp, err := plugin.GetProjectedCost(context.Background(), &pbc.GetProjectedCostRequest{
		Resource: &pbc.ResourceDescriptor{
			Provider:     "aws",
			ResourceType: "rds",
			Sku:          "db.unknown.large",
			Region:       "us-east-1",
		},
	})

	if err != nil {
		t.Fatalf("GetProjectedCost() returned error: %v", err)
	}

	// Should return $0 with explanation
	if resp.CostPerMonth != 0 {
		t.Errorf("CostPerMonth = %v, want 0 for unknown instance type", resp.CostPerMonth)
	}

	if resp.BillingDetail == "" {
		t.Error("BillingDetail should explain why cost is $0")
	}
	if !strings.Contains(resp.BillingDetail, "not found") {
		t.Errorf("BillingDetail should mention not found, got: %s", resp.BillingDetail)
	}
}

// TestGetProjectedCost_RDS_AllEngines tests all supported database engines
func TestGetProjectedCost_RDS_AllEngines(t *testing.T) {
	tests := []struct {
		name               string
		engineTag          string
		expectedNormalized string
	}{
		{"MySQL", "mysql", "MySQL"},
		{"PostgreSQL", "postgres", "PostgreSQL"},
		{"PostgreSQL alias", "postgresql", "PostgreSQL"},
		{"MariaDB", "mariadb", "MariaDB"},
		{"Oracle", "oracle", "Oracle"},
		{"Oracle SE2", "oracle-se2", "Oracle"},
		{"SQL Server", "sqlserver", "SQL Server"},
		{"SQL Server Express", "sqlserver-ex", "SQL Server"},
		{"SQL Server Alias", "sql-server", "SQL Server"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newMockPricingClient("us-east-1", "USD")
			logger := zerolog.New(nil).Level(zerolog.InfoLevel)
			mock.rdsInstancePrices["db.t3.micro/"+tt.expectedNormalized] = 0.05
			mock.rdsStoragePrices["gp2"] = 0.115
			plugin := NewAWSPublicPlugin("us-east-1", mock, logger)

			resp, err := plugin.GetProjectedCost(context.Background(), &pbc.GetProjectedCostRequest{
				Resource: &pbc.ResourceDescriptor{
					Provider:     "aws",
					ResourceType: "rds",
					Sku:          "db.t3.micro",
					Region:       "us-east-1",
					Tags: map[string]string{
						"engine": tt.engineTag,
					},
				},
			})

			if err != nil {
				t.Fatalf("GetProjectedCost() returned error: %v", err)
			}

			// Should find pricing for the normalized engine
			if resp.UnitPrice == 0 {
				t.Errorf("UnitPrice = 0, expected non-zero for engine %s", tt.engineTag)
			}

			// BillingDetail should show normalized engine name
			if !strings.Contains(resp.BillingDetail, tt.expectedNormalized) {
				t.Errorf("BillingDetail should contain %s, got: %s", tt.expectedNormalized, resp.BillingDetail)
			}
		})
	}
}

// TestGetProjectedCost_RDS_InvalidStorageSize tests invalid storage size handling
func TestGetProjectedCost_RDS_InvalidStorageSize(t *testing.T) {
	mock := newMockPricingClient("us-east-1", "USD")
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	mock.rdsInstancePrices["db.t3.micro/MySQL"] = 0.017
	mock.rdsStoragePrices["gp2"] = 0.115
	plugin := NewAWSPublicPlugin("us-east-1", mock, logger)

	tests := []struct {
		name        string
		storageSize string
	}{
		{"negative size", "-100"},
		{"zero size", "0"},
		{"non-numeric", "abc"},
		{"empty string", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := plugin.GetProjectedCost(context.Background(), &pbc.GetProjectedCostRequest{
				Resource: &pbc.ResourceDescriptor{
					Provider:     "aws",
					ResourceType: "rds",
					Sku:          "db.t3.micro",
					Region:       "us-east-1",
					Tags: map[string]string{
						"engine":       "mysql",
						"storage_size": tt.storageSize,
					},
				},
			})

			if err != nil {
				t.Fatalf("GetProjectedCost() returned error: %v", err)
			}

			// Should default to 20GB storage
			// Instance cost: 0.017 * 730 = 12.41
			// Storage cost: 0.115 * 20 = 2.30
			expectedStorageCost := 0.115 * 20.0
			expectedInstanceCost := 0.017 * 730.0
			expectedTotal := expectedInstanceCost + expectedStorageCost

			if resp.CostPerMonth != expectedTotal {
				t.Errorf("CostPerMonth = %v, want %v (with default 20GB storage)", resp.CostPerMonth, expectedTotal)
			}

			// Should mention defaulted
			if !strings.Contains(resp.BillingDetail, "defaulted") {
				t.Errorf("BillingDetail should mention defaulted, got: %s", resp.BillingDetail)
			}
		})
	}
}

func TestDetectService(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// Exact matches
		{"simple ec2", "ec2", "ec2"},
		{"pulumi ec2/instance format", "aws:ec2/instance:Instance", "ec2"},
		{"pulumi ec2 format", "aws:ec2:Instance", "ec2"},
		{"pulumi ebs/volume format", "aws:ebs/volume:Volume", "ebs"},
		{"pulumi ec2/volume format", "aws:ec2/volume:Volume", "ebs"},

		// Containment fallbacks
		{"custom ec2/instance variant", "custom:ec2/instance:Something", "ec2"},
		{"custom ebs/volume variant", "custom:ebs/volume:Something", "ebs"},

		// Stub services
		{"s3 bucket", "aws:s3/bucket:Bucket", "s3"},
		{"lambda function", "aws:lambda/function:Function", "lambda"},

		// Unsupported - should return input as-is
		{"unsupported service", "aws:unknown:Service", "aws:unknown:Service"},
		{"completely unknown", "foobar", "foobar"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectService(tt.input)
			if got != tt.expected {
				t.Errorf("detectService(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

// TestGetProjectedCost_EKS_StandardSupport tests EKS standard support cost estimation
func TestGetProjectedCost_EKS_StandardSupport(t *testing.T) {
	mock := newMockPricingClient("us-east-1", "USD")
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	mock.eksStandardPrice = 0.10 // $0.10/hour for standard support
	mock.eksExtendedPrice = 0.50 // $0.50/hour for extended support
	plugin := NewAWSPublicPlugin("us-east-1", mock, logger)

	resp, err := plugin.GetProjectedCost(context.Background(), &pbc.GetProjectedCostRequest{
		Resource: &pbc.ResourceDescriptor{
			Provider:     "aws",
			ResourceType: "eks",
			Sku:          "cluster",
			Region:       "us-east-1",
		},
	})

	if err != nil {
		t.Fatalf("GetProjectedCost() returned error: %v", err)
	}

	// Verify cost calculation: 0.10 * 730 = 73.00
	expectedCost := 0.10 * 730.0
	if resp.CostPerMonth != expectedCost {
		t.Errorf("CostPerMonth = %v, want %v", resp.CostPerMonth, expectedCost)
	}

	if resp.UnitPrice != 0.10 {
		t.Errorf("UnitPrice = %v, want 0.10", resp.UnitPrice)
	}

	if resp.Currency != "USD" {
		t.Errorf("Currency = %q, want %q", resp.Currency, "USD")
	}

	// Verify billing detail mentions standard support and control plane only
	expectedDetail := "EKS cluster (standard support), 730 hrs/month (control plane only, excludes worker nodes)"
	if resp.BillingDetail != expectedDetail {
		t.Errorf("BillingDetail = %q, want %q", resp.BillingDetail, expectedDetail)
	}

	// Verify pricing client was called
	if mock.eksPriceCalled != 1 {
		t.Errorf("EKSClusterPricePerHour called %d times, want 1", mock.eksPriceCalled)
	}
}

// TestGetProjectedCost_EKS_ExtendedSupport tests EKS extended support cost estimation
func TestGetProjectedCost_EKS_ExtendedSupport(t *testing.T) {
	mock := newMockPricingClient("us-east-1", "USD")
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	mock.eksStandardPrice = 0.10 // $0.10/hour for standard support
	mock.eksExtendedPrice = 0.50 // $0.50/hour for extended support
	plugin := NewAWSPublicPlugin("us-east-1", mock, logger)

	resp, err := plugin.GetProjectedCost(context.Background(), &pbc.GetProjectedCostRequest{
		Resource: &pbc.ResourceDescriptor{
			Provider:     "aws",
			ResourceType: "eks",
			Sku:          "cluster-extended",
			Region:       "us-east-1",
		},
	})

	if err != nil {
		t.Fatalf("GetProjectedCost() returned error: %v", err)
	}

	// Verify cost calculation: 0.50 * 730 = 365.00
	expectedCost := 0.50 * 730.0
	if resp.CostPerMonth != expectedCost {
		t.Errorf("CostPerMonth = %v, want %v", resp.CostPerMonth, expectedCost)
	}

	if resp.UnitPrice != 0.50 {
		t.Errorf("UnitPrice = %v, want 0.50", resp.UnitPrice)
	}

	// Verify billing detail mentions extended support
	expectedDetail := "EKS cluster (extended support), 730 hrs/month (control plane only, excludes worker nodes)"
	if resp.BillingDetail != expectedDetail {
		t.Errorf("BillingDetail = %q, want %q", resp.BillingDetail, expectedDetail)
	}
}

// TestGetProjectedCost_EKS_MissingPricing tests behavior when EKS pricing data is unavailable.
// This mirrors TestGetProjectedCost_UnknownInstanceType for EC2 and
// TestGetProjectedCost_RDS_UnknownInstance for RDS.
func TestGetProjectedCost_EKS_MissingPricing(t *testing.T) {
	mock := newMockPricingClient("us-east-1", "USD")
	// Don't set eksStandardPrice or eksExtendedPrice - pricing is missing
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	plugin := NewAWSPublicPlugin("us-east-1", mock, logger)

	resp, err := plugin.GetProjectedCost(context.Background(), &pbc.GetProjectedCostRequest{
		Resource: &pbc.ResourceDescriptor{
			Provider:     "aws",
			ResourceType: "eks",
			Sku:          "cluster",
			Region:       "us-east-1",
		},
	})

	if err != nil {
		t.Fatalf("GetProjectedCost() returned error: %v", err)
	}

	// Should return $0 with explanation
	if resp.CostPerMonth != 0 {
		t.Errorf("CostPerMonth = %v, want 0 for missing pricing", resp.CostPerMonth)
	}

	if resp.UnitPrice != 0 {
		t.Errorf("UnitPrice = %v, want 0 for missing pricing", resp.UnitPrice)
	}

	if resp.BillingDetail == "" {
		t.Error("BillingDetail should explain missing pricing")
	}

	if !strings.Contains(resp.BillingDetail, "not available") {
		t.Errorf("BillingDetail should mention not available: %s", resp.BillingDetail)
	}
}

// TestGetProjectedCost_EKS_ExtendedSupportViaTags tests EKS extended support via tags
func TestGetProjectedCost_EKS_ExtendedSupportViaTags(t *testing.T) {
	mock := newMockPricingClient("us-east-1", "USD")
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	mock.eksStandardPrice = 0.10 // $0.10/hour for standard support
	mock.eksExtendedPrice = 0.50 // $0.50/hour for extended support
	plugin := NewAWSPublicPlugin("us-east-1", mock, logger)

	resp, err := plugin.GetProjectedCost(context.Background(), &pbc.GetProjectedCostRequest{
		Resource: &pbc.ResourceDescriptor{
			Provider:     "aws",
			ResourceType: "eks",
			Sku:          "cluster",
			Region:       "us-east-1",
			Tags: map[string]string{
				"support_type": "extended",
			},
		},
	})

	if err != nil {
		t.Fatalf("GetProjectedCost() returned error: %v", err)
	}

	// Verify cost calculation: 0.50 * 730 = 365.00
	expectedCost := 0.50 * 730.0
	if resp.CostPerMonth != expectedCost {
		t.Errorf("CostPerMonth = %v, want %v", resp.CostPerMonth, expectedCost)
	}

	// Verify billing detail mentions extended support
	expectedDetail := "EKS cluster (extended support), 730 hrs/month (control plane only, excludes worker nodes)"
	if resp.BillingDetail != expectedDetail {
		t.Errorf("BillingDetail = %q, want %q", resp.BillingDetail, expectedDetail)
	}
}

// TestGetProjectedCost_EKS_SupportTypeCaseInsensitive verifies that support_type tag comparison
// is case-insensitive. This is a regression test for issue #89 which identified that users
// setting support_type: Extended or support_type: EXTENDED would incorrectly receive
// standard pricing instead of extended pricing.
func TestGetProjectedCost_EKS_SupportTypeCaseInsensitive(t *testing.T) {
	tests := []struct {
		name        string
		supportType string
	}{
		{"Uppercase Extended", "Extended"},
		{"All caps EXTENDED", "EXTENDED"},
		{"Mixed case ExTeNdEd", "ExTeNdEd"},
		{"Lowercase extended", "extended"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newMockPricingClient("us-east-1", "USD")
			logger := zerolog.New(nil).Level(zerolog.InfoLevel)
			mock.eksStandardPrice = 0.10
			mock.eksExtendedPrice = 0.50
			plugin := NewAWSPublicPlugin("us-east-1", mock, logger)

			resp, err := plugin.GetProjectedCost(context.Background(), &pbc.GetProjectedCostRequest{
				Resource: &pbc.ResourceDescriptor{
					Provider:     "aws",
					ResourceType: "eks",
					Sku:          "cluster",
					Region:       "us-east-1",
					Tags: map[string]string{
						"support_type": tt.supportType,
					},
				},
			})

			if err != nil {
				t.Fatalf("GetProjectedCost() returned error: %v", err)
			}

			// Should use extended support pricing ($0.50/hour)
			expectedCost := 0.50 * 730.0
			if resp.CostPerMonth != expectedCost {
				t.Errorf("CostPerMonth = %v, want %v (extended pricing)", resp.CostPerMonth, expectedCost)
			}

			if resp.UnitPrice != 0.50 {
				t.Errorf("UnitPrice = %v, want 0.50 (extended pricing)", resp.UnitPrice)
			}

			// Verify billing detail shows extended support
			if !strings.Contains(resp.BillingDetail, "extended support") {
				t.Errorf("BillingDetail should mention extended support, got: %s", resp.BillingDetail)
			}
		})
	}
}

// TestExtractAWSSKU tests SDK-style SKU extraction with priority ordering
func TestExtractAWSSKU(t *testing.T) {
	tests := []struct {
		name     string
		tags     map[string]string
		expected string
	}{
		{
			name:     "nil tags",
			tags:     nil,
			expected: "",
		},
		{
			name:     "empty tags",
			tags:     map[string]string{},
			expected: "",
		},
		{
			name: "instanceType priority",
			tags: map[string]string{
				"type":           "t2.micro",
				"instance_class": "m5",
				"instanceType":   "t3.micro",
				"volumeType":     "gp3",
				"volume_type":    "io1",
			},
			expected: "t3.micro",
		},
		{
			name: "instance_class priority over type",
			tags: map[string]string{
				"type":           "t2.micro",
				"instance_class": "m5",
				"volumeType":     "gp3",
			},
			expected: "m5",
		},
		{
			name: "type priority over volume types",
			tags: map[string]string{
				"type":        "t2.micro",
				"volumeType":  "gp3",
				"volume_type": "io1",
			},
			expected: "t2.micro",
		},
		{
			name: "volumeType priority",
			tags: map[string]string{
				"volumeType":  "gp3",
				"volume_type": "io1",
			},
			expected: "gp3",
		},
		{
			name: "volume_type fallback",
			tags: map[string]string{
				"volume_type": "io1",
			},
			expected: "io1",
		},
		{
			name: "type alone fallback",
			tags: map[string]string{
				"type": "t3.micro",
			},
			expected: "t3.micro",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractAWSSKU(tt.tags)
			if result != tt.expected {
				t.Errorf("extractAWSSKU() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestExtractAWSRegion tests SDK-style region extraction
func TestExtractAWSRegion(t *testing.T) {
	tests := []struct {
		name     string
		tags     map[string]string
		expected string
	}{
		{
			name:     "nil tags",
			tags:     nil,
			expected: "",
		},
		{
			name: "direct region tag",
			tags: map[string]string{
				"region": "us-west-2",
			},
			expected: "us-west-2",
		},
		{
			name: "availability zone extraction",
			tags: map[string]string{
				"availabilityZone": "us-east-1a",
			},
			expected: "us-east-1",
		},
		{
			name: "region priority over availability zone",
			tags: map[string]string{
				"region":           "us-west-2",
				"availabilityZone": "us-east-1b",
			},
			expected: "us-west-2", // SDK: region has priority over availabilityZone
		},
		{
			name: "availability zone with trailing letter (d)",
			tags: map[string]string{
				"availabilityZone": "invalid",
			},
			expected: "invali", // SDK strips trailing lowercase letter 'd'
		},
		{
			name: "empty availability zone",
			tags: map[string]string{
				"availabilityZone": "",
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractAWSRegion(tt.tags)
			if result != tt.expected {
				t.Errorf("extractAWSRegion() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestGetProjectedCost_EBS_VolumeSizeAlias tests volume_size alias extraction (T055)
func TestGetProjectedCost_EBS_VolumeSizeAlias(t *testing.T) {
	mock := newMockPricingClient("us-east-1", "USD")
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	mock.ebsPrices["gp3"] = 0.08 // $0.08/GB-month

	plugin := NewAWSPublicPlugin("us-east-1", mock, logger)

	tests := []struct {
		name          string
		tags          map[string]string
		expectSize    int
		expectAssumed bool
	}{
		{
			name: "size tag priority",
			tags: map[string]string{
				"volumeType":  "gp3",
				"size":        "100",
				"volume_size": "200", // Should be ignored due to priority
			},
			expectSize:    100,
			expectAssumed: false,
		},
		{
			name: "volume_size alias",
			tags: map[string]string{
				"volumeType":  "gp3",
				"volume_size": "150",
			},
			expectSize:    150,
			expectAssumed: false,
		},
		{
			name: "default size when no tags",
			tags: map[string]string{
				"volumeType": "gp3",
			},
			expectSize:    8,
			expectAssumed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &pbc.GetProjectedCostRequest{
				Resource: &pbc.ResourceDescriptor{
					Provider:     "aws",
					ResourceType: "aws:ebs/volume:Volume",
					Sku:          "gp3",
					Region:       "us-east-1",
					Tags:         tt.tags,
				},
			}

			resp, err := plugin.GetProjectedCost(context.Background(), req)
			if err != nil {
				t.Fatalf("GetProjectedCost failed: %v", err)
			}

			// Verify size is correctly extracted
			expectedCost := 0.08 * float64(tt.expectSize)
			if resp.CostPerMonth != expectedCost {
				t.Errorf("CostPerMonth = %v, want %v", resp.CostPerMonth, expectedCost)
			}

			// Verify billing detail includes size and defaulted annotation
			if tt.expectAssumed {
				if !strings.Contains(resp.BillingDetail, "(defaulted)") {
					t.Errorf("BillingDetail should contain '(defaulted)' for assumed size, got: %s", resp.BillingDetail)
				}
			} else {
				if strings.Contains(resp.BillingDetail, "(defaulted)") {
					t.Errorf("BillingDetail should not contain '(defaulted)' for explicit size, got: %s", resp.BillingDetail)
				}
			}
		})
	}
}

// TestGetProjectedCost_RDS_EngineDefaulted tests engine default tracking (T057)
func TestGetProjectedCost_RDS_EngineDefaulted(t *testing.T) {
	mock := newMockPricingClient("us-east-1", "USD")
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	mock.rdsInstancePrices["db.t3.micro/MySQL"] = 0.017

	plugin := NewAWSPublicPlugin("us-east-1", mock, logger)

	tests := []struct {
		name            string
		tags            map[string]string
		expectDefaulted bool
	}{
		{
			name: "explicit engine",
			tags: map[string]string{
				"instanceType": "db.t3.micro",
				"engine":       "postgres",
			},
			expectDefaulted: false,
		},
		{
			name: "defaulted engine",
			tags: map[string]string{
				"instanceType": "db.t3.micro",
				// No engine tag - should default to MySQL
			},
			expectDefaulted: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &pbc.GetProjectedCostRequest{
				Resource: &pbc.ResourceDescriptor{
					Provider:     "aws",
					ResourceType: "aws:rds/instance:Instance",
					Sku:          "db.t3.micro",
					Region:       "us-east-1",
					Tags:         tt.tags,
				},
			}

			resp, err := plugin.GetProjectedCost(context.Background(), req)
			if err != nil {
				t.Fatalf("GetProjectedCost failed: %v", err)
			}

			// Verify billing detail includes defaulted annotation when expected
			if tt.expectDefaulted {
				if !strings.Contains(resp.BillingDetail, "engine defaulted to MySQL") {
					t.Errorf("BillingDetail should contain 'engine defaulted to MySQL', got: %s", resp.BillingDetail)
				}
			} else {
				if strings.Contains(resp.BillingDetail, "defaulted") {
					t.Errorf("BillingDetail should not contain 'defaulted' for explicit engine, got: %s", resp.BillingDetail)
				}
			}
		})
	}
}

// TestGetProjectedCost_Lambda_Basic tests basic Lambda cost estimation
func TestGetProjectedCost_Lambda_Basic(t *testing.T) {
	mock := newMockPricingClient("us-east-1", "USD")
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	mock.lambdaPrices["request"] = 0.0000002     // $0.20 per 1M
	mock.lambdaPrices["gb-second"] = 0.0000166667 // Standard price
	plugin := NewAWSPublicPlugin("us-east-1", mock, logger)

	resp, err := plugin.GetProjectedCost(context.Background(), &pbc.GetProjectedCostRequest{
		Resource: &pbc.ResourceDescriptor{
			Provider:     "aws",
			ResourceType: "aws:lambda/function:Function",
			Sku:          "512", // 512 MB
			Region:       "us-east-1",
			Tags: map[string]string{
				"requests_per_month": "1000000", // 1M requests
				"avg_duration_ms":    "200",     // 200 ms
			},
		},
	})

	if err != nil {
		t.Fatalf("GetProjectedCost() returned error: %v", err)
	}

	// Calculation:
	// Memory: 512 MB / 1024 = 0.5 GB
	// Duration: 200 ms / 1000 = 0.2 s
	// GB-Seconds: 0.5 * 0.2 * 1,000,000 = 100,000
	// Request Cost: 1,000,000 * 0.0000002 = 0.20
	// Compute Cost: 100,000 * 0.0000166667 = 1.66667
	// Total: 1.86667

	expectedCost := 1.86667
	tolerance := 0.00001
	if diff := resp.CostPerMonth - expectedCost; diff < -tolerance || diff > tolerance {
		t.Errorf("CostPerMonth = %v, want %v", resp.CostPerMonth, expectedCost)
	}

	if resp.UnitPrice != 0.0000166667 {
		t.Errorf("UnitPrice = %v, want 0.0000166667", resp.UnitPrice)
	}

	if !strings.Contains(resp.BillingDetail, "Lambda 512MB") {
		t.Errorf("BillingDetail missing memory info: %s", resp.BillingDetail)
	}

	// FR-011: Verify architecture is shown (defaults to x86_64)
	if !strings.Contains(resp.BillingDetail, "x86_64") {
		t.Errorf("BillingDetail missing architecture info: %s", resp.BillingDetail)
	}
}

// TestGetProjectedCost_Lambda_Defaults tests Lambda with missing tags (default values)
func TestGetProjectedCost_Lambda_Defaults(t *testing.T) {
	mock := newMockPricingClient("us-east-1", "USD")
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	mock.lambdaPrices["request"] = 0.0000002
	mock.lambdaPrices["gb-second"] = 0.0000166667
	plugin := NewAWSPublicPlugin("us-east-1", mock, logger)

	resp, err := plugin.GetProjectedCost(context.Background(), &pbc.GetProjectedCostRequest{
		Resource: &pbc.ResourceDescriptor{
			Provider:     "aws",
			ResourceType: "aws:lambda/function:Function",
			Sku:          "128",
			Region:       "us-east-1",
			// No tags - should default to 0 requests
		},
	})

	if err != nil {
		t.Fatalf("GetProjectedCost() returned error: %v", err)
	}

	// Should be 0 cost because default requests = 0
	if resp.CostPerMonth != 0 {
		t.Errorf("CostPerMonth = %v, want 0", resp.CostPerMonth)
	}

	if !strings.Contains(resp.BillingDetail, "defaulted") {
		t.Errorf("BillingDetail should mention defaults: %s", resp.BillingDetail)
	}
}

// TestGetProjectedCost_Lambda_InvalidMemory tests Lambda with invalid memory SKU
func TestGetProjectedCost_Lambda_InvalidMemory(t *testing.T) {
	mock := newMockPricingClient("us-east-1", "USD")
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	mock.lambdaPrices["request"] = 0.0000002
	mock.lambdaPrices["gb-second"] = 0.0000166667
	plugin := NewAWSPublicPlugin("us-east-1", mock, logger)

	resp, err := plugin.GetProjectedCost(context.Background(), &pbc.GetProjectedCostRequest{
		Resource: &pbc.ResourceDescriptor{
			Provider:     "aws",
			ResourceType: "aws:lambda/function:Function",
			Sku:          "unknown", // Invalid SKU
			Region:       "us-east-1",
			Tags: map[string]string{
				"requests_per_month": "1000000",
			},
		},
	})

	if err != nil {
		t.Fatalf("GetProjectedCost() returned error: %v", err)
	}

	// Should default to 128MB
	// 128MB = 0.125 GB
	// Default duration = 100ms = 0.1s
	// GB-Seconds = 0.125 * 0.1 * 1,000,000 = 12,500
	// Request Cost: 0.20
	// Compute Cost: 12,500 * 0.0000166667 = 0.20833375
	// Total: 0.40833375

	expectedCost := 0.40833375
	tolerance := 0.00001
	if diff := resp.CostPerMonth - expectedCost; diff < -tolerance || diff > tolerance {
		t.Errorf("CostPerMonth = %v, want %v (with default 128MB)", resp.CostPerMonth, expectedCost)
	}

	if !strings.Contains(resp.BillingDetail, "defaulted") {
		t.Errorf("BillingDetail should mention defaults: %s", resp.BillingDetail)
	}
}

// TestGetProjectedCost_Lambda_ARM64 tests Lambda with arm64 architecture (FR-011).
// ARM architecture is approximately 20% cheaper than x86_64 for compute duration.
func TestGetProjectedCost_Lambda_ARM64(t *testing.T) {
	mock := newMockPricingClient("us-east-1", "USD")
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	mock.lambdaPrices["request"] = 0.0000002
	mock.lambdaPrices["gb-second"] = 0.0000166667      // x86 price
	mock.lambdaPrices["gb-second-arm64"] = 0.0000133334 // ARM price (~20% cheaper)
	plugin := NewAWSPublicPlugin("us-east-1", mock, logger)

	resp, err := plugin.GetProjectedCost(context.Background(), &pbc.GetProjectedCostRequest{
		Resource: &pbc.ResourceDescriptor{
			Provider:     "aws",
			ResourceType: "aws:lambda/function:Function",
			Sku:          "512", // 512 MB
			Region:       "us-east-1",
			Tags: map[string]string{
				"requests_per_month": "1000000", // 1M requests
				"avg_duration_ms":    "200",     // 200 ms
				"arch":               "arm64",   // FR-011: ARM architecture
			},
		},
	})

	if err != nil {
		t.Fatalf("GetProjectedCost() returned error: %v", err)
	}

	// Calculation with ARM pricing:
	// Memory: 512 MB / 1024 = 0.5 GB
	// Duration: 200 ms / 1000 = 0.2 s
	// GB-Seconds: 0.5 * 0.2 * 1,000,000 = 100,000
	// Request Cost: 1,000,000 * 0.0000002 = 0.20
	// Compute Cost: 100,000 * 0.0000133334 = 1.33334 (ARM price)
	// Total: 1.53334

	expectedCost := 1.53334
	tolerance := 0.00001
	if diff := resp.CostPerMonth - expectedCost; diff < -tolerance || diff > tolerance {
		t.Errorf("CostPerMonth = %v, want %v (with ARM pricing)", resp.CostPerMonth, expectedCost)
	}

	// Verify ARM pricing used
	if resp.UnitPrice != 0.0000133334 {
		t.Errorf("UnitPrice = %v, want 0.0000133334 (ARM rate)", resp.UnitPrice)
	}

	// Verify billing detail mentions ARM architecture
	if !strings.Contains(resp.BillingDetail, "arm64") {
		t.Errorf("BillingDetail should mention arm64: %s", resp.BillingDetail)
	}
}

// TestGetProjectedCost_Lambda_ArchitectureVariants tests various architecture tag formats (FR-011).
// The plugin should accept multiple formats: arm64, arm, x86_64, x86, or architecture tag.
func TestGetProjectedCost_Lambda_ArchitectureVariants(t *testing.T) {
	tests := []struct {
		name         string
		tags         map[string]string
		wantARMPrice bool
		wantArch     string
	}{
		{
			name: "arch tag arm64",
			tags: map[string]string{
				"requests_per_month": "1000",
				"avg_duration_ms":    "100",
				"arch":               "arm64",
			},
			wantARMPrice: true,
			wantArch:     "arm64",
		},
		{
			name: "arch tag arm",
			tags: map[string]string{
				"requests_per_month": "1000",
				"avg_duration_ms":    "100",
				"arch":               "arm",
			},
			wantARMPrice: true,
			wantArch:     "arm",
		},
		{
			name: "architecture tag arm64",
			tags: map[string]string{
				"requests_per_month": "1000",
				"avg_duration_ms":    "100",
				"architecture":       "arm64",
			},
			wantARMPrice: true,
			wantArch:     "arm64",
		},
		{
			name: "arch tag x86_64",
			tags: map[string]string{
				"requests_per_month": "1000",
				"avg_duration_ms":    "100",
				"arch":               "x86_64",
			},
			wantARMPrice: false,
			wantArch:     "x86_64",
		},
		{
			name: "arch tag x86",
			tags: map[string]string{
				"requests_per_month": "1000",
				"avg_duration_ms":    "100",
				"arch":               "x86",
			},
			wantARMPrice: false,
			wantArch:     "x86",
		},
		{
			name: "no arch defaults to x86_64",
			tags: map[string]string{
				"requests_per_month": "1000",
				"avg_duration_ms":    "100",
			},
			wantARMPrice: false,
			wantArch:     "x86_64",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newMockPricingClient("us-east-1", "USD")
			logger := zerolog.New(nil).Level(zerolog.InfoLevel)
			mock.lambdaPrices["request"] = 0.0000002
			mock.lambdaPrices["gb-second"] = 0.0000166667
			mock.lambdaPrices["gb-second-arm64"] = 0.0000133334
			plugin := NewAWSPublicPlugin("us-east-1", mock, logger)

			resp, err := plugin.GetProjectedCost(context.Background(), &pbc.GetProjectedCostRequest{
				Resource: &pbc.ResourceDescriptor{
					Provider:     "aws",
					ResourceType: "aws:lambda/function:Function",
					Sku:          "128",
					Region:       "us-east-1",
					Tags:         tt.tags,
				},
			})

			if err != nil {
				t.Fatalf("GetProjectedCost() returned error: %v", err)
			}

			expectedPrice := 0.0000166667
			if tt.wantARMPrice {
				expectedPrice = 0.0000133334
			}

			if resp.UnitPrice != expectedPrice {
				t.Errorf("UnitPrice = %v, want %v", resp.UnitPrice, expectedPrice)
			}

			// Verify billing detail mentions correct architecture
			if !strings.Contains(resp.BillingDetail, tt.wantArch) {
				t.Errorf("BillingDetail should contain %s: %s", tt.wantArch, resp.BillingDetail)
			}
		})
	}
}

// TestGetProjectedCost_Lambda_ARMFallbackToX86 tests that ARM falls back to x86 when ARM pricing unavailable.
// This handles edge cases where a region might not have ARM pricing data.
func TestGetProjectedCost_Lambda_ARMFallbackToX86(t *testing.T) {
	mock := newMockPricingClient("us-east-1", "USD")
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	mock.lambdaPrices["request"] = 0.0000002
	mock.lambdaPrices["gb-second"] = 0.0000166667
	// Note: No ARM pricing set (gb-second-arm64)
	plugin := NewAWSPublicPlugin("us-east-1", mock, logger)

	resp, err := plugin.GetProjectedCost(context.Background(), &pbc.GetProjectedCostRequest{
		Resource: &pbc.ResourceDescriptor{
			Provider:     "aws",
			ResourceType: "aws:lambda/function:Function",
			Sku:          "128",
			Region:       "us-east-1",
			Tags: map[string]string{
				"requests_per_month": "1000",
				"avg_duration_ms":    "100",
				"arch":               "arm64", // Request ARM but no ARM pricing
			},
		},
	})

	if err != nil {
		t.Fatalf("GetProjectedCost() returned error: %v", err)
	}

	// Should fall back to x86 pricing when ARM not available
	if resp.UnitPrice != 0.0000166667 {
		t.Errorf("UnitPrice = %v, want 0.0000166667 (fallback to x86)", resp.UnitPrice)
	}
}

// ============================================================================
// Carbon Estimation Tests (T017-T019)
// ============================================================================

// TestGetProjectedCost_EC2_WithCarbonMetrics tests that EC2 responses include carbon metrics (T017)
func TestGetProjectedCost_EC2_WithCarbonMetrics(t *testing.T) {
	mock := newMockPricingClient("us-east-1", "USD")
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	mock.ec2Prices["t3.micro/Linux/Shared"] = 0.0104
	plugin := NewAWSPublicPlugin("us-east-1", mock, logger)

	resp, err := plugin.GetProjectedCost(context.Background(), &pbc.GetProjectedCostRequest{
		Resource: &pbc.ResourceDescriptor{
			Provider:     "aws",
			ResourceType: "ec2",
			Sku:          "t3.micro",
			Region:       "us-east-1",
		},
	})

	if err != nil {
		t.Fatalf("GetProjectedCost() returned error: %v", err)
	}

	// Verify financial cost is still present
	expectedCost := 0.0104 * 730.0
	if resp.CostPerMonth != expectedCost {
		t.Errorf("CostPerMonth = %v, want %v", resp.CostPerMonth, expectedCost)
	}

	// Verify carbon metrics are present
	if len(resp.ImpactMetrics) == 0 {
		t.Fatal("ImpactMetrics should not be empty for known EC2 instance type")
	}

	// Find carbon footprint metric
	var carbonMetric *pbc.ImpactMetric
	for _, m := range resp.ImpactMetrics {
		if m.Kind == pbc.MetricKind_METRIC_KIND_CARBON_FOOTPRINT {
			carbonMetric = m
			break
		}
	}

	if carbonMetric == nil {
		t.Fatal("ImpactMetrics should contain METRIC_KIND_CARBON_FOOTPRINT")
	}

	// Verify carbon value is reasonable for t3.micro monthly in us-east-1
	// Expected ~3500 gCO2e based on CCF formula; allow 2000-5000 range for variance
	if carbonMetric.Value < 2000 || carbonMetric.Value > 5000 {
		t.Errorf("Carbon value = %v, want between 2000 and 5000 gCO2e", carbonMetric.Value)
	}

	// Verify unit is correct
	if carbonMetric.Unit != "gCO2e" {
		t.Errorf("Carbon unit = %q, want %q", carbonMetric.Unit, "gCO2e")
	}
}

// TestGetProjectedCost_EC2_CarbonZeroForUnknownInstance tests that carbon is 0 for unknown instance types (T018)
func TestGetProjectedCost_EC2_CarbonZeroForUnknownInstance(t *testing.T) {
	mock := newMockPricingClient("us-east-1", "USD")
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	mock.ec2Prices["unknown.instance/Linux/Shared"] = 0.01 // Add pricing so financial cost works
	plugin := NewAWSPublicPlugin("us-east-1", mock, logger)

	resp, err := plugin.GetProjectedCost(context.Background(), &pbc.GetProjectedCostRequest{
		Resource: &pbc.ResourceDescriptor{
			Provider:     "aws",
			ResourceType: "ec2",
			Sku:          "unknown.instance",
			Region:       "us-east-1",
		},
	})

	if err != nil {
		t.Fatalf("GetProjectedCost() returned error: %v", err)
	}

	// Financial cost should still work
	if resp.CostPerMonth == 0 {
		t.Error("CostPerMonth should be non-zero for instance with pricing")
	}

	// Carbon metrics should be empty for unknown instance type
	if len(resp.ImpactMetrics) > 0 {
		t.Errorf("ImpactMetrics should be empty for unknown instance type, got %d metrics", len(resp.ImpactMetrics))
	}
}

// TestGetProjectedCost_EC2_RegionAffectsCarbon tests that region affects carbon value (T019)
func TestGetProjectedCost_EC2_RegionAffectsCarbon(t *testing.T) {
	// Test with us-east-1 plugin
	mockUSEast := newMockPricingClient("us-east-1", "USD")
	mockUSEast.ec2Prices["t3.micro/Linux/Shared"] = 0.0104
	loggerUSEast := zerolog.New(nil).Level(zerolog.InfoLevel)
	pluginUSEast := NewAWSPublicPlugin("us-east-1", mockUSEast, loggerUSEast)

	respUSEast, err := pluginUSEast.GetProjectedCost(context.Background(), &pbc.GetProjectedCostRequest{
		Resource: &pbc.ResourceDescriptor{
			Provider:     "aws",
			ResourceType: "ec2",
			Sku:          "t3.micro",
			Region:       "us-east-1",
		},
	})
	if err != nil {
		t.Fatalf("GetProjectedCost(us-east-1) error: %v", err)
	}

	// Test with eu-north-1 plugin (Sweden - very low carbon grid)
	mockEUNorth := newMockPricingClient("eu-north-1", "USD")
	mockEUNorth.ec2Prices["t3.micro/Linux/Shared"] = 0.0116
	loggerEUNorth := zerolog.New(nil).Level(zerolog.InfoLevel)
	pluginEUNorth := NewAWSPublicPlugin("eu-north-1", mockEUNorth, loggerEUNorth)

	respEUNorth, err := pluginEUNorth.GetProjectedCost(context.Background(), &pbc.GetProjectedCostRequest{
		Resource: &pbc.ResourceDescriptor{
			Provider:     "aws",
			ResourceType: "ec2",
			Sku:          "t3.micro",
			Region:       "eu-north-1",
		},
	})
	if err != nil {
		t.Fatalf("GetProjectedCost(eu-north-1) error: %v", err)
	}

	// Both should have carbon metrics
	if len(respUSEast.ImpactMetrics) == 0 {
		t.Fatal("us-east-1 should have ImpactMetrics")
	}
	if len(respEUNorth.ImpactMetrics) == 0 {
		t.Fatal("eu-north-1 should have ImpactMetrics")
	}

	var carbonUSEast, carbonEUNorth float64
	for _, m := range respUSEast.ImpactMetrics {
		if m.Kind == pbc.MetricKind_METRIC_KIND_CARBON_FOOTPRINT {
			carbonUSEast = m.Value
		}
	}
	for _, m := range respEUNorth.ImpactMetrics {
		if m.Kind == pbc.MetricKind_METRIC_KIND_CARBON_FOOTPRINT {
			carbonEUNorth = m.Value
		}
	}

	// EU North (Sweden) should have much lower carbon than US East (Virginia)
	// Grid factor ratio is roughly 43x (0.000379 / 0.0000088)
	// We use 30x as threshold to allow some margin while validating the ratio is significant
	if carbonUSEast <= carbonEUNorth*30 {
		t.Errorf("us-east-1 carbon (%v) should be at least 30x higher than eu-north-1 carbon (%v)",
			carbonUSEast, carbonEUNorth)
	}

	t.Logf("Carbon comparison: us-east-1=%v gCO2e, eu-north-1=%v gCO2e (ratio: %.1fx)",
		carbonUSEast, carbonEUNorth, carbonUSEast/carbonEUNorth)
}

// TestGetProjectedCost_EC2_RequestLevelUtilization tests request-level utilization override (T031)
func TestGetProjectedCost_EC2_RequestLevelUtilization(t *testing.T) {
	mockClient := newMockPricingClient("us-east-1", "USD")
	mockClient.ec2Prices["t3.micro/Linux/Shared"] = 0.0104
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	plugin := NewAWSPublicPlugin("us-east-1", mockClient, logger)

	// Request with high utilization (80%)
	respHigh, err := plugin.GetProjectedCost(context.Background(), &pbc.GetProjectedCostRequest{
		UtilizationPercentage: 0.8,
		Resource: &pbc.ResourceDescriptor{
			Provider:     "aws",
			ResourceType: "ec2",
			Sku:          "t3.micro",
			Region:       "us-east-1",
		},
	})
	if err != nil {
		t.Fatalf("GetProjectedCost(high util) error: %v", err)
	}

	// Request with low utilization (20%)
	respLow, err := plugin.GetProjectedCost(context.Background(), &pbc.GetProjectedCostRequest{
		UtilizationPercentage: 0.2,
		Resource: &pbc.ResourceDescriptor{
			Provider:     "aws",
			ResourceType: "ec2",
			Sku:          "t3.micro",
			Region:       "us-east-1",
		},
	})
	if err != nil {
		t.Fatalf("GetProjectedCost(low util) error: %v", err)
	}

	// Extract carbon values
	var carbonHigh, carbonLow float64
	for _, m := range respHigh.ImpactMetrics {
		if m.Kind == pbc.MetricKind_METRIC_KIND_CARBON_FOOTPRINT {
			carbonHigh = m.Value
		}
	}
	for _, m := range respLow.ImpactMetrics {
		if m.Kind == pbc.MetricKind_METRIC_KIND_CARBON_FOOTPRINT {
			carbonLow = m.Value
		}
	}

	// Higher utilization should result in higher carbon
	if carbonHigh <= carbonLow {
		t.Errorf("high utilization carbon (%v) should be greater than low utilization carbon (%v)",
			carbonHigh, carbonLow)
	}

	t.Logf("Utilization impact: 80%%=%v gCO2e, 20%%=%v gCO2e", carbonHigh, carbonLow)
}

// TestGetProjectedCost_EC2_PerResourceUtilization tests per-resource utilization override (T032)
func TestGetProjectedCost_EC2_PerResourceUtilization(t *testing.T) {
	mockClient := newMockPricingClient("us-east-1", "USD")
	mockClient.ec2Prices["t3.micro/Linux/Shared"] = 0.0104
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	plugin := NewAWSPublicPlugin("us-east-1", mockClient, logger)

	perResourceUtil := 0.9 // 90% utilization
	resp, err := plugin.GetProjectedCost(context.Background(), &pbc.GetProjectedCostRequest{
		UtilizationPercentage: 0.5, // This should be overridden by per-resource
		Resource: &pbc.ResourceDescriptor{
			Provider:              "aws",
			ResourceType:          "ec2",
			Sku:                   "t3.micro",
			Region:                "us-east-1",
			UtilizationPercentage: &perResourceUtil,
		},
	})
	if err != nil {
		t.Fatalf("GetProjectedCost error: %v", err)
	}

	// Also test with default (no override)
	respDefault, err := plugin.GetProjectedCost(context.Background(), &pbc.GetProjectedCostRequest{
		Resource: &pbc.ResourceDescriptor{
			Provider:     "aws",
			ResourceType: "ec2",
			Sku:          "t3.micro",
			Region:       "us-east-1",
		},
	})
	if err != nil {
		t.Fatalf("GetProjectedCost(default) error: %v", err)
	}

	// Extract carbon values
	var carbonWithOverride, carbonDefault float64
	for _, m := range resp.ImpactMetrics {
		if m.Kind == pbc.MetricKind_METRIC_KIND_CARBON_FOOTPRINT {
			carbonWithOverride = m.Value
		}
	}
	for _, m := range respDefault.ImpactMetrics {
		if m.Kind == pbc.MetricKind_METRIC_KIND_CARBON_FOOTPRINT {
			carbonDefault = m.Value
		}
	}

	// 90% utilization should produce more carbon than default 50%
	if carbonWithOverride <= carbonDefault {
		t.Errorf("90%% utilization carbon (%v) should be greater than default 50%% carbon (%v)",
			carbonWithOverride, carbonDefault)
	}

	t.Logf("Per-resource override: 90%%=%v gCO2e, default 50%%=%v gCO2e", carbonWithOverride, carbonDefault)
}

// TestGetProjectedCost_EC2_UtilizationPriority tests utilization priority order (T033)
func TestGetProjectedCost_EC2_UtilizationPriority(t *testing.T) {
	mockClient := newMockPricingClient("us-east-1", "USD")
	mockClient.ec2Prices["t3.micro/Linux/Shared"] = 0.0104
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	plugin := NewAWSPublicPlugin("us-east-1", mockClient, logger)

	// Test 1: Per-resource should override request-level
	perResourceUtil := 0.95 // 95%
	respPerResource, err := plugin.GetProjectedCost(context.Background(), &pbc.GetProjectedCostRequest{
		UtilizationPercentage: 0.2, // This should be ignored
		Resource: &pbc.ResourceDescriptor{
			Provider:              "aws",
			ResourceType:          "ec2",
			Sku:                   "t3.micro",
			Region:                "us-east-1",
			UtilizationPercentage: &perResourceUtil,
		},
	})
	if err != nil {
		t.Fatalf("GetProjectedCost(perResource) error: %v", err)
	}

	// Test 2: Request-level with no per-resource
	respRequest, err := plugin.GetProjectedCost(context.Background(), &pbc.GetProjectedCostRequest{
		UtilizationPercentage: 0.95, // Same as per-resource above for comparison
		Resource: &pbc.ResourceDescriptor{
			Provider:     "aws",
			ResourceType: "ec2",
			Sku:          "t3.micro",
			Region:       "us-east-1",
		},
	})
	if err != nil {
		t.Fatalf("GetProjectedCost(request) error: %v", err)
	}

	// Extract carbon values
	var carbonPerResource, carbonRequest float64
	for _, m := range respPerResource.ImpactMetrics {
		if m.Kind == pbc.MetricKind_METRIC_KIND_CARBON_FOOTPRINT {
			carbonPerResource = m.Value
		}
	}
	for _, m := range respRequest.ImpactMetrics {
		if m.Kind == pbc.MetricKind_METRIC_KIND_CARBON_FOOTPRINT {
			carbonRequest = m.Value
		}
	}

	// Both should use 95% utilization, so carbon should be approximately equal
	// Allow 1% tolerance for floating point
	tolerance := carbonPerResource * 0.01
	diff := carbonPerResource - carbonRequest
	if diff < 0 {
		diff = -diff
	}
	if diff > tolerance {
		t.Errorf("per-resource priority not working: per-resource carbon (%v) != request carbon (%v)",
			carbonPerResource, carbonRequest)
	}

	t.Logf("Priority verification: per-resource(95%%)=%v gCO2e, request(95%%)=%v gCO2e",
		carbonPerResource, carbonRequest)
}

// TestGetProjectedCost_EC2_GPUInstance tests GPU instance types still return financial cost (T044)
// GPU power consumption is not included in carbon estimates for v1.
func TestGetProjectedCost_EC2_GPUInstance(t *testing.T) {
	mockClient := newMockPricingClient("us-east-1", "USD")
	// GPU instance pricing
	mockClient.ec2Prices["p3.2xlarge/Linux/Shared"] = 3.06
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	plugin := NewAWSPublicPlugin("us-east-1", mockClient, logger)

	resp, err := plugin.GetProjectedCost(context.Background(), &pbc.GetProjectedCostRequest{
		Resource: &pbc.ResourceDescriptor{
			Provider:     "aws",
			ResourceType: "ec2",
			Sku:          "p3.2xlarge", // NVIDIA V100 GPU instance
			Region:       "us-east-1",
		},
	})
	if err != nil {
		t.Fatalf("GetProjectedCost(GPU) error: %v", err)
	}

	// Financial cost should still be returned
	if resp.CostPerMonth < 2000 {
		t.Errorf("GPU instance cost should be > $2000/month, got %v", resp.CostPerMonth)
	}

	// Carbon metrics may or may not be present depending on CCF data
	// If present, they won't include GPU power (known limitation)
	var hasCarbon bool
	for _, m := range resp.ImpactMetrics {
		if m.Kind == pbc.MetricKind_METRIC_KIND_CARBON_FOOTPRINT {
			hasCarbon = true
			t.Logf("GPU instance carbon: %v gCO2e (CPU only, GPU power not included)", m.Value)
		}
	}

	t.Logf("p3.2xlarge: $%.2f/month, carbon metrics present=%v", resp.CostPerMonth, hasCarbon)
}

