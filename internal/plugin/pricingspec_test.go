package plugin

import (
	"context"
	"testing"

	"github.com/rs/zerolog"
	"github.com/rshade/pulumicost-plugin-aws-public/internal/pricing"
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
	plugin := NewAWSPublicPlugin("us-east-1", "test-version", mock, logger)

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
	plugin := NewAWSPublicPlugin("us-east-1", "test-version", mock, logger)

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
	plugin := NewAWSPublicPlugin("us-east-1", "test-version", mock, logger)

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
	plugin := NewAWSPublicPlugin("us-east-1", "test-version", mock, logger)

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
	plugin := NewAWSPublicPlugin("us-east-1", "test-version", mock, logger)

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
	plugin := NewAWSPublicPlugin("us-east-1", "test-version", mock, logger)

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

// TestGetPricingSpec_S3 verifies S3 pricing specification retrieval.
func TestGetPricingSpec_S3(t *testing.T) {
	mock := newMockPricingClient("us-east-1", "USD")
	mock.s3Prices["STANDARD"] = 0.023
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	plugin := NewAWSPublicPlugin("us-east-1", "test-version", mock, logger)

	resp, err := plugin.GetPricingSpec(context.Background(), &pbc.GetPricingSpecRequest{
		Resource: &pbc.ResourceDescriptor{
			Provider:     "aws",
			ResourceType: "s3",
			Sku:          "STANDARD",
			Region:       "us-east-1",
		},
	})

	require.NoError(t, err)
	require.NotNil(t, resp.Spec)
	assert.Equal(t, "per_gb_month", resp.Spec.BillingMode)
	assert.Equal(t, 0.023, resp.Spec.RatePerUnit)
	assert.Equal(t, "GB-month", resp.Spec.Unit)
	assert.Contains(t, resp.Spec.Description, "STANDARD")
	assert.Equal(t, "aws-public", resp.Spec.Source)
}

// TestGetPricingSpec_Lambda verifies Lambda pricing specification retrieval.
func TestGetPricingSpec_Lambda(t *testing.T) {
	mock := newMockPricingClient("us-east-1", "USD")
	mock.lambdaPrices["request"] = 0.0000002
	mock.lambdaPrices["gb-second"] = 0.0000166667 // Mock uses "gb-second" key
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	plugin := NewAWSPublicPlugin("us-east-1", "test-version", mock, logger)

	resp, err := plugin.GetPricingSpec(context.Background(), &pbc.GetPricingSpecRequest{
		Resource: &pbc.ResourceDescriptor{
			Provider:     "aws",
			ResourceType: "lambda",
			Sku:          "x86_64",
			Region:       "us-east-1",
		},
	})

	require.NoError(t, err)
	require.NotNil(t, resp.Spec)
	assert.Equal(t, "per_request_and_gb_second", resp.Spec.BillingMode)
	assert.Equal(t, 0.0000166667, resp.Spec.RatePerUnit)
	assert.Equal(t, "GB-second", resp.Spec.Unit)
	assert.Contains(t, resp.Spec.Description, "x86_64")
	assert.Equal(t, "aws-public", resp.Spec.Source)
}

// TestGetPricingSpec_RDS verifies RDS pricing specification retrieval.
func TestGetPricingSpec_RDS(t *testing.T) {
	mock := newMockPricingClient("us-east-1", "USD")
	mock.rdsInstancePrices["db.t3.medium/mysql"] = 0.068
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	plugin := NewAWSPublicPlugin("us-east-1", "test-version", mock, logger)

	resp, err := plugin.GetPricingSpec(context.Background(), &pbc.GetPricingSpecRequest{
		Resource: &pbc.ResourceDescriptor{
			Provider:     "aws",
			ResourceType: "rds",
			Sku:          "db.t3.medium",
			Region:       "us-east-1",
			Tags:         map[string]string{"engine": "mysql"},
		},
	})

	require.NoError(t, err)
	require.NotNil(t, resp.Spec)
	assert.Equal(t, "per_hour", resp.Spec.BillingMode)
	assert.Equal(t, 0.068, resp.Spec.RatePerUnit)
	assert.Equal(t, "hour", resp.Spec.Unit)
	assert.Contains(t, resp.Spec.Description, "mysql")
	assert.Equal(t, "aws-public", resp.Spec.Source)
}

// TestGetPricingSpec_DynamoDB_OnDemand verifies DynamoDB on-demand pricing spec.
func TestGetPricingSpec_DynamoDB_OnDemand(t *testing.T) {
	mock := newMockPricingClient("us-east-1", "USD")
	mock.dynamoDBPrices["on-demand-read"] = 0.00000025
	mock.dynamoDBPrices["on-demand-write"] = 0.00000125
	mock.dynamoDBPrices["storage"] = 0.25
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	plugin := NewAWSPublicPlugin("us-east-1", "test-version", mock, logger)

	resp, err := plugin.GetPricingSpec(context.Background(), &pbc.GetPricingSpecRequest{
		Resource: &pbc.ResourceDescriptor{
			Provider:     "aws",
			ResourceType: "dynamodb",
			Sku:          "on-demand",
			Region:       "us-east-1",
		},
	})

	require.NoError(t, err)
	require.NotNil(t, resp.Spec)
	assert.Equal(t, "on_demand", resp.Spec.BillingMode)
	assert.Equal(t, 0.25, resp.Spec.RatePerUnit) // Storage rate
	assert.Equal(t, "GB-month", resp.Spec.Unit)
	assert.Equal(t, "aws-public", resp.Spec.Source)
}

// TestGetPricingSpec_DynamoDB_Provisioned verifies DynamoDB provisioned pricing spec.
func TestGetPricingSpec_DynamoDB_Provisioned(t *testing.T) {
	mock := newMockPricingClient("us-east-1", "USD")
	mock.dynamoDBPrices["provisioned-rcu"] = 0.00013
	mock.dynamoDBPrices["provisioned-wcu"] = 0.00065
	mock.dynamoDBPrices["storage"] = 0.25
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	plugin := NewAWSPublicPlugin("us-east-1", "test-version", mock, logger)

	resp, err := plugin.GetPricingSpec(context.Background(), &pbc.GetPricingSpecRequest{
		Resource: &pbc.ResourceDescriptor{
			Provider:     "aws",
			ResourceType: "dynamodb",
			Sku:          "provisioned",
			Region:       "us-east-1",
		},
	})

	require.NoError(t, err)
	require.NotNil(t, resp.Spec)
	assert.Equal(t, "provisioned_capacity", resp.Spec.BillingMode)
	assert.Equal(t, 0.00013, resp.Spec.RatePerUnit) // RCU rate
	assert.Equal(t, "RCU-hour", resp.Spec.Unit)
	assert.Equal(t, "aws-public", resp.Spec.Source)
}

// TestGetPricingSpec_EKS verifies EKS pricing specification retrieval.
func TestGetPricingSpec_EKS(t *testing.T) {
	mock := newMockPricingClient("us-east-1", "USD")
	mock.eksStandardPrice = 0.10
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	plugin := NewAWSPublicPlugin("us-east-1", "test-version", mock, logger)

	resp, err := plugin.GetPricingSpec(context.Background(), &pbc.GetPricingSpecRequest{
		Resource: &pbc.ResourceDescriptor{
			Provider:     "aws",
			ResourceType: "eks",
			Sku:          "cluster", // SKU is required by validation
			Region:       "us-east-1",
		},
	})

	require.NoError(t, err)
	require.NotNil(t, resp.Spec)
	assert.Equal(t, "per_hour", resp.Spec.BillingMode)
	assert.Equal(t, 0.10, resp.Spec.RatePerUnit)
	assert.Equal(t, "hour", resp.Spec.Unit)
	assert.Contains(t, resp.Spec.Description, "standard")
	assert.Equal(t, "aws-public", resp.Spec.Source)
}

// TestGetPricingSpec_ALB verifies ALB pricing specification retrieval.
func TestGetPricingSpec_ALB(t *testing.T) {
	mock := newMockPricingClient("us-east-1", "USD")
	mock.albHourlyPrice = 0.0225
	mock.albLCUPrice = 0.008
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	plugin := NewAWSPublicPlugin("us-east-1", "test-version", mock, logger)

	resp, err := plugin.GetPricingSpec(context.Background(), &pbc.GetPricingSpecRequest{
		Resource: &pbc.ResourceDescriptor{
			Provider:     "aws",
			ResourceType: "elb",
			Sku:          "alb",
			Region:       "us-east-1",
		},
	})

	require.NoError(t, err)
	require.NotNil(t, resp.Spec)
	assert.Equal(t, "per_hour_plus_lcu", resp.Spec.BillingMode)
	assert.Equal(t, 0.0225, resp.Spec.RatePerUnit)
	assert.Equal(t, "hour", resp.Spec.Unit)
	assert.Contains(t, resp.Spec.Description, "Application Load Balancer")
	assert.Equal(t, "aws-public", resp.Spec.Source)
}

// TestGetPricingSpec_NLB verifies NLB pricing specification retrieval.
func TestGetPricingSpec_NLB(t *testing.T) {
	mock := newMockPricingClient("us-east-1", "USD")
	mock.nlbHourlyPrice = 0.0225
	mock.nlbNLCUPrice = 0.006
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	plugin := NewAWSPublicPlugin("us-east-1", "test-version", mock, logger)

	resp, err := plugin.GetPricingSpec(context.Background(), &pbc.GetPricingSpecRequest{
		Resource: &pbc.ResourceDescriptor{
			Provider:     "aws",
			ResourceType: "elb",
			Sku:          "nlb",
			Region:       "us-east-1",
		},
	})

	require.NoError(t, err)
	require.NotNil(t, resp.Spec)
	assert.Equal(t, "per_hour_plus_nlcu", resp.Spec.BillingMode)
	assert.Equal(t, 0.0225, resp.Spec.RatePerUnit)
	assert.Equal(t, "hour", resp.Spec.Unit)
	assert.Contains(t, resp.Spec.Description, "Network Load Balancer")
	assert.Equal(t, "aws-public", resp.Spec.Source)
}

// TestGetPricingSpec_NATGateway verifies NAT Gateway pricing spec retrieval.
func TestGetPricingSpec_NATGateway(t *testing.T) {
	mock := newMockPricingClient("us-east-1", "USD")
	mock.natgwHourlyPrice = 0.045
	mock.natgwDataPrice = 0.045
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	plugin := NewAWSPublicPlugin("us-east-1", "test-version", mock, logger)

	resp, err := plugin.GetPricingSpec(context.Background(), &pbc.GetPricingSpecRequest{
		Resource: &pbc.ResourceDescriptor{
			Provider:     "aws",
			ResourceType: "natgw",
			Sku:          "default", // SKU is required by validation
			Region:       "us-east-1",
		},
	})

	require.NoError(t, err)
	require.NotNil(t, resp.Spec)
	assert.Equal(t, "per_hour_plus_data", resp.Spec.BillingMode)
	assert.Equal(t, 0.045, resp.Spec.RatePerUnit)
	assert.Equal(t, "hour", resp.Spec.Unit)
	assert.Contains(t, resp.Spec.Description, "NAT Gateway")
	assert.Equal(t, "aws-public", resp.Spec.Source)
}

// TestGetPricingSpec_CloudWatch_Logs verifies CloudWatch logs pricing spec.
func TestGetPricingSpec_CloudWatch_Logs(t *testing.T) {
	mock := newMockPricingClient("us-east-1", "USD")
	mock.cwLogsIngestionTiers = []pricing.TierRate{
		{UpTo: 10000, Rate: 0.50},
		{UpTo: 30000, Rate: 0.25},
		{UpTo: 1e15, Rate: 0.10},
	}
	mock.cwLogsStorageRate = 0.03
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	plugin := NewAWSPublicPlugin("us-east-1", "test-version", mock, logger)

	resp, err := plugin.GetPricingSpec(context.Background(), &pbc.GetPricingSpecRequest{
		Resource: &pbc.ResourceDescriptor{
			Provider:     "aws",
			ResourceType: "cloudwatch",
			Sku:          "logs",
			Region:       "us-east-1",
		},
	})

	require.NoError(t, err)
	require.NotNil(t, resp.Spec)
	assert.Equal(t, "tiered_ingestion_plus_storage", resp.Spec.BillingMode)
	assert.Equal(t, 0.50, resp.Spec.RatePerUnit) // First tier rate
	assert.Equal(t, "GB-ingested", resp.Spec.Unit)
	assert.Contains(t, resp.Spec.Description, "CloudWatch Logs")
	assert.Equal(t, "aws-public", resp.Spec.Source)
}

// TestGetPricingSpec_CloudWatch_Metrics verifies CloudWatch metrics pricing spec.
func TestGetPricingSpec_CloudWatch_Metrics(t *testing.T) {
	mock := newMockPricingClient("us-east-1", "USD")
	mock.cwMetricsTiers = []pricing.TierRate{
		{UpTo: 10000, Rate: 0.30},
		{UpTo: 250000, Rate: 0.10},
		{UpTo: 1e15, Rate: 0.05},
	}
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	plugin := NewAWSPublicPlugin("us-east-1", "test-version", mock, logger)

	resp, err := plugin.GetPricingSpec(context.Background(), &pbc.GetPricingSpecRequest{
		Resource: &pbc.ResourceDescriptor{
			Provider:     "aws",
			ResourceType: "cloudwatch",
			Sku:          "metrics",
			Region:       "us-east-1",
		},
	})

	require.NoError(t, err)
	require.NotNil(t, resp.Spec)
	assert.Equal(t, "tiered_per_metric", resp.Spec.BillingMode)
	assert.Equal(t, 0.30, resp.Spec.RatePerUnit) // First tier rate
	assert.Equal(t, "metric-month", resp.Spec.Unit)
	assert.Contains(t, resp.Spec.Description, "custom metrics")
	assert.Equal(t, "aws-public", resp.Spec.Source)
}

// TestGetPricingSpec_UnknownResourceType verifies handling of unsupported resource types.
func TestGetPricingSpec_UnknownResourceType(t *testing.T) {
	mock := newMockPricingClient("us-east-1", "USD")
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	plugin := NewAWSPublicPlugin("us-east-1", "test-version", mock, logger)

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
	plugin := NewAWSPublicPlugin("us-east-1", "test-version", mock, logger)

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
	plugin := NewAWSPublicPlugin("us-east-1", "test-version", mock, logger)

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
	plugin := NewAWSPublicPlugin("us-east-1", "test-version", mock, logger)

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
			plugin := NewAWSPublicPlugin("us-east-1", "test-version", mock, logger)

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
			plugin := NewAWSPublicPlugin("us-east-1", "test-version", mock, logger)

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
