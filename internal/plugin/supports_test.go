package plugin

import (
	"context"
	"strings"
	"testing"

	pbc "github.com/rshade/pulumicost-spec/sdk/go/proto/pulumicost/v1"
)

func TestSupports(t *testing.T) {
	mock := newMockPricingClient("us-east-1", "USD")
	plugin := NewAWSPublicPlugin("us-east-1", mock)

	tests := []struct {
		name             string
		req              *pbc.SupportsRequest
		wantSupported    bool
		wantReasonSubstr string // substring to check in reason
	}{
		// Fully supported resource types in correct region
		{
			name: "EC2 in correct region",
			req: &pbc.SupportsRequest{
				Resource: &pbc.ResourceDescriptor{
					Provider:     "aws",
					ResourceType: "ec2",
					Region:       "us-east-1",
				},
			},
			wantSupported:    true,
			wantReasonSubstr: "",
		},
		{
			name: "EBS in correct region",
			req: &pbc.SupportsRequest{
				Resource: &pbc.ResourceDescriptor{
					Provider:     "aws",
					ResourceType: "ebs",
					Region:       "us-east-1",
				},
			},
			wantSupported:    true,
			wantReasonSubstr: "",
		},

		// Stub/limited support resource types
		{
			name: "S3 with limited support",
			req: &pbc.SupportsRequest{
				Resource: &pbc.ResourceDescriptor{
					Provider:     "aws",
					ResourceType: "s3",
					Region:       "us-east-1",
				},
			},
			wantSupported:    true,
			wantReasonSubstr: "Limited support",
		},
		{
			name: "Lambda with limited support",
			req: &pbc.SupportsRequest{
				Resource: &pbc.ResourceDescriptor{
					Provider:     "aws",
					ResourceType: "lambda",
					Region:       "us-east-1",
				},
			},
			wantSupported:    true,
			wantReasonSubstr: "Limited support",
		},
		{
			name: "RDS with limited support",
			req: &pbc.SupportsRequest{
				Resource: &pbc.ResourceDescriptor{
					Provider:     "aws",
					ResourceType: "rds",
					Region:       "us-east-1",
				},
			},
			wantSupported:    true,
			wantReasonSubstr: "Limited support",
		},
		{
			name: "DynamoDB with limited support",
			req: &pbc.SupportsRequest{
				Resource: &pbc.ResourceDescriptor{
					Provider:     "aws",
					ResourceType: "dynamodb",
					Region:       "us-east-1",
				},
			},
			wantSupported:    true,
			wantReasonSubstr: "Limited support",
		},

		// Wrong region
		{
			name: "EC2 in wrong region",
			req: &pbc.SupportsRequest{
				Resource: &pbc.ResourceDescriptor{
					Provider:     "aws",
					ResourceType: "ec2",
					Region:       "eu-west-1",
				},
			},
			wantSupported:    false,
			wantReasonSubstr: "Region not supported",
		},
		{
			name: "EBS in wrong region",
			req: &pbc.SupportsRequest{
				Resource: &pbc.ResourceDescriptor{
					Provider:     "aws",
					ResourceType: "ebs",
					Region:       "us-west-2",
				},
			},
			wantSupported:    false,
			wantReasonSubstr: "Region not supported",
		},

		// Wrong provider
		{
			name: "GCP provider",
			req: &pbc.SupportsRequest{
				Resource: &pbc.ResourceDescriptor{
					Provider:     "gcp",
					ResourceType: "compute",
					Region:       "us-east-1",
				},
			},
			wantSupported:    false,
			wantReasonSubstr: "Provider",
		},
		{
			name: "Azure provider",
			req: &pbc.SupportsRequest{
				Resource: &pbc.ResourceDescriptor{
					Provider:     "azure",
					ResourceType: "vm",
					Region:       "us-east-1",
				},
			},
			wantSupported:    false,
			wantReasonSubstr: "Provider",
		},

		// Unknown resource types (edge case coverage)
		{
			name: "Unknown resource type",
			req: &pbc.SupportsRequest{
				Resource: &pbc.ResourceDescriptor{
					Provider:     "aws",
					ResourceType: "unknown-service",
					Region:       "us-east-1",
				},
			},
			wantSupported:    false,
			wantReasonSubstr: "not supported",
		},
		{
			name: "CloudFront not implemented",
			req: &pbc.SupportsRequest{
				Resource: &pbc.ResourceDescriptor{
					Provider:     "aws",
					ResourceType: "cloudfront",
					Region:       "us-east-1",
				},
			},
			wantSupported:    false,
			wantReasonSubstr: "not supported",
		},

		// Invalid requests
		{
			name:             "Nil request",
			req:              nil,
			wantSupported:    false,
			wantReasonSubstr: "Invalid request",
		},
		{
			name: "Nil resource descriptor",
			req: &pbc.SupportsRequest{
				Resource: nil,
			},
			wantSupported:    false,
			wantReasonSubstr: "Invalid request",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := plugin.Supports(context.Background(), tt.req)
			if err != nil {
				t.Fatalf("Supports() returned error: %v", err)
			}

			// Supports() never returns nil response
			if resp.Supported != tt.wantSupported {
				t.Errorf("Supported = %v, want %v", resp.Supported, tt.wantSupported)
			}

			if tt.wantReasonSubstr != "" && !strings.Contains(resp.Reason, tt.wantReasonSubstr) {
				t.Errorf("Reason = %q, want substring %q", resp.Reason, tt.wantReasonSubstr)
			}

			if tt.wantReasonSubstr == "" && resp.Reason != "" {
				t.Errorf("Reason = %q, want empty string", resp.Reason)
			}
		})
	}
}
