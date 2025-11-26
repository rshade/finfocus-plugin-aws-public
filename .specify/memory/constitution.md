<!--
Sync Impact Report - Constitution v2.1.0
========================================
Version Change: 2.0.0 → 2.1.0
Rationale: MINOR update to document GetActualCost fallback implementation.
           Feature 004-actual-cost-fallback implements pro-rated cost estimation
           using projected monthly cost × (runtime_hours / 730).

Modified Principles:
  - III. Protocol & Interface Consistency: Updated GetActualCost method description

Previous Sync Impact Report (v2.0.0):
  - MAJOR update to align with gRPC protocol instead of stdin/stdout JSON
  - Complete rewrite for gRPC in Protocol section
  - Updated RPC latency targets in Performance section
  - Added Thread Safety requirements
  - Added gRPC Error Code enum compliance

Templates Requiring Updates:
  ✅ .specify/templates/plan-template.md - Constitution Check section verified
  ✅ .specify/templates/spec-template.md - Requirements alignment verified
  ✅ .specify/templates/tasks-template.md - Task categorization verified

Follow-up TODOs:
  - None
-->

# PulumiCost Plugin AWS Public Constitution

## Core Principles

### I. Code Quality & Simplicity

**MUST enforce:**

- Keep It Simple, Stupid (KISS): No premature abstraction or over-engineering
- Single Responsibility Principle: Each package, type, and function does ONE thing well
- Explicit is better than implicit: No magic, hidden behavior, or surprising side effects
- Stateless components preferred: Each gRPC invocation is independent unless state is absolutely required

**Rationale:** This plugin is called as an external gRPC service by PulumiCost core. Complexity compounds debugging difficulty when troubleshooting RPC interactions. Simple, obvious code reduces maintenance burden and makes contribution easier.

**File size guidance:**

- Aim for focused, single-purpose files (typically <300 lines)
- Prefer logical separation over arbitrary line limits
- Large files are acceptable when they serve a single, cohesive purpose (e.g., comprehensive test suites, well-structured service implementations)

### II. Testing Discipline

**MUST enforce:**

- Unit tests for pure transformation functions and stateless logic (pricing lookups, cost calculations)
- Integration tests for gRPC service methods (can use in-memory mock pricing clients)
- No mocking of dependencies you don't own (e.g., proto messages, pluginsdk)
- Test quality indicators:
  - Each test has a distinct, clear purpose
  - Table-driven tests for variations on the same behavior
  - Simple setup, clear assertions
  - Fast execution (< 1s for unit suite, < 5s for integration suite)
- Tests MUST run via `make test` and pass before any commit
- Test coverage goal: Focus on critical paths (pricing lookups, cost calculations, gRPC handlers) rather than arbitrary percentage targets

**What NOT to test:**

- Proto message serialization (trust the proto compiler)
- pluginsdk.Serve() lifecycle (trust the SDK)
- Over-engineered mocking infrastructure (no `unsafe.Pointer` conversions, no complex helper functions wrapping struct literals)

**Rationale:** Testing validates correctness of cost estimation logic, which is the core value proposition. Poor tests (redundant, over-complicated, or "AI slop") waste time and create false confidence. Good tests enable safe refactoring and catch regressions early.

### III. Protocol & Interface Consistency

**MUST enforce:**

- **gRPC CostSourceService protocol is sacred:**
  - NEVER log to stdout except PORT announcement
  - All diagnostic logs go to stderr with `[pulumicost-plugin-aws-public]` prefix
  - Use `pluginsdk.Serve()` for lifecycle management
- **PORT announcement:** Plugin MUST write `PORT=<port>` to stdout exactly once, then serve gRPC on 127.0.0.1
- **Proto-defined types only:**
  - Use `ResourceDescriptor`, `GetProjectedCostResponse`, `SupportsResponse` from pulumicost.v1
  - NO custom JSON types or envelopes
- **Error codes MUST use proto ErrorCode enum:**
  - `ERROR_CODE_INVALID_RESOURCE` (6): Missing required ResourceDescriptor fields
  - `ERROR_CODE_UNSUPPORTED_REGION` (9): Region mismatch (return via gRPC status with details)
  - `ERROR_CODE_DATA_CORRUPTION` (11): Embedded pricing data load failed
  - NO custom error codes outside the proto enum
- **Thread safety:** All gRPC method handlers MUST be thread-safe (concurrent calls expected)
- **Region-specific binaries MUST embed only their region's pricing data**
- **Build tags MUST ensure exactly one embed file is selected at build time**

**gRPC Method Requirements:**

- `Name()` → returns `NameResponse{name: "aws-public"}`
- `Supports(ResourceDescriptor)` → checks region and resource_type, returns `SupportsResponse{supported, reason}`
- `GetProjectedCost(ResourceDescriptor)` → returns `GetProjectedCostResponse{unit_price, currency, cost_per_month, billing_detail}`
- `GetActualCost(GetActualCostRequest)` → returns `GetActualCostResponse`
  with fallback estimate (`projected_monthly_cost × runtime_hours / 730`)
- `GetPricingSpec()` → optional, may return detailed pricing info in future

**Rationale:** PulumiCost core depends on predictable gRPC protocol behavior. Breaking protocol compatibility breaks the integration. Using proto-defined types ensures compatibility across all PulumiCost plugins. Thread safety is critical because gRPC handles concurrent requests.

### IV. Performance & Reliability

**MUST enforce:**

- **Embedded pricing data:**
  - MUST be parsed once using `sync.Once` and cached
  - Pricing lookups MUST use indexed data structures (maps, not linear scans)
  - MUST be thread-safe for concurrent gRPC calls
- **Latency targets:**
  - Plugin startup time: < 500ms (includes pricing data parse)
  - PORT announcement: < 1 second after start
  - GetProjectedCost() RPC: < 100ms per call
  - Supports() RPC: < 10ms per call
- **Resource limits:**
  - Memory footprint: < 50MB per region binary (including embedded pricing data)
  - Binary size: < 10MB per region binary (before compression)
  - Concurrent RPC calls: Support at least 100 concurrent GetProjectedCost() calls

**Performance monitoring:**

- Log stderr warnings if pricing lookup takes > 50ms
- Use structured logging for RPC timing if observability is added

**Rationale:** The plugin may handle hundreds of concurrent RPC calls during a Pulumi stack analysis. Slow startup or inefficient lookups create poor user experience. Embedded data + indexing + thread-safe access ensures predictable performance without external dependencies.

### V. Build & Release Quality

**MUST enforce:**

- All code MUST pass `make lint` before commit (golangci-lint with project config)
- All tests MUST pass `make test` before commit
- GoReleaser builds MUST succeed for all supported regions (us-east-1, us-west-2, eu-west-1)
- Region-specific build tags MUST compile cleanly:
  - `region_use1` → us-east-1
  - `region_usw2` → us-west-2
  - `region_euw1` → eu-west-1
- Before hooks MUST generate pricing data (`tools/generate-pricing`) successfully
- Binaries MUST be named `pulumicost-plugin-aws-public-<region>`
- **gRPC service MUST be functional:** Manual testing with grpcurl before release

**Rationale:** Consistent build quality prevents regressions and ensures that PulumiCost core can reliably fetch and execute region-specific binaries. Linting catches common Go mistakes; tests validate correctness; GoReleaser ensures reproducible releases. gRPC functionality testing catches integration issues.

## Security Requirements

**MUST enforce:**

- No credentials or secrets in logs, error messages, or gRPC responses
- Pricing data fetching (future real AWS API integration) MUST use read-only IAM permissions
- No network calls at runtime (all pricing data embedded at build time for v1)
- Input validation: Reject malformed ResourceDescriptor gracefully (return gRPC InvalidArgument error)
- Dependency scanning: Use `govulncheck` in CI to detect known vulnerabilities
- **gRPC security:** Serve on loopback only (127.0.0.1), no TLS required for local communication

**Rationale:** The plugin processes user infrastructure definitions via gRPC and outputs cost data. Leaking credentials or allowing arbitrary code execution via malformed input is unacceptable. Embedded pricing data eliminates runtime AWS API dependency and reduces attack surface. Loopback-only serving prevents unauthorized network access.

## Development Workflow

**MUST enforce:**

- Feature branches named `###-feature-name` (where ### is issue/feature number)
- Commits MUST follow conventional commit format (verified via commitlint):
  - `feat:`, `fix:`, `docs:`, `chore:`, `test:`, `refactor:`
  - No "🤖 Generated with [Claude Code]" or "Co-Authored-By: Claude" in commit messages
- Pull requests MUST:
  - Reference related issue/feature number
  - Include updated tests if logic changes
  - Pass all CI checks (lint, test, build)
  - Update CLAUDE.md if new conventions or patterns emerge
- Markdown files MUST be linted with markdownlint after editing
- **gRPC changes:** Update proto definitions in pulumicost-spec if protocol changes needed

**Code review requirements:**

- At least one approval before merge
- Verify constitution compliance (simplicity, testing, gRPC protocol adherence)
- Check for "AI slop": redundant tests, unused fields, over-complicated helpers
- **Protocol compatibility:** Verify no breaking changes to gRPC interface

**Rationale:** Consistent workflow reduces friction in collaboration and code review. Conventional commits enable automated changelog generation. Constitution compliance checks ensure long-term maintainability. gRPC protocol compatibility is critical for integration with PulumiCost core.

## Governance

**Amendment procedure:**

1. Propose amendment via GitHub issue or PR with rationale
2. Document impact on existing code and templates
3. Update version per semantic versioning:
   - MAJOR: Backward incompatible principle removals or redefinitions (like 1.0 → 2.0 for gRPC migration)
   - MINOR: New principle/section added or materially expanded guidance
   - PATCH: Clarifications, wording, typo fixes
4. Propagate changes to dependent templates (plan, spec, tasks)
5. Update this file with Sync Impact Report (HTML comment at top)

**Versioning policy:**

- Constitution version MUST increment with each substantive change
- Version MUST be documented in Sync Impact Report
- RATIFICATION_DATE is the original adoption date (does not change)
- LAST_AMENDED_DATE updates to today's date when amended

**Compliance review:**

- All PRs MUST verify compliance with constitution principles
- Use `.specify/templates/plan-template.md` Constitution Check section as gate
- Complexity violations MUST be justified in plan.md Complexity Tracking table
- Constitution supersedes ad-hoc practices; when in doubt, refer to this document

**Runtime development guidance:**

- Use CLAUDE.md for agent-specific guidance and project conventions
- Constitution defines non-negotiable rules; CLAUDE.md provides practical implementation details
- When CLAUDE.md conflicts with constitution, constitution wins

**Version**: 2.1.0 | **Ratified**: 2025-11-16 | **Last Amended**: 2025-11-25
