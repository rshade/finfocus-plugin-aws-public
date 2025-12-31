# Research: DynamoDB Hardening Bundle

**Feature**: 020-dynamodb-hardening
**Date**: 2025-12-31

## Overview

This feature is an internal hardening effort. No external research was required as all patterns are established within the codebase.

## Decisions

### 1. Validation Approach

**Decision**: Create reusable validation helper methods on AWSPublicPlugin

**Rationale**: Keeps validation logic DRY and testable. The pattern can be reused for other services that parse numeric tags.

**Alternatives Considered**:
- Inline validation in estimateDynamoDB(): Rejected - duplicates code for int64 vs float64, harder to test
- Package-level validation functions: Rejected - would require passing logger, traceID; method on plugin is cleaner

### 2. Warning Log Level

**Decision**: Use `log.Warn()` for both validation failures and missing pricing data

**Rationale**: Both conditions are recoverable (default to 0, return $0 cost) but indicate something the user should be aware of. Error level would suggest a failure.

**Alternatives Considered**:
- `log.Info()`: Rejected - too easy to miss, these are actionable warnings
- `log.Error()`: Rejected - operation succeeds with degraded accuracy, not a failure

### 3. Billing Detail Format

**Decision**: Append `(pricing unavailable: component1, component2)` to billing_detail

**Rationale**: Provides machine-parseable format while remaining human-readable. Lists specific components that are missing.

**Alternatives Considered**:
- Generic "(pricing unavailable)": Rejected - doesn't indicate which components
- Separate response field: Rejected - would change proto, out of scope

### 4. Integration Test Structure

**Decision**: Create new `integration_dynamodb_test.go` following existing pattern

**Rationale**: Keeps DynamoDB tests organized separately. Existing integration tests use the same gRPC client pattern.

**Alternatives Considered**:
- Add to existing `integration_test.go`: Rejected - file is already 300+ lines, separate file is cleaner
- Use unit tests with mock client: Rejected - FR-016 specifically requires integration tests with real pricing

## Technology Choices

No new dependencies. All implementations use:
- Go standard library (`strconv`, `strings`, `fmt`)
- zerolog (already in use)
- pluginsdk (already in use)
