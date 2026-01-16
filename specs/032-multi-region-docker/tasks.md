# Implementation Tasks: Multi-Region Docker Image

**Feature**: `032-multi-region-docker`
**Status**: Completed

## Phase 1: Setup
*Initialize project structure and directories.*

- [x] T001 Create build directory for Docker resources in `build/`
- [x] T002 Create directory for metrics aggregator tool in `cmd/metrics-aggregator/`
- [x] T003 Create directory for contracts if missing in `specs/032-multi-region-docker/contracts/`
- [x] T004 Create test directory for k8s manifests in `test/k8s/`

## Phase 2: Foundational
*Build dependencies and scripts required for the Docker image.*

- [x] T005 [P] Implement metrics aggregator main logic in `cmd/metrics-aggregator/main.go` (scrape 12 ports, expose on 9090)
- [x] T006 [P] Implement metrics aggregator configuration/flags in `cmd/metrics-aggregator/config.go`
- [x] T007 [P] Create and run unit tests for metrics aggregator in `cmd/metrics-aggregator/aggregator_test.go`
- [x] T008 [P] Create entrypoint script in `build/entrypoint.sh` (start 12 binaries, inject region into JSON logs, env var propagation)
- [x] T009 [P] Implement signal handling and graceful shutdown logic in `build/entrypoint.sh` (trap SIGTERM)
- [x] T010 [P] Implement binary retry logic in `build/entrypoint.sh` (retry 3 times on failure)
- [x] T011 [P] Create healthcheck script in `build/healthcheck.sh` (verify all 12 endpoints)
- [x] T012 [P] Define Dockerfile in `build/Dockerfile` (Alpine 3.19 base, Tini, dependencies)

## Phase 3: User Story 1 - Deploy to Kubernetes (Priority: P1)
*As a DevOps engineer, I want to deploy the plugin to a Kubernetes cluster using a single Docker image.*

- [x] T013 [US1] Update Dockerfile to accept VERSION build arg and download 12 release tarballs in `build/Dockerfile`
- [x] T014 [US1] Update Dockerfile to compile and install metrics-aggregator in `build/Dockerfile`
- [x] T015 [US1] Configure Dockerfile to use non-root user (UID 65532) and set entrypoint in `build/Dockerfile`
- [ ] T016 [US1] Build Docker image locally with test version tag (manual step)
- [ ] T017 [US1] Verify image size meets revised requirement (~2.0GB) (manual step)
- [ ] T018 [US1] Run container locally and verify all 12 regional ports (8001-8012) are listening
- [ ] T019 [US1] Verify metrics endpoint on port 9090 returns aggregated metrics
- [ ] T020 [US1] Verify log output contains injected region field (e.g. `{"region":"us-east-1"...}`)
- [x] T021 [US1] Run local security scan (Trivy) on built image in `test/security/scan.sh` (create script if needed)
- [x] T022 [US1] Create local K8s deployment manifest for testing in `test/k8s/deployment.yaml`
- [ ] T023 [US1] Verify pod startup and readiness probe in local cluster (manual step)

## Phase 4: User Story 2 - Run Locally for Testing (Priority: P2)
*As a developer, I want to run the plugin locally using Docker so that I can verify behavior.*

- [ ] T024 [US2] Verify container starts with default env vars (`FINFOCUS_PLUGIN_WEB_ENABLED=true`)
- [ ] T025 [US2] Test graceful shutdown by sending SIGTERM to running container (SC-003)
- [ ] T026 [US2] Verify exit code is 0 after graceful shutdown

## Phase 5: Polish & Cross-Cutting
*CI/CD integration and final documentation.*

- [x] T027 Create GitHub Action workflow for Docker publish in `.github/workflows/docker-publish.yml`
- [ ] T028 Configure workflow triggers (release creation, workflow_dispatch) in `.github/workflows/docker-publish.yml`
- [ ] T029 Add build steps to GitHub Action (login to GHCR, build with tags) in `.github/workflows/docker-publish.yml`
- [ ] T030 Add step to install and run `golangci-lint` for aggregator code in `.github/workflows/docker-publish.yml`
- [ ] T031 Add step to verify image size does not exceed limit (warn if >2.2GB) in `.github/workflows/docker-publish.yml`
- [ ] T032 Add step to check regional binary sizes during CI (alert if >240MB per binary) in `.github/workflows/docker-publish.yml`
- [x] T033 Update README.md with Docker usage instructions in `README.md`
- [x] T034 Write integration test for Docker image build and verification in `test/integration/docker_test.go`
- [x] T035 Write integration test for Kubernetes deployment with Kind in `test/integration/kind_test.go`
- [x] T036 Update GitHub workflow to integrate with release-please in `.github/workflows/docker-publish.yml`

## Dependencies

1. **Phase 2 (Foundational)** must be completed before **Phase 3 (US1)** because the Dockerfile depends on the scripts and aggregator tool.
2. **Phase 3 (US1)** validates the core image functionality required for **Phase 4 (US2)**.
3. **Phase 5 (Polish)** depends on the finalized Dockerfile and build process from previous phases.

## Parallel Execution Opportunities

- **Phase 2**: `metrics-aggregator` (T005-T007) can be developed in parallel with shell scripts (T008-T011) and Dockerfile definition (T012).
- **Phase 3**: K8s manifest creation (T022) can be done in parallel with Docker build testing.

## Implementation Strategy

1. **MVP (Phase 2 & 3)**: Focus on getting a functional Docker image that runs all 12 binaries and the metrics aggregator. This delivers the core value of "single artifact".
2. **Robustness (Phase 4)**: Ensure the "happy path" isn't the only path working—verify shutdown and signals.
3. **Automation (Phase 5)**: Once the build process is proven locally, automate it in GitHub Actions.
