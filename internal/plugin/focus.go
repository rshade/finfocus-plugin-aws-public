package plugin

import (
	"fmt"
	"time"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// AWS service name mappings for FOCUS ServiceName field.
// These follow AWS's official service naming conventions.
var awsServiceNames = map[string]string{
	serviceEC2:        "Amazon EC2",
	serviceEBS:        "Amazon EBS",
	serviceS3:         "Amazon S3",
	serviceRDS:        "Amazon RDS",
	serviceLambda:     "AWS Lambda",
	serviceDynamoDB:   "Amazon DynamoDB",
	serviceEKS:        "Amazon EKS",
	serviceELB:        "Elastic Load Balancing",
	serviceNATGW:      "Amazon VPC NAT Gateway",
	serviceCloudWatch: "Amazon CloudWatch",
}

// buildFocusRecord creates a FocusCostRecord for public pricing estimates.
//
// For public pricing fallback estimates:
//   - BilledCost = EffectiveCost = ListCost (no discounts applied)
//   - PricingCategory is always STANDARD (on-demand pricing)
//   - ChargeCategory is USAGE (consumption-based charges)
//
// This function populates the essential FOCUS 1.2 fields that the plugin can
// reliably determine. Fields requiring billing account data (BillingAccountId,
// InvoiceId, etc.) are intentionally left empty.
//
// Parameters:
//   - serviceType: Normalized service type (e.g., "ec2", "ebs", "rds")
//   - resourceType: Original resource type from request (preserved for accuracy)
//   - region: AWS region identifier (e.g., "us-east-1")
//   - cost: The calculated cost amount
//   - unitPrice: The unit price used for calculation
//   - pricingUnit: The pricing unit (e.g., "Hours", "GB-Mo")
//   - start: Charge period start timestamp
//   - end: Charge period end timestamp
//   - sku: The SKU identifier (e.g., instance type "t3.micro")
func buildFocusRecord(
	serviceType, resourceType, region string,
	cost, unitPrice float64,
	pricingUnit string,
	start, end time.Time,
	sku string,
) *pbc.FocusCostRecord {
	return &pbc.FocusCostRecord{
		// Cost fields - all equal for public pricing (no discounts)
		BilledCost:    cost,
		EffectiveCost: cost,
		ListCost:      cost,

		// List unit price for transparency
		ListUnitPrice: unitPrice,

		// Service classification
		ServiceCategory: mapServiceCategory(serviceType),
		ServiceName:     getServiceName(serviceType),

		// Charge classification
		ChargeCategory:  pbc.FocusChargeCategory_FOCUS_CHARGE_CATEGORY_USAGE,
		ChargeClass:     pbc.FocusChargeClass_FOCUS_CHARGE_CLASS_REGULAR,
		ChargeFrequency: pbc.FocusChargeFrequency_FOCUS_CHARGE_FREQUENCY_USAGE_BASED,

		// Pricing
		PricingCategory: pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_STANDARD,
		PricingUnit:     pricingUnit,

		// Period
		ChargePeriodStart: timestamppb.New(start),
		ChargePeriodEnd:   timestamppb.New(end),

		// Location
		RegionId: region,

		// Currency (always USD for AWS public pricing)
		BillingCurrency: "USD",

		// Resource identification
		ResourceType: resourceType,
		SkuId:        sku,

		// Provider identification (FOCUS 1.3+)
		ServiceProviderName: "AWS",

		// Charge description for human readability
		ChargeDescription: fmt.Sprintf("Public pricing estimate for %s in %s", resourceType, region),
	}
}

// mapServiceCategory maps AWS service types to FOCUS service categories.
// This follows the FinOps FOCUS 1.2 standard service category definitions.
//
// Categories are based on the primary function of each AWS service:
//   - COMPUTE: Processing resources (EC2, Lambda, EKS worker nodes)
//   - STORAGE: Data persistence (S3, EBS)
//   - DATABASE: Managed database services (RDS, DynamoDB)
//   - NETWORK: Networking infrastructure (ELB, NAT Gateway)
//   - MANAGEMENT: Monitoring and operations (CloudWatch)
func mapServiceCategory(serviceType string) pbc.FocusServiceCategory {
	switch serviceType {
	case serviceEC2, serviceLambda:
		return pbc.FocusServiceCategory_FOCUS_SERVICE_CATEGORY_COMPUTE
	case serviceEBS, serviceS3:
		return pbc.FocusServiceCategory_FOCUS_SERVICE_CATEGORY_STORAGE
	case serviceRDS, serviceDynamoDB:
		return pbc.FocusServiceCategory_FOCUS_SERVICE_CATEGORY_DATABASE
	case serviceELB, serviceNATGW:
		return pbc.FocusServiceCategory_FOCUS_SERVICE_CATEGORY_NETWORK
	case serviceCloudWatch:
		return pbc.FocusServiceCategory_FOCUS_SERVICE_CATEGORY_MANAGEMENT
	case serviceEKS:
		// EKS control plane is compute; worker nodes would be EC2
		return pbc.FocusServiceCategory_FOCUS_SERVICE_CATEGORY_COMPUTE
	default:
		return pbc.FocusServiceCategory_FOCUS_SERVICE_CATEGORY_OTHER
	}
}

// getServiceName returns the AWS service display name for FOCUS ServiceName.
// Falls back to a formatted version of the service type if not found.
func getServiceName(serviceType string) string {
	if name, ok := awsServiceNames[serviceType]; ok {
		return name
	}
	// Fallback: capitalize and prefix with AWS
	return fmt.Sprintf("AWS %s", serviceType)
}

// getPricingUnitForService returns the appropriate pricing unit for a service.
// This is used when the caller doesn't have a specific pricing unit available.
func getPricingUnitForService(serviceType string) string {
	switch serviceType {
	case serviceEC2, serviceRDS, serviceEKS, serviceELB, serviceALB, serviceNLB, serviceNATGW:
		return "Hours"
	case serviceEBS, serviceS3:
		return "GB-Mo"
	case serviceLambda:
		return "GB-Seconds"
	case serviceDynamoDB:
		return "Requests" // Simplified; actual has RCU/WCU
	case serviceCloudWatch:
		return "GB" // For log ingestion
	default:
		return "Units"
	}
}
