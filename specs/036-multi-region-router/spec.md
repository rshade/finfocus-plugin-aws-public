# Feature Specification: Multi-Region Router Plugin

**Feature Branch**: `036-multi-region-router`
**Created**: 2026-02-12
**Status**: Draft
**Input**: User description: "Build a shim router plugin that becomes the default install for aws-public, discovering or auto-downloading region-specific binaries and delegating gRPC requests to the correct region child process"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Default Install Just Works (Priority: P1)

A user installs the aws-public plugin without specifying a region. Today this
silently installs the us-east-1 binary, which returns $0 costs for resources in
any other region. With the router, the default install gets a thin routing binary
that handles any region automatically by downloading the needed region binary on
first request.

**Why this priority**: This is the core problem. If the default install doesn't
work for all regions, the entire plugin appears broken to users outside us-east-1.
The plugin author themselves hit this bug.

**Independent Test**: Install the router binary, send a cost request for a
resource in us-west-2, and verify a non-zero cost is returned.

**Acceptance Scenarios**:

1. **Given** the user runs `finfocus plugin install aws-public` with no metadata,
   **When** the core launches the installed binary,
   **Then** the router starts, announces PORT, and accepts connections.

2. **Given** the router is running and receives a `GetProjectedCost` request for
   an EC2 t3.micro in us-west-2,
   **When** no us-west-2 region binary exists locally,
   **Then** the router downloads the us-west-2 binary from the matching release,
   launches it, delegates the request, and returns a non-zero cost.

3. **Given** the router has already downloaded and launched the us-west-2 child,
   **When** a second request arrives for us-west-2,
   **Then** the router reuses the existing child connection without re-download
   or re-launch.

---

### User Story 2 - Air-Gapped / Offline Operation (Priority: P2)

Users in restricted environments (government, financial, air-gapped networks)
cannot download binaries at runtime. They need the router to work with
pre-installed region binaries only, with clear error messages when a region is
unavailable.

**Why this priority**: Air-gapped operation is a strategic guardrail of the
plugin. Runtime downloads must be optional, not required.

**Independent Test**: Set `FINFOCUS_PLUGIN_OFFLINE=true`, place a region binary
in the router's directory, and verify it routes correctly. Request a missing
region and verify a helpful error message.

**Acceptance Scenarios**:

1. **Given** `FINFOCUS_PLUGIN_OFFLINE=true` is set and the us-east-1 region
   binary exists alongside the router,
   **When** a request arrives for us-east-1,
   **Then** the router discovers the sibling binary, launches it, and delegates
   successfully.

2. **Given** `FINFOCUS_PLUGIN_OFFLINE=true` is set and no us-west-2 binary
   exists,
   **When** a request arrives for us-west-2,
   **Then** the router returns an error with the message:
   "Region us-west-2 not available. Install with: finfocus plugin install
   aws-public --metadata=region=us-west-2".

3. **Given** the user has pre-installed 3 region binaries alongside the router,
   **When** a `Supports` request arrives for each region,
   **Then** the router delegates to the correct child for each and reports
   support accurately.

---

### User Story 3 - Graceful Child Lifecycle Management (Priority: P2)

The router manages region-specific child processes. It must handle startup
failures, child crashes, and clean shutdown without leaking processes.

**Why this priority**: Process management bugs cause resource leaks, zombie
processes, and port exhaustion. This must be solid for production use.

**Independent Test**: Start the router, trigger a child launch, kill the child
process externally, send another request, and verify the child is restarted.

**Acceptance Scenarios**:

1. **Given** the router is running with one child process for us-east-1,
   **When** the router receives a shutdown signal (SIGTERM),
   **Then** the router terminates all children and exits cleanly with no
   orphaned processes.

2. **Given** a child process crashes during operation,
   **When** the next request arrives for that region,
   **Then** the router detects the unhealthy child, restarts it, and completes
   the request.

3. **Given** a child binary fails to start (corrupt binary, port conflict),
   **When** the router attempts to launch it,
   **Then** the router returns a clear error to the caller and does not retry
   indefinitely.

---

### User Story 4 - Multi-Region Recommendations Fan-Out (Priority: P3)

The `GetRecommendations` request accepts a batch of resources from potentially
multiple regions. The router must fan out to the correct children per region and
merge results.

**Why this priority**: Recommendations are a secondary feature. The core cost
estimation (P1) must work first.

**Independent Test**: Send a recommendations request with resources from 2
different regions, verify results from both are merged in the response.

**Acceptance Scenarios**:

1. **Given** the router has children for us-east-1 and us-west-2,
   **When** a `GetRecommendations` request contains resources from both regions,
   **Then** the router sends separate requests to each child and merges the
   recommendations into a single response.

2. **Given** one region child fails during fan-out,
   **When** the other region succeeds,
   **Then** the router returns the successful recommendations and logs a warning
   for the failed region.

---

### User Story 5 - Release Pipeline Produces Router Binary (Priority: P1)

The router binary must be built and released alongside region binaries so that
the default install works from the first release containing the router.

**Why this priority**: Without the release pipeline, the router cannot be
installed by users. This is a prerequisite for User Story 1.

**Independent Test**: Run the release build, verify the router archive exists
in the dist/ directory with the correct naming pattern.

**Acceptance Scenarios**:

1. **Given** the release pipeline includes the router build configuration,
   **When** a release is triggered,
   **Then** the router archive is produced with the naming pattern
   `finfocus-plugin-aws-public_{version}_{OS}_{arch}.tar.gz` (no region suffix).

2. **Given** the registry in finfocus-core has `default_region: ""`,
   **When** a user runs `finfocus plugin install aws-public`,
   **Then** the core downloads the router archive (clean name, no region suffix).

---

### Edge Cases

- What happens when the router binary is the only file installed (no region
  binaries, no network)? The router starts but all RPCs return informative errors.
- What happens when a request has no region field? The router returns
  `InvalidArgument` for cost RPCs.
- What happens when two concurrent requests for the same not-yet-launched region
  arrive simultaneously? Only one child is started (concurrent launch prevention).
- What happens when the release for the router's version has no region binary for
  a requested region? The download fails gracefully with an install command
  suggestion.
- What happens when the child's PORT announcement takes longer than 30 seconds?
  The router times out and returns an error.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The router MUST implement all 11 cost source service RPCs
  (Name, GetPluginInfo, Supports, GetProjectedCost, GetActualCost,
  GetPricingSpec, EstimateCost, GetRecommendations, DismissRecommendation,
  GetBudgets, DryRun).
- **FR-002**: The router MUST discover region-specific binaries in its own
  directory by scanning for files named `finfocus-plugin-aws-public-{region}`.
- **FR-003**: The router MUST lazily launch child processes only when the first
  request for a given region arrives (not at startup).
- **FR-004**: The router MUST communicate with children using the Connect
  protocol via the existing SDK client.
- **FR-005**: The router MUST reuse child connections for subsequent requests
  to the same region (no re-launch per request).
- **FR-006**: The router MUST support auto-downloading region binaries from
  the matching release when a region binary is not found locally (connected mode).
  Downloaded binaries MUST be verified via SHA256 checksum against the release's
  `checksums.txt` before execution.
- **FR-007**: The router MUST support an offline mode
  (`FINFOCUS_PLUGIN_OFFLINE=true`) that disables all network downloads.
- **FR-008**: The router MUST gracefully shut down all child processes when
  it receives SIGTERM or SIGINT.
- **FR-009**: The router MUST restart unhealthy child processes on the next
  request for that region, with a maximum of 3 startup retries per request
  before returning an error with an actionable message.
- **FR-010**: The router MUST return clear, actionable error messages when a
  region binary is unavailable, including the install command to fix it.
- **FR-011**: The router MUST fan out `GetRecommendations` requests by grouping
  resources by region and delegating to the correct children in parallel.
- **FR-012**: The router binary MUST be produced by the release pipeline with
  the archive name `finfocus-plugin-aws-public_{version}_{OS}_{arch}.tar.gz`
  (no region suffix).
- **FR-013**: The finfocus-core registry MUST be updated to set
  `default_region` to empty string so the router is installed by default.
- **FR-014**: The router MUST announce `PORT=<port>` to stdout on startup,
  following the standard plugin protocol.
- **FR-015**: The router MUST capture each child's `PORT=<port>` announcement
  from stdout within a 30-second timeout.
- **FR-016**: The router MUST handle concurrent requests for the same
  not-yet-launched region without starting duplicate child processes.
- **FR-017**: The router MUST propagate trace_id from incoming gRPC metadata
  to child process requests for end-to-end distributed tracing.
- **FR-018**: The router MUST use zerolog (stderr) for structured logging of
  routing events: child launch, binary download, request delegation, and errors.

### Key Entities

- **Router Plugin**: The thin routing binary installed as
  `finfocus-plugin-aws-public`. Contains no pricing data. Manages child
  lifecycle and delegates RPCs.
- **Region Child**: A region-specific binary
  (`finfocus-plugin-aws-public-{region}`) running as a child process with
  embedded pricing data for one AWS region.
- **Region Binary Discovery**: The mechanism by which the router finds region
  binaries on disk (sibling directory scan).
- **Region Binary Downloader**: The component that fetches region binaries
  from the matching release when not found locally (connected mode only).

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A user running `finfocus plugin install aws-public` with no
  metadata can estimate costs for resources in any supported region without
  additional manual steps (connected mode).
- **SC-002**: The router adds less than 100ms of overhead per RPC compared to
  directly calling the region binary (delegation latency).
- **SC-003**: The router binary is under 15MB (no embedded pricing data),
  compared to approximately 150MB per region binary.
- **SC-004**: Child processes start within 30 seconds of first request for a
  region (including download time on a standard connection).
- **SC-005**: All existing unit tests and integration tests continue to pass
  with no modifications (backward compatibility).
- **SC-006**: The router correctly handles all 11 cost source service RPCs
  (6 delegated, 1 fan-out, 4 local).
- **SC-007**: Air-gapped users can operate the router with pre-installed
  region binaries and receive clear error messages for unavailable regions.
- **SC-008**: The release pipeline produces the router binary alongside all
  region binaries with correct archive naming.

## Assumptions

- Region binaries are backward-compatible with the router version (same
  major.minor version). The router downloads binaries matching its own version.
  Pre-installed binaries with a different version trigger a warning log but are
  not blocked from launching.
- The Connect protocol is available on all child processes (the router sets
  `FINFOCUS_PLUGIN_WEB_ENABLED=true` for children).
- Releases are accessible for auto-download in connected mode. The URL pattern
  is hardcoded to GitHub Releases:
  `https://github.com/rshade/finfocus-plugin-aws-public/releases/download/v{version}/{tarball}`
  (same pattern as the Dockerfile).
- The router and region binaries run on the same machine (127.0.0.1 loopback).
- Maximum 12 concurrent child processes (one per supported AWS region).

## Scope Boundaries

**In scope:**

- Router binary implementation (new entry point)
- Child process management (discovery, launch, health check, restart, shutdown)
- Auto-download from releases (connected mode)
- Offline mode with pre-installed binaries
- Release pipeline changes (GoReleaser, CI workflows, scripts)
- Registry change in finfocus-core

**Out of scope:**

- Docker image changes (current entrypoint.sh pattern continues to work)
- Changes to existing region binary behavior or pricing logic
- New AWS region support (regions.yaml changes)
- Plugin metadata install feature (being built separately as the "belt")

## Clarifications

### Session 2026-02-12

- Q: Should the router verify integrity of downloaded region binaries? → A: SHA256 checksum verification against the release's `checksums.txt` file (same pattern as Docker build)
- Q: What observability should the router provide? → A: Propagate trace_id to children via gRPC metadata + zerolog for router events (child launch, download, delegate, errors)
- Q: What URL pattern for auto-downloading region binaries? → A: Hardcoded GitHub Releases URL pattern (same as Dockerfile): `https://github.com/rshade/finfocus-plugin-aws-public/releases/download/v{version}/{tarball}`
- Q: What retry policy for child startup failures? → A: 3 retries per request, then return error with actionable message
- Q: What happens on router/child version mismatch? → A: Log a warning but proceed — don't block on version mismatch
