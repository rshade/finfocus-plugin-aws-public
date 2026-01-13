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
- `./finfocus-plugin-aws-public-ca-central-1`
- `./finfocus-plugin-aws-public-sa-east-1`

## Running the Plugin

Start the binary directly (usually handled by FinFocus core):

```bash
./finfocus-plugin-aws-public-ca-central-1
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
[finfocus-plugin-aws-public] {"level":"info","message":"serving aws-public pricing for region ca-central-1"}
```