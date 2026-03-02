# Feature Specification: Port Binding & Contract Violation Fixes

**Feature Branch**: `001-fix-port-contract-bugs`  
**Created**: Thursday, February 12, 2026  
**Status**: Draft  
**Input**: User description: "# Plan: Fix Port Binding + Contract Violation Bugs ## Context `finfocus cost actual --pulumi-state stack.json` returns `$0` for all resources due to three compounding issues: (1) plugin ignores `--port` CLI flag so Core can't connect, (2) wrong region binary installed (Core-side, no changes here), (3) when pricing data is unavailable, plugin returns `$0` instead of empty results, preventing Core from falling back to other plugins. ## Issue 1: Port Binding — `--port` Flag Ignored **Root cause:** `main.go` never calls `flag.Parse()` or `pluginsdk.ParsePortFlag()`. The SDK (v0.5.6) registers `flag.Int("port", 0, ...)` at init time, but the plugin only checks env vars. **File:** `cmd/finfocus-plugin-aws-public/main.go` **Changes:** 1. Add `"flag"` to imports 2. Add `flag.Parse()` as first line of `run()` (before any SDK calls) 3. Insert `pluginsdk.ParsePortFlag()` as highest priority in port chain: - `--port` flag > `FINFOCUS_PLUGIN_PORT` env > legacy `PORT` env > ephemeral Current (line 67): ```go port := pluginsdk.GetPort() ``` New: ```go port := pluginsdk.ParsePortFlag() // Priority 1: --port CLI flag if port == 0 { port = pluginsdk.GetPort() // Priority 2: FINFOCUS_PLUGIN_PORT env } ``` ## Issue 2: Wrong Region — No Changes Needed Core-side `registry.json` configuration issue. Plugin correctly exposes region via `PluginInfo.Metadata["region"]`. ## Issue 3: Contract Violation — $0 Instead of Empty Results ### 3a. Define `PricingUnavailableError` type **New file:** `internal/plugin/errors.go` ```go type PricingUnavailableError struct { Service string // "EC2", "EBS", etc. SKU string // Instance type or resource identifier BillingDetail string // Human-readable explanation (for GetProjectedCost backward compat) } ``` ### 3b. Change estimators to return error when pricing not found **File:** `internal/plugin/projected.go` For each service with all-or-nothing pricing, change from returning `($0 response, nil)` to returning `(nil, &PricingUnavailableError{...})` when the primary pricing lookup returns `found=false`: | Estimator | Pricing Lookup | Current line | |-----------|---------------|--------------| | `estimateEC2` | `EC2OnDemandPricePerHour()` | ~258 | | `estimateEBS` | `EBSPricePerGBMonth()` | ~347 | | `estimateEKS` | `EKSClusterPricePerHour()` | ~1140 | | `estimateS3` | `S3StoragePricePerGBMonth()` | ~441 | | `estimateLambda` | request + compute prices | ~1256 | | `estimateElastiCache` | node type price | ~1700 | | `estimateELB` | hourly rate | ~842 | | `estimateNATGateway` | hourly rate | ~1368 | **Keep current behavior** for partial/complex services: `estimateDynamoDB`, `estimateCloudWatch` (they track unavailable components individually and return partial costs). **`estimateRDS`** already returns an error for missing instance pricing — wrap it in `PricingUnavailableError` so GetActualCost can distinguish it from validation errors. ### 3c. Catch error in `GetProjectedCost` for backward compatibility **File:** `internal/plugin/projected.go` (lines 207-210) After the switch/case, catch `PricingUnavailableError` and convert to $0 response: ```go if err != nil { var pue *PricingUnavailableError if errors.As(err, &pue) { resp = &pbc.GetProjectedCostResponse{ CostPerMonth: 0, UnitPrice: 0, Currency: "USD", BillingDetail: pue.BillingDetail, } } else { p.logErrorWithID(traceID, "GetProjectedCost", err, ...) return nil, err } } ``` ### 3d. Catch error in `GetActualCost` — return empty results **File:** `internal/plugin/plugin.go` (lines 367-372) ```go projectedResp, err := p.getProjectedForResource(traceID, resource, resolver) if err != nil { var pue *PricingUnavailableError if errors.As(err, &pue) { // Empty results = "I cannot answer" → Core falls back to other plugins return &pbc.GetActualCostResponse{Results: []*pbc.ActualCostResult{}}, nil } // Other errors propagate as gRPC errors errCode := extractErrorCode(err) p.logErrorWithID(traceID, "GetActualCost", err, errCode) return nil, err } ``` ### 3e. Fix routing inconsistency in `getProjectedForResource` **File:** `internal/plugin/actual.go` (lines 245-271) **Bug found:** `getProjectedForResource` routes `s3`, `lambda`, `rds`, `dynamodb` to `estimateStub` (always $0), while `GetProjectedCost` routes them to their real estimators. This means GetActualCost ALWAYS returns $0 for these services regardless of pricing data. **Fix:** Route to real estimators + add zero-cost resource handling: ```go switch serviceType { case "ec2": return p.estimateEC2(...) case "ebs": return p.estimateEBS(...) case "eks": return p.estimateEKS(...) case "elb": return p.estimateELB(...) case "natgw": return p.estimateNATGateway(...) case "cloudwatch": return p.estimateCloudWatch(...) case "elasticache": return p.estimateElastiCache(...) case "s3": return p.estimateS3(traceID, resource) // was: estimateStub case "lambda": return p.estimateLambda(traceID, resource) // was: estimateStub case "rds": return p.estimateRDS(traceID, resource) // was: estimateStub case "dynamodb": return p.estimateDynamoDB(traceID, resource) // was: estimateStub case "vpc", "securitygroup", "subnet", "iam": return p.estimateZeroCostResource(traceID, resource, serviceType), nil default: ... } ``` ## Files Modified | File | Change | |------|--------| | `cmd/finfocus-plugin-aws-public/main.go` | Add `flag.Parse()` + `ParsePortFlag()` | | `internal/plugin/errors.go` | **New:** `PricingUnavailableError` type | | `internal/plugin/projected.go` | Estimators return error; `GetProjectedCost` catches for compat | | `internal/plugin/plugin.go` | `GetActualCost` catches error → empty results | | `internal/plugin/actual.go` | Fix routing: real estimators + zero-cost handling | | `internal/plugin/errors_test.go` | **New:** Error type tests | | `internal/plugin/actual_test.go` | Empty results + zero-cost regression tests | | `internal/plugin/projected_test.go` | Backward compat $0 tests | ## Verification ```bash make test # All unit tests pass make lint # Linter passes (includes embed verification) make build-default-region # Compiles with real pricing data # Specific test validation: go test -v ./internal/plugin/... -run TestPricingUnavailableError go test -v ./internal/plugin/... -run TestGetActualCost_PricingUnavailable go test -v ./internal/plugin/... -run TestGetProjectedCost_PricingUnavailable go test -v ./internal/plugin/... -run TestGetActualCost_ZeroCostResource ```"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Explicit Port Selection (Priority: P1)

As a system administrator or orchestration tool, I want to be able to start the plugin on a specific port using the `--port` flag so that I can manage network resource allocation and avoid port conflicts with other services.

**Why this priority**: Essential for reliable deployment in environments where port allocation is strictly managed or where multiple plugins are running.

**Independent Test**: Start the plugin with `./finfocus-plugin-aws-public --port 9090` and verify it listens on port 9090.

**Acceptance Scenarios**:

1. **Given** the plugin binary is available, **When** it is started with `--port 9090`, **Then** the log output SHOULD show `PORT=9090`.
2. **Given** the `FINFOCUS_PLUGIN_PORT` environment variable is set to `8080` and the `--port 9090` flag is provided, **When** the plugin starts, **Then** it SHOULD prioritize the flag and listen on port `9090`.

---

### User Story 2 - Fail-Open Contract Compliance (Priority: P1)

As a FinFocus Core user, I want the AWS Public plugin to return empty results instead of $0 when it lacks pricing data for a resource, so that the Core can fall back to more accurate data sources (like AWS Actual Cost) instead of assuming the resource is free.

**Why this priority**: Prevents massive underestimation of costs by ensuring the plugin only reports costs it is certain about. Returning $0 for unknown resources is a contract violation that blocks fallbacks.

**Independent Test**: Request `GetActualCost` for an instance type not present in the local pricing database and verify the response contains an empty `Results` slice.

**Acceptance Scenarios**:

1. **Given** a resource request for an instance type that does NOT exist in the embedded pricing data, **When** `GetActualCost` is called, **Then** the response MUST contain 0 results.
2. **Given** the same resource request, **When** `GetProjectedCost` is called, **Then** the response SHOULD return $0 with a `BillingDetail` explaining that pricing data was unavailable (for backward compatibility).

---

### User Story 3 - Full Resource Routing for Actual Cost (Priority: P2)

As a FinFocus user, I want services like S3, Lambda, and DynamoDB to have their costs estimated during "Actual Cost" runs just as they are in "Projected Cost" runs, so that my actual cost reports are as complete as possible.

**Why this priority**: Fixes a bug where several major services were hardcoded to return $0 in Actual Cost mode, leading to incomplete reports even when pricing was available.

**Independent Test**: Run an Actual Cost estimation on a Pulumi state containing S3 buckets and Lambda functions and verify non-zero costs are returned.

**Acceptance Scenarios**:

1. **Given** a Pulumi state with an S3 bucket, **When** `GetActualCost` is called, **Then** the plugin SHOULD calculate cost based on storage size instead of returning a stub $0.
2. **Given** a request for a zero-cost resource like an IAM Role, **When** `GetActualCost` is called, **Then** it SHOULD return $0 with a clear message indicating it is a zero-cost resource.

---

### Edge Cases

- **Missing SKU**: What happens when a resource has a valid type but a malformed or missing SKU/InstanceType? The system should treat this as "pricing unavailable" and trigger the empty results fallback.
- **Mixed Results**: How does the system handle a batch request where some resources have pricing and others don't? Each resource should be evaluated independently; those with pricing return results, those without return nothing (or an error depending on the RPC type).

## Requirements *(mandatory)*

### Assumptions

- **Backward Compatibility**: It is assumed that `GetProjectedCost` MUST remain compatible with older versions of FinFocus Core that expect a non-error response even when pricing is missing, hence the $0 + BillingDetail fallback.
- **Contract Enforcement**: It is assumed that `GetActualCost` returning empty results is the correct signal for the Core to perform a multi-plugin fallback.
- **Port Priority**: CLI flags are assumed to always be the highest priority for configuration, overriding both standard and legacy environment variables.

### Functional Requirements

- **FR-001**: Plugin MUST parse the `--port` CLI flag using `flag.Parse()`.
- **FR-002**: Plugin MUST prioritize `--port` flag > `FINFOCUS_PLUGIN_PORT` > `PORT` > Ephemeral port.
- **FR-003**: System MUST define a custom `PricingUnavailableError` to distinguish missing data from system errors.
- **FR-004**: `GetActualCost` MUST catch `PricingUnavailableError` and return a response with an empty `Results` list.
- **FR-005**: `GetProjectedCost` MUST catch `PricingUnavailableError` and return $0 with a descriptive `BillingDetail` message.
- **FR-006**: The actual cost routing logic MUST be updated to include S3 (`estimateS3`), Lambda (`estimateLambda`), RDS (`estimateRDS`), and DynamoDB (`estimateDynamoDB`) using their real estimators.
- **FR-007**: Estimators MUST return `PricingUnavailableError` instead of $0 when the primary pricing lookup fails.

### Key Entities *(include if feature involves data)*

- **PricingUnavailableError**: An internal error type containing the service name, SKU, and a human-readable explanation.
- **ActualCostResult**: The gRPC response entity which MUST be empty if pricing is unknown.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Plugin listens on the port specified via `--port` in 100% of test cases.
- **SC-002**: Core fallback is triggered (empty results returned) for at least 5 common "unknown" instance types in test suites.
- **SC-003**: Actual cost estimation for S3 and Lambda resources returns non-zero values matching the projected cost logic.
- **SC-004**: All unit tests pass with 0 regressions in existing pricing logic.
