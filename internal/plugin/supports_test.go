package plugin

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/rs/zerolog"
	pb "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

func TestSupports(t *testing.T) {
	mock := newMockPricingClient("us-east-1", "USD")
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	plugin := NewAWSPublicPlugin("us-east-1", "test-version", mock, logger)

	tests := []struct {
		name             string
		req              *pb.SupportsRequest
		wantSupported    bool
		wantReasonSubstr string // substring to check in reason
	}{
		// Fully supported resource types in correct region
		{
			name: "EC2 in correct region",
			req: &pb.SupportsRequest{
				Resource: &pb.ResourceDescriptor{
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
			req: &pb.SupportsRequest{
				Resource: &pb.ResourceDescriptor{
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
			req: &pb.SupportsRequest{
				Resource: &pb.ResourceDescriptor{
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
			req: &pb.SupportsRequest{
				Resource: &pb.ResourceDescriptor{
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
			req: &pb.SupportsRequest{
				Resource: &pb.ResourceDescriptor{
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
			req: &pb.SupportsRequest{
				Resource: &pb.ResourceDescriptor{
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
			req: &pb.SupportsRequest{
				Resource: &pb.ResourceDescriptor{
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
			req: &pb.SupportsRequest{
				Resource: &pb.ResourceDescriptor{
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
			req: &pb.SupportsRequest{
				Resource: &pb.ResourceDescriptor{
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
			req: &pb.SupportsRequest{
				Resource: &pb.ResourceDescriptor{
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
			req: &pb.SupportsRequest{
				Resource: &pb.ResourceDescriptor{
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
			req: &pb.SupportsRequest{
				Resource: &pb.ResourceDescriptor{
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
			req: &pb.SupportsRequest{
				Resource: &pb.ResourceDescriptor{
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
			req: &pb.SupportsRequest{
				Resource: &pb.ResourceDescriptor{
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
			req: &pb.SupportsRequest{
				Resource: &pb.ResourceDescriptor{
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
			req: &pb.SupportsRequest{
				Resource: &pb.ResourceDescriptor{
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
			req: &pb.SupportsRequest{
				Resource: &pb.ResourceDescriptor{
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
			req: &pb.SupportsRequest{
				Resource: &pb.ResourceDescriptor{
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
			req: &pb.SupportsRequest{
				Resource: &pb.ResourceDescriptor{
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
			req: &pb.SupportsRequest{
				Resource: &pb.ResourceDescriptor{
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
			req: &pb.SupportsRequest{
				Resource: &pb.ResourceDescriptor{
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
			req: &pb.SupportsRequest{
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

// TestSupports_ZeroCost_Resources tests that zero-cost resources are supported and have "zero-cost" type in logs.
// Includes empty region case to verify fallback behavior consistent with ValidateProjectedCostRequest.
func TestSupports_ZeroCost_Resources(t *testing.T) {
	mock := newMockPricingClient("us-east-1", "USD")
	var logBuf bytes.Buffer
	logger := zerolog.New(&logBuf).Level(zerolog.InfoLevel)
	plugin := NewAWSPublicPlugin("us-east-1", "test-version", mock, logger)

	tests := []struct {
		name         string
		resourceType string
		region       string
	}{
		{"vpc", "vpc", "us-east-1"},
		{"securitygroup", "securitygroup", "us-east-1"},
		{"subnet", "subnet", "us-east-1"},
		{"vpc empty region", "vpc", ""},
		{"securitygroup empty region", "securitygroup", ""},
		{"subnet empty region", "subnet", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logBuf.Reset()

			resp, err := plugin.Supports(context.Background(), &pb.SupportsRequest{
				Resource: &pb.ResourceDescriptor{
					Provider:     "aws",
					ResourceType: tt.resourceType,
					Region:       tt.region,
				},
			})
			if err != nil {
				t.Fatalf("Supports() returned error: %v", err)
			}
			if !resp.Supported {
				t.Errorf("%s should be supported", tt.name)
			}

			// Verify logs contain "zero-cost"
			if !strings.Contains(logBuf.String(), `"cost_type":"zero-cost"`) {
				t.Errorf("Logs should contain cost_type:zero-cost, got: %s", logBuf.String())
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

	req := &pb.SupportsRequest{
		Resource: &pb.ResourceDescriptor{
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
		req              *pb.SupportsRequest
		wantSupported    bool
		wantReasonSubstr string
	}{
		{
			name: "EC2 in ca-central-1",
			req: &pb.SupportsRequest{
				Resource: &pb.ResourceDescriptor{
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
			req: &pb.SupportsRequest{
				Resource: &pb.ResourceDescriptor{
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
			req: &pb.SupportsRequest{
				Resource: &pb.ResourceDescriptor{
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
			req: &pb.SupportsRequest{
				Resource: &pb.ResourceDescriptor{
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
			req: &pb.SupportsRequest{
				Resource: &pb.ResourceDescriptor{
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
		req              *pb.SupportsRequest
		wantSupported    bool
		wantReasonSubstr string
	}{
		{
			name: "EC2 in sa-east-1",
			req: &pb.SupportsRequest{
				Resource: &pb.ResourceDescriptor{
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
			req: &pb.SupportsRequest{
				Resource: &pb.ResourceDescriptor{
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
			req: &pb.SupportsRequest{
				Resource: &pb.ResourceDescriptor{
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
			req: &pb.SupportsRequest{
				Resource: &pb.ResourceDescriptor{
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
			req: &pb.SupportsRequest{
				Resource: &pb.ResourceDescriptor{
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

	resp, err := plugin.Supports(context.Background(), &pb.SupportsRequest{
		Resource: &pb.ResourceDescriptor{
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
		if m == pb.MetricKind_METRIC_KIND_CARBON_FOOTPRINT {
			foundCarbon = true
			break
		}
	}

	if !foundCarbon {
		t.Errorf("SupportedMetrics should contain METRIC_KIND_CARBON_FOOTPRINT, got %v", resp.SupportedMetrics)
	}
}

// TestSupports_DynamoDB_HasCarbonSupport tests that DynamoDB includes carbon in supported_metrics (T027)
// DynamoDB carbon estimation uses storage-based calculation with SSD × 3× replication factor.
func TestSupports_DynamoDB_HasCarbonSupport(t *testing.T) {
	mock := newMockPricingClient("us-east-1", "USD")
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	plugin := NewAWSPublicPlugin("us-east-1", "test-version", mock, logger)

	resp, err := plugin.Supports(context.Background(), &pb.SupportsRequest{
		Resource: &pb.ResourceDescriptor{
			Provider:     "aws",
			ResourceType: "dynamodb",
			Region:       "us-east-1",
		},
	})

	if err != nil {
		t.Fatalf("Supports() returned error: %v", err)
	}

	if !resp.Supported {
		t.Fatal("DynamoDB should be supported")
	}

	// DynamoDB has carbon estimation (storage-based, SSD × 3× replication)
	hasCarbon := false
	for _, m := range resp.SupportedMetrics {
		if m == pb.MetricKind_METRIC_KIND_CARBON_FOOTPRINT {
			hasCarbon = true
			break
		}
	}
	if !hasCarbon {
		t.Errorf("SupportedMetrics should include CARBON_FOOTPRINT for DynamoDB, got %v", resp.SupportedMetrics)
	}
}

// TestSupports_AllResourceTypes_SupportedMetrics tests supported_metrics for all resource types.
// All services with carbon estimation implemented should advertise METRIC_KIND_CARBON_FOOTPRINT.
func TestSupports_AllResourceTypes_SupportedMetrics(t *testing.T) {
	mock := newMockPricingClient("us-east-1", "USD")
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	plugin := NewAWSPublicPlugin("us-east-1", "test-version", mock, logger)

	tests := []struct {
		resourceType string
		wantCarbon   bool
	}{
		{"ec2", true},                       // EC2 has carbon (CPU/GPU power × utilization × grid factor)
		{"aws:ec2/instance:Instance", true}, // Pulumi format
		{"ebs", true},                       // EBS has carbon (storage energy × replication)
		{"eks", true},                       // EKS has carbon (control plane returns 0)
		{"rds", true},                       // RDS has carbon (compute + storage, Multi-AZ 2×)
		{"s3", true},                        // S3 has carbon (storage × replication by class)
		{"lambda", true},                    // Lambda has carbon (vCPU-equiv × duration)
		{"dynamodb", true},                  // DynamoDB has carbon (storage × SSD × 3× replication)
		{"elasticache", true},               // ElastiCache has carbon (EC2-equiv × nodes)
		{"elb", false},                      // ELB no carbon yet
		{"natgw", false},                    // NAT Gateway no carbon yet
		{"cloudwatch", false},               // CloudWatch no carbon yet
	}

	for _, tt := range tests {
		t.Run(tt.resourceType, func(t *testing.T) {
			resp, err := plugin.Supports(context.Background(), &pb.SupportsRequest{
				Resource: &pb.ResourceDescriptor{
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
				if m == pb.MetricKind_METRIC_KIND_CARBON_FOOTPRINT {
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
		req              *pb.SupportsRequest
		wantSupported    bool
		wantReasonSubstr string
	}{
		{
			name: "EC2 in ap-southeast-1",
			req: &pb.SupportsRequest{
				Resource: &pb.ResourceDescriptor{
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
			req: &pb.SupportsRequest{
				Resource: &pb.ResourceDescriptor{
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
			req: &pb.SupportsRequest{
				Resource: &pb.ResourceDescriptor{
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
			req: &pb.SupportsRequest{
				Resource: &pb.ResourceDescriptor{
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
			req: &pb.SupportsRequest{
				Resource: &pb.ResourceDescriptor{
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

// TestSupports_ZeroCost_MixedCase tests that zero-cost resources are handled case-insensitively.
// Resource types like "VPC", "SecurityGroup", and "Subnet" should be normalized to lowercase
// and correctly identified as zero-cost resources.
func TestSupports_ZeroCost_MixedCase(t *testing.T) {
	mock := newMockPricingClient("us-east-1", "USD")
	var logBuf bytes.Buffer
	logger := zerolog.New(&logBuf).Level(zerolog.InfoLevel)
	plugin := NewAWSPublicPlugin("us-east-1", "test-version", mock, logger)

	tests := []struct {
		name         string
		resourceType string
	}{
		{"VPC uppercase", "VPC"},
		{"Vpc titlecase", "Vpc"},
		{"SecurityGroup mixed", "SecurityGroup"},
		{"SECURITYGROUP uppercase", "SECURITYGROUP"},
		{"Subnet titlecase", "Subnet"},
		{"SUBNET uppercase", "SUBNET"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logBuf.Reset()
			resp, err := plugin.Supports(context.Background(), &pb.SupportsRequest{
				Resource: &pb.ResourceDescriptor{
					Provider:     "aws",
					ResourceType: tt.resourceType,
					Region:       "us-east-1",
				},
			})

			if err != nil {
				t.Fatalf("Supports() returned error: %v", err)
			}

			if !resp.Supported {
				t.Errorf("%s should be supported (case-insensitive)", tt.resourceType)
			}

			if !strings.Contains(logBuf.String(), `"cost_type":"zero-cost"`) {
				t.Errorf("Logs should contain cost_type:zero-cost for %s, got: %s", tt.resourceType, logBuf.String())
			}
		})
	}
}

// TestSupports_USWest1_UnsupportedResourceType tests FR-007 behavior.
// When a resource type is not supported in us-west-1, Supports() returns supported=false
// with a reason indicating the resource type is not supported.
func TestSupports_USWest1_UnsupportedResourceType(t *testing.T) {
	mock := newMockPricingClient("us-west-1", "USD")
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	plugin := NewAWSPublicPlugin("us-west-1", "test-version", mock, logger)

	tests := []struct {
		name         string
		resourceType string
		wantSupport  bool
		wantReason   string
	}{
		{
			name:         "cloudfront not implemented",
			resourceType: "cloudfront",
			wantSupport:  false,
			wantReason:   "not supported",
		},
		{
			name:         "route53 not implemented",
			resourceType: "route53",
			wantSupport:  false,
			wantReason:   "not supported",
		},
		{
			name:         "unknown aws service",
			resourceType: "aws:unknown/service:Service",
			wantSupport:  false,
			wantReason:   "not supported",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := plugin.Supports(context.Background(), &pb.SupportsRequest{
				Resource: &pb.ResourceDescriptor{
					Provider:     "aws",
					ResourceType: tt.resourceType,
					Region:       "us-west-1",
				},
			})

			if err != nil {
				t.Fatalf("Supports() returned unexpected error: %v", err)
			}

			if resp.Supported != tt.wantSupport {
				t.Errorf("Supported = %v, want %v", resp.Supported, tt.wantSupport)
			}

			if !strings.Contains(resp.Reason, tt.wantReason) {
				t.Errorf("Reason = %q, want substring %q", resp.Reason, tt.wantReason)
			}
		})
	}
}

// TestSupports_USWest1 tests support for us-west-1 (N. California) region binary.
// FR-001: System MUST support us-west-1 as a valid region identifier.
func TestSupports_USWest1(t *testing.T) {
	mock := newMockPricingClient("us-west-1", "USD")
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	plugin := NewAWSPublicPlugin("us-west-1", "test-version", mock, logger)

	tests := []struct {
		name             string
		req              *pb.SupportsRequest
		wantSupported    bool
		wantReasonSubstr string
	}{
		{
			name: "EC2 in us-west-1",
			req: &pb.SupportsRequest{
				Resource: &pb.ResourceDescriptor{
					Provider:     "aws",
					ResourceType: "ec2",
					Region:       "us-west-1",
				},
			},
			wantSupported:    true,
			wantReasonSubstr: "",
		},
		{
			name: "EBS in us-west-1",
			req: &pb.SupportsRequest{
				Resource: &pb.ResourceDescriptor{
					Provider:     "aws",
					ResourceType: "ebs",
					Region:       "us-west-1",
				},
			},
			wantSupported:    true,
			wantReasonSubstr: "",
		},
		{
			name: "RDS in us-west-1",
			req: &pb.SupportsRequest{
				Resource: &pb.ResourceDescriptor{
					Provider:     "aws",
					ResourceType: "rds",
					Region:       "us-west-1",
				},
			},
			wantSupported:    true,
			wantReasonSubstr: "",
		},
		{
			name: "S3 in us-west-1",
			req: &pb.SupportsRequest{
				Resource: &pb.ResourceDescriptor{
					Provider:     "aws",
					ResourceType: "s3",
					Region:       "us-west-1",
				},
			},
			wantSupported:    true,
			wantReasonSubstr: "",
		},
		{
			name: "Lambda in us-west-1",
			req: &pb.SupportsRequest{
				Resource: &pb.ResourceDescriptor{
					Provider:     "aws",
					ResourceType: "lambda",
					Region:       "us-west-1",
				},
			},
			wantSupported:    true,
			wantReasonSubstr: "",
		},
		{
			name: "EC2 in us-east-1 (wrong for us-west-1 binary)",
			req: &pb.SupportsRequest{
				Resource: &pb.ResourceDescriptor{
					Provider:     "aws",
					ResourceType: "ec2",
					Region:       "us-east-1",
				},
			},
			wantSupported:    false,
			wantReasonSubstr: "Region not supported",
		},
		{
			name: "EC2 in us-west-2 (different US west region)",
			req: &pb.SupportsRequest{
				Resource: &pb.ResourceDescriptor{
					Provider:     "aws",
					ResourceType: "ec2",
					Region:       "us-west-2",
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

// TestSupports_ZeroCost_SupportedMetricsNil verifies that zero-cost resources return
// nil for SupportedMetrics since they have no carbon footprint or other metrics.
func TestSupports_ZeroCost_SupportedMetricsNil(t *testing.T) {
	mock := newMockPricingClient("us-east-1", "USD")
	var logBuf bytes.Buffer
	logger := zerolog.New(&logBuf).Level(zerolog.InfoLevel)
	plugin := NewAWSPublicPlugin("us-east-1", "test-version", mock, logger)

	zeroCostTypes := []string{"vpc", "securitygroup", "subnet"}

	for _, rt := range zeroCostTypes {
		t.Run(rt, func(t *testing.T) {
			resp, err := plugin.Supports(context.Background(), &pb.SupportsRequest{
				Resource: &pb.ResourceDescriptor{
					Provider:     "aws",
					ResourceType: rt,
					Region:       "us-east-1",
				},
			})

			if err != nil {
				t.Fatalf("Supports() returned error: %v", err)
			}

			if resp.SupportedMetrics != nil {
				t.Errorf("SupportedMetrics should be nil for zero-cost resource %s, got: %v", rt, resp.SupportedMetrics)
			}
		})
	}
}

func TestSupports_IAM(t *testing.T) {
	mock := newMockPricingClient("us-east-1", "USD")
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	plugin := NewAWSPublicPlugin("us-east-1", "test-version", mock, logger)

	tests := []struct {
		name         string
		resourceType string
		want         bool
	}{
		{"IAM User", "aws:iam/user:User", true},
		{"IAM Role", "aws:iam/role:Role", true},
		{"IAM Canonical", "iam", true},
		{"IAM Mixed Case", "AWS:IAM/USER:USER", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := plugin.Supports(context.Background(), &pb.SupportsRequest{
				Resource: &pb.ResourceDescriptor{
					Provider:     "aws",
					ResourceType: tt.resourceType,
					Region:       "us-east-1",
				},
			})

			if err != nil {
				t.Fatalf("Supports() returned error: %v", err)
			}

			if resp.Supported != tt.want {
				t.Errorf("Supported = %v, want %v", resp.Supported, tt.want)
			}
		})
	}
}
