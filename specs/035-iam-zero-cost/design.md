# Design: IAM Zero-Cost Resource Handling

## Data Model (Internal)

### Zero-Cost Resource Types Map
Location: `internal/plugin/validation.go`

Current:
```go
var zeroCostResourceTypes = map[string]bool{
    "vpc":           true,
    "securitygroup": true,
    "subnet":        true,
}
```

Proposed:
```go
var zeroCostResourceTypes = map[string]bool{
    "vpc":           true,
    "securitygroup": true,
    "subnet":        true,
    "iam":           true, // New entry
}
```

## API Contracts (gRPC Behavior)

### `Supports` Method

**Input**:
- `ResourceDescriptor.ResourceType`: Any string starting with `aws:iam/` (case-insensitive).

**Output**:
- `SupportsResponse.Supported`: `true`
- `SupportsResponse.Reason`: "" (empty implies supported)

### `GetProjectedCost` Method

**Input**:
- `ResourceDescriptor.ResourceType`: Any string starting with `aws:iam/` (case-insensitive).

**Output**:
```json
{
  "projected_cost": {
    "currency_code": "USD",
    "units": "1",
    "total_monthly_cost": 0.0,
    "billing_detail": "IAM - no direct AWS charges"
  }
}
```

## Logic Flow

1.  **Incoming Request**: `GetProjectedCost` receives `ResourceDescriptor`.
2.  **Normalization (`normalizeResourceType`)**:
    - Check if input starts with `aws:iam/` (case-insensitive).
    - If yes, return `"iam"`.
3.  **Service Detection (`detectService`)**:
    - Maps `"iam"` -> `"iam"`.
4.  **Cost Estimation (`GetProjectedCost`)**:
    - Switch case on service type.
    - Case `"iam"`: Call `estimateZeroCostResource`.
5.  **Zero Cost Helper (`estimateZeroCostResource`)**:
    - Returns constructed $0 response.
