package plugin

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog"
	pbc "github.com/rshade/pulumicost-spec/sdk/go/proto/pulumicost/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// mockPricingClientActual implements pricing.PricingClient for actual cost testing.
type mockPricingClientActual struct {
	region            string
	ec2Prices         map[string]float64
	ebsPrices         map[string]float64
	s3Prices          map[string]float64
	rdsInstancePrices map[string]float64
	rdsStoragePrices  map[string]float64
}

func (m *mockPricingClientActual) Region() string {
	return m.region
}

func (m *mockPricingClientActual) Currency() string {
	return "USD"
}

func (m *mockPricingClientActual) EC2OnDemandPricePerHour(instanceType, _, _ string) (float64, bool) {
	price, ok := m.ec2Prices[instanceType]
	return price, ok
}

func (m *mockPricingClientActual) EBSPricePerGBMonth(volumeType string) (float64, bool) {
	price, ok := m.ebsPrices[volumeType]
	return price, ok
}

func (m *mockPricingClientActual) S3PricePerGBMonth(storageClass string) (float64, bool) {
	price, ok := m.s3Prices[storageClass]
	return price, ok
}

func (m *mockPricingClientActual) RDSOnDemandPricePerHour(instanceType, engine string) (float64, bool) {
	if m.rdsInstancePrices == nil {
		return 0, false
	}
	key := instanceType + "/" + engine
	price, ok := m.rdsInstancePrices[key]
	return price, ok
}

func (m *mockPricingClientActual) RDSStoragePricePerGBMonth(volumeType string) (float64, bool) {
	if m.rdsStoragePrices == nil {
		return 0, false
	}
	price, ok := m.rdsStoragePrices[volumeType]
	return price, ok
}

func (m *mockPricingClientActual) EKSClusterPricePerHour(extendedSupport bool) (float64, bool) {
	if extendedSupport {
		return 0.50, true // Extended EKS rate
	}
	return 0.10, true // Standard EKS rate
}

func newTestPluginForActual() *AWSPublicPlugin {
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	return NewAWSPublicPlugin("us-east-1", &mockPricingClientActual{
		region: "us-east-1",
		ec2Prices: map[string]float64{
			"t3.micro": 0.0104, // $7.592/month
			"m5.large": 0.096,  // $70.08/month
		},
		ebsPrices: map[string]float64{
			"gp3": 0.08, // $0.08/GB-month
			"gp2": 0.10, // $0.10/GB-month
		},
	}, logger)
}

// makeResourceJSON creates a JSON-encoded ResourceDescriptor for testing.
func makeResourceJSON(provider, resourceType, sku, region string, tags map[string]string) string {
	rd := map[string]interface{}{
		"provider":      provider,
		"resource_type": resourceType,
		"sku":           sku,
		"region":        region,
	}
	if tags != nil {
		rd["tags"] = tags
	}
	b, _ := json.Marshal(rd)
	return string(b)
}

// TestCalculateRuntimeHours tests the runtime hours calculation helper.
func TestCalculateRuntimeHours(t *testing.T) {
	tests := []struct {
		name        string
		from        time.Time
		to          time.Time
		wantHours   float64
		wantErr     bool
		errContains string
	}{
		{
			name:      "24 hours",
			from:      time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			to:        time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC),
			wantHours: 24.0,
			wantErr:   false,
		},
		{
			name:      "1 week (168 hours)",
			from:      time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			to:        time.Date(2025, 1, 8, 0, 0, 0, 0, time.UTC),
			wantHours: 168.0,
			wantErr:   false,
		},
		{
			name:      "8 hours same day",
			from:      time.Date(2025, 1, 1, 9, 0, 0, 0, time.UTC),
			to:        time.Date(2025, 1, 1, 17, 0, 0, 0, time.UTC),
			wantHours: 8.0,
			wantErr:   false,
		},
		{
			name:      "zero duration",
			from:      time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC),
			to:        time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC),
			wantHours: 0.0,
			wantErr:   false,
		},
		{
			name:      "fractional hours (1.5)",
			from:      time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC),
			to:        time.Date(2025, 1, 1, 11, 30, 0, 0, time.UTC),
			wantHours: 1.5,
			wantErr:   false,
		},
		{
			name:        "invalid range (from > to)",
			from:        time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC),
			to:          time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			wantHours:   0,
			wantErr:     true,
			errContains: "invalid time range",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hours, err := calculateRuntimeHours(tt.from, tt.to)
			if tt.wantErr {
				if err == nil {
					t.Errorf("calculateRuntimeHours() expected error, got nil")
				} else if tt.errContains != "" && !containsString(err.Error(), tt.errContains) {
					t.Errorf("calculateRuntimeHours() error = %v, want containing %q", err, tt.errContains)
				}
				return
			}
			if err != nil {
				t.Errorf("calculateRuntimeHours() unexpected error: %v", err)
				return
			}
			if hours != tt.wantHours {
				t.Errorf("calculateRuntimeHours() = %v, want %v", hours, tt.wantHours)
			}
		})
	}
}

// TestGetActualCostEC2 tests actual cost calculation for EC2 instances.
func TestGetActualCostEC2(t *testing.T) {
	plugin := newTestPluginForActual()
	ctx := context.Background()

	tests := []struct {
		name         string
		instanceType string
		runtimeHours float64
		wantCost     float64
		wantErr      bool
	}{
		{
			name:         "t3.micro 24 hours",
			instanceType: "t3.micro",
			runtimeHours: 24,
			// $0.0104/hr * 730 hrs = $7.592/month
			// $7.592 * (24/730) = $0.2496
			wantCost: 0.2496,
			wantErr:  false,
		},
		{
			name:         "t3.micro 168 hours (1 week)",
			instanceType: "t3.micro",
			runtimeHours: 168,
			// $7.592 * (168/730) = $1.7472
			wantCost: 1.7472,
			wantErr:  false,
		},
		{
			name:         "m5.large 24 hours",
			instanceType: "m5.large",
			runtimeHours: 24,
			// $0.096/hr * 730 hrs = $70.08/month
			// $70.08 * (24/730) = $2.304
			wantCost: 2.304,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			from := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
			to := from.Add(time.Duration(tt.runtimeHours) * time.Hour)

			// Use Tags to pass resource info (actual proto structure)
			req := &pbc.GetActualCostRequest{
				ResourceId: makeResourceJSON("aws", "ec2", tt.instanceType, "us-east-1", nil),
				Start:      timestamppb.New(from),
				End:        timestamppb.New(to),
			}

			resp, err := plugin.GetActualCost(ctx, req)
			if tt.wantErr {
				if err == nil {
					t.Errorf("GetActualCost() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("GetActualCost() unexpected error: %v", err)
				return
			}
			if resp == nil {
				t.Errorf("GetActualCost() returned nil response")
				return
			}
			if len(resp.Results) == 0 {
				t.Errorf("GetActualCost() returned empty results")
				return
			}

			result := resp.Results[0]

			// Allow 0.01% tolerance for floating point
			tolerance := tt.wantCost * 0.0001
			if diff := result.Cost - tt.wantCost; diff > tolerance || diff < -tolerance {
				t.Errorf("GetActualCost() cost = %v, want %v (tolerance %v)", result.Cost, tt.wantCost, tolerance)
			}

			if result.Source == "" {
				t.Errorf("GetActualCost() source (billing_detail) is empty")
			}
		})
	}
}

// TestGetActualCostEC2_PulumiFormat tests actual cost calculation for EC2 instances
// using Pulumi resource type format.
func TestGetActualCostEC2_PulumiFormat(t *testing.T) {
	plugin := newTestPluginForActual()
	ctx := context.Background()

	from := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	to := from.Add(24 * time.Hour) // 24 hours

	// Use Tags to pass resource info with Pulumi resource type format
	req := &pbc.GetActualCostRequest{
		ResourceId: makeResourceJSON("aws", "aws:ec2/instance:Instance", "t3.micro", "us-east-1", nil),
		Start:      timestamppb.New(from),
		End:        timestamppb.New(to),
	}

	resp, err := plugin.GetActualCost(ctx, req)
	if err != nil {
		t.Fatalf("GetActualCost() with Pulumi format failed: %v", err)
	}

	if resp == nil {
		t.Errorf("GetActualCost() returned nil response")
		return
	}
	if len(resp.Results) == 0 {
		t.Errorf("GetActualCost() returned empty results")
		return
	}

	result := resp.Results[0]

	// $0.0104/hr * 730 hrs = $7.592/month
	// $7.592 * (24/730) = $0.2496
	expectedCost := 0.2496
	tolerance := expectedCost * 0.0001 // 0.01% tolerance
	if diff := result.Cost - expectedCost; diff > tolerance || diff < -tolerance {
		t.Errorf("GetActualCost() cost = %v, want %v (tolerance %v)", result.Cost, expectedCost, tolerance)
	}

	if result.Source == "" {
		t.Errorf("GetActualCost() source (billing_detail) is empty")
	}
}

// TestGetActualCostEBS_PulumiFormat tests actual cost calculation for EBS volumes
// using Pulumi resource type format.
func TestGetActualCostEBS_PulumiFormat(t *testing.T) {
	plugin := newTestPluginForActual()
	ctx := context.Background()

	from := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	to := from.Add(168 * time.Hour) // 1 week

	// Use ResourceId JSON with tags for size and Pulumi resource type format
	req := &pbc.GetActualCostRequest{
		ResourceId: makeResourceJSON("aws", "aws:ebs/volume:Volume", "gp3", "us-east-1", map[string]string{"size": "100"}),
		Start:      timestamppb.New(from),
		End:        timestamppb.New(to),
	}

	resp, err := plugin.GetActualCost(ctx, req)
	if err != nil {
		t.Fatalf("GetActualCost() with EBS Pulumi format failed: %v", err)
	}

	if resp == nil {
		t.Errorf("GetActualCost() returned nil response")
		return
	}
	if len(resp.Results) == 0 {
		t.Errorf("GetActualCost() returned empty results")
		return
	}

	result := resp.Results[0]

	// $0.08/GB-month * 100GB = $8.00/month
	// $8.00 * (168/730) = $1.8411
	expectedCost := 1.8410958904109589
	tolerance := expectedCost * 0.0001 // 0.01% tolerance
	if diff := result.Cost - expectedCost; diff > tolerance || diff < -tolerance {
		t.Errorf("GetActualCost() cost = %v, want %v (tolerance %v)", result.Cost, expectedCost, tolerance)
	}

	if result.Source == "" {
		t.Errorf("GetActualCost() source (billing_detail) is empty")
	}
}

// TestGetActualCostEBS tests actual cost calculation for EBS volumes.
func TestGetActualCostEBS(t *testing.T) {
	plugin := newTestPluginForActual()
	ctx := context.Background()

	tests := []struct {
		name         string
		volumeType   string
		sizeGB       string
		runtimeHours float64
		wantCost     float64
	}{
		{
			name:         "gp3 100GB 168 hours",
			volumeType:   "gp3",
			sizeGB:       "100",
			runtimeHours: 168,
			// $0.08/GB-month * 100GB = $8.00/month
			// $8.00 * (168/730) = $1.8411
			wantCost: 1.8410958904109589,
		},
		{
			name:         "gp2 50GB 24 hours",
			volumeType:   "gp2",
			sizeGB:       "50",
			runtimeHours: 24,
			// $0.10/GB-month * 50GB = $5.00/month
			// $5.00 * (24/730) = $0.1644
			wantCost: 0.16438356164383562,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			from := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
			to := from.Add(time.Duration(tt.runtimeHours) * time.Hour)

			// Use ResourceId JSON with tags for size
			req := &pbc.GetActualCostRequest{
				ResourceId: makeResourceJSON("aws", "ebs", tt.volumeType, "us-east-1", map[string]string{"size": tt.sizeGB}),
				Start:      timestamppb.New(from),
				End:        timestamppb.New(to),
			}

			resp, err := plugin.GetActualCost(ctx, req)
			if err != nil {
				t.Errorf("GetActualCost() unexpected error: %v", err)
				return
			}
			if resp == nil {
				t.Errorf("GetActualCost() returned nil response")
				return
			}
			if len(resp.Results) == 0 {
				t.Errorf("GetActualCost() returned empty results")
				return
			}

			result := resp.Results[0]

			// Allow 0.01% tolerance for floating point
			tolerance := tt.wantCost * 0.0001
			if diff := result.Cost - tt.wantCost; diff > tolerance || diff < -tolerance {
				t.Errorf("GetActualCost() cost = %v, want %v", result.Cost, tt.wantCost)
			}
		})
	}
}

// TestGetActualCostInvalidRange tests error handling for invalid time ranges.
func TestGetActualCostInvalidRange(t *testing.T) {
	plugin := newTestPluginForActual()
	ctx := context.Background()

	from := time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC)
	to := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC) // Before from

	req := &pbc.GetActualCostRequest{
		ResourceId: makeResourceJSON("aws", "ec2", "t3.micro", "us-east-1", nil),
		Start:      timestamppb.New(from),
		End:        timestamppb.New(to),
	}

	_, err := plugin.GetActualCost(ctx, req)
	if err == nil {
		t.Errorf("GetActualCost() expected error for invalid time range, got nil")
	}
}

// TestGetActualCostNilTimestamps tests error handling for nil timestamps.
func TestGetActualCostNilTimestamps(t *testing.T) {
	plugin := newTestPluginForActual()
	ctx := context.Background()

	tests := []struct {
		name  string
		start *timestamppb.Timestamp
		end   *timestamppb.Timestamp
	}{
		{
			name:  "nil start",
			start: nil,
			end:   timestamppb.New(time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC)),
		},
		{
			name:  "nil end",
			start: timestamppb.New(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)),
			end:   nil,
		},
		{
			name:  "both nil",
			start: nil,
			end:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &pbc.GetActualCostRequest{
				ResourceId: makeResourceJSON("aws", "ec2", "t3.micro", "us-east-1", nil),
				Start:      tt.start,
				End:        tt.end,
			}

			_, err := plugin.GetActualCost(ctx, req)
			if err == nil {
				t.Errorf("GetActualCost() expected error for nil timestamp, got nil")
			}
		})
	}
}

// TestGetActualCostZeroDuration tests handling of zero-duration time ranges.
func TestGetActualCostZeroDuration(t *testing.T) {
	plugin := newTestPluginForActual()
	ctx := context.Background()

	ts := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	req := &pbc.GetActualCostRequest{
		ResourceId: makeResourceJSON("aws", "ec2", "t3.micro", "us-east-1", nil),
		Start:      timestamppb.New(ts),
		End:        timestamppb.New(ts),
	}

	resp, err := plugin.GetActualCost(ctx, req)
	if err != nil {
		t.Errorf("GetActualCost() unexpected error: %v", err)
		return
	}
	if resp == nil {
		t.Errorf("GetActualCost() returned nil response")
		return
	}
	if len(resp.Results) == 0 {
		t.Errorf("GetActualCost() returned empty results")
		return
	}

	result := resp.Results[0]
	if result.Cost != 0 {
		t.Errorf("GetActualCost() cost = %v, want 0 for zero duration", result.Cost)
	}
}

// TestGetActualCostStubServices tests stub service responses.
func TestGetActualCostStubServices(t *testing.T) {
	plugin := newTestPluginForActual()
	ctx := context.Background()

	stubServices := []string{"s3", "lambda", "dynamodb"}

	for _, service := range stubServices {
		t.Run(service, func(t *testing.T) {
			from := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
			to := time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC)

			req := &pbc.GetActualCostRequest{
				ResourceId: makeResourceJSON("aws", service, "test-sku", "us-east-1", nil),
				Start:      timestamppb.New(from),
				End:        timestamppb.New(to),
			}

			resp, err := plugin.GetActualCost(ctx, req)
			if err != nil {
				t.Errorf("GetActualCost() unexpected error for %s: %v", service, err)
				return
			}
			if resp == nil {
				t.Errorf("GetActualCost() returned nil response for %s", service)
				return
			}
			if len(resp.Results) == 0 {
				t.Errorf("GetActualCost() returned empty results for %s", service)
				return
			}

			result := resp.Results[0]
			if result.Cost != 0 {
				t.Errorf("GetActualCost() cost = %v, want 0 for stub service %s", result.Cost, service)
			}
			if result.Source == "" {
				t.Errorf("GetActualCost() source (billing_detail) is empty for %s", service)
			}
		})
	}
}

// BenchmarkGetActualCost benchmarks the GetActualCost method to verify SC-003.
// Target: < 10ms per request.
func BenchmarkGetActualCost(b *testing.B) {
	plugin := newTestPluginForActual()
	ctx := context.Background()

	from := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC)

	req := &pbc.GetActualCostRequest{
		ResourceId: makeResourceJSON("aws", "ec2", "t3.micro", "us-east-1", nil),
		Start:      timestamppb.New(from),
		End:        timestamppb.New(to),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = plugin.GetActualCost(ctx, req)
	}
}

// TestGetActualCost_ConcurrentCalls tests thread safety with 100+ parallel GetActualCost calls.
// This validates concurrent RPC handling per coding guidelines requirement.
func TestGetActualCost_ConcurrentCalls(t *testing.T) {
	plugin := newTestPluginForActual()

	const numGoroutines = 20
	const callsPerGoroutine = 5
	totalCalls := numGoroutines * callsPerGoroutine

	errors := make(chan error, totalCalls)
	done := make(chan bool, totalCalls)

	from := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC)

	// Launch concurrent goroutines making GetActualCost calls
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			for j := 0; j < callsPerGoroutine; j++ {
				var req *pbc.GetActualCostRequest

				// Alternate between EC2 and EBS requests
				if (id+j)%2 == 0 {
					req = &pbc.GetActualCostRequest{
						ResourceId: makeResourceJSON("aws", "ec2", "t3.micro", "us-east-1", nil),
						Start:      timestamppb.New(from),
						End:        timestamppb.New(to),
					}
				} else {
					req = &pbc.GetActualCostRequest{
						ResourceId: makeResourceJSON("aws", "ebs", "gp3", "us-east-1", map[string]string{"size": "100"}),
						Start:      timestamppb.New(from),
						End:        timestamppb.New(to),
					}
				}

				_, err := plugin.GetActualCost(context.Background(), req)
				if err != nil {
					errors <- err
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
			t.Errorf("Concurrent GetActualCost call failed: %v", err)
		}
	}

	if errorCount > 0 {
		t.Errorf("Failed %d out of %d concurrent GetActualCost calls", errorCount, totalCalls)
	}

	t.Logf("Successfully completed %d concurrent GetActualCost calls across %d goroutines", totalCalls, numGoroutines)
}

// containsString checks if substr is in s using the standard library.
func containsString(s, substr string) bool {
	return strings.Contains(s, substr)
}

// TestGetActualCostWithInvalidResourceJSON tests behavior when ResourceId is invalid JSON and Tags are missing.
func TestGetActualCostWithInvalidResourceJSON(t *testing.T) {
	plugin := newTestPluginForActual()
	ctx := context.Background()

	from := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	to := from.Add(1 * time.Hour)

	req := &pbc.GetActualCostRequest{
		ResourceId: "{invalid-json-garbage", // Malformed JSON
		Tags:       nil,                     // No tags to fallback to
		Start:      timestamppb.New(from),
		End:        timestamppb.New(to),
	}

	_, err := plugin.GetActualCost(ctx, req)
	if err == nil {
		t.Error("Expected error for invalid JSON ResourceId with no Tags, got nil")
	} else {
		// Should fail in parseResourceFromRequest
		if !strings.Contains(err.Error(), "missing resource information") {
			t.Errorf("Expected 'missing resource information' error, got: %v", err)
		}
	}
}
