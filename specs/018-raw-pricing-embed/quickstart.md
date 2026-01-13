# Quickstart: Embed Raw AWS Pricing JSON Per Service

**Feature**: 018-raw-pricing-embed
**Date**: 2025-12-20

## Prerequisites

- Go 1.25+
- `make` available
- Internet access (for fetching AWS pricing data)

## Development Setup

### 1. Generate Pricing Data (Single Region)

For development, generate pricing for us-east-1 only:

```bash
# Clean old combined files
rm -f data/aws_pricing_*.json

# Generate per-service pricing files for us-east-1
go run ./tools/generate-pricing --regions us-east-1 --out-dir ./data
```

This creates 7 files:

```text
data/
├── ec2_us-east-1.json      (~120MB)
├── s3_us-east-1.json       (~1MB)
├── rds_us-east-1.json      (~15MB)
├── eks_us-east-1.json      (~3MB)
├── lambda_us-east-1.json   (~2MB)
├── dynamodb_us-east-1.json (~500KB)
└── elb_us-east-1.json      (~500KB)
```

### 2. Build with Region Tag

```bash
# Build us-east-1 binary
go build -tags=region_use1 -o finfocus-plugin-aws-public-us-east-1 \
  ./cmd/finfocus-plugin-aws-public
```

### 3. Run Tests

```bash
# Unit tests with region tag
go test -tags=region_use1 ./internal/pricing/...

# All tests via make
make test
```

### 4. Verify Pricing Data

```bash
# Check embedded data sizes
go test -tags=region_use1 -run TestEmbeddedPricing ./internal/pricing/...

# Check specific service metadata
jq '.offerCode, .version, .publicationDate' data/ec2_us-east-1.json
```

## Common Tasks

### Generate Pricing for All Regions

```bash
go run ./tools/generate-pricing \
  --regions us-east-1,us-west-2,eu-west-1,ca-central-1,sa-east-1,ap-southeast-1,ap-southeast-2,ap-northeast-1,ap-south-1 \
  --out-dir ./data
```

### Build All Region Binaries

```bash
goreleaser build --snapshot --clean
```

### Inspect Service Pricing File

```bash
# Count products
jq '.products | length' data/ec2_us-east-1.json

# Check offer code
jq '.offerCode' data/elb_us-east-1.json

# List product families
jq '[.products[].productFamily] | unique' data/rds_us-east-1.json
```

## Troubleshooting

### "file not found" errors during build

Ensure pricing data is generated before building:

```bash
ls -la data/*.json
# If empty, run:
go run ./tools/generate-pricing --regions us-east-1 --out-dir ./data
```

### Tests fail with size threshold errors

The pricing data may be corrupted or incomplete. Regenerate:

```bash
rm -f data/*.json
go run ./tools/generate-pricing --regions us-east-1 --out-dir ./data
```

### Binary returns $0 prices

Verify the binary was built with region tags:

```bash
# Wrong (no pricing data):
go build ./cmd/finfocus-plugin-aws-public

# Correct (with pricing data):
go build -tags=region_use1 ./cmd/finfocus-plugin-aws-public
```

## File Size Reference

Expected sizes for us-east-1 (December 2025):

| Service | File Size | Test Threshold |
|---------|-----------|----------------|
| EC2 | ~120MB | 100MB |
| RDS | ~15MB | 10MB |
| EKS | ~3MB | 2MB |
| Lambda | ~2MB | 1MB |
| S3 | ~1MB | 500KB |
| DynamoDB | ~500KB | 400KB |
| ELB | ~500KB | 400KB |
