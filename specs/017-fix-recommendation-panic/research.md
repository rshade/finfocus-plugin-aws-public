# Research: Bug Fix and Documentation Sprint - Dec 2025

**Feature**: 017-fix-recommendation-panic

## Decision Log

### 1. Recommendation Impact Panic Fix (#123)
- **Problem**: `pctx.BatchStats.TotalSavings += rec.Impact.GetEstimatedSavings()` panics if `rec.Impact` is nil.
- **Decision**: Add an explicit nil check for `rec.Impact` before accessing `GetEstimatedSavings()`.
- **Rationale**: Prevent plugin crash during batch processing when a single recommendation is incomplete (e.g., missing pricing data).
- **Location**: `internal/plugin/recommendations.go` at line 124.

### 2. Carbon CSV Parsing Logging (#142)
- **Problem**: `parseInstanceSpecs` fails silently if CSV parsing fails.
- **Decision**: Add a `SetLogger` function to the `carbon` package to allow the plugin to inject its logger. Update `parseInstanceSpecs` to log errors using this logger.
- **Rationale**: Observability into data integrity issues without breaking the standalone nature of the package.
- **Location**: `internal/carbon/instance_specs.go`.

### 3. S3 Global Service Region Fallback (#113)
- **Problem**: S3 ARNs (e.g., `arn:aws:s3:::bucket`) have empty regions, which may cause validation failures or incorrect resource identification in regional binaries.
- **Decision**: Ensure that `Supports()` and `GetProjectedCost()` handle empty regions for S3 by defaulting to the plugin's own region. This aligns with `ValidateActualCostRequest` behavior.
- **Rationale**: S3 buckets are accessible from any region, but pricing is regional. Defaulting to the binary's region is the safest fallback.
- **Location**: `internal/plugin/supports.go` and `internal/plugin/validation.go`.

### 4. PORT Environment Variable Deprecation (#116)
- **Problem**: `PORT` is used as a legacy fallback but is being replaced by `PULUMICOST_PLUGIN_PORT`.
- **Decision**: Add a `logger.Warn()` message when the `PORT` environment variable is used, explicitly stating it is deprecated and will be removed in v0.x.x.
- **Rationale**: Give users clear notice before removing support for the legacy environment variable.
- **Location**: `cmd/pulumicost-plugin-aws-public/main.go`.

### 5. EC2 OS Mapping Documentation (#63)
- **Problem**: Mapping from tags like `platform` to OS identifiers for pricing is implicit.
- **Decision**: Document the mapping logic in `internal/pricing/client.go` or `internal/plugin/ec2_attrs.go`.
- **Rationale**: Clarify for developers and users how "Windows" vs "Linux" pricing is selected.
- **Internal Logic**: `windows` (case-insensitive) maps to "Windows", all others map to "Linux".

### 6. Documentation and Style Cleanup (#143, #145, #128, #60)
- **Docstrings**: Consolidate triple docstrings for `GetUtilization` in `internal/carbon/utilization.go`.
- **Trailing Newlines**: Fix missing newlines in `estimator.go`, `instance_specs.go`, and `utilization.go`.
- **Correlation IDs**: Add GoDoc explaining `ResourceId` and `Name` tag population in `internal/plugin/recommendations.go`.
- **Troubleshooting**: Create a dedicated `TROUBLESHOOTING.md` in the repository root.

## Technical Unknowns Resolved

- **S3 ARN Fallback**: Confirmed that `ValidateActualCostRequest` already implements fallback to `p.region` for global ARNs. `Supports()` needs similar logic to avoid rejecting S3 requests with empty regions.
- **Carbon Logging**: Decided to use a setter for the logger to avoid direct dependency from `internal/carbon` back to `internal/plugin`.
- **OS Mapping**: Confirmed normalization logic in `ec2_attrs.go`.

## Alternatives Considered

- **Logging in Carbon**: Considered `fmt.Fprintf(os.Stderr, ...)` but rejected in favor of structured `zerolog` via a setter to comply with the project's constitution.
- **PORT Removal**: Considered immediate removal but rejected to follow a standard deprecation cycle (warn first, remove later).
