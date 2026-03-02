# Research: Multi-Region Router Plugin

**Branch**: `036-multi-region-router` | **Date**: 2026-02-12

## R1: Router-to-Child Communication Protocol

**Decision**: Use `pluginsdk.NewConnectClient(baseURL)` from `finfocus-spec` SDK.

**Rationale**: The SDK already provides a full Connect client wrapper
(`pluginsdk.Client`) that implements all 11 CostSourceService RPCs. The router
creates one `pluginsdk.Client` per child process, reusing HTTP connections via
the built-in connection pool. This avoids importing raw gRPC dependencies and
keeps the router aligned with the SDK's intended usage pattern.

**Alternatives considered**:

- Raw `grpc.NewClient()` + `pbc.NewCostSourceServiceClient()`: Used in
  integration tests but requires HTTP/2 and adds gRPC-specific dependencies.
  Connect is simpler (HTTP/1.1 + JSON) and matches the SDK recommendation.
- Direct HTTP proxy (reverse proxy): Would avoid proto deserialization
  overhead but loses type safety and trace_id propagation. Rejected.

**Key API surface**:

```go
client := pluginsdk.NewConnectClient(fmt.Sprintf("http://localhost:%d", childPort))
defer client.Close()

resp, err := client.GetProjectedCost(ctx, req)
```

## R2: Child Process Lifecycle Management

**Decision**: `os/exec.Command` with stdout pipe for PORT capture, process
group for cleanup.

**Rationale**: The Docker `entrypoint.sh` already uses this pattern (FIFO +
PID tracking). The Go equivalent is `exec.Command` with `StdoutPipe()` for
PORT announcement capture. Integration tests already demonstrate this pattern
in `integration_test.go`.

**Alternatives considered**:

- Shell wrapper (`/bin/sh -c`): Adds unnecessary indirection. Direct
  `exec.Command` is cleaner in Go.
- Unix sockets instead of TCP: Would eliminate port management but the plugin
  protocol requires TCP (PORT announcement is TCP port).
- Process supervisor library (e.g., `hashicorp/go-plugin`): Over-engineered
  for this use case. Our child lifecycle is simple: start, capture PORT, health
  check, kill.

**Implementation details**:

- Set `cmd.Env` to include `FINFOCUS_PLUGIN_WEB_ENABLED=true` for Connect
  protocol support on children.
- Parse `PORT=(\d+)` from child stdout within 30-second timeout.
- Store child PID for graceful shutdown via `cmd.Process.Signal(syscall.SIGTERM)`.
- Health check: attempt a `Name()` RPC; if it fails, mark child unhealthy.

## R3: Concurrent Launch Prevention

**Decision**: `sync.Mutex` per region using `singleflight`-style pattern in
the child registry.

**Rationale**: When two concurrent requests arrive for the same not-yet-launched
region, only one should trigger the child launch. Go's `sync.Once` is not ideal
because it doesn't support retry on failure. Instead, use a mutex-guarded
launch state per region that transitions through: `idle → launching → ready →
unhealthy`.

**Alternatives considered**:

- `golang.org/x/sync/singleflight`: Good for deduplication but doesn't persist
  the result (child process) across calls. We need the child to remain in the
  registry.
- `sync.Once` per region: Doesn't support retry after failure. A failed launch
  would permanently block the region.
- Global `sync.Mutex`: Too coarse — blocks all regions during any single
  launch. Per-region locking is better.

## R4: Binary Download and Verification

**Decision**: HTTP GET from GitHub Releases + SHA256 checksum verification
against `checksums.txt`.

**Rationale**: Matches the existing Docker build pattern exactly. The
`checksums.txt` file is already uploaded as a release asset by the CI pipeline.

**Implementation details**:

1. Download `checksums.txt` from release (cache for session).
2. Download region tarball.
3. Compute SHA256 of downloaded tarball.
4. Verify against `checksums.txt` entry.
5. Extract binary from tarball.
6. Set executable permission (`os.Chmod 0755`).

**URL pattern** (hardcoded, matches Dockerfile):

```text
https://github.com/rshade/finfocus-plugin-aws-public/releases/download/v{version}/checksums.txt
https://github.com/rshade/finfocus-plugin-aws-public/releases/download/v{version}/finfocus-plugin-aws-public_{version}_{OS}_{arch}_{region}.tar.gz
```

**Alternatives considered**:

- Configurable download URL: Rejected per clarification — hardcoded is
  sufficient. Air-gapped users use offline mode.
- No verification (HTTPS only): Rejected per clarification — SHA256 checksum
  verification required.

## R5: Router Binary Build and Release

**Decision**: Add router build to `tools/generate-goreleaser/main.go` template.

**Rationale**: GoReleaser config is auto-generated from `regions.yaml`. The
router build is a simple addition: same `cmd/` entry point pattern, no build
tags, no embedded pricing data.

**Implementation details**:

- Router build: `cmd/finfocus-plugin-aws-public-router/main.go`
- Binary name inside archive: `finfocus-plugin-aws-public` (no region suffix)
- Archive name: `finfocus-plugin-aws-public_{version}_{OS}_{arch}.tar.gz`
  (no region suffix)
- Separate archive rule in GoReleaser to avoid region suffix in name
- Release workflow: build router alongside regions, include in checksums

**Alternatives considered**:

- Separate release pipeline for router: Unnecessary complexity. Router is part
  of the same release.
- Router as Go plugin (`.so`): Go plugins are fragile and
  platform-specific. Separate binary is simpler and matches existing patterns.

## R6: Trace ID Propagation

**Decision**: Extract trace_id from incoming gRPC metadata, inject into
outgoing Connect client context.

**Rationale**: The existing plugin extracts trace_id via
`pluginsdk.TraceIDFromContext(ctx)` or direct metadata lookup. The router
does the same extraction, then sets the trace_id as a Connect request header
when delegating to children.

**Implementation details**:

```go
// Extract from incoming request
traceID := pluginsdk.TraceIDFromContext(ctx)

// Inject into outgoing Connect request
connectReq := connect.NewRequest(protoReq)
connectReq.Header().Set(pluginsdk.TraceIDMetadataKey, traceID)
```

The `pluginsdk.Client` methods accept `context.Context`, so the router
wraps the context with trace_id metadata before calling.

## R7: Region Extraction from Requests

**Decision**: Extract region from `ResourceDescriptor.Region` for most RPCs;
fan-out by region for `GetRecommendations`.

**Rationale**: All single-resource RPCs (GetProjectedCost, GetActualCost,
Supports, EstimateCost, GetPricingSpec) include a `ResourceDescriptor` with a
`Region` field. The router reads this field to determine which child to
delegate to.

**Special cases**:

- `Name()`: Returns `"finfocus-plugin-aws-public"` directly (no delegation).
- `GetPluginInfo()`: Returns router metadata directly (no delegation).
- `GetRecommendations()`: Groups `target_resources` by region, delegates to
  each child in parallel, merges responses.
- `DismissRecommendation()`: Returns Unimplemented (stateless plugin).
- `GetBudgets()`: Returns Unimplemented (no budget support).
- `DryRun()`: Delegates to first available child (region-agnostic introspection).
