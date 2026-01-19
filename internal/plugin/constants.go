// Package plugin provides the CostSourceService gRPC implementation for AWS public pricing.
package plugin

// providerAWS is the constant identifier for the AWS provider.
const providerAWS = "aws"

// PricingNotFoundTemplate is the standard message template for missing pricing data.
// Use with fmt.Sprintf to format specific resource details.
//
// Example: fmt.Sprintf(PricingNotFoundTemplate, "EC2 instance type", "t3.micro")
// Result: "EC2 instance type \"t3.micro\" not found in pricing data"
const PricingNotFoundTemplate = "%s %q not found in pricing data"

// PricingUnavailableTemplate is the standard message template for region-level pricing unavailability.
// Use with fmt.Sprintf to format service and region.
//
// Example: fmt.Sprintf(PricingUnavailableTemplate, "CloudWatch", "ap-northeast-3")
// Result: "CloudWatch pricing data not available for region ap-northeast-3"
const PricingUnavailableTemplate = "%s pricing data not available for region %s"

// Hours per month for production and development modes
const (
	HoursPerMonthProd = 730 // Production: 24 hours/day * 30 days
	HoursPerMonthDev  = 160 // Development: 8 hours/day * 5 days/week * 4 weeks
)

// RelationshipAttachedTo represents a direct attachment relationship (EBS → EC2).
const RelationshipAttachedTo = "attached_to"

// RelationshipWithin represents a containment relationship (RDS → VPC).
const RelationshipWithin = "within"

// RelationshipManagedBy represents a management relationship (ELB → EC2).
const RelationshipManagedBy = "managed_by"

// ZeroCostServices is the canonical set of AWS resource types that have no direct charges.
// These resources (VPC, Security Groups, Subnets) are "free tier" networking infrastructure.
// Use IsZeroCostService() for membership checks.
//
// When adding new zero-cost resources:
// 1. Add the canonical service name here
// 2. Add the Pulumi pattern to ZeroCostPulumiPatterns below, OR implement dedicated prefix matching in normalizeResourceType()
var ZeroCostServices = map[string]bool{
	"vpc":           true,
	"securitygroup": true,
	"subnet":        true,
	"iam":           true,
}

// ZeroCostPulumiPatterns maps Pulumi resource type path segments to canonical service names.
// Used by normalizeResourceType() to detect zero-cost resources from Pulumi format.
// Example: "ec2/vpc" in "aws:ec2/vpc:Vpc" maps to "vpc"
var ZeroCostPulumiPatterns = map[string]string{
	"ec2/vpc":           "vpc",
	"ec2/securitygroup": "securitygroup",
	"ec2/subnet":        "subnet",
}

// IsZeroCostService returns true if the canonical service name has no direct AWS charges.
func IsZeroCostService(service string) bool {
	return ZeroCostServices[service]
}
