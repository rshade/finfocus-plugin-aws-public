# Feature Specification: CloudWatch Cost Estimation

**Feature Branch**: `019-cloudwatch-cost`  
**Created**: 2025-12-30  
**Status**: Draft  
**Input**: User description: Implement CloudWatch cost estimation for logs and metrics.

## Clarifications

### Session 2025-12-30

- Q: Should we strictly enforce the first-tier pricing ($0.30) for ALL metrics? → A: Implement full tiered pricing (Option A) to avoid massive overestimation for large-scale users.
- Q: How should the system behave if pricing data is unavailable for the requested region? → A: Soft Failure (Option A) - Return $0.00 cost and log a warning to avoid blocking the report.
- Q: What data source strategy should be used for CloudWatch pricing data? → A: Embed JSON from AWS Price List API (like EC2/EBS) - consistent with existing architecture.

## User Scenarios & Testing *(mandatory)*

<!--
  IMPORTANT: User stories should be PRIORITIZED as user journeys ordered by importance.
  Each user story/journey must be INDEPENDENTLY TESTABLE - meaning if you implement just ONE of them,
  you should still have a viable MVP (Minimum Viable Product) that delivers value.
-->

### User Story 1 - Estimate Log Costs (Ingestion & Storage) (Priority: P1)

As a cloud cost analyst, I want to estimate the monthly cost of CloudWatch Logs based on expected ingestion volume and storage retention, so that I can predict the cost of logging for my applications.

**Why this priority**: Log ingestion and storage are typically the largest components of CloudWatch costs in production environments.

**Independent Test**: Can be fully tested by providing a ResourceDescriptor with `log_ingestion_gb` and `log_storage_gb` tags and verifying the output cost matches the region's tiered pricing (e.g., first 10TB @ $0.50/GB, next 20TB @ $0.25/GB for ingestion; flat $0.03/GB-mo for storage in us-east-1).

**Acceptance Scenarios**:

1. **Given** a CloudWatch resource with `sku: logs` and tag `log_ingestion_gb: 100`, **When** calculating projected cost, **Then** result equals tiered ingestion cost (100GB falls in Tier 1 @ $0.50/GB = $50.00 in us-east-1).
2. **Given** a CloudWatch resource with `sku: logs` and tag `log_storage_gb: 500`, **When** calculating projected cost, **Then** result equals 500 * [Region Storage Rate].
3. **Given** a CloudWatch resource with both ingestion and storage tags, **When** calculating projected cost, **Then** result equals sum of both calculations.

---

### User Story 2 - Estimate Custom Metric Costs (Priority: P2)

As a cloud cost analyst, I want to estimate the cost of CloudWatch Custom Metrics based on the number of metrics I plan to send, so that I can budget for monitoring infrastructure.

**Why this priority**: Custom metrics are a significant cost driver for monitoring-heavy applications, though often secondary to high-volume logging.

**Independent Test**: Can be tested by providing a ResourceDescriptor with `custom_metrics` tag and verifying cost matches tiered pricing (e.g., first 10k @ $0.30, next 240k @ $0.10, next 750k @ $0.05, over 1M @ $0.02/metric).

**Acceptance Scenarios**:

1. **Given** a CloudWatch resource with `sku: metrics` and tag `custom_metrics: 50`, **When** calculating projected cost, **Then** result equals 50 * [First Tier Metric Rate].
2. **Given** a CloudWatch resource with `custom_metrics: 0`, **When** calculating projected cost, **Then** result is $0.00.

---

### User Story 3 - Combined Usage Estimation (Priority: P3)

As a DevOps engineer, I want to see the total projected cost for a CloudWatch configuration that includes both logs and metrics, so that I have a holistic view of the service cost.

**Why this priority**: Represents the real-world scenario where a single service or logical grouping consumes both logs and metrics.

**Independent Test**: Provide a ResourceDescriptor with all tags (`log_ingestion_gb`, `log_storage_gb`, `custom_metrics`) and verify total is the sum of all components.

**Acceptance Scenarios**:

1. **Given** a CloudWatch resource with `sku: combined` and tags for ingestion, storage, and metrics, **When** calculating projected cost, **Then** result equals sum of ingestion + storage + metric costs.
2. **Given** a CloudWatch resource with missing tags, **When** calculating projected cost, **Then** missing dimensions are treated as 0 usage.

---

### Edge Cases

- What happens when tags are non-numeric strings? -> Default to 0 AND log a warning message (enables debugging while allowing report to complete).
- What happens when pricing data is missing for a region? -> Return $0.00 cost and log a warning (Soft Failure).
- What happens when usage is extremely high? -> Calculation should handle standard float64 range without overflow (unlikely to be reached).

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST identify and support CloudWatch resources when presented with them.
- **FR-002**: System MUST retrieve pricing rates for Log Ingestion from the pricing data source.
- **FR-003**: System MUST retrieve pricing rates for Log Storage from the pricing data source.
- **FR-004**: System MUST retrieve pricing rates for Custom Metrics from the pricing data source.
- **FR-005**: System MUST calculate projected cost for Log Ingestion based on provided usage volume (e.g., in GB).
- **FR-006**: System MUST calculate projected cost for Log Storage based on provided storage volume (e.g., in GB-Months).
- **FR-007**: System MUST calculate projected cost for Custom Metrics using standard AWS tiered pricing logic (e.g., first 10k at Tier 1, next 240k at Tier 2, etc.).
- **FR-008**: System MUST support cost estimation for logs, metrics, or both combined in a single resource.
- **FR-009**: System MUST default usage values to 0 if usage information is missing.
- **FR-010**: System MUST include CloudWatch pricing data for all supported regions.

### Key Entities *(include if feature involves data)*

- **Pricing Rates**: The set of unit costs for ingestion, storage, and metrics specific to a region.
- **Resource Usage**: The input data defining the volume of logs (ingestion/storage) and count of metrics.

### Assumptions

- Pricing for alarms, dashboards, and other CloudWatch features is out of scope.
- "Vended logs" (free tier/AWS service logs) are not distinguished; all logs are treated as paid custom ingestion.
- CloudWatch pricing data will be fetched via AWS Price List API and embedded as JSON at build time, following the established pattern used for EC2, EBS, and other services.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Projected cost calculations match manual calculation using AWS public pricing examples for us-east-1 within $0.01 tolerance.
- **SC-002**: System correctly reports "supported" status for valid CloudWatch resources.
- **SC-003**: Plugin binary size increase remains within acceptable limits (< 5MB) after adding CloudWatch pricing data.
- **SC-004**: System returns valid cost projections (non-error state) for all standard usage scenarios in integration tests.
