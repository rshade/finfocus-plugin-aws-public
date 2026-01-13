# Quickstart: Core Protocol Intelligence

**Feature**: 030-core-protocol-intelligence
**Date**: 2026-01-05

## Overview

This feature adds intelligence metadata to `GetProjectedCostResponse`:

1. **Growth Type Hints** (P2) - Tell Core how resource costs grow over time
2. **Dev Mode** (P1) - Adjust estimates for dev/test environments (160 hrs vs 730)
3. **Topology Linking** (P3) - Identify parent/child resource relationships

**Key Principles**:
- Zero breaking changes to existing cost calculations (SC-004)
- Metadata enrichment is separate from pricing logic
- Use feature detection (runtime type assertions) for protocol fields
- All enrichment functions are pure and stateless
- Thread-safe: read-only service classification map, no locks

---

## Implementation Order

### Phase 1: Service Classification Map (Foundation)

**File**: `internal/plugin/classification.go` (NEW)

Create the service classification map as the single source of truth:

```go
package plugin

import pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"

// ServiceClassification defines metadata for cost estimation enrichment
type ServiceClassification struct {
    GrowthType        pbc.GrowthType
    AffectedByDevMode bool
    ParentTagKeys     []string // Priority order for parent extraction
    ParentType        string
    Relationship      string
}

// serviceClassifications is a read-only map of AWS service types to their metadata
var serviceClassifications = map[string]ServiceClassification{
    "aws:ec2:instance": {
        GrowthType:        pbc.GrowthType_GROWTH_TYPE_STATIC,
        AffectedByDevMode: true,  // Instance hours
        ParentTagKeys:     nil,
    },
    "aws:ebs:volume": {
        GrowthType:        pbc.GrowthType_GROWTH_TYPE_STATIC,
        AffectedByDevMode: false,  // Storage is not time-based
        ParentTagKeys:     []string{"instance_id"},
        ParentType:        "aws:ec2:instance:Instance",
        Relationship:      "attached_to",
    },
    "aws:eks:cluster": {
        GrowthType:        pbc.GrowthType_GROWTH_TYPE_STATIC,
        AffectedByDevMode: true,  // Cluster hours
        ParentTagKeys:     nil,
    },
    "aws:s3:bucket": {
        GrowthType:        pbc.GrowthType_GROWTH_TYPE_LINEAR,
        AffectedByDevMode: false,  // Storage is not time-based
        ParentTagKeys:     nil,
    },
    "aws:lambda:function": {
        GrowthType:        pbc.GrowthType_GROWTH_TYPE_STATIC,
        AffectedByDevMode: false,  // Usage-based
        ParentTagKeys:     nil,
    },
    "aws:dynamodb:table": {
        GrowthType:        pbc.GrowthType_GROWTH_TYPE_LINEAR,
        AffectedByDevMode: false,  // Usage-based
        ParentTagKeys:     nil,
    },
    "aws:elasticloadbalancing:loadbalancer": {
        GrowthType:        pbc.GrowthType_GROWTH_TYPE_STATIC,
        AffectedByDevMode: true,  // Load balancer hours
        ParentTagKeys:     []string{"vpc_id"},
        ParentType:        "aws:ec2:vpc:Vpc",
        Relationship:      "within",
    },
    "aws:ec2:nat-gateway": {
        GrowthType:        pbc.GrowthType_GROWTH_TYPE_STATIC,
        AffectedByDevMode: true,  // Gateway hours
        ParentTagKeys:     []string{"vpc_id", "subnet_id"},
        ParentType:        "aws:ec2:vpc:Vpc",
        Relationship:      "within",
    },
    "aws:cloudwatch:metric": {
        GrowthType:        pbc.GrowthType_GROWTH_TYPE_STATIC,
        AffectedByDevMode: false,  // Ingestion is throughput
        ParentTagKeys:     nil,
    },
    "aws:elasticache:cluster": {
        GrowthType:        pbc.GrowthType_GROWTH_TYPE_STATIC,
        AffectedByDevMode: true,  // Node hours
        ParentTagKeys:     []string{"vpc_id"},
        ParentType:        "aws:ec2:vpc:Vpc",
        Relationship:      "within",
    },
    "aws:rds:instance": {
        GrowthType:        pbc.GrowthType_GROWTH_TYPE_STATIC,
        AffectedByDevMode: true,  // Instance hours
        ParentTagKeys:     []string{"vpc_id"},
        ParentType:        "aws:ec2:vpc:Vpc",
        Relationship:      "within",
    },
}
```

---

### Phase 2: Feature Detection (Enabler)

**File**: `internal/plugin/enrichment.go` (NEW)

Add helper functions to check for protocol field availability:

```go
package plugin

import pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"

// hasUsageProfile checks if UsageProfile field exists in request (feature detection)
func hasUsageProfile(req *pbc.ResourceDescriptor) bool {
    _, ok := req.(interface{ GetUsageProfile() pbc.UsageProfile })
    return ok
}

// hasGrowthHint checks if GrowthType field exists in response (feature detection)
func hasGrowthHint(resp *pbc.GetProjectedCostResponse) bool {
    _, ok := resp.(interface{ SetGrowthType(pbc.GrowthType) })
    return ok
}

// hasLineage checks if Lineage field exists in response (feature detection)
func hasLineage(resp *pbc.GetProjectedCostResponse) bool {
    _, ok := resp.(interface{ SetLineage(*pbc.CostAllocationLineage) })
    return ok
}
```

---

### Phase 3: Growth Type Hints (P2) - READY NOW

**Files**: `internal/plugin/enrichment.go` (MODIFY), `internal/plugin/projected.go` (MODIFY)

**Step 1**: Add growth hint enrichment function:

```go
// setGrowthHint sets the growth_type field based on service classification
func setGrowthHint(serviceType string, resp *pbc.GetProjectedCostResponse) {
    if !hasGrowthHint(resp) {
        return // Field not available in this spec version
    }

    classification, ok := serviceClassifications[serviceType]
    if ok {
        resp.GrowthType = classification.GrowthType
    }
}
```

**Step 2**: Call enrichment in existing cost estimation functions:

```go
// In each cost estimation function (estimateEC2Cost, estimateEBSCost, etc.)
// AFTER calculating cost, BEFORE returning:

func estimateEC2Cost(...) (*pbc.GetProjectedCostResponse, error) {
    // ... existing cost calculation logic ...

    // Apply growth hint enrichment
    setGrowthHint("aws:ec2:instance", response)

    return response, nil
}
```

**Step 3**: Add unit tests:

```go
func TestSetGrowthHint(t *testing.T) {
    tests := []struct {
        name     string
        service  string
        expected pbc.GrowthType
    }{
        {"S3 bucket", "aws:s3:bucket", pbc.GrowthType_GROWTH_TYPE_LINEAR},
        {"DynamoDB table", "aws:dynamodb:table", pbc.GrowthType_GROWTH_TYPE_LINEAR},
        {"EC2 instance", "aws:ec2:instance", pbc.GrowthType_GROWTH_TYPE_STATIC},
        {"EBS volume", "aws:ebs:volume", pbc.GrowthType_GROWTH_TYPE_STATIC},
        {"EKS cluster", "aws:eks:cluster", pbc.GrowthType_GROWTH_TYPE_STATIC},
        {"Lambda function", "aws:lambda:function", pbc.GrowthType_GROWTH_TYPE_STATIC},
        {"ELB", "aws:elasticloadbalancing:loadbalancer", pbc.GrowthType_GROWTH_TYPE_STATIC},
        {"NAT Gateway", "aws:ec2:nat-gateway", pbc.GrowthType_GROWTH_TYPE_STATIC},
        {"CloudWatch", "aws:cloudwatch:metric", pbc.GrowthType_GROWTH_TYPE_STATIC},
        {"ElastiCache", "aws:elasticache:cluster", pbc.GrowthType_GROWTH_TYPE_STATIC},
        {"RDS instance", "aws:rds:instance", pbc.GrowthType_GROWTH_TYPE_STATIC},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            resp := &pbc.GetProjectedCostResponse{}
            setGrowthHint(tt.service, resp)
            assert.Equal(t, tt.expected, resp.GrowthType)
        })
    }
}
```

---

### Phase 4: Dev Mode (P1) - BLOCKED ON PROTO

**Protocol Change Required** in `rshade/finfocus-spec`:

Create PR to add `UsageProfile` enum:

```protobuf
enum UsageProfile {
  USAGE_PROFILE_UNSPECIFIED = 0;  // Default: production (730 hrs/month)
  USAGE_PROFILE_PRODUCTION = 1;   // 24/7 operation (730 hrs/month)
  USAGE_PROFILE_DEVELOPMENT = 2;  // Business hours only (160 hrs/month)
  USAGE_PROFILE_BURST = 3;        // Reserved (same as PRODUCTION)
}

message GetProjectedCostRequest {
  // ... existing fields ...
  UsageProfile usage_profile = 10;  // OPTIONAL: operational context
}
```

**After proto merge**, implementation steps:

**Step 1**: Add dev mode constants to `internal/plugin/constants.go`:

```go
const (
    hoursPerMonthProd = 730  // Production: 24 hours/day * 30 days
    hoursPerMonthDev  = 160  // Development: 8 hours/day * 5 days/week * 4 weeks
)
```

**Step 2**: Add dev mode enrichment function:

```go
import "context"
import "github.com/rs/zerolog"

// applyDevMode adjusts costs for DEVELOPMENT usage profile
func applyDevMode(ctx context.Context, req *pbc.ResourceDescriptor, resp *pbc.GetProjectedCostResponse) {
    if !hasUsageProfile(req) {
        return // Field not available in this spec version
    }

    if req.GetUsageProfile() != pbc.UsageProfile_DEVELOPMENT {
        return // Only DEVELOPMENT profile triggers adjustment
    }

    classification, ok := serviceClassifications[req.ResourceType]
    if !ok || !classification.AffectedByDevMode {
        return // Not a time-based service
    }

    // Apply dev mode cost reduction: 160/730 = ~22%
    resp.CostPerMonth = resp.CostPerMonth * hoursPerMonthDev / hoursPerMonthProd

    // Add billing detail annotation
    resp.BillingDetail += " (dev profile)"

    // Log enrichment event
    zerolog.Ctx(ctx).Info().
        Str("usage_profile", "DEVELOPMENT").
        Str("resource_type", req.ResourceType).
        Float64("original_cost", resp.CostPerMonth*hoursPerMonthProd/hoursPerMonthDev).
        Msg("Dev mode cost adjustment applied")
}
```

**Step 3**: Call enrichment in GetProjectedCost handler:

```go
func (s *Server) GetProjectedCost(ctx context.Context, req *pbc.ResourceDescriptor) (*pbc.GetProjectedCostResponse, error) {
    resp, err := s.calculateCost(req) // Existing logic
    if err != nil {
        return nil, err
    }

    // Apply dev mode enrichment (P1)
    applyDevMode(ctx, req, resp)

    // Apply growth hint enrichment (P2)
    setGrowthHint(req.ResourceType, resp)

    return resp, nil
}
```

**Step 4**: Add unit tests:

```go
func TestApplyDevMode(t *testing.T) {
    ctx := context.Background()

    tests := []struct {
        name           string
        service        string
        usageProfile   pbc.UsageProfile
        expectedRatio  float64  // devCost / prodCost
        expectBilling  bool
    }{
        {"EC2 dev mode", "aws:ec2:instance", pbc.UsageProfile_DEVELOPMENT, 160.0/730.0, true},
        {"EC2 production", "aws:ec2:instance", pbc.UsageProfile_PRODUCTION, 1.0, false},
        {"S3 dev mode", "aws:s3:bucket", pbc.UsageProfile_DEVELOPMENT, 1.0, false}, // Storage not time-based
        {"EBS dev mode", "aws:ebs:volume", pbc.UsageProfile_DEVELOPMENT, 1.0, false}, // Storage not time-based
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            req := &pbc.ResourceDescriptor{
                ResourceType: tt.service,
                UsageProfile: tt.usageProfile,
            }
            resp := &pbc.GetProjectedCostResponse{CostPerMonth: 100.0}

            applyDevMode(ctx, req, resp)

            assert.Equal(t, 100.0*tt.expectedRatio, resp.CostPerMonth)
            if tt.expectBilling {
                assert.Contains(t, resp.BillingDetail, "(dev profile)")
            }
        })
    }
}
```

---

### Phase 5: Topology Linking (P3) - BLOCKED ON PROTO

**Protocol Change Required** in `rshade/finfocus-spec`:

Create PR to add `CostAllocationLineage` message:

```protobuf
message CostAllocationLineage {
  string parent_resource_id = 1;     // ID of parent resource
  string parent_resource_type = 2;    // Type of parent resource
  string relationship = 3;             // "attached_to", "within", "managed_by"
}

message GetProjectedCostResponse {
  // ... existing fields ...
  CostAllocationLineage lineage = 8;  // OPTIONAL: topology information
}
```

**After proto merge**, implementation steps:

**Step 1**: Add lineage enrichment function:

```go
// extractLineage extracts parent/child relationships from tags
func extractLineage(ctx context.Context, req *pbc.ResourceDescriptor, resp *pbc.GetProjectedCostResponse) {
    if !hasLineage(resp) {
        return // Field not available in this spec version
    }

    classification, ok := serviceClassifications[req.ResourceType]
    if !ok || len(classification.ParentTagKeys) == 0 {
        return // No parent extraction for this service
    }

    // Extract first matching tag key by priority
    for _, tagKey := range classification.ParentTagKeys {
        if parentID, exists := req.Tags[tagKey]; exists && parentID != "" {
            resp.Lineage = &pbc.CostAllocationLineage{
                ParentResourceId:   parentID,
                ParentResourceType: classification.ParentType,
                Relationship:       classification.Relationship,
            }

            // Log enrichment event
            zerolog.Ctx(ctx).Info().
                Str("parent_detected", "true").
                Str("parent_type", classification.ParentType).
                Str("relationship", classification.Relationship).
                Str("resource_type", req.ResourceType).
                Msg("Parent-child lineage detected")

            return
        }
    }
}
```

**Step 2**: Call enrichment in GetProjectedCost handler:

```go
func (s *Server) GetProjectedCost(ctx context.Context, req *pbc.ResourceDescriptor) (*pbc.GetProjectedCostResponse, error) {
    resp, err := s.calculateCost(req) // Existing logic
    if err != nil {
        return nil, err
    }

    // Apply enrichment in priority order
    applyDevMode(ctx, req, resp)        // P1: Dev Mode
    setGrowthHint(req.ResourceType, resp)  // P2: Growth Hints
    extractLineage(ctx, req, resp)      // P3: Topology Linking

    return resp, nil
}
```

**Step 3**: Add unit tests:

```go
func TestExtractLineage(t *testing.T) {
    ctx := context.Background()

    tests := []struct {
        name          string
        service       string
        tags          map[string]string
        expectedParent string
    }{
        {"EBS with instance_id", "aws:ebs:volume", map[string]string{"instance_id": "i-abc123"}, "i-abc123"},
        {"EBS without tags", "aws:ebs:volume", map[string]string{}, ""},
        {"ELB with vpc_id", "aws:elasticloadbalancing:loadbalancer", map[string]string{"vpc_id": "vpc-xyz"}, "vpc-xyz"},
        {"NAT with vpc_id", "aws:ec2:nat-gateway", map[string]string{"vpc_id": "vpc-xyz", "subnet_id": "subnet-123"}, "vpc-xyz"}, // vpc_id has priority
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            req := &pbc.ResourceDescriptor{
                ResourceType: tt.service,
                Tags:         tt.tags,
            }
            resp := &pbc.GetProjectedCostResponse{}

            extractLineage(ctx, req, resp)

            if tt.expectedParent != "" {
                assert.NotNil(t, resp.Lineage)
                assert.Equal(t, tt.expectedParent, resp.Lineage.ParentResourceId)
            } else {
                assert.Nil(t, resp.Lineage)
            }
        })
    }
}
```

---

## Testing Strategy

### Unit Tests

**File**: `internal/plugin/projected_test.go`

- **Service classification tests**: Verify map entries are correct
- **Feature detection tests**: Verify type assertions work correctly
- **Growth hint tests**: Table-driven for all 11 services
- **Dev mode tests**: Verify cost reduction ratio and billing detail
- **Lineage extraction tests**: Verify parent tag priority and relationship mapping
- **Edge cases**: Unknown services, missing tags, invalid enum values

### Integration Tests

**File**: `test/integration/metadata_enrichment_test.go`

```go
func TestMetadataEnrichmentIntegration(t *testing.T) {
    // Start plugin server
    // Use grpcurl or plugin SDK client to call GetProjectedCost
    // Verify response includes:
    //   - growth_type field populated
    //   - lineage field populated (when parent tags present)
    //   - dev mode cost reduction (when UsageProfile=DEVELOPMENT)
}
```

---

## Verification Commands

```bash
# Run all tests
make test

# Run specific test
go test ./internal/plugin/... -run TestSetGrowthHint -v

# Run tests with build tag
go test -tags=region_use1 ./internal/plugin/... -run TestApplyDevMode -v

# Verify lint passes
make lint

# Build for specific region
make build-region REGION=us-east-1

# Check binary size
ls -lh finfocus-plugin-aws-public-us-east-1
```

---

## Success Metrics

| Metric | Target | Verification |
|--------|--------|--------------|
| Growth hints populated | 100% of 11 services | Unit tests |
| Dev mode cost reduction | 160/730 = 21.9% | Integration test |
| Parent extraction | 100% when parent tag present | Unit tests |
| Backward compatibility | Zero cost calculation changes | Regression tests |
| Performance | < 10ms enrichment overhead | Benchmark tests |
| Thread safety | No race conditions | `go test -race` |

---

## References

- Data model: data-model.md
- Research findings: research.md
- Feature spec: spec.md
- Implementation plan: plan.md
- Constitution: .specify/memory/constitution.md
