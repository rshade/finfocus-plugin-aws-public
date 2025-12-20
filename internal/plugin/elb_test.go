package plugin

import (
	"context"
	"testing"

	"github.com/rs/zerolog"
	pbc "github.com/rshade/pulumicost-spec/sdk/go/proto/pulumicost/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createELBMockPlugin creates a test plugin with ELB pricing configured.
func createELBMockPlugin(region string) *AWSPublicPlugin {
	mock := newMockPricingClient(region, "USD")
	mock.albHourlyPrice = 0.0225
	mock.albLCUPrice = 0.008
	mock.nlbHourlyPrice = 0.0225
	mock.nlbNLCUPrice = 0.006
	return NewAWSPublicPlugin(region, mock, zerolog.Nop())
}

// TestEstimateELB_ALB verifies ALB cost estimation with fixed hourly rate and LCU charges.
//
// This test validates:
// - ALB detection from resource SKU
// - LCU extraction from tags
// - Correct hourly and capacity-based cost calculation
// - Proper billing detail formatting
func TestEstimateELB_ALB(t *testing.T) {
	tests := []struct {
		name              string
		sku               string
		lcuPerHour        string
		expectedCostRange [2]float64 // [min, max] to account for regional variation
		wantErr           bool
	}{
		{
			name:              "ALB with explicit SKU and LCU tags",
			sku:               "alb",
			lcuPerHour:        "1.5",
			expectedCostRange: [2]float64{24.0, 26.0}, // 730 * $0.0225 + 730 * 1.5 * $0.008 = 16.425 + 8.76 = 25.185
			wantErr:           false,
		},
		{
			name:              "ALB with application SKU variant",
			sku:               "application",
			lcuPerHour:        "2.0",
			expectedCostRange: [2]float64{27.0, 29.5}, // 730 * $0.0225 + 730 * 2.0 * $0.008 = 16.425 + 11.68 = 28.105
			wantErr:           false,
		},
		{
			name:              "ALB with zero LCU",
			sku:               "alb",
			lcuPerHour:        "0",
			expectedCostRange: [2]float64{16.0, 17.0}, // 730 * $0.0225 only
			wantErr:           false,
		},
		{
			name:              "ALB defaults to ALB when SKU not provided",
			sku:               "",
			lcuPerHour:        "1.0",
			expectedCostRange: [2]float64{21.0, 25.0},
			wantErr:           false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plugin := createELBMockPlugin("unknown")

			resource := &pbc.ResourceDescriptor{
				Provider:     "aws",
				ResourceType: "elb",
				Sku:          tt.sku,
				Region:       "unknown",
				Tags: map[string]string{
					"lcu_per_hour": tt.lcuPerHour,
				},
			}

			resp, err := plugin.estimateELB("test-trace", resource)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, resp)
			assert.Equal(t, "USD", resp.Currency)

			// Verify cost is in expected range
			assert.GreaterOrEqual(t, resp.CostPerMonth, tt.expectedCostRange[0], "Cost should be at least %.2f", tt.expectedCostRange[0])
			assert.LessOrEqual(t, resp.CostPerMonth, tt.expectedCostRange[1], "Cost should be at most %.2f", tt.expectedCostRange[1])

			// Verify billing detail includes LCU metric
			assert.Contains(t, resp.BillingDetail, "ALB")
			assert.Contains(t, resp.BillingDetail, "LCU")
			assert.Contains(t, resp.BillingDetail, "730 hrs/month")
		})
	}
}

// TestEstimateELB_NLB verifies NLB cost estimation with fixed hourly rate and NLCU charges.
//
// This test validates:
// - NLB detection from resource SKU
// - NLCU extraction from tags
// - Correct hourly and capacity-based cost calculation
// - Proper billing detail formatting with NLCU metric
func TestEstimateELB_NLB(t *testing.T) {
	tests := []struct {
		name              string
		sku               string
		nlcuPerHour       string
		expectedCostRange [2]float64 // [min, max] for regional variation
		wantErr           bool
	}{
		{
			name:              "NLB with explicit SKU and NLCU tags",
			sku:               "nlb",
			nlcuPerHour:       "1.0",
			expectedCostRange: [2]float64{20.0, 23.0}, // 730 * $0.0225 + 730 * 1.0 * $0.006
			wantErr:           false,
		},
		{
			name:              "NLB with network SKU variant",
			sku:               "network",
			nlcuPerHour:       "2.5",
			expectedCostRange: [2]float64{27.0, 30.0}, // 730 * $0.0225 + 730 * 2.5 * $0.006
			wantErr:           false,
		},
		{
			name:              "NLB with zero NLCU",
			sku:               "nlb",
			nlcuPerHour:       "0",
			expectedCostRange: [2]float64{16.0, 17.0}, // 730 * $0.0225 only
			wantErr:           false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plugin := createELBMockPlugin("unknown")

			resource := &pbc.ResourceDescriptor{
				Provider:     "aws",
				ResourceType: "elb",
				Sku:          tt.sku,
				Region:       "unknown",
				Tags: map[string]string{
					"nlcu_per_hour": tt.nlcuPerHour,
				},
			}

			resp, err := plugin.estimateELB("test-trace", resource)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, resp)
			assert.Equal(t, "USD", resp.Currency)

			// Verify cost is in expected range
			assert.GreaterOrEqual(t, resp.CostPerMonth, tt.expectedCostRange[0], "Cost should be at least %.2f", tt.expectedCostRange[0])
			assert.LessOrEqual(t, resp.CostPerMonth, tt.expectedCostRange[1], "Cost should be at most %.2f", tt.expectedCostRange[1])

			// Verify billing detail includes NLCU metric
			assert.Contains(t, resp.BillingDetail, "NLB")
			assert.Contains(t, resp.BillingDetail, "NLCU")
			assert.Contains(t, resp.BillingDetail, "730 hrs/month")
		})
	}
}

// TestEstimateELB_FallbackCapacityUnits verifies generic capacity_units tag fallback.
//
// This test validates:
// - Generic capacity_units tag is used when type-specific tag not found
// - Fallback works for both ALB and NLB
func TestEstimateELB_FallbackCapacityUnits(t *testing.T) {
	tests := []struct {
		name        string
		sku         string
		capacityTag string
		wantMetric  string
	}{
		{
			name:        "ALB with generic capacity_units",
			sku:         "alb",
			capacityTag: "1.0",
			wantMetric:  "LCU",
		},
		{
			name:        "NLB with generic capacity_units",
			sku:         "nlb",
			capacityTag: "2.0",
			wantMetric:  "NLCU",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plugin := createELBMockPlugin("unknown")

			resource := &pbc.ResourceDescriptor{
				Provider:     "aws",
				ResourceType: "elb",
				Sku:          tt.sku,
				Region:       "unknown",
				Tags: map[string]string{
					"capacity_units": tt.capacityTag,
				},
			}

			resp, err := plugin.estimateELB("test-trace", resource)

			require.NoError(t, err)
			assert.NotNil(t, resp)
			assert.Contains(t, resp.BillingDetail, tt.wantMetric)
		})
	}
}

// TestEstimateELB_InvalidTag verifies that non-numeric tags default to 0.
//
// This test validates:
// - Non-numeric capacity unit values are gracefully handled
// - Cost calculation defaults to fixed rate only
// - No error is returned for invalid tag values
func TestEstimateELB_InvalidTag(t *testing.T) {
	plugin := createELBMockPlugin("unknown")

	resource := &pbc.ResourceDescriptor{
		Provider:     "aws",
		ResourceType: "elb",
		Sku:          "alb",
		Region:       "unknown",
		Tags: map[string]string{
			"lcu_per_hour": "not-a-number",
		},
	}

	resp, err := plugin.estimateELB("test-trace", resource)

	require.NoError(t, err)
	assert.NotNil(t, resp)
	// Should default to fixed rate only (0 capacity units)
	assert.Contains(t, resp.BillingDetail, "0.0 LCU")
}

// TestEstimateELB_RoundTripEstimate validates end-to-end ALB and NLB estimation.
//
// This is an integration test that calls GetProjectedCost through the RPC interface
// to ensure proper wiring from the public API through to estimateELB.
func TestEstimateELB_RoundTripEstimate(t *testing.T) {
	tests := []struct {
		name         string
		resourceType string
		sku          string
		lbType       string
	}{
		{
			name:         "Round-trip ALB estimation",
			resourceType: "elb",
			sku:          "alb",
			lbType:       "ALB",
		},
		{
			name:         "Round-trip NLB estimation",
			resourceType: "elb",
			sku:          "nlb",
			lbType:       "NLB",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plugin := createELBMockPlugin("unknown")

			req := &pbc.GetProjectedCostRequest{
				Resource: &pbc.ResourceDescriptor{
					Provider:     "aws",
					ResourceType: tt.resourceType,
					Sku:          tt.sku,
					Region:       "unknown",
					Tags: map[string]string{
						"capacity_units": "1.5",
					},
				},
			}

			resp, err := plugin.GetProjectedCost(context.Background(), req)

			require.NoError(t, err)
			assert.NotNil(t, resp)
			assert.Greater(t, resp.CostPerMonth, 0.0)
			assert.Contains(t, resp.BillingDetail, tt.lbType)
		})
	}
}
