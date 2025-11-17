<!--
Sync Impact Report - Constitution v1.0.0
========================================
Version Change: INITIAL → 1.0.0
Rationale: Initial constitution creation with 5 core principles focused on code quality,
           testing standards, UX consistency, and performance requirements

Modified Principles: N/A (initial creation)
Added Sections:
  - I. Code Quality & Simplicity
  - II. Testing Discipline
  - III. Protocol & Interface Consistency
  - IV. Performance & Reliability
  - V. Build & Release Quality

Removed Sections: N/A (initial creation)

Templates Requiring Updates:
  ✅ .specify/templates/plan-template.md - Constitution Check section verified
  ✅ .specify/templates/spec-template.md - Requirements alignment verified
  ✅ .specify/templates/tasks-template.md - Task categorization verified

Follow-up TODOs: None
-->

# PulumiCost Plugin AWS Public Constitution

## Core Principles

### I. Code Quality & Simplicity

**MUST enforce:**

- Keep It Simple, Stupid (KISS): No premature abstraction or over-engineering
- Single Responsibility Principle: Each package, type, and function does ONE thing well
- Explicit is better than implicit: No magic, hidden behavior, or surprising side effects
- Stateless components preferred: Each invocation is independent unless state is absolutely required

**Rationale:** This plugin is called as an external binary by PulumiCost core. Complexity compounds debugging difficulty when troubleshooting stdin/stdout protocol interactions. Simple, obvious code reduces maintenance burden and makes contribution easier.

**File size guidance:**

- Aim for focused, single-purpose files (typically <300 lines)
- Prefer logical separation over arbitrary line limits
- Large files are acceptable when they serve a single, cohesive purpose (e.g., comprehensive test suites, well-structured service implementations)

### II. Testing Discipline

**MUST enforce:**

- Unit tests for pure transformation functions and stateless logic
- Integration tests in `examples/` for components requiring HTTP clients or external dependencies
- No mocking of dependencies you don't own (e.g., AWS SDK, HTTP clients)
- Test quality indicators:
  - Each test has a distinct, clear purpose
  - Table-driven tests for variations on the same behavior
  - Simple setup, clear assertions
  - Fast execution (< 1s for entire suite)
- Tests MUST run via `make test` and pass before any commit
- Test coverage goal: Focus on critical paths (pricing lookups, cost calculations, protocol handling) rather than arbitrary percentage targets

**What NOT to test:**

- CRUD operations requiring HTTP clients (test these as integration tests in `examples/`)
- Methods that primarily delegate to external services
- Over-engineered mocking infrastructure (no `unsafe.Pointer` conversions, no complex helper functions wrapping struct literals)

**Rationale:** Testing validates correctness of cost estimation logic, which is the core value proposition. Poor tests (redundant, over-complicated, or "AI slop") waste time and create false confidence. Good tests enable safe refactoring and catch regressions early.

### III. Protocol & Interface Consistency

**MUST enforce:**

- stdin/stdout JSON protocol is sacred: NEVER log to stdout (use stderr with `[pulumicost-plugin-aws-public]` prefix)
- PluginResponse envelope format is versioned and backward-compatible
- Error codes are semantically meaningful and documented:
  - `INVALID_INPUT`: stdin JSON parsing failed
  - `PRICING_INIT_FAILED`: embedded pricing data load failed
  - `UNSUPPORTED_REGION`: region mismatch (includes `meta.pluginRegion` and `meta.requiredRegion`)
  - `NOT_IMPLEMENTED`: placeholder for future functionality
- Exit code 0 for success (`status: "ok"`), non-zero for errors (`status: "error"`)
- Region-specific binaries MUST embed only their region's pricing data
- Build tags MUST ensure exactly one embed file is selected at build time

**Rationale:** PulumiCost core depends on predictable, machine-parseable protocol behavior. Breaking protocol compatibility breaks the integration. Consistency in error handling enables PulumiCost core to make intelligent decisions (e.g., fetching the correct region binary when `UNSUPPORTED_REGION` is returned).

### IV. Performance & Reliability

**MUST enforce:**

- Embedded pricing data MUST be parsed once using `sync.Once` and cached
- Pricing lookups MUST use indexed data structures (maps, not linear scans)
- Plugin startup time: < 100ms on modern hardware (M1 MacBook, typical Linux server)
- Memory footprint: < 50MB per region binary (including embedded pricing data)
- Cost estimation latency: < 10ms per resource (excluding I/O)
- Binary size: < 20MB per region binary (compressed with UPX if needed)

**Performance monitoring:**

- Log stderr warnings if pricing lookup takes > 5ms
- Include timing metadata in PluginResponse for observability

**Rationale:** The plugin may be invoked hundreds of times during a Pulumi stack analysis. Slow startup or inefficient lookups create poor user experience. Embedded data + indexing ensures predictable performance without external dependencies.

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

**Rationale:** Consistent build quality prevents regressions and ensures that PulumiCost core can reliably fetch and execute region-specific binaries. Linting catches common Go mistakes; tests validate correctness; GoReleaser ensures reproducible releases.

## Security Requirements

**MUST enforce:**

- No credentials or secrets in logs, error messages, or PluginResponse output
- Pricing data fetching (future real AWS API integration) MUST use read-only IAM permissions
- No network calls at runtime (all pricing data embedded at build time for v1)
- Input validation: Reject malformed JSON, unknown resource types gracefully (return warnings, not crashes)
- Dependency scanning: Use `govulncheck` in CI to detect known vulnerabilities

**Rationale:** The plugin processes user infrastructure definitions and outputs cost data. Leaking credentials or allowing arbitrary code execution via malformed input is unacceptable. Embedded pricing data eliminates runtime AWS API dependency and reduces attack surface.

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

**Code review requirements:**

- At least one approval before merge
- Verify constitution compliance (simplicity, testing, protocol adherence)
- Check for "AI slop": redundant tests, unused fields, over-complicated helpers

**Rationale:** Consistent workflow reduces friction in collaboration and code review. Conventional commits enable automated changelog generation. Constitution compliance checks ensure long-term maintainability.

## Governance

**Amendment procedure:**

1. Propose amendment via GitHub issue or PR with rationale
2. Document impact on existing code and templates
3. Update version per semantic versioning:
   - MAJOR: Backward incompatible principle removals or redefinitions
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

**Version**: 1.0.0 | **Ratified**: 2025-11-16 | **Last Amended**: 2025-11-16
