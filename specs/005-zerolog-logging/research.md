# Research: Zerolog Structured Logging with Trace Propagation

**Date**: 2025-11-26
**Feature**: 005-zerolog-logging

## SDK Logging Utilities

### Decision: Use finfocus-spec/sdk/go/pluginsdk logging utilities

**Rationale**: The SDK provides pre-built utilities specifically designed for
FinFocus plugin logging, ensuring consistency across all plugins.

**Alternatives considered**:

- Custom zerolog setup: Rejected - would diverge from SDK standards
- Standard library log: Rejected - no structured logging support
- logrus: Rejected - zerolog is SDK standard, better performance

### Available SDK Functions

Location: `github.com/rshade/finfocus-spec/sdk/go/pluginsdk/logging.go`

1. **NewPluginLogger(pluginName, version string, level zerolog.Level, w io.Writer)**
   - Creates zerolog logger with plugin_name and plugin_version pre-configured
   - Outputs to provided writer (defaults to os.Stderr if nil)
   - Includes timestamp automatically

2. **TracingUnaryServerInterceptor()**
   - gRPC server interceptor for trace_id extraction
   - Reads `x-finfocus-trace-id` from gRPC metadata
   - Stores in context via ContextWithTraceID()

3. **TraceIDFromContext(ctx context.Context) string**
   - Extracts trace_id from context
   - Returns empty string if not present

4. **LogOperation(logger zerolog.Logger, operation string) func()**
   - Helper for timing operations
   - Returns deferred function that logs duration

### SDK Field Name Constants

```go
FieldTraceID       = "trace_id"
FieldComponent     = "component"
FieldOperation     = "operation"
FieldDurationMs    = "duration_ms"
FieldResourceURN   = "resource_urn"
FieldResourceType  = "resource_type"
FieldPluginName    = "plugin_name"
FieldPluginVersion = "plugin_version"
FieldCostMonthly   = "cost_monthly"
FieldAdapter       = "adapter"
FieldErrorCode     = "error_code"
```

## Structured Logging vs Text Prefix (C1 Remediation)

### Decision: JSON `plugin_name` field replaces text prefix convention

**Rationale**: The CLAUDE.md constitution mentions using `[finfocus-plugin-aws-public]`
prefix for stderr diagnostic messages. With structured JSON logging via zerolog,
this convention is superseded by the `plugin_name` field in every log entry.

**Equivalence**:

- Old convention: `[finfocus-plugin-aws-public] error: ...`
- New convention: `{"plugin_name":"aws-public","level":"error",...}`

The JSON `plugin_name` field provides the same log source identification while
enabling machine-parseable output for log aggregation systems.

**Action**: CLAUDE.md logging section should be updated after this feature lands
to reflect the structured logging pattern.

## Interceptor Integration (U1 Remediation)

### Decision: Manual trace_id extraction in handlers (workaround)

**Rationale**: The pluginsdk.Serve() function manages the gRPC server lifecycle,
but ServeConfig does NOT support gRPC interceptors.

**Research findings** (verified 2025-11-26):

- `pluginsdk.ServeConfig` has only: Plugin, Port, Registry fields
- `Serve()` calls `grpc.NewServer()` with NO interceptor options
- `TracingUnaryServerInterceptor()` exists but cannot be registered

**Workaround**: Since we cannot register the interceptor, we must:

1. Read trace_id directly from gRPC metadata in each handler
2. Use `pluginsdk.TraceIDFromContext()` after manual context enrichment
3. Generate UUID when trace_id is missing (per FR-003)

**Implementation pattern**:

```go
func (p *AWSPublicPlugin) getTraceID(ctx context.Context) string {
    // First try SDK helper (works if interceptor was somehow registered)
    traceID := pluginsdk.TraceIDFromContext(ctx)
    if traceID != "" {
        return traceID
    }

    // Fallback: read directly from gRPC metadata
    if md, ok := metadata.FromIncomingContext(ctx); ok {
        if values := md.Get(pluginsdk.TraceIDMetadataKey); len(values) > 0 {
            return values[0]
        }
    }

    // Generate UUID if not present
    return uuid.New().String()
}
```

**Action**: Created GitHub issue rshade/finfocus-core#188 to add UnaryInterceptors
support to ServeConfig for future enhancement

## Missing trace_id Handling

### Decision: Generate UUID when trace_id is missing

**Rationale**: Clarified during /speckit.clarify session. Ensures all log
entries have a valid trace_id for correlation, following OpenTelemetry
conventions.

**Implementation approach**:

```go
traceID := pluginsdk.TraceIDFromContext(ctx)
if traceID == "" {
    traceID = uuid.New().String()
}
```

**Dependency**: May need to add `github.com/google/uuid` or use crypto/rand

## trace_id Validation

### Decision: Delegated to SDK interceptor

**Rationale**: Clarified during /speckit.clarify session. Created issue
rshade/finfocus-spec#94 to add validation to TracingUnaryServerInterceptor.

**Current state**: SDK interceptor passes through trace_id without validation.
Plugin trusts whatever the SDK provides.

## Log Level Configuration

### Decision: Use LOG_LEVEL environment variable

**Rationale**: Standard practice for containerized applications. Allows runtime
configuration without code changes.

**Implementation**:

```go
level := zerolog.InfoLevel
if lvl := os.Getenv("LOG_LEVEL"); lvl != "" {
    if parsed, err := zerolog.ParseLevel(lvl); err == nil {
        level = parsed
    }
}
logger := pluginsdk.NewPluginLogger("aws-public", version, level, nil)
```

**Supported levels**: trace, debug, info, warn, error, fatal, panic

## AWS-Specific Fields

### Decision: Add custom fields beyond SDK constants

**Rationale**: AWS resource operations need additional context for debugging.

**Custom fields** (not in SDK):

- `aws_service`: "ec2", "ebs", "s3", etc.
- `aws_region`: AWS region code
- `instance_type`: EC2 instance type (when applicable)
- `storage_type`: EBS volume type (when applicable)

**Implementation**: Add as inline fields in log statements, not constants.

## Performance Considerations

### Decision: Measure with benchmarks, target <1ms overhead

**Rationale**: SC-005 requires <1ms logging overhead per request.

**zerolog performance characteristics**:

- Allocation-free in hot path
- Lock-free, thread-safe
- JSON encoding is optimized

**Benchmark approach**:

```go
func BenchmarkGetProjectedCostWithLogging(b *testing.B) {
    // Compare with/without logging
    // Verify delta < 1ms
}
```

## Dependencies to Add

### go.mod additions

```go
require (
    github.com/rs/zerolog v1.34.0
    github.com/google/uuid v1.6.0  // For missing trace_id generation
)
```

Note: zerolog is likely already a transitive dependency via finfocus-spec.
Need to verify and potentially add explicit require.
