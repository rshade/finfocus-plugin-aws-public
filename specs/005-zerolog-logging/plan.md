# Implementation Plan: Zerolog Structured Logging with Trace Propagation

**Branch**: `005-zerolog-logging` | **Date**: 2025-11-26 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/005-zerolog-logging/spec.md`

## Summary

Adopt zerolog v1.34.0+ structured logging using PulumiCost SDK utilities
(`NewPluginLogger`, `TracingUnaryServerInterceptor`, `TraceIDFromContext`) to
enable distributed tracing correlation with pulumicost-core. All gRPC handlers
will log operations with consistent field names, trace_id propagation, and
measurable performance overhead (<1ms).

## Technical Context

**Language/Version**: Go 1.25+
**Primary Dependencies**:

- github.com/rs/zerolog v1.34.0+ (logging)
- github.com/rshade/pulumicost-spec v0.3.0+ (SDK logging utilities)
- github.com/rshade/pulumicost-core v0.1.0 (pluginsdk.Serve)
- google.golang.org/grpc v1.77.0 (gRPC server)

**Storage**: N/A (logs to stderr only)
**Testing**: Go testing with table-driven tests, benchmarks for latency validation
**Target Platform**: Linux server (gRPC plugin binary)
**Project Type**: Single Go module - gRPC plugin service
**Performance Goals**:

- Logging overhead: <1ms per request (SC-005)
- Startup log: within 100ms of process start (SC-004)

**Constraints**:

- stdout reserved for PORT announcement only
- stderr for all logs (JSON format)
- Thread-safe for concurrent gRPC calls

**Scale/Scope**: 100+ concurrent RPC calls supported

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### I. Code Quality & Simplicity

- [x] **KISS**: Logger injection via struct field, no complex abstractions
- [x] **Single Responsibility**: Logging code in handlers, not separate middleware
- [x] **Explicit**: SDK utilities explicitly called, no hidden behavior
- [x] **Stateless**: Logger instance is stateless, safe for concurrent use

### II. Testing Discipline

- [x] **Unit tests**: Table-driven tests for logging output validation
- [x] **Benchmarks**: Verify <1ms overhead per SC-005
- [x] **No mocking SDK**: Use real zerolog, validate output content
- [x] **Fast execution**: Logging tests should complete in <1s

### III. Protocol & Interface Consistency

- [x] **gRPC protocol**: No changes to proto definitions
- [x] **PORT announcement**: Unchanged (stdout)
- [x] **Logs to stderr**: All logging via zerolog to stderr
- [x] **Thread safety**: zerolog is lock-free, context-based trace_id

### IV. Performance & Reliability

- [x] **Latency target**: <1ms logging overhead (measured)
- [x] **Memory**: Negligible increase (zerolog is allocation-free)
- [x] **Concurrent calls**: zerolog handles high concurrency

### V. Build & Release Quality

- [x] **make lint**: No new lint issues expected
- [x] **make test**: All existing tests + new logging tests
- [x] **GoReleaser**: No build changes needed (same binaries)

**Constitution Check Status**: PASS - No violations

## Project Structure

### Documentation (this feature)

```text
specs/005-zerolog-logging/
├── plan.md              # This file
├── spec.md              # Feature specification
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output (log entry schema)
├── quickstart.md        # Phase 1 output (integration guide)
├── contracts/           # Phase 1 output (log schema)
└── tasks.md             # Phase 2 output (from /speckit.tasks)
```

### Source Code (repository root)

```text
cmd/
└── pulumicost-plugin-aws-public/
    └── main.go              # Add logger initialization, pass to plugin

internal/
├── plugin/
│   ├── plugin.go            # Add logger field, instrument handlers
│   ├── projected.go         # Add logging to GetProjectedCost
│   ├── supports.go          # Add logging to Supports
│   ├── actual.go            # Add logging to GetActualCost
│   └── *_test.go            # Add logging validation tests
└── pricing/
    └── client.go            # Optional: Add debug logging for lookups
```

**Structure Decision**: Existing Go plugin structure maintained. Logger added as
a field to AWSPublicPlugin struct and passed from main.go. No new packages
needed - logging is instrumentation of existing code.

## Complexity Tracking

No complexity violations - this is a straightforward instrumentation feature.
