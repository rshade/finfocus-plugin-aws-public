# Implementation Plan - Add us-west-1 (N. California) Region Support

**Feature Branch**: `033-add-us-west-1`
**Feature Spec**: `specs/033-add-us-west-1/spec.md`

## Technical Context

### Technical Stack

- **Language**: Go 1.25+
- **Protocol**: gRPC (CostSourceService from finfocus.v1)
- **Data Source**: Embedded JSON pricing data (via `//go:embed`)
- **Distribution**: Docker (multi-stage build with regional binaries)
- **Deployment**: AWS ECS (via Docker image)

### Architecture Overview

This feature follows the established regional expansion pattern in `finfocus-plugin-aws-public`. The architecture relies on:
1.  **Region-Specific Binaries**: A dedicated binary `finfocus-plugin-aws-public-us-west-1` that embeds pricing data specific to N. California.
2.  **Build Tagging**: Uses Go build tags (`region_usw1`) to select the correct embedded data at compile time.
3.  **Pricing Generator**: The `tools/generate-pricing` tool fetches raw AWS pricing API data and stores it in `data/` for embedding.
4.  **Carbon Estimation**: Uses region-specific grid emission factors (CAISO for N. California).

### Core Dependencies

- **AWS Pricing API**: Source of truth for cost data (OnDemand, standard rates).
- **Carbon Factors**: Fixed constants in `internal/carbon/grid_factors.go`.
- **Docker**: Distribution mechanism requiring multi-binary orchestration in `entrypoint.sh`.

### Key Design Decisions

1.  **Explicit Error Handling**: As clarified in the spec, requests for resources unsupported in `us-west-1` will return `UnsupportedResource` (Code 9) rather than $0.00.
2.  **Fail-Fast Build**: If pricing data cannot be generated during the build, the process fails immediately to prevent stale data release.
3.  **Port Allocation**: `us-west-1` will use port **8010** to avoid conflicts with existing regions (8001-8009).

---

## Constitution Check

### Principle I: Code Quality & Simplicity

- [x] **KISS**: Uses existing patterns; no new frameworks or abstractions.
- [x] **Single Responsibility**: The new region is just data and config; logic remains shared.
- [x] **Stateless**: No new state introduced.

### Principle II: Testing Discipline

- [x] **Unit Tests**: Will add `_test.go` files for the new region tags.
- [x] **Integration Tests**: Docker health checks verify the binary runs.
- [x] **No Mocking**: Uses real embedded data or standard test fixtures.

### Principle III: Protocol & Interface Consistency

- [x] **gRPC Protocol**: strictly adheres to `CostSourceService`.
- [x] **Error Codes**: Uses standard proto `ErrorCode` enum.
- [x] **Logging**: Uses `zerolog` to stderr.

### Principle IV: Performance & Reliability

- [x] **Binary Size**: `us-west-1` pricing data is expected to be ~150MB, well within the <250MB limit.
- [x] **Memory**: Mapped pricing data fits within <400MB constraints.
- [x] **Latency**: Pre-computation/embedding ensures <100ms RPC response.

### Principle V: Build & Release Quality

- [x] **Linting**: Standard `golangci-lint` applies.
- [x] **GoReleaser**: Configuration will be updated for the new artifact.
- [x] **Tags**: `region_usw1` tag ensures clean compilation.

---

## Gated Phases

### Phase 0: Research & Validation

**Goal**: Verify external dependencies and internal configuration requirements.

#### Research Tasks

- [x] **Task**: Verify AWS Pricing API availability and data volume for `us-west-1` to ensure it fits binary size limits. *(Completed: See research.md Section 1)*
- [x] **Task**: Confirm CAISO grid emission factor value in `internal/carbon/grid_factors.go`. *(Completed: See research.md Section 2 - value is 0.000322)*
- [x] **Task**: Verify port 8010 is free and not reserved by any other process or convention in the container. *(Completed: See research.md Section 3)*

#### Outcome
- `specs/033-add-us-west-1/research.md` created.
- All technical unknowns resolved.

### Phase 1: Design & Contracts

**Goal**: Define data models and interfaces.

#### Design Tasks

- [x] **Task**: Create `specs/033-add-us-west-1/data-model.md` (mostly validating existing schemas against new region data). *(Completed: See data-model.md)*
- [ ] **Task**: Update `internal/pricing/regions.yaml` schema definition.
- [x] **Task**: Draft `contracts/` updates (if any - likely none as gRPC is stable). *(Completed: No changes needed - gRPC protocol unchanged)*

#### Outcome
- `specs/033-add-us-west-1/data-model.md`
- Agent context updated.

### Phase 2: Implementation Breakdown

**Goal**: Create actionable tasks for the `speckit.tasks` agent.

#### Implementation Scope

1.  **Configuration**: Update `regions.yaml`, `.goreleaser.yaml`.
2.  **Data Generation**: Run `generate-pricing`, `generate-embeds`.
3.  **Code**: Update `grid_factors.go`, add `embed_usw1.go`.
4.  **Distribution**: Update `Dockerfile`, `entrypoint.sh`, `healthcheck.sh`.
5.  **Documentation**: Update `CLAUDE.md`, `README.md`.

#### Complexity Tracking

| Complexity Score | Component | Justification |
| :--- | :--- | :--- |
| Low | Configuration | Simple YAML/Text updates. |
| Low | Data Gen | Existing tools do the heavy lifting. |
| Low | Code | Minimal Go code; mostly generated. |
| Low | Docker | simple list expansion. |

**Total Complexity**: Low (Standard region addition).
