# Quickstart: Testing NAT Gateway Cost Estimation

## Local Development

1. **Generate Pricing Data**:
   ```bash
   go run ./tools/generate-pricing --regions us-east-1 --service AmazonVPC
   ```

2. **Build for us-east-1**:
   ```bash
   make build-region REGION=us-east-1
   ```

3. **Run the Plugin**:
   ```bash
   ./pulumicost-plugin-aws-public-us-east-1
   ```

## Test Requests

### Scenario 1: Basic Hourly Cost (No data)
```json
{
  "resource": {
    "provider": "aws",
    "resource_type": "nat_gateway",
    "region": "us-east-1"
  }
}
```

### Scenario 2: Hourly + Data Processing
```json
{
  "resource": {
    "provider": "aws",
    "resource_type": "natgw",
    "region": "us-east-1",
    "tags": {
      "data_processed_gb": "100"
    }
  }
}
```

## Troubleshooting
- If `Supports()` returns `false`, check `normalizedType` in `internal/plugin/supports.go`.
- If cost is `$0`, check if `vpc_us-east-1.json` was correctly generated and embedded.
