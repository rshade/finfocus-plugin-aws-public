# Data Model: Port Binding & Contract Violation Fixes

## Key Entities

### PricingUnavailableError
Represents a situation where the plugin cannot calculate a cost because the required pricing data is missing from its embedded database.

- **Service**: `string` (e.g., "EC2", "S3")
- **SKU**: `string` (The unique identifier used for lookup)
- **BillingDetail**: `string` (Human-readable explanation for the user)

## State Transitions & Logic

### GetProjectedCost Logic
1. Call Estimator.
2. If Estimator returns `PricingUnavailableError`:
   - Return response with `CostPerMonth: 0` and `UnitPrice: 0`.
   - Set `BillingDetail` to the error's message.
   - Status: OK.

### GetActualCost Logic
1. Call `getProjectedForResource`.
2. If `getProjectedForResource` returns `PricingUnavailableError`:
   - Return response with `Results: []` (empty list).
   - Status: OK.
3. If any other error occurs:
   - Propagate as gRPC error.

### Estimator Behavior
- If primary lookup (e.g., `EC2OnDemandPricePerHour`) returns `found=false`:
  - Return `nil, &PricingUnavailableError{...}`.
