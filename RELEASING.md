# Releasing pulumicost-plugin-aws-public

This document describes the release process for the PulumiCost AWS Public Plugin.

## Overview

Releases are automated using:

- **release-please** - Manages versions and changelogs
- **GoReleaser** - Builds multi-region binaries
- **GitHub Actions** - Orchestrates the release workflow

## Release Process

### 1. Develop and Merge Features

All development happens on feature branches following conventional commit messages:

- `feat: add new feature` - Minor version bump
- `fix: resolve bug` - Patch version bump
- `feat!: breaking change` or `BREAKING CHANGE:` in footer - Major version bump
- `docs:`, `chore:`, `test:` - No version bump (excluded from changelog)

### 2. Release-Please Creates Release PR

When commits are pushed to `main`, the release-please workflow automatically:

1. Analyzes commits since last release
2. Determines version bump (major/minor/patch)
3. Creates or updates a release PR with:
   - Updated version in relevant files
   - Generated CHANGELOG.md with conventional commit sections
   - Title: `chore: release vX.Y.Z`

**Example Release PR**:

```yaml
Title: chore: release v1.2.0

Changes:
- Updated version to 1.2.0
- Updated CHANGELOG.md with new features

Commits included:
- feat: add support for io2 EBS volumes
- fix: handle missing region tag gracefully
- docs: improve README examples
```

### 3. Review and Merge Release PR

**Pre-Release Checklist**:

- [ ] Review CHANGELOG.md for accuracy
- [ ] Verify version bump is correct (major/minor/patch)
- [ ] Ensure all CI checks pass (tests, lint, build)
- [ ] Confirm no breaking changes without major version bump
- [ ] Review commits included in the release

**To Merge**:

```bash
# Via GitHub UI: Merge the release PR
# Or via CLI:
gh pr merge <PR_NUMBER> --squash
```

### 4. Automated Release Workflow

When the release PR is merged, release-please automatically:

1. **Creates Git Tag**: `vX.Y.Z`
2. **Triggers GoReleaser Workflow**: `.github/workflows/release.yml`

GoReleaser then:

1. Generates pricing data for all regions (dummy data currently)
2. Builds binaries for each region:
   - `pulumicost-plugin-aws-public-us-east-1` (Linux, Darwin, Windows × amd64, arm64)
   - `pulumicost-plugin-aws-public-us-west-2` (Linux, Darwin, Windows × amd64, arm64)
   - `pulumicost-plugin-aws-public-eu-west-1` (Linux, Darwin, Windows × amd64, arm64)
3. Creates GitHub Release with:
   - Changelog from release-please
   - Binary archives (tar.gz for Linux/Darwin, zip for Windows)
   - Checksums file

**Total Binaries**: 18 (3 regions × 3 OS × 2 architectures)

### 5. Verify Release

After the release workflow completes:

```bash
# Check GitHub Releases page
gh release list

# Download and test a binary
gh release download vX.Y.Z --pattern '*us-east-1*linux_amd64*'
tar -xzf pulumicost-plugin-aws-public_vX.Y.Z_Linux_x86_64.tar.gz
./pulumicost-plugin-aws-public-us-east-1
# Should output: PORT=<port>
```

**Post-Release Checklist**:

- [ ] Verify GitHub Release exists with all 18 binaries
- [ ] Download and smoke test at least one binary per OS
- [ ] Confirm checksums.txt is present
- [ ] Verify CHANGELOG.md is updated on main branch
- [ ] Test installation instructions in README.md

## Manual Release (Emergency)

If automated release fails, you can create a manual release:

### Option 1: Retry GitHub Actions Workflow

```bash
# Re-run the failed workflow from GitHub UI
# Or via CLI:
gh workflow run release.yml --ref vX.Y.Z
```

### Option 2: Local GoReleaser Release

```bash
# Ensure you're on the correct tag
git checkout vX.Y.Z

# Generate pricing data
go run ./tools/generate-pricing --regions us-east-1,us-west-2,eu-west-1 --out-dir ./internal/pricing/data

# Run GoReleaser (requires GITHUB_TOKEN)
export GITHUB_TOKEN=<your-token>
goreleaser release --clean

# This will create the GitHub Release with binaries
```

### Option 3: Manual Tag Creation

If release-please fails to create a tag:

```bash
# Manually create and push tag
git tag -a vX.Y.Z -m "Release vX.Y.Z"
git push origin vX.Y.Z

# This triggers the release workflow
```

## Version Strategy

We follow [Semantic Versioning 2.0.0](https://semver.org/):

- **Major (X.0.0)**: Breaking changes (incompatible API changes)
- **Minor (0.X.0)**: New features (backwards-compatible functionality)
- **Patch (0.0.X)**: Bug fixes (backwards-compatible fixes)

**Examples**:

- `feat: add RDS cost estimation` → v1.1.0 (minor)
- `fix: correct EBS gp3 pricing` → v1.0.1 (patch)
- `feat!: change ResourceDescriptor format` → v2.0.0 (major)

## Pre-Release Versions

For testing releases before general availability:

```bash
# Create a pre-release tag manually
git tag -a v1.2.0-rc.1 -m "Release candidate 1 for v1.2.0"
git push origin v1.2.0-rc.1

# GoReleaser will mark it as pre-release in GitHub
```

## Hotfix Process

For critical bugs in production:

1. Create hotfix branch from tag: `git checkout -b hotfix/v1.2.1 v1.2.0`
2. Fix the bug with commit: `fix: resolve critical issue`
3. Open PR to main
4. Merge PR (release-please creates patch release)

## Configuration Files

### `.release-please-manifest.json`

Tracks current version:

```json
{
  ".": "1.2.0"
}
```

### `release-please-config.json`

Configures release-please behavior:

- Release type: `go`
- Changelog sections: Features, Bug Fixes, Documentation
- Hidden sections: Chore, Test commits

### `.goreleaser.yaml`

Configures binary builds:

- 3 build configurations (one per region)
- Build tags for region-specific pricing
- Archive formats: tar.gz (Linux/Darwin), zip (Windows)
- Before hook: Generate pricing data

## Troubleshooting

### Release-Please PR Not Created

**Cause**: No releasable commits since last version

**Solution**: Ensure commits use conventional format (`feat:`, `fix:`, etc.)

```bash
# Check commits since last release
git log v1.2.0..HEAD --oneline

# Must contain at least one feat/fix commit
```

### GoReleaser Build Fails

**Cause**: Missing pricing data or build errors

**Check**:

```bash
# Verify pricing data generation
go run ./tools/generate-pricing --regions us-east-1,us-west-2,eu-west-1 --out-dir ./internal/pricing/data

# Test local build
goreleaser build --snapshot --clean
```

### Binary Missing from Release

**Cause**: GoReleaser configuration issue

**Solution**:

```bash
# Validate .goreleaser.yaml
goreleaser check

# Test snapshot build locally
goreleaser build --snapshot --clean
ls -la dist/
```

### Release Workflow Permissions Error

**Cause**: Missing GITHUB_TOKEN permissions

**Solution**: Ensure workflow has `contents: write` permission in `.github/workflows/release.yml`

## Pricing Data

The `tools/generate-pricing` tool fetches real pricing data from the AWS Price List API:

- No AWS credentials required - uses public pricing endpoint
- Data is fetched during build via GoReleaser before hook
- Each region binary embeds its own pricing data

## Support

For issues with releases:

- Check [GitHub Actions workflows](../../actions)
- Review [release-please documentation](https://github.com/google-github-actions/release-please-action)
- See [GoReleaser documentation](https://goreleaser.com/intro/)
- Open issue in repository
