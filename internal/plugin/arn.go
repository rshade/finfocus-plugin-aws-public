package plugin

import (
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws/arn"
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
	parsed, err := arn.Parse(arnString)
	if err != nil {
		return nil, fmt.Errorf("invalid ARN: %w", err)
	}

	// The SDK accepts empty service and resource; validate them for our use case.
	if parsed.Service == "" {
		return nil, errors.New("invalid ARN: service is empty")
	}
	if parsed.Resource == "" {
		return nil, errors.New("invalid ARN: resource part is empty")
	}

	// Validate partition - only commercial partitions are supported because
	// this plugin uses AWS public pricing data, which is not available for
	// isolated partitions (aws-iso, aws-iso-b).
	if !isValidPartition(parsed.Partition) {
		if parsed.Partition == "aws-iso" || parsed.Partition == "aws-iso-b" {
			return nil, fmt.Errorf(
				"unsupported ARN partition %q: isolated partitions (aws-iso, aws-iso-b) do not have public pricing data available",
				parsed.Partition,
			)
		}
		return nil, fmt.Errorf("invalid ARN partition: %q", parsed.Partition)
	}

	// Parse resource type and ID from the combined Resource field.
	// The SDK keeps everything after the 5th colon as a single Resource string,
	// but we need to split it into type and ID (e.g., "instance/i-abc123" → "instance", "i-abc123").
	resourceType, resourceID := parseResourcePart(parsed.Resource)

	return &ARNComponents{
		Partition:    parsed.Partition,
		Service:      parsed.Service,
		Region:       parsed.Region,
		AccountID:    parsed.AccountID,
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
func parseResourcePart(resourcePart string) (string, string) {
	idxColon := strings.Index(resourcePart, ":")
	idxSlash := strings.Index(resourcePart, "/")

	switch {
	case idxColon >= 0 && (idxSlash < 0 || idxColon < idxSlash):
		return resourcePart[:idxColon], resourcePart[idxColon+1:]
	case idxSlash >= 0:
		return resourcePart[:idxSlash], resourcePart[idxSlash+1:]
	}

	// No separator - the whole part is the resource (e.g., S3 bucket name)
	return resourcePart, ""
}

// ToPulumiResourceType maps the ARN service/resource type to a FinFocus resource type.
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
	// EC2 service has multiple sub-resource types that need distinct mapping
	if a.Service == serviceEC2 {
		switch a.ResourceType {
		case "volume":
			return serviceEBS
		case "vpc":
			return "vpc"
		case "subnet":
			return "subnet"
		case "security-group":
			return "securitygroup"
		case "launch-template":
			return serviceLaunchTmpl
		default:
			return serviceEC2
		}
	}

	// For most services, the service name is the resource type
	switch a.Service {
	case serviceRDS:
		return serviceRDS
	case serviceS3:
		return serviceS3
	case serviceLambda:
		return serviceLambda
	case serviceDynamoDB:
		return serviceDynamoDB
	case serviceEKS:
		return serviceEKS
	case serviceIAM:
		return serviceIAM
	case "autoscaling":
		switch a.ResourceType {
		case "launchConfiguration", "launch-configuration":
			return serviceLaunchConfig
		case "autoScalingGroup", "auto-scaling-group":
			return serviceASG
		default:
			return a.Service
		}
	default:
		// Return the service name as-is for unsupported services
		return a.Service
	}
}

// IsGlobalService returns true if the service is global (region may be empty in ARN).
func (a *ARNComponents) IsGlobalService() bool {
	return a.Service == serviceS3 || a.Service == serviceIAM
}
