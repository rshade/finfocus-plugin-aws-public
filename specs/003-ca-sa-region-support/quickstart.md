# Quickstart: Using Canada and South America Regions

**Feature**: `003-ca-sa-region-support`

## Building the Binaries

To build the new regional binaries locally:

```bash
# Build Canada Central
make build-region REGION=ca-central-1

# Build South America (São Paulo)
make build-region REGION=sa-east-1
```

This will produce:
- `./pulumicost-plugin-aws-public-ca-central-1`
- `./pulumicost-plugin-aws-public-sa-east-1`

## Running the Plugin

Start the binary directly (usually handled by PulumiCost core):

```bash
./pulumicost-plugin-aws-public-ca-central-1
```

**Expected Output:**
```text
PORT=12345
```
(The plugin is now listening on port 12345 for gRPC requests)

## Verifying Support

You can verify the binary supports the correct region using `grpcurl` (if available) or by checking logs.

**Log Verification:**
The plugin logs to stderr on startup. Look for:
```text
[pulumicost-plugin-aws-public] {"level":"info","message":"serving aws-public pricing for region ca-central-1"}
```