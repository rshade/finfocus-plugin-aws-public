package plugin

import (
	"fmt"
	"time"

	pbc "github.com/rshade/pulumicost-spec/sdk/go/proto/pulumicost/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/rshade/pulumicost-spec/sdk/go/pluginsdk"
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
		stWithDetails, err := st.WithDetails(errDetail)
		if err != nil {
			// Log a warning if details could not be attached
			p.logger.Warn().
				Str(pluginsdk.FieldTraceID, traceID). // Use traceID directly here
				Str("grpc_code", codes.FailedPrecondition.String()).
				Str("message", errDetail.Message).
				Str("error_code", pbc.ErrorCode_ERROR_CODE_UNSUPPORTED_REGION.String()).
				Err(err). // Log the error returned by WithDetails
				Msg("failed to attach error details to gRPC status for region mismatch in actual cost calculation")
			return nil, st.Err() // Return original status without details
		}
		return nil, stWithDetails.Err()
	}

	// Normalize resource type (handles Pulumi formats like aws:ec2/instance:Instance)
	serviceType := detectService(resource.ResourceType)

	// Route to appropriate estimator based on normalized resource type
	switch serviceType {
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
