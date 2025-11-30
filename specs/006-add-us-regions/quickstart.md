# Quickstart: Add support for additional US regions

## Prerequisites

- Go 1.25.4 or later
- Make sure `make lint` and `make test` pass on main branch

## Building New Regions

### Build us-west-1 region
```bash
make build-region REGION=us-west-1
```

### Build us-gov-west-1 region
```bash
make build-region REGION=us-gov-west-1
```

### Build us-gov-east-1 region
```bash
make build-region REGION=us-gov-east-1
```

## Testing New Regions

### Run all tests
```bash
make test
```

### Test specific region integration
```bash
# Test us-west-1
go test ./internal/plugin -run TestIntegration_usw1 -v

# Test us-gov-west-1
go test ./internal/plugin -run TestIntegration_govw1 -v

# Test us-gov-east-1
go test ./internal/plugin -run TestIntegration_gove1 -v
```

## Verification

1. Check that binaries are created in `dist/` directory
2. Verify pricing data is embedded correctly
3. Test gRPC endpoints with grpcurl
4. Confirm GovCloud pricing differs from commercial regions

## Troubleshooting

- If build fails, check that pricing data generation completed successfully
- Ensure build tags are correctly defined in Go files
- Verify .goreleaser.yaml includes new region configurations