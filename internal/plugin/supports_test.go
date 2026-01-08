package plugin

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/rs/zerolog"
	pbc "github.com/rshade/pulumicost-spec/sdk/go/proto/pulumicost/v1"
)

func TestSupports(t *testing.T) {
	mock := newMockPricingClient("us-east-1", "USD")
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	plugin := NewAWSPublicPlugin("us-east-1", "test-version", mock, logger)

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
			name: "S3 fully supported",
			req: &pbc.SupportsRequest{
				Resource: &pbc.ResourceDescriptor{
					Provider:     "aws",
					ResourceType: "s3",
					Region:       "us-east-1",
				},
			},
			wantSupported:    true,
			wantReasonSubstr: "",
		},
		{
			name: "S3 global fallback (empty region)",
			req: &pbc.SupportsRequest{
				Resource: &pbc.ResourceDescriptor{
					Provider:     "aws",
					ResourceType: "s3",
					Region:       "", // Empty region should fallback to plugin region
				},
			},
			wantSupported:    true,
			wantReasonSubstr: "",
		},
		{
			name: "Lambda fully supported",
			req: &pbc.SupportsRequest{
				Resource: &pbc.ResourceDescriptor{
					Provider:     "aws",
					ResourceType: "lambda",
					Region:       "us-east-1",
				},
			},
			wantSupported:    true,
			wantReasonSubstr: "",
		},
		{
			name: "RDS fully supported",
			req: &pbc.SupportsRequest{
				Resource: &pbc.ResourceDescriptor{
					Provider:     "aws",
					ResourceType: "rds",
					Region:       "us-east-1",
				},
			},
			wantSupported:    true,
			wantReasonSubstr: "",
		},
		{
			name: "DynamoDB fully supported",
			req: &pbc.SupportsRequest{
				Resource: &pbc.ResourceDescriptor{
					Provider:     "aws",
					ResourceType: "dynamodb",
					Region:       "us-east-1",
				},
			},
			wantSupported:    true,
			wantReasonSubstr: "",
		},
		// ElastiCache support (T050)
		{
			name: "ElastiCache supported",
			req: &pbc.SupportsRequest{
				Resource: &pbc.ResourceDescriptor{
					Provider:     "aws",
					ResourceType: "elasticache",
					Region:       "us-east-1",
				},
			},
			wantSupported:    true,
			wantReasonSubstr: "",
		},
		// ElastiCache Pulumi Cluster format (T051)
		{
			name: "ElastiCache Pulumi Cluster format",
			req: &pbc.SupportsRequest{
				Resource: &pbc.ResourceDescriptor{
					Provider:     "aws",
					ResourceType: "aws:elasticache/cluster:Cluster",
					Region:       "us-east-1",
				},
			},
			wantSupported:    true,
			wantReasonSubstr: "",
		},
		// ElastiCache Pulumi ReplicationGroup format (T052)
		{
			name: "ElastiCache Pulumi ReplicationGroup format",
			req: &pbc.SupportsRequest{
				Resource: &pbc.ResourceDescriptor{
					Provider:     "aws",
					ResourceType: "aws:elasticache/replicationGroup:ReplicationGroup",
					Region:       "us-east-1",
				},
			},
			wantSupported:    true,
			wantReasonSubstr: "",
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
		{
			name: "EC2 in ap-southeast-1 (wrong for us-east-1 binary)",
			req: &pbc.SupportsRequest{
				Resource: &pbc.ResourceDescriptor{
					Provider:     "aws",
					ResourceType: "ec2",
					Region:       "ap-southeast-1",
				},
			},
			wantSupported:    false,
			wantReasonSubstr: "Region not supported",
		},
		{
			name: "EBS in ap-southeast-1 (wrong for us-east-1 binary)",
			req: &pbc.SupportsRequest{
				Resource: &pbc.ResourceDescriptor{
					Provider:     "aws",
					ResourceType: "ebs",
					Region:       "ap-southeast-1",
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

		// Pulumi resource type format support
		{
			name: "EC2 with Pulumi format",
			req: &pbc.SupportsRequest{
				Resource: &pbc.ResourceDescriptor{
					Provider:     "aws",
					ResourceType: "aws:ec2/instance:Instance",
					Region:       "us-east-1",
				},
			},
			wantSupported:    true,
			wantReasonSubstr: "",
		},
		{
			name: "EBS with Pulumi format",
			req: &pbc.SupportsRequest{
				Resource: &pbc.ResourceDescriptor{
					Provider:     "aws",
					ResourceType: "aws:ebs/volume:Volume",
					Region:       "us-east-1",
				},
			},
			wantSupported:    true,
			wantReasonSubstr: "",
		},
		{
			name: "S3 with Pulumi format",
			req: &pbc.SupportsRequest{
				Resource: &pbc.ResourceDescriptor{
					Provider:     "aws",
					ResourceType: "aws:s3/bucket:Bucket",
					Region:       "us-east-1",
				},
			},
			wantSupported:    true,
			wantReasonSubstr: "",
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

// T028: Test Supports logs contain required structured fields
func TestSupportsLogsContainRequiredFields(t *testing.T) {
	var logBuf bytes.Buffer
	mock := newMockPricingClient("us-east-1", "USD")
	logger := zerolog.New(&logBuf).Level(zerolog.InfoLevel)
	plugin := NewAWSPublicPlugin("us-east-1", "test-version", mock, logger)

	req := &pbc.SupportsRequest{
		Resource: &pbc.ResourceDescriptor{
			Provider:     "aws",
			ResourceType: "ec2",
			Region:       "us-east-1",
		},
	}

	_, err := plugin.Supports(context.Background(), req)
	if err != nil {
		t.Fatalf("Supports() error: %v", err)
	}

	// Parse log output and verify required fields
	var logEntry map[string]interface{}
	if err := json.Unmarshal(logBuf.Bytes(), &logEntry); err != nil {
		t.Fatalf("Failed to parse log output as JSON: %v", err)
	}

	// Required fields per data-model.md and tasks.md T024
	requiredFields := []string{
		"trace_id",
		"operation",
		"resource_type",
		"aws_region",
		"supported",
		"duration_ms",
		"message",
	}

	for _, field := range requiredFields {
		if _, ok := logEntry[field]; !ok {
			t.Errorf("Supports log missing required field: %s", field)
		}
	}

	// Verify specific values
	if op, ok := logEntry["operation"].(string); ok {
		if op != "Supports" {
			t.Errorf("operation = %q, want %q", op, "Supports")
		}
	}

	if rt, ok := logEntry["resource_type"].(string); ok {
		if rt != "ec2" {
			t.Errorf("resource_type = %q, want %q", rt, "ec2")
		}
	}

	if region, ok := logEntry["aws_region"].(string); ok {
		if region != "us-east-1" {
			t.Errorf("aws_region = %q, want %q", region, "us-east-1")
		}
	}

	// supported should be true for this request
	if supported, ok := logEntry["supported"].(bool); ok {
		if !supported {
			t.Errorf("supported = %v, want true", supported)
		}
	}

	// duration_ms should be non-negative
	if durationMs, ok := logEntry["duration_ms"].(float64); ok {
		if durationMs < 0 {
			t.Errorf("duration_ms = %v, should be non-negative", durationMs)
		}
	}
}

// TestSupports_CACentral1 tests support for ca-central-1 region binary
func TestSupports_CACentral1(t *testing.T) {
	mock := newMockPricingClient("ca-central-1", "USD")
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	plugin := NewAWSPublicPlugin("ca-central-1", "test-version", mock, logger)

	tests := []struct {
		name             string
		req              *pbc.SupportsRequest
		wantSupported    bool
		wantReasonSubstr string
	}{
		{
			name: "EC2 in ca-central-1",
			req: &pbc.SupportsRequest{
				Resource: &pbc.ResourceDescriptor{
					Provider:     "aws",
					ResourceType: "ec2",
					Region:       "ca-central-1",
				},
			},
			wantSupported:    true,
			wantReasonSubstr: "",
		},
		{
			name: "EBS in ca-central-1",
			req: &pbc.SupportsRequest{
				Resource: &pbc.ResourceDescriptor{
					Provider:     "aws",
					ResourceType: "ebs",
					Region:       "ca-central-1",
				},
			},
			wantSupported:    true,
			wantReasonSubstr: "",
		},
		{
			name: "S3 in ca-central-1 (fully supported)",
			req: &pbc.SupportsRequest{
				Resource: &pbc.ResourceDescriptor{
					Provider:     "aws",
					ResourceType: "s3",
					Region:       "ca-central-1",
				},
			},
			wantSupported:    true,
			wantReasonSubstr: "",
		},
		{
			name: "EC2 in us-east-1 (wrong for ca-central-1 binary)",
			req: &pbc.SupportsRequest{
				Resource: &pbc.ResourceDescriptor{
					Provider:     "aws",
					ResourceType: "ec2",
					Region:       "us-east-1",
				},
			},
			wantSupported:    false,
			wantReasonSubstr: "Region not supported",
		},
		{
			name: "EC2 in sa-east-1 (different region)",
			req: &pbc.SupportsRequest{
				Resource: &pbc.ResourceDescriptor{
					Provider:     "aws",
					ResourceType: "ec2",
					Region:       "sa-east-1",
				},
			},
			wantSupported:    false,
			wantReasonSubstr: "Region not supported",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := plugin.Supports(context.Background(), tt.req)
			if err != nil {
				t.Fatalf("Supports() returned error: %v", err)
			}

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

// TestSupports_SAEast1 tests support for sa-east-1 region binary
func TestSupports_SAEast1(t *testing.T) {
	mock := newMockPricingClient("sa-east-1", "USD")
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	plugin := NewAWSPublicPlugin("sa-east-1", "test-version", mock, logger)

	tests := []struct {
		name             string
		req              *pbc.SupportsRequest
		wantSupported    bool
		wantReasonSubstr string
	}{
		{
			name: "EC2 in sa-east-1",
			req: &pbc.SupportsRequest{
				Resource: &pbc.ResourceDescriptor{
					Provider:     "aws",
					ResourceType: "ec2",
					Region:       "sa-east-1",
				},
			},
			wantSupported:    true,
			wantReasonSubstr: "",
		},
		{
			name: "EBS in sa-east-1",
			req: &pbc.SupportsRequest{
				Resource: &pbc.ResourceDescriptor{
					Provider:     "aws",
					ResourceType: "ebs",
					Region:       "sa-east-1",
				},
			},
			wantSupported:    true,
			wantReasonSubstr: "",
		},
		{
			name: "Lambda in sa-east-1 (fully supported)",
			req: &pbc.SupportsRequest{
				Resource: &pbc.ResourceDescriptor{
					Provider:     "aws",
					ResourceType: "lambda",
					Region:       "sa-east-1",
				},
			},
			wantSupported:    true,
			wantReasonSubstr: "",
		},
		{
			name: "EC2 in eu-west-1 (wrong for sa-east-1 binary)",
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
			name: "EC2 in ca-central-1 (different region)",
			req: &pbc.SupportsRequest{
				Resource: &pbc.ResourceDescriptor{
					Provider:     "aws",
					ResourceType: "ec2",
					Region:       "ca-central-1",
				},
			},
			wantSupported:    false,
			wantReasonSubstr: "Region not supported",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := plugin.Supports(context.Background(), tt.req)
			if err != nil {
				t.Fatalf("Supports() returned error: %v", err)
			}

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

// ============================================================================
// Carbon Estimation Supports Tests (T026-T027)
// ============================================================================

// TestSupports_EC2_SupportedMetrics tests that EC2 returns supported_metrics with carbon footprint (T026)
func TestSupports_EC2_SupportedMetrics(t *testing.T) {
	mock := newMockPricingClient("us-east-1", "USD")
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	plugin := NewAWSPublicPlugin("us-east-1", "test-version", mock, logger)

	resp, err := plugin.Supports(context.Background(), &pbc.SupportsRequest{
		Resource: &pbc.ResourceDescriptor{
			Provider:     "aws",
			ResourceType: "ec2",
			Region:       "us-east-1",
		},
	})

	if err != nil {
		t.Fatalf("Supports() returned error: %v", err)
	}

	if !resp.Supported {
		t.Fatal("EC2 should be supported")
	}

	// Verify SupportedMetrics contains METRIC_KIND_CARBON_FOOTPRINT
	if len(resp.SupportedMetrics) == 0 {
		t.Fatal("SupportedMetrics should not be empty for EC2")
	}

	foundCarbon := false
	for _, m := range resp.SupportedMetrics {
		if m == pbc.MetricKind_METRIC_KIND_CARBON_FOOTPRINT {
			foundCarbon = true
			break
		}
	}

	if !foundCarbon {
		t.Errorf("SupportedMetrics should contain METRIC_KIND_CARBON_FOOTPRINT, got %v", resp.SupportedMetrics)
	}
}

// TestSupports_DynamoDB_NoSupportedMetrics tests that DynamoDB doesn't include carbon in supported_metrics (T027)
func TestSupports_DynamoDB_NoSupportedMetrics(t *testing.T) {
	mock := newMockPricingClient("us-east-1", "USD")
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	plugin := NewAWSPublicPlugin("us-east-1", "test-version", mock, logger)

	resp, err := plugin.Supports(context.Background(), &pbc.SupportsRequest{
		Resource: &pbc.ResourceDescriptor{
			Provider:     "aws",
			ResourceType: "dynamodb",
			Region:       "us-east-1",
		},
	})

	if err != nil {
		t.Fatalf("Supports() returned error: %v", err)
	}

	// DynamoDB is stub-supported but should NOT have carbon metrics
	if !resp.Supported {
		t.Fatal("DynamoDB should be supported (with limited support)")
	}

	// SupportedMetrics should be empty for DynamoDB (no carbon estimation)
	if len(resp.SupportedMetrics) > 0 {
		t.Errorf("SupportedMetrics should be empty for DynamoDB, got %v", resp.SupportedMetrics)
	}
}

// TestSupports_AllResourceTypes_SupportedMetrics tests supported_metrics for all resource types
func TestSupports_AllResourceTypes_SupportedMetrics(t *testing.T) {
	mock := newMockPricingClient("us-east-1", "USD")
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	plugin := NewAWSPublicPlugin("us-east-1", "test-version", mock, logger)

	tests := []struct {
		resourceType string
		wantCarbon   bool
	}{
		{"ec2", true},                       // EC2 has carbon
		{"aws:ec2/instance:Instance", true}, // Pulumi format
		{"ebs", false},                      // EBS no carbon (v1)
		{"eks", false},                      // EKS no carbon (v1)
		{"rds", false},                      // RDS no carbon (v1)
		{"s3", false},                       // S3 no carbon (v1)
		{"lambda", false},                   // Lambda no carbon (v1)
		{"dynamodb", false},                 // DynamoDB no carbon (stub)
		{"elasticache", false},              // ElastiCache no carbon (v1)
	}

	for _, tt := range tests {
		t.Run(tt.resourceType, func(t *testing.T) {
			resp, err := plugin.Supports(context.Background(), &pbc.SupportsRequest{
				Resource: &pbc.ResourceDescriptor{
					Provider:     "aws",
					ResourceType: tt.resourceType,
					Region:       "us-east-1",
				},
			})

			if err != nil {
				t.Fatalf("Supports() returned error: %v", err)
			}

			hasCarbon := false
			for _, m := range resp.SupportedMetrics {
				if m == pbc.MetricKind_METRIC_KIND_CARBON_FOOTPRINT {
					hasCarbon = true
					break
				}
			}

			if hasCarbon != tt.wantCarbon {
				t.Errorf("carbon in SupportedMetrics = %v, want %v", hasCarbon, tt.wantCarbon)
			}
		})
	}
}

// TestSupports_APSoutheast1 tests support for ap-southeast-1 region binary (T010)
func TestSupports_APSoutheast1(t *testing.T) {
	mock := newMockPricingClient("ap-southeast-1", "USD")
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	plugin := NewAWSPublicPlugin("ap-southeast-1", "test-version", mock, logger)

	tests := []struct {
		name             string
		req              *pbc.SupportsRequest
		wantSupported    bool
		wantReasonSubstr string
	}{
		{
			name: "EC2 in ap-southeast-1",
			req: &pbc.SupportsRequest{
				Resource: &pbc.ResourceDescriptor{
					Provider:     "aws",
					ResourceType: "ec2",
					Region:       "ap-southeast-1",
				},
			},
			wantSupported:    true,
			wantReasonSubstr: "",
		},
		{
			name: "EBS in ap-southeast-1",
			req: &pbc.SupportsRequest{
				Resource: &pbc.ResourceDescriptor{
					Provider:     "aws",
					ResourceType: "ebs",
					Region:       "ap-southeast-1",
				},
			},
			wantSupported:    true,
			wantReasonSubstr: "",
		},
		{
			name: "S3 in ap-southeast-1 (fully supported)",
			req: &pbc.SupportsRequest{
				Resource: &pbc.ResourceDescriptor{
					Provider:     "aws",
					ResourceType: "s3",
					Region:       "ap-southeast-1",
				},
			},
			wantSupported:    true,
			wantReasonSubstr: "",
		},
		{
			name: "EC2 in us-east-1 (wrong for ap-southeast-1 binary)",
			req: &pbc.SupportsRequest{
				Resource: &pbc.ResourceDescriptor{
					Provider:     "aws",
					ResourceType: "ec2",
					Region:       "us-east-1",
				},
			},
			wantSupported:    false,
			wantReasonSubstr: "Region not supported",
		},
		{
			name: "EC2 in ap-southeast-2 (different AP region)",
			req: &pbc.SupportsRequest{
				Resource: &pbc.ResourceDescriptor{
					Provider:     "aws",
					ResourceType: "ec2",
					Region:       "ap-southeast-2",
				},
			},
			wantSupported:    false,
			wantReasonSubstr: "Region not supported",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := plugin.Supports(context.Background(), tt.req)
			if err != nil {
				t.Fatalf("Supports() returned error: %v", err)
			}

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
