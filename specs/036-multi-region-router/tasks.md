# Tasks: Multi-Region Router Plugin

**Input**: Design documents from `/specs/036-multi-region-router/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/routing.md

**Tests**: Included — unit tests are part of each phase to validate correctness.

**Organization**: Tasks grouped by user story priority (P1 → P2 → P3).

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2)
- Include exact file paths in descriptions

## Phase 1: Setup

**Purpose**: Create project structure and entry point skeleton

- [x] T001 Create router package directory structure: `internal/router/` and `cmd/finfocus-plugin-aws-public-router/`
- [x] T002 Create router entry point skeleton in `cmd/finfocus-plugin-aws-public-router/main.go` — parse env vars (FINFOCUS_LOG_LEVEL, FINFOCUS_PLUGIN_OFFLINE, PORT, FINFOCUS_PLUGIN_WEB_ENABLED), init zerolog logger via `pluginsdk.NewPluginLogger()`, placeholder router init, call `pluginsdk.Serve(ctx, config)` with `PluginInfo{Name: "finfocus-plugin-aws-public", Providers: []string{"aws"}}`
- [x] T003 [P] Create `scripts/build-router.sh` — build router binary with `go build -o dist/finfocus-plugin-aws-public ./cmd/finfocus-plugin-aws-public-router`, no region build tags needed

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure shared by ALL user stories — child process management, discovery, and router skeleton

**CRITICAL**: No user story work can begin until this phase is complete

- [x] T004 Implement ChildState enum (Idle, Launching, Ready, Unhealthy, Failed) and ChildProcess struct (region, binaryPath, cmd, port, client, state, mu, version) in `internal/router/child.go`
- [x] T005 Implement child launch function in `internal/router/child.go` — `exec.Command` with env `FINFOCUS_PLUGIN_WEB_ENABLED=true`, `PORT=0`, inherit `FINFOCUS_LOG_LEVEL`; capture stdout via `StdoutPipe()`; parse `PORT=(\d+)` regex within 30-second timeout; create `pluginsdk.NewConnectClient(fmt.Sprintf("http://localhost:%d", port))`; transition state Idle → Launching → Ready
- [x] T006 [P] Implement Discovery function in `internal/router/discovery.go` — `Discover(dir string) map[string]string` scans directory for files matching `finfocus-plugin-aws-public-{region}` pattern, returns map of region → absolute binary path; use `filepath.Glob` with the naming convention from `internal/pricing/regions.yaml`
- [x] T007 Implement ChildRegistry in `internal/router/registry.go` — struct with `sync.RWMutex` + `map[string]*ChildProcess`; `GetOrLaunch(region string) (*pluginsdk.Client, error)` with per-region mutex to prevent concurrent duplicate launches (singleflight-style); `Get(region)` for read-only lookup; `ShutdownAll(ctx)` sends SIGTERM to all children and waits
- [x] T008 Implement RouterPlugin struct in `internal/router/router.go` — fields: version, logger, registry, downloader, offline, binaryDir; constructor `NewRouterPlugin(version, logger, binaryDir, offline)` that runs Discovery and pre-populates registry with idle children; implement `Name() string` returning `"finfocus-plugin-aws-public"` and `GetPluginInfo()` returning router metadata (version, specVersion, providers, no region in metadata)
- [x] T009 [P] Implement region extraction helper `extractRegion(req)` and trace_id propagation helper `propagateTraceID(ctx, traceID)` in `internal/router/router.go` — extract region from `ResourceDescriptor.Region` for single-resource RPCs; extract trace_id via `pluginsdk.TraceIDFromContext(ctx)` or generate UUID; create outgoing context with trace_id header for Connect client calls

**Checkpoint**: Foundation ready — child process launch, registry, discovery, and router skeleton are operational

---

## Phase 3: User Story 1 — Default Install Just Works (Priority: P1) MVP

**Goal**: Router delegates single-resource RPCs to lazily-launched child processes and auto-downloads missing region binaries from GitHub Releases

**Independent Test**: Build router, send GetProjectedCost for EC2 t3.micro in us-west-2, verify non-zero cost returned

### Implementation for User Story 1

- [x] T010 [US1] Implement single-resource RPC delegation methods in `internal/router/router.go` — `Supports()`, `GetProjectedCost()`, `GetActualCost()`, `EstimateCost()`, `GetPricingSpec()`, `DryRun()`: each extracts region via `extractRegion()`, calls `registry.GetOrLaunch(region)`, delegates to child's `pluginsdk.Client` with propagated trace_id; return `InvalidArgument` with `ERROR_CODE_INVALID_RESOURCE` if region is empty
- [x] T011 [P] [US1] Implement stub RPCs in `internal/router/router.go` — `DismissRecommendation()` and `GetBudgets()` return `connect.CodeUnimplemented` (stateless plugin, no budget support)
- [x] T012 [US1] Implement Downloader struct in `internal/router/downloader.go` — fields: version, baseURL (`https://github.com/rshade/finfocus-plugin-aws-public/releases/download/v{version}/`), targetDir, checksums map, httpClient, mu; `fetchChecksums()` downloads and parses `checksums.txt` (cache for session); `verify(filepath, expectedHash) error` computes SHA256 and compares
- [x] T013 [US1] Implement `Download(region, goos, goarch) (string, error)` in `internal/router/downloader.go` — construct tarball name `finfocus-plugin-aws-public_{version}_{OS}_{arch}_{region}.tar.gz`; HTTP GET tarball; verify SHA256 against cached checksums; extract binary from tar.gz; set `os.Chmod(path, 0755)`; return path to extracted binary
- [x] T014 [US1] Wire Downloader into ChildRegistry.GetOrLaunch fallback in `internal/router/registry.go` — when region has no local binary and offline==false, call `downloader.Download(region, runtime.GOOS, runtime.GOARCH)`; on success, create ChildProcess with downloaded binary path and launch; on failure, return error with actionable message
- [x] T015 [US1] Wire complete RouterPlugin into entry point in `cmd/finfocus-plugin-aws-public-router/main.go` — initialize Downloader (if not offline), create RouterPlugin with all dependencies, pass to `pluginsdk.Serve()` via ServeConfig; handle version from ldflags `-X main.version`
- [x] T016 [US1] Add unit tests for router delegation logic in `internal/router/router_test.go` — test extractRegion() with valid/empty/missing region; test Name() returns correct name; test GetPluginInfo() returns router metadata; test delegation returns InvalidArgument for empty region; table-driven tests for each RPC method
- [x] T017 [P] [US1] Add unit tests for Downloader in `internal/router/downloader_test.go` — test URL construction for various OS/arch/region combos; test SHA256 verification (valid hash, mismatch, missing entry); test checksums.txt parsing; test Download with httptest mock server returning tarball and checksums

**Checkpoint**: Router binary delegates all RPCs and auto-downloads region binaries — User Story 1 is fully functional

---

## Phase 4: User Story 5 — Release Pipeline Produces Router Binary (Priority: P1)

**Goal**: GoReleaser builds and releases the router archive alongside region binaries with correct naming (`finfocus-plugin-aws-public_{version}_{OS}_{arch}.tar.gz`, no region suffix)

**Independent Test**: Run `make generate-goreleaser && goreleaser check` — verify router build config is present and valid

### Implementation for User Story 5

- [x] T018 [US5] Add router build target to GoReleaser template in `tools/generate-goreleaser/main.go` — add a non-templated `router` build block after the region loop: main `./cmd/finfocus-plugin-aws-public-router`, binary name `finfocus-plugin-aws-public`, no build tags, ldflags `-s -w -X main.version={{.Version}}`, targets: linux/darwin/windows × amd64/arm64
- [x] T019 [US5] Add separate archive rule for router in `tools/generate-goreleaser/main.go` — router archives use `finfocus-plugin-aws-public_{{.Version}}_{{title .Os}}_{{.Arch}}` naming (no region suffix); exclude router build from region archive rules; region archives continue with `_{region}` suffix
- [x] T020 [US5] Regenerate `.goreleaser.yaml` by running `go run ./tools/generate-goreleaser` and verify output includes router build and archive sections
- [x] T021 [US5] Update release workflow in `.github/workflows/release.yml` — add router build step after region builds but before checksum generation; router has no pricing data generation step; include router archives in `checksums.txt`
- [x] T022 [P] [US5] Update binary verification script `scripts/verify-release-binaries.sh` — add check for router archive existence (no region suffix); verify router binary size < 15MB (no embedded pricing); skip the 100MB minimum size check for router binary

**Checkpoint**: Release pipeline produces router + region binaries with correct naming and checksums

---

## Phase 5: User Story 2 — Air-Gapped / Offline Operation (Priority: P2)

**Goal**: Router works with pre-installed region binaries only when `FINFOCUS_PLUGIN_OFFLINE=true`, with clear error messages for unavailable regions

**Independent Test**: Set `FINFOCUS_PLUGIN_OFFLINE=true`, place us-east-1 binary alongside router, verify routing works; request missing region and verify error message includes install command

### Implementation for User Story 2

- [x] T023 [US2] Implement offline error response in `internal/router/registry.go` — when `GetOrLaunch()` finds no local binary and offline==true, return gRPC error with `ERROR_CODE_UNSUPPORTED_REGION` and message: `"Region {region} not available. Install with: finfocus plugin install aws-public --metadata=region={region}"`
- [x] T024 [US2] Add unit tests for offline mode in `internal/router/router_test.go` — test that offline=true prevents download attempts; test error message format includes region name and install command; test that pre-discovered binaries still launch in offline mode
- [x] T025 [P] [US2] Add unit tests for Discovery in `internal/router/discovery_test.go` — test Discover() finds binaries matching naming pattern; test with no matching files returns empty map; test with mixed valid/invalid filenames; test with multiple regions; use `t.TempDir()` for isolation

**Checkpoint**: Air-gapped users can operate with pre-installed binaries and receive clear install instructions for missing regions

---

## Phase 6: User Story 3 — Graceful Child Lifecycle Management (Priority: P2)

**Goal**: Router handles SIGTERM/SIGINT shutdown, detects child crashes, and restarts unhealthy children with 3-retry limit

**Independent Test**: Start router with a child, kill the child process externally, send another request, verify child restarts successfully

### Implementation for User Story 3

- [x] T026 [US3] Implement graceful shutdown in `internal/router/router.go` — register signal handler for SIGTERM/SIGINT; on signal, call `registry.ShutdownAll(ctx)` which sends SIGTERM to all children, waits up to 30 seconds, then SIGKILL remaining; log each child termination via zerolog
- [x] T027 [US3] Implement child health check in `internal/router/child.go` — `HealthCheck() error` sends `Name()` RPC to child; if error, mark state as Unhealthy; called before delegation when child state is Ready but process may have exited (check `cmd.Process` state)
- [x] T028 [US3] Implement restart with retry limit in `internal/router/registry.go` — when `GetOrLaunch()` finds unhealthy child, attempt restart (kill old process, re-launch); track retry count per request; after 3 failures, mark child Failed and return error: `"Child for region {region} failed to start after 3 attempts"`; reset retry count on successful restart
- [x] T029 [US3] Implement concurrent launch prevention in `internal/router/registry.go` — per-region mutex ensures only one goroutine launches a child at a time; other goroutines wait for launch to complete and reuse the result; test with `sync.WaitGroup` in unit test
- [x] T030 [US3] Add unit tests for lifecycle management in `internal/router/child_test.go` — test health check detects exited process; test state transitions (Ready → Unhealthy → Ready after restart); test state transitions (Unhealthy → Failed after 3 retries)
- [x] T031 [P] [US3] Add unit tests for concurrent launch prevention in `internal/router/registry_test.go` — launch 10 goroutines requesting same region simultaneously; verify only 1 child process is created; verify all goroutines receive the same client

**Checkpoint**: Router handles shutdown signals, crash recovery, and retry limits without leaking processes

---

## Phase 7: User Story 4 — Multi-Region Recommendations Fan-Out (Priority: P3)

**Goal**: `GetRecommendations` groups resources by region, delegates to children in parallel, and merges results with partial failure tolerance

**Independent Test**: Send GetRecommendations with resources from 2 regions, verify merged results from both

### Implementation for User Story 4

- [x] T032 [US4] Implement GetRecommendations fan-out in `internal/router/router.go` — group `target_resources` by `resource.Region`; for each region group, launch goroutine that calls `registry.GetOrLaunch(region)` and delegates `GetRecommendations` with filtered resources; use `sync.WaitGroup` + `sync.Mutex` to collect results; merge all recommendation responses
- [x] T033 [US4] Implement partial failure handling in `internal/router/router.go` — if a region child fails during fan-out, log <warning> with zerolog (region, error); continue collecting from successful children; return merged successful results; if ALL children fail, return error
- [x] T034 [US4] Add unit tests for fan-out logic in `internal/router/router_test.go` — test grouping resources by region; test parallel delegation to 2 regions returns merged results; test partial failure (1 region fails, 1 succeeds) returns partial results with warning; test all-fail returns error; test single-region falls through to standard delegation

**Checkpoint**: All 5 user stories are independently functional and testable

---

## Phase 8: Polish and Cross-Cutting Concerns

**Purpose**: Validation, linting, backward compatibility, and documentation

- [x] T035 Run `make lint` across all new files and fix any golangci-lint issues
- [x] T036 Run `make test` and verify all existing tests still pass (backward compatibility SC-005)
- [x] T037 Verify router binary size < 15MB by building with `go build -o /tmp/router ./cmd/finfocus-plugin-aws-public-router && ls -lh /tmp/router` (SC-003)
- [x] T038 [P] Update ROADMAP.md — move "Multi-Region Router" from Planned to Done section; add single-sentence summary of what was delivered
- [x] T039 [P] Update CLAUDE.md — add router-specific conventions: entry point path, build command, env vars, test commands
- [x] T040 Run `npx markdownlint-cli2` on any new or modified markdown files
- [x] T041 [P] Add integration test for end-to-end router-child delegation in `internal/router/router_integration_test.go` — build a region binary with `go build -tags=region_use1`, start router, send `GetProjectedCost` for EC2 t3.micro in us-east-1 via Connect client, verify non-zero cost returned; use `//go:build integration` tag; validates SC-006
- [x] T042 [P] Add benchmark test for delegation latency in `internal/router/router_bench_test.go` — `BenchmarkDelegationOverhead` measures RPC round-trip through router vs direct child call; assert overhead < 100ms per RPC; validates SC-002
