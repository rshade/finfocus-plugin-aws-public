# Implementation Plan: Canada and South America Region Support

**Branch**: `003-ca-sa-region-support` | **Date**: 2025-11-20 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/003-ca-sa-region-support/spec.md`

## Summary

Add AWS region support for ca-central-1 (Canada) and sa-east-1 (South America) by creating region-specific embed files with build tags, updating GoReleaser configuration, extending the pricing generator tool, and adding comprehensive tests following established patterns from 002-ap-region-support.

## Technical Context

**Language/Version**: Go 1.25+
**Primary Dependencies**: pulumicost-core/pkg/pluginsdk, pulumicost-spec/sdk/go/proto
**Storage**: Embedded JSON files (go:embed) - no external storage
**Testing**: Go testing with table-driven tests, make test
**Target Platform**: Linux server (gRPC service)
**Project Type**: Single project - gRPC plugin
**Performance Goals**: GetProjectedCost <100ms, Supports <10ms, startup <500ms
**Constraints**: Binary size <20MB, memory <50MB, 100+ concurrent RPCs
**Scale/Scope**: 2 new regions (ca-central-1, sa-east-1)

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Code Quality & Simplicity | ✅ PASS | Follows existing region pattern, no new abstractions |
| II. Testing Discipline | ✅ PASS | Table-driven tests, unit + integration coverage |
| III. Protocol & Interface Consistency | ✅ PASS | Uses proto ErrorCode enum, gRPC methods unchanged |
| IV. Performance & Reliability | ✅ PASS | sync.Once parsing, indexed lookups, thread-safe |
| V. Build & Release Quality | ✅ PASS | GoReleaser config, make lint/test gates |

**All gates pass. No constitution violations.**

## Project Structure

### Documentation (this feature)

```text
specs/003-ca-sa-region-support/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output (N/A - no new APIs)
└── tasks.md             # Phase 2 output (/speckit.tasks command)
```

### Source Code (repository root)

```text
internal/
├── pricing/
│   ├── client.go           # Existing pricing client
│   ├── embed_cac1.go       # NEW: ca-central-1 embed
│   ├── embed_sae1.go       # NEW: sa-east-1 embed
│   └── embed_fallback.go   # UPDATE: exclude new tags
├── plugin/
│   ├── plugin.go           # Existing plugin implementation
│   ├── supports.go         # No changes needed
│   └── projected.go        # No changes needed
tools/
└── generate-pricing/
    └── main.go             # UPDATE: add ca-central-1, sa-east-1
data/
├── aws_pricing_ca-central-1.json  # GENERATED
└── aws_pricing_sa-east-1.json     # GENERATED
.goreleaser.yaml                   # UPDATE: add 2 build targets
```

**Structure Decision**: Single project structure (Option 1). This feature extends existing internal packages without adding new top-level directories.

## Complexity Tracking

No constitution violations to justify. This implementation follows established patterns.
