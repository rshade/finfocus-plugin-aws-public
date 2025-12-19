# Quickstart: Testing Batch Recommendations

## Prerequisites
- Built plugin binary: `make build` (or specific region)
- `grpcurl` installed (for manual testing) or unit tests.

## Using Unit Tests (Recommended)

Run the new table-driven tests added for this feature:

```bash
go test -v ./internal/plugin -run TestGetRecommendations_Batch
```

## Manual Verification (Mock Client)

Since `grpcurl` is hard to use with complex proto messages without reflection set up perfectly, use the Go test suite.

### Scenario 1: Batch Request
1. Construct a `GetRecommendationsRequest` with 2 resources in `TargetResources`.
2. Leave `Filter` empty.
3. Call `GetRecommendations`.
4. Expect 2 results.

### Scenario 2: Filtered Batch
1. Construct a `GetRecommendationsRequest` with 2 resources (one matches filter, one doesn't).
2. Set `Filter.Region` to match only the first resource.
3. Call `GetRecommendations`.
4. Expect 1 result.
