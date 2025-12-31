package plugin

import (
	"context"
	"testing"
	"time"

	"github.com/rs/zerolog"
	pbc "github.com/rshade/pulumicost-spec/sdk/go/proto/pulumicost/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestValidateProjectedCostRequest(t *testing.T) {
	logger := zerolog.Nop()
	p := NewAWSPublicPlugin("us-east-1", nil, logger)
	ctx := context.Background()

	tests := []struct {
		name      string
		req       *pbc.GetProjectedCostRequest
		wantError bool
		errCode   codes.Code
		pbcCode   pbc.ErrorCode
	}{
		{
			name: "valid request",
			req: &pbc.GetProjectedCostRequest{
				Resource: &pbc.ResourceDescriptor{
					Provider:     "aws",
					ResourceType: "aws:ec2/instance:Instance",
					Sku:          "t3.micro",
					Region:       "us-east-1",
				},
			},
			wantError: false,
		},
		{
			name:      "nil request",
			req:       nil,
			wantError: true,
			errCode:   codes.InvalidArgument,
			pbcCode:   pbc.ErrorCode_ERROR_CODE_INVALID_RESOURCE,
		},
		{
			name: "missing resource",
			req: &pbc.GetProjectedCostRequest{
				// Resource is nil
			},
			wantError: true,
			errCode:   codes.InvalidArgument,
			pbcCode:   pbc.ErrorCode_ERROR_CODE_INVALID_RESOURCE,
		},
		{
			name: "region mismatch",
			req: &pbc.GetProjectedCostRequest{
				Resource: &pbc.ResourceDescriptor{
					Provider:     "aws",
					ResourceType: "aws:ec2/instance:Instance",
					Sku:          "t3.micro",
					Region:       "us-west-2",
				},
			},
			wantError: true,
			errCode:   codes.FailedPrecondition,
			pbcCode:   pbc.ErrorCode_ERROR_CODE_UNSUPPORTED_REGION,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := p.ValidateProjectedCostRequest(ctx, tt.req)
			if tt.wantError {
				require.Error(t, err)
				st, ok := status.FromError(err)
				require.True(t, ok)
				assert.Equal(t, tt.errCode, st.Code())

				// Check details
				details := st.Details()
				require.NotEmpty(t, details)
				errDetail, ok := details[0].(*pbc.ErrorDetail)
				require.True(t, ok)
				assert.Equal(t, tt.pbcCode, errDetail.Code)
				assert.Contains(t, errDetail.Details, "trace_id")
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.req.Resource, res)
			}
		})
	}
}

func TestValidateActualCostRequest(t *testing.T) {
	logger := zerolog.Nop()
	p := NewAWSPublicPlugin("us-east-1", nil, logger)
	ctx := context.Background()
	now := time.Now()

	tests := []struct {
		name      string
		req       *pbc.GetActualCostRequest
		wantError bool
		errCode   codes.Code
		pbcCode   pbc.ErrorCode
	}{
		{
			name: "valid request with tags",
			req: &pbc.GetActualCostRequest{
				ResourceId: "invalid-json", // Satisfy SDK validation, trigger tag fallback
				Start:      timestamppb.New(now.Add(-1 * time.Hour)),
				End:        timestamppb.New(now),
				Tags: map[string]string{
					"provider":      "aws",
					"resource_type": "aws:ec2/instance:Instance",
					"sku":           "t3.micro",
					"region":        "us-east-1",
				},
			},
			wantError: false,
		},
		{
			name:      "nil request",
			req:       nil,
			wantError: true,
			errCode:   codes.InvalidArgument,
			pbcCode:   pbc.ErrorCode_ERROR_CODE_INVALID_RESOURCE,
		},
		{
			name: "missing timestamps",
			req: &pbc.GetActualCostRequest{
				ResourceId: "invalid-json",
				Tags:       map[string]string{"region": "us-east-1"},
			},
			wantError: true,
			errCode:   codes.InvalidArgument,
			pbcCode:   pbc.ErrorCode_ERROR_CODE_INVALID_RESOURCE,
		},
		{
			name: "region mismatch",
			req: &pbc.GetActualCostRequest{
				ResourceId: "invalid-json",
				Start:      timestamppb.New(now.Add(-1 * time.Hour)),
				End:        timestamppb.New(now),
				Tags: map[string]string{
					"provider":      "aws",
					"resource_type": "aws:ec2/instance:Instance",
					"sku":           "t3.micro",
					"region":        "us-west-2",
				},
			},
			wantError: true,
			errCode:   codes.FailedPrecondition,
			pbcCode:   pbc.ErrorCode_ERROR_CODE_UNSUPPORTED_REGION,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, _, err := p.ValidateActualCostRequest(ctx, tt.req)
			if tt.wantError {
				require.Error(t, err)
				st, ok := status.FromError(err)
				require.True(t, ok)
				assert.Equal(t, tt.errCode, st.Code())

				// Check details
				details := st.Details()
				require.NotEmpty(t, details)
				errDetail, ok := details[0].(*pbc.ErrorDetail)
				require.True(t, ok)
				assert.Equal(t, tt.pbcCode, errDetail.Code)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.req.Tags["region"], res.Region)
			}
		})
	}
}

func TestRegionMismatchError(t *testing.T) {
	logger := zerolog.Nop()
	p := NewAWSPublicPlugin("us-east-1", nil, logger)

	err := p.RegionMismatchError("trace-123", "us-west-2")
	require.Error(t, err)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.FailedPrecondition, st.Code())

	details := st.Details()
	require.Len(t, details, 1)

	errDetail, ok := details[0].(*pbc.ErrorDetail)
	require.True(t, ok)
	assert.Equal(t, pbc.ErrorCode_ERROR_CODE_UNSUPPORTED_REGION, errDetail.Code)
	assert.Equal(t, "trace-123", errDetail.Details["trace_id"])
	assert.Equal(t, "us-east-1", errDetail.Details["plugin_region"])
	assert.Equal(t, "us-west-2", errDetail.Details["resource_region"])
}

// TestValidateActualCostRequest_ARN tests ARN-based resource identification (T072)
func TestValidateActualCostRequest_ARN(t *testing.T) {
	logger := zerolog.Nop()
	p := NewAWSPublicPlugin("us-east-1", nil, logger)
	ctx := context.Background()
	now := time.Now()

	tests := []struct {
		name         string
		req          *pbc.GetActualCostRequest
		wantError    bool
		errCode      codes.Code
		pbcCode      pbc.ErrorCode
		wantResource *pbc.ResourceDescriptor
	}{
		{
			name: "valid ARN with SKU in tags",
			req: &pbc.GetActualCostRequest{
				Arn:   "arn:aws:ec2:us-east-1:123456789012:instance/i-abc123",
				Start: timestamppb.New(now.Add(-1 * time.Hour)),
				End:   timestamppb.New(now),
				Tags: map[string]string{
					"sku": "t3.micro",
				},
			},
			wantError: false,
			wantResource: &pbc.ResourceDescriptor{
				Provider:     "aws",
				ResourceType: "ec2",
				Sku:          "t3.micro",
				Region:       "us-east-1",
				Tags:         map[string]string{},
			},
		},
		{
			name: "ARN with instanceType alias",
			req: &pbc.GetActualCostRequest{
				Arn:   "arn:aws:ec2:us-east-1:123456789012:instance/i-abc123",
				Start: timestamppb.New(now.Add(-1 * time.Hour)),
				End:   timestamppb.New(now),
				Tags: map[string]string{
					"instanceType": "m5.large",
				},
			},
			wantError: false,
			wantResource: &pbc.ResourceDescriptor{
				Provider:     "aws",
				ResourceType: "ec2",
				Sku:          "m5.large",
				Region:       "us-east-1",
				Tags:         map[string]string{},
			},
		},
		{
			name: "EBS volume ARN maps to ebs resource type",
			req: &pbc.GetActualCostRequest{
				Arn:   "arn:aws:ec2:us-east-1:123456789012:volume/vol-xyz789",
				Start: timestamppb.New(now.Add(-1 * time.Hour)),
				End:   timestamppb.New(now),
				Tags: map[string]string{
					"volumeType": "gp3",
				},
			},
			wantError: false,
			wantResource: &pbc.ResourceDescriptor{
				Provider:     "aws",
				ResourceType: "ebs",
				Sku:          "gp3",
				Region:       "us-east-1",
				Tags:         map[string]string{},
			},
		},
		{
			name: "ARN region mismatch",
			req: &pbc.GetActualCostRequest{
				Arn:   "arn:aws:ec2:us-west-2:123456789012:instance/i-abc123",
				Start: timestamppb.New(now.Add(-1 * time.Hour)),
				End:   timestamppb.New(now),
				Tags: map[string]string{
					"sku": "t3.micro",
				},
			},
			wantError: true,
			errCode:   codes.FailedPrecondition,
			pbcCode:   pbc.ErrorCode_ERROR_CODE_UNSUPPORTED_REGION,
		},
		{
			name: "ARN without SKU tag",
			req: &pbc.GetActualCostRequest{
				Arn:   "arn:aws:ec2:us-east-1:123456789012:instance/i-abc123",
				Start: timestamppb.New(now.Add(-1 * time.Hour)),
				End:   timestamppb.New(now),
				Tags:  map[string]string{}, // Missing SKU
			},
			wantError: true,
			errCode:   codes.InvalidArgument,
			pbcCode:   pbc.ErrorCode_ERROR_CODE_INVALID_RESOURCE,
		},
		{
			name: "invalid ARN format",
			req: &pbc.GetActualCostRequest{
				Arn:   "not-an-arn",
				Start: timestamppb.New(now.Add(-1 * time.Hour)),
				End:   timestamppb.New(now),
				Tags: map[string]string{
					"sku": "t3.micro",
				},
			},
			wantError: true,
			errCode:   codes.InvalidArgument,
			pbcCode:   pbc.ErrorCode_ERROR_CODE_INVALID_RESOURCE,
		},
		{
			name: "S3 global ARN (empty region) uses plugin region",
			req: &pbc.GetActualCostRequest{
				Arn:   "arn:aws:s3:::my-bucket",
				Start: timestamppb.New(now.Add(-1 * time.Hour)),
				End:   timestamppb.New(now),
				Tags: map[string]string{
					"sku": "STANDARD",
				},
			},
			wantError: false,
			wantResource: &pbc.ResourceDescriptor{
				Provider:     "aws",
				ResourceType: "s3",
				Sku:          "STANDARD",
				Region:       "us-east-1", // Plugin's region used for global service
				Tags:         map[string]string{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, _, err := p.ValidateActualCostRequest(ctx, tt.req)
			if tt.wantError {
				require.Error(t, err)
				st, ok := status.FromError(err)
				require.True(t, ok)
				assert.Equal(t, tt.errCode, st.Code())

				// Check error details
				details := st.Details()
				require.NotEmpty(t, details)
				errDetail, ok := details[0].(*pbc.ErrorDetail)
				require.True(t, ok)
				assert.Equal(t, tt.pbcCode, errDetail.Code)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantResource.Provider, res.Provider)
				assert.Equal(t, tt.wantResource.ResourceType, res.ResourceType)
				assert.Equal(t, tt.wantResource.Sku, res.Sku)
				assert.Equal(t, tt.wantResource.Region, res.Region)
			}
		})
	}
}

// TestRegionFallbackGlobalServices tests that global services (S3, IAM) with empty regions
// are properly handled using the plugin's region for validation (issue: missing region fallback test coverage).
// This test verifies the defensive checks in ValidateActualCostRequest for ARN parsing.
func TestRegionFallbackGlobalServices(t *testing.T) {
	logger := zerolog.Nop()
	p := NewAWSPublicPlugin("us-east-1", nil, logger)
	ctx := context.Background()
	now := time.Now()

	tests := []struct {
		name      string
		req       *pbc.GetActualCostRequest
		wantError bool
		checkRegion bool
	}{
		{
			name: "ActualCost: S3 ARN (empty region) uses plugin region for validation",
			req: &pbc.GetActualCostRequest{
				Arn:   "arn:aws:s3:::my-bucket",
				Start: timestamppb.New(now.Add(-1 * time.Hour)),
				End:   timestamppb.New(now),
				Tags: map[string]string{
					"sku": "STANDARD",
				},
			},
			wantError:   false,
			checkRegion: true,
		},
		{
			name: "ActualCost: IAM ARN (empty region) uses plugin region for validation",
			req: &pbc.GetActualCostRequest{
				Arn:   "arn:aws:iam::123456789012:role/MyRole",
				Start: timestamppb.New(now.Add(-1 * time.Hour)),
				End:   timestamppb.New(now),
				Tags: map[string]string{
					"sku": "role",
				},
			},
			wantError:   false,
			checkRegion: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, _, err := p.ValidateActualCostRequest(ctx, tt.req)
			if tt.wantError {
				require.Error(t, err, "expected validation error")
			} else {
				require.NoError(t, err, "validation should pass for global services")
				assert.NotNil(t, res)
				if tt.checkRegion {
					assert.Equal(t, "us-east-1", res.Region, "global service should have plugin region assigned")
				}
			}
		})
	}
}
