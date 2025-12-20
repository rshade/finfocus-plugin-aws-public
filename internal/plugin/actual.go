package plugin

import (
	"fmt"
	"time"

	pbc "github.com/rshade/pulumicost-spec/sdk/go/proto/pulumicost/v1"
)

// calculateRuntimeHours computes the duration between two timestamps in hours.
// Returns a float64 for precise fractional hour calculations.
// Returns an error if from > to (negative duration).
func calculateRuntimeHours(from, to time.Time) (float64, error) {
	duration := to.Sub(from)
	if duration < 0 {
		return 0, fmt.Errorf("invalid time range: from (%v) is after to (%v)", from, to)
	}
	return duration.Hours(), nil
}

// getProjectedForResource retrieves the projected monthly cost for a resource
// by routing to the appropriate estimator (EC2, EBS, or stub).
// This reuses the existing GetProjectedCost logic without proto marshaling overhead.
// traceID is passed from the parent handler for consistent trace correlation.
//
// PRECONDITION: resource must be non-nil and already validated by caller.
// This function is called internally after validation in GetActualCost.
func (p *AWSPublicPlugin) getProjectedForResource(traceID string, resource *pbc.ResourceDescriptor) (*pbc.GetProjectedCostResponse, error) {
	// Defensive nil check - callers should validate, but be safe
	if resource == nil {
		return nil, fmt.Errorf("resource descriptor is nil (caller must validate)")
	}

	// Normalize resource type (handles Pulumi formats like aws:ec2/instance:Instance)
	serviceType := detectService(resource.ResourceType)

	// Route to appropriate estimator based on normalized resource type.
	// For GetActualCost, we construct a minimal request with just the resource.
	// This means UtilizationPercentage is 0, which falls through to default (50%).
	switch serviceType {
	case "ec2":
		return p.estimateEC2(traceID, resource, &pbc.GetProjectedCostRequest{Resource: resource})
	case "ebs":
		return p.estimateEBS(traceID, resource)
	case "s3", "lambda", "rds", "dynamodb":
		return p.estimateStub(resource)
	default:
		// Unknown resource type - return $0 with explanation
		return &pbc.GetProjectedCostResponse{
			CostPerMonth:  0,
			UnitPrice:     0,
			Currency:      "USD",
			BillingDetail: fmt.Sprintf("Resource type %q not supported for cost estimation", resource.ResourceType),
		}, nil
	}
}

// formatActualBillingDetail creates a human-readable billing detail string
// that explains the fallback calculation basis.
func formatActualBillingDetail(projectedDetail string, runtimeHours float64, actualCost float64) string {
	return fmt.Sprintf("Fallback estimate: %s × %.2f hours / 730 = $%.4f", projectedDetail, runtimeHours, actualCost)
}
