# Implementation Plan: Support target_resources Scope

**Branch**: `014-target-resources-scope` | **Date**: 2025-12-19 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/014-target-resources-scope/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/commands/plan.md` for the execution workflow.

## Summary

This feature implements batch processing support in the `GetRecommendations` RPC method. It updates the plugin to handle the `target_resources` field in the request (introduced in `pulumicost-spec` v0.4.9). The plugin will iterate over this list, apply any filters (AND logic), and return a list of cost recommendations. It maintains backward compatibility by falling back to `Filter.Sku` if `target_resources` is empty. Crucially, it implements a correlation strategy to map outputs back to inputs using `ResourceId`/`Arn`/`Name` and a consolidated logging strategy to maintain observability without noise.

## Technical Context

**Language/Version**: Go 1.25+
**Primary Dependencies**: `github.com/rshade/pulumicost-spec` (v0.4.9 already in go.mod), `google.golang.org/grpc`
**Storage**: N/A (Stateless)
**Testing**: Go standard library `testing`
**Target Platform**: Linux/Cross-platform (Go binary)
**Project Type**: CLI/Plugin (gRPC Server)
**Performance Goals**: Support batch size up to 100 with negligible overhead (<1ms processing time excluding lookups).
**Constraints**: Must run as a standalone binary, no external network calls at runtime.
**Scale/Scope**: Logic change confined to `GetRecommendations` handler.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Code Quality & Simplicity**: Passed. Implementation is a simple iteration loop.
- **Testing Discipline**: Passed. Unit tests will cover batch logic and filtering.
- **Protocol & Interface Consistency**: Passed. Adheres to gRPC spec, error codes, and uses `zerolog` (updated logging strategy).
- **Performance & Reliability**: Passed. Validates batch size limit (100).
- **Build & Release Quality**: Passed. No changes to build process.
- **Security Requirements**: Passed. No new attack surface; input validation included.

## Project Structure

### Documentation (this feature)

```text
specs/014-target-resources-scope/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output
│   └── pulumicost-v1-batch.yaml
└── tasks.md             # Phase 2 output (to be created)
```

### Source Code (repository root)

```text
src/
├── internal/
│   └── plugin/
│       ├── plugin.go          # Main handler logic update
│       ├── validation.go      # New validation logic for batch size
│       └── plugin_test.go     # New batch tests
```

**Structure Decision**: Option 1: Single project (DEFAULT). Modification of existing internal package.

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| None | N/A | N/A |