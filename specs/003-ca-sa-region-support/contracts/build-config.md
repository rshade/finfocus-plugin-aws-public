# Contract: Build Configuration

**Feature**: `003-ca-sa-region-support`

## Build Tags
The build system uses Go build tags to select the correct pricing data for the target region.

| Region | Tag | Description |
| :--- | :--- | :--- |
| Canada Central | `region_cac1` | Selects `embed_cac1.go` |
| South America (São Paulo) | `region_sae1` | Selects `embed_sae1.go` |

## Build Artifacts
The build process (via GoReleaser) produces the following binaries:

| ID | Binary Name | Tags Used |
| :--- | :--- | :--- |
| `ca-central-1` | `finfocus-plugin-aws-public-ca-central-1` | `region_cac1` |
| `sa-east-1` | `finfocus-plugin-aws-public-sa-east-1` | `region_sae1` |

## Usage Contract
The plugin does not accept runtime flags for region selection. The region is fixed at compile time.

- **Input**: No CLI arguments required.
- **Output**: Standard output prints `PORT=<port>` on startup.
- **Behavior**: Returns pricing *only* for the compiled region. Requests for other regions return `ERROR_CODE_UNSUPPORTED_REGION`.
