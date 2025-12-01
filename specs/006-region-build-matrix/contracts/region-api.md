# Quickstart: Adding AWS Regions

**Date**: 2025-11-30
**Feature**: specs/006-region-build-matrix/spec.md

## Adding a New AWS Region

### Step 1: Update regions.yaml
Add the new region to `internal/pricing/regions.yaml`:

```yaml
regions:
  - id: euw3  # Short code from region-tag.sh
    name: eu-west-3  # Full AWS region name
    tag: region_euw3  # Build tag
```

### Step 2: Generate Files
Run the generation scripts:

```bash
make generate-embeds
make generate-goreleaser
```

This creates:
- `internal/pricing/embed_euw3.go` (embed file)
- Updates `.goreleaser.yaml` (build config)

### Step 3: Verify
Run the verification script:

```bash
./scripts/verify-regions.sh
```

### Step 4: Test Build
Test the new region builds:

```bash
make build-region REGION=eu-west-3
```

### Step 5: Commit Changes
The generated files are committed to version control.

## Verification

- Embed file exists: `internal/pricing/embed_euw3.go`
- Build tag present: `//go:build region_euw3`
- GoReleaser config includes eu-west-3 build block
- Verification script passes

## Troubleshooting

- **Invalid region name**: Check AWS region naming conventions
- **Tag mismatch**: Ensure id matches region-tag.sh output
- **Build fails**: Verify pricing data exists for the region
- **Verification fails**: Check generated files match regions.yaml</content>
<parameter name="filePath">specs/006-region-build-matrix/quickstart.md