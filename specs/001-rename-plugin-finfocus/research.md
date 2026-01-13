# Research Findings: Plugin Rename

**Feature Branch:** `001-rename-plugin-finfocus`
**Status:** Completed

## Unknowns & Clarifications

### 1. Scope of `tools/` Changes
- **Question:** Are there any hardcoded references in the `tools/` directory that need special handling?
- **Finding:** Grep search confirmed zero occurrences of `finfocus` in the `tools/` directory.
- **Decision:** No updates required for `tools/` logic, only verify they continue to work with new paths if they rely on implicit directory structures (unlikely based on `Makefile`).

### 2. Status of `specs/` Updates
- **Question:** How many files in `specs/` need updating?
- **Finding:** 263 occurrences of `finfocus-plugin-aws-public` found in `specs/`.
- **Decision:** Systematic find-and-replace is required across the `specs/` directory to maintain historical consistency and searchability, as per the clarification.

### 3. Environment Variable Strategy
- **Question:** Which legacy variables need support?
- **Finding:** 
    - `FINFOCUS_TEST_MODE` replaces `FINFOCUS_TEST_MODE` (and legacy `TEST_MODE`).
    - `FINFOCUS_MAX_BATCH_SIZE` replaces `MAX_BATCH_SIZE` (implied legacy).
    - `FINFOCUS_STRICT_VALIDATION` replaces `STRICT_VALIDATION` (implied legacy).
- **Decision:** Explicitly support `FINFOCUS_` prefixed variables for `MAX_BATCH_SIZE` and `STRICT_VALIDATION` if they were ever used, or at least document the transition. The code currently supports `MAX_BATCH_SIZE` as deprecated. I will add `FINFOCUS_` equivalents to `testmode.go` style logic if they aren't already covered by the generic legacy handling.

### 4. Build System State
- **Question:** Is the `Makefile` already updated?
- **Finding:** Yes, `Makefile` and `.goreleaser.yaml` already use `finfocus-plugin-aws-public`.
- **Decision:** Verify these files essentially just need a "sanity check" and potentially a clean build to confirm.

## Technical Decisions

- **Global Rename:** Execute a `sed` based replacement for `specs/`.
- **Legacy Support:** Ensure `FINFOCUS_` vars are recognized and log a warning.
- **Verification:** Run the full test suite (`make test`) which covers unit and integration tests.
