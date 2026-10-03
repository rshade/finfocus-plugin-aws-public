# Tooling Baseline

This document describes the standard tooling configuration for finfocus plugins. All plugin repositories should maintain identical tooling setup to ensure consistent development, testing, and release workflows across the organization.

## Files That Define the Baseline

The tooling baseline is defined by the following files in this repository:

| File | Purpose |
|------|---------|
| `mise.toml` | Tool versions (Go, linters, security scanners, build tools) |
| `.golangci.yml` | Go linting configuration |
| `.markdownlint.json` | Markdown linting rules |
| `.vale.ini` | Prose linting configuration (Google style) |
| `.vale/styles/config/vocabularies/Plugin/accept.txt` | Domain-specific vocabulary exceptions |
| `commitlint.config.js` | Commit message validation rules |
| `.github/workflows/test.yml` | Unit tests, linting, build verification |
| `.github/workflows/prose.yml` | Prose linting with reviewdog |
| `.github/workflows/commitlint.yml` | Commit message validation |
| `.github/workflows/docker-publish.yml` | Docker image publishing |
| `.github/workflows/release-please.yml` | Automated release creation |
| `.github/workflows/release.yml` | Release build and publishing |
| `Makefile` | Development and CI task automation |
| `.gitignore` | File exclusion rules (especially `.vale/styles/`) |

## Tool Versions

All plugin Repos must use the following tool versions (pinned in `mise.toml`):

```toml
[tools]
go = "1.27.1"
golangci-lint = "2.14.0"
actionlint = "1.7.12"
"go:golang.org/x/vuln/cmd/govulncheck" = "1.8.0"
"pipx:specify-cli" = "1.0.11"
node = "24"
goreleaser = "2.18.2"
vale = "3.20.0"
"aqua:reviewdog/reviewdog" = "0.21.0"
"npm:@commitlint/cli" = "20.5.2"
"npm:@commitlint/config-conventional" = "20.5.3"
"npm:markdownlint-cli2" = "0.23.3"
```

**Rationale:**
- **Go 1.27.1**: Matches core finfocus and finfocus-spec
- **golangci-lint 2.14.0**: Matches core repo; enforces consistent lint rules
- **govulncheck 1.8.0**: Security scanning for known vulnerabilities
- **goreleaser 2.18.2**: Multi-platform binary release automation
- **vale 3.20.0**: Prose linting with Google style guide
- **reviewdog 0.21.0**: GitHub PR comment integration for linting
- **commitlint 20.5.2**: Enforces conventional commit format
- **config-conventional 20.5.3**: Shared Conventional Commits rules for commitlint
- **markdownlint-cli2 0.23.3**: Markdown structure linting

## Adoption Steps for New Plugin Repos

To set up a new plugin repository with the baseline tooling:

1. **Copy tooling files from this repo:**
   - `mise.toml`
   - `.golangci.yml`
   - `.markdownlint.json`
   - `.vale.ini`
   - `.vale/styles/config/vocabularies/` (create directory structure)
   - `commitlint.config.js`

2. **Copy workflow files to `.github/workflows/`:**
   - `test.yml`
   - `prose.yml`
   - `commitlint.yml`
   - `docker-publish.yml` (if applicable)
   - `release-please.yml`
   - `release.yml`

3. **Add Makefile targets** (or copy the entire file):
   - `lint` - Run golangci-lint
   - `vuln` - Run govulncheck
   - `lint-prose-sync` - Download Vale styles
   - `lint-prose` - Lint markdown documentation
   - `lint-prose-ci` - Output for CI/reviewdog
   - `test` - Run tests

4. **Update `.gitignore`:**
   - Add `.vale/styles/` to exclude downloaded style packages

5. **Customize for plugin:**
   - Update `.vale/styles/config/vocabularies/Plugin/accept.txt` with domain-specific terms
   - Adjust `lint-prose` Makefile target to include correct documentation paths
   - Update `.markdownlint.json` if plugin-specific rules are needed

## Workflow Behavior

### CI Workflows (GitHub Actions)

| Workflow | Trigger | Purpose |
|----------|---------|---------|
| `test.yml` | PR, push to main | Unit tests, linting, build verification |
| `prose.yml` | PR, manual dispatch | Prose linting via reviewdog (PR comments) |
| `commitlint.yml` | PR opened/updated | Commit message validation |
| `release-please.yml` | Push to main | Automated release management |
| `release.yml` | Release created | Build and publish binaries |
| `docker-publish.yml` | Release created | Build and publish Docker images |

### Workflow Implementation Rules

**CRITICAL**: Workflow steps must call tasks that exist. When implementing GitHub Actions workflows:

1. **mise run can only execute tasks defined in mise.toml** - Makefiles are separate task runners
2. **make commands invoke Makefile targets** - Always verify the target exists in Makefile before adding to workflow
3. **Workflow step pattern**:
   - If task is defined in `mise.toml` `[tasks]` section: use `mise run <task-name>`
   - If task is a Makefile target: use `make <target-name>`
   - Do NOT mix — `mise run <make-target>` will fail with "task not found"

**Example workflow pattern** (from prose.yml in this repo):

```yaml
# Step 1: Install tools via mise-action
- uses: jdx/mise-action@v2
  with:
    install_args: vale aqua:reviewdog/reviewdog

# Step 2: Run Makefile targets (NOT mise tasks)
- name: Fetch Vale styles
  run: make lint-prose-sync

- name: Vale -> reviewdog
  run: |
    make lint-prose-ci | reviewdog ...
```

### Local Development

```bash
# Install all tools
mise install

# Run tests
make test

# Run linting
make lint
make vuln
make lint-prose

# Validate commits before pushing
npx commitlint --from HEAD~1
```

## Security Scanning

Each plugin repository runs automated security checks:

1. **Dependency vulnerabilities** via `govulncheck` (see note below)
2. **Code linting** via `golangci-lint` (includes gosec for security issues)
3. **SBOM generation** (if build system includes it)

### govulncheck in CI

The `test.yml` workflow includes a non-blocking govulncheck step. It is marked `continue-on-error: true` because all plugins currently depend on `google.golang.org/grpc` v1.84.0, which has a known vulnerability (GO-2026-6443) fixed only in an unreleased development version. This is an upstream issue affecting every gRPC-based plugin and does not require action by plugin maintainers. Once `google.golang.org/grpc` v1.85.0 is released, the step can be changed to blocking.

## Documentation Standards

### Prose Style Guide

All markdown documentation follows Google's style guide (enforced via Vale):

- Sentence case for body text
- Title Case for headings (exception to Google guide)
- Spaced em dashes (—) instead of dashes
- No first-person language in formal docs
- Code examples must be tested

### Required Documentation

Each plugin should include:

- `README.md` - Overview, features, installation
- `CONTRIBUTING.md` - Development setup, PR process
- `CHANGELOG.md` - Release notes in Keep a Changelog format
- `docs/` directory (if needed) - Extended documentation

## Dependency Management

### Production Dependencies

- Use `go mod` for dependency management
- Pin semantic versions in `go.mod`
- Run `go mod tidy` before commits

### Development Tools

- All tools are pinned in `mise.toml`
- Never use global `go install`; use `mise` instead
- Run `mise install` in new checkout

## Plugin Distribution and Asset Naming

### Asset Naming Rule for Plugin Installers

Plugin releases must use consistent asset names so the finfocus installer can locate the correct binary. All asset names must follow this pattern:

```text
<name>_<version>_<os>_<arch><region><extension>
```

**Required components:**

- `<name>`: Plugin binary name (e.g., `finfocus-plugin-aws-public`)
- `<version>`: Release version (e.g., `v0.1.9`)
- `<os>`: Operating system name (Linux, Darwin, Windows)
  - These spellings are defined by the installer's asset matching logic
  - See `internal/registry/github.go` for exact valid spellings
- `<arch>`: Architecture (x86_64, arm64, etc.)
  - Must match the installer's arch naming conventions
- `<region>`: (optional) AWS region suffix for multi-region binaries (e.g., `-us-east-1`)
- `<extension>`: Archive format extension (e.g., `.tar.gz`, `.zip`)

**Example valid names:**
- `finfocus-plugin-aws-public_v0.1.9_Linux_x86_64.tar.gz`
- `finfocus-plugin-aws-public_v0.1.9_Darwin_arm64.tar.gz`
- `finfocus-plugin-aws-public_v0.1.9_Windows_x86_64.zip`
- `finfocus-plugin-aws-public-us-east-1_v0.1.9_Linux_x86_64.tar.gz` (region-specific)

### goreleaser Configuration

When configuring `.goreleaser.yaml` for a plugin:

- **Archive extensions come from the `formats` section**, not from `name_template`
  - Use `formats: [tar.gz, zip]` to control which archives are created
  - The `name_template` field should NOT include extension logic
  - Example: `formats: [tar.gz]` creates `.tar.gz` files; `formats: [zip]` creates `.zip` files
- OS and architecture spellings must match those expected by the installer
- Use explicit `name_template` for consistent multi-region asset naming
- Region-specific binaries should include region in the asset name (e.g., `-us-east-1`)

## Continuous Improvement

When updates are needed:

1. **Tool version bumps**: Update `mise.toml` in one repo, then propagate
2. **Linting rules changes**: Update `.golangci.yml` consistently
3. **Workflow improvements**: Update in this repo first, then copy to others
4. **Documentation format changes**: Update `.vale.ini` and vale vocabulary
5. **Release automation**: Update `.goreleaser.yaml` consistently
6. **Asset naming**: Ensure all asset names follow the `<name>_<version>_<os>_<arch>` pattern

All changes must be synchronized across all plugin repositories to maintain consistency.

## Reference

- [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) - Changelog format
- [Google Developer Style Guide](https://developers.google.com/style) - Prose style
- [Conventional Commits](https://www.conventionalcommits.org/) - Commit message format
- [goreleaser](https://goreleaser.com/) - Release automation
- [Vale](https://vale.sh/) - Prose linting
- [commitlint](https://commitlint.js.org/) - Commit validation
