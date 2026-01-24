package plugin

import (
	"encoding/json"
	"fmt"
	"time"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

// Standard tag keys for Pulumi state metadata.
// These keys are injected by finfocus-core from Pulumi state.
const (
	TagPulumiCreated  = "pulumi:created"
	TagPulumiModified = "pulumi:modified"
	TagPulumiExternal = "pulumi:external"
)

// ConfidenceLevel represents the estimation confidence for actual cost calculations.
// The level affects how the cost data should be interpreted by consumers.
type ConfidenceLevel string

const (
	// ConfidenceHigh indicates precise calculation with known timestamps.
	// Used when: explicit request timestamps OR native resource with pulumi:created.
	ConfidenceHigh ConfidenceLevel = "HIGH"

	// ConfidenceMedium indicates reasonable estimate with caveats.
	// Used when: imported resource (pulumi:external=true) - timestamp is import time.
	ConfidenceMedium ConfidenceLevel = "MEDIUM"

	// ConfidenceLow indicates rough estimate or fallback.
	// Used when: unsupported resource, missing data, or significant assumptions.
	ConfidenceLow ConfidenceLevel = "LOW"
)

// TimestampResolution captures the resolved timestamps and their source.
// This struct is used internally to track how timestamps were determined
// and whether the resource was imported.
type TimestampResolution struct {
	// Start is the resolved start timestamp for cost calculation.
	Start time.Time

	// End is the resolved end timestamp for cost calculation.
	End time.Time

	// Source indicates how timestamps were resolved.
	// Values: "explicit", "pulumi:created", "mixed"
	Source string

	// IsImported is true when the resource has pulumi:external=true.
	// This affects confidence level (MEDIUM vs HIGH).
	IsImported bool
}

// extractPulumiCreated parses the pulumi:created timestamp from tags.
// Returns (timestamp, true) if valid RFC3339, or (zero, false) if missing/invalid.
// This function is thread-safe and has no side effects.
func extractPulumiCreated(tags map[string]string) (time.Time, bool) {
	if tags == nil {
		return time.Time{}, false
	}
	createdStr, exists := tags[TagPulumiCreated]
	if !exists || createdStr == "" {
		return time.Time{}, false
	}
	t, err := time.Parse(time.RFC3339, createdStr)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

// isImportedResource checks if the resource has pulumi:external=true.
// The comparison is case-sensitive per the spec.
func isImportedResource(tags map[string]string) bool {
	if tags == nil {
		return false
	}
	return tags[TagPulumiExternal] == "true"
}

// formatSourceWithConfidence creates the source string with embedded confidence.
// Format: "aws-public-fallback[confidence:LEVEL] optional_note"
func formatSourceWithConfidence(confidence ConfidenceLevel, note string) string {
	base := fmt.Sprintf("aws-public-fallback[confidence:%s]", confidence)
	if note != "" {
		return base + " " + note
	}
	return base
}

// mergeTagsFromRequest extracts tags from both req.Tags and ResourceId JSON.
// Tags in req.Tags take precedence over those in ResourceId JSON.
// This ensures Pulumi metadata tags work regardless of where they're placed.
func mergeTagsFromRequest(req *pbc.GetActualCostRequest) map[string]string {
	merged := make(map[string]string)

	// First, try to extract tags from ResourceId JSON
	if req.ResourceId != "" {
		var resource struct {
			Tags map[string]string `json:"tags"`
		}
		if json.Unmarshal([]byte(req.ResourceId), &resource) == nil && resource.Tags != nil {
			for k, v := range resource.Tags {
				merged[k] = v
			}
		}
	}

	// Then overlay req.Tags (takes precedence)
	if req.Tags != nil {
		for k, v := range req.Tags {
			merged[k] = v
		}
	}

	return merged
}

// resolveTimestamps applies priority-based timestamp resolution.
// Priority order: (1) explicit req.Start/End, (2) pulumi:created tag, (3) error
//
// This function MUST be called BEFORE validateTimestamps() to allow automatic
// timestamp detection from Pulumi state metadata. If explicit timestamps are
// provided, they take precedence over tags.
//
// Tags are merged from both req.Tags and ResourceId JSON (if present).
// req.Tags takes precedence over ResourceId JSON tags.
//
// For end time: if explicit End is provided, use it; otherwise default to now.
//
// Returns TimestampResolution with source tracking for confidence determination.
// Returns error if no valid timestamps can be resolved.
func resolveTimestamps(req *pbc.GetActualCostRequest) (*TimestampResolution, error) {
	if req == nil {
		return nil, fmt.Errorf("request is nil")
	}

	// Merge tags from both req.Tags and ResourceId JSON
	mergedTags := mergeTagsFromRequest(req)

	resolution := &TimestampResolution{
		IsImported: isImportedResource(mergedTags),
	}

	// Track which timestamp sources are used
	hasExplicitStart := req.Start != nil
	hasExplicitEnd := req.End != nil

	// Resolve start time
	if hasExplicitStart {
		resolution.Start = req.Start.AsTime()
		resolution.Source = "explicit"
	} else {
		// Try pulumi:created from merged tags
		if created, found := extractPulumiCreated(mergedTags); found {
			resolution.Start = created
			resolution.Source = "pulumi:created"
		} else {
			return nil, fmt.Errorf("start time required: provide explicit Start or pulumi:created tag")
		}
	}

	// Resolve end time
	if hasExplicitEnd {
		resolution.End = req.End.AsTime()
		// Update source if we have mixed sources
		if !hasExplicitStart && resolution.Source == "pulumi:created" {
			resolution.Source = "mixed"
		}
	} else {
		// Default end to now
		resolution.End = time.Now().UTC()
		// Update source if we had explicit start only
		if hasExplicitStart {
			resolution.Source = "mixed"
		}
	}

	return resolution, nil
}

// determineConfidence maps resolution source and import status to confidence level.
//
// Confidence rules:
//   - HIGH: Explicit timestamps OR native resource with pulumi:created
//   - MEDIUM: Imported resource (pulumi:external=true) with pulumi:created
//   - LOW: Unsupported resource type or significant assumptions (handled elsewhere)
func determineConfidence(resolution *TimestampResolution) ConfidenceLevel {
	if resolution == nil {
		return ConfidenceLow
	}

	// Explicit timestamps always get HIGH confidence
	if resolution.Source == "explicit" {
		return ConfidenceHigh
	}

	// For pulumi:created or mixed sources, check if imported
	if resolution.IsImported {
		return ConfidenceMedium
	}

	// Native resource with pulumi:created
	return ConfidenceHigh
}

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
//
// The resolver parameter is optional. If provided, it's used to get the cached
// service type. If nil, a new resolver is created internally (for backward compatibility).
func (p *AWSPublicPlugin) getProjectedForResource(traceID string, resource *pbc.ResourceDescriptor, resolver *serviceResolver) (*pbc.GetProjectedCostResponse, error) {
	// Defensive nil check - callers should validate, but be safe
	if resource == nil {
		return nil, fmt.Errorf("resource descriptor is nil (caller must validate)")
	}

	// Use provided resolver or create one if not provided (backward compatibility)
	if resolver == nil {
		resolver = newServiceResolver(resource.ResourceType)
	}
	serviceType := resolver.ServiceType()

	// Route to appropriate estimator based on normalized resource type.
	// For GetActualCost, we construct a minimal request with just the resource.
	// This means UtilizationPercentage is 0, which falls through to default (50%).
	switch serviceType {
	case "ec2":
		return p.estimateEC2(traceID, resource, &pbc.GetProjectedCostRequest{Resource: resource})
	case "ebs":
		return p.estimateEBS(traceID, resource)
	case "eks":
		return p.estimateEKS(traceID, resource)
	case "elb":
		return p.estimateELB(traceID, resource)
	case "natgw":
		return p.estimateNATGateway(traceID, resource)
	case "cloudwatch":
		return p.estimateCloudWatch(traceID, resource)
	case "elasticache":
		return p.estimateElastiCache(traceID, resource)
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
