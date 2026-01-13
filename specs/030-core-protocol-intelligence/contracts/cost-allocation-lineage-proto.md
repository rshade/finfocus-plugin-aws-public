# Protocol Extension: CostAllocationLineage Message

**Feature**: 030-core-protocol-intelligence (P3 - Topology Linking)
**Status**: PROPOSED - Requires PR to rshade/finfocus-spec
**Date**: 2026-01-05

## Purpose

Add `CostAllocationLineage` message to `GetProjectedCostResponse` to enable Blast Radius topology visualization. This allows Core to build parent/child dependency graphs for impact analysis.

## Proto Definition

```protobuf
// CostAllocationLineage describes parent/child relationships for topology visualization
message CostAllocationLineage {
  // ID of parent resource (e.g., "i-abc123", "vpc-xyz")
  string parent_resource_id = 1;

  // Type of parent resource (full type string, e.g., "aws:ec2:instance:Instance")
  string parent_resource_type = 2;

  // Relationship type describing how this resource relates to its parent
  // Valid values: "attached_to", "within", "managed_by"
  string relationship = 3;
}
```

## Integration

Add to `GetProjectedCostResponse` message:

```protobuf
message GetProjectedCostResponse {
  // ... existing fields ...

  // OPTIONAL: Parent-child lineage information for topology visualization
  // Omitted if no parent relationship detected or field unavailable
  CostAllocationLineage lineage = 8;
}
```

## Backward Compatibility

- Message is **optional** - entire field omitted if not applicable
- All fields are **optional strings** - no breaking changes to response structure
- Existing plugins without this field will simply omit `lineage`
- No breaking changes to existing `GetProjectedCostResponse` structure

## Implementation Notes

### Parent Tag Keys per Service

| Service | Tag Key | Relationship | Parent Type | Priority |
|---------|----------|--------------|---------------|-----------|
| EBS volume | `instance_id` | `attached_to` | aws:ec2:instance:Instance | 1 |
| ELB | `vpc_id` | `within` | aws:ec2:vpc:Vpc | 2 |
| NAT Gateway | `vpc_id` | `within` | aws:ec2:vpc:Vpc | 2 |
| NAT Gateway | `subnet_id` | `within` | aws:ec2:subnet:Subnet | 3 |
| ElastiCache | `vpc_id` | `within` | aws:ec2:vpc:Vpc | 2 |
| RDS | `vpc_id` | `within` | aws:ec2:vpc:Vpc | 2 |

### Relationship Types

| Relationship | Description | Example |
|--------------|-------------|-----------|
| `attached_to` | Direct attachment (child attaches to parent) | EBS volume → EC2 instance |
| `within` | Containment (child exists within parent's scope) | RDS instance → VPC |
| `managed_by` | Management relationship (child managed by parent) | Reserved for future use |

### Extraction Logic

```go
// For each resource type, check parent tag keys in priority order
// Use first non-empty tag value

func extractLineage(serviceType string, tags map[string]string) *CostAllocationLineage {
    classification := serviceClassifications[serviceType]

    for _, tagKey := range classification.ParentTagKeys {
        if parentID, exists := tags[tagKey]; exists && parentID != "" {
            return &CostAllocationLineage{
                ParentResourceId:   parentID,
                ParentResourceType: classification.ParentType,
                Relationship:       classification.Relationship,
            }
        }
    }

    return nil // No parent found
}
```

### Tag Key Priority

When multiple parent tags exist (e.g., NAT Gateway with both `vpc_id` and `subnet_id`):

1. Check tags in order defined by priority
2. Use first non-empty match
3. Stop after first match (single parent only)

**Rationale**: Provides deterministic, predictable behavior for topology graph construction.

## Example Usage

### Request

```protobuf
{
  "resource_type": "aws:ebs:volume:Volume",
  "region": "us-east-1",
  "volume_type": "gp2",
  "size_gb": 100,
  "tags": {
    "instance_id": "i-abc123",
    "environment": "production"
  }
}
```

### Response

```protobuf
{
  "unit_price": 0.10,
  "currency": "USD",
  "cost_per_month": 10.00,
  "billing_detail": "$0.10/GB × 100 GB",
  "lineage": {
    "parent_resource_id": "i-abc123",
    "parent_resource_type": "aws:ec2:instance:Instance",
    "relationship": "attached_to"
  }
}
```

### Complex Example (NAT Gateway)

```protobuf
{
  "resource_type": "aws:ec2:nat-gateway:NatGateway",
  "region": "us-east-1",
  "tags": {
    "vpc_id": "vpc-xyz",
    "subnet_id": "subnet-123"
  }
}
```

```protobuf
{
  "lineage": {
    "parent_resource_id": "vpc-xyz",  // vpc_id has priority
    "parent_resource_type": "aws:ec2:vpc:Vpc",
    "relationship": "within"
  }
}
```

## Testing

### Unit Tests

```go
func TestExtractLineage(t *testing.T) {
    tests := []struct {
        name          string
        service       string
        tags          map[string]string
        expected      *CostAllocationLineage
    }{
        {
            name:    "EBS with instance_id",
            service: "aws:ebs:volume:Volume",
            tags:    map[string]string{"instance_id": "i-abc123"},
            expected: &CostAllocationLineage{
                ParentResourceId:   "i-abc123",
                ParentResourceType: "aws:ec2:instance:Instance",
                Relationship:       "attached_to",
            },
        },
        {
            name:    "EBS without tags",
            service: "aws:ebs:volume:Volume",
            tags:    map[string]string{},
            expected: nil,
        },
        {
            name:    "NAT with vpc_id and subnet_id",
            service: "aws:ec2:nat-gateway:NatGateway",
            tags:    map[string]string{"vpc_id": "vpc-xyz", "subnet_id": "subnet-123"},
            expected: &CostAllocationLineage{
                ParentResourceId:   "vpc-xyz",  // vpc_id has priority
                ParentResourceType: "aws:ec2:vpc:Vpc",
                Relationship:       "within",
            },
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := extractLineage(tt.service, tt.tags)
            assert.Equal(t, tt.expected, result)
        })
    }
}
```

### Integration Tests

```go
func TestLineageInGRPCResponse(t *testing.T) {
    req := &pbc.ResourceDescriptor{
        ResourceType: "aws:ebs:volume:Volume",
        Tags:        map[string]string{"instance_id": "i-abc123"},
    }

    resp, err := plugin.GetProjectedCost(req)
    assert.NoError(t, err)
    assert.NotNil(t, resp.Lineage)
    assert.Equal(t, "i-abc123", resp.Lineage.ParentResourceId)
    assert.Equal(t, "attached_to", resp.Lineage.Relationship)
}
```

## References

- Feature spec: spec.md (User Story 3 - Resource Topology Linking)
- Data model: data-model.md (CostAllocationLineage message)
- Implementation guide: quickstart.md (Phase 5: Topology Linking)
