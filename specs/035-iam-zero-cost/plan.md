# Implementation Plan - IAM Zero-Cost Resource Handling

**Feature Branch**: `035-iam-zero-cost`
**Goal**: Correctly identify and estimate AWS IAM resources in Pulumi stacks as having zero cost.

## Technical Context

### Architecture & Dependencies

- **Component**: `finfocus-plugin-aws-public` (gRPC plugin)
- **Files to Modify**:
    - `internal/plugin/constants.go`: Add `iam` to `ZeroCostServices` map.
    - `internal/plugin/supports.go`: Update `Supports` method to handle `iam`.
    - `internal/plugin/projected.go`:
        - Update `GetProjectedCost` to handle `iam` case (return $0).
        - Update `detectService` to map `iam` correctly.
        - Update `normalizeResourceType` to handle `aws:iam/*` prefix normalization.
- **Dependencies**: None (pure logic update).

### Integration Points

- **Input**: `ResourceDescriptor` from gRPC `GetProjectedCost` and `Supports` calls.
    - Specifically, `resource_type` field (e.g., `aws:iam/user:User`).
- **Output**:
    - `SupportsResponse`: `supported: true` for IAM resources.
    - `GetProjectedCostResponse`:
        - `cost_per_month: 0.0`
        - `billing_detail: "IAM - no direct AWS charges"`

### Unknowns & Risks

- **Unknowns**: None.
- **Risks**:
    - Normalization logic might be too broad or too narrow if not tested correctly against various Pulumi resource type formats. *Mitigation: Comprehensive unit tests for normalization.*

## Constitution Check

### Core Principles

- [x] **I. Code Quality**: Changes are simple, stateless, and explicit.
- [x] **II. Testing**: Plan includes unit tests for normalization and cost estimation.
- [x] **III. Protocol**: Adheres to gRPC protocol, returning valid proto responses.
- [x] **IV. Performance**: Zero-cost check is immediate (<1ms), no external calls.
- [x] **V. Build**: `make lint` and `make test` will be enforced.

### Protocol Compliance

- [x] **State**: Stateless.
- [x] **Logging**: Uses zerolog (if any logging is added).
- [x] **Errors**: Uses standard proto error codes if needed (likely not applicable for successful zero-cost return).

### Performance

- [x] **Latency**: Negligible impact.
- [x] **Resource Usage**: No additional memory/cpu usage.

## Phases

### Phase 0: Outline & Research

*Goal: Confirm normalization strategy.*

1.  **Research**: None required. Spec clarification confirmed prefix match (`aws:iam/*`) and case-insensitive normalization.

### Phase 1: Implementation & Unit Tests

*Goal: Implement logic and verify with unit tests.*

1.  **Logic Implementation**:
    - Update `ZeroCostServices` in `constants.go`.
    - Update `Supports` in `supports.go`.
    - Update `GetProjectedCost`, `detectService`, `normalizeResourceType` in `projected.go`.
2.  **Unit Testing**:
    - Test `isZeroCostResource` with "iam".
    - Test `Supports` with various IAM resource types.
    - Test `normalizeResourceType` with mixed case and various `aws:iam/*` inputs.
    - Test `GetProjectedCost` returns $0 and correct billing detail.

### Phase 2: Integration Verification

*Goal: Verify end-to-end behavior.*

1.  **Integration Test**:
    - Add a test case in `internal/plugin` (or existing integration suite) that constructs a `ResourceDescriptor` for an IAM user and asserts the response.
    - Verify no regression for existing zero-cost resources (VPC, etc.).