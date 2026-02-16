# Implementation Plan: Multi-Region Router Plugin

**Branch**: `036-multi-region-router` | **Date**: 2026-02-12 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/036-multi-region-router/spec.md`

## Summary

Build a thin router plugin binary that becomes the default install for
`aws-public`. The router discovers or auto-downloads region-specific binaries
and delegates gRPC/Connect requests to the correct region child process. This
replaces the current single-region default install that silently returns $0 for
non-matching regions.

**Technical approach**: The router implements the `pluginsdk.Plugin` interface
(same as region binaries) and uses `pluginsdk.NewConnectClient()` to delegate
RPCs to lazily-launched child processes. Each child is a standard region binary
with `FINFOCUS_PLUGIN_WEB_ENABLED=true`. The router contains no embedded
pricing data (~15MB vs ~150MB per region binary).

## Technical Context

**Language/Version**: Go 1.25+
**Primary Dependencies**: `finfocus-spec` v0.5.6 (`pluginsdk`, `pbcconnect`), `connectrpc.com/connect` v1.19.1, `zerolog`
**Storage**: N/A (in-memory child process registry only)
**Testing**: Go testing (`make test`), integration tests with real child binaries
**Target Platform**: Linux/Darwin/Windows (amd64, arm64) — same as existing binaries
**Project Type**: Single Go binary (new `cmd/` entry point + `internal/router/` package)
**Performance Goals**: <100ms delegation overhead per RPC, <500ms router startup, <30s child startup (including download)
**Constraints**: <15MB router binary (no pricing data), offline-capable (`FINFOCUS_PLUGIN_OFFLINE=true`), max 12 concurrent children
**Scale/Scope**: 12 AWS regions, 11 CostSourceService RPCs to delegate

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Code Quality & Simplicity | PASS | Router is a thin delegation layer with no business logic |
| II. Testing Discipline | PASS | Unit tests for routing logic, integration tests for end-to-end delegation |
| III. Protocol & Interface Consistency | PASS | Router uses `pluginsdk.Serve()` and announces PORT to stdout; uses proto error codes |
| IV. Performance & Reliability | PASS | Router binary <15MB (no pricing data); <100ms overhead; thread-safe via `sync.RWMutex` |
| V. Build & Release Quality | PASS | GoReleaser build added for router; `make lint` and `make test` enforced |
| Security | PASS | SHA256 checksum verification for downloads; loopback only; input validation |

**Post-Phase-1 Re-check**: No violations. Router adds no new architectural
patterns — it reuses `pluginsdk.Serve()`, `pluginsdk.NewConnectClient()`,
`zerolog`, and the existing build/release infrastructure.

## Project Structure

### Documentation (this feature)

```text
specs/036-multi-region-router/
├── plan.md              # This file
├── research.md          # Phase 0: technology decisions
├── data-model.md        # Phase 1: entities and state model
├── quickstart.md        # Phase 1: getting started guide
├── contracts/           # Phase 1: routing contract documentation
│   └── routing.md       # Request routing and delegation contract
└── tasks.md             # Phase 2 output (/speckit.tasks)
```

### Source Code (repository root)

```text
cmd/
├── finfocus-plugin-aws-public/          # Existing region binary entry point
│   └── main.go
└── finfocus-plugin-aws-public-router/   # NEW: Router entry point
    └── main.go                          # ~100 lines: logger, router init, pluginsdk.Serve()

internal/
├── plugin/                              # Existing plugin implementation (unchanged)
│   └── ...
└── router/                              # NEW: Router package
    ├── router.go                        # RouterPlugin struct implementing pluginsdk.Plugin
    ├── router_test.go                   # Unit tests for routing logic
    ├── child.go                         # ChildProcess struct, launch, health check
    ├── child_test.go                    # Unit tests for child lifecycle
    ├── registry.go                      # ChildRegistry (thread-safe map of region → child)
    ├── registry_test.go                 # Unit tests for registry operations
    ├── downloader.go                    # Binary download + SHA256 verification
    ├── downloader_test.go              # Unit tests for download logic
    ├── discovery.go                     # Local binary discovery (sibling scan)
    └── discovery_test.go               # Unit tests for discovery

tools/
└── generate-goreleaser/
    └── main.go                          # MODIFIED: Add router build to template

scripts/
└── build-router.sh                      # NEW: Build router binary for local dev

.goreleaser.yaml                         # MODIFIED (auto-generated): Router build config
.github/workflows/release.yml            # MODIFIED: Build router in release pipeline
```

**Structure Decision**: Follows existing project layout conventions. New
`internal/router/` package keeps router logic separate from plugin pricing
logic. New `cmd/finfocus-plugin-aws-public-router/` follows the established
pattern of one `cmd/` entry per binary.

## Complexity Tracking

No constitution violations. The router is a straightforward delegation layer
using existing SDK primitives.
