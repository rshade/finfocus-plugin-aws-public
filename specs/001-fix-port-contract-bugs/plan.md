# Implementation Plan: Port Binding & Contract Violation Fixes

**Branch**: `001-fix-port-contract-bugs` | **Date**: Thursday, February 12, 2026 | **Spec**: [specs/001-fix-port-contract-bugs/spec.md](spec.md)
**Input**: Feature specification from `/specs/001-fix-port-contract-bugs/spec.md`

## Summary

This feature fixes three critical issues: (1) The plugin ignores the `--port` CLI flag, preventing managed port allocation. (2) The plugin returns `$0` when pricing is missing, violating the contract that requires empty results for fallback. (3) Several services are hardcoded to `$0` in Actual Cost mode. We will implement `flag.Parse()` for port binding, a custom `PricingUnavailableError` for contract-compliant fallback signaling, and unified resource routing for Actual Cost estimation.

## Technical Context

**Language/Version**: Go 1.25+
**Primary Dependencies**: gRPC, finfocus-spec/sdk/go/pluginsdk
**Storage**: N/A (Embedded pricing data)
**Testing**: Go testing (unit + integration)
**Target Platform**: Linux, macOS, Windows (cross-compiled)
**Project Type**: single (Go gRPC Plugin)
**Performance Goals**: Startup < 1s, RPC latency < 100ms
**Constraints**: < 250MB binary size, < 400MB memory footprint
**Scale/Scope**: Unified routing for all supported AWS services (EC2, EBS, EKS, S3, Lambda, RDS, DynamoDB, ELB, NATGW)

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **I. Code Quality & Simplicity**: PASS. The solution uses standard Go error handling and SDK patterns.
- **II. Testing Discipline**: PASS. New tests for error cases and routing are planned.
- **III. Protocol & Interface Consistency**: PASS. Fixes a gRPC contract violation and respects port binding protocols.
- **IV. Performance & Reliability**: PASS. No changes to the core pricing lookup performance; improves reliability via correct fallback.
- **V. Build & Release Quality**: PASS. Follows existing build tags and region management.

## Project Structure

### Documentation (this feature)

```text
specs/001-fix-port-contract-bugs/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── checklists/
│   └── requirements.md  # Spec quality checklist
└── spec.md              # Feature specification
```

### Source Code (repository root)

```text
cmd/finfocus-plugin-aws-public/
└── main.go              # Added flag.Parse() and ParsePortFlag()

internal/plugin/
├── errors.go            # [NEW] Defined PricingUnavailableError
├── plugin.go            # Updated GetActualCost to handle PricingUnavailableError
├── projected.go         # Updated estimators to return PricingUnavailableError; updated GetProjectedCost
├── actual.go            # Fixed routing in getProjectedForResource
├── errors_test.go       # [NEW] Error type tests
├── actual_test.go       # Regression tests for routing and fallback
└── projected_test.go    # Regression tests for $0 backward compatibility
```

**Structure Decision**: Single project (Standard Go layout).

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| Unified Routing in `actual.go` | Ensures `GetActualCost` always matches `GetProjectedCost` accuracy for all services. | Keeping `estimateStub` is simpler but leads to contract violations ($0 results) when pricing is available. |
| Custom Error Type | Allows unambiguous signaling of "data missing" vs "system error" to trigger correct gRPC response (empty vs error). | Using sentinel errors or $0 checks is brittle and fails for complex/partial services like DynamoDB. |
