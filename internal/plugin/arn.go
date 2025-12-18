package plugin

import (
	"fmt"
	"strings"
)

// ARNComponents represents the parsed components of an AWS ARN.
// ARN format: arn:partition:service:region:account-id:resource-type/resource-id
// See: https://docs.aws.amazon.com/general/latest/gr/aws-arns-and-namespaces.html
type ARNComponents struct {
	Partition    string // e.g., "aws", "aws-cn", "aws-us-gov"
	Service      string // e.g., "ec2", "s3", "rds", "lambda"
	Region       string // e.g., "us-east-1" (may be empty for global services like S3)
	AccountID    string // 12-digit AWS account ID
	ResourceType string // e.g., "instance", "volume", "bucket"
	ResourceID   string // e.g., "i-abc123", "vol-xyz789"
}

// ParseARN parses an AWS ARN string into its component parts.
// Returns an error if the ARN format is invalid.
//
// Supported ARN formats:
//   - arn:partition:service:region:account-id:resource-type/resource-id
//   - arn:partition:service:region:account-id:resource-type:resource-id
//   - arn:partition:s3:::bucket-name (S3 bucket - note empty region and account)
//
// Examples:
//   - arn:aws:ec2:us-east-1:123456789012:instance/i-abc123
//   - arn:aws:ec2:us-east-1:123456789012:volume/vol-xyz789
//   - arn:aws:s3:::my-bucket
//   - arn:aws:rds:us-west-2:123456789012:db:mydb
//   - arn:aws:lambda:eu-west-1:123456789012:function:my-function
func ParseARN(arnString string) (*ARNComponents, error) {
	if arnString == "" {
		return nil, fmt.Errorf("ARN is empty")
	}

	// Split by ":" - ARNs have at least 6 parts
	parts := strings.SplitN(arnString, ":", 6)
	if len(parts) < 6 {
		return nil, fmt.Errorf("invalid ARN format: expected at least 6 colon-separated parts, got %d", len(parts))
	}

	// Validate "arn" prefix
	if parts[0] != "arn" {
		return nil, fmt.Errorf("invalid ARN: must start with 'arn:', got %q", parts[0])
	}

	// Validate partition
	partition := parts[1]
	if partition == "" {
		return nil, fmt.Errorf("invalid ARN: partition is empty")
	}
	if !isValidPartition(partition) {
		// Provide specific guidance for isolated partitions
		if partition == "aws-iso" || partition == "aws-iso-b" {
			return nil, fmt.Errorf("unsupported ARN partition %q: isolated partitions (aws-iso, aws-iso-b) do not have public pricing data available", partition)
		}
		return nil, fmt.Errorf("invalid ARN partition: %q", partition)
	}

	// Validate service
	service := parts[2]
	if service == "" {
		return nil, fmt.Errorf("invalid ARN: service is empty")
	}

	// Region can be empty (e.g., S3 buckets, IAM)
	region := parts[3]

	// Account ID can be empty (e.g., S3 buckets)
	accountID := parts[4]

	// Resource part - may contain "/" or ":" as separator
	resourcePart := parts[5]
	if resourcePart == "" {
		return nil, fmt.Errorf("invalid ARN: resource part is empty")
	}

	// Parse resource type and ID
	// Try "/" separator first (more common), then ":" separator
	resourceType, resourceID := parseResourcePart(resourcePart)

	return &ARNComponents{
		Partition:    partition,
		Service:      service,
		Region:       region,
		AccountID:    accountID,
		ResourceType: resourceType,
		ResourceID:   resourceID,
	}, nil
}

// isValidPartition checks if the partition is a valid AWS partition.
// Only commercial partitions (aws, aws-cn, aws-us-gov) are supported because
// this plugin uses AWS public pricing data, which is not available for
// isolated partitions (aws-iso, aws-iso-b).
func isValidPartition(partition string) bool {
	switch partition {
	case "aws", "aws-cn", "aws-us-gov":
		return true
	default:
		return false
	}
}

// parseResourcePart extracts resource type and ID from the resource part of an ARN.
// Handles both "/" and ":" separators.
func parseResourcePart(resourcePart string) (resourceType, resourceID string) {
	// Try "/" separator first
	if idx := strings.Index(resourcePart, "/"); idx != -1 {
		return resourcePart[:idx], resourcePart[idx+1:]
	}

	// Try ":" separator
	if idx := strings.Index(resourcePart, ":"); idx != -1 {
		return resourcePart[:idx], resourcePart[idx+1:]
	}

	// No separator - the whole part is the resource (e.g., S3 bucket name)
	return resourcePart, ""
}

// ToPulumiResourceType maps the ARN service/resource type to a PulumiCost resource type.
// This handles the mapping differences between AWS ARN format and Pulumi resource types.
//
// Notable mappings:
//   - ec2:instance -> ec2
//   - ec2:volume   -> ebs (EBS volumes use "ec2" service in ARN)
//   - rds:db       -> rds
//   - s3:bucket    -> s3
//   - lambda:function -> lambda
//   - dynamodb:table -> dynamodb
//   - eks:cluster  -> eks
func (a *ARNComponents) ToPulumiResourceType() string {
	// Special case: EBS volumes are under ec2 service but should map to "ebs"
	if a.Service == "ec2" && a.ResourceType == "volume" {
		return "ebs"
	}

	// For most services, the service name is the resource type
	switch a.Service {
	case "ec2":
		return "ec2"
	case "rds":
		return "rds"
	case "s3":
		return "s3"
	case "lambda":
		return "lambda"
	case "dynamodb":
		return "dynamodb"
	case "eks":
		return "eks"
	default:
		// Return the service name as-is for unsupported services
		return a.Service
	}
}

// IsGlobalService returns true if the service is global (region may be empty in ARN).
func (a *ARNComponents) IsGlobalService() bool {
	return a.Service == "s3" || a.Service == "iam"
}
