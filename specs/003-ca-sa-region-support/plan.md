# Implementation Plan: Canada and South America Region Support

**Branch**: `003-ca-sa-region-support` | **Date**: 2025-11-29 | **Spec**: [specs/003-ca-sa-region-support/spec.md](specs/003-ca-sa-region-support/spec.md)
**Input**: Feature specification from `/specs/003-ca-sa-region-support/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/commands/plan.md` for the execution workflow.

## Summary

Add support for AWS regions `ca-central-1` (Canada Central) and `sa-east-1` (South America / São Paulo) by creating region-specific embedded pricing data files, updating build configurations, and ensuring proper test coverage. This follows the existing pattern used for `us-east-1`, `us-west-2`, etc.

## Technical Context

**Language/Version**: Go 1.25+
**Primary Dependencies**: `rs/zerolog` (logging), `google.golang.org/grpc` (protocol), existing embedded JSON mechanism.
**Storage**: In-memory (embedded JSON pricing data).
**Testing**: Go standard library testing (`testing` package), `testify/assert` (if already used), integration tests via `go test -tags=integration`.
**Target Platform**: Linux/Darwin/Windows (cross-compiled binaries via GoReleaser).
**Project Type**: CLI / gRPC Plugin.
**Performance Goals**: < 100ms per cost lookup, < 50MB binary size.
**Constraints**: Thread-safe access to pricing data; distinct binary per region.
**Scale/Scope**: 2 new regions, ~50-100kb of pricing data each.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
| :--- | :--- | :--- |
| I. Code Quality | PASS | Uses simple file embedding pattern; no complex abstraction. |
| II. Testing | PASS | Standard unit/integration tests required; strictly enforced. |
| III. Protocol | PASS | strictly adheres to existing `CostSourceService` gRPC definition. |
| IV. Performance | PASS | Embedded data ensures fast lookup and reliability. |
| V. Build Quality | PASS | GoReleaser config will be updated for new targets. |
| Security | PASS | No runtime network calls; read-only embedded data. |

## Project Structure

### Documentation (this feature)

```text
specs/003-ca-sa-region-support/
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 output (/speckit.plan command)
├── data-model.md        # Phase 1 output (/speckit.plan command)
├── quickstart.md        # Phase 1 output (/speckit.plan command)
├── contracts/           # Phase 1 output (/speckit.plan command)
└── tasks.md             # Phase 2 output (/speckit.tasks command - NOT created by /speckit.plan)
```

### Source Code (repository root)

```text
src/
├── cmd/
│   └── finfocus-plugin-aws-public/  # Main entry point (unchanged)
├── internal/
│   ├── pricing/
│   │   ├── embed_cac1.go              # NEW: Canada Central embed
│   │   └── embed_sae1.go              # NEW: South America embed
│   └── plugin/                        # Existing plugin logic (unchanged)
├── tools/
│   └── generate-pricing/              # Updates to support new regions
├── .goreleaser.yaml                   # Updates for new build targets
└── Makefile                           # Updates for build commands
```

**Structure Decision**: Follows existing "Option 1: Single project" structure with region-specific build tags and embed files.

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| None | N/A | N/A |