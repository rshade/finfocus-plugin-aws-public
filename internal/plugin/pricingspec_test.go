package plugin

import (
	"context"
	"testing"

	"github.com/rs/zerolog"
	pbc "github.com/rshade/pulumicost-spec/sdk/go/proto/pulumicost/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TestGetPricingSpec_EC2 verifies EC2 pricing specification retrieval.
//
// This test validates that GetPricingSpec returns correct pricing details
// for EC2 instances, including hourly rate, billing mode, and assumptions.
func TestGetPricingSpec_EC2(t *testing.T) {
	mock := newMockPricingClient("us-east-1", "USD")
	mock.ec2Prices["t3.micro/Linux/Shared"] = 0.0104
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	plugin := NewAWSPublicPlugin("us-east-1", mock, logger)

	resp, err := plugin.GetPricingSpec(context.Background(), &pbc.GetPricingSpecRequest{
		Resource: &pbc.ResourceDescriptor{
			Provider:     "aws",
			ResourceType: "ec2",
			Sku:          "t3.micro",
			Region:       "us-east-1",
		},
	})

	require.NoError(t, err)
	require.NotNil(t, resp.Spec)
	assert.Equal(t, "aws", resp.Spec.Provider)
	assert.Equal(t, "ec2", resp.Spec.ResourceType)
	assert.Equal(t, "t3.micro", resp.Spec.Sku)
	assert.Equal(t, "us-east-1", resp.Spec.Region)
	assert.Equal(t, "per_hour", resp.Spec.BillingMode)
	assert.Equal(t, 0.0104, resp.Spec.RatePerUnit)
	assert.Equal(t, "USD", resp.Spec.Currency)
	assert.Equal(t, "hour", resp.Spec.Unit)
	assert.Equal(t, "aws-public", resp.Spec.Source)
	assert.Contains(t, resp.Spec.Description, "Linux")
	assert.Contains(t, resp.Spec.Description, "Shared")
	assert.NotEmpty(t, resp.Spec.Assumptions)
}

// TestGetPricingSpec_EC2_PulumiFormat tests EC2 pricing spec with Pulumi resource type format.
func TestGetPricingSpec_EC2_PulumiFormat(t *testing.T) {
	mock := newMockPricingClient("us-east-1", "USD")
	mock.ec2Prices["t3.micro/Linux/Shared"] = 0.0104
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	plugin := NewAWSPublicPlugin("us-east-1", mock, logger)

	resp, err := plugin.GetPricingSpec(context.Background(), &pbc.GetPricingSpecRequest{
		Resource: &pbc.ResourceDescriptor{
			Provider:     "aws",
			ResourceType: "aws:ec2/instance:Instance",
			Sku:          "t3.micro",
			Region:       "us-east-1",
		},
	})

	require.NoError(t, err)
	require.NotNil(t, resp.Spec)
	assert.Equal(t, "aws", resp.Spec.Provider)
	assert.Equal(t, "aws:ec2/instance:Instance", resp.Spec.ResourceType) // Should preserve original format
	assert.Equal(t, "t3.micro", resp.Spec.Sku)
	assert.Equal(t, "us-east-1", resp.Spec.Region)
	assert.Equal(t, "per_hour", resp.Spec.BillingMode)
	assert.Equal(t, 0.0104, resp.Spec.RatePerUnit)
	assert.Equal(t, "USD", resp.Spec.Currency)
	assert.Equal(t, "hour", resp.Spec.Unit)
	assert.Equal(t, "aws-public", resp.Spec.Source)
	assert.Contains(t, resp.Spec.Description, "Linux")
	assert.Contains(t, resp.Spec.Description, "Shared")
	assert.NotEmpty(t, resp.Spec.Assumptions)
}

// TestGetPricingSpec_EC2_NotFound verifies handling of unknown instance types.
func TestGetPricingSpec_EC2_NotFound(t *testing.T) {
	mock := newMockPricingClient("us-east-1", "USD")
	// Don't add any prices
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	plugin := NewAWSPublicPlugin("us-east-1", mock, logger)

	resp, err := plugin.GetPricingSpec(context.Background(), &pbc.GetPricingSpecRequest{
		Resource: &pbc.ResourceDescriptor{
			Provider:     "aws",
			ResourceType: "ec2",
			Sku:          "unknown.type",
			Region:       "us-east-1",
		},
	})

	require.NoError(t, err)
	require.NotNil(t, resp.Spec)
	assert.Equal(t, "per_hour", resp.Spec.BillingMode)
	assert.Equal(t, float64(0), resp.Spec.RatePerUnit)
	assert.Contains(t, resp.Spec.Description, "not found")
}

// TestGetPricingSpec_EBS verifies EBS pricing specification retrieval.
func TestGetPricingSpec_EBS(t *testing.T) {
	mock := newMockPricingClient("us-east-1", "USD")
	mock.ebsPrices["gp3"] = 0.08
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	plugin := NewAWSPublicPlugin("us-east-1", mock, logger)

	resp, err := plugin.GetPricingSpec(context.Background(), &pbc.GetPricingSpecRequest{
		Resource: &pbc.ResourceDescriptor{
			Provider:     "aws",
			ResourceType: "ebs",
			Sku:          "gp3",
			Region:       "us-east-1",
		},
	})

	require.NoError(t, err)
	require.NotNil(t, resp.Spec)
	assert.Equal(t, "aws", resp.Spec.Provider)
	assert.Equal(t, "ebs", resp.Spec.ResourceType)
	assert.Equal(t, "gp3", resp.Spec.Sku)
	assert.Equal(t, "us-east-1", resp.Spec.Region)
	assert.Equal(t, "per_gb_month", resp.Spec.BillingMode)
	assert.Equal(t, 0.08, resp.Spec.RatePerUnit)
	assert.Equal(t, "USD", resp.Spec.Currency)
	assert.Equal(t, "GB-month", resp.Spec.Unit)
	assert.Equal(t, "aws-public", resp.Spec.Source)
	assert.Contains(t, resp.Spec.Description, "gp3")
	assert.NotEmpty(t, resp.Spec.Assumptions)
}

// TestGetPricingSpec_EBS_PulumiFormat tests EBS pricing spec with Pulumi resource type format.
func TestGetPricingSpec_EBS_PulumiFormat(t *testing.T) {
	mock := newMockPricingClient("us-east-1", "USD")
	mock.ebsPrices["gp3"] = 0.08
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	plugin := NewAWSPublicPlugin("us-east-1", mock, logger)

	resp, err := plugin.GetPricingSpec(context.Background(), &pbc.GetPricingSpecRequest{
		Resource: &pbc.ResourceDescriptor{
			Provider:     "aws",
			ResourceType: "aws:ebs/volume:Volume",
			Sku:          "gp3",
			Region:       "us-east-1",
		},
	})

	require.NoError(t, err)
	require.NotNil(t, resp.Spec)
	assert.Equal(t, "aws", resp.Spec.Provider)
	assert.Equal(t, "aws:ebs/volume:Volume", resp.Spec.ResourceType) // Should preserve original format
	assert.Equal(t, "gp3", resp.Spec.Sku)
	assert.Equal(t, "us-east-1", resp.Spec.Region)
	assert.Equal(t, "per_gb_month", resp.Spec.BillingMode)
	assert.Equal(t, 0.08, resp.Spec.RatePerUnit)
	assert.Equal(t, "USD", resp.Spec.Currency)
	assert.Equal(t, "GB-month", resp.Spec.Unit)
	assert.Equal(t, "aws-public", resp.Spec.Source)
	assert.Contains(t, resp.Spec.Description, "gp3")
	assert.NotEmpty(t, resp.Spec.Assumptions)
}

// TestGetPricingSpec_EBS_NotFound verifies handling of unknown volume types.
func TestGetPricingSpec_EBS_NotFound(t *testing.T) {
	mock := newMockPricingClient("us-east-1", "USD")
	// Don't add any prices
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	plugin := NewAWSPublicPlugin("us-east-1", mock, logger)

	resp, err := plugin.GetPricingSpec(context.Background(), &pbc.GetPricingSpecRequest{
		Resource: &pbc.ResourceDescriptor{
			Provider:     "aws",
			ResourceType: "ebs",
			Sku:          "unknown-type",
			Region:       "us-east-1",
		},
	})

	require.NoError(t, err)
	require.NotNil(t, resp.Spec)
	assert.Equal(t, "per_gb_month", resp.Spec.BillingMode)
	assert.Equal(t, float64(0), resp.Spec.RatePerUnit)
	assert.Contains(t, resp.Spec.Description, "not found")
}

// TestGetPricingSpec_StubServices verifies placeholder specs for stub services.
func TestGetPricingSpec_StubServices(t *testing.T) {
	tests := []struct {
		name         string
		resourceType string
	}{
		{"S3", "s3"},
		{"Lambda", "lambda"},
		{"RDS", "rds"},
		{"DynamoDB", "dynamodb"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newMockPricingClient("us-east-1", "USD")
			logger := zerolog.New(nil).Level(zerolog.InfoLevel)
			plugin := NewAWSPublicPlugin("us-east-1", mock, logger)

			resp, err := plugin.GetPricingSpec(context.Background(), &pbc.GetPricingSpecRequest{
				Resource: &pbc.ResourceDescriptor{
					Provider:     "aws",
					ResourceType: tt.resourceType,
					Sku:          "test-sku",
					Region:       "us-east-1",
				},
			})

			require.NoError(t, err)
			require.NotNil(t, resp.Spec)
			assert.Equal(t, "unknown", resp.Spec.BillingMode)
			assert.Equal(t, float64(0), resp.Spec.RatePerUnit)
			assert.Contains(t, resp.Spec.Description, "not fully implemented")
			assert.Equal(t, "aws-public", resp.Spec.Source)
		})
	}
}

// TestGetPricingSpec_UnknownResourceType verifies handling of unsupported resource types.
func TestGetPricingSpec_UnknownResourceType(t *testing.T) {
	mock := newMockPricingClient("us-east-1", "USD")
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	plugin := NewAWSPublicPlugin("us-east-1", mock, logger)

	resp, err := plugin.GetPricingSpec(context.Background(), &pbc.GetPricingSpecRequest{
		Resource: &pbc.ResourceDescriptor{
			Provider:     "aws",
			ResourceType: "cloudfront",
			Sku:          "test-sku",
			Region:       "us-east-1",
		},
	})

	require.NoError(t, err)
	require.NotNil(t, resp.Spec)
	assert.Equal(t, "unknown", resp.Spec.BillingMode)
	assert.Equal(t, float64(0), resp.Spec.RatePerUnit)
	assert.Contains(t, resp.Spec.Description, "not supported")
}

// TestGetPricingSpec_RegionMismatch verifies error handling for region mismatches.
func TestGetPricingSpec_RegionMismatch(t *testing.T) {
	mock := newMockPricingClient("us-east-1", "USD")
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	plugin := NewAWSPublicPlugin("us-east-1", mock, logger)

	_, err := plugin.GetPricingSpec(context.Background(), &pbc.GetPricingSpecRequest{
		Resource: &pbc.ResourceDescriptor{
			Provider:     "aws",
			ResourceType: "ec2",
			Sku:          "t3.micro",
			Region:       "eu-west-1", // different from plugin region
		},
	})

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.FailedPrecondition, st.Code())
}

// TestGetPricingSpec_NilRequest verifies error handling for nil request.
func TestGetPricingSpec_NilRequest(t *testing.T) {
	mock := newMockPricingClient("us-east-1", "USD")
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	plugin := NewAWSPublicPlugin("us-east-1", mock, logger)

	_, err := plugin.GetPricingSpec(context.Background(), nil)

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

// TestGetPricingSpec_NilResource verifies error handling for nil resource.
func TestGetPricingSpec_NilResource(t *testing.T) {
	mock := newMockPricingClient("us-east-1", "USD")
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	plugin := NewAWSPublicPlugin("us-east-1", mock, logger)

	_, err := plugin.GetPricingSpec(context.Background(), &pbc.GetPricingSpecRequest{
		Resource: nil,
	})

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

// TestGetPricingSpec_MissingFields verifies error handling for missing required fields.
func TestGetPricingSpec_MissingFields(t *testing.T) {
	tests := []struct {
		name     string
		resource *pbc.ResourceDescriptor
	}{
		{
			name: "missing provider",
			resource: &pbc.ResourceDescriptor{
				Provider:     "",
				ResourceType: "ec2",
				Sku:          "t3.micro",
				Region:       "us-east-1",
			},
		},
		{
			name: "missing resource_type",
			resource: &pbc.ResourceDescriptor{
				Provider:     "aws",
				ResourceType: "",
				Sku:          "t3.micro",
				Region:       "us-east-1",
			},
		},
		{
			name: "missing sku",
			resource: &pbc.ResourceDescriptor{
				Provider:     "aws",
				ResourceType: "ec2",
				Sku:          "",
				Region:       "us-east-1",
			},
		},
		{
			name: "missing region",
			resource: &pbc.ResourceDescriptor{
				Provider:     "aws",
				ResourceType: "ec2",
				Sku:          "t3.micro",
				Region:       "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newMockPricingClient("us-east-1", "USD")
			logger := zerolog.New(nil).Level(zerolog.InfoLevel)
			plugin := NewAWSPublicPlugin("us-east-1", mock, logger)

			_, err := plugin.GetPricingSpec(context.Background(), &pbc.GetPricingSpecRequest{
				Resource: tt.resource,
			})

			require.Error(t, err)
			st, ok := status.FromError(err)
			require.True(t, ok)
			assert.Equal(t, codes.InvalidArgument, st.Code())
		})
	}
}

// TestGetPricingSpec_AllVolumeTypes verifies EBS pricing for different volume types.
func TestGetPricingSpec_AllVolumeTypes(t *testing.T) {
	volumeTypes := []struct {
		name       string
		volumeType string
		pricePerGB float64
	}{
		{"gp3", "gp3", 0.08},
		{"gp2", "gp2", 0.10},
		{"io1", "io1", 0.125},
		{"io2", "io2", 0.125},
		{"st1", "st1", 0.045},
		{"sc1", "sc1", 0.025},
	}

	for _, tt := range volumeTypes {
		t.Run(tt.name, func(t *testing.T) {
			mock := newMockPricingClient("us-east-1", "USD")
			mock.ebsPrices[tt.volumeType] = tt.pricePerGB
			logger := zerolog.New(nil).Level(zerolog.InfoLevel)
			plugin := NewAWSPublicPlugin("us-east-1", mock, logger)

			resp, err := plugin.GetPricingSpec(context.Background(), &pbc.GetPricingSpecRequest{
				Resource: &pbc.ResourceDescriptor{
					Provider:     "aws",
					ResourceType: "ebs",
					Sku:          tt.volumeType,
					Region:       "us-east-1",
				},
			})

			require.NoError(t, err)
			require.NotNil(t, resp.Spec)
			assert.Equal(t, tt.pricePerGB, resp.Spec.RatePerUnit)
			assert.Equal(t, "per_gb_month", resp.Spec.BillingMode)
		})
	}
}
