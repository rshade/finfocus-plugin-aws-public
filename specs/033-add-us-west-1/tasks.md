# Tasks: Add us-west-1 (N. California) Region Support

**Feature Branch**: `033-add-us-west-1`
**Feature Spec**: `specs/033-add-us-west-1/spec.md`

## Implementation Strategy

- **MVP Scope**: User Story 1 (Cost Estimation) and User Story 2 (Deployment). This provides the core functionality and the means to distribute it.
- **Incremental Delivery**:
    1.  Configuration & Data Generation (Foundational)
    2.  Binary Compilation & Embedding (US1)
    3.  Docker Distribution (US2)
    4.  Carbon Verification (US3)
- **Testing**:
    - Unit tests for the new region embedding.
    - Build verification to ensure the binary compiles with the new tag.
    - Docker health checks to verify runtime stability.

## Dependencies

- **User Story 1** (Cost Estimation)
  - Depends on: Phase 1 (Config) & Phase 2 (Data Generation)
- **User Story 2** (Deployment)
  - Depends on: User Story 1 (Binary availability)
- **User Story 3** (Carbon)
  - Independent, but logically grouped with regional support.

## Phase 1: Setup

**Goal**: Configure the project to recognize the new region.

- [X] T001 Update regions configuration in `internal/pricing/regions.yaml`
- [X] T002 Update GoReleaser config in `.goreleaser.yaml`

## Phase 2: Foundational

**Goal**: Generate the necessary pricing data for embedding. This is blocking for binary compilation.

- [X] T003 Run pricing generator for us-west-1: `make generate-pricing REGION=us-west-1` or `go run ./tools/generate-pricing -region us-west-1`

## Phase 3: User Story 1 - Estimate Costs in us-west-1 (P1)

**Goal**: Enable the plugin to provide cost estimates for resources in `us-west-1`.

**Independent Test**: Build the `us-west-1` binary and verify it returns valid costs for a sample resource.

- [X] T004 [US1] Create embedding code in `internal/pricing/embed_usw1.go`
- [X] T005 [US1] Create unit test for data loading in `internal/pricing/embed_usw1_test.go`
- [X] T006 [US1] Verify build succeeds with `go build -tags region_usw1 ./cmd/finfocus-plugin-aws-public`
- [X] T006a [US1] (FR-001) Add integration test verifying `Supports()` returns true for us-west-1 resources in `internal/plugin/supports_test.go`
- [X] T006b [US1] (FR-007) Add test verifying `ERROR_CODE_INVALID_RESOURCE` is returned for unsupported resource types in us-west-1

## Phase 4: User Story 2 - Deploy Plugin with us-west-1 Support (P1)

**Goal**: Update Docker distribution to include the new regional binary.

**Independent Test**: Build the Docker image and verify the `us-west-1` service starts and passes health checks.

- [X] T007 [US2] Update Dockerfile to include us-west-1 region, EXPOSE 8010, and REGIONS list in `docker/Dockerfile`
- [X] T008 [US2] Update entrypoint script to launch us-west-1 binary in `docker/entrypoint.sh`
- [X] T009 [US2] Update healthcheck script to verify port 8010 in `docker/healthcheck.sh`

## Phase 5: User Story 3 - Carbon Estimation for us-west-1 (P2)

**Goal**: Ensure accurate carbon emission factors are used for N. California.

**Independent Test**: Verify `GetGridFactor("us-west-1")` returns the WECC value (0.000322).

- [X] T010 [US3] Verify and add explicit test case for us-west-1 grid factor in `internal/carbon/grid_factors_test.go`

## Phase 6: Polish

**Goal**: Update documentation and finalize the release.

- [X] T011 Update region count and list in `CLAUDE.md`
- [X] T012 Update supported regions list in `README.md`
