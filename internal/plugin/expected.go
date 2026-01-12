package plugin

import (
	"fmt"
	"math"

	"github.com/rshade/finfocus-plugin-aws-public/internal/carbon"
)

// ExpectedCostRange defines expected cost values with tolerance for E2E test validation.
// These are reference values based on AWS public pricing at the ReferenceDate.
type ExpectedCostRange struct {
	ResourceType     string  // "ec2" or "ebs"
	SKU              string  // Instance type (e.g., "t3.micro") or volume type (e.g., "gp2")
	Region           string  // AWS region (e.g., "us-east-1")
	UnitPrice        float64 // Hourly rate (EC2) or GB-month rate (EBS)
	MonthlyEstimate  float64 // Expected monthly cost for standard configuration
	TolerancePercent float64 // Allowed deviation percentage (1.0 for EC2, 5.0 for EBS)
	ReferenceDate    string  // When pricing was captured (ISO date format)
}

// Min returns the minimum acceptable cost within tolerance.
func (e ExpectedCostRange) Min() float64 {
	return e.MonthlyEstimate * (1 - e.TolerancePercent/100)
}

// Max returns the maximum acceptable cost within tolerance.
func (e ExpectedCostRange) Max() float64 {
	return e.MonthlyEstimate * (1 + e.TolerancePercent/100)
}

// ExpectedCostRanges contains all documented expected cost ranges for test resources.
// Keys are formatted as "resourceType:sku:region" for efficient lookup.
var ExpectedCostRanges = map[string]ExpectedCostRange{
	// EC2 t3.micro in us-east-1 - Standard E2E test resource
	"ec2:t3.micro:us-east-1": {
		ResourceType:     "ec2",
		SKU:              "t3.micro",
		Region:           "us-east-1",
		UnitPrice:        0.0104, // $/hour
		MonthlyEstimate:  7.592,  // 0.0104 * 730 hours
		TolerancePercent: 1.0,    // 1% tolerance for EC2
		ReferenceDate:    "2025-12-01",
	},
	// EBS gp2 in us-east-1 - Standard E2E test resource (8GB default)
	"ebs:gp2:us-east-1": {
		ResourceType:     "ebs",
		SKU:              "gp2",
		Region:           "us-east-1",
		UnitPrice:        0.10, // $/GB-month
		MonthlyEstimate:  0.80, // 0.10 * 8 GB
		TolerancePercent: 5.0,  // 5% tolerance for EBS
		ReferenceDate:    "2025-12-01",
	},
}

// buildExpectedRangeKey constructs the map key for ExpectedCostRanges lookup.
func buildExpectedRangeKey(resourceType, sku, region string) string {
	return fmt.Sprintf("%s:%s:%s", resourceType, sku, region)
}

// GetExpectedRange looks up the expected cost range for a test resource.
// Returns the range and true if found, or zero value and false if not found.
func GetExpectedRange(resourceType, sku, region string) (ExpectedCostRange, bool) {
	key := buildExpectedRangeKey(resourceType, sku, region)
	r, found := ExpectedCostRanges[key]
	return r, found
}

// IsWithinTolerance checks if an actual cost is within the expected tolerance range.
// Returns true if the actual value is within ±tolerancePercent of expected.
// Special case: if expected is 0, actual must also be 0.
func IsWithinTolerance(actual, expected, tolerancePercent float64) bool {
	if expected == 0 {
		return actual == 0
	}
	deviation := math.Abs(actual-expected) / expected * 100
	return deviation <= tolerancePercent
}

// CalculateExpectedActualCost computes the expected actual cost using the fallback formula.
// Formula: projected_monthly_cost × (runtime_hours / 730)
// This is used for validating GetActualCost responses in E2E tests.
func CalculateExpectedActualCost(projectedMonthlyCost, runtimeHours float64) float64 {
	return projectedMonthlyCost * (runtimeHours / carbon.HoursPerMonth)
}
