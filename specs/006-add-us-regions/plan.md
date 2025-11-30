# Implementation Plan: Add support for additional US regions (us-west-1, us-gov-west-1, us-gov-east-1)

**Branch**: `006-add-us-regions` | **Date**: 2025-11-30 | **Spec**: specs/006-add-us-regions/spec.md
**Input**: Feature specification from `/specs/006-add-us-regions/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/commands/plan.md` for the execution workflow.

## Summary

Add pricing data and build configurations for us-west-1, us-gov-west-1, and us-gov-east-1 regions to the PulumiCost AWS plugin. Implement region-specific binaries with embedded pricing data, following the existing pattern of build tags and GoReleaser configuration.

## Technical Context

<!--
  ACTION REQUIRED: Replace the content in this section with the technical details
  for the project. The structure here is presented in advisory capacity to guide
  the iteration process.
-->

**Language/Version**: Go 1.25.4 minimum
**Primary Dependencies**: gRPC (pulumicost.v1), pluginsdk, embedded pricing data
**Storage**: Embedded pricing data (no external storage)
**Testing**: Go testing framework, integration tests for gRPC methods
**Target Platform**: Linux (cross-compiled Go binaries)
**Project Type**: Single project (gRPC plugin service)
**Performance Goals**: GetProjectedCost() < 100ms, Supports() < 10ms, startup < 500ms
**Constraints**: Thread-safe concurrent RPC calls, region-specific binaries with build tags, embedded data only
**Scale/Scope**: Support 3 additional regions, handle 100+ concurrent RPC calls

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

✅ **PASS**: Feature complies with all constitution principles

- **Code Quality & Simplicity**: Adding region support follows existing patterns (build tags, embed files) without introducing complexity
- **Testing Discipline**: Will add unit and integration tests following existing patterns
- **Protocol & Interface Consistency**: Uses existing gRPC methods, adds new build tags for regions
- **Performance & Reliability**: Maintains embedded data approach and performance targets
- **Build & Release Quality**: Extends existing GoReleaser configuration for new regions
- **Security**: No changes to security model, maintains embedded data approach
- **Development Workflow**: Follows established branch naming and commit conventions

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
<!--
  ACTION REQUIRED: Replace the placeholder tree below with the concrete layout
  for this feature. Delete unused options and expand the chosen structure with
  real paths (e.g., apps/admin, packages/something). The delivered plan must
  not include Option labels.
-->

```text
# Go project structure (single binary plugin)
cmd/pulumicost-plugin-aws-public/
└── main.go                    # Plugin entry point

internal/
├── plugin/
│   ├── plugin.go              # gRPC service implementation
│   ├── supports.go            # Supports() method
│   ├── estimate.go            # GetProjectedCost() method
│   ├── actual.go              # GetActualCost() method
│   ├── supports_test.go       # Unit tests
│   ├── estimate_test.go       # Unit tests
│   └── integration_test.go    # Integration tests
└── pricing/
    ├── client.go              # Pricing data client
    ├── types.go               # Pricing data types
    ├── embed_use1.go          # Existing us-east-1 pricing data
    ├── embed_usw2.go          # Existing us-west-2 pricing data
    ├── embed_euw1.go          # Existing eu-west-1 pricing data
    ├── embed_usw1.go          # NEW: us-west-1 pricing data
    ├── embed_govw1.go         # NEW: us-gov-west-1 pricing data
    ├── embed_gove1.go         # NEW: us-gov-east-1 pricing data
    └── client_test.go         # Unit tests

scripts/
├── build-region.sh           # Build script for regions
├── region-tag.sh             # Region tag management
└── release-region.sh         # Release script for regions

tools/generate-pricing/
└── main.go                   # Pricing data generator

.goreleaser.yaml              # UPDATED: New region builds
Makefile                      # Build targets
```

**Structure Decision**: Single Go project following existing repository structure. New embed files added to internal/pricing/ following the established pattern. Build configuration updated in .goreleaser.yaml to include new region binaries.

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| [e.g., 4th project] | [current need] | [why 3 projects insufficient] |
| [e.g., Repository pattern] | [specific problem] | [why direct DB access insufficient] |
