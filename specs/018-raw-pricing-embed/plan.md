# Implementation Plan: Embed Raw AWS Pricing JSON Per Service

**Branch**: `018-raw-pricing-embed` | **Date**: 2025-12-20 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/018-raw-pricing-embed/spec.md`

## Summary

Refactor AWS pricing data embedding from a combined blob to per-service raw JSON files.
This eliminates the data combining step that caused v0.0.10/v0.0.11 pricing bugs ($0 returns).
Each service's pricing will be fetched from AWS Price List API and embedded verbatim.

## Technical Context

**Language/Version**: Go 1.25+
**Primary Dependencies**: encoding/json, sync.Once, go:embed, zerolog, gRPC (finfocus-spec)
**Storage**: Embedded JSON files via `//go:embed` (no external storage)
**Testing**: Go testing with build tags (`-tags=region_use1`)
**Target Platform**: Linux/Windows/macOS binaries per region
**Project Type**: Single Go module with region-specific binaries
**Performance Goals**: Initialization < 2s, pricing lookup < 100ms
**Constraints**: Binary size ~16MB per region, thread-safe concurrent access
**Scale/Scope**: 7 services × 12 regions = 84 embedded JSON files total

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Code Quality & Simplicity | ✅ PASS | Eliminates combining step = simpler code |
| II. Testing Discipline | ✅ PASS | Per-service threshold tests, unit tests for parsing |
| III. Protocol & Interface Consistency | ✅ PASS | No gRPC API changes, backward compatible |
| IV. Performance & Reliability | ✅ PASS | Same sync.Once pattern, thread-safe |
| V. Build & Release Quality | ✅ PASS | Build tags preserved, goreleaser compatible |
| Security Requirements | ✅ PASS | No runtime network calls, embedded data only |
| Development Workflow | ✅ PASS | Branch naming, conventional commits |

**No violations.** Gate passed.

## Project Structure

### Documentation (this feature)

```text
specs/018-raw-pricing-embed/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # N/A - no API changes
└── tasks.md             # Phase 2 output (/speckit.tasks command)
```

### Source Code (repository root)

```text
internal/pricing/
├── client.go            # MODIFY: Parse multiple service files
├── client_test.go       # MODIFY: Add per-service parsing tests
├── types.go             # NO CHANGE: awsPricing struct already matches AWS format
├── embed_use1.go        # REWRITE: Multiple //go:embed directives per service
├── embed_usw2.go        # REWRITE: Multiple //go:embed directives per service
├── embed_euw1.go        # REWRITE: Multiple //go:embed directives per service
├── embed_*.go           # REWRITE: Same pattern for all 12 regions
├── embed_fallback.go    # MODIFY: Fallback for testing without real data
├── embed_test.go        # MODIFY: Per-service threshold tests
├── data/                # Generated pricing files (not in git)
│   ├── ec2_us-east-1.json
│   ├── s3_us-east-1.json
│   ├── rds_us-east-1.json
│   ├── eks_us-east-1.json
│   ├── lambda_us-east-1.json
│   ├── dynamodb_us-east-1.json
│   ├── elb_us-east-1.json
│   └── ... (7 files × 12 regions)
└── regions.yaml         # NO CHANGE: Region configuration

tools/generate-pricing/
└── main.go              # REWRITE: Save per-service files instead of combining
```

**Structure Decision**: Reuses existing package layout. Changes are localized to
`internal/pricing/` (embed files, client init) and `tools/generate-pricing/` (output format).

## Complexity Tracking

> **No violations to justify.** Constitution check passed without complexity exceptions.
