# Feature Specification: Multi-Region Docker Image

**Feature Branch**: `032-multi-region-docker`
**Created**: 2026-01-14
**Status**: Draft
**Input**: User description provided via CLI.

## User Scenarios & Testing

### User Story 1 - Deploy to Kubernetes (Priority: P1)

As a DevOps engineer, I want to deploy the plugin to a Kubernetes cluster using a single Docker image so that I can manage one artifact for all 12 supported regions without complex build pipelines.

**Why this priority**: This is the primary driver for the feature; enabling containerized deployments.

**Independent Test**: Can be tested by deploying the image to a local K8s cluster (e.g., Kind or Minikube) and verifying pod health.

**Acceptance Scenarios**:

1. **Given** a Kubernetes cluster, **When** I apply a deployment manifest using the image, **Then** the pod starts successfully and passes readiness probes.
2. **Given** the pod is running, **When** I query the logs, **Then** I see region-prefixed output (e.g., `[us-east-1] PORT=...`) for all 12 regions.
3. **Given** the pod is running, **When** I send an HTTP request to the web endpoint on port 8001 (us-east-1), **Then** I receive a valid response.

---

### User Story 2 - Run Locally for Testing (Priority: P2)

As a developer, I want to run the plugin locally using Docker so that I can verify behavior without installing Go or compiling binaries manually.

**Why this priority**: simplifies developer onboarding and local testing.

**Independent Test**: Run `docker run -p 8001:8001 ...` and verify connectivity.

**Acceptance Scenarios**:

1. **Given** Docker is installed, **When** I run `docker run ghcr.io/rshade/finfocus-plugin-aws-public:latest`, **Then** the container starts and logs startup messages.
2. **Given** the container is running, **When** I send a SIGTERM (Ctrl+C), **Then** the container shuts down gracefully within 5 seconds.

---

## Clarifications

### Session 2026-01-14
- Q: How should logs be handled to differentiate between regions? → A: Inject region field into structured JSON logs (e.g., `{"region":"us-east-1", ...}`) or prefix non-JSON lines.
- Q: Will the Docker build download a single combined tarball or 12 separate archives? → A: Update spec to download 12 separate tarballs (one per region).
- Q: Should environment variables be applied globally or individually to the 12 binaries? → A: Global: Environment variables passed to the container apply to all 12 binaries.
- Q: What determines container health with 12 binaries running? → A: All 12 regional endpoints must respond successfully to a health check.
- Q: When should the GitHub Action build and push the Docker image? → A: Trigger on `release` creation AND `workflow_dispatch` (manual).
- Q: How should authentication and authorization be handled for the 12 regional gRPC endpoints running in the container? → A: Same as standalone binaries.
- Q: What should happen when individual regional binaries fail during container operation? → A: Retry then stop.
- Q: What are the memory and CPU resource limits for the container running all 12 binaries? → A: No limits specified.
- Q: How should the build handle missing GitHub releases for specific regions during Docker build? → A: Fail entire build.
- Q: What level of observability should be implemented for monitoring all 12 regional binaries? → A: Both logs and metrics.

## Requirements

### Functional Requirements

- **FR-001**: The Docker image MUST be based on `alpine:3.19`.
- **FR-002**: The build process MUST accept a `VERSION` build argument (e.g., `v0.1.0`).
- **FR-003**: The build process MUST download 12 separate release tarballs (one per region) from GitHub Releases matching the target architecture.
- **FR-004**: The image MUST contain executable binaries for all 12 supported AWS regions (extracted from the regional tarballs).
- **FR-005**: The image MUST include an entrypoint script that starts all 12 regional binaries in the background.
- **FR-006**: The entrypoint script MUST trap SIGTERM signals and forward them to all child processes to ensure graceful shutdown.
- **FR-007**: The image MUST run as a non-root user named `plugin` with UID 65532.
- **FR-008**: The image MUST default the environment variable `FINFOCUS_PLUGIN_WEB_ENABLED` to `true`.
- **FR-009**: The image MUST default the environment variable `FINFOCUS_PLUGIN_HEALTH_ENDPOINT` to `true`.
- **FR-010**: The solution MUST include a GitHub Action workflow (`docker-publish.yml`) that builds and pushes the image to GitHub Container Registry (GHCR) on release creation and via manual trigger (`workflow_dispatch`).
- **FR-011**: The GitHub Action MUST tag the image with `latest` and the release version (e.g., `v1.2.3`).
- **FR-012**: The image MUST expose the following ports:
    - 8001 (us-east-1)
    - 8002 (us-west-2)
    - 8003 (eu-west-1)
    - 8004 (ap-southeast-1)
    - 8005 (ap-southeast-2)
    - 8006 (ap-northeast-1)
    - 8007 (ap-south-1)
    - 8008 (ca-central-1)
    - 8009 (sa-east-1)
    - 8010 (us-gov-west-1)
    - 8011 (us-gov-east-1)
    - 8012 (us-west-1)
- **FR-013**: The entrypoint script MUST inject a `region` field into the structured JSON output of each binary (e.g., `{"region":"us-east-1", ...}`) to differentiate logs while maintaining JSON validity.
- **FR-014**: The entrypoint script MUST propagate all relevant environment variables (like `FINFOCUS_LOG_LEVEL`) to all 12 regional binaries.
- **FR-015**: The health check script MUST verify that all 12 regional HTTP endpoints are responding successfully before reporting the container as healthy.
- **FR-016**: All 12 regional binaries MUST run their HTTP endpoints (not gRPC) for web UI and health check accessibility.
- **FR-017**: When a regional binary fails, the entrypoint script MUST retry starting it up to 3 times; if still failing after 3 attempts, the entire container MUST shut down.
- **FR-018**: The Docker build process MUST fail immediately if any of the 12 required regional release tarballs are missing from GitHub Releases.
- **FR-019**: The image MUST expose a Prometheus metrics endpoint on port 9090 with per-region health status and request metrics.
- **FR-020**: All regional binaries MUST output structured JSON logs with region, timestamp, and level fields for aggregation and parsing.

### Key Entities

- **Docker Image**: The final artifact containing all binaries.
- **Entrypoint Script**: The bash script managing the subprocesses.

## Success Criteria

### Measurable Outcomes

- **SC-001**: The final Docker image size is approximately 2.0GB (+/- 10%) due to embedded pricing data for 12 regions.
- **SC-002**: All 12 regional endpoints are reachable within 10 seconds of container start.
- **SC-003**: The container shuts down cleanly (exit code 0) within 5 seconds of receiving SIGTERM.
- **SC-004**: The image passes Trivy security scans with no critical or high-severity vulnerabilities (medium and below are acceptable).