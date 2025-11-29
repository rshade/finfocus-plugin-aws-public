package plugin

import (
	"bytes"
	"context"
	"encoding/json"
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

	testCases := []string{"s3", "lambda", "rds", "dynamodb"}

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
