# Contract: GetProjectedCost() RPC

**Service**: `finfocus.v1.CostSourceService`
**Method**: `GetProjectedCost`
**Purpose**: Estimates monthly cost for a single AWS resource using public pricing

---

## RPC Signature

```protobuf
rpc GetProjectedCost(GetProjectedCostRequest) returns (GetProjectedCostResponse);
```

---

## Request

```protobuf
message GetProjectedCostRequest {
  ResourceDescriptor resource = 1;
}

message ResourceDescriptor {
  string provider = 1;       // "aws"
  string resource_type = 2;  // "ec2", "ebs", "s3", etc.
  string sku = 3;            // Instance type or volume type
  string region = 4;         // AWS region
  map<string, string> tags = 5;  // Additional metadata
}
```

**Required Fields**:
- `resource.provider` - MUST be "aws"
- `resource.resource_type` - Service type (e.g., "ec2", "ebs")
- `resource.sku` - SKU identifier (instance type for EC2, volume type for EBS)
- `resource.region` - MUST match plugin's compiled region

**Optional Fields**:
- `resource.tags` - Used for EBS size lookup (keys: "size" or "volume_size")

---

## Response

```protobuf
message GetProjectedCostResponse {
  double unit_price = 1;      // Per-unit rate ($/hour or $/GB-month)
  string currency = 2;        // Currency code
  double cost_per_month = 3;  // Estimated monthly cost
  string billing_detail = 4;  // Human-readable assumptions
}
```

**Contract**:
- `unit_price` - The rate per unit (e.g., $/hour for EC2, $/GB-month for EBS)
- `currency` - MUST be "USD" for v1
- `cost_per_month` - Calculated monthly cost based on assumptions
- `billing_detail` - MUST document all assumptions (hours/month, OS, tenancy, etc.)

---

## Validation & Errors

### Invalid Provider

```go
if req.Resource.Provider != "aws" {
    return nil, status.Error(codes.InvalidArgument, "provider must be aws")
}
```

**gRPC Error**: `InvalidArgument` (code 3)

---

### Missing Required Fields

```go
if req.Resource.ResourceType == "" {
    return nil, status.Error(codes.InvalidArgument, "missing resource_type")
}

if req.Resource.Region == "" {
    return nil, status.Error(codes.InvalidArgument, "missing region")
}

if req.Resource.Sku == "" && req.Resource.ResourceType == "ec2" {
    return nil, status.Error(codes.InvalidArgument, "EC2 resource missing sku (instance type)")
}
```

**gRPC Error**: `InvalidArgument` (code 3)

---

### Region Mismatch

```go
if req.Resource.Region != p.region {
    st := status.New(codes.FailedPrecondition, fmt.Sprintf(
        "Resource in region %s but plugin compiled for %s",
        req.Resource.Region, p.region,
    ))
    // TODO: Add ErrorDetail with proto ErrorCode enum
    return nil, st.Err()
}
```

**gRPC Error**: `FailedPrecondition` (code 9)
**Proto Error Code**: `ERROR_CODE_UNSUPPORTED_REGION` (9)

**Details Map** (future enhancement):
```go
{
    "pluginRegion": "us-east-1",
    "requiredRegion": "us-west-2"
}
```

**Usage**: FinFocus core detects this error and fetches the correct region binary.

---

### Pricing Data Corruption

```go
if err := p.pricing.init(); err != nil {
    return nil, status.Error(codes.Internal, "failed to load pricing data")
}
```

**gRPC Error**: `Internal` (code 13)
**Proto Error Code**: `ERROR_CODE_DATA_CORRUPTION` (11)

---

## Resource-Specific Estimation

### EC2 Instances

**Input**:
```json
{
  "resource": {
    "provider": "aws",
    "resource_type": "ec2",
    "sku": "t3.micro",
    "region": "us-east-1"
  }
}
```

**Assumptions**:
- Operating System: Linux
- Tenancy: Shared
- Hours per Month: 730 (24×7 on-demand)
- Pricing Model: On-demand

**Calculation**:
```
monthlyCost = hourlyRate * 730
            = 0.0104 * 730
            = 7.592
```

**Response**:
```json
{
  "unit_price": 0.0104,
  "currency": "USD",
  "cost_per_month": 7.592,
  "billing_detail": "On-demand Linux/Shared instance, t3.micro, 730 hrs/month"
}
```

**Pricing Not Found**:
```json
{
  "unit_price": 0,
  "currency": "USD",
  "cost_per_month": 0,
  "billing_detail": "EC2 t3.mega pricing not found in region us-east-1"
}
```

---

### EBS Volumes

**Input with Size**:
```json
{
  "resource": {
    "provider": "aws",
    "resource_type": "ebs",
    "sku": "gp3",
    "region": "us-east-1",
    "tags": {
      "size": "100"
    }
  }
}
```

**Calculation**:
```
monthlyCost = sizeGB * ratePerGBMonth
            = 100 * 0.08
            = 8.0
```

**Response**:
```json
{
  "unit_price": 0.08,
  "currency": "USD",
  "cost_per_month": 8.0,
  "billing_detail": "EBS gp3 storage, 100GB"
}
```

---

**Input without Size** (uses default):
```json
{
  "resource": {
    "provider": "aws",
    "resource_type": "ebs",
    "sku": "gp3",
    "region": "us-east-1"
  }
}
```

**Calculation**:
```
sizeGB = 8 (default)
monthlyCost = 8 * 0.08 = 0.64
```

**Response**:
```json
{
  "unit_price": 0.08,
  "currency": "USD",
  "cost_per_month": 0.64,
  "billing_detail": "EBS gp3 storage, 8GB, defaulted to 8GB"
}
```

**Note**: `billing_detail` documents that size was defaulted.

---

### Stub Services (S3, Lambda, RDS, DynamoDB)

**Input**:
```json
{
  "resource": {
    "provider": "aws",
    "resource_type": "s3",
    "region": "us-east-1"
  }
}
```

**Response**:
```json
{
  "unit_price": 0,
  "currency": "USD",
  "cost_per_month": 0,
  "billing_detail": "S3 cost estimation not implemented - returning $0"
}
```

**Rationale**: Stub services return $0 to acknowledge the resource without failing. This gives users a complete view of their stack.

---

## Success Criteria

- Response time < 100ms per call (SC-001)
- All responses include non-empty `billing_detail` (SC-008)
- EC2 costs within 5% of actual AWS pricing (SC-005)
- EBS costs within 2% of actual AWS pricing (SC-006)
- Handles 100+ concurrent calls without errors (SC-004)

---

## Testing

```bash
# grpcurl test - EC2
grpcurl -plaintext \
  -d '{
    "resource": {
      "provider": "aws",
      "resource_type": "ec2",
      "sku": "t3.micro",
      "region": "us-east-1"
    }
  }' \
  localhost:12345 \
  finfocus.v1.CostSourceService/GetProjectedCost

# Expected response:
{
  "unitPrice": 0.0104,
  "currency": "USD",
  "costPerMonth": 7.592,
  "billingDetail": "On-demand Linux/Shared instance, t3.micro, 730 hrs/month"
}

# grpcurl test - EBS with size
grpcurl -plaintext \
  -d '{
    "resource": {
      "provider": "aws",
      "resource_type": "ebs",
      "sku": "gp3",
      "region": "us-east-1",
      "tags": {"size": "100"}
    }
  }' \
  localhost:12345 \
  finfocus.v1.CostSourceService/GetProjectedCost

# Expected response:
{
  "unitPrice": 0.08,
  "currency": "USD",
  "costPerMonth": 8.0,
  "billingDetail": "EBS gp3 storage, 100GB"
}

# grpcurl test - Region mismatch
grpcurl -plaintext \
  -d '{
    "resource": {
      "provider": "aws",
      "resource_type": "ec2",
      "sku": "t3.micro",
      "region": "us-west-2"
    }
  }' \
  localhost:12345 \
  finfocus.v1.CostSourceService/GetProjectedCost

# Expected error:
ERROR:
  Code: FailedPrecondition
  Message: Resource in region us-west-2 but plugin compiled for us-east-1
```

**Unit Tests**:
```go
func TestGetProjectedCost_EC2(t *testing.T) {
    pricing := &mockPricingClient{
        region:   "us-east-1",
        currency: "USD",
        ec2Prices: map[string]float64{
            "t3.micro/Linux/Shared": 0.0104,
        },
    }
    p := NewAWSPublicPlugin("us-east-1", pricing)

    resp, err := p.GetProjectedCost(context.Background(), &pbc.GetProjectedCostRequest{
        Resource: &pbc.ResourceDescriptor{
            Provider:     "aws",
            ResourceType: "ec2",
            Sku:          "t3.micro",
            Region:       "us-east-1",
        },
    })

    require.NoError(t, err)
    require.NotNil(t, resp)

    assert.Equal(t, 0.0104, resp.UnitPrice)
    assert.Equal(t, "USD", resp.Currency)
    assert.InDelta(t, 7.592, resp.CostPerMonth, 0.01)
    assert.Contains(t, resp.BillingDetail, "On-demand")
    assert.Contains(t, resp.BillingDetail, "t3.micro")
}

func TestGetProjectedCost_EBS_DefaultSize(t *testing.T) {
    pricing := &mockPricingClient{
        region:   "us-east-1",
        currency: "USD",
        ebsPrices: map[string]float64{
            "gp3": 0.08,
        },
    }
    p := NewAWSPublicPlugin("us-east-1", pricing)

    resp, err := p.GetProjectedCost(context.Background(), &pbc.GetProjectedCostRequest{
        Resource: &pbc.ResourceDescriptor{
            Provider:     "aws",
            ResourceType: "ebs",
            Sku:          "gp3",
            Region:       "us-east-1",
            // No tags - should default to 8GB
        },
    })

    require.NoError(t, err)
    assert.Equal(t, 0.08, resp.UnitPrice)
    assert.Equal(t, 0.64, resp.CostPerMonth) // 8 * 0.08
    assert.Contains(t, resp.BillingDetail, "defaulted to 8GB")
}

func TestGetProjectedCost_RegionMismatch(t *testing.T) {
    pricing := &mockPricingClient{region: "us-east-1", currency: "USD"}
    p := NewAWSPublicPlugin("us-east-1", pricing)

    _, err := p.GetProjectedCost(context.Background(), &pbc.GetProjectedCostRequest{
        Resource: &pbc.ResourceDescriptor{
            Provider:     "aws",
            ResourceType: "ec2",
            Sku:          "t3.micro",
            Region:       "us-west-2", // Mismatch!
        },
    })

    require.Error(t, err)
    st, ok := status.FromError(err)
    require.True(t, ok)
    assert.Equal(t, codes.FailedPrecondition, st.Code())
    assert.Contains(t, st.Message(), "us-west-2")
    assert.Contains(t, st.Message(), "us-east-1")
}
```

---

## Usage by FinFocus Core

1. Core calls Supports() first to check compatibility
2. If `supported=true`, core calls GetProjectedCost() for each resource
3. Core aggregates `cost_per_month` across all resources
4. Core displays total estimated monthly cost to user
5. Core may display `billing_detail` for individual resources in detailed view

**Frequency**: Once per resource in the stack (after Supports check)

**Concurrency**: Core may call GetProjectedCost() concurrently for multiple resources

**Timeout**: Core may set RPC timeout (e.g., 5 seconds) to prevent hangs

---

## Performance Characteristics

**Latency Breakdown**:
- Input validation: <1ms
- Pricing lookup (O(1) map access): <1ms
- Cost calculation: <1ms
- Proto serialization: <5ms
- **Total**: <10ms (well under 100ms target)

**Memory**: O(1) per request (no allocations except response proto)

**Thread Safety**: All pricing lookups are thread-safe (maps are read-only after initialization)
