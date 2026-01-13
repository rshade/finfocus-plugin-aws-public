# Data Model: Fallback GetActualCost

**Date**: 2025-11-25
**Feature**: 004-actual-cost-fallback

## Overview

This feature uses existing proto-defined entities from `finfocus.v1`.
No new entities are created; this document describes the relevant
structures and their usage in the GetActualCost flow.

## Entities

### GetActualCostRequest (proto-defined, v0.3.0)

**Source**: `finfocus.v1.GetActualCostRequest`

| Field       | Type                      | Description                    |
| ----------- | ------------------------- | ------------------------------ |
| resource_id | string                    | JSON-encoded ResourceDescriptor|
| start       | google.protobuf.Timestamp | Start of time range            |
| end         | google.protobuf.Timestamp | End of time range              |
| tags        | map<string, string>       | Alternative resource info      |

**Resource identification**: The `resource_id` field contains a JSON-encoded
ResourceDescriptor with provider, resource_type, sku, region, and tags.

**Validation rules**:

- `resource_id` MUST be valid JSON containing ResourceDescriptor fields
- Parsed `provider` MUST be "aws"
- Parsed `resource_type` MUST be one of: ec2, ebs, s3, lambda, rds, dynamodb
- Parsed `sku` MUST NOT be empty (instance type or volume type)
- Parsed `region` MUST match plugin binary region
- `start` MUST NOT be nil
- `end` MUST NOT be nil
- `start` MUST be before or equal to `end`

### GetActualCostResponse (proto-defined, v0.3.0)

**Source**: `finfocus.v1.GetActualCostResponse`

| Field   | Type                       | Description                     |
| ------- | -------------------------- | ------------------------------- |
| results | repeated ActualCostResult  | Array of cost results           |

### ActualCostResult (proto-defined, v0.3.0)

**Source**: `finfocus.v1.ActualCostResult`

| Field        | Type                      | Description                   |
| ------------ | ------------------------- | ----------------------------- |
| timestamp    | google.protobuf.Timestamp | Start of calculation period   |
| cost         | double                    | Calculated actual cost        |
| usage_amount | double                    | Runtime hours                 |
| usage_unit   | string                    | "hours"                       |
| source       | string                    | Billing detail / calc basis   |

**Response states** (single result in array):

| Condition     | cost | source field contains              |
| ------------- | ---- | ---------------------------------- |
| Valid EC2/EBS | > 0  | Fallback formula explanation       |
| Stub service  | 0.0  | Service not implemented message    |
| Zero duration | 0.0  | "aws-public-fallback"              |
| Unknown SKU   | 0.0  | SKU not found explanation          |

### ResourceDescriptor (proto-defined)

**Source**: `finfocus.v1.ResourceDescriptor`

| Field         | Type                 | Description                         |
| ------------- | -------------------- | ----------------------------------- |
| provider      | string               | Cloud provider ("aws")              |
| resource_type | string               | Resource category (ec2, ebs, etc.)  |
| sku           | string               | Specific type (t3.micro, gp3, etc.) |
| region        | string               | AWS region (us-east-1, etc.)        |
| tags          | map<string, string>  | Optional metadata (size for EBS)    |

## Internal Structures

### Runtime Calculation (internal)

Not a persisted entity - computed values during request processing:

```go
type runtimeCalc struct {
    startTime    time.Time  // Converted from proto Start timestamp
    endTime      time.Time  // Converted from proto End timestamp
    durationHrs  float64    // Calculated runtime in hours
    monthlyRate  float64    // From GetProjectedCost
    actualCost   float64    // durationHrs / 730 * monthlyRate
}
```

## Data Flow

```text
GetActualCostRequest
        │
        ▼
┌───────────────────┐
│ Parse ResourceId  │──► InvalidArgument error (invalid JSON)
│ - JSON decode     │
│ - Extract fields  │
└───────────────────┘
        │
        ▼
┌───────────────────┐
│ Validate Request  │──► InvalidArgument error
│ - nil checks      │
│ - region match    │──► UnsupportedRegion error
│ - time range      │──► InvalidArgument error
└───────────────────┘
        │
        ▼
┌───────────────────┐
│ Parse Timestamps  │
│ - Start.AsTime()  │
│ - End.AsTime()    │
│ - Calculate hours │
└───────────────────┘
        │
        ▼
┌───────────────────┐
│ Get Monthly Rate  │
│ - Route by type   │
│ - Lookup pricing  │
└───────────────────┘
        │
        ▼
┌───────────────────┐
│ Apply Formula     │
│ cost = monthly *  │
│ (hours / 730)     │
└───────────────────┘
        │
        ▼
GetActualCostResponse
(Results array with
 single ActualCostResult)
```

## Relationships

```text
GetActualCostRequest ──contains──► resource_id (JSON-encoded ResourceDescriptor)
                     ──contains──► Timestamp (start)
                     ──contains──► Timestamp (end)
                     ──contains──► tags (fallback resource info)

GetActualCostResponse ──contains──► ActualCostResult[]
                                    (single result for fallback)

ActualCostResult ──derived from──► GetProjectedCostResponse
                                   (monthly rate used in formula)
```

## State Transitions

N/A - This is a stateless calculation. Each RPC call is independent with
no persisted state between calls.
