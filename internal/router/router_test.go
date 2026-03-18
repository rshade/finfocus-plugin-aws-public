package router

import (
	"context"
	"testing"

	"github.com/goccy/go-json"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/rs/zerolog"
	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
)

// TestPlugin_Name verifies that the router returns the correct plugin name.
func TestPlugin_Name(t *testing.T) {
	logger := zerolog.New(zerolog.NewTestWriter(t))
	r := NewPlugin("1.0.0", logger, t.TempDir(), true, nil)

	assert.Equal(t, "finfocus-plugin-aws-public", r.Name())
}

// TestPlugin_GetPluginInfo verifies the router returns correct metadata
// including the multi-region-router type designation.
func TestPlugin_GetPluginInfo(t *testing.T) {
	logger := zerolog.New(zerolog.NewTestWriter(t))
	r := NewPlugin("1.0.0", logger, t.TempDir(), true, nil)

	resp, err := r.GetPluginInfo(context.Background(), &pbc.GetPluginInfoRequest{})

	require.NoError(t, err)
	assert.Equal(t, "finfocus-plugin-aws-public", resp.GetName())
	assert.Equal(t, "1.0.0", resp.GetVersion())
	assert.Equal(t, pluginsdk.SpecVersion, resp.GetSpecVersion())
	assert.Equal(t, []string{"aws"}, resp.GetProviders())
	assert.Equal(t, "multi-region-router", resp.GetMetadata()["type"])
}

// TestPlugin_DismissRecommendation_Unimplemented verifies that
// DismissRecommendation returns Unimplemented error as the router is stateless.
func TestPlugin_DismissRecommendation_Unimplemented(t *testing.T) {
	logger := zerolog.New(zerolog.NewTestWriter(t))
	r := NewPlugin("1.0.0", logger, t.TempDir(), true, nil)

	_, err := r.DismissRecommendation(context.Background(), &pbc.DismissRecommendationRequest{})

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Unimplemented, st.Code())
}

// TestPlugin_GetBudgets_Unimplemented verifies that GetBudgets returns
// Unimplemented error as the router has no budget support.
func TestPlugin_GetBudgets_Unimplemented(t *testing.T) {
	logger := zerolog.New(zerolog.NewTestWriter(t))
	r := NewPlugin("1.0.0", logger, t.TempDir(), true, nil)

	_, err := r.GetBudgets(context.Background(), &pbc.GetBudgetsRequest{})

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Unimplemented, st.Code())
}

// TestExtractRegionFromResource verifies region extraction from a ResourceDescriptor.
func TestExtractRegionFromResource(t *testing.T) {
	tests := []struct {
		name     string
		resource *pbc.ResourceDescriptor
		expected string
	}{
		{
			name:     "valid region",
			resource: &pbc.ResourceDescriptor{Region: "us-east-1"},
			expected: "us-east-1",
		},
		{
			name:     "empty region",
			resource: &pbc.ResourceDescriptor{Region: ""},
			expected: "",
		},
		{
			name:     "nil resource",
			resource: nil,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractRegionFromResource(tt.resource)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestExtractRegionFromActualCostRequest verifies region extraction from GetActualCostRequest.
func TestExtractRegionFromActualCostRequest(t *testing.T) {
	resourceJSON, err := json.Marshal(&pbc.ResourceDescriptor{
		Provider:     "aws",
		ResourceType: "ec2",
		Sku:          "t3.micro",
		Region:       "eu-west-1",
	})
	require.NoError(t, err)

	tests := []struct {
		name     string
		req      *pbc.GetActualCostRequest
		expected string
	}{
		{
			name: "region in tags",
			req: &pbc.GetActualCostRequest{
				Tags: map[string]string{"region": "us-west-2"},
			},
			expected: "us-west-2",
		},
		{
			name: "region in resource id json",
			req: &pbc.GetActualCostRequest{
				ResourceId: string(resourceJSON),
			},
			expected: "eu-west-1",
		},
		{
			name: "tags take precedence over resource id json region",
			req: &pbc.GetActualCostRequest{
				ResourceId: string(resourceJSON),
				Tags:       map[string]string{"region": "us-west-1"},
			},
			expected: "us-west-1",
		},
		{
			name: "no region in tags",
			req: &pbc.GetActualCostRequest{
				Tags: map[string]string{"provider": "aws"},
			},
			expected: "",
		},
		{
			name: "region from ARN-format resource id",
			req: &pbc.GetActualCostRequest{
				ResourceId: "arn:aws:ec2:us-east-1:123456789012:instance/i-abc123",
			},
			expected: "us-east-1",
		},
		{
			name: "region from ARN with different region",
			req: &pbc.GetActualCostRequest{
				ResourceId: "arn:aws:rds:ap-southeast-1:123456789012:db:mydb",
			},
			expected: "ap-southeast-1",
		},
		{
			name: "region from GovCloud ARN",
			req: &pbc.GetActualCostRequest{
				ResourceId: "arn:aws-us-gov:ec2:us-gov-west-1:123456789012:instance/i-abc123",
			},
			expected: "us-gov-west-1",
		},
		{
			name: "tags take precedence over ARN region",
			req: &pbc.GetActualCostRequest{
				ResourceId: "arn:aws:ec2:us-east-1:123456789012:instance/i-abc123",
				Tags:       map[string]string{"region": "us-west-2"},
			},
			expected: "us-west-2",
		},
		{
			name: "ARN with empty region (S3 global)",
			req: &pbc.GetActualCostRequest{
				ResourceId: "arn:aws:s3:::my-bucket",
			},
			expected: "",
		},
		{
			name: "non-ARN non-JSON resource id",
			req: &pbc.GetActualCostRequest{
				ResourceId: "i-abc123",
			},
			expected: "",
		},
		{
			name: "invalid resource id json",
			req: &pbc.GetActualCostRequest{
				ResourceId: "{invalid",
			},
			expected: "",
		},
		{
			name:     "nil tags",
			req:      &pbc.GetActualCostRequest{},
			expected: "",
		},
		{
			name:     "nil request",
			req:      nil,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractRegionFromActualCostRequest(tt.req)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestExtractRegionFromEstimateCostRequest verifies region extraction from EstimateCostRequest
// by parsing the Attributes protobuf Struct.
func TestExtractRegionFromEstimateCostRequest(t *testing.T) {
	tests := []struct {
		name     string
		req      *pbc.EstimateCostRequest
		expected string
	}{
		{
			name: "region in attributes",
			req: &pbc.EstimateCostRequest{
				Attributes: &structpb.Struct{
					Fields: map[string]*structpb.Value{
						"region": structpb.NewStringValue("eu-west-1"),
					},
				},
			},
			expected: "eu-west-1",
		},
		{
			name: "no region in attributes",
			req: &pbc.EstimateCostRequest{
				Attributes: &structpb.Struct{
					Fields: map[string]*structpb.Value{
						"instanceType": structpb.NewStringValue("t3.micro"),
					},
				},
			},
			expected: "",
		},
		{
			name:     "nil attributes",
			req:      &pbc.EstimateCostRequest{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractRegionFromEstimateCostRequest(tt.req)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestPlugin_GetProjectedCost_EmptyRegion verifies that GetProjectedCost returns
// InvalidArgument when the ResourceDescriptor has no region.
func TestPlugin_GetProjectedCost_EmptyRegion(t *testing.T) {
	logger := zerolog.New(zerolog.NewTestWriter(t))
	r := NewPlugin("1.0.0", logger, t.TempDir(), true, nil)

	_, err := r.GetProjectedCost(context.Background(), &pbc.GetProjectedCostRequest{
		Resource: &pbc.ResourceDescriptor{
			Provider:     "aws",
			ResourceType: "ec2",
			Sku:          "t3.micro",
			Region:       "", // empty region
		},
	})

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

// TestPlugin_Supports_EmptyRegion verifies that Supports returns
// InvalidArgument when the ResourceDescriptor has no region.
func TestPlugin_Supports_EmptyRegion(t *testing.T) {
	logger := zerolog.New(zerolog.NewTestWriter(t))
	r := NewPlugin("1.0.0", logger, t.TempDir(), true, nil)

	_, err := r.Supports(context.Background(), &pbc.SupportsRequest{
		Resource: &pbc.ResourceDescriptor{
			Provider:     "aws",
			ResourceType: "ec2",
		},
	})

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

// TestPlugin_GetActualCost_EmptyRegion verifies that GetActualCost returns
// InvalidArgument when no region can be extracted from the request.
func TestPlugin_GetActualCost_EmptyRegion(t *testing.T) {
	logger := zerolog.New(zerolog.NewTestWriter(t))
	r := NewPlugin("1.0.0", logger, t.TempDir(), true, nil)

	_, err := r.GetActualCost(context.Background(), &pbc.GetActualCostRequest{
		Tags: map[string]string{"provider": "aws"},
	})

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

// TestPlugin_EstimateCost_EmptyRegion verifies that EstimateCost returns
// InvalidArgument when no region can be extracted from the request.
func TestPlugin_EstimateCost_EmptyRegion(t *testing.T) {
	logger := zerolog.New(zerolog.NewTestWriter(t))
	r := NewPlugin("1.0.0", logger, t.TempDir(), true, nil)

	_, err := r.EstimateCost(context.Background(), &pbc.EstimateCostRequest{})

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

// TestPlugin_GetPricingSpec_EmptyRegion verifies that GetPricingSpec returns
// InvalidArgument when the ResourceDescriptor has no region.
func TestPlugin_GetPricingSpec_EmptyRegion(t *testing.T) {
	logger := zerolog.New(zerolog.NewTestWriter(t))
	r := NewPlugin("1.0.0", logger, t.TempDir(), true, nil)

	_, err := r.GetPricingSpec(context.Background(), &pbc.GetPricingSpecRequest{
		Resource: &pbc.ResourceDescriptor{},
	})

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

// TestPlugin_GetRecommendations_EmptyResources verifies that GetRecommendations
// returns InvalidArgument when no target resources are provided.
func TestPlugin_GetRecommendations_EmptyResources(t *testing.T) {
	logger := zerolog.New(zerolog.NewTestWriter(t))
	r := NewPlugin("1.0.0", logger, t.TempDir(), true, nil)

	_, err := r.GetRecommendations(context.Background(), &pbc.GetRecommendationsRequest{})

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

// TestPlugin_OfflineMode_NoChild verifies that in offline mode, requesting a
// region with no pre-installed binary returns a helpful error message.
func TestPlugin_OfflineMode_NoChild(t *testing.T) {
	logger := zerolog.New(zerolog.NewTestWriter(t))
	r := NewPlugin("1.0.0", logger, t.TempDir(), true, nil)

	_, err := r.GetProjectedCost(context.Background(), &pbc.GetProjectedCostRequest{
		Resource: &pbc.ResourceDescriptor{
			Provider:     "aws",
			ResourceType: "ec2",
			Sku:          "t3.micro",
			Region:       "us-east-1",
		},
	})

	require.Error(t, err)
	// Should include install instructions
	assert.Contains(t, err.Error(), "us-east-1")
}

// TestGetTraceID_FromContext verifies trace_id extraction from gRPC metadata.
func TestGetTraceID_FromContext(t *testing.T) {
	logger := zerolog.New(zerolog.NewTestWriter(t))
	r := NewPlugin("1.0.0", logger, t.TempDir(), true, nil)

	// Test with trace_id in metadata
	md := metadata.New(map[string]string{
		pluginsdk.TraceIDMetadataKey: "test-trace-123",
	})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	traceID := r.getTraceID(ctx)
	assert.Equal(t, "test-trace-123", traceID)
}

// TestGetTraceID_GeneratesUUID verifies that a UUID is generated when no trace_id
// is present in the context.
func TestGetTraceID_GeneratesUUID(t *testing.T) {
	logger := zerolog.New(zerolog.NewTestWriter(t))
	r := NewPlugin("1.0.0", logger, t.TempDir(), true, nil)

	traceID := r.getTraceID(context.Background())

	// Should be a non-empty UUID
	assert.NotEmpty(t, traceID)
	assert.Len(t, traceID, 36) // UUID format: 8-4-4-4-12
}

// TestPropagateTraceID verifies that trace_id is set in outgoing gRPC metadata
// without dropping any existing outgoing metadata.
func TestPropagateTraceID(t *testing.T) {
	base := metadata.NewOutgoingContext(context.Background(), metadata.Pairs("x-existing", "1"))
	ctx := propagateTraceID(base, "my-trace-id")

	md, ok := metadata.FromOutgoingContext(ctx)
	require.True(t, ok)

	values := md.Get(pluginsdk.TraceIDMetadataKey)
	require.Len(t, values, 1)
	assert.Equal(t, "my-trace-id", values[0])

	assert.Equal(t, []string{"1"}, md.Get("x-existing"))
}

// TestNewInvalidRegionError verifies the error structure for missing region.
func TestNewInvalidRegionError(t *testing.T) {
	err := newInvalidRegionError("GetProjectedCost")
	require.Error(t, err)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	assert.Contains(t, st.Message(), "region is required")
	assert.Contains(t, st.Message(), "GetProjectedCost")
}

// TestWrapRegionError verifies the error wrapping for region-related failures.
func TestWrapRegionError(t *testing.T) {
	err := wrapRegionError("us-east-1", assert.AnError)
	require.Error(t, err)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Unavailable, st.Code())
	assert.Contains(t, st.Message(), "us-east-1")
}
