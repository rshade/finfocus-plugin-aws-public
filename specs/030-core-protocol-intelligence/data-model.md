# Data Model: Core Protocol Intelligence

**Feature**: 030-core-protocol-intelligence
**Date**: 2026-01-05
**Phase**: Phase 1 - Design & Contracts

## Overview

This document defines the data model for metadata enrichment in cost estimation responses. The data model includes protocol enums, service classifications, and lineage relationships. All entities are stateless and read-only for thread safety.

## Protocol Enums

### UsageProfile

**Purpose**: Indicates operational context for cost estimation, enabling Dev Mode realistic estimates.

**Proto Definition**:
```protobuf
enum UsageProfile {
  USAGE_PROFILE_UNSPECIFIED = 0;  // Default: production (730 hrs/month)
  USAGE_PROFILE_PRODUCTION = 1;   // 24/7 operation (730 hrs/month)
  USAGE_PROFILE_DEVELOPMENT = 2;  // Business hours only (160 hrs/month)
  USAGE_PROFILE_BURST = 3;        // Reserved for future use (same as PRODUCTION)
}
```

**Validation Rules**:
- Default to `USAGE_PROFILE_UNSPECIFIED` if field not set or unavailable
- Treat `USAGE_PROFILE_BURST` as production (730 hrs/month)
- Only `DEVELOPMENT` triggers hour reduction to 160

**Usage**:
- **Input field**: `GetProjectedCostRequest.usage_profile`
- **Scope**: Affects hourly-based services only (EC2, EKS, ELB, NAT Gateway, ElastiCache, RDS)
- **Impact**: Multiplies cost by `160/730` (~22%) when `DEVELOPMENT`

---

### GrowthType

**Purpose**: Indicates cost growth pattern for forecasting models (Cost Time Machine).

**Proto Definition** (exists in pulumicost-spec v0.4.12):
```protobuf
enum GrowthType {
  GROWTH_TYPE_UNSPECIFIED = 0;  // Default: static cost
  GROWTH_TYPE_STATIC = 1;       // Fixed cost regardless of time/usage
  GROWTH_TYPE_LINEAR = 2;       // Accumulates linearly over time (e.g., storage)
  GROWTH_TYPE_EXPONENTIAL = 3;  // Compounding growth (reserved)
}
```

**Validation Rules**:
- Default to `GROWTH_TYPE_UNSPECIFIED` if field not set or unavailable
- Must set explicit value for all supported services
- Omit field for unsupported resources

**Usage**:
- **Output field**: `GetProjectedCostResponse.growth_type`
- **Scope**: Applied to all 10 supported AWS services
- **Impact**: Core uses this to select growth model for forecasting

---

## Protocol Messages

### CostAllocationLineage

**Purpose**: Describes parent/child relationships for topology visualization (Blast Radius).

**Proto Definition**:
```protobuf
message CostAllocationLineage {
  string parent_resource_id = 1;     // ID of parent resource (e.g., "i-abc123")
  string parent_resource_type = 2;    // Type of parent (e.g., "aws:ec2:instance:Instance")
  string relationship = 3;             // Relationship type: "attached_to", "within", "managed_by"
}
```

**Validation Rules**:
- All fields are optional (string types)
- If `parent_resource_id` is empty, omit entire message
- `relationship` must be one of: `attached_to`, `within`, `managed_by`

**Usage**:
- **Output field**: `GetProjectedCostResponse.lineage`
- **Scope**: Applied to resources with parent tags (EBS, ELB, NAT Gateway, ElastiCache, RDS)
- **Impact**: Core builds dependency graph for impact analysis

---

## Service Classifications

### ServiceClassification Entity

**Purpose**: Static metadata per AWS service type for growth classification, dev mode applicability, and parent tag extraction.

**Go Definition**:
```go
type ServiceClassification struct {
    GrowthType        pulumicostv1.GrowthType  // Growth pattern
    AffectedByDevMode bool                      // Whether dev mode reduces cost
    ParentTagKeys     []string                  // Tag keys to extract parent (priority order)
    ParentType        string                    // Parent resource type string
    Relationship      string                    // Relationship to parent
}
```

**Validation Rules**:
- All fields are read-only (constant map)
- Map keys are `aws:service:resource` format (e.g., "aws:ec2:instance:Instance")
- Map lookups are case-sensitive
- Empty `ParentTagKeys` means no parent extraction for this service

---

### Service Classification Map

**Purpose**: Single source of truth for all supported AWS services.

**Go Definition**:
```go
var serviceClassifications = map[string]ServiceClassification{
    "aws:ec2:instance": {
        GrowthType:        pulumicostv1.GrowthType_GROWTH_TYPE_STATIC,
        AffectedByDevMode: true,  // Instance hours
        ParentTagKeys:     nil,    // No parent
    },
    "aws:ebs:volume": {
        GrowthType:        pulumicostv1.GrowthType_GROWTH_TYPE_STATIC,
        AffectedByDevMode: false,  // Storage is not time-based
        ParentTagKeys:     []string{"instance_id"},
        ParentType:        "aws:ec2:instance:Instance",
        Relationship:      "attached_to",
    },
    "aws:eks:cluster": {
        GrowthType:        pulumicostv1.GrowthType_GROWTH_TYPE_STATIC,
        AffectedByDevMode: true,  // Cluster hours
        ParentTagKeys:     nil,    // No parent
    },
    "aws:s3:bucket": {
        GrowthType:        pulumicostv1.GrowthType_GROWTH_TYPE_LINEAR,
        AffectedByDevMode: false,  // Storage is not time-based
        ParentTagKeys:     nil,    // No parent
    },
    "aws:lambda:function": {
        GrowthType:        pulumicostv1.GrowthType_GROWTH_TYPE_STATIC,
        AffectedByDevMode: false,  // Usage-based (requests/compute)
        ParentTagKeys:     nil,    // No parent
    },
    "aws:dynamodb:table": {
        GrowthType:        pulumicostv1.GrowthType_GROWTH_TYPE_LINEAR,
        AffectedByDevMode: false,  // Usage-based (throughput/storage)
        ParentTagKeys:     nil,    // No parent
    },
    "aws:elasticloadbalancing:loadbalancer": {
        GrowthType:        pulumicostv1.GrowthType_GROWTH_TYPE_STATIC,
        AffectedByDevMode: true,  // Load balancer hours
        ParentTagKeys:     []string{"vpc_id"},
        ParentType:        "aws:ec2:vpc:Vpc",
        Relationship:      "within",
    },
    "aws:ec2:nat-gateway": {
        GrowthType:        pulumicostv1.GrowthType_GROWTH_TYPE_STATIC,
        AffectedByDevMode: true,  // Gateway hours
        ParentTagKeys:     []string{"vpc_id", "subnet_id"},
        ParentType:        "aws:ec2:vpc:Vpc",
        Relationship:      "within",
    },
    "aws:cloudwatch:metric": {
        GrowthType:        pulumicostv1.GrowthType_GROWTH_TYPE_STATIC,
        AffectedByDevMode: false,  // Ingestion is throughput
        ParentTagKeys:     nil,    // No parent
    },
    "aws:elasticache:cluster": {
        GrowthType:        pulumicostv1.GrowthType_GROWTH_TYPE_STATIC,
        AffectedByDevMode: true,  // Node hours
        ParentTagKeys:     []string{"vpc_id"},
        ParentType:        "aws:ec2:vpc:Vpc",
        Relationship:      "within",
    },
    "aws:rds:instance": {
        GrowthType:        pulumicostv1.GrowthType_GROWTH_TYPE_STATIC,
        AffectedByDevMode: true,  // Instance hours
        ParentTagKeys:     []string{"vpc_id"},
        ParentType:        "aws:ec2:vpc:Vpc",
        Relationship:      "within",
    },
}
```

---

## Constants

### Dev Mode Hours

**Purpose**: Defines hour multipliers for dev vs production cost calculations.

**Go Definition**:
```go
const (
    hoursPerMonthProd = 730  // 24 hours/day * 30 days (production default)
    hoursPerMonthDev  = 160  // 8 hours/day * 5 days/week * 4 weeks/month (development)
)
```

**Validation Rules**:
- Both constants are read-only
- Used only for time-based services (not usage-based)
- Calculation: `devCost = prodCost * hoursPerMonthDev / hoursPerMonthProd`

---

### Relationship Types

**Purpose**: Valid relationship values for CostAllocationLineage.

**Go Definition**:
```go
const (
    relationshipAttachedTo = "attached_to"  // Direct attachment (EBS → EC2)
    relationshipWithin     = "within"       // Containment (RDS → VPC)
    relationshipManagedBy  = "managed_by"  // Management relationship
)
```

**Validation Rules**:
- All are string constants
- Mapped from service classification at enrichment time
- Default to `relationshipWithin` if not specified in classification

---

## Data Flow

### GetProjectedCost Request Flow

```text
1. Receive ResourceDescriptor
   ├─ Check usage_profile field (feature detection)
   ├─ Extract resource_type
   └─ Extract tags (parent tags)

2. Calculate Cost (existing logic)
   └─ Returns GetProjectedCostResponse with cost data

3. Apply Dev Mode (if applicable)
   ├─ Check hasUsageProfile(req)
   ├─ Check req.usage_profile == DEVELOPMENT
   ├─ Lookup serviceClassifications[resource_type]
   ├─ If classification.AffectedByDevMode:
   │   ├─ resp.cost_per_month *= 160 / 730
   │   └─ resp.billing_detail += " (dev profile)"
   └─ Log INFO with usage_profile, resource_type

4. Set Growth Hint
   ├─ Check hasGrowthHint(resp)
   ├─ Lookup serviceClassifications[resource_type]
   ├─ resp.growth_type = classification.GrowthType
   └─ Log INFO with growth_type, resource_type

5. Extract Lineage (if applicable)
   ├─ Check hasLineage(resp)
   ├─ Lookup serviceClassifications[resource_type]
   ├─ For each tagKey in classification.ParentTagKeys:
   │   └─ If req.tags[tagKey] exists:
   │       ├─ resp.lineage = CostAllocationLineage{
   │       │     parent_resource_id: req.tags[tagKey],
   │       │     parent_resource_type: classification.ParentType,
   │       │     relationship: classification.Relationship,
   │       │   }
   │       ├─ Log INFO with parent_detected, parent_type, relationship
   │       └─ Break (use first match)
   └─ Return enriched GetProjectedCostResponse
```

---

## Validation Rules

### Input Validation (ResourceDescriptor)

1. **UsageProfile**: If field exists and set to invalid value, treat as UNSPECIFIED
2. **Tags**: Case-sensitive lookup for parent tag keys
3. **ResourceType**: Must match exact key in serviceClassifications map

### Output Validation (GetProjectedCostResponse)

1. **GrowthType**: Only set if field exists in proto, use GROWTH_TYPE_UNSPECIFIED for unknown resources
2. **Lineage**: Only set if parent_resource_id is non-empty string
3. **BillingDetail**: Only append " (dev profile)" if dev mode actually applied

### Feature Detection

```go
// Type assertions for field availability
func hasUsageProfile(req *pulumicostv1.ResourceDescriptor) bool {
    _, ok := req.(interface{ GetUsageProfile() pulumicostv1.UsageProfile })
    return ok
}

func hasGrowthHint(resp *pulumicostv1.GetProjectedCostResponse) bool {
    _, ok := resp.(interface{ SetGrowthType(pulumicostv1.GrowthType) })
    return ok
}

func hasLineage(resp *pulumicostv1.GetProjectedCostResponse) bool {
    _, ok := resp.(interface{ SetLineage(*pulumicostv1.CostAllocationLineage) })
    return ok
}
```

---

## Thread Safety

**Design Principle**: All data structures are read-only, no shared mutable state.

| Entity | Mutability | Thread Safety |
|---------|-------------|---------------|
| `serviceClassifications` map | Read-only constant | ✅ Safe (concurrent reads) |
| `hoursPerMonthProd` | Read-only constant | ✅ Safe |
| `hoursPerMonthDev` | Read-only constant | ✅ Safe |
| `relationship*` constants | Read-only constants | ✅ Safe |
| `GetProjectedCostResponse` | Created per request | ✅ Safe (no sharing) |

**No locks required**: All enrichment functions are pure transformations with no shared state.

---

## Performance Characteristics

| Operation | Time Complexity | Expected Latency |
|------------|------------------|-------------------|
| Service classification lookup | O(1) | < 100ns |
| Feature detection (type assertion) | O(1) | < 1μs |
| Parent tag extraction | O(n) where n=tag keys (≤3) | < 10μs |
| Field assignment | O(1) | < 1μs |
| Logging (if metadata applied) | O(1) | < 5ms |
| **Total enrichment overhead** | - | **< 10ms** |

---

## References

- Protocol definitions: pulumicost-spec (rshade/pulumicost-spec)
- Research findings: research.md (RQ-1 through RQ-10)
- Feature specification: spec.md
- Constitution compliance: .specify/memory/constitution.md
