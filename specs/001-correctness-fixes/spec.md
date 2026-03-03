# Feature Specification: Correctness Fixes Batch

**Feature Branch**: `001-correctness-fixes`
**Created**: 2026-02-16
**Status**: Draft
**Input**: Batch fix for issues #294, #293, #289 (issue #292 deferred)

## Clarifications

### Session 2026-02-16

- Q: Confidence signal delivery mechanism (Option A/B/C from #292)? → A: Option A intent (metadata map), adapted to structured BillingDetail suffix due to proto lacking metadata map field. See research.md R1.
- Q: Confidence level threshold definition? → A: Ratio-based: 0 defaults = high, 1-49% of applicable properties defaulted = medium, 50%+ = low.

## User Scenarios & Testing

### User Story 1 - Zero-Cost Configuration Resources Return $0 (Priority: P1)

A FinFocus user estimates costs for an AWS stack that includes
LaunchTemplates, LaunchConfigurations, VPCs, Subnets, Security Groups,
and IAM resources. The plugin returns $0 for all configuration-only
resources without errors, regardless of whether the request comes through
the projected cost or actual cost path.

**Why this priority**: LaunchTemplates incorrectly priced at ~$35/mo
directly inflates total cost estimates, misleading users. GetActualCost
errors for VPCs/Subnets generate noise in debug output and cause
resources to fall back to the wrong plugin adapter.

**Independent Test**: Send a GetProjectedCost and GetActualCost request
for each zero-cost resource type and verify $0 response with no errors.

**Acceptance Scenarios**:

1. **Given** a LaunchTemplate resource descriptor, **When**
   GetProjectedCost is called, **Then** the response returns $0.00/mo
   with billing detail indicating a configuration-only resource.
2. **Given** a LaunchConfiguration resource descriptor, **When**
   GetProjectedCost is called, **Then** the response returns $0.00/mo.
3. **Given** a VPC ARN, **When** GetActualCost is called, **Then** the
   response returns $0.00/mo without "tags missing sku" errors.
4. **Given** a Subnet ARN, **When** GetActualCost is called, **Then**
   the response returns $0.00/mo without errors.
5. **Given** a SecurityGroup ARN, **When** GetActualCost is called,
   **Then** the response returns $0.00/mo without errors.
6. **Given** an IAM resource ARN, **When** GetActualCost is called,
   **Then** the response returns $0.00/mo without errors.

---

### User Story 2 - Sparse Properties Signal Uncertainty (DEFERRED)

**Status**: Deferred - blocked on rshade/finfocus-spec#381 (add metadata
map to GetProjectedCostResponse). Will be implemented as a separate
feature branch once the proto change lands.

**Original issue**: #292

---

### User Story 3 - Clean Lint Compliance (Priority: P2)

A project maintainer runs `make lint` and receives zero issues. The
golangci-lint configuration is committed to the repository and protected
from accidental deletion. All high-priority code correctness issues
(unchecked errors, incorrect error comparisons) and medium-priority code
quality issues (proto getter usage, naming conventions) are resolved.

**Why this priority**: Lint compliance prevents bugs from accumulating
and ensures consistent code quality. It also unblocks the golangci-lint
upgrade (#291) and CI enforcement.

**Independent Test**: Run `make lint` and verify zero issues. Verify
`.golangci.yml` is tracked in git.

**Acceptance Scenarios**:

1. **Given** the repository with `.golangci.yml` committed, **When**
   `make lint` is run, **Then** zero issues are reported.
2. **Given** a PR that deletes `.golangci.yml`, **When** the CI pipeline
   runs, **Then** the lint step fails, preventing the deletion.
3. **Given** code that uses direct proto field access, **When** lint runs
   after fixes, **Then** no `protogetter` violations are found.
4. **Given** code with `err == io.EOF` comparisons, **When** lint runs
   after fixes, **Then** no `errorlint` violations are found.

---

### Edge Cases

- What happens when a LaunchTemplate has an `instanceType` tag? The
  plugin should still return $0 (the template itself has no cost).
- What happens when GetActualCost receives an ARN for an unknown
  resource type that is not in the zero-cost list? It should follow the
  existing error path (require SKU from tags).
- What happens when `make lint` is run without `.golangci.yml`? The
  Makefile should fail with a clear error message.

## Requirements

### Functional Requirements

- **FR-001**: The plugin MUST return $0 cost for `ec2/launchtemplate`
  and `ec2/launchconfiguration` Pulumi resource types in both
  GetProjectedCost and GetActualCost paths.
- **FR-002**: The GetActualCost path MUST check zero-cost resource
  patterns before requiring SKU extraction from tags.
- **FR-003**: Zero-cost resources in GetActualCost MUST return with
  the plugin's adapter name (not fall back to another plugin).
- **FR-004**: The `.golangci.yml` configuration MUST be committed and
  tracked in version control.
- **FR-005**: All `errcheck` violations (unchecked `proto.Clone()` error
  returns) MUST be fixed.
- **FR-006**: All `errorlint` violations (`err == io.EOF` instead of
  `errors.Is()`) MUST be fixed.
- **FR-007**: All `protogetter` violations (direct proto field access
  instead of getter methods) MUST be fixed.
- **FR-008**: `make lint` MUST pass with zero issues after all fixes.
- **FR-009**: `make test` MUST pass with zero failures after all fixes.

### Key Entities

- **Zero-Cost Resource**: An AWS resource type that has no direct
  runtime cost (VPC, Subnet, SecurityGroup, IAM, LaunchTemplate,
  LaunchConfiguration). Identified by service name or Pulumi resource
  type pattern.
- **Property Confidence** (DEFERRED): Blocked on
  rshade/finfocus-spec#381. Will use proto metadata map once available.

## Success Criteria

### Measurable Outcomes

- **SC-001**: All zero-cost resource types return $0 through both
  projected and actual cost paths with zero error log entries.
- **SC-002**: `make lint` reports zero issues across the entire
  codebase.
- **SC-003**: `make test` passes with zero failures and no regressions
  in existing test coverage.

## Assumptions

- LaunchTemplate and LaunchConfiguration are always zero-cost regardless
  of their tags or attributes.
- The golangci-lint fix scope covers high-priority (8 issues) and
  medium-priority (73 issues) categories. Low-priority style issues
  (308 items) will be addressed selectively with justified exclusions.
- The `wastedassign` issue in `projected.go` will be fixed as part of
  the lint compliance work.

## Dependencies

- **Blocked (deferred)**: rshade/finfocus-spec#381 (metadata map proto
  field) blocks #292 (sparse property confidence signal).
- **Downstream**: #291 (golangci-lint upgrade) is unblocked by this
  batch completing the lint compliance work.

## Related Issues

- #294 - LaunchTemplate incorrectly priced as EC2 instance
- #293 - GetActualCost should handle zero-cost resources without error
- #292 - Handle sparse OldState properties (DEFERRED, blocked on finfocus-spec#381)
- #289 - Resolve all golangci-lint issues and protect .golangci.yml
