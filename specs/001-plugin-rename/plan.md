# Implementation Plan: Plugin Rename to FinFocus

**Branch**: `001-plugin-rename` | **Date**: 2026-01-11 | **Spec**: spec.md
**Input**: Feature specification from `/specs/001-plugin-rename/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/commands/plan.md` for the execution workflow.

## Summary

Rename the plugin from `finfocus-plugin-aws-public` to `finfocus-plugin-aws-public` across all code, configuration, documentation, and build artifacts. This is a breaking change (v0.2.0) as part of the FinFocus migration, updating module name, dependencies, imports, logging prefixes, and binary naming from `finfocus` to `finfocus`. Technical approach involves systematic text replacement across 40+ files, directory renaming, and verification via existing build/test/lint commands.

## Technical Context

**Language/Version**: Go 1.25.5+
**Primary Dependencies**: gRPC (pluginsdk), finfocus-spec v0.5.0, zerolog
**Storage**: Memory-mapped JSON pricing data (embedded at build time)
**Testing**: Go testing with table-driven tests, make test
**Target Platform**: Linux (region-specific binaries: us-east-1, us-west-2, eu-west-1)
**Project Type**: single (gRPC plugin with embedded data)
**Performance Goals**: Startup <500ms, GetProjectedCost <100ms, Supports <10ms
**Constraints**: No runtime network calls, serve on 127.0.0.1 only, <250MB binary
**Scale/Scope**: 40+ files requiring updates, 3 region binaries

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Check | Status | Notes |
|-----------|-------|--------|-------|
| I. Code Quality & Simplicity | No over-engineering, maintain KISS | ✅ PASS | Refactoring preserves existing patterns |
| I. Code Quality & Simplicity | Single Responsibility Principle | ✅ PASS | No new functionality added |
| I. Code Quality & Simplicity | No magic or side effects | ✅ PASS | Text replacement only |
| II. Testing Discipline | Tests pass after rename | ⚠️ VERIFY | Make test must succeed post-rename |
| II. Testing Discipline | No test code quality degradation | ✅ PASS | Tests unchanged except imports |
| II. Testing Discipline | Fast execution | ✅ PASS | No performance impact |
| III. Protocol & Interface Consistency | gRPC protocol unchanged | ✅ PASS | Proto package rename only (v1 → v1) |
| III. Protocol & Interface Consistency | Zerolog with [finfocus-plugin-aws-public] prefix | ✅ PASS | Logging prefixes updated |
| III. Protocol & Interface Consistency | Thread-safe gRPC handlers | ✅ PASS | No code logic changes |
| III. Protocol & Interface Consistency | Region-specific binaries | ✅ PASS | Build system updated |
| IV. Performance & Reliability | Startup <500ms | ✅ PASS | No performance impact |
| IV. Performance & Reliability | GetProjectedCost <100ms | ✅ PASS | No logic changes |
| IV. Performance & Reliability | Memory <400MB | ✅ PASS | Embedded data unchanged |
| IV. Performance & Reliability | Binary <250MB | ✅ PASS | Data unchanged |
| IV. Performance & Reliability | No pricing data filtering | ✅ PASS | Critical - full data retained |
| V. Build & Release Quality | Make lint passes | ⚠️ VERIFY | Must verify post-rename |
| V. Build & Release Quality | Make test passes | ⚠️ VERIFY | Must verify post-rename |
| V. Build & Release Quality | GoReleaser builds succeed | ⚠️ VERIFY | Must update goreleaser config |
| V. Build & Release Quality | Region build tags work | ⚠️ VERIFY | Must verify all regions |
| V. Build & Release Quality | gRPC functional | ⚠️ VERIFY | Manual test required |
| Security | No credentials in logs | ✅ PASS | No behavior changes |
| Security | No runtime network calls | ✅ PASS | Still embedded data only |
| Security | Serve on 127.0.0.1 only | ✅ PASS | No behavior changes |

**GATE STATUS (Post-Phase 1)**: ⚠️ **CONDITIONAL PASS** - All principles satisfied, design review complete with no changes to architecture or data model. Verification gates (lint, test, build, gRPC) must be confirmed in Phase 4 of implementation. No constitution violations requiring justification.

**Design Review Summary**:
- Data model unchanged (no new entities or relationships)
- gRPC interface unchanged (proto package rename only)
- Performance characteristics preserved
- Security posture unchanged
- Build system updates are mechanical (naming only)

## Project Structure

### Documentation (this feature)

```text
specs/001-plugin-rename/
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 output (/speckit.plan command)
├── data-model.md        # Phase 1 output (/speckit.plan command)
├── quickstart.md        # Phase 1 output (/speckit.plan command)
├── contracts/           # Phase 1 output (/speckit.plan command)
└── tasks.md             # Phase 2 output (/speckit.tasks command - NOT created by /speckit.plan)
```

### Source Code (repository root)

```text
cmd/finfocus-plugin-aws-public/  # RENAMED TO cmd/finfocus-plugin-aws-public/
└── main.go                        # Entry point (updated for finfocus naming)

internal/
├── pricing/                       # Pricing lookup logic
│   ├── loader.go                  # Embedded data loading
│   ├── lookup.go                  # Indexed lookups
│   └── test/
├── plugin/                        # gRPC service implementation
│   ├── server.go                  # gRPC handlers
│   ├── costs.go                   # Cost calculations
│   └── test/
└── tools/                         # Build tools
    └── generate-pricing/           # Pricing data generator

tools/generate-pricing/            # Pricing data generation tools
└── main.go

go.mod                             # Module name: finfocus-plugin-aws-public
Makefile                           # Build targets (updated for finfocus naming)
.goreleaser.yaml                   # Release config (updated for finfocus naming)

examples/                          # Integration test examples
└── example/

internal/testdata/                 # Test pricing data (region-specific embed)
├── pricing_use1.json              # us-east-1 pricing
├── pricing_usw2.json              # us-west-2 pricing
└── pricing_euw1.json              # eu-west-1 pricing
```

**Structure Decision**: Go gRPC plugin with embedded pricing data. This is a single project type (no separate frontend/backend). The primary structure change is renaming `cmd/finfocus-plugin-aws-public/` to `cmd/finfocus-plugin-aws-public/` and updating the module name in `go.mod`. All other directories remain unchanged; only imports and references within files are updated.

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

No constitution violations requiring justification. This is a systematic rename operation with no architecture changes.

## Phase 0: Research

### Research Tasks

Since Technical Context has no NEEDS CLARIFICATION markers, no external research is required. All technical decisions are established:

- **Language/Version**: Go 1.25.5+ (existing)
- **Primary Dependencies**: gRPC (pluginsdk), finfocus-spec v0.5.0 (replacing finfocus-spec v0.4.14)
- **Storage**: Memory-mapped JSON pricing data (existing)
- **Testing**: Go testing with table-driven tests (existing)
- **Target Platform**: Linux region-specific binaries (existing)
- **Project Type**: Single gRPC plugin (existing)
- **Performance Goals**: Unchanged (<500ms startup, <100ms GetProjectedCost)
- **Constraints**: Unchanged (<250MB binary, <400MB memory)

### Research Findings

**No research tasks executed** - all technical context is known from existing codebase and RENAME-PLAN.md Phase 3 requirements.

## Phase 1: Design

### Data Model

**Status**: No changes required. This is a rename operation only.

The plugin uses embedded pricing data with these entities (unchanged):
- **PricingData**: Raw AWS pricing JSON parsed from embedded files
- **Product**: AWS service product definitions (EC2, S3, RDS, EKS, Lambda, DynamoDB, ELB)
- **PriceTerm**: OnDemand price terms with currency and price values
- **ResourceDescriptor**: Input from FinFocus core (via proto)
- **CostResponse**: Output to FinFocus core (via proto)

All entities remain structurally identical; only import paths change from `finfocus.v1` to `finfocus.v1`.

### API Contracts

**Status**: No changes required. gRPC interface is unchanged.

The plugin implements the `CostSourceService` interface from `finfocus.v1` (formerly `finfocus.v1`):
- `Name()` - Returns service name
- `Supports(ResourceDescriptor)` - Checks region and resource type
- `GetProjectedCost(ResourceDescriptor)` - Calculates projected costs
- `GetActualCost(GetActualCostRequest)` - Returns actual costs with fallback

Proto message structure is identical; only package name changes.

### Quick Start Guide

**Status**: See `quickstart.md` for developer onboarding.

## Generated Artifacts

- `research.md` - Phase 0 research findings
- `data-model.md` - Phase 1 data model documentation
- `quickstart.md` - Phase 1 developer quick start
- `contracts/` - Phase 1 API contracts (no changes for this rename)
