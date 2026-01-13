# Contract: Supports() RPC

**Service**: `finfocus.v1.CostSourceService`
**Method**: `Supports`
**Purpose**: Determines if the plugin can estimate costs for a given resource

---

## RPC Signature

```protobuf
rpc Supports(SupportsRequest) returns (SupportsResponse);
```

---

## Request

```protobuf
message SupportsRequest {
  ResourceDescriptor resource = 1;
}

message ResourceDescriptor {
  string provider = 1;
  string resource_type = 2;
  string sku = 3;
  string region = 4;
  map<string, string> tags = 5;
}
```

**Required Fields**:
- `resource.provider` - Must be "aws"
- `resource.resource_type` - Service type (e.g., "ec2", "ebs")
- `resource.region` - AWS region

**Optional Fields** (not evaluated by Supports):
- `resource.sku` - Not checked by Supports (only by GetProjectedCost)
- `resource.tags` - Not checked by Supports

---

## Response

```protobuf
message SupportsResponse {
  bool supported = 1;
  string reason = 2;
}
```

**Contract**:
- `supported` = `true` if plugin can estimate costs for this resource
- `supported` = `false` if plugin cannot estimate costs
- `reason` MUST explain why `supported` is true or false

---

## Decision Logic

### Provider Check

```go
if req.Resource.Provider != "aws" {
    return &pbc.SupportsResponse{
        Supported: false,
        Reason:    "Plugin only supports AWS resources",
    }, nil
}
```

---

### Region Check

```go
if req.Resource.Region != p.region {
    return &pbc.SupportsResponse{
        Supported: false,
        Reason:    fmt.Sprintf("Region %s not supported by this binary (compiled for %s)",
                               req.Resource.Region, p.region),
    }, nil
}
```

**Critical**: This is how FinFocus core detects it needs a different region-specific binary.

---

### Resource Type Check

```go
switch req.Resource.ResourceType {
case "ec2", "ebs":
    return &pbc.SupportsResponse{
        Supported: true,
        Reason:    "Fully supported",
    }, nil

case "s3", "lambda", "rds", "dynamodb":
    return &pbc.SupportsResponse{
        Supported: true,
        Reason:    "Limited support - returns $0 estimate",
    }, nil

default:
    return &pbc.SupportsResponse{
        Supported: false,
        Reason:    fmt.Sprintf("Resource type %s not recognized", req.Resource.ResourceType),
    }, nil
}
```

---

## Success Criteria

- Returns response in < 10ms (SC-002)
- Never returns gRPC error (uses `supported=false` instead)
- `reason` is human-readable and explains the decision

---

## Error Cases

**None** - Supports() never returns gRPC errors.

Invalid or missing fields result in `supported=false` with explanatory `reason`.

---

## Testing

```bash
# grpcurl test - EC2 in correct region
grpcurl -plaintext \
  -d '{"resource": {"provider": "aws", "resource_type": "ec2", "region": "us-east-1"}}' \
  localhost:12345 \
  finfocus.v1.CostSourceService/Supports

# Expected response:
{
  "supported": true,
  "reason": "Fully supported"
}

# grpcurl test - Wrong region
grpcurl -plaintext \
  -d '{"resource": {"provider": "aws", "resource_type": "ec2", "region": "us-west-2"}}' \
  localhost:12345 \
  finfocus.v1.CostSourceService/Supports

# Expected response:
{
  "supported": false,
  "reason": "Region us-west-2 not supported by this binary (compiled for us-east-1)"
}
```

**Unit Tests**:
```go
func TestSupports_EC2(t *testing.T) {
    p := NewAWSPublicPlugin("us-east-1", &mockPricingClient{region: "us-east-1"})

    resp, err := p.Supports(context.Background(), &pbc.SupportsRequest{
        Resource: &pbc.ResourceDescriptor{
            Provider:     "aws",
            ResourceType: "ec2",
            Region:       "us-east-1",
        },
    })

    require.NoError(t, err)
    assert.True(t, resp.Supported)
    assert.Contains(t, resp.Reason, "Fully supported")
}

func TestSupports_WrongRegion(t *testing.T) {
    p := NewAWSPublicPlugin("us-east-1", &mockPricingClient{region: "us-east-1"})

    resp, err := p.Supports(context.Background(), &pbc.SupportsRequest{
        Resource: &pbc.ResourceDescriptor{
            Provider:     "aws",
            ResourceType: "ec2",
            Region:       "us-west-2",
        },
    })

    require.NoError(t, err)
    assert.False(t, resp.Supported)
    assert.Contains(t, resp.Reason, "not supported by this binary")
}

func TestSupports_StubService(t *testing.T) {
    p := NewAWSPublicPlugin("us-east-1", &mockPricingClient{region: "us-east-1"})

    resp, err := p.Supports(context.Background(), &pbc.SupportsRequest{
        Resource: &pbc.ResourceDescriptor{
            Provider:     "aws",
            ResourceType: "s3",
            Region:       "us-east-1",
        },
    })

    require.NoError(t, err)
    assert.True(t, resp.Supported)
    assert.Contains(t, resp.Reason, "Limited support")
}
```

---

## Usage by FinFocus Core

1. Core analyzes Pulumi stack and extracts resources
2. For each resource, core calls Supports() to check compatibility
3. If `supported=false` due to region mismatch:
   - Core fetches or starts the appropriate region-specific binary
   - Core retries the request with the correct binary
4. If `supported=false` for other reasons:
   - Core skips cost estimation for that resource
   - Core logs the `reason` for user visibility
5. If `supported=true`:
   - Core proceeds to call GetProjectedCost()

**Frequency**: Once per resource in the stack (before GetProjectedCost)

**Performance Requirement**: <10ms per call to avoid stack analysis slowdown
