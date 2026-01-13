# Implementation Plan: Fallback GetActualCost

**Branch**: `004-actual-cost-fallback` | **Date**: 2025-11-25 | **Spec**: spec.md
**Input**: Feature specification from `/specs/004-actual-cost-fallback/spec.md`

## Summary

Implement GetActualCost RPC method using fallback calculation based on projected
monthly cost pro-rated by runtime hours.
Formula: `actual_cost = projected_monthly_cost × (runtime_hours / 730)`.

## Technical Context

**Language/Version**: Go 1.25+
**Primary Dependencies**: finfocus-core/pkg/pluginsdk, finfocus-spec v0.3.0
**Storage**: N/A (stateless calculation)
**Testing**: Go testing with table-driven tests
**Target Platform**: Linux (gRPC plugin binary)
**Project Type**: Single project (Go plugin)
**Performance Goals**: < 10ms latency per RPC call (SC-003)
**Constraints**: Thread-safe, stateless, no external API calls
**Scale/Scope**: Single resource per RPC call

## Constitution Check

GATE: Passed - no violations

- ✅ No new dependencies added
- ✅ Reuses existing pricing lookup infrastructure
- ✅ Follows existing error code patterns (proto-defined only)
- ✅ Maintains region-specific binary architecture

## Project Structure

### Documentation (this feature)

```text
specs/004-actual-cost-fallback/
├── plan.md              # This file
├── research.md          # Technical decisions (v0.3.0 proto structure)
├── data-model.md        # Entity documentation (v0.3.0 proto)
├── spec.md              # Feature specification
└── tasks.md             # Implementation tasks (all complete)
```

### Source Code (repository root)

```text
internal/
├── plugin/
│   ├── plugin.go        # MODIFIED: GetActualCost implementation
│   ├── actual.go        # CREATED: Helper functions
│   ├── actual_test.go   # CREATED: Comprehensive test suite
│   ├── projected.go     # Existing: Reused for monthly cost lookup
│   └── supports.go      # Existing: Region validation
└── pricing/
    └── client.go        # Existing: Thread-safe pricing lookups
```

**Structure Decision**: Single project structure - added actual.go and
actual_test.go alongside existing plugin files.

## Implementation Summary

### Completed Components

1. **GetActualCost in plugin.go**
   - Validates request (nil checks, timestamps)
   - Parses ResourceDescriptor from JSON-encoded ResourceId (v0.3.0)
   - Calculates runtime hours from Start/End timestamps
   - Routes to existing pricing helpers (estimateEC2, estimateEBS, estimateStub)
   - Applies formula and returns ActualCostResult array

2. **Helper Functions in actual.go**
   - `calculateRuntimeHours(from, to time.Time)`: Duration calculation with validation
   - `getProjectedForResource(ctx, resource)`: Unified routing to pricing helpers
   - `formatActualBillingDetail(detail, hours, cost)`: Billing detail formatting
   - `parseResourceFromRequest(req)`: JSON parsing of ResourceId field

3. **Test Suite in actual_test.go**
   - Table-driven tests for all scenarios
   - Mock pricing client for isolated testing
   - Tests: EC2, EBS, stub services, invalid ranges, nil timestamps, zero duration
   - Benchmark: 3.3μs/op (well under 10ms requirement)

### Proto v0.3.0 Alignment

The implementation uses the actual proto v0.3.0 structure:

- Request: `resource_id` (JSON string), `start`/`end` timestamps, `tags` map
- Response: `Results []*ActualCostResult` array
- ActualCostResult: `timestamp`, `cost`, `usage_amount`, `usage_unit`, `source`

## Complexity Tracking

No violations - implementation follows existing patterns and reuses
infrastructure.

## Validation Results

- `make lint`: ✅ Passed
- `make test`: ✅ All tests pass
- Benchmark: 3.3μs/op (SC-003: < 10ms requirement met)
- Coverage: All new code paths tested
