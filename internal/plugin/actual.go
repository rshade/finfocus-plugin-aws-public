package plugin

import (
	"fmt"
	"time"

	pbc "github.com/rshade/pulumicost-spec/sdk/go/proto/pulumicost/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
func (p *AWSPublicPlugin) getProjectedForResource(traceID string, resource *pbc.ResourceDescriptor) (*pbc.GetProjectedCostResponse, error) {
	// Validate required fields
	if resource == nil {
		return nil, status.Error(codes.InvalidArgument, "missing resource descriptor")
	}

	if resource.Provider == "" || resource.ResourceType == "" || resource.Sku == "" || resource.Region == "" {
		return nil, status.Error(codes.InvalidArgument, "resource descriptor missing required fields (provider, resource_type, sku, region)")
	}

	// Check region match
	if resource.Region != p.region {
		details := map[string]string{
			"trace_id":       traceID,
			"pluginRegion":   p.region,
			"requiredRegion": resource.Region,
		}

		errDetail := &pbc.ErrorDetail{
			Code:    pbc.ErrorCode_ERROR_CODE_UNSUPPORTED_REGION,
			Message: fmt.Sprintf("Resource region %q does not match plugin region %q", resource.Region, p.region),
			Details: details,
		}

		st := status.New(codes.FailedPrecondition, errDetail.Message)
		st, _ = st.WithDetails(errDetail)
		return nil, st.Err()
	}

	// Route to appropriate estimator based on resource type
	switch resource.ResourceType {
	case "ec2":
		return p.estimateEC2(traceID, resource)
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
