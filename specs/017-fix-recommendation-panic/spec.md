# Feature Specification: Bug Fix and Documentation Sprint - Dec 2025

**Feature Branch**: `017-fix-recommendation-panic`  
**Created**: 2025-12-19  
**Status**: Draft  
**Input**: User description: "gh issue list --label 'bug' && gh issue list --label 'documentation' && gh issue list --label 'sdkupgrade'"

## Clarifications

### Session 2025-12-19
- Q: What is the specific milestone for the PORT environment variable removal? → A: Immediate deprecation (warn now, remove in next minor v0.x.x).
- Q: Where should the troubleshooting section be primary located? → A: Dedicated TROUBLESHOOTING.md file.
- Q: What format should be used for carbon CSV parsing error logs? → A: Structured JSON (zerolog).
- Q: What is the definitive source of truth for EC2 platform-to-OS mapping? → A: Internal mapping (hardcoded).
- Q: Where should correlation ID population for recommendations be documented? → A: Code Comments (GoDoc).

## User Scenarios & Testing *(mandatory)*

### User Story 1 - System Stability & Robustness (Priority: P1)

As a FinFocus user, I want the AWS public plugin to be stable and predictable, handling edge cases like missing pricing data or invalid region fallbacks without crashing or producing incorrect results, so that I can rely on the cost estimates.

**Why this priority**: Correctness and stability are the core value propositions of a cost estimation tool.

**Independent Test**: Verified by running the plugin against resources known to trigger edge cases (e.g., S3 buckets in global regions, resources with missing pricing) and ensuring it doesn't panic and returns valid or appropriately handled errors.

**Acceptance Scenarios**:

1. **Given** a recommendation with nil `Impact`, **When** processed in a batch, **Then** the plugin must not panic and should continue processing other resources (#123).
2. **Given** a malformed CSV in carbon data, **When** parsed, **Then** the system must log an error instead of failing silently (#142).
3. **Given** an S3 bucket in a global service context, **When** calculating costs, **Then** the system must produce valid ARNs even during region fallback (#113).
4. **Given** a request for projected costs, **When** validation is performed, **Then** it must strictly enforce all required parameters (#111).

---

### User Story 2 - Documentation Clarity (Priority: P2)

As a developer or user of the plugin, I want clear, concise, and up-to-date documentation and logging, so that I can understand how the tool works and troubleshoot issues effectively.

**Why this priority**: Good documentation reduces support overhead and improves developer experience.

**Independent Test**: Verified by reviewing the generated GoDoc, README, and log outputs to ensure they reflect the documented behavior.

**Acceptance Scenarios**:

1. **Given** the `GetUtilization` function, **When** viewing documentation, **Then** there should be no duplicate docstrings (#143).
2. **Given** a recommendation response, **When** viewing docs, **Then** the correlation ID population logic must be clearly explained (#128).
3. **Given** the `PORT` environment variable, **When** the plugin starts, **Then** the deprecation timeline must be clearly communicated (#116).
4. **Given** EC2 pricing results, **When** checking OS mapping, **Then** the documentation must clearly define how platforms map to operating systems (#63).

---

### Edge Cases

- **Carbon Data Integrity**: Handling missing trailing newlines in carbon package files (#145).
- **Troubleshooting**: Providing a dedicated section for common error scenarios (#60).

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001 (Panic Fix)**: System MUST verify `Impact` is non-nil before accessing properties in recommendations ([#123](https://github.com/rshade/finfocus-plugin-aws-public/issues/123)).
- **FR-002 (Carbon Logging)**: System MUST add structured JSON error logging (via `zerolog`) for CSV parsing failures in `parseInstanceSpecs` ([#142](https://github.com/rshade/finfocus-plugin-aws-public/issues/142)).
- **FR-003 (Carbon Style)**: System MUST ensure trailing newlines are present in carbon package files ([#145](https://github.com/rshade/finfocus-plugin-aws-public/issues/145)).
- **FR-004 (S3 Validation)**: System MUST fix region fallback for S3 global services to prevent invalid ARN generation ([#113](https://github.com/rshade/finfocus-plugin-aws-public/issues/113)).
- **FR-005 (Cost Validation)**: System MUST restore comprehensive validation in `getProjectedCost` ([#111](https://github.com/rshade/finfocus-plugin-aws-public/issues/111)).
- **FR-006 (Doc Cleanup)**: System MUST consolidate duplicate docstrings in `GetUtilization` ([#143](https://github.com/rshade/finfocus-plugin-aws-public/issues/143)).
- **FR-007 (Correlation Docs)**: System MUST document how correlation IDs are populated for recommendations using Code Comments (GoDoc) ([#128](https://github.com/rshade/finfocus-plugin-aws-public/issues/128)).
- **FR-008 (Deprecation Docs)**: System MUST add a deprecation timeline for the `PORT` environment variable fallback, specifying removal in the next minor version v0.x.x as it is already deprecated ([#116](https://github.com/rshade/finfocus-plugin-aws-public/issues/116)).
- **FR-009 (OS Mapping Docs)**: System MUST document the internal platform-to-OS mapping logic for EC2 pricing, which is the source of truth ([#63](https://github.com/rshade/finfocus-plugin-aws-public/issues/63)).
- **FR-010 (Troubleshooting)**: System MUST add a dedicated `TROUBLESHOOTING.md` file with examples of common error scenarios ([#60](https://github.com/rshade/finfocus-plugin-aws-public/issues/60)).

### Key Entities *(include if feature involves data)*

- **Carbon Data**: Embedded CSV data containing instance power specifications.
- **S3 Resource**: Representation of an S3 bucket with its region and ARN.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Zero panics in recommendation processing.
- **SC-002**: 100% of reported bugs (#145, #142, #123, #113, #111) are verified fixed via unit or integration tests.
- **SC-003**: 100% of documentation tasks (#143, #128, #116, #63, #60) are completed and visible in the codebase or docs.
- **SC-004**: No regression in existing performance benchmarks after adding validation and logging.