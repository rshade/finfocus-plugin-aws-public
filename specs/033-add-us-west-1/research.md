# Research Report: Add us-west-1 (N. California) Region Support

## 1. AWS Pricing API Data Volume

**Decision**: The `us-west-1` pricing data is expected to be compatible with existing binary size constraints (<250MB).

**Rationale**:

- `us-west-1` (N. California) is a major commercial region but typically has fewer instance types than `us-east-1` (N. Virginia) or `us-west-2` (Oregon).
- Existing `us-east-1` data fits within the limits.
- The pricing data structure is identical across regions.
- Fail-fast mechanism (as decided in specs) protects against unexpected data bloat during build.

**Alternatives Considered**:
- **Filtering Data**: Rejected by Constitution (Critical principle IV).
- **External Fetch**: Rejected for v1 architecture (requires embedded data).

## 2. Carbon Grid Emission Factor

**Decision**: Use `0.000322` metric tons CO2e/kWh for `us-west-1`.

**Rationale**:
- `internal/carbon/grid_factors.go` already defines `us-west-1` with this value.
- Both `us-west-1` (N. California) and `us-west-2` (Oregon) are part of the Western Electricity Coordinating Council (WECC) grid interconnection, which spans the western United States. The WECC regional average is used rather than state-specific factors, as AWS data centers draw from the broader interconnected grid.
- Source: Cloud Carbon Footprint methodology (already cited in code).

## 3. Port Allocation

**Decision**: Use port **8010** for `us-west-1`.

**Rationale**:
- Existing ports: `8001` (us-east-1) through `8009` (sa-east-1).
- `8010` is the next sequential integer.
- Dockerfile `EXPOSE` and `healthcheck.sh` arrays can be cleanly extended.
- No conflicts found in `docker/entrypoint.sh` or `Dockerfile`.
