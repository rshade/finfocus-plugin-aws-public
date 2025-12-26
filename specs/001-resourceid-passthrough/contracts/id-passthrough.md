# Contract: Resource ID Passthrough

**Feature**: 001-resourceid-passthrough
**Date**: 2025-12-26
**Protocol**: gRPC (pulumicost.v1.CostSourceService)

## Contract Summary

This document defines the expected behavior for resource ID passthrough in the `GetRecommendations` RPC method.

## Proto Reference

From `pulumicost-spec v0.4.11+`:

```protobuf
message ResourceDescriptor {
  string id = 7;              // NEW: Unique resource identifier
  string provider = 1;
  string resource_type = 2;
  string sku = 3;
  string region = 4;
  map<string, string> tags = 5;
}

message ResourceRecommendationInfo {
  string id = 1;              // Populated from ResourceDescriptor.id
  string name = 2;            // Populated from tags["name"]
  string provider = 3;
  string resource_type = 4;
  string region = 5;
  string sku = 6;
}
```

## Behavioral Contract

### Input → Output Mapping

| Input Condition | Output `Resource.Id` |
|-----------------|----------------------|
| `ResourceDescriptor.id = "abc-123"` | `"abc-123"` |
| `ResourceDescriptor.id = ""`, `tags["resource_id"] = "xyz"` | `"xyz"` |
| `ResourceDescriptor.id = "   "` (whitespace), `tags["resource_id"] = "xyz"` | `"xyz"` |
| `ResourceDescriptor.id = "native"`, `tags["resource_id"] = "tag"` | `"native"` |
| `ResourceDescriptor.id = ""`, no `resource_id` tag | `""` |

### Invariants

1. **Deterministic**: Same input always produces same `Resource.Id` output
2. **Idempotent**: Repeated calls with same input yield identical results
3. **Independent**: ID resolution for one resource doesn't affect others in batch
4. **Consistent**: All recommendations from same resource have identical `Resource.Id`

## Test Scenarios

### Scenario 1: Native ID Passthrough

```text
Input:
  ResourceDescriptor:
    id: "urn:pulumi:dev::myproject::aws:ec2/instance:Instance::web-server"
    sku: "m5.large"
    tags: {}

Expected Output:
  Recommendation.Resource.Id: "urn:pulumi:dev::myproject::aws:ec2/instance:Instance::web-server"
```

### Scenario 2: Tag Fallback

```text
Input:
  ResourceDescriptor:
    id: ""
    sku: "m5.large"
    tags: {"resource_id": "legacy-resource-123"}

Expected Output:
  Recommendation.Resource.Id: "legacy-resource-123"
```

### Scenario 3: Native ID Priority

```text
Input:
  ResourceDescriptor:
    id: "native-id"
    sku: "m5.large"
    tags: {"resource_id": "tag-id"}

Expected Output:
  Recommendation.Resource.Id: "native-id"
```

### Scenario 4: Whitespace Handling

```text
Input:
  ResourceDescriptor:
    id: "   "
    sku: "m5.large"
    tags: {"resource_id": "fallback-id"}

Expected Output:
  Recommendation.Resource.Id: "fallback-id"
```

### Scenario 5: No ID Available

```text
Input:
  ResourceDescriptor:
    id: ""
    sku: "m5.large"
    tags: {}

Expected Output:
  Recommendation.Resource.Id: ""
```

## Error Conditions

No new error conditions. ID passthrough is best-effort:

- Missing ID does not cause error (recommendation still generated)
- Invalid ID format is passed through as-is (validation is caller's responsibility)
