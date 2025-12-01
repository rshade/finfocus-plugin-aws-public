# Implementation Plan: Automated Build Matrix for AWS Regions

**Branch**: `006-region-build-matrix` | **Date**: 2025-11-30 | **Spec**: specs/006-region-build-matrix/spec.md
**Input**: Feature specification from `/specs/006-region-build-matrix/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/commands/plan.md` for the execution workflow.

## Summary

Create an automated system that generates build configurations, embed files, and tests from a central regions.yaml file, reducing manual errors and ensuring consistency across AWS regions while respecting build image disk constraints.

## Technical Context

<!--
  ACTION REQUIRED: Replace the content in this section with the technical details
  for the project. The structure here is presented in advisory capacity to guide
  the iteration process.
-->

**Language/Version**: Go 1.25.4
**Primary Dependencies**: GoReleaser, gRPC, build tags
**Storage**: Embedded JSON files in Go binaries
**Testing**: Go testing framework
**Target Platform**: Linux/Darwin/Windows binaries
**Project Type**: CLI/plugin
**Performance Goals**: Region addition <5 minutes, verification script <30 seconds
**Constraints**: Build image disk space limits requiring sequential region builds
**Scale/Scope**: 9 AWS regions, automated generation scripts

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Code Quality & Simplicity**: PASS - Feature maintains KISS principle with simple YAML-driven generation
- **Testing Discipline**: PASS - Existing test structure preserved, new scripts will be testable
- **Protocol & Interface Consistency**: PASS - No changes to gRPC protocol, maintains region-specific binaries
- **Performance & Reliability**: PASS - Respects latency targets, maintains embedded data approach
- **Build & Release Quality**: PASS - Uses existing GoReleaser patterns, maintains lint/test requirements
- **Security Requirements**: PASS - No new network calls, maintains embedded data security
- **Development Workflow**: PASS - Follows existing branch/PR/commit conventions

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

```text
scripts/
├── generate-regions.sh    # New: Generate embed files and goreleaser config from regions.yaml
├── verify-regions.sh      # New: Verification script for region configurations
└── [existing scripts]

internal/pricing/
├── embed_*.go             # Existing: Region-specific embed files (auto-generated)
├── regions.yaml           # New: Central region configuration
└── [existing files]

.github/workflows/
└── [existing CI/CD files, updated for region matrix]
```

**Structure Decision**: Extends existing structure with new generation scripts and central config file, maintaining compatibility with current build system.

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| [e.g., 4th project] | [current need] | [why 3 projects insufficient] |
| [e.g., Repository pattern] | [specific problem] | [why direct DB access insufficient] |
