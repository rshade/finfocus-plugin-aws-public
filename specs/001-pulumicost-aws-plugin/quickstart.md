# Quickstart: PulumiCost AWS Public Plugin Development

**Phase**: 1 - Design
**Date**: 2025-11-16
**Audience**: Developers implementing the plugin

---

## Prerequisites

1. **Go 1.21+** installed
2. **Git** for cloning dependencies
3. **grpcurl** for manual gRPC testing
4. **make** for build automation

---

## Project Setup

### 1. Initialize Go Module

```bash
cd /mnt/c/GitHub/go/src/github.com/rshade/pulumicost-plugin-aws-public

go mod init github.com/rshade/pulumicost-plugin-aws-public
```

---

### 2. Add Dependencies

```bash
# Add proto definitions
go get github.com/rshade/pulumicost-spec/sdk/go/proto/pulumicost/v1

# Add plugin SDK
go get github.com/rshade/pulumicost-core/pkg/pluginsdk

# Add gRPC and protobuf
go get google.golang.org/grpc
go get google.golang.org/protobuf
```

---

### 3. Create Directory Structure

```bash
mkdir -p cmd/pulumicost-plugin-aws-public
mkdir -p internal/plugin
mkdir -p internal/pricing
mkdir -p internal/config
mkdir -p tools/generate-pricing
mkdir -p data
```

---

## Implementation Order

Follow this order to build the plugin incrementally with tests at each step:

### Phase 1: Pricing Client (internal/pricing)

**Why first**: Foundation for all cost calculations. Can be unit tested without gRPC.

**Files to create**:
1. `internal/pricing/types.go` - Define pricingData, ec2OnDemandPrice, ebsVolumePrice
2. `internal/pricing/client.go` - Implement PricingClient interface with sync.Once
3. `internal/pricing/embed_fallback.go` - Dummy data for development
4. `internal/pricing/client_test.go` - Unit tests for parsing and lookups

**Test first**:
```go
// internal/pricing/client_test.go
func TestClient_EC2Lookup(t *testing.T) {
    client, err := NewClient()
    require.NoError(t, err)

    price, found := client.EC2OnDemandPricePerHour("t3.micro", "Linux", "Shared")
    assert.True(t, found)
    assert.Greater(t, price, 0.0)
}
```

**Run**:
```bash
go test ./internal/pricing -v
```

---

### Phase 2: Plugin Struct (internal/plugin)

**Why second**: Defines the gRPC service implementation structure.

**Files to create**:
1. `internal/plugin/plugin.go` - Define AWSPublicPlugin struct and constructor
2. `internal/plugin/name.go` - Implement Name() RPC
3. `internal/plugin/name_test.go` - Unit test for Name()

**Test first**:
```go
// internal/plugin/name_test.go
func TestName(t *testing.T) {
    p := NewAWSPublicPlugin("us-east-1", &mockPricingClient{})

    resp, err := p.Name(context.Background(), &pbc.NameRequest{})

    require.NoError(t, err)
    assert.Equal(t, "aws-public", resp.Name)
}
```

**Run**:
```bash
go test ./internal/plugin -v
```

---

### Phase 3: Supports RPC (internal/plugin)

**Why third**: Simpler than GetProjectedCost, validates provider/region/resource_type logic.

**Files to create**:
1. `internal/plugin/supports.go` - Implement Supports() RPC
2. `internal/plugin/supports_test.go` - Table-driven tests for all cases

**Test first**:
```go
// internal/plugin/supports_test.go
func TestSupports(t *testing.T) {
    tests := []struct {
        name          string
        resourceType  string
        region        string
        wantSupported bool
        wantReason    string
    }{
        {"EC2 in correct region", "ec2", "us-east-1", true, "Fully supported"},
        {"EC2 in wrong region", "ec2", "us-west-2", false, "not supported by this binary"},
        {"S3 stub", "s3", "us-east-1", true, "Limited support"},
        {"Unknown type", "unknown", "us-east-1", false, "not recognized"},
    }

    p := NewAWSPublicPlugin("us-east-1", &mockPricingClient{region: "us-east-1"})

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            resp, err := p.Supports(context.Background(), &pbc.SupportsRequest{
                Resource: &pbc.ResourceDescriptor{
                    Provider:     "aws",
                    ResourceType: tt.resourceType,
                    Region:       tt.region,
                },
            })

            require.NoError(t, err)
            assert.Equal(t, tt.wantSupported, resp.Supported)
            assert.Contains(t, resp.Reason, tt.wantReason)
        })
    }
}
```

**Run**:
```bash
go test ./internal/plugin -v
```

---

### Phase 4: GetProjectedCost RPC (internal/plugin)

**Why fourth**: Core cost estimation logic. Uses pricing client from Phase 1.

**Files to create**:
1. `internal/plugin/projected.go` - Implement GetProjectedCost() RPC with estimateEC2, estimateEBS, estimateStub
2. `internal/plugin/projected_test.go` - Table-driven tests for EC2, EBS, stubs, errors

**Test first**:
```go
// internal/plugin/projected_test.go
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
    assert.InDelta(t, 7.592, resp.CostPerMonth, 0.01)
    assert.Contains(t, resp.BillingDetail, "On-demand")
}
```

**Run**:
```bash
go test ./internal/plugin -v
```

---

### Phase 5: Main Entrypoint (cmd/)

**Why fifth**: Ties everything together with pluginsdk.Serve().

**Files to create**:
1. `cmd/pulumicost-plugin-aws-public/main.go` - Initialize pricing client, create plugin, call pluginsdk.Serve()

**Implementation**:
```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/rshade/pulumicost-core/pkg/pluginsdk"
    "github.com/rshade/pulumicost-plugin-aws-public/internal/plugin"
    "github.com/rshade/pulumicost-plugin-aws-public/internal/pricing"
)

func main() {
    // Initialize pricing client
    pricingClient, err := pricing.NewClient()
    if err != nil {
        fmt.Fprintf(os.Stderr, "[pulumicost-plugin-aws-public] Failed to initialize pricing: %v\n", err)
        os.Exit(1)
    }

    // Log initialization to stderr
    fmt.Fprintf(os.Stderr, "[pulumicost-plugin-aws-public] Initialized for region: %s\n", pricingClient.Region())

    // Create plugin
    p := plugin.NewAWSPublicPlugin(pricingClient.Region(), pricingClient)

    // Serve gRPC
    ctx := context.Background()
    if err := pluginsdk.Serve(ctx, pluginsdk.ServeConfig{
        Plugin: p,
        Port:   0, // 0 = use PORT env or ephemeral
    }); err != nil {
        log.Fatalf("[pulumicost-plugin-aws-public] Serve failed: %v", err)
    }
}
```

**Test manually**:
```bash
go build -o pulumicost-plugin-aws-public ./cmd/pulumicost-plugin-aws-public

./pulumicost-plugin-aws-public
# Expected output to stdout: PORT=12345
# Stderr: [pulumicost-plugin-aws-public] Initialized for region: unknown
```

---

### Phase 6: Build-Time Pricing Tool (tools/)

**Why sixth**: Generates pricing data for production builds.

**Files to create**:
1. `tools/generate-pricing/main.go` - CLI tool with --dummy flag

**Implementation**:
```go
package main

import (
    "encoding/json"
    "flag"
    "fmt"
    "os"
    "strings"
)

func main() {
    regions := flag.String("regions", "us-east-1", "Comma-separated regions")
    outDir := flag.String("out-dir", "./data", "Output directory")
    dummy := flag.Bool("dummy", false, "Generate dummy data")

    flag.Parse()

    regionList := strings.Split(*regions, ",")

    for _, region := range regionList {
        if err := generatePricingData(region, *outDir, *dummy); err != nil {
            fmt.Fprintf(os.Stderr, "Failed to generate pricing for %s: %v\n", region, err)
            os.Exit(1)
        }
    }

    fmt.Println("Pricing data generated successfully")
}

func generatePricingData(region, outDir string, dummy bool) error {
    // For v1, only dummy mode is implemented
    if !dummy {
        return fmt.Errorf("real AWS pricing fetch not implemented yet")
    }

    data := map[string]interface{}{
        "region":   region,
        "currency": "USD",
        "ec2": map[string]interface{}{
            "t3.micro": map[string]interface{}{
                "instance_type":    "t3.micro",
                "operating_system": "Linux",
                "tenancy":          "Shared",
                "hourly_rate":      0.0104,
            },
        },
        "ebs": map[string]interface{}{
            "gp3": map[string]interface{}{
                "volume_type":       "gp3",
                "rate_per_gb_month": 0.08,
            },
        },
    }

    outFile := fmt.Sprintf("%s/aws_pricing_%s.json", outDir, region)
    f, err := os.Create(outFile)
    if err != nil {
        return err
    }
    defer f.Close()

    enc := json.NewEncoder(f)
    enc.SetIndent("", "  ")
    return enc.Encode(data)
}
```

**Test**:
```bash
go run ./tools/generate-pricing --regions us-east-1,us-west-2,eu-west-1 --out-dir ./data --dummy

ls data/
# Expected: aws_pricing_us-east-1.json, aws_pricing_us-west-2.json, aws_pricing_eu-west-1.json
```

---

### Phase 7: Region-Specific Embed Files (internal/pricing)

**Why seventh**: Enables region-specific binary builds.

**Files to create**:
1. `internal/pricing/embed_use1.go` - Embeds data/aws_pricing_us-east-1.json
2. `internal/pricing/embed_usw2.go` - Embeds data/aws_pricing_us-west-2.json
3. `internal/pricing/embed_euw1.go` - Embeds data/aws_pricing_eu-west-1.json

**Example** (`internal/pricing/embed_use1.go`):
```go
//go:build region_use1

package pricing

import _ "embed"

//go:embed ../../data/aws_pricing_us-east-1.json
var rawPricingJSON []byte
```

**Test**:
```bash
# Build with us-east-1 tag
go build -tags region_use1 -o pulumicost-plugin-aws-public-us-east-1 ./cmd/pulumicost-plugin-aws-public

./pulumicost-plugin-aws-public-us-east-1
# Stderr should show: Initialized for region: us-east-1
```

---

### Phase 8: GoReleaser Setup

**Why eighth**: Automates multi-region builds.

**Files to create**:
1. `.goreleaser.yaml` - Define builds for all regions

**Test**:
```bash
goreleaser build --snapshot --clean

ls dist/
# Expected: Binaries for us-east-1, us-west-2, eu-west-1 × OS/arch combinations
```

---

## Development Workflow

### Running Unit Tests

```bash
# All tests
go test ./...

# Specific package with verbose output
go test ./internal/plugin -v

# With coverage
go test ./... -cover
```

---

### Manual gRPC Testing

**Start the plugin**:
```bash
go run ./cmd/pulumicost-plugin-aws-public
# Output: PORT=12345
```

**Test Name RPC**:
```bash
grpcurl -plaintext localhost:12345 pulumicost.v1.CostSourceService/Name
```

**Test Supports RPC**:
```bash
grpcurl -plaintext \
  -d '{"resource": {"provider": "aws", "resource_type": "ec2", "region": "us-east-1"}}' \
  localhost:12345 \
  pulumicost.v1.CostSourceService/Supports
```

**Test GetProjectedCost RPC**:
```bash
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
  pulumicost.v1.CostSourceService/GetProjectedCost
```

---

### Linting

```bash
# Install golangci-lint
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Run linter
golangci-lint run ./...

# Or via make
make lint
```

---

## Common Pitfalls

### 1. Writing to stdout

**DON'T**:
```go
fmt.Println("Debug message")  // BAD - writes to stdout
```

**DO**:
```go
fmt.Fprintf(os.Stderr, "[pulumicost-plugin-aws-public] Debug message\n")  // GOOD
```

**Why**: stdout is reserved for PORT announcement only.

---

### 2. Mocking Proto Messages

**DON'T**:
```go
type mockResourceDescriptor struct {
    // Custom mock implementation
}
```

**DO**:
```go
// Use real proto messages
rd := &pbc.ResourceDescriptor{
    Provider:     "aws",
    ResourceType: "ec2",
    Sku:          "t3.micro",
    Region:       "us-east-1",
}
```

**Why**: Constitution prohibits mocking dependencies we don't own.

---

### 3. Forgetting sync.Once

**DON'T**:
```go
func (c *Client) init() error {
    // Parse every time
    var data pricingData
    json.Unmarshal(rawPricingJSON, &data)
    return nil
}
```

**DO**:
```go
func (c *Client) init() error {
    c.once.Do(func() {
        var data pricingData
        c.err = json.Unmarshal(rawPricingJSON, &data)
        // Build indexes
    })
    return c.err
}
```

**Why**: Thread safety for concurrent gRPC calls.

---

### 4. Region-Specific Build Tags

**DON'T**:
```go
//go:build use1  // BAD - inconsistent naming
```

**DO**:
```go
//go:build region_use1  // GOOD - consistent prefix
```

**Why**: GoReleaser expects exact tag names.

---

## Next Steps

After implementing all phases:

1. **Create Makefile** with lint, test, build targets
2. **Write README.md** documenting gRPC protocol and usage
3. **Write RELEASING.md** with release checklist
4. **Update CLAUDE.md** with implementation notes
5. **Create tasks.md** via `/speckit.tasks` command
6. **Begin implementation** following tasks.md

---

## Getting Help

- **Constitution**: See `.specify/memory/constitution.md` for rules and principles
- **Spec**: See `specs/001-pulumicost-aws-plugin/spec.md` for requirements
- **Contracts**: See `specs/001-pulumicost-aws-plugin/contracts/` for API details
- **Proto Definitions**: Check `github.com/rshade/pulumicost-spec/proto/`

---

## Success Criteria Checklist

Before considering implementation complete:

- [ ] All unit tests pass (`go test ./...`)
- [ ] All linting passes (`make lint`)
- [ ] Manual grpcurl testing succeeds for all RPCs
- [ ] GetProjectedCost responds in <100ms
- [ ] Supports responds in <10ms
- [ ] Plugin announces PORT within 1 second
- [ ] Region-specific binaries build successfully
- [ ] GoReleaser builds all 3 regions
- [ ] Binary sizes <10MB per region
- [ ] Graceful shutdown on Ctrl+C works
