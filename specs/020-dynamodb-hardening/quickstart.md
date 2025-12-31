# Quickstart: DynamoDB Hardening Bundle

**Feature**: 020-dynamodb-hardening
**Date**: 2025-12-31

## Prerequisites

- Go 1.25+ installed
- Repository cloned and on branch `020-dynamodb-hardening`
- Carbon data generated (`make generate-carbon-data`)

## Implementation Steps

### Step 1: Add Validation Helpers

Add two helper methods to `internal/plugin/projected.go`:

1. `validateNonNegativeInt64(traceID, tagName, value string) int64`
2. `validateNonNegativeFloat64(traceID, tagName, value string) float64`

Both should:
- Parse the value
- Log warning if parse fails or value < 0
- Return 0 on any error, valid value otherwise

### Step 2: Update estimateDynamoDB()

Modify `internal/plugin/projected.go:estimateDynamoDB()`:

1. **Replace storage parsing** (line ~488):

   ```go
   // Before
   if v, err := strconv.ParseFloat(s, 64); err == nil && v > 0 {

   // After
   storageGB = p.validateNonNegativeFloat64(traceID, "storage_gb", s)
   ```

2. **Replace capacity parsing** (provisioned mode, lines ~514-518):

   ```go
   // Before
   readUnits, _ = strconv.ParseInt(s, 10, 64)

   // After
   readUnits = p.validateNonNegativeInt64(traceID, "read_capacity_units", s)
   ```

3. **Check pricing lookup results** (lines ~522-523, ~567-568):
   - Capture the `bool` return value
   - Log `Warn()` if false
   - Track unavailable components in a slice
   - Append to billing_detail if any unavailable

### Step 3: Add Unit Tests

Add to `internal/plugin/projected_test.go`:

1. Test negative value validation (table-driven)
2. Test parse error handling
3. Test warning log emission (capture log output)

### Step 4: Add Pricing Extraction Tests

Add to `internal/pricing/client_test.go`:

1. Test all 5 DynamoDB pricing methods return non-zero values
2. Requires `//go:build region_use1` tag

### Step 5: Add Integration Tests

Create `internal/plugin/integration_dynamodb_test.go`:

1. Build binary with `region_use1` tag
2. Test provisioned mode via gRPC
3. Test on-demand mode via gRPC
4. Verify non-zero costs returned

## Verification Commands

```bash
# Run unit tests
go test -tags=region_use1 ./internal/plugin/... ./internal/pricing/...

# Run integration tests
go test -tags=integration,region_use1 ./internal/plugin/... -run DynamoDB

# Run all tests
make test

# Run linting
make lint
```

## Success Criteria Verification

- [ ] SC-001: Request estimate without pricing data → warning logs appear
- [ ] SC-002: Request estimate with negative tags → validation warnings appear
- [ ] SC-003: All 5 pricing methods return non-zero for us-east-1
- [ ] SC-004: Integration tests pass for both modes
- [ ] SC-005: `make test` passes
- [ ] SC-006: `make lint` passes
- [ ] SC-007: `make test` with new tests passes
