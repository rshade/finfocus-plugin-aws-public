package plugin

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/rshade/pulumicost-plugin-aws-public/internal/pricing"
	pbc "github.com/rshade/pulumicost-spec/sdk/go/proto/pulumicost/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
	lambdaPrices      map[string]float64
	natgwHourlyPrice  float64
	natgwDataPrice    float64
}

func (m *mockPricingClientActual) Region() string {
	return m.region
}

func (m *mockPricingClientActual) Currency() string {
	return "USD"
}

func (m *mockPricingClientActual) LambdaPricePerRequest() (float64, bool) {
	if m.lambdaPrices == nil {
		return 0, false
	}
	price, ok := m.lambdaPrices["request"]
	return price, ok
}

func (m *mockPricingClientActual) LambdaPricePerGBSecond(arch string) (float64, bool) {
	if m.lambdaPrices == nil {
		return 0, false
	}
	// FR-011: Support ARM architecture pricing
	switch strings.ToLower(arch) {
	case "arm64", "arm":
		if price, found := m.lambdaPrices["gb-second-arm64"]; found {
			return price, true
		}
	}
	// Default to x86 pricing
	price, ok := m.lambdaPrices["gb-second"]
	return price, ok
}

func (m *mockPricingClientActual) DynamoDBOnDemandReadPrice() (float64, bool) {
	return 0.25 / 1_000_000, true
}

func (m *mockPricingClientActual) DynamoDBOnDemandWritePrice() (float64, bool) {
	return 1.25 / 1_000_000, true
}

func (m *mockPricingClientActual) DynamoDBStoragePricePerGBMonth() (float64, bool) {
	return 0.25, true
}

func (m *mockPricingClientActual) DynamoDBProvisionedRCUPrice() (float64, bool) {
	return 0.00013, true
}

func (m *mockPricingClientActual) DynamoDBProvisionedWCUPrice() (float64, bool) {
	return 0.00065, true
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

func (m *mockPricingClientActual) ALBPricePerHour() (float64, bool) {
	return 0.0225, true
}

func (m *mockPricingClientActual) ALBPricePerLCU() (float64, bool) {
	return 0.008, true
}

func (m *mockPricingClientActual) NLBPricePerHour() (float64, bool) {
	return 0.0225, true
}

func (m *mockPricingClientActual) NLBPricePerNLCU() (float64, bool) {
	return 0.006, true
}

func (m *mockPricingClientActual) NATGatewayPrice() (*pricing.NATGatewayPrice, bool) {
	if m.natgwHourlyPrice > 0 {
		return &pricing.NATGatewayPrice{
			HourlyRate:         m.natgwHourlyPrice,
			DataProcessingRate: m.natgwDataPrice,
			Currency:           "USD",
		}, true
	}
	return nil, false
}

func (m *mockPricingClientActual) CloudWatchLogsIngestionTiers() ([]pricing.TierRate, bool) {
	return nil, false
}

func (m *mockPricingClientActual) CloudWatchLogsStoragePrice() (float64, bool) {
	return 0, false
}

func (m *mockPricingClientActual) CloudWatchMetricsTiers() ([]pricing.TierRate, bool) {
	return nil, false
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

// TestMergeTagsFromRequest tests the mergeTagsFromRequest helper function.
// This function merges tags from req.Tags and ResourceId JSON, with req.Tags taking precedence.
func TestMergeTagsFromRequest(t *testing.T) {
	tests := []struct {
		name         string
		req          *pbc.GetActualCostRequest
		wantTags     map[string]string
		wantContains []string // Keys that must be present
	}{
		{
			name: "tags only from ResourceId JSON",
			req: &pbc.GetActualCostRequest{
				ResourceId: makeResourceJSON("aws", "ec2", "t3.micro", "us-east-1", map[string]string{
					TagPulumiCreated: "2025-01-01T00:00:00Z",
				}),
			},
			wantContains: []string{TagPulumiCreated},
		},
		{
			name: "tags only from req.Tags",
			req: &pbc.GetActualCostRequest{
				Tags: map[string]string{
					TagPulumiCreated: "2025-01-01T00:00:00Z",
				},
			},
			wantContains: []string{TagPulumiCreated},
		},
		{
			name: "req.Tags overrides ResourceId JSON",
			req: &pbc.GetActualCostRequest{
				ResourceId: makeResourceJSON("aws", "ec2", "t3.micro", "us-east-1", map[string]string{
					TagPulumiCreated: "2025-01-01T00:00:00Z", // Will be overridden
				}),
				Tags: map[string]string{
					TagPulumiCreated: "2025-06-15T00:00:00Z", // Takes precedence
				},
			},
			wantTags: map[string]string{
				TagPulumiCreated: "2025-06-15T00:00:00Z",
			},
		},
		{
			name: "merge tags from both sources",
			req: &pbc.GetActualCostRequest{
				ResourceId: makeResourceJSON("aws", "ec2", "t3.micro", "us-east-1", map[string]string{
					TagPulumiCreated:  "2025-01-01T00:00:00Z",
					TagPulumiExternal: "true",
				}),
				Tags: map[string]string{
					"custom_tag": "custom_value",
				},
			},
			wantContains: []string{TagPulumiCreated, TagPulumiExternal, "custom_tag"},
		},
		{
			name: "empty request",
			req:  &pbc.GetActualCostRequest{},
			wantTags: map[string]string{},
		},
		{
			name: "invalid JSON in ResourceId uses only req.Tags",
			req: &pbc.GetActualCostRequest{
				ResourceId: "not-valid-json",
				Tags: map[string]string{
					TagPulumiCreated: "2025-01-01T00:00:00Z",
				},
			},
			wantTags: map[string]string{
				TagPulumiCreated: "2025-01-01T00:00:00Z",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mergeTagsFromRequest(tt.req)

			// Check expected tags if specified
			if tt.wantTags != nil {
				for k, v := range tt.wantTags {
					if got[k] != v {
						t.Errorf("mergeTagsFromRequest()[%s] = %q, want %q", k, got[k], v)
					}
				}
			}

			// Check that required keys are present
			for _, key := range tt.wantContains {
				if _, exists := got[key]; !exists {
					t.Errorf("mergeTagsFromRequest() missing key %q", key)
				}
			}
		})
	}
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
// Feature 016 changed behavior: nil End now defaults to "now", but nil Start
// without pulumi:created tag still errors.
func TestGetActualCostNilTimestamps(t *testing.T) {
	plugin := newTestPluginForActual()
	ctx := context.Background()

	tests := []struct {
		name      string
		start     *timestamppb.Timestamp
		end       *timestamppb.Timestamp
		wantError bool // Feature 016: nil end is OK (defaults to now)
	}{
		{
			name:      "nil start without pulumi:created",
			start:     nil,
			end:       timestamppb.New(time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC)),
			wantError: true, // No pulumi:created tag to fall back to
		},
		{
			name:      "nil end",
			start:     timestamppb.New(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)),
			end:       nil,
			wantError: false, // Feature 016: defaults to now
		},
		{
			name:      "both nil without pulumi:created",
			start:     nil,
			end:       nil,
			wantError: true, // No pulumi:created tag to fall back to
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
			if tt.wantError {
				if err == nil {
					t.Errorf("GetActualCost() expected error for nil timestamp, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("GetActualCost() unexpected error: %v", err)
				}
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

	_, err := plugin.GetActualCost(ctx, req)
	if err == nil {
		t.Errorf("GetActualCost() expected error for zero duration (SDK enforcement), got nil")
		return
	}

	// Verify it's an InvalidArgument error
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.InvalidArgument {
		t.Errorf("GetActualCost() expected InvalidArgument error, got %v", err)
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

// TestExtractPulumiCreated tests the extractPulumiCreated helper function.
// This tests RFC3339 timestamp parsing from Pulumi state metadata tags.
func TestExtractPulumiCreated(t *testing.T) {
	tests := []struct {
		name      string
		tags      map[string]string
		wantTime  time.Time
		wantFound bool
	}{
		{
			name:      "nil tags",
			tags:      nil,
			wantTime:  time.Time{},
			wantFound: false,
		},
		{
			name:      "empty tags",
			tags:      map[string]string{},
			wantTime:  time.Time{},
			wantFound: false,
		},
		{
			name:      "missing pulumi:created key",
			tags:      map[string]string{"other": "value"},
			wantTime:  time.Time{},
			wantFound: false,
		},
		{
			name:      "empty pulumi:created value",
			tags:      map[string]string{TagPulumiCreated: ""},
			wantTime:  time.Time{},
			wantFound: false,
		},
		{
			name:      "valid RFC3339 timestamp",
			tags:      map[string]string{TagPulumiCreated: "2025-01-15T10:30:00Z"},
			wantTime:  time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
			wantFound: true,
		},
		{
			name:      "valid RFC3339 with timezone offset",
			tags:      map[string]string{TagPulumiCreated: "2025-01-15T10:30:00+05:00"},
			wantTime:  time.Date(2025, 1, 15, 10, 30, 0, 0, time.FixedZone("+05:00", 5*3600)),
			wantFound: true,
		},
		{
			name:      "invalid RFC3339 format",
			tags:      map[string]string{TagPulumiCreated: "2025-01-15 10:30:00"},
			wantTime:  time.Time{},
			wantFound: false,
		},
		{
			name:      "invalid RFC3339 garbage",
			tags:      map[string]string{TagPulumiCreated: "not-a-timestamp"},
			wantTime:  time.Time{},
			wantFound: false,
		},
		{
			name:      "Unix timestamp (invalid)",
			tags:      map[string]string{TagPulumiCreated: "1705314600"},
			wantTime:  time.Time{},
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotTime, gotFound := extractPulumiCreated(tt.tags)
			if gotFound != tt.wantFound {
				t.Errorf("extractPulumiCreated() found = %v, want %v", gotFound, tt.wantFound)
			}
			if tt.wantFound && !gotTime.Equal(tt.wantTime) {
				t.Errorf("extractPulumiCreated() time = %v, want %v", gotTime, tt.wantTime)
			}
		})
	}
}

// TestIsImportedResource tests the isImportedResource helper function.
// This validates case-sensitive comparison for pulumi:external=true.
func TestIsImportedResource(t *testing.T) {
	tests := []struct {
		name       string
		tags       map[string]string
		wantResult bool
	}{
		{
			name:       "nil tags",
			tags:       nil,
			wantResult: false,
		},
		{
			name:       "empty tags",
			tags:       map[string]string{},
			wantResult: false,
		},
		{
			name:       "missing pulumi:external key",
			tags:       map[string]string{"other": "value"},
			wantResult: false,
		},
		{
			name:       "pulumi:external=true (exact match)",
			tags:       map[string]string{TagPulumiExternal: "true"},
			wantResult: true,
		},
		{
			name:       "pulumi:external=True (case-sensitive, should fail)",
			tags:       map[string]string{TagPulumiExternal: "True"},
			wantResult: false,
		},
		{
			name:       "pulumi:external=TRUE (case-sensitive, should fail)",
			tags:       map[string]string{TagPulumiExternal: "TRUE"},
			wantResult: false,
		},
		{
			name:       "pulumi:external=false",
			tags:       map[string]string{TagPulumiExternal: "false"},
			wantResult: false,
		},
		{
			name:       "pulumi:external=yes (not a boolean)",
			tags:       map[string]string{TagPulumiExternal: "yes"},
			wantResult: false,
		},
		{
			name:       "pulumi:external=1 (not a boolean)",
			tags:       map[string]string{TagPulumiExternal: "1"},
			wantResult: false,
		},
		{
			name:       "pulumi:external=empty",
			tags:       map[string]string{TagPulumiExternal: ""},
			wantResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isImportedResource(tt.tags)
			if got != tt.wantResult {
				t.Errorf("isImportedResource() = %v, want %v", got, tt.wantResult)
			}
		})
	}
}

// TestResolveTimestamps tests the resolveTimestamps() function priority logic.
// This validates Feature 016 timestamp resolution: explicit > pulumi:created > error
func TestResolveTimestamps(t *testing.T) {
	// Helper time values
	explicitStart := time.Date(2025, 1, 10, 0, 0, 0, 0, time.UTC)
	explicitEnd := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
	pulumiCreated := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	pulumiCreatedStr := pulumiCreated.Format(time.RFC3339)

	tests := []struct {
		name       string
		req        *pbc.GetActualCostRequest
		wantStart  time.Time
		wantSource string
		wantError  bool
		errMsg     string
	}{
		{
			name: "explicit start and end - highest priority",
			req: &pbc.GetActualCostRequest{
				Start: timestamppb.New(explicitStart),
				End:   timestamppb.New(explicitEnd),
				Tags:  map[string]string{TagPulumiCreated: pulumiCreatedStr},
			},
			wantStart:  explicitStart,
			wantSource: "explicit",
			wantError:  false,
		},
		{
			name: "pulumi:created as fallback",
			req: &pbc.GetActualCostRequest{
				Start: nil,
				End:   timestamppb.New(explicitEnd),
				Tags:  map[string]string{TagPulumiCreated: pulumiCreatedStr},
			},
			wantStart:  pulumiCreated,
			wantSource: "mixed", // pulumi:created for start, explicit for end
			wantError:  false,
		},
		{
			name: "pulumi:created for start, now for end",
			req: &pbc.GetActualCostRequest{
				Start: nil,
				End:   nil,
				Tags:  map[string]string{TagPulumiCreated: pulumiCreatedStr},
			},
			wantStart:  pulumiCreated,
			wantSource: "pulumi:created",
			wantError:  false,
		},
		{
			name: "no timestamps and no pulumi:created - error",
			req: &pbc.GetActualCostRequest{
				Start: nil,
				End:   nil,
				Tags:  map[string]string{},
			},
			wantError: true,
			errMsg:    "start time required",
		},
		{
			name: "nil request - error",
			req:  nil,
			wantError: true,
			errMsg:    "request is nil",
		},
		{
			name: "invalid pulumi:created format - error",
			req: &pbc.GetActualCostRequest{
				Start: nil,
				End:   timestamppb.New(explicitEnd),
				Tags:  map[string]string{TagPulumiCreated: "invalid-timestamp"},
			},
			wantError: true,
			errMsg:    "start time required",
		},
		{
			name: "imported resource tracks flag",
			req: &pbc.GetActualCostRequest{
				Start: timestamppb.New(explicitStart),
				End:   timestamppb.New(explicitEnd),
				Tags:  map[string]string{TagPulumiExternal: "true"},
			},
			wantStart:  explicitStart,
			wantSource: "explicit",
			wantError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolution, err := resolveTimestamps(tt.req)
			if tt.wantError {
				if err == nil {
					t.Errorf("resolveTimestamps() expected error containing %q, got nil", tt.errMsg)
				} else if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("resolveTimestamps() error = %v, want containing %q", err, tt.errMsg)
				}
				return
			}
			if err != nil {
				t.Fatalf("resolveTimestamps() unexpected error: %v", err)
			}
			if resolution == nil {
				t.Fatal("resolveTimestamps() returned nil resolution")
			}
			if !resolution.Start.Equal(tt.wantStart) {
				t.Errorf("resolveTimestamps() Start = %v, want %v", resolution.Start, tt.wantStart)
			}
			if resolution.Source != tt.wantSource {
				t.Errorf("resolveTimestamps() Source = %q, want %q", resolution.Source, tt.wantSource)
			}
			// Check imported flag for the relevant test case
			if tt.name == "imported resource tracks flag" && !resolution.IsImported {
				t.Error("resolveTimestamps() IsImported = false, want true")
			}
		})
	}
}

// TestDetermineConfidence tests the determineConfidence() function.
// This validates Feature 016 confidence level determination.
func TestDetermineConfidence(t *testing.T) {
	tests := []struct {
		name       string
		resolution *TimestampResolution
		want       ConfidenceLevel
	}{
		{
			name:       "nil resolution - LOW",
			resolution: nil,
			want:       ConfidenceLow,
		},
		{
			name: "explicit timestamps - HIGH",
			resolution: &TimestampResolution{
				Source:     "explicit",
				IsImported: false,
			},
			want: ConfidenceHigh,
		},
		{
			name: "explicit timestamps with imported - HIGH (explicit wins)",
			resolution: &TimestampResolution{
				Source:     "explicit",
				IsImported: true,
			},
			want: ConfidenceHigh,
		},
		{
			name: "pulumi:created native resource - HIGH",
			resolution: &TimestampResolution{
				Source:     "pulumi:created",
				IsImported: false,
			},
			want: ConfidenceHigh,
		},
		{
			name: "pulumi:created imported resource - MEDIUM",
			resolution: &TimestampResolution{
				Source:     "pulumi:created",
				IsImported: true,
			},
			want: ConfidenceMedium,
		},
		{
			name: "mixed source native resource - HIGH",
			resolution: &TimestampResolution{
				Source:     "mixed",
				IsImported: false,
			},
			want: ConfidenceHigh,
		},
		{
			name: "mixed source imported resource - MEDIUM",
			resolution: &TimestampResolution{
				Source:     "mixed",
				IsImported: true,
			},
			want: ConfidenceMedium,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := determineConfidence(tt.resolution)
			if got != tt.want {
				t.Errorf("determineConfidence() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestRuntimeHoursZeroDuration tests the edge case where zero duration returns zero cost.
// This validates T012 requirement from tasks.md.
func TestRuntimeHoursZeroDuration(t *testing.T) {
	// Zero duration should return zero hours, not an error
	sameTime := time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC)
	hours, err := calculateRuntimeHours(sameTime, sameTime)
	if err != nil {
		t.Errorf("calculateRuntimeHours(same, same) unexpected error: %v", err)
	}
	if hours != 0 {
		t.Errorf("calculateRuntimeHours(same, same) = %v, want 0", hours)
	}
}

// TestResolveTimestampsPartialExplicit tests T028 - partial explicit scenarios.
// This validates mixed source handling: explicit start with pulumi:created end, etc.
func TestResolveTimestampsPartialExplicit(t *testing.T) {
	explicitStart := time.Date(2025, 1, 10, 0, 0, 0, 0, time.UTC)
	explicitEnd := time.Date(2025, 1, 20, 0, 0, 0, 0, time.UTC)
	pulumiCreated := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	pulumiCreatedStr := pulumiCreated.Format(time.RFC3339)

	tests := []struct {
		name       string
		start      *timestamppb.Timestamp
		end        *timestamppb.Timestamp
		tags       map[string]string
		wantStart  time.Time
		wantEnd    time.Time // Use zero value for "now" check
		wantSource string
	}{
		{
			name:       "explicit start only (end defaults to now)",
			start:      timestamppb.New(explicitStart),
			end:        nil,
			tags:       map[string]string{TagPulumiCreated: pulumiCreatedStr},
			wantStart:  explicitStart,
			wantSource: "mixed", // explicit start, default end
		},
		{
			name:       "explicit end only with pulumi:created",
			start:      nil,
			end:        timestamppb.New(explicitEnd),
			tags:       map[string]string{TagPulumiCreated: pulumiCreatedStr},
			wantStart:  pulumiCreated,
			wantEnd:    explicitEnd,
			wantSource: "mixed", // pulumi:created start, explicit end
		},
		{
			name:       "neither explicit with pulumi:created (end defaults to now)",
			start:      nil,
			end:        nil,
			tags:       map[string]string{TagPulumiCreated: pulumiCreatedStr},
			wantStart:  pulumiCreated,
			wantSource: "pulumi:created",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &pbc.GetActualCostRequest{
				Start: tt.start,
				End:   tt.end,
				Tags:  tt.tags,
			}
			resolution, err := resolveTimestamps(req)
			if err != nil {
				t.Fatalf("resolveTimestamps() unexpected error: %v", err)
			}
			if !resolution.Start.Equal(tt.wantStart) {
				t.Errorf("Start = %v, want %v", resolution.Start, tt.wantStart)
			}
			if resolution.Source != tt.wantSource {
				t.Errorf("Source = %q, want %q", resolution.Source, tt.wantSource)
			}
			// For explicit end, check exact match
			if !tt.wantEnd.IsZero() && !resolution.End.Equal(tt.wantEnd) {
				t.Errorf("End = %v, want %v", resolution.End, tt.wantEnd)
			}
			// For default end (now), just check it's after start
			if tt.wantEnd.IsZero() && !resolution.End.After(resolution.Start) {
				t.Errorf("End %v should be after Start %v", resolution.End, resolution.Start)
			}
		})
	}
}

// TestPulumiModifiedNotUsed validates T034 - pulumi:modified is NOT used as fallback.
// The feature explicitly only uses pulumi:created, not pulumi:modified.
func TestPulumiModifiedNotUsed(t *testing.T) {
	pulumiModified := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC).Format(time.RFC3339)

	req := &pbc.GetActualCostRequest{
		Start: nil,
		End:   timestamppb.New(time.Date(2025, 1, 20, 0, 0, 0, 0, time.UTC)),
		Tags: map[string]string{
			TagPulumiModified: pulumiModified,
			// Note: NO pulumi:created tag
		},
	}

	_, err := resolveTimestamps(req)
	if err == nil {
		t.Error("resolveTimestamps() should error when only pulumi:modified is provided (not pulumi:created)")
	}
	if !strings.Contains(err.Error(), "start time required") {
		t.Errorf("Error should mention start time, got: %v", err)
	}
}

// TestGetActualCost_WithPulumiCreated tests Feature 016 end-to-end.
// This validates that GetActualCost correctly uses pulumi:created timestamp
// when explicit timestamps are not provided.
func TestGetActualCost_WithPulumiCreated(t *testing.T) {
	plugin := newTestPluginForActual()
	ctx := context.Background()

	// Create resource with pulumi:created tag but no explicit timestamps
	created := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2025, 1, 8, 0, 0, 0, 0, time.UTC) // 7 days later = 168 hours

	req := &pbc.GetActualCostRequest{
		ResourceId: makeResourceJSON("aws", "ec2", "t3.micro", "us-east-1", map[string]string{
			TagPulumiCreated: created.Format(time.RFC3339),
		}),
		Start: nil, // Let it resolve from pulumi:created
		End:   timestamppb.New(end),
	}

	resp, err := plugin.GetActualCost(ctx, req)
	if err != nil {
		t.Fatalf("GetActualCost() unexpected error: %v", err)
	}
	if resp == nil || len(resp.Results) == 0 {
		t.Fatal("GetActualCost() returned empty response")
	}

	result := resp.Results[0]

	// Should use pulumi:created for start, so runtime = 168 hours
	// $0.0104/hr * 730 hrs = $7.592/month
	// $7.592 * (168/730) = $1.7472
	expectedCost := 1.7472
	tolerance := expectedCost * 0.0001
	if diff := result.Cost - expectedCost; diff > tolerance || diff < -tolerance {
		t.Errorf("GetActualCost() cost = %v, want %v", result.Cost, expectedCost)
	}

	// Verify confidence is encoded in source (Feature 016)
	if !strings.Contains(result.Source, "[confidence:") {
		t.Errorf("Source should contain confidence encoding, got: %s", result.Source)
	}
}

// TestGetActualCost_ImportedResourceConfidence tests Feature 016 US2.
// This validates that imported resources show MEDIUM confidence.
func TestGetActualCost_ImportedResourceConfidence(t *testing.T) {
	plugin := newTestPluginForActual()
	ctx := context.Background()

	created := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2025, 1, 8, 0, 0, 0, 0, time.UTC)

	req := &pbc.GetActualCostRequest{
		ResourceId: makeResourceJSON("aws", "ec2", "t3.micro", "us-east-1", map[string]string{
			TagPulumiCreated:  created.Format(time.RFC3339),
			TagPulumiExternal: "true", // Imported resource
		}),
		Start: nil, // Let it resolve from pulumi:created
		End:   timestamppb.New(end),
	}

	resp, err := plugin.GetActualCost(ctx, req)
	if err != nil {
		t.Fatalf("GetActualCost() unexpected error: %v", err)
	}
	if resp == nil || len(resp.Results) == 0 {
		t.Fatal("GetActualCost() returned empty response")
	}

	result := resp.Results[0]

	// Imported resource with pulumi:created should show MEDIUM confidence
	if !strings.Contains(result.Source, "[confidence:MEDIUM]") {
		t.Errorf("Imported resource should have MEDIUM confidence, got: %s", result.Source)
	}

	// Should also include "imported resource" note
	if !strings.Contains(result.Source, "imported resource") {
		t.Errorf("Source should mention 'imported resource', got: %s", result.Source)
	}
}

// TestGetActualCost_ExplicitOverridesPulumiCreated tests Feature 016 US3.
// This validates that explicit timestamps take precedence over pulumi:created.
func TestGetActualCost_ExplicitOverridesPulumiCreated(t *testing.T) {
	plugin := newTestPluginForActual()
	ctx := context.Background()

	// pulumi:created says resource is 30 days old
	created := time.Date(2024, 12, 1, 0, 0, 0, 0, time.UTC)
	// But explicit timestamps say only query 7 days
	explicitStart := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	explicitEnd := time.Date(2025, 1, 8, 0, 0, 0, 0, time.UTC) // 7 days = 168 hours

	req := &pbc.GetActualCostRequest{
		ResourceId: makeResourceJSON("aws", "ec2", "t3.micro", "us-east-1", map[string]string{
			TagPulumiCreated: created.Format(time.RFC3339),
		}),
		Start: timestamppb.New(explicitStart), // Explicit overrides pulumi:created
		End:   timestamppb.New(explicitEnd),
	}

	resp, err := plugin.GetActualCost(ctx, req)
	if err != nil {
		t.Fatalf("GetActualCost() unexpected error: %v", err)
	}
	if resp == nil || len(resp.Results) == 0 {
		t.Fatal("GetActualCost() returned empty response")
	}

	result := resp.Results[0]

	// Should use explicit timestamps (168 hours), NOT pulumi:created (would be ~730 hours)
	// $0.0104/hr * 730 hrs = $7.592/month
	// $7.592 * (168/730) = $1.7472
	expectedCost := 1.7472
	tolerance := expectedCost * 0.0001
	if diff := result.Cost - expectedCost; diff > tolerance || diff < -tolerance {
		t.Errorf("GetActualCost() cost = %v, want %v (explicit should override pulumi:created)", result.Cost, expectedCost)
	}

	// Explicit timestamps get HIGH confidence
	if !strings.Contains(result.Source, "[confidence:HIGH]") {
		t.Errorf("Explicit timestamps should have HIGH confidence, got: %s", result.Source)
	}
}

// TestFormatSourceWithConfidence tests the formatSourceWithConfidence helper function.
// This validates the semantic encoding format in the source field.
func TestFormatSourceWithConfidence(t *testing.T) {
	tests := []struct {
		name       string
		confidence ConfidenceLevel
		note       string
		want       string
	}{
		{
			name:       "HIGH confidence no note",
			confidence: ConfidenceHigh,
			note:       "",
			want:       "aws-public-fallback[confidence:HIGH]",
		},
		{
			name:       "MEDIUM confidence no note",
			confidence: ConfidenceMedium,
			note:       "",
			want:       "aws-public-fallback[confidence:MEDIUM]",
		},
		{
			name:       "LOW confidence no note",
			confidence: ConfidenceLow,
			note:       "",
			want:       "aws-public-fallback[confidence:LOW]",
		},
		{
			name:       "HIGH confidence with note",
			confidence: ConfidenceHigh,
			note:       "explicit timestamps",
			want:       "aws-public-fallback[confidence:HIGH] explicit timestamps",
		},
		{
			name:       "MEDIUM confidence with imported note",
			confidence: ConfidenceMedium,
			note:       "imported resource",
			want:       "aws-public-fallback[confidence:MEDIUM] imported resource",
		},
		{
			name:       "LOW confidence with explanation",
			confidence: ConfidenceLow,
			note:       "unsupported resource type",
			want:       "aws-public-fallback[confidence:LOW] unsupported resource type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatSourceWithConfidence(tt.confidence, tt.note)
			if got != tt.want {
				t.Errorf("formatSourceWithConfidence() = %q, want %q", got, tt.want)
			}
		})
	}
}
