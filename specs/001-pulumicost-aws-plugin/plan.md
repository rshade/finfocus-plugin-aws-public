# Implementation Plan: PulumiCost AWS Public Plugin

**Branch**: `001-pulumicost-aws-plugin` | **Date**: 2025-11-16 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/001-pulumicost-aws-plugin/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/commands/plan.md` for the execution workflow.

## Summary

Implement a gRPC-based PulumiCost plugin that provides AWS cost estimates using public on-demand pricing data embedded at build time. The plugin implements the CostSourceService gRPC interface, serves on loopback via pluginsdk.Serve(), and supports region-specific binary distribution with EC2/EBS fully implemented and S3/Lambda/RDS/DynamoDB as stubs returning $0.

## Technical Context

**Language/Version**: Go 1.21+
**Primary Dependencies**:
- `github.com/rshade/pulumicost-spec/sdk/go/proto/pulumicost/v1` (proto definitions)
- `github.com/rshade/pulumicost-core/pkg/pluginsdk` (plugin SDK and Serve function)
- `google.golang.org/grpc` (gRPC runtime)
- `google.golang.org/protobuf` (protobuf runtime)

**Storage**: Embedded JSON files (go:embed) - No external storage required
**Testing**: `go test` with table-driven tests, mockPricingClient for unit tests, grpcurl for manual integration testing
**Target Platform**: Cross-platform Go binaries (Linux, macOS, Windows) - serves gRPC on loopback (127.0.0.1)
**Project Type**: Single Go project with gRPC service
**Performance Goals**:
- GetProjectedCost RPC: <100ms per call
- Supports RPC: <10ms per call
- Plugin startup: <500ms (includes pricing data parse)
- PORT announcement: <1 second
- Concurrent RPC capacity: 100+ concurrent calls

**Constraints**:
- Thread-safe for concurrent gRPC calls (sync.Once for initialization)
- Binary size: <10MB per region (with embedded pricing data)
- Memory footprint: <50MB per region binary
- No stdout output except PORT announcement
- Loopback-only serving (no TLS required)

**Scale/Scope**:
- 3 AWS regions initially (us-east-1, us-west-2, eu-west-1)
- 6 AWS service types (EC2, EBS fully implemented; S3, Lambda, RDS, DynamoDB stubs)
- ~48 functional requirements across gRPC lifecycle, cost estimation, and error handling

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### I. Code Quality & Simplicity ✅

- **KISS Principle**: Plugin is a straightforward gRPC service with pricing lookup logic - no premature abstraction
- **Single Responsibility**: Each package has clear purpose (plugin = gRPC handlers, pricing = embedded data + lookups, config = settings)
- **Explicit over Implicit**: All assumptions (730 hrs/month, Linux OS, Shared tenancy) are documented and explicit in code
- **Stateless Components**: Each gRPC RPC call is independent - no mutable state between calls
- **File Size**: Expect focused files <300 lines (exception: comprehensive test suites)

**PASS**: Design follows simplicity principles with clear separation of concerns.

### II. Testing Discipline ✅

- **Unit Tests**: Pure functions (pricing lookups, cost calculations) - use mockPricingClient interface
- **Integration Tests**: gRPC service methods tested with in-memory mock pricing
- **No Mocking Dependencies We Don't Own**: No mocking of proto messages, pluginsdk, or gRPC internals
- **Test Quality**: Table-driven tests for EC2/EBS variations, simple setup, clear assertions
- **Fast Execution**: <1s for unit suite, <5s for integration suite
- **Critical Path Coverage**: Focus on pricing lookups, cost calculations, gRPC handlers

**PASS**: Testing strategy focuses on core logic without over-complicated mocking.

### III. Protocol & Interface Consistency ✅

- **gRPC CostSourceService**: Implements all required RPCs (Name, Supports, GetProjectedCost)
- **PORT Announcement**: Writes `PORT=<port>` to stdout exactly once via pluginsdk.Serve()
- **Proto-Defined Types Only**: Uses ResourceDescriptor, GetProjectedCostResponse, SupportsResponse from pulumicost.v1
- **Error Codes**: Uses proto ErrorCode enum (ERROR_CODE_INVALID_RESOURCE, ERROR_CODE_UNSUPPORTED_REGION, ERROR_CODE_DATA_CORRUPTION)
- **Thread Safety**: Pricing client uses sync.Once, all RPC handlers are thread-safe
- **Region-Specific Binaries**: Each binary embeds only its region's pricing data
- **Build Tags**: Exactly one embed file selected at build time (region_use1, region_usw2, region_euw1)

**PASS**: Full compliance with gRPC protocol requirements from constitution v2.0.0.

### IV. Performance & Reliability ✅

- **Embedded Pricing Data**: Parsed once using sync.Once, cached in memory
- **Indexed Lookups**: Maps for O(1) pricing queries, no linear scans
- **Latency Targets**:
  - Plugin startup: <500ms (SC-003)
  - PORT announcement: <1s (SC-010)
  - GetProjectedCost RPC: <100ms (SC-001)
  - Supports RPC: <10ms (SC-002)
- **Resource Limits**:
  - Memory: <50MB per region binary
  - Binary size: <10MB per region (SC-011)
  - Concurrent RPCs: 100+ concurrent calls (SC-004)

**PASS**: Performance goals are realistic and meet constitution requirements.

### V. Build & Release Quality ✅

- **Linting**: All code must pass `make lint` (golangci-lint)
- **Testing**: All tests must pass `make test`
- **GoReleaser**: Builds all region-specific binaries (us-east-1, us-west-2, eu-west-1)
- **Build Tags**: region_use1, region_usw2, region_euw1 compile cleanly
- **Before Hooks**: tools/generate-pricing runs successfully before build
- **Binary Naming**: pulumicost-plugin-aws-public-<region>
- **gRPC Functional Testing**: Manual grpcurl testing before release

**PASS**: Build process aligns with constitution requirements.

### Security Requirements ✅

- **No Credentials in Logs**: All logging to stderr, no sensitive data exposed
- **No Network Calls at Runtime**: All pricing data embedded at build time
- **Input Validation**: Malformed ResourceDescriptor rejected with gRPC InvalidArgument error
- **Loopback-Only Serving**: 127.0.0.1 only, no TLS required for local communication

**PASS**: Security requirements satisfied for local-only plugin.

### Overall Constitution Compliance: ✅ PASS

No violations requiring justification. All gates pass for Phase 0 research.

## Project Structure

### Documentation (this feature)

```text
specs/[###-feature]/
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 output (/speckit.plan command)
├── data-model.md        # Phase 1 output (/speckit.plan command)
├── quickstart.md        # Phase 1 output (/speckit.plan command)
├── contracts/           # Phase 1 output (/speckit.plan command)
└── tasks.md             # Phase 2 output (/speckit.tasks command - NOT created by /speckit.plan)
```

### Source Code (repository root)

```text
pulumicost-plugin-aws-public/
├── cmd/
│   └── pulumicost-plugin-aws-public/
│       ├── main.go                    # Entrypoint: pluginsdk.Serve() + pricing client init
│       └── main_test.go               # Optional: integration test with subprocess
│
├── internal/
│   ├── plugin/
│   │   ├── plugin.go                  # AWSPublicPlugin struct implementing CostSourceService
│   │   ├── name.go                    # Name() RPC implementation
│   │   ├── supports.go                # Supports() RPC implementation
│   │   ├── projected.go               # GetProjectedCost() RPC implementation
│   │   ├── projected_test.go          # Unit tests with mockPricingClient
│   │   └── supports_test.go           # Unit tests for Supports()
│   │
│   ├── pricing/
│   │   ├── client.go                  # PricingClient interface + implementation
│   │   ├── client_test.go             # Unit tests for pricing lookups
│   │   ├── embed_use1.go              # go:build region_use1 - embeds data/aws_pricing_us-east-1.json
│   │   ├── embed_usw2.go              # go:build region_usw2 - embeds data/aws_pricing_us-west-2.json
│   │   ├── embed_euw1.go              # go:build region_euw1 - embeds data/aws_pricing_eu-west-1.json
│   │   └── embed_fallback.go          # go:build !(region_use1||region_usw2||region_euw1)
│   │
│   └── config/
│       └── config.go                  # Currency, discount factor (minimal for v1)
│
├── tools/
│   └── generate-pricing/
│       └── main.go                    # Build-time tool: fetch/trim AWS pricing data
│
├── data/
│   ├── aws_pricing_us-east-1.json     # Generated pricing data (not in git)
│   ├── aws_pricing_us-west-2.json     # Generated pricing data (not in git)
│   └── aws_pricing_eu-west-1.json     # Generated pricing data (not in git)
│
├── .goreleaser.yaml                   # Multi-binary build config with build tags
├── Makefile                           # lint, test, build targets
├── go.mod                             # Dependencies (pulumicost-spec, pluginsdk, grpc)
├── go.sum
├── README.md                          # gRPC protocol, usage, integration docs
├── RELEASING.md                       # Release checklist
└── CLAUDE.md                          # Agent guidance (gRPC architecture)
```

**Structure Decision**: Single Go project with gRPC service. Uses standard Go project layout with `cmd/` for entrypoint, `internal/` for private packages, and `tools/` for build-time utilities. Region-specific builds handled via build tags in `internal/pricing/embed_*.go` files.

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

**No violations.** All Constitution Check gates pass without requiring complexity justification.
