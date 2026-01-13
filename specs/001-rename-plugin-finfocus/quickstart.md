# Quickstart: Rename Plugin to FinFocus

This feature renames the plugin to `finfocus-plugin-aws-public`.

## Building the Plugin

The build commands have been updated to produce `finfocus-*` binaries.

```bash
# Build for us-east-1
make build-region REGION=us-east-1

# Verify the output
ls -lh finfocus-plugin-aws-public-us-east-1
```

## Running the Plugin

The binary name has changed.

```bash
# Start the plugin
./finfocus-plugin-aws-public-us-east-1
```

## Environment Variables

Use `FINFOCUS_` prefixed variables. `FINFOCUS_` variables are deprecated.

```bash
# Correct usage
export FINFOCUS_TEST_MODE=true
./finfocus-plugin-aws-public-us-east-1
```

## Verifying the Rename

Check the logs for the new component prefix:

```bash
./finfocus-plugin-aws-public-us-east-1 2>&1 | grep "\[finfocus-plugin-aws-public\]"
```
