# Feature Specification: Zerolog Structured Logging with Trace Propagation

**Feature Branch**: `005-zerolog-logging`
**Created**: 2025-11-26
**Status**: Draft
**Input**: GitHub Issue #22 - Adopt zerolog v1.34.0+ structured logging using
FinFocus SDK utilities for distributed tracing correlation with finfocus-core.

## Clarifications

### Session 2025-11-26

- Q: When trace_id is missing from gRPC metadata, should logs use empty string,
  generate a UUID, or omit the field? → A: Generate UUID for untraced requests
- Q: Should plugin validate/sanitize malformed trace_id values? → A: No,
  delegated to SDK interceptor (rshade/finfocus-spec#94)

## User Scenarios & Testing *(mandatory)*

### User Story 1 - End-to-End Request Tracing (Priority: P1)

As an operator debugging a cost estimation issue, I need to trace a single
request from finfocus-core through the aws-public plugin so that I can
identify where failures or slowdowns occur in the distributed system.

**Why this priority**: Distributed tracing is the primary motivation for this
feature. Without trace correlation, operators cannot effectively debug issues
spanning multiple services.

**Independent Test**: Can be fully tested by sending a request with a trace_id
in gRPC metadata and verifying the same trace_id appears in all plugin log
entries for that request.

**Acceptance Scenarios**:

1. **Given** a gRPC request arrives with trace_id in metadata, **When** the
   plugin processes the request, **Then** all log entries include the same
   trace_id field
2. **Given** a gRPC request arrives without trace_id, **When** the plugin
   processes the request, **Then** log entries include a newly generated UUID
   as the trace_id
3. **Given** multiple concurrent requests with different trace_ids, **When**
   processed simultaneously, **Then** each request's logs contain only its own
   trace_id

---

### User Story 2 - Structured Operation Logging (Priority: P2)

As an operator monitoring plugin health, I need all operations logged with
consistent field names so that I can build dashboards, alerts, and queries
across all FinFocus components.

**Why this priority**: Consistent field naming enables cross-component
monitoring and is required for integration with the broader FinFocus
observability stack.

**Independent Test**: Can be verified by examining log output and confirming
all entries use SDK-defined field name constants (e.g., "operation",
"resource_type", "cost_monthly", "duration_ms").

**Acceptance Scenarios**:

1. **Given** a GetProjectedCost request, **When** processing completes,
   **Then** logs include operation name, resource type, calculated cost,
   and duration
2. **Given** a Supports request, **When** processing completes, **Then** logs
   include operation name, resource type, region, and support status
3. **Given** any gRPC handler execution, **When** an error occurs, **Then**
   logs include error details with contextual fields

---

### User Story 3 - Plugin Startup Logging (Priority: P3)

As an operator deploying the plugin, I need to see startup information in logs
so that I can verify the plugin initialized correctly and know which version
is running.

**Why this priority**: Startup logging is essential for deployment verification
but is a one-time event per process lifecycle.

**Independent Test**: Can be verified by starting the plugin and checking logs
for version and initialization messages.

**Acceptance Scenarios**:

1. **Given** the plugin process starts, **When** initialization completes,
   **Then** logs include plugin name, version, and configured region
2. **Given** the plugin fails to initialize, **When** an error occurs during
   startup, **Then** logs include the failure reason with context

---

### User Story 4 - Cost Calculation Debugging (Priority: P3)

As a developer debugging pricing discrepancies, I need detailed logs showing
SKU resolution and pricing decisions so that I can understand how costs were
calculated.

**Why this priority**: Debug-level logging aids development and troubleshooting
but is not required for normal operations.

**Independent Test**: Can be verified by enabling debug logging and checking
that SKU lookup results and pricing calculations are logged.

**Acceptance Scenarios**:

1. **Given** debug logging is enabled, **When** processing an EC2 cost request,
   **Then** logs show instance type lookup and hourly rate found
2. **Given** debug logging is enabled, **When** processing an EBS cost request,
   **Then** logs show volume type lookup and GB-month rate found
3. **Given** a pricing lookup fails, **When** the SKU is not found, **Then**
   logs include the attempted SKU and reason for failure

---

### Edge Cases

- Malformed/oversized trace_id: Handled by SDK interceptor (rshade/finfocus-spec#94)
- Disk full/stderr unavailable: zerolog handles gracefully (drops logs, no crash)
- High concurrency logging: zerolog is lock-free, handles high throughput
- Sensitive field exposure: No sensitive data logged; costs are not PII

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Plugin MUST use SDK-provided `NewPluginLogger()` function to
  create the logger instance
- **FR-002**: Plugin MUST register SDK-provided `TracingUnaryServerInterceptor()`
  with the gRPC server
- **FR-003**: Plugin MUST extract trace_id using `TraceIDFromContext()` in all
  gRPC handlers
- **FR-004**: Plugin MUST log all operations using SDK-defined field name
  constants
- **FR-005**: Plugin MUST log startup with plugin name, version, and region
  information
- **FR-006**: Plugin MUST log GetProjectedCost operations with resource type,
  cost result, and duration
- **FR-007**: Plugin MUST log Supports operations with resource type, region,
  and support status
- **FR-008**: Plugin MUST log GetActualCost operations with resource ID, time
  range, cost result, and duration
- **FR-009**: Plugin MUST log all errors with appropriate context fields
- **FR-010**: Plugin MUST write logs to stderr only (stdout reserved for PORT
  announcement)
- **FR-011**: Plugin MUST support configurable log levels (at minimum: info,
  debug)
- **FR-012**: Plugin MUST include AWS-specific fields for resource operations
  (aws_service, aws_region, instance_type, storage_type)

### Key Entities

- **Log Entry**: A structured JSON object containing timestamp, level, message,
  trace_id, operation, and contextual fields
- **Trace ID**: A unique identifier propagated via gRPC metadata to correlate
  logs across services
- **Operation**: The gRPC method being executed (Name, Supports,
  GetProjectedCost, GetActualCost)
- **Logger**: A zerolog logger instance configured via SDK utilities

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% of gRPC handler invocations produce at least one log entry
  with the operation field
- **SC-002**: 100% of log entries for requests containing trace_id metadata
  include the trace_id field
- **SC-003**: All log field names match SDK-defined constants (verifiable by
  log schema validation)
- **SC-004**: Plugin startup produces log entry with plugin name and version
  within 100ms of process start
- **SC-005**: Logging overhead adds less than 1ms latency to request processing
  (measured via benchmarks)
- **SC-006**: All cost calculation results are logged with the cost_monthly
  field populated
- **SC-007**: All errors are logged with error context before being returned
  to the caller

## Assumptions

- SDK utilities (`NewPluginLogger`, `TracingUnaryServerInterceptor`,
  `TraceIDFromContext`) are available in finfocus-spec v0.3.0+
- SDK provides standard field name constants that this plugin must use
- zerolog v1.34.0+ is the required logging library version
- Log output format is JSON (standard zerolog behavior)
- Log level defaults to Info unless configured otherwise via environment
  variable
- finfocus-core will send trace_id in gRPC metadata using a standardized key

## Dependencies

- **External**: finfocus-spec v0.3.0+ with SDK logging utilities
  (rshade/finfocus-spec#75)
- **External**: zerolog v1.34.0+ library
- **External**: SDK trace_id validation in interceptor (rshade/finfocus-spec#94)
- **Related**: finfocus-core logging implementation (rshade/finfocus-core#170)

## Out of Scope

- Log aggregation or shipping (handled by deployment infrastructure)
- Log rotation or file management (plugin writes to stderr only)
- Custom log formats beyond JSON (zerolog default)
- Metrics or telemetry collection (separate concern)
- Distributed tracing spans/segments (only trace_id correlation)
- trace_id validation/sanitization (delegated to SDK interceptor)
