# Troubleshooting Guide: FinFocus AWS Public Plugin

This guide provides solutions for common issues encountered when using the AWS Public Pricing plugin.

## Common Error Scenarios

### 1. "Region not supported by this binary"

**Error Code**: `ERROR_CODE_UNSUPPORTED_REGION`

**Cause**: You are trying to estimate costs for a resource in a region that does not match the compiled-in pricing data of the plugin binary.

**Solution**:

- Ensure you are using the correct regional binary (e.g., `finfocus-plugin-aws-public-us-east-1` for `us-east-1` resources).
- For global services like S3, the plugin will now automatically fallback to its own region if the request region is empty.

### 2. "EC2 instance type not found in pricing data"

**Cause**: The requested instance type (SKU) is not present in the embedded pricing data for the specified region.

**Solution**:

- Verify the instance type is a valid AWS instance type (e.g., `t3.micro`).
- Check if the instance type is available in the target region.
- If it's a very new instance type, the pricing data may need to be regenerated using `make generate-pricing`.

### 3. "failed to initialize pricing client"

**Cause**: The plugin failed to load the embedded pricing data at startup.

**Solution**:

- Ensure the binary was built correctly with region tags.
- If building from source, run `make generate-pricing` before `make build`.

### 4. "PORT environment variable is deprecated"

**Warning**: `PORT environment variable is deprecated and will be removed in v0.1.0.`

**Solution**:

- Update your deployment configuration to use `FINFOCUS_PLUGIN_PORT` instead of `PORT`.

## Debugging

### Enabling Test Mode

To get more detailed logs, including request/response details, set the `FINFOCUS_TEST_MODE` environment variable:

```bash
export FINFOCUS_TEST_MODE=true
```

### Checking Logs

All diagnostic logs are written to `stderr` in structured JSON format. You can use `jq` to format them for readability:

```bash
./finfocus-plugin-aws-public-us-east-1 2>&1 | jq .
```

### Verifying gRPC connectivity

Use `grpcurl` to test the plugin manually:

```bash
# Capture the PORT from stdout
PORT=12345 # replace with actual port from plugin startup output (e.g., PORT=12345)
grpcurl -plaintext localhost:$PORT finfocus.v1.CostSourceService/Name
```
