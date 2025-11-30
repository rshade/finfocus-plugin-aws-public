# Contract: Name() RPC

**Service**: `pulumicost.v1.CostSourceService`
**Method**: `Name`
**Purpose**: Returns the plugin identifier for routing and discovery

---

## RPC Signature

```protobuf
rpc Name(NameRequest) returns (NameResponse);
```

---

## Request

```protobuf
message NameRequest {
  // Empty - no parameters needed
}
```

**Validation**: None required (empty message)

---

## Response

```protobuf
message NameResponse {
  string name = 1;  // Plugin identifier
}
```

**Contract**:
- `name` MUST be `"aws-public"` for this plugin
- `name` MUST be consistent across all invocations
- `name` MUST match the plugin identifier expected by PulumiCost core

---

## Implementation

```go
func (p *AWSPublicPlugin) Name(
    ctx context.Context,
    req *pbc.NameRequest,
) (*pbc.NameResponse, error) {
    return &pbc.NameResponse{
        Name: "aws-public",
    }, nil
}
```

---

## Success Criteria

- Returns `NameResponse{name: "aws-public"}` in < 10ms
- Never returns an error
- Response is identical for all region-specific binaries

---

## Error Cases

**None** - this RPC always succeeds.

---

## Testing

```bash
# grpcurl test
grpcurl -plaintext localhost:12345 pulumicost.v1.CostSourceService/Name

# Expected response:
{
  "name": "aws-public"
}
```

**Unit Test**:
```go
func TestName(t *testing.T) {
    p := NewAWSPublicPlugin("us-west-1", &mockPricingClient{})

    resp, err := p.Name(context.Background(), &pbc.NameRequest{})

    require.NoError(t, err)
    assert.Equal(t, "aws-public", resp.Name)
}
```

---

## Usage by PulumiCost Core

1. Core starts the plugin subprocess
2. Core reads PORT from stdout
3. Core connects gRPC client
4. Core calls Name() to verify plugin identity
5. Core uses the name for logging and routing

**Frequency**: Once per plugin instance (on startup)