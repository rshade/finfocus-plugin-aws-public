# Quickstart: Canada and South America Region Support

**Branch**: `003-ca-sa-region-support` | **Date**: 2025-11-20

## Prerequisites

- Go 1.25+
- GoReleaser (for building release binaries)
- grpcurl (for testing gRPC service)

## Development Workflow

### 1. Generate Pricing Data

```bash
# Generate dummy pricing data for all regions including new ones
go run ./tools/generate-pricing \
  --regions us-east-1,us-west-2,eu-west-1,ap-southeast-1,ap-southeast-2,ap-northeast-1,ap-south-1,ca-central-1,sa-east-1 \
  --out-dir ./internal/pricing/data \
  --dummy
```

### 2. Build Region-Specific Binary

```bash
# Build ca-central-1 binary
go build -tags region_cac1 -o pulumicost-plugin-aws-public-ca-central-1 ./cmd/pulumicost-plugin-aws-public

# Build sa-east-1 binary
go build -tags region_sae1 -o pulumicost-plugin-aws-public-sa-east-1 ./cmd/pulumicost-plugin-aws-public
```

### 3. Run Tests

```bash
# Run all tests
make test

# Run specific package tests
go test ./internal/pricing -v
go test ./internal/plugin -v
```

### 4. Lint Code

```bash
make lint
```

### 5. Build All Region Binaries

```bash
# Build snapshot release (all regions)
goreleaser build --snapshot --clean
```

## Testing the Plugin

### Start the Plugin

```bash
# Start ca-central-1 binary
./pulumicost-plugin-aws-public-ca-central-1
# Output: PORT=<port>
```

### Test with grpcurl

```bash
# Get plugin name
grpcurl -plaintext localhost:<port> pulumicost.v1.CostSourceService/Name

# Check support for ca-central-1 resource
grpcurl -plaintext -d '{
  "resource": {
    "provider": "aws",
    "resource_type": "ec2",
    "sku": "t3.micro",
    "region": "ca-central-1"
  }
}' localhost:<port> pulumicost.v1.CostSourceService/Supports

# Get projected cost
grpcurl -plaintext -d '{
  "resource": {
    "provider": "aws",
    "resource_type": "ec2",
    "sku": "t3.micro",
    "region": "ca-central-1"
  }
}' localhost:<port> pulumicost.v1.CostSourceService/GetProjectedCost
```

## Validation Checklist

- [ ] Both binaries build without errors
- [ ] Binary size < 20MB each
- [ ] All tests pass
- [ ] Lint passes
- [ ] Region mismatch returns ERROR_CODE_UNSUPPORTED_REGION
- [ ] Concurrent RPC calls handled correctly

## Common Issues

### Build fails with "missing embed file"

Ensure pricing data is generated first:
```bash
go run ./tools/generate-pricing --regions ca-central-1,sa-east-1 --out-dir ./internal/pricing/data --dummy
```

### Wrong region in binary

Check build tags:
```bash
# Must use correct tag
go build -tags region_cac1 ...  # NOT region_ca_central_1
```

### Region mismatch error

Ensure ResourceDescriptor.region matches the binary's embedded region:
- ca-central-1 binary only handles ca-central-1 resources
- sa-east-1 binary only handles sa-east-1 resources
