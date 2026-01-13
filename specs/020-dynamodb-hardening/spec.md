# Feature Specification: DynamoDB Hardening Bundle

**Feature Branch**: `020-dynamodb-hardening`
**Created**: 2025-12-31
**Status**: Draft
**Input**: User description: "Consolidate four related DynamoDB improvements into a single hardening release that improves validation, error handling, observability, and test coverage"

**Related Issues**: #147, #149, #151, #152

## Clarifications

### Session 2025-12-31

- Q: Should `storage_gb` be validated for >= 0 like other numeric tags? → A: Yes, validate `storage_gb` >= 0 (consistent with other tags)

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Transparent Cost Estimation Feedback (Priority: P1)

As a developer using the FinFocus plugin, I want to receive clear warnings when my DynamoDB cost estimate may be incomplete or inaccurate, so that I can make informed decisions about my infrastructure costs.

**Why this priority**: Silent $0 estimates with no indication of failure directly mislead users about costs. This is the most impactful issue as it affects trust in the tool.

**Independent Test**: Can be fully tested by requesting a DynamoDB cost estimate with the fallback (no-pricing) embed and verifying that warning logs are emitted and billing_detail indicates unavailable pricing.

**Acceptance Scenarios**:

1. **Given** DynamoDB pricing data is missing for provisioned RCU, **When** a cost estimate is requested for a provisioned table, **Then** a warning is logged indicating "DynamoDB provisioned RCU pricing unavailable" and billing_detail includes "(pricing unavailable)"
2. **Given** DynamoDB pricing data is missing for on-demand read requests, **When** a cost estimate is requested for an on-demand table, **Then** a warning is logged indicating "DynamoDB on-demand read pricing unavailable" and billing_detail includes "(pricing unavailable)"
3. **Given** all DynamoDB pricing data is available, **When** a cost estimate is requested, **Then** no pricing warnings are logged and billing_detail does not contain "(pricing unavailable)"

---

### User Story 2 - Invalid Input Detection and Recovery (Priority: P2)

As a developer integrating the FinFocus plugin, I want the system to detect and handle invalid capacity/request values gracefully, so that erroneous inputs don't produce misleading results.

**Why this priority**: Invalid inputs (like negative capacity units) being silently accepted can produce negative or nonsensical cost estimates, compounding user confusion.

**Independent Test**: Can be fully tested by submitting DynamoDB cost requests with negative tag values and verifying warnings are logged and values default to 0.

**Acceptance Scenarios**:

1. **Given** a DynamoDB cost request with `read_capacity_units: "-100"`, **When** the estimate is calculated, **Then** a warning is logged indicating "negative value for read_capacity_units, defaulting to 0" and the calculation uses 0
2. **Given** a DynamoDB cost request with `write_requests_per_month: "-50000"`, **When** the estimate is calculated, **Then** a warning is logged indicating "negative value for write_requests_per_month, defaulting to 0" and the calculation uses 0
3. **Given** a DynamoDB cost request with `storage_gb: "0"`, **When** the estimate is calculated, **Then** no warning is logged and 0 is accepted as valid
4. **Given** a DynamoDB cost request with valid positive values, **When** the estimate is calculated, **Then** no validation warnings are logged

---

### User Story 3 - Accurate Pricing Extraction (Priority: P3)

As a FinFocus maintainer, I want DynamoDB pricing lookups to precisely match the correct product families, so that provisioned and on-demand pricing are never confused.

**Why this priority**: Incorrect pricing matches can silently produce wrong cost estimates. While less visible than $0 estimates, this affects accuracy for all users.

**Independent Test**: Can be fully tested by verifying pricing extraction tests return distinct, non-zero prices for all 5 DynamoDB pricing lookups (storage, RCU, WCU, on-demand read, on-demand write).

**Acceptance Scenarios**:

1. **Given** embedded pricing data for us-east-1, **When** provisioned RCU price is looked up, **Then** the returned price matches AWS's published provisioned RCU rate (non-zero)
2. **Given** embedded pricing data for us-east-1, **When** on-demand read price is looked up, **Then** the returned price matches AWS's published on-demand read request rate (non-zero)
3. **Given** embedded pricing data for us-east-1, **When** all 5 DynamoDB pricing methods are called, **Then** each returns a distinct, non-zero value appropriate to its pricing model

---

### User Story 4 - End-to-End Validation via Integration Tests (Priority: P4)

As a FinFocus developer, I want comprehensive integration tests for DynamoDB cost estimation, so that regressions in pricing extraction or estimation logic are caught before release.

**Why this priority**: Integration tests provide confidence that the entire pipeline works correctly, but are less urgent than fixing the underlying issues that cause silent failures.

**Independent Test**: Can be fully tested by running integration tests against a built binary with real pricing data and verifying correct cost estimates.

**Acceptance Scenarios**:

1. **Given** a binary built with `region_use1` tag, **When** a DynamoDB provisioned table estimate is requested via gRPC, **Then** the response contains non-zero cost_per_month and accurate billing_detail
2. **Given** a binary built with `region_use1` tag, **When** a DynamoDB on-demand table estimate is requested via gRPC, **Then** the response contains non-zero cost_per_month and accurate billing_detail
3. **Given** all integration tests, **When** `go test -tags=integration ./internal/plugin/...` is run, **Then** all DynamoDB tests pass

---

### Edge Cases

- What happens when all DynamoDB pricing data is missing? The system should still return a response with $0 cost and comprehensive warning logs.
- How does the system handle extremely large valid values (e.g., `storage_gb: "999999999999"`)? The system should accept them without validation warnings.
- What happens when parsing a tag value fails entirely (e.g., `read_capacity_units: "abc"`)? The system should log a parse error and default to 0.
- How does the system handle mixed valid/invalid inputs in a single request? Each tag should be validated independently.

## Requirements *(mandatory)*

### Functional Requirements

**Pricing Lookup Error Handling (#147)**

- **FR-001**: System MUST check the `ok` return value for all 5 DynamoDB pricing lookup methods (storage, provisioned RCU, provisioned WCU, on-demand read, on-demand write)
- **FR-002**: System MUST log a warning message when any DynamoDB pricing lookup returns `ok=false`
- **FR-003**: System MUST include "(pricing unavailable)" notation in `billing_detail` for each component where pricing data was not found
- **FR-004**: System MUST still return a valid response (with $0 for unavailable components) rather than returning an error when pricing is missing

**Input Validation (#151)**

- **FR-005**: System MUST validate that `read_capacity_units` tag value is >= 0
- **FR-006**: System MUST validate that `write_capacity_units` tag value is >= 0
- **FR-007**: System MUST validate that `read_requests_per_month` tag value is >= 0
- **FR-008**: System MUST validate that `write_requests_per_month` tag value is >= 0
- **FR-009**: System MUST validate that `storage_gb` tag value is >= 0
- **FR-010**: System MUST log a warning when negative values are detected, including the tag name and invalid value
- **FR-011**: System MUST default negative values to 0 for calculation purposes

**Product Family Matching (#149)**

- **FR-012**: System MUST use product family filters specific enough to distinguish provisioned capacity pricing from on-demand pricing
- **FR-013**: System SHOULD log when multiple potential matches are found for a pricing lookup (ambiguous match detection)

**Testing (#152)**

- **FR-014**: System MUST include unit tests for negative value validation scenarios
- **FR-015**: System MUST include unit tests for missing pricing data warning behavior
- **FR-016**: System MUST include integration tests that verify real pricing extraction from embedded data
- **FR-017**: All tests MUST pass with `go test -tags=region_use1 ./...`

### Key Entities

- **ResourceDescriptor**: The input containing DynamoDB table configuration via tags (capacity units, request counts, storage size, SKU mode)
- **GetProjectedCostResponse**: The output containing unit_price, cost_per_month, currency, and billing_detail
- **PricingClient**: The component responsible for looking up prices from embedded JSON data
- **DynamoDB Pricing Components**: Storage (per GB-month), Provisioned RCU (per hour), Provisioned WCU (per hour), On-Demand Read (per million requests), On-Demand Write (per million requests)

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% of DynamoDB cost requests with missing pricing data produce warning logs (currently 0%)
- **SC-002**: 100% of negative capacity/request values produce validation warnings (currently 0%)
- **SC-003**: All 5 DynamoDB pricing lookups return correct, non-zero values when tested against us-east-1 embedded data
- **SC-004**: Integration test coverage for DynamoDB cost estimation increases from 0 tests to at least 2 end-to-end tests (provisioned and on-demand modes)
- **SC-005**: All existing unit tests continue to pass after changes (no regressions)
- **SC-006**: `make lint` passes with no new warnings
- **SC-007**: `make test` passes with all new tests included

## Assumptions

- The existing DynamoDB pricing data embedded in `region_use1` binaries contains valid pricing for all 5 components (storage, RCU, WCU, on-demand read, on-demand write)
- The `zerolog` logger is already configured and available in the plugin context
- The `storage_gb` tag is parsed as a float64, while capacity unit tags are parsed as int64
- Warning logs should use `log.Warn()` level, not `log.Error()` since missing pricing is recoverable
- Integration tests follow the existing pattern in `internal/plugin/integration_*.go` files
