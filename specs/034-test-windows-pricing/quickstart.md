# Quickstart: Windows vs Linux Pricing Integration Tests

## Running the Tests

To run the integration tests for pricing differentiation:

```bash
# Ensure you are in the project root
go test -v -tags=integration ./internal/plugin/... -run TestIntegration_PricingDifferentiation
```

## Test Matrix Coverage

| Platform | Tenancy | Expected Result |
|----------|---------|-----------------|
| Linux    | Shared  | Baseline        |
| Windows  | Shared  | > Linux Baseline|
| Linux    | Dedicated| > Linux Shared  |
| Windows  | Dedicated| Highest Price   |

## Verification

The tests verify:
1. `windows` platform returns higher price than `linux`.
2. `dedicated` tenancy returns higher price than `shared`.
3. `billing_detail` contains the correct platform and tenancy strings.
