# Research: Windows vs Linux Pricing Integration Tests

## Decision: Testing Framework & Approach
- **Decision**: Use Go's built-in testing package with the `integration` build tag.
- **Rationale**: This follows the established pattern in the codebase (e.g., `internal/plugin/integration_test.go`).
- **Alternatives considered**: Unit tests with mocked pricing. Rejected because the requirement explicitly calls for an end-to-end integration test that verifies the binary and embedded data.

## Decision: Region Selection
- **Decision**: Primary tests will target `us-east-1` (build tag `region_use1`).
- **Rationale**: `us-east-1` has the most comprehensive pricing data and is the project's standard region for testing.

## Decision: Platform & Tenancy Tags
- **Decision**: Use `platform: linux`, `platform: windows`, `tenancy: shared`, and `tenancy: dedicated`.
- **Rationale**: These match the extraction logic in `internal/plugin/ec2_attrs.go`. Note that "dedicated" maps to "Dedicated Instance" in AWS pricing.

## Decision: Architecture Selection
- **Decision**: Test both `x86_64` and `arm64`.
- **Rationale**: Per clarified requirements, architecture differentiation must be verified. Linux pricing is usually lower on ARM, while Windows pricing availability varies.

## Decision: Failure Mode
- **Decision**: Tests will explicitly fail if pricing data is missing or returns $0.
- **Rationale**: To prevent silent regressions where a lookup failure might fallback to a default (incorrect) price.
