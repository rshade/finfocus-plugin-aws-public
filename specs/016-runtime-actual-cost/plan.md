# Implementation Plan: Runtime-Based Actual Cost Estimation

**Branch**: `016-runtime-actual-cost` | **Date**: 2025-12-31 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/016-runtime-actual-cost/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/commands/plan.md` for the execution workflow.

## Summary

Enhance `GetActualCost` to automatically detect resource runtime from Pulumi state metadata (`pulumi:created`, `pulumi:external`) when explicit timestamps are not provided. The fallback formula (`projected_monthly_cost × runtime_hours / 730`) remains unchanged; this feature adds smart timestamp resolution and confidence indicators.

## Technical Context

**Language/Version**: Go 1.25+
**Primary Dependencies**: gRPC, finfocus-spec (proto), zerolog, google.golang.org/protobuf (timestamppb)
**Storage**: N/A (embedded pricing data, stateless service)
**Testing**: Go testing with table-driven tests, integration tests with `-tags=integration`
**Target Platform**: Linux server (gRPC service on 127.0.0.1)
**Project Type**: Single project (gRPC plugin)
**Performance Goals**: GetActualCost RPC < 100ms per call (existing target)
**Constraints**: Thread-safe handlers, no external network calls at runtime
**Scale/Scope**: Extends existing `GetActualCost` handler in `internal/plugin/`

### Technical Clarifications (RESOLVED)

All clarifications have been resolved in [research.md](./research.md).

| Clarification | Decision | Reference |
|---------------|----------|-----------|
| Confidence encoding | Semantic encoding in `source` field | research.md R1 |
| Pulumi metadata keys | Read from `req.Tags` map | research.md R2 |
| Timestamp priority | Explicit > pulumi:created > error | research.md R3 |

## Constitution Check

*GATE: All checks pass post-Phase 1 design.*

### I. Code Quality & Simplicity ✅

| Principle | Status | Justification |
|-----------|--------|---------------|
| KISS | ✅ | Single-purpose timestamp resolution logic |
| Single Responsibility | ✅ | New functions: `resolveTimestamps()`, `determineConfidence()` |
| Explicit behavior | ✅ | Priority order documented in code comments |
| Stateless | ✅ | No state changes; each RPC independent |

### II. Testing Discipline ✅

| Requirement | Status | Justification |
|-------------|--------|---------------|
| Unit tests for transformations | ✅ | Table-driven tests for timestamp parsing |
| Integration tests for gRPC | ✅ | Test with `pulumi:created` in tags |
| No external mocking | ✅ | Uses existing mock pricing client |
| < 1s unit suite | ✅ | Pure functions, no I/O |

### III. Protocol & Interface Consistency ✅

| Requirement | Status | Justification |
|-------------|--------|---------------|
| Proto-defined types | ✅ | Uses existing `source` field semantically (no proto change) |
| Error codes from enum | ✅ | Existing error handling pattern |
| Thread safety | ✅ | Stateless timestamp parsing |
| Logging (zerolog) | ✅ | Existing pattern with trace_id |

### IV. Performance & Reliability ✅

| Requirement | Status | Justification |
|-------------|--------|---------------|
| GetActualCost < 100ms | ✅ | Adds ~1ms for timestamp parsing |
| Memory < 400MB | ✅ | No additional memory allocation |
| Concurrent calls | ✅ | Stateless functions |

### V. Build & Release Quality ✅

| Requirement | Status | Justification |
|-------------|--------|---------------|
| `make lint` pass | ✅ | Standard Go code |
| `make test` pass | ✅ | New tests added |
| Markdown lint | ✅ | Documentation follows conventions |

## Project Structure

### Documentation (this feature)

```text
specs/016-runtime-actual-cost/
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 output - COMPLETE
├── data-model.md        # Phase 1 output - COMPLETE
├── quickstart.md        # Phase 1 output - COMPLETE
├── contracts/           # Phase 1 output - COMPLETE (no proto changes needed)
│   └── README.md
└── tasks.md             # Phase 2 output (/speckit.tasks command - NOT created by /speckit.plan)
```

### Source Code (repository root)

```text
internal/
├── plugin/
│   ├── actual.go            # MODIFY: Add resolveTimestamps(), determineConfidence(), etc.
│   ├── actual_test.go       # MODIFY: Add table-driven tests
│   ├── plugin.go            # MODIFY: Update GetActualCost handler
│   └── validation.go        # MINOR: May adjust timestamp validation flow
└── ...

test/
└── fixtures/
    └── actual-cost/         # NEW: Test fixtures with Pulumi metadata
        ├── with-created.json
        ├── with-external.json
        └── explicit-override.json
```

**Structure Decision**: Single project structure. Changes isolated to `internal/plugin/` package. No new packages required.

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| None identified | N/A | N/A |

---

## Phase 0: Research (COMPLETE)

All research items resolved. See [research.md](./research.md) for details.

### R1: Confidence Field Representation ✅

**Decision**: Semantic encoding in `source` field.
**Format**: `aws-public-fallback[confidence:HIGH|MEDIUM|LOW]`

### R2: Pulumi Metadata Injection ✅

**Decision**: Read from `req.Tags` map with keys:
- `pulumi:created` (RFC3339)
- `pulumi:external` ("true")
- `pulumi:modified` (RFC3339, not used for fallback)

### R3: Timestamp Priority Semantics ✅

**Decision**: Two-phase resolution:
1. Pre-validation: Resolve timestamps from explicit or tags
2. Post-resolution: Standard validation with guaranteed non-nil timestamps

Priority: Explicit request > pulumi:created > error

---

## Phase 1: Design & Contracts (COMPLETE)

### Artifacts Generated

| Artifact | Status | Description |
|----------|--------|-------------|
| [data-model.md](./data-model.md) | ✅ | Entity definitions, relationships, function signatures |
| [contracts/README.md](./contracts/README.md) | ✅ | API contract (no proto changes needed) |
| [quickstart.md](./quickstart.md) | ✅ | Testing guide with grpcurl examples |
| CLAUDE.md | ✅ | Agent context updated |

### Key Design Decisions

1. **No Proto Changes**: Confidence encoded in `source` field semantically
2. **TimestampResolution Struct**: New value object to track resolution provenance
3. **ConfidenceLevel Type**: String enum (HIGH/MEDIUM/LOW) for clarity
4. **Backward Compatible**: Existing callers unaffected

---

## Next Steps

Run `/speckit.tasks` to generate `tasks.md` with implementation tasks.

The implementation will consist of:

1. **New functions in actual.go**:
   - `resolveTimestamps()` - Priority-based timestamp resolution
   - `extractPulumiCreated()` - Parse RFC3339 from tags
   - `isImportedResource()` - Check pulumi:external
   - `determineConfidence()` - Map resolution to confidence
   - `formatSourceWithConfidence()` - Encode confidence in source

2. **Modified GetActualCost in plugin.go**:
   - Call `resolveTimestamps()` before validation
   - Use resolved timestamps for calculation
   - Include confidence in response

3. **Tests in actual_test.go**:
   - Table-driven tests for each new function
   - Integration tests for end-to-end scenarios

4. **Test fixtures**:
   - JSON fixtures for manual/automated testing
