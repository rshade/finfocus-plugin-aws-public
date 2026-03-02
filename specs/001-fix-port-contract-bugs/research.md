# Research: Port Binding & Contract Compliance

## Decisions

### Decision 1: Use `flag.Parse()` and `pluginsdk.ParsePortFlag()`
- **Rationale**: The `pluginsdk` (v0.5.6) registers the `--port` flag at init time. However, Go's `flag` package requires an explicit `flag.Parse()` call to populate the values. `pluginsdk.ParsePortFlag()` provides a unified way to retrieve this value from the parsed flags.
- **Alternatives considered**: 
  - Manual flag parsing: Rejected because it duplicates SDK logic.
  - Env vars only: Rejected because it breaks standard CLI expectations and doesn't satisfy the requirement.

### Decision 2: Custom `PricingUnavailableError` Type
- **Rationale**: Returning `$0` as an error value is ambiguous (is it free or missing?). A custom error type allows `GetActualCost` and `GetProjectedCost` to distinguish "pricing not found" from validation or system errors and apply different fallback logic (empty results vs $0 + explanation).
- **Alternatives considered**: 
  - Standard `errors.New`: Too brittle for type-assertion.
  - `grpc.Status` errors: Harder to handle internally for different fallback paths.

### Decision 3: Unify Actual Cost Routing
- **Rationale**: Several services (S3, Lambda, RDS, DynamoDB) were previously using `estimateStub` in `GetActualCost` mode, which always returned $0. Using the real estimators ensures that if pricing data is available, it is used.
- **Alternatives considered**:
  - Keep `estimateStub`: Rejected because it causes underestimation even when data is present.

## Best Practices

- **Port Priority**: CLI Flags > Environment Variables > Defaults.
- **Fail-Open (Contract)**: When a data source "doesn't know", it should say "I don't know" (empty response) rather than providing a false "free" ($0) value. This allows the consumer to seek other sources.

## Unresolved Items (NEEDS CLARIFICATION)
- NONE. The technical path is well-defined by the current implementation and SDK version.
