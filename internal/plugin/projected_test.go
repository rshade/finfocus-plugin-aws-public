package plugin

import (
	"context"
	"testing"

	pbc "github.com/rshade/pulumicost-spec/sdk/go/proto/pulumicost/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TestGetProjectedCost_EC2 tests EC2 cost estimation (T040)
func TestGetProjectedCost_EC2(t *testing.T) {
	mock := newMockPricingClient("us-east-1", "USD")
	mock.ec2Prices["t3.micro/Linux/Shared"] = 0.0104
	plugin := NewAWSPublicPlugin("us-east-1", mock)

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
	mock.ebsPrices["gp3"] = 0.08
	plugin := NewAWSPublicPlugin("us-east-1", mock)

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
	mock.ebsPrices["gp2"] = 0.10
	plugin := NewAWSPublicPlugin("us-east-1", mock)

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
	plugin := NewAWSPublicPlugin("us-east-1", mock)

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
	plugin := NewAWSPublicPlugin("us-east-1", mock)

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
	// Don't add any pricing data
	plugin := NewAWSPublicPlugin("us-east-1", mock)

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
	plugin := NewAWSPublicPlugin("us-east-1", mock)

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
