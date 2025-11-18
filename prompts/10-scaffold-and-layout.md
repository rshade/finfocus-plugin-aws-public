# Prompt: Scaffold module and basic layout for `pulumicost-plugin-aws-public`

You are implementing the `pulumicost-plugin-aws-public` Go plugin.
Repo: `pulumicost-plugin-aws-public`.

## Goal

Create the initial Go module, directory layout, and core plugin structure for the AWS public pricing plugin.

This step is **scaffolding only**:
- No AWS pricing logic yet.
- No GoReleaser config yet (that's a later prompt).
- Just the `go.mod`, directory structure, and plugin interface implementation.

## Requirements

### 1. Go module initialization

- If `go.mod` does not exist, create it with:
  - Module path: `github.com/rshade/pulumicost-plugin-aws-public`
  - Go version: at least `1.21` (or the repo's preferred Go version if already present).
- If `go.mod` already exists, **do not overwrite** it; only adjust if necessary (e.g., add missing `module` line).
- Add required dependencies:
  ```
  github.com/rshade/pulumicost-spec/sdk/go/proto/pulumicost/v1
  github.com/rshade/pulumicost-core/pkg/pluginsdk
  google.golang.org/grpc
  ```

### 2. Directory layout

Create the following structure if it does not exist:

```text
cmd/
  pulumicost-plugin-aws-public/
    main.go
internal/
  plugin/
    plugin.go       # Implements pluginsdk.Plugin interface
    supports.go     # Supports() logic
    projected.go    # GetProjectedCost() logic
    pricingspec.go  # Optional GetPricingSpec() logic (stub for now)
  pricing/
    client.go       # Pricing client with thread-safe lookups
  config/
    config.go       # Configuration (currency, discount factor)
tools/
  generate-pricing/
    main.go         # stub only, real logic in later prompt
data/
  .gitkeep        # placeholder so directory is tracked
```

### 3. Plugin interface implementation (`internal/plugin/plugin.go`)

Implement the `Plugin` interface from `pulumicost-core/pkg/pluginsdk`:

```go
package plugin

import (
	"context"

	pbc "github.com/rshade/pulumicost-spec/sdk/go/proto/pulumicost/v1"
)

// AWSPublicPlugin implements the pluginsdk.Plugin interface
type AWSPublicPlugin struct {
	region  string
	pricing PricingClient // interface to be defined
	// TODO: add config when needed
}

// PricingClient defines the interface for pricing lookups
// (implemented by internal/pricing.Client)
type PricingClient interface {
	Region() string
	Currency() string
	EC2OnDemandPricePerHour(instanceType, operatingSystem, tenancy string) (float64, bool)
	EBSPricePerGBMonth(volumeType string) (float64, bool)
}

func NewAWSPublicPlugin(region string, pricing PricingClient) *AWSPublicPlugin {
	return &AWSPublicPlugin{
		region:  region,
		pricing: pricing,
	}
}

// Name implements pluginsdk.Plugin
func (p *AWSPublicPlugin) Name() string {
	return "aws-public"
}

// GetProjectedCost implements pluginsdk.Plugin
func (p *AWSPublicPlugin) GetProjectedCost(
	ctx context.Context,
	req *pbc.GetProjectedCostRequest,
) (*pbc.GetProjectedCostResponse, error) {
	// TODO: implement in 30-estimation-logic-ec2-ebs.md
	return nil, nil
}

// GetActualCost implements pluginsdk.Plugin (but not applicable for public pricing)
func (p *AWSPublicPlugin) GetActualCost(
	ctx context.Context,
	req *pbc.GetActualCostRequest,
) (*pbc.GetActualCostResponse, error) {
	// Not implemented for public pricing plugin
	return nil, nil
}
```

### 4. Supports() logic stub (`internal/plugin/supports.go`)

Create a stub for the `Supports()` method (will be implemented in later prompt):

```go
package plugin

import (
	"context"

	pbc "github.com/rshade/pulumicost-spec/sdk/go/proto/pulumicost/v1"
)

// Supports checks if the plugin can estimate costs for a given resource
// This will be called via the gRPC server wrapper, not directly part of Plugin interface
func (p *AWSPublicPlugin) Supports(
	ctx context.Context,
	req *pbc.SupportsRequest,
) (*pbc.SupportsResponse, error) {
	// TODO: implement region and resource_type checks
	// For now, return not supported
	return &pbc.SupportsResponse{
		Supported: false,
		Reason:    "Not implemented yet",
	}, nil
}
```

### 5. GetProjectedCost() stub (`internal/plugin/projected.go`)

Create placeholder for cost estimation:

```go
package plugin

import (
	"context"

	pbc "github.com/rshade/pulumicost-spec/sdk/go/proto/pulumicost/v1"
)

// Implementation will be added in 30-estimation-logic-ec2-ebs.md
// This file is a placeholder to establish the structure
```

### 6. GetPricingSpec() stub (`internal/plugin/pricingspec.go`)

Create optional pricing spec endpoint:

```go
package plugin

import (
	"context"

	pbc "github.com/rshade/pulumicost-spec/sdk/go/proto/pulumicost/v1"
)

// GetPricingSpec returns detailed pricing information (optional)
// This will be implemented in a later version
func (p *AWSPublicPlugin) GetPricingSpec(
	ctx context.Context,
	req *pbc.GetPricingSpecRequest,
) (*pbc.GetPricingSpecResponse, error) {
	// TODO: implement in future version
	return nil, nil
}
```

### 7. Stub config file (`internal/config/config.go`)

Create a very small config type that we can extend later:

```go
package config

type Config struct {
	Currency              string  `json:"currency"`
	AccountDiscountFactor float64 `json:"accountDiscountFactor"`
}

func Default() Config {
	return Config{
		Currency:              "USD",
		AccountDiscountFactor: 1.0,
	}
}
```

- Leave TODOs for loading from env/flags in later prompts.

### 8. Stub pricing client (`internal/pricing/client.go`)

For now, just define the interface and placeholder struct; no real logic yet:

```go
package pricing

import "sync"

// Client provides thread-safe access to embedded AWS public pricing for a specific region
type Client struct {
	region   string
	currency string

	once sync.Once
	err  error

	// TODO: add parsed pricing structures and indexes in next prompt
}

const (
	// TODO: Region will be set via build-tag-specific files, e.g. embed_use1.go
	// For now, define a placeholder.
	DefaultRegion = "us-east-1"
)

func NewClient() (*Client, error) {
	// TODO: parse embedded JSON once and build lookup maps
	return &Client{
		region:   DefaultRegion,
		currency: "USD",
	}, nil
}

func (c *Client) Region() string {
	return c.region
}

func (c *Client) Currency() string {
	return c.currency
}

// EC2OnDemandPricePerHour looks up EC2 on-demand hourly price
// Returns (price, true) if found, (0, false) otherwise
func (c *Client) EC2OnDemandPricePerHour(instanceType, operatingSystem, tenancy string) (float64, bool) {
	// TODO: implement in next prompt
	return 0, false
}

// EBSPricePerGBMonth looks up EBS price per GB-month
// Returns (price, true) if found, (0, false) otherwise
func (c *Client) EBSPricePerGBMonth(volumeType string) (float64, bool) {
	// TODO: implement in next prompt
	return 0, false
}
```

- Later prompts will add `rawPricingJSON` via `//go:embed` in region-specific files.

### 9. Stub generator tool (`tools/generate-pricing/main.go`)

Create a small skeleton tool that accepts a `--regions` flag and **does nothing real yet**:

```go
package main

import (
	"flag"
	"fmt"
	"log"
	"strings"
)

func main() {
	var regionsCSV string
	var outDir string
	var dummy bool

	flag.StringVar(&regionsCSV, "regions", "", "comma-separated list of AWS regions (e.g. us-east-1,us-west-2)")
	flag.StringVar(&outDir, "out-dir", "./data", "output directory for pricing JSON files")
	flag.BoolVar(&dummy, "dummy", false, "generate dummy pricing data for development")
	flag.Parse()

	if regionsCSV == "" {
		log.Fatal("missing --regions")
	}

	regions := strings.Split(regionsCSV, ",")
	fmt.Printf("Stub generate-pricing tool. Regions: %v, Output: %s, Dummy: %v\n", regions, outDir, dummy)

	// TODO: in a later prompt, implement:
	// - Fetch AWS pricing (or generate dummy data if --dummy flag set)
	// - Trim to required services
	// - Write to data/aws_pricing_<region>.json
}
```

### 10. Initial main file (`cmd/pulumicost-plugin-aws-public/main.go`)

Create a minimal gRPC service entrypoint that uses pluginsdk.Serve():

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

	// Create plugin instance
	p := plugin.NewAWSPublicPlugin(pricingClient.Region(), pricingClient)

	// Serve gRPC using pluginsdk
	ctx := context.Background()
	if err := pluginsdk.Serve(ctx, pluginsdk.ServeConfig{
		Plugin: p,
		Port:   0, // 0 = use PORT env or ephemeral
	}); err != nil {
		log.Fatalf("[pulumicost-plugin-aws-public] Serve failed: %v", err)
	}
}
```

## Acceptance criteria

- `go mod tidy` succeeds
- `go build ./...` succeeds
- The plugin binary builds and can be started (will announce PORT but not accept calls yet until estimation logic is implemented)
- All stub files compile without errors

Make these changes now in the repo.
