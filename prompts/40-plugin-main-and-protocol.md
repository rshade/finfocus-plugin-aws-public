# Prompt: Implement gRPC service entrypoint with pluginsdk

You are implementing the `pulumicost-plugin-aws-public` Go plugin.
Repo: `pulumicost-plugin-aws-public`.

## Goal

Update `cmd/pulumicost-plugin-aws-public/main.go` to:

- Initialize the pricing client (using embedded JSON)
- Create the plugin instance
- Serve gRPC using `pluginsdk.Serve()`
- Handle PORT announcement and lifecycle correctly

This is the final integration point that ties everything together.

## 1. Main entrypoint

Update `cmd/pulumicost-plugin-aws-public/main.go`:

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
	// Initialize pricing client (loads and parses embedded JSON)
	pricingClient, err := pricing.NewClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "[pulumicost-plugin-aws-public] Failed to initialize pricing: %v\n", err)
		os.Exit(1)
	}

	// Log initialization (to stderr only)
	fmt.Fprintf(os.Stderr, "[pulumicost-plugin-aws-public] Initialized for region: %s\n", pricingClient.Region())

	// Create plugin instance
	p := plugin.NewAWSPublicPlugin(pricingClient.Region(), pricingClient)

	// Serve gRPC using pluginsdk
	// - This will announce PORT=<port> to stdout
	// - Then serve gRPC on 127.0.0.1:<port>
	// - Handle graceful shutdown on context cancellation
	ctx := context.Background()
	if err := pluginsdk.Serve(ctx, pluginsdk.ServeConfig{
		Plugin: p,
		Port:   0, // 0 = use PORT env or ephemeral
	}); err != nil {
		log.Fatalf("[pulumicost-plugin-aws-public] Serve failed: %v", err)
	}
}
```

## 2. Key behaviors

### PORT announcement

- `pluginsdk.Serve()` will write `PORT=<port>` to stdout as its **only** stdout output
- All other logging must go to stderr
- Core reads this PORT and connects via gRPC

### Graceful shutdown

- `pluginsdk.Serve()` blocks until context is cancelled
- On cancellation, it performs graceful gRPC shutdown
- No additional cleanup needed in main()

### Error handling

- If pricing client fails to initialize (corrupt embedded data), exit with code 1
- If `pluginsdk.Serve()` fails, exit with code 1 via `log.Fatalf()`
- Normal gRPC errors (e.g., invalid requests) are handled by the plugin methods returning gRPC status errors

## 3. Thread safety

The plugin struct must be thread-safe because gRPC will call methods concurrently:

- `pricing.Client` must use `sync.Once` for initialization and `sync.RWMutex` if needed for lookups
- `plugin.AWSPublicPlugin` struct should be immutable after construction (no mutable state)
- Each RPC call (GetProjectedCost, Supports, etc.) operates independently

Verify in `internal/pricing/client.go` that initialization uses `sync.Once`:

```go
type Client struct {
	region   string
	currency string

	once sync.Once // Used to parse rawPricingJSON exactly once
	err  error

	ec2Index map[string]ec2OnDemandPrice
	ebsIndex map[string]ebsVolumePrice
}

func (c *Client) init() error {
	var initErr error
	c.once.Do(func() {
		// Parse rawPricingJSON and build indexes
		// Set c.err if parsing fails
	})
	return c.err
}
```

## 4. Testing the gRPC service

Add a simple integration test or developer note to verify:

### Manual testing

```bash
# Build the plugin
go build -o pulumicost-plugin-aws-public ./cmd/pulumicost-plugin-aws-public

# Run it (will announce PORT)
./pulumicost-plugin-aws-public
# Output: PORT=12345
# (plugin now serving on 127.0.0.1:12345)

# In another terminal, use grpcurl to test:
grpcurl -plaintext \
  -d '{"resource": {"provider": "aws", "resource_type": "ec2", "sku": "t3.micro", "region": "us-east-1"}}' \
  localhost:12345 \
  pulumicost.v1.CostSourceService/GetProjectedCost
```

### Integration test (optional)

Create `cmd/pulumicost-plugin-aws-public/main_test.go`:

```go
package main

import (
	"context"
	"os/exec"
	"testing"
	"time"

	pbc "github.com/rshade/pulumicost-spec/sdk/go/proto/pulumicost/v1"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func TestPluginGRPC(t *testing.T) {
	t.Skip("Integration test - requires plugin binary and dummy pricing data")

	// Start plugin as subprocess
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "./pulumicost-plugin-aws-public")
	// TODO: Capture stdout to read PORT=<port>
	// TODO: Connect gRPC client to that port
	// TODO: Call GetProjectedCost and verify response

	require.NoError(t, cmd.Start())
	defer cmd.Process.Kill()

	// Test implementation left as exercise
}
```

## 5. Observability (optional)

For production, consider adding basic health logging to stderr:

```go
// In main() after successful initialization
fmt.Fprintf(os.Stderr, "[pulumicost-plugin-aws-public] Region: %s, Currency: %s\n",
	pricingClient.Region(),
	pricingClient.Currency(),
)
```

But keep it minimal - stdout is reserved for PORT announcement only.

## Acceptance criteria

- `go build ./cmd/pulumicost-plugin-aws-public` succeeds
- Running the binary announces PORT to stdout (e.g., `PORT=54321`)
- Plugin serves gRPC on the announced port
- Manual testing with grpcurl can successfully call GetProjectedCost, Supports, and Name methods
- Ctrl+C (context cancellation) triggers graceful shutdown

Implement these changes now.
