# Research: Core Protocol Intelligence

**Feature**: 030-core-protocol-intelligence
**Date**: 2026-01-05
**Status**: Complete

## Research Questions

### RQ-1: What proto fields exist for Growth Type Hints?

**Finding**: `GrowthType` enum and `growth_type` field EXIST in pulumicost-spec v0.4.12

**Evidence**:

```go
// GetProjectedCostResponse.growth_type (field 6)
GrowthType GrowthType `protobuf:"varint,6,opt,name=growth_type,json=growthType,proto3,enum=pulumicost.v1.GrowthType"`

// GrowthType enum values
GrowthType_GROWTH_TYPE_UNSPECIFIED  // 0 - Consumption-based (default)
GrowthType_GROWTH_TYPE_LINEAR       // 1 - Accumulation-based (S3, logs)
GrowthType_GROWTH_TYPE_EXPONENTIAL  // 2 - Compounding growth (rare)
```

**Decision**: Implement Growth Type Hints (P2) immediately using existing proto support.

**Rationale**: No proto changes needed. Can ship value to users now.

**Alternatives Considered**:

- Wait for all three features → Rejected: Delays value delivery unnecessarily
- Use string field instead of enum → Rejected: Enum exists, use it

---

### RQ-2: What proto fields exist for Dev Mode (UsageProfile)?

**Finding**: `UsageProfile` enum does NOT exist in pulumicost-spec v0.4.12

**Evidence**:

```bash
$ go doc github.com/rshade/pulumicost-spec/.../v1.UsageProfile
UsageProfile not found
```

**Decision**: Create PR to add `UsageProfile` to pulumicost-spec before implementing Dev Mode.

**Rationale**: Cannot implement without proto definition. Proto-first approach ensures compatibility.

**Proposed Proto Changes**:

```protobuf
// UsageProfile indicates the operational context for cost estimation
enum UsageProfile {
  USAGE_PROFILE_UNSPECIFIED = 0;  // Default: production (730 hrs/month)
  USAGE_PROFILE_PRODUCTION = 1;   // 24/7 operation (730 hrs/month)
  USAGE_PROFILE_DEVELOPMENT = 2;  // Business hours only (160 hrs/month)
  USAGE_PROFILE_BURST = 3;        // Reserved for future use
}

// Add to GetProjectedCostRequest
message GetProjectedCostRequest {
  // ... existing fields ...
  UsageProfile usage_profile = 10;  // OPTIONAL: operational context
}
```

**Alternatives Considered**:

- Use tags for dev mode hint → Rejected: Not discoverable, pollutes tags
- Use billing_detail string → Rejected: Not machine-readable

---

### RQ-3: What proto fields exist for Topology Linking?

**Finding**: `CostAllocationLineage` message does NOT exist in pulumicost-spec v0.4.12

**Evidence**:

```bash
$ go doc github.com/rshade/pulumicost-spec/.../v1.CostAllocationLineage
CostAllocationLineage not found
```

**Decision**: Create PR to add `CostAllocationLineage` to pulumicost-spec before implementing Topology Linking.

**Rationale**: Cannot implement without proto definition. Proto-first approach ensures compatibility.

**Proposed Proto Changes**:

```protobuf
// CostAllocationLineage describes parent/child relationships for topology
message CostAllocationLineage {
  // parent_resource_id is the ID of the parent resource (e.g., "i-abc123")
  string parent_resource_id = 1;
  // parent_resource_type is the type of the parent (e.g., "aws:ec2/instance:Instance")
  string parent_resource_type = 2;
  // relationship describes how this resource relates to its parent
  string relationship = 3;  // "attached_to", "within", "managed_by"
}

// Add to GetProjectedCostResponse
message GetProjectedCostResponse {
  // ... existing fields ...
  CostAllocationLineage lineage = 8;  // OPTIONAL: topology information
}
```

**Alternatives Considered**:

- Return parent info in billing_detail → Rejected: Not machine-readable
- Use tags in response → Rejected: Tags are input, not output

---

### RQ-4: Service Classification for Growth Types

**Finding**: Clear binary classification based on cost behavior

| Service | GrowthType | Rationale |
|---------|------------|-----------|
| EC2 | STATIC | Instance hours - no data accumulation |
| EBS | STATIC | Volume hours - size is fixed at creation |
| EKS | STATIC | Cluster hours - no data accumulation |
| S3 | LINEAR | Storage grows as objects are added |
| Lambda | STATIC | Pay per invocation - no accumulation |
| DynamoDB | LINEAR | Storage grows as items are added |
| ELB | STATIC | LB hours - no data accumulation |
| NAT Gateway | STATIC | Gateway hours + data (but data is throughput, not accumulation) |
| CloudWatch | STATIC | Ingestion is throughput, not accumulation |
| ElastiCache | STATIC | Node hours - no data accumulation |
| RDS | STATIC | Instance hours - storage size is provisioned |

**Decision**: Use static map for service → GrowthType classification.

**Rationale**: Classifications are inherent to service behavior, not configurable.

---

### RQ-5: Parent Tag Key Extraction Rules

**Finding**: Well-defined tag keys per service type

| Service | Tag Key | Relationship | Parent Type |
|---------|---------|--------------|-------------|
| EBS | `instance_id` | `attached_to` | EC2 Instance |
| ELB | `vpc_id` | `within` | VPC |
| NAT Gateway | `vpc_id` | `within` | VPC |
| NAT Gateway | `subnet_id` | `within` | Subnet (if vpc_id absent) |
| ElastiCache | `vpc_id` | `within` | VPC |
| RDS | `vpc_id` | `within` | VPC |

**Priority Rule**: When multiple parent tags exist, use first match in order:
`instance_id` > `cluster_name` > `vpc_id` > `subnet_id`

**Decision**: Extract parent from first matching tag key per priority.

**Rationale**: Provides deterministic, predictable behavior.

---

### RQ-6: Feature Detection Pattern for Protocol Field Availability

**Finding**: Use Go protobuf reflection or type assertion to check for protocol field availability at runtime, not version checking.

**Evidence**:
- pulumicost-spec versions may not correlate 1:1 with feature availability
- Proto optional fields default to zero values when unset
- Reflection on proto messages is performant and idiomatic

**Decision**: Use runtime feature detection via type assertion for protocol fields.

**Rationale**:
- Graceful degradation: older plugins ignore new fields
- No version-specific code paths
- Maintains backward compatibility with spec versions
- Aligns with NFR-001 requirement

**Implementation Pattern**:
```go
// Check if UsageProfile field exists in request
func hasUsageProfile(req *pulumicostv1.ResourceDescriptor) bool {
    _, ok := req.(interface{ GetUsageProfile() pulumicostv1.UsageProfile })
    return ok
}

// Check if GrowthType field exists in response
func hasGrowthHint(resp *pulumicostv1.GetProjectedCostResponse) bool {
    _, ok := resp.(interface{ SetGrowthType(pulumicostv1.GrowthType) })
    return ok
}

// Check if Lineage field exists in response
func hasLineage(resp *pulumicostv1.GetProjectedCostResponse) bool {
    _, ok := resp.(interface{ SetLineage(*pulumicostv1.CostAllocationLineage) })
    return ok
}
```

**Alternatives Considered**:
- Version string comparison → Rejected: pulumicost-spec version is TBD
- Build tags → Rejected: requires multiple binary builds
- Compile-time interface → Rejected: breaks compatibility

---

### RQ-7: Metadata Enrichment Architecture Pattern

**Finding**: Apply metadata enrichment after cost calculation using pure transformation functions.

**Evidence**:
- Cost calculation logic is existing and tested
- Metadata enrichment is orthogonal to pricing
- Zero breaking changes requirement (SC-004)

**Decision**: Append metadata to GetProjectedCostResponse after cost calculation using pure functions.

**Rationale**:
- Separates concerns (constitution Principle I: Single Responsibility)
- Easy to test independently
- Maintains backward compatibility
- No regression risk to existing cost calculations

**Implementation Pattern**:
```go
// Pure transformation functions (no side effects)
func applyDevMode(req *pulumicostv1.ResourceDescriptor, resp *pulumicostv1.GetProjectedCostResponse) {
    if !hasUsageProfile(req) {
        return // Field not available in this spec version
    }

    if req.GetUsageProfile() == pulumicostv1.UsageProfile_DEVELOPMENT {
        classification, ok := serviceClassifications[req.ResourceType]
        if ok && classification.AffectedByDevMode {
            resp.CostPerMonth = resp.CostPerMonth * hoursPerMonthDev / hoursPerMonthProd
            resp.BillingDetail += " (dev profile)"
        }
    }
}

func setGrowthHint(serviceType string, resp *pulumicostv1.GetProjectedCostResponse) {
    if !hasGrowthHint(resp) {
        return // Field not available
    }

    classification, ok := serviceClassifications[serviceType]
    if ok {
        resp.GrowthType = classification.GrowthType
    }
}

func extractLineage(req *pulumicostv1.ResourceDescriptor, resp *pulumicostv1.GetProjectedCostResponse) {
    if !hasLineage(resp) {
        return // Field not available
    }

    classification, ok := serviceClassifications[req.ResourceType]
    if !ok || len(classification.ParentTagKeys) == 0 {
        return
    }

    // Extract first matching tag key by priority
    for _, tagKey := range classification.ParentTagKeys {
        if parentID, exists := req.Tags[tagKey]; exists && parentID != "" {
            resp.Lineage = &pulumicostv1.CostAllocationLineage{
                ParentResourceId:   parentID,
                ParentResourceType: classification.ParentType,
                Relationship:       classification.Relationship,
            }
            return
        }
    }
}

// In gRPC handler - apply enrichment in order
func (s *Server) GetProjectedCost(ctx context.Context, req *pulumicostv1.ResourceDescriptor) (*pulumicostv1.GetProjectedCostResponse, error) {
    resp, err := s.calculateCost(req) // Existing logic - DO NOT MODIFY
    if err != nil {
        return nil, err
    }

    // Apply metadata enrichment (order matters)
    applyDevMode(req, resp)        // P1: Dev Mode
    setGrowthHint(req.ResourceType, resp)  // P2: Growth Hints
    extractLineage(req, resp)      // P3: Topology Linking

    return resp, nil
}
```

**Alternatives Considered**:
- Decorator pattern → Rejected: over-abstraction for simple field assignment
- Inline changes to cost calculation → Rejected: mixes concerns, regression risk
- Middleware → Rejected: this is response post-processing, not request interception

---

### RQ-8: Thread Safety and State Management

**Finding**: All metadata enrichment functions must be pure and stateless.

**Evidence**:
- Constitution Principle I: "Stateless components preferred"
- Constitution Principle III: "Thread-safe gRPC handlers required"
- Service classifications are static (spec assumption)

**Decision**: Use read-only constant map for service classifications, no shared mutable state.

**Rationale**:
- No locks needed (no concurrent writes)
- Thread-safe by design
- Meets constitution requirements

**Implementation Pattern**:
```go
// Read-only constant map (thread-safe by design)
type ServiceClassification struct {
    GrowthType        pulumicostv1.GrowthType
    AffectedByDevMode bool
    ParentTagKeys     []string // Priority order
    ParentType        string
    Relationship      string
}

var serviceClassifications = map[string]ServiceClassification{
    "aws:ec2:instance": {
        GrowthType:        pulumicostv1.GrowthType_GROWTH_TYPE_STATIC,
        AffectedByDevMode: true,
        ParentTagKeys:     nil,
    },
    "aws:s3:bucket": {
        GrowthType:        pulumicostv1.GrowthType_GROWTH_TYPE_LINEAR,
        AffectedByDevMode: false,
        ParentTagKeys:     nil,
    },
    "aws:ebs:volume": {
        GrowthType:        pulumicostv1.GrowthType_GROWTH_TYPE_STATIC,
        AffectedByDevMode: false, // Storage is not time-based
        ParentTagKeys:     []string{"instance_id"},
        ParentType:        "aws:ec2:instance:Instance",
        Relationship:      "attached_to",
    },
    "aws:elasticloadbalancing:loadbalancer": {
        GrowthType:        pulumicostv1.GrowthType_GROWTH_TYPE_STATIC,
        AffectedByDevMode: true, // Load balancer hours
        ParentTagKeys:     []string{"vpc_id"},
        ParentType:        "aws:ec2:vpc:Vpc",
        Relationship:      "within",
    },
    // ... other services
}

// Constants for dev mode hours (thread-safe - read-only)
const (
    hoursPerMonthProd = 730
    hoursPerMonthDev  = 160
)
```

**Alternatives Considered**:
- Mutex-protected shared state → Rejected: unnecessary complexity
- sync.Pool for response objects → Not needed: Go runtime handles allocation
- Dynamic classification discovery → Explicitly out-of-scope (spec assumption)

---

### RQ-9: Logging and Observability Pattern

**Finding**: Use zerolog with structured fields for metadata enrichment.

**Evidence**:
- Constitution Principle III: "Use zerolog for structured JSON logging to stderr"
- Spec requirement NFR-003: Log at INFO level when metadata added
- Spec requirement NFR-004: Include structured fields

**Decision**: Add structured logging when dev mode, growth hints, or lineage metadata is applied.

**Rationale**:
- Operational visibility into feature usage
- Debugging aid for metadata enrichment
- Aligns with project logging conventions

**Implementation Pattern**:
```go
import (
    "github.com/rs/zerolog"
)

// In applyDevMode function
func applyDevMode(ctx context.Context, req *pulumicostv1.ResourceDescriptor, resp *pulumicostv1.GetProjectedCostResponse) {
    if req.GetUsageProfile() == pulumicostv1.UsageProfile_DEVELOPMENT {
        // Apply dev mode...

        // Log metadata enrichment
        zerolog.Ctx(ctx).Info().
            Str("usage_profile", "DEVELOPMENT").
            Str("resource_type", req.ResourceType).
            Float64("original_cost", resp.CostPerMonth).
            Msg("Dev mode cost adjustment applied")
    }
}

// In extractLineage function
func extractLineage(ctx context.Context, req *pulumicostv1.ResourceDescriptor, resp *pulumicostv1.GetProjectedCostResponse) {
    if parentID, exists := req.Tags["instance_id"]; exists && parentID != "" {
        // Set lineage...

        // Log metadata enrichment
        zerolog.Ctx(ctx).Info().
            Str("parent_detected", "true").
            Str("parent_type", "aws:ec2:instance:Instance").
            Str("relationship", "attached_to").
            Str("resource_type", req.ResourceType).
            Msg("Parent-child lineage detected")
    }
}
```

**Alternatives Considered**:
- No logging → Rejected: violates NFR-003
- DEBUG level → Rejected: NFR-003 requires INFO
- Unstructured log messages → Rejected: violates NFR-004

---

### RQ-10: Performance Considerations

**Finding**: Metadata enrichment must be < 10ms overhead to stay within 100ms GetProjectedCost() target.

**Evidence**:
- Spec requirement SC-006: < 100ms total including enrichment
- Existing cost calculation takes ~50-80ms (from constitution benchmarks)
- Metadata enrichment is simple field assignments + map lookups

**Decision**: Optimize metadata enrichment for minimal overhead (target < 10ms).

**Rationale**:
- Meets SC-006 performance target
- Simple map lookups are O(1) and fast
- No external calls or complex computation

**Optimization Pattern**:
```go
// Fast path: check if feature detection passes early
func setGrowthHint(serviceType string, resp *pulumicostv1.GetProjectedCostResponse) {
    // Early return if field not available (no allocation)
    if !hasGrowthHint(resp) {
        return
    }

    // Map lookup is O(1) - very fast
    classification, ok := serviceClassifications[serviceType]
    if !ok {
        return
    }

    // Single field assignment
    resp.GrowthType = classification.GrowthType
}

// Logging only when metadata is actually applied (reduce overhead)
if parentID != "" {
    zerolog.Ctx(ctx).Info()... // Only log if lineage detected
}
```

**Performance Targets**:
- Feature detection (type assertion): < 1μs
- Map lookup: < 100ns
- Field assignment: < 1μs
- Total enrichment overhead: < 10ms

**Alternatives Considered**:
- Cached response objects → Rejected: unnecessary for simple field assignments
- Batch enrichment → Rejected: one resource per RPC (constitution)
- Skip logging → Rejected: violates NFR-003

---

## Research Summary

| Feature | Proto Status | Implementation Status | Key Decisions |
|---------|--------------|----------------------|----------------|
| Growth Type Hints (P2) | EXISTS | READY - Implement now | Static service classification map, pure transformation functions |
| Dev Mode (P1) | MISSING | BLOCKED - Proto PR needed | Feature detection via type assertion, 160hrs constant, structured logging |
| Topology Linking (P3) | MISSING | BLOCKED - Proto PR needed | Parent tag extraction priority, CostAllocationLineage message, thread-safe |

**Architecture Decisions**:
- Feature detection (runtime type assertion) instead of version checking
- Pure transformation functions for metadata enrichment (separate from cost calculation)
- Read-only constant map for service classifications (thread-safe, no locks)
- Zerolog structured logging at INFO level for metadata events
- < 10ms enrichment overhead to meet 100ms GetProjectedCost() target

**Protocol Extensions Required**:
1. `UsageProfile` enum in `GetProjectedCostRequest` (P1 - Dev Mode)
2. `CostAllocationLineage` message in `GetProjectedCostResponse` (P3 - Topology Linking)
3. `GrowthType` enum already exists (P2 - Growth Hints)

**Constitution Compliance**:
- ✅ Code Quality & Simplicity: Pure functions, single responsibility, no premature abstraction
- ✅ Testing Discipline: Unit tests for transformations, integration tests for gRPC
- ✅ Protocol & Interface Consistency: Proto-defined types, feature detection, thread-safe
- ✅ Performance & Reliability: < 100ms target, embedded data, no locks needed
- ✅ Build & Release Quality: make lint/test, GoReleaser compatible

## Next Steps

1. Implement Growth Type Hints (P2) using existing `GrowthType` enum - can start immediately
2. Create pulumicost-spec PR for `UsageProfile` enum (P1) and `CostAllocationLineage` message (P3)
3. Implement Dev Mode and Topology Linking after proto merges
4. Proceed to Phase 1: Design & Contracts
