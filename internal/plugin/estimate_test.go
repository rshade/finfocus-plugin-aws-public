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
	"google.golang.org/protobuf/types/known/structpb"
)

// TestParsePulumiResourceType verifies parsing of Pulumi resource type strings.
//
// Pulumi resource types follow the format "provider:module/resource:Type".
// This test ensures correct parsing and appropriate error handling for
// malformed resource type strings.
func TestParsePulumiResourceType(t *testing.T) {
	tests := []struct {
		name         string
		resourceType string
		wantProvider string
		wantModule   string
		wantResource string
		wantErr      bool
	}{
		{
			name:         "valid EC2 instance",
			resourceType: "aws:ec2/instance:Instance",
			wantProvider: "aws",
			wantModule:   "ec2",
			wantResource: "Instance",
			wantErr:      false,
		},
		{
			name:         "valid EBS volume",
			resourceType: "aws:ebs/volume:Volume",
			wantProvider: "aws",
			wantModule:   "ebs",
			wantResource: "Volume",
			wantErr:      false,
		},
		{
			name:         "valid S3 bucket",
			resourceType: "aws:s3/bucket:Bucket",
			wantProvider: "aws",
			wantModule:   "s3",
			wantResource: "Bucket",
			wantErr:      false,
		},
		{
			name:         "valid GCP compute instance",
			resourceType: "gcp:compute/instance:Instance",
			wantProvider: "gcp",
			wantModule:   "compute",
			wantResource: "Instance",
			wantErr:      false,
		},
		{
			name:         "missing provider separator",
			resourceType: "awsec2/instance:Instance",
			wantErr:      true,
		},
		{
			name:         "missing module separator",
			resourceType: "aws:ec2instance:Instance",
			wantErr:      true,
		},
		{
			name:         "missing resource separator",
			resourceType: "aws:ec2/instanceInstance",
			wantErr:      true,
		},
		{
			name:         "empty string",
			resourceType: "",
			wantErr:      true,
		},
		{
			name:         "just provider",
			resourceType: "aws",
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, err := parsePulumiResourceType(tt.resourceType)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantProvider, info.provider)
			assert.Equal(t, tt.wantModule, info.module)
			assert.Equal(t, tt.wantResource, info.resource)
		})
	}
}

// TestGetStringAttr verifies string attribute extraction from protobuf Struct.
func TestGetStringAttr(t *testing.T) {
	tests := []struct {
		name    string
		attrs   *structpb.Struct
		key     string
		wantVal string
		wantOk  bool
	}{
		{
			name: "existing string attribute",
			attrs: &structpb.Struct{
				Fields: map[string]*structpb.Value{
					"instanceType": structpb.NewStringValue("t3.micro"),
				},
			},
			key:     "instanceType",
			wantVal: "t3.micro",
			wantOk:  true,
		},
		{
			name: "missing attribute",
			attrs: &structpb.Struct{
				Fields: map[string]*structpb.Value{
					"other": structpb.NewStringValue("value"),
				},
			},
			key:     "instanceType",
			wantVal: "",
			wantOk:  false,
		},
		{
			name: "empty string value",
			attrs: &structpb.Struct{
				Fields: map[string]*structpb.Value{
					"instanceType": structpb.NewStringValue(""),
				},
			},
			key:     "instanceType",
			wantVal: "",
			wantOk:  false,
		},
		{
			name:    "nil attrs",
			attrs:   nil,
			key:     "instanceType",
			wantVal: "",
			wantOk:  false,
		},
		{
			name: "nil fields",
			attrs: &structpb.Struct{
				Fields: nil,
			},
			key:     "instanceType",
			wantVal: "",
			wantOk:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, ok := getStringAttr(tt.attrs, tt.key)
			assert.Equal(t, tt.wantVal, val)
			assert.Equal(t, tt.wantOk, ok)
		})
	}
}

// TestGetNumberAttr verifies number attribute extraction from protobuf Struct.
func TestGetNumberAttr(t *testing.T) {
	tests := []struct {
		name    string
		attrs   *structpb.Struct
		key     string
		wantVal float64
		wantOk  bool
	}{
		{
			name: "existing number attribute",
			attrs: &structpb.Struct{
				Fields: map[string]*structpb.Value{
					"size": structpb.NewNumberValue(100),
				},
			},
			key:     "size",
			wantVal: 100,
			wantOk:  true,
		},
		{
			name: "string number attribute",
			attrs: &structpb.Struct{
				Fields: map[string]*structpb.Value{
					"size": structpb.NewStringValue("50"),
				},
			},
			key:     "size",
			wantVal: 50,
			wantOk:  true,
		},
		{
			name: "missing attribute",
			attrs: &structpb.Struct{
				Fields: map[string]*structpb.Value{
					"other": structpb.NewNumberValue(100),
				},
			},
			key:     "size",
			wantVal: 0,
			wantOk:  false,
		},
		{
			name: "zero value",
			attrs: &structpb.Struct{
				Fields: map[string]*structpb.Value{
					"size": structpb.NewNumberValue(0),
				},
			},
			key:     "size",
			wantVal: 0,
			wantOk:  true, // zero is a valid value when the attribute exists
		},
		{
			name: "invalid string",
			attrs: &structpb.Struct{
				Fields: map[string]*structpb.Value{
					"size": structpb.NewStringValue("not-a-number"),
				},
			},
			key:     "size",
			wantVal: 0,
			wantOk:  false,
		},
		{
			name:    "nil attrs",
			attrs:   nil,
			key:     "size",
			wantVal: 0,
			wantOk:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, ok := getNumberAttr(tt.attrs, tt.key)
			assert.Equal(t, tt.wantVal, val)
			assert.Equal(t, tt.wantOk, ok)
		})
	}
}

// TestEstimateCost_EC2 verifies EC2 instance cost estimation via EstimateCost API.
//
// This test validates that the EstimateCost method correctly parses Pulumi
// resource types for EC2 instances and calculates monthly costs based on
// the embedded pricing data.
func TestEstimateCost_EC2(t *testing.T) {
	mock := newMockPricingClient("us-east-1", "USD")
	mock.ec2Prices["t3.micro/Linux/Shared"] = 0.0104
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	plugin := NewAWSPublicPlugin("us-east-1", "test-version", mock, logger)

	attrs, err := structpb.NewStruct(map[string]interface{}{
		"instanceType": "t3.micro",
	})
	require.NoError(t, err)

	resp, err := plugin.EstimateCost(context.Background(), &pbc.EstimateCostRequest{
		ResourceType: "aws:ec2/instance:Instance",
		Attributes:   attrs,
	})

	require.NoError(t, err)
	assert.Equal(t, "USD", resp.Currency)
	// 0.0104 * 730 = 7.592
	assert.InDelta(t, 7.592, resp.CostMonthly, 0.001)
}

// TestEstimateCost_EBS verifies EBS volume cost estimation via EstimateCost API.
func TestEstimateCost_EBS(t *testing.T) {
	mock := newMockPricingClient("us-east-1", "USD")
	mock.ebsPrices["gp3"] = 0.08
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	plugin := NewAWSPublicPlugin("us-east-1", "test-version", mock, logger)

	attrs, err := structpb.NewStruct(map[string]interface{}{
		"type": "gp3",
		"size": float64(100),
	})
	require.NoError(t, err)

	resp, err := plugin.EstimateCost(context.Background(), &pbc.EstimateCostRequest{
		ResourceType: "aws:ebs/volume:Volume",
		Attributes:   attrs,
	})

	require.NoError(t, err)
	assert.Equal(t, "USD", resp.Currency)
	// 0.08 * 100 = 8.0
	assert.InDelta(t, 8.0, resp.CostMonthly, 0.001)
}

// TestEstimateCost_EBS_DefaultSize verifies EBS defaults to 8GB when size not specified.
func TestEstimateCost_EBS_DefaultSize(t *testing.T) {
	mock := newMockPricingClient("us-east-1", "USD")
	mock.ebsPrices["gp2"] = 0.10
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	plugin := NewAWSPublicPlugin("us-east-1", "test-version", mock, logger)

	attrs, err := structpb.NewStruct(map[string]interface{}{
		"type": "gp2",
		// no size specified
	})
	require.NoError(t, err)

	resp, err := plugin.EstimateCost(context.Background(), &pbc.EstimateCostRequest{
		ResourceType: "aws:ebs/volume:Volume",
		Attributes:   attrs,
	})

	require.NoError(t, err)
	// 0.10 * 8 (default) = 0.80
	assert.InDelta(t, 0.80, resp.CostMonthly, 0.001)
}

// TestEstimateCost_RegionFromAvailabilityZone verifies region extraction from AZ.
func TestEstimateCost_RegionFromAvailabilityZone(t *testing.T) {
	mock := newMockPricingClient("us-east-1", "USD")
	mock.ec2Prices["t3.micro/Linux/Shared"] = 0.0104
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	plugin := NewAWSPublicPlugin("us-east-1", "test-version", mock, logger)

	attrs, err := structpb.NewStruct(map[string]interface{}{
		"instanceType":     "t3.micro",
		"availabilityZone": "us-east-1a",
	})
	require.NoError(t, err)

	resp, err := plugin.EstimateCost(context.Background(), &pbc.EstimateCostRequest{
		ResourceType: "aws:ec2/instance:Instance",
		Attributes:   attrs,
	})

	require.NoError(t, err)
	assert.InDelta(t, 7.592, resp.CostMonthly, 0.001)
}

// TestEstimateCost_WrongRegion verifies $0 is returned for wrong region.
func TestEstimateCost_WrongRegion(t *testing.T) {
	mock := newMockPricingClient("us-east-1", "USD")
	mock.ec2Prices["t3.micro/Linux/Shared"] = 0.0104
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	plugin := NewAWSPublicPlugin("us-east-1", "test-version", mock, logger)

	attrs, err := structpb.NewStruct(map[string]interface{}{
		"instanceType": "t3.micro",
		"region":       "eu-west-1", // different from plugin region
	})
	require.NoError(t, err)

	resp, err := plugin.EstimateCost(context.Background(), &pbc.EstimateCostRequest{
		ResourceType: "aws:ec2/instance:Instance",
		Attributes:   attrs,
	})

	require.NoError(t, err)
	assert.Equal(t, float64(0), resp.CostMonthly)
}

// TestEstimateCost_NonAWSProvider verifies $0 for non-AWS providers.
func TestEstimateCost_NonAWSProvider(t *testing.T) {
	mock := newMockPricingClient("us-east-1", "USD")
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	plugin := NewAWSPublicPlugin("us-east-1", "test-version", mock, logger)

	resp, err := plugin.EstimateCost(context.Background(), &pbc.EstimateCostRequest{
		ResourceType: "gcp:compute/instance:Instance",
		Attributes:   &structpb.Struct{Fields: make(map[string]*structpb.Value)},
	})

	require.NoError(t, err)
	assert.Equal(t, float64(0), resp.CostMonthly)
}

// TestEstimateCost_UnsupportedModule verifies $0 for unsupported AWS modules.
func TestEstimateCost_UnsupportedModule(t *testing.T) {
	mock := newMockPricingClient("us-east-1", "USD")
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	plugin := NewAWSPublicPlugin("us-east-1", "test-version", mock, logger)

	resp, err := plugin.EstimateCost(context.Background(), &pbc.EstimateCostRequest{
		ResourceType: "aws:rds/instance:Instance",
		Attributes:   &structpb.Struct{Fields: make(map[string]*structpb.Value)},
	})

	require.NoError(t, err)
	assert.Equal(t, float64(0), resp.CostMonthly)
}

// TestEstimateCost_NilRequest verifies error handling for nil request.
func TestEstimateCost_NilRequest(t *testing.T) {
	mock := newMockPricingClient("us-east-1", "USD")
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	plugin := NewAWSPublicPlugin("us-east-1", "test-version", mock, logger)

	_, err := plugin.EstimateCost(context.Background(), nil)

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

// TestEstimateCost_MissingResourceType verifies error handling for missing resource type.
func TestEstimateCost_MissingResourceType(t *testing.T) {
	mock := newMockPricingClient("us-east-1", "USD")
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	plugin := NewAWSPublicPlugin("us-east-1", "test-version", mock, logger)

	_, err := plugin.EstimateCost(context.Background(), &pbc.EstimateCostRequest{
		ResourceType: "",
	})

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

// TestEstimateCost_InvalidResourceTypeFormat verifies error for malformed resource types.
func TestEstimateCost_InvalidResourceTypeFormat(t *testing.T) {
	mock := newMockPricingClient("us-east-1", "USD")
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	plugin := NewAWSPublicPlugin("us-east-1", "test-version", mock, logger)

	_, err := plugin.EstimateCost(context.Background(), &pbc.EstimateCostRequest{
		ResourceType: "invalid-format",
	})

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

// TestEstimateCost_NilAttributes verifies handling of nil attributes.
func TestEstimateCost_NilAttributes(t *testing.T) {
	mock := newMockPricingClient("us-east-1", "USD")
	mock.ec2Prices["t3.micro/Linux/Shared"] = 0.0104
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	plugin := NewAWSPublicPlugin("us-east-1", "test-version", mock, logger)

	resp, err := plugin.EstimateCost(context.Background(), &pbc.EstimateCostRequest{
		ResourceType: "aws:ec2/instance:Instance",
		Attributes:   nil, // nil attributes
	})

	// Should not error, but return $0 since no instanceType specified
	require.NoError(t, err)
	assert.Equal(t, float64(0), resp.CostMonthly)
}

// TestEstimateCost_MissingInstanceType verifies $0 when EC2 has no instanceType.
func TestEstimateCost_MissingInstanceType(t *testing.T) {
	mock := newMockPricingClient("us-east-1", "USD")
	mock.ec2Prices["t3.micro/Linux/Shared"] = 0.0104
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	plugin := NewAWSPublicPlugin("us-east-1", "test-version", mock, logger)

	attrs, err := structpb.NewStruct(map[string]interface{}{
		"ami": "ami-12345678",
		// missing instanceType
	})
	require.NoError(t, err)

	resp, err := plugin.EstimateCost(context.Background(), &pbc.EstimateCostRequest{
		ResourceType: "aws:ec2/instance:Instance",
		Attributes:   attrs,
	})

	require.NoError(t, err)
	assert.Equal(t, float64(0), resp.CostMonthly)
}

// TestEstimateCost_UnknownInstanceType verifies $0 for unknown instance types.
func TestEstimateCost_UnknownInstanceType(t *testing.T) {
	mock := newMockPricingClient("us-east-1", "USD")
	// Don't add any prices - instance type won't be found
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	plugin := NewAWSPublicPlugin("us-east-1", "test-version", mock, logger)

	attrs, err := structpb.NewStruct(map[string]interface{}{
		"instanceType": "unknown.type",
	})
	require.NoError(t, err)

	resp, err := plugin.EstimateCost(context.Background(), &pbc.EstimateCostRequest{
		ResourceType: "aws:ec2/instance:Instance",
		Attributes:   attrs,
	})

	require.NoError(t, err)
	assert.Equal(t, float64(0), resp.CostMonthly)
}

// TestEstimateCost_NonInstanceEC2Resource verifies $0 for non-Instance EC2 resources.
func TestEstimateCost_NonInstanceEC2Resource(t *testing.T) {
	mock := newMockPricingClient("us-east-1", "USD")
	mock.ec2Prices["t3.micro/Linux/Shared"] = 0.0104
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	plugin := NewAWSPublicPlugin("us-east-1", "test-version", mock, logger)

	resp, err := plugin.EstimateCost(context.Background(), &pbc.EstimateCostRequest{
		ResourceType: "aws:ec2/securityGroup:SecurityGroup",
		Attributes:   &structpb.Struct{Fields: make(map[string]*structpb.Value)},
	})

	require.NoError(t, err)
	assert.Equal(t, float64(0), resp.CostMonthly)
}
