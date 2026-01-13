# Implementation Plan: EKS Cluster Cost Estimation

**Branch**: `010-eks-cost-estimation` | **Date**: 2025-12-06 | **Spec**: /mnt/c/GitHub/go/src/github.com/rshade/finfocus-plugin-aws-public/specs/010-eks-cost-estimation/spec.md
**Input**: Feature specification from `/specs/010-eks-cost-estimation/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/commands/plan.md` for the execution workflow.

## Summary

Implement EKS cluster cost estimation as a new resource type in the AWS Public plugin. Add "eks" support with fixed hourly rates ($0.10 standard, $0.50 extended) using embedded pricing data, following existing EC2 patterns.

## Technical Context

<!--
  ACTION REQUIRED: Replace the content in this section with the technical details
  for the project. The structure here is presented in advisory capacity to guide
  the iteration process.
-->

**Language/Version**: Go 1.25.4
**Primary Dependencies**: gRPC, zerolog, embedded JSON pricing data
**Storage**: Embedded JSON pricing data (no runtime storage)
**Testing**: Go testing framework
**Target Platform**: Linux (gRPC service binaries)
**Project Type**: Single Go project (plugin)
**Performance Goals**: <100ms per GetProjectedCost RPC, <500ms startup
**Constraints**: Thread-safe concurrent access, <50MB memory footprint, <10MB binary size
**Scale/Scope**: 9 regional binaries, support 1000+ concurrent RPC calls

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

PASS - Feature follows existing patterns (embedded pricing, gRPC methods, thread-safe lookups). No constitution violations.

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
cmd/finfocus-plugin-aws-public/
├── main.go

internal/
├── plugin/
│   ├── supports.go
│   ├── projected.go
│   ├── actual.go
│   ├── plugin.go
│   └── *_test.go
├── pricing/
│   ├── client.go
│   ├── types.go
│   ├── embed_*.go
│   └── *_test.go

tools/generate-pricing/
└── main.go
```

**Structure Decision**: Single Go project following existing plugin architecture. New EKS support added to existing files (pricing client, plugin handlers).

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| [e.g., 4th project] | [current need] | [why 3 projects insufficient] |
| [e.g., Repository pattern] | [specific problem] | [why direct DB access insufficient] |
