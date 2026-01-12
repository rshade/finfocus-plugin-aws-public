package plugin

import (
	"testing"
	"time"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

// TestMapServiceCategory verifies that AWS service types are correctly mapped
// to FOCUS service categories. This is critical for FinOps reporting consistency.
func TestMapServiceCategory(t *testing.T) {
	tests := []struct {
		name        string
		serviceType string
		want        pbc.FocusServiceCategory
	}{
		// Compute services
		{
			name:        "EC2 maps to Compute",
			serviceType: "ec2",
			want:        pbc.FocusServiceCategory_FOCUS_SERVICE_CATEGORY_COMPUTE,
		},
		{
			name:        "Lambda maps to Compute",
			serviceType: "lambda",
			want:        pbc.FocusServiceCategory_FOCUS_SERVICE_CATEGORY_COMPUTE,
		},
		{
			name:        "EKS maps to Compute",
			serviceType: "eks",
			want:        pbc.FocusServiceCategory_FOCUS_SERVICE_CATEGORY_COMPUTE,
		},
		// Storage services
		{
			name:        "EBS maps to Storage",
			serviceType: "ebs",
			want:        pbc.FocusServiceCategory_FOCUS_SERVICE_CATEGORY_STORAGE,
		},
		{
			name:        "S3 maps to Storage",
			serviceType: "s3",
			want:        pbc.FocusServiceCategory_FOCUS_SERVICE_CATEGORY_STORAGE,
		},
		// Database services
		{
			name:        "RDS maps to Database",
			serviceType: "rds",
			want:        pbc.FocusServiceCategory_FOCUS_SERVICE_CATEGORY_DATABASE,
		},
		{
			name:        "DynamoDB maps to Database",
			serviceType: "dynamodb",
			want:        pbc.FocusServiceCategory_FOCUS_SERVICE_CATEGORY_DATABASE,
		},
		// Network services
		{
			name:        "ELB maps to Network",
			serviceType: "elb",
			want:        pbc.FocusServiceCategory_FOCUS_SERVICE_CATEGORY_NETWORK,
		},
		{
			name:        "NAT Gateway maps to Network",
			serviceType: "natgw",
			want:        pbc.FocusServiceCategory_FOCUS_SERVICE_CATEGORY_NETWORK,
		},
		// Management services
		{
			name:        "CloudWatch maps to Management",
			serviceType: "cloudwatch",
			want:        pbc.FocusServiceCategory_FOCUS_SERVICE_CATEGORY_MANAGEMENT,
		},
		// Unknown/other services
		{
			name:        "Unknown service maps to Other",
			serviceType: "unknown",
			want:        pbc.FocusServiceCategory_FOCUS_SERVICE_CATEGORY_OTHER,
		},
		{
			name:        "Empty string maps to Other",
			serviceType: "",
			want:        pbc.FocusServiceCategory_FOCUS_SERVICE_CATEGORY_OTHER,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mapServiceCategory(tt.serviceType)
			if got != tt.want {
				t.Errorf("mapServiceCategory(%q) = %v, want %v", tt.serviceType, got, tt.want)
			}
		})
	}
}

// TestGetServiceName verifies AWS service name lookup for FOCUS ServiceName field.
func TestGetServiceName(t *testing.T) {
	tests := []struct {
		serviceType string
		want        string
	}{
		{"ec2", "Amazon EC2"},
		{"ebs", "Amazon EBS"},
		{"s3", "Amazon S3"},
		{"rds", "Amazon RDS"},
		{"lambda", "AWS Lambda"},
		{"dynamodb", "Amazon DynamoDB"},
		{"eks", "Amazon EKS"},
		{"elb", "Elastic Load Balancing"},
		{"natgw", "Amazon VPC NAT Gateway"},
		{"cloudwatch", "Amazon CloudWatch"},
		{"unknown", "AWS unknown"}, // Fallback
	}

	for _, tt := range tests {
		t.Run(tt.serviceType, func(t *testing.T) {
			got := getServiceName(tt.serviceType)
			if got != tt.want {
				t.Errorf("getServiceName(%q) = %q, want %q", tt.serviceType, got, tt.want)
			}
		})
	}
}

// TestGetPricingUnitForService verifies pricing unit lookup for each service type.
func TestGetPricingUnitForService(t *testing.T) {
	tests := []struct {
		serviceType string
		want        string
	}{
		{"ec2", "Hours"},
		{"rds", "Hours"},
		{"eks", "Hours"},
		{"elb", "Hours"},
		{"alb", "Hours"},  // ALB variant
		{"nlb", "Hours"},  // NLB variant
		{"natgw", "Hours"},
		{"ebs", "GB-Mo"},
		{"s3", "GB-Mo"},
		{"lambda", "GB-Seconds"},
		{"dynamodb", "Requests"},
		{"cloudwatch", "GB"},
		{"unknown", "Units"},
	}

	for _, tt := range tests {
		t.Run(tt.serviceType, func(t *testing.T) {
			got := getPricingUnitForService(tt.serviceType)
			if got != tt.want {
				t.Errorf("getPricingUnitForService(%q) = %q, want %q", tt.serviceType, got, tt.want)
			}
		})
	}
}

// TestBuildFocusRecord verifies that FocusCostRecord is correctly constructed
// with all essential FOCUS 1.2 fields populated for public pricing estimates.
func TestBuildFocusRecord(t *testing.T) {
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC)

	record := buildFocusRecord(
		"ec2",
		"aws:ec2/instance:Instance",
		"us-east-1",
		100.50,     // cost
		0.1376,     // unit price
		"Hours",    // pricing unit
		start, end, // period
		"t3.micro", // sku
	)

	// Verify cost fields (all equal for public pricing)
	if record.BilledCost != 100.50 {
		t.Errorf("BilledCost = %v, want 100.50", record.BilledCost)
	}
	if record.EffectiveCost != 100.50 {
		t.Errorf("EffectiveCost = %v, want 100.50", record.EffectiveCost)
	}
	if record.ListCost != 100.50 {
		t.Errorf("ListCost = %v, want 100.50", record.ListCost)
	}
	if record.ListUnitPrice != 0.1376 {
		t.Errorf("ListUnitPrice = %v, want 0.1376", record.ListUnitPrice)
	}

	// Verify service classification
	if record.ServiceCategory != pbc.FocusServiceCategory_FOCUS_SERVICE_CATEGORY_COMPUTE {
		t.Errorf("ServiceCategory = %v, want COMPUTE", record.ServiceCategory)
	}
	if record.ServiceName != "Amazon EC2" {
		t.Errorf("ServiceName = %q, want 'Amazon EC2'", record.ServiceName)
	}

	// Verify charge classification
	if record.ChargeCategory != pbc.FocusChargeCategory_FOCUS_CHARGE_CATEGORY_USAGE {
		t.Errorf("ChargeCategory = %v, want USAGE", record.ChargeCategory)
	}
	if record.ChargeClass != pbc.FocusChargeClass_FOCUS_CHARGE_CLASS_REGULAR {
		t.Errorf("ChargeClass = %v, want REGULAR", record.ChargeClass)
	}
	if record.ChargeFrequency != pbc.FocusChargeFrequency_FOCUS_CHARGE_FREQUENCY_USAGE_BASED {
		t.Errorf("ChargeFrequency = %v, want USAGE_BASED", record.ChargeFrequency)
	}

	// Verify pricing
	if record.PricingCategory != pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_STANDARD {
		t.Errorf("PricingCategory = %v, want STANDARD", record.PricingCategory)
	}
	if record.PricingUnit != "Hours" {
		t.Errorf("PricingUnit = %q, want 'Hours'", record.PricingUnit)
	}

	// Verify timestamps
	if record.ChargePeriodStart == nil {
		t.Error("ChargePeriodStart is nil")
	} else if !record.ChargePeriodStart.AsTime().Equal(start) {
		t.Errorf("ChargePeriodStart = %v, want %v", record.ChargePeriodStart.AsTime(), start)
	}
	if record.ChargePeriodEnd == nil {
		t.Error("ChargePeriodEnd is nil")
	} else if !record.ChargePeriodEnd.AsTime().Equal(end) {
		t.Errorf("ChargePeriodEnd = %v, want %v", record.ChargePeriodEnd.AsTime(), end)
	}

	// Verify location and currency
	if record.RegionId != "us-east-1" {
		t.Errorf("RegionId = %q, want 'us-east-1'", record.RegionId)
	}
	if record.BillingCurrency != "USD" {
		t.Errorf("BillingCurrency = %q, want 'USD'", record.BillingCurrency)
	}

	// Verify resource identification
	if record.ResourceType != "aws:ec2/instance:Instance" {
		t.Errorf("ResourceType = %q, want 'aws:ec2/instance:Instance'", record.ResourceType)
	}
	if record.SkuId != "t3.micro" {
		t.Errorf("SkuId = %q, want 't3.micro'", record.SkuId)
	}

	// Verify provider
	if record.ServiceProviderName != "AWS" {
		t.Errorf("ServiceProviderName = %q, want 'AWS'", record.ServiceProviderName)
	}

	// Verify description exists
	if record.ChargeDescription == "" {
		t.Error("ChargeDescription is empty")
	}
}

// TestBuildFocusRecordStorageService verifies FocusCostRecord for storage services
// to ensure different service types produce correct categorization.
func TestBuildFocusRecordStorageService(t *testing.T) {
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC)

	record := buildFocusRecord(
		"s3",
		"aws:s3/bucket:Bucket",
		"us-west-2",
		25.00,
		0.023,
		"GB-Mo",
		start, end,
		"STANDARD",
	)

	// Verify storage category
	if record.ServiceCategory != pbc.FocusServiceCategory_FOCUS_SERVICE_CATEGORY_STORAGE {
		t.Errorf("ServiceCategory = %v, want STORAGE", record.ServiceCategory)
	}
	if record.ServiceName != "Amazon S3" {
		t.Errorf("ServiceName = %q, want 'Amazon S3'", record.ServiceName)
	}
	if record.PricingUnit != "GB-Mo" {
		t.Errorf("PricingUnit = %q, want 'GB-Mo'", record.PricingUnit)
	}
}

// TestBuildFocusRecordDatabaseService verifies FocusCostRecord for database services.
func TestBuildFocusRecordDatabaseService(t *testing.T) {
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC)

	record := buildFocusRecord(
		"rds",
		"aws:rds/instance:Instance",
		"eu-west-1",
		150.00,
		0.205,
		"Hours",
		start, end,
		"db.t3.medium",
	)

	// Verify database category
	if record.ServiceCategory != pbc.FocusServiceCategory_FOCUS_SERVICE_CATEGORY_DATABASE {
		t.Errorf("ServiceCategory = %v, want DATABASE", record.ServiceCategory)
	}
	if record.ServiceName != "Amazon RDS" {
		t.Errorf("ServiceName = %q, want 'Amazon RDS'", record.ServiceName)
	}
}

// TestBuildFocusRecordNetworkService verifies FocusCostRecord for network services.
func TestBuildFocusRecordNetworkService(t *testing.T) {
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC)

	record := buildFocusRecord(
		"natgw",
		"aws:ec2/natGateway:NatGateway",
		"ap-southeast-1",
		50.00,
		0.045,
		"Hours",
		start, end,
		"",
	)

	// Verify network category
	if record.ServiceCategory != pbc.FocusServiceCategory_FOCUS_SERVICE_CATEGORY_NETWORK {
		t.Errorf("ServiceCategory = %v, want NETWORK", record.ServiceCategory)
	}
	if record.ServiceName != "Amazon VPC NAT Gateway" {
		t.Errorf("ServiceName = %q, want 'Amazon VPC NAT Gateway'", record.ServiceName)
	}
}
