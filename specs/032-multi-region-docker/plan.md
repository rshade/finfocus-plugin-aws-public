# Implementation Plan: Multi-Region Docker Image

**Branch**: `032-multi-region-docker` | **Date**: 2026-01-14 | **Spec**: [specs/032-multi-region-docker/spec.md](./spec.md)
**Input**: Feature specification from `/specs/032-multi-region-docker/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/commands/plan.md` for the execution workflow.

## Summary

The goal is to package the `finfocus-plugin-aws-public` into a single Docker image containing binaries for all 12 supported AWS regions. This simplifies Kubernetes deployments by providing a "one-stop" artifact. The image must run as a non-root user, manage 12 subprocesses via a custom entrypoint (handling graceful shutdown), and expose individual ports for each region plus a consolidated metrics endpoint.

## Technical Context

**Language/Version**: Dockerfile, Bash (entrypoint), YAML (GitHub Actions)
**Primary Dependencies**: `alpine:3.19`, `curl` (healthchecks), custom metrics aggregator (Go)
**Storage**: N/A (Stateless container)
**Testing**: Local Docker execution, Kubernetes (Kind/Minikube), Container Structure Tests
**Target Platform**: Kubernetes / Docker (Linux/amd64, arm64)
**Project Type**: Infrastructure / Containerization
**Performance Goals**: 
- Container startup < 10s
- Graceful shutdown < 5s
- Support 12 concurrent processes
**Constraints**: 
- Non-root user (UID 65532)
- **RESOLVED SIZE CONSTRAINT**: Spec updated to ~2.0GB image to accommodate embedded pricing data (12 * 150MB) for all regions.
**Scale/Scope**: 12 concurrent binary processes within one container.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Check | Status | Notes |
|-----------|-------|--------|-------|
| **I. Code Quality** | Simple, Single Responsibility | ⚠️ WARNING | Running 12 processes in one container challenges "Single Responsibility". Aggregating metrics adds complexity. |
| **II. Testing** | Unit/Integration Tests | ✅ PASS | Can test entrypoint logic and final image. |
| **III. Protocol** | gRPC/HTTP Standards | ✅ PASS | Ports 8001-8012 exposed as requested. |
| **IV. Performance** | Resource Limits & Size | ✅ PASS | **SC-001** updated to ~2.0GB to accommodate embedded pricing data (12 * 150MB). |

**Gate Decision**: **PASS** - Spec updated to reflect realistic image size.

## Project Structure

### Documentation (this feature)

```text
specs/032-multi-region-docker/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output
└── tasks.md             # Phase 2 output
```

### Source Code (repository root)

```text
# Containerization files
build/
├── Dockerfile
├── entrypoint.sh
└── healthcheck.sh

# CI/CD
.github/
└── workflows/
    └── docker-publish.yml

# Tools (if needed for metrics aggregation)
cmd/
└── metrics-aggregator/ (Potential need for FR-019)

# Tests
test/
└── k8s/
    └── deployment.yaml
```

**Structure Decision**: Standard Docker build context in `build/` (or root if preferred, but `build/` keeps it clean) with CI in `.github/workflows`.

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| Multi-process Container | Requirement to have single artifact for all regions | Running 12 separate pods (standard K8s pattern) was rejected by User Story 1 "manage one artifact". |
| Metrics Aggregator | FR-019 requires single port 9090 for all regions | Scraping 12 ports individually is standard but spec requires aggregation. |
