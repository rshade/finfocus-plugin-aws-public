package plugin

import (
	"strings"
	"testing"
)

// TestParseARN tests ARN parsing for all supported AWS services (T065).
func TestParseARN(t *testing.T) {
	tests := []struct {
		name     string
		arn      string
		expected *ARNComponents
		wantErr  bool
	}{
		{
			name: "EC2 instance",
			arn:  "arn:aws:ec2:us-east-1:123456789012:instance/i-abc123",
			expected: &ARNComponents{
				Partition:    "aws",
				Service:      "ec2",
				Region:       "us-east-1",
				AccountID:    "123456789012",
				ResourceType: "instance",
				ResourceID:   "i-abc123",
			},
		},
		{
			name: "EBS volume",
			arn:  "arn:aws:ec2:us-west-2:123456789012:volume/vol-xyz789",
			expected: &ARNComponents{
				Partition:    "aws",
				Service:      "ec2",
				Region:       "us-west-2",
				AccountID:    "123456789012",
				ResourceType: "volume",
				ResourceID:   "vol-xyz789",
			},
		},
		{
			name: "RDS instance with colon separator",
			arn:  "arn:aws:rds:eu-west-1:123456789012:db:mydb",
			expected: &ARNComponents{
				Partition:    "aws",
				Service:      "rds",
				Region:       "eu-west-1",
				AccountID:    "123456789012",
				ResourceType: "db",
				ResourceID:   "mydb",
			},
		},
		{
			name: "S3 bucket (empty region and account)",
			arn:  "arn:aws:s3:::my-bucket",
			expected: &ARNComponents{
				Partition:    "aws",
				Service:      "s3",
				Region:       "",
				AccountID:    "",
				ResourceType: "my-bucket",
				ResourceID:   "",
			},
		},
		{
			name: "Lambda function",
			arn:  "arn:aws:lambda:ap-northeast-1:123456789012:function:my-function",
			expected: &ARNComponents{
				Partition:    "aws",
				Service:      "lambda",
				Region:       "ap-northeast-1",
				AccountID:    "123456789012",
				ResourceType: "function",
				ResourceID:   "my-function",
			},
		},
		{
			name: "DynamoDB table",
			arn:  "arn:aws:dynamodb:us-east-1:123456789012:table/my-table",
			expected: &ARNComponents{
				Partition:    "aws",
				Service:      "dynamodb",
				Region:       "us-east-1",
				AccountID:    "123456789012",
				ResourceType: "table",
				ResourceID:   "my-table",
			},
		},
		{
			name: "EKS cluster",
			arn:  "arn:aws:eks:us-east-1:123456789012:cluster/my-cluster",
			expected: &ARNComponents{
				Partition:    "aws",
				Service:      "eks",
				Region:       "us-east-1",
				AccountID:    "123456789012",
				ResourceType: "cluster",
				ResourceID:   "my-cluster",
			},
		},
		{
			name: "China partition",
			arn:  "arn:aws-cn:ec2:cn-north-1:123456789012:instance/i-abc123",
			expected: &ARNComponents{
				Partition:    "aws-cn",
				Service:      "ec2",
				Region:       "cn-north-1",
				AccountID:    "123456789012",
				ResourceType: "instance",
				ResourceID:   "i-abc123",
			},
		},
		{
			name: "GovCloud partition",
			arn:  "arn:aws-us-gov:ec2:us-gov-west-1:123456789012:instance/i-abc123",
			expected: &ARNComponents{
				Partition:    "aws-us-gov",
				Service:      "ec2",
				Region:       "us-gov-west-1",
				AccountID:    "123456789012",
				ResourceType: "instance",
				ResourceID:   "i-abc123",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseARN(tt.arn)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseARN(%q) expected error, got nil", tt.arn)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseARN(%q) unexpected error: %v", tt.arn, err)
			}

			if result.Partition != tt.expected.Partition {
				t.Errorf("Partition = %q, want %q", result.Partition, tt.expected.Partition)
			}
			if result.Service != tt.expected.Service {
				t.Errorf("Service = %q, want %q", result.Service, tt.expected.Service)
			}
			if result.Region != tt.expected.Region {
				t.Errorf("Region = %q, want %q", result.Region, tt.expected.Region)
			}
			if result.AccountID != tt.expected.AccountID {
				t.Errorf("AccountID = %q, want %q", result.AccountID, tt.expected.AccountID)
			}
			if result.ResourceType != tt.expected.ResourceType {
				t.Errorf("ResourceType = %q, want %q", result.ResourceType, tt.expected.ResourceType)
			}
			if result.ResourceID != tt.expected.ResourceID {
				t.Errorf("ResourceID = %q, want %q", result.ResourceID, tt.expected.ResourceID)
			}
		})
	}
}

// TestParseARN_InvalidFormats tests error handling for invalid ARN formats (T066).
func TestParseARN_InvalidFormats(t *testing.T) {
	tests := []struct {
		name string
		arn  string
	}{
		{"empty string", ""},
		{"not an ARN", "not-an-arn"},
		{"missing prefix", "aws:ec2:us-east-1:123456789012:instance/i-abc"},
		{"wrong prefix", "urn:aws:ec2:us-east-1:123456789012:instance/i-abc"},
		{"too few parts", "arn:aws:ec2"},
		{"invalid partition", "arn:invalid:ec2:us-east-1:123456789012:instance/i-abc"},
		{"empty service", "arn:aws::us-east-1:123456789012:instance/i-abc"},
		{"empty resource", "arn:aws:ec2:us-east-1:123456789012:"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseARN(tt.arn)
			if err == nil {
				t.Errorf("ParseARN(%q) expected error, got nil", tt.arn)
			}
		})
	}
}

// TestARNComponents_ToPulumiResourceType tests resource type mapping (T065).
func TestARNComponents_ToPulumiResourceType(t *testing.T) {
	tests := []struct {
		name     string
		arn      *ARNComponents
		expected string
	}{
		{
			name: "EC2 instance maps to ec2",
			arn: &ARNComponents{
				Service:      "ec2",
				ResourceType: "instance",
			},
			expected: "ec2",
		},
		{
			name: "EBS volume maps to ebs (not ec2)",
			arn: &ARNComponents{
				Service:      "ec2",
				ResourceType: "volume",
			},
			expected: "ebs",
		},
		{
			name: "RDS maps to rds",
			arn: &ARNComponents{
				Service:      "rds",
				ResourceType: "db",
			},
			expected: "rds",
		},
		{
			name: "S3 maps to s3",
			arn: &ARNComponents{
				Service:      "s3",
				ResourceType: "bucket",
			},
			expected: "s3",
		},
		{
			name: "Lambda maps to lambda",
			arn: &ARNComponents{
				Service:      "lambda",
				ResourceType: "function",
			},
			expected: "lambda",
		},
		{
			name: "DynamoDB maps to dynamodb",
			arn: &ARNComponents{
				Service:      "dynamodb",
				ResourceType: "table",
			},
			expected: "dynamodb",
		},
		{
			name: "EKS maps to eks",
			arn: &ARNComponents{
				Service:      "eks",
				ResourceType: "cluster",
			},
			expected: "eks",
		},
		{
			name: "Unknown service returns service name",
			arn: &ARNComponents{
				Service:      "kinesis",
				ResourceType: "stream",
			},
			expected: "kinesis",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.arn.ToPulumiResourceType()
			if result != tt.expected {
				t.Errorf("ToPulumiResourceType() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestARNComponents_IsGlobalService tests S3 global service handling (T067).
func TestARNComponents_IsGlobalService(t *testing.T) {
	tests := []struct {
		name     string
		service  string
		expected bool
	}{
		{"S3 is global", "s3", true},
		{"IAM is global", "iam", true},
		{"EC2 is not global", "ec2", false},
		{"RDS is not global", "rds", false},
		{"Lambda is not global", "lambda", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			arn := &ARNComponents{Service: tt.service}
			result := arn.IsGlobalService()
			if result != tt.expected {
				t.Errorf("IsGlobalService() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestParseARN_IsolatedPartitions tests that isolated partitions (aws-iso, aws-iso-b)
// return specific error messages explaining why they are unsupported.
func TestParseARN_IsolatedPartitions(t *testing.T) {
	tests := []struct {
		name           string
		arn            string
		wantErrContain string
	}{
		{
			name:           "aws-iso partition (C2S)",
			arn:            "arn:aws-iso:ec2:us-iso-east-1:123456789012:instance/i-abc123",
			wantErrContain: "isolated partitions (aws-iso, aws-iso-b) do not have public pricing data",
		},
		{
			name:           "aws-iso-b partition (SC2S)",
			arn:            "arn:aws-iso-b:ec2:us-isob-east-1:123456789012:instance/i-abc123",
			wantErrContain: "isolated partitions (aws-iso, aws-iso-b) do not have public pricing data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseARN(tt.arn)
			if err == nil {
				t.Fatalf("ParseARN(%q) expected error, got nil", tt.arn)
			}
			if !strings.Contains(err.Error(), tt.wantErrContain) {
				t.Errorf("ParseARN(%q) error = %q, want error containing %q", tt.arn, err.Error(), tt.wantErrContain)
			}
		})
	}
}

// TestParseARN_S3EmptyRegion tests S3 bucket ARN handling with empty region (T067).
func TestParseARN_S3EmptyRegion(t *testing.T) {
	// S3 buckets have empty region and account in ARN
	arn, err := ParseARN("arn:aws:s3:::my-bucket-name")
	if err != nil {
		t.Fatalf("ParseARN() unexpected error: %v", err)
	}

	if arn.Region != "" {
		t.Errorf("Region = %q, want empty string for S3", arn.Region)
	}
	if arn.AccountID != "" {
		t.Errorf("AccountID = %q, want empty string for S3", arn.AccountID)
	}
	if !arn.IsGlobalService() {
		t.Error("S3 should be identified as global service")
	}
	if arn.ToPulumiResourceType() != "s3" {
		t.Errorf("ToPulumiResourceType() = %q, want %q", arn.ToPulumiResourceType(), "s3")
	}
}
