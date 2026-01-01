# Grid Emission Factor Updates

This document describes the process for updating regional grid emission factors
used in carbon footprint estimation.

## Overview

Grid emission factors represent the carbon intensity of electricity generation
in each AWS region, measured in **metric tons CO2e per kWh**. These values are
critical for accurate carbon estimation as they vary significantly by region:

| Region | Grid Factor | Description |
|--------|-------------|-------------|
| eu-north-1 | 0.0000088 | Sweden - Very clean (hydroelectric) |
| sa-east-1 | 0.0000617 | Brazil - Clean (hydroelectric) |
| us-east-1 | 0.000379 | Virginia - Mixed grid |
| ap-south-1 | 0.000708 | Mumbai - Coal-heavy grid |

Regional variation can cause **80× differences** in carbon footprint for the
same workload.

## Update Frequency

**Recommendation: Update annually in Q1**

Grid emission factors should be updated yearly to reflect:

- Changes in power generation mix (more renewables, coal retirement)
- Updated data from authoritative sources
- New AWS regions

### Calendar Reminder

Add these reminders to your calendar:

- **January 15**: Check for CCF repository updates
- **January 31**: Run update tool and verify results
- **February 15**: Release updated plugin binaries

## Data Sources

### Primary: Cloud Carbon Footprint (CCF)

The [Cloud Carbon Footprint](https://www.cloudcarbonfootprint.org/) project
maintains a comprehensive dataset of grid emission factors.

**Repository:** [cloud-carbon-coefficients](https://github.com/cloud-carbon-footprint/cloud-carbon-coefficients)

**Data file:** `data/grid-emissions-factors-aws.json`

### Secondary: EPA eGRID (US Regions Only)

For US regions, the EPA's [eGRID Power Profiler](https://www.epa.gov/egrid/power-profiler)
provides authoritative regional emission data.

## Update Process

### Step 1: Run the Update Tool

```bash
# Dry run to preview changes
go run ./tools/update-grid-factors --dry-run

# Apply updates
go run ./tools/update-grid-factors --output ./internal/carbon/grid_factors.go
```

### Step 2: Validate the Updates

The tool performs automatic validation:

- All factors must be in range 0.0 to 2.0 metric tons CO2e/kWh
- Known regions must have non-zero values
- Format must match expected Go code

Additionally, run the validation tests:

```bash
go test ./internal/carbon/... -run "TestGrid"
```

### Step 3: Review Changes

Review the diff to ensure:

1. No regions were accidentally removed
2. Values changed in the expected direction (generally decreasing as grids clean up)
3. New regions were added if AWS launched any

```bash
git diff internal/carbon/grid_factors.go
```

### Step 4: Test Carbon Estimates

Run full tests to ensure carbon estimates are still reasonable:

```bash
make test
```

### Step 5: Update Data Vintage

The `grid_factors.go` file includes a data vintage comment:

```go
// Data vintage: 2024 (update annually from CCF repository)
```

Update this to the current year.

### Step 6: Commit and Release

```bash
git add internal/carbon/grid_factors.go
git commit -m "chore: update grid emission factors for 2025"
```

## Validation Rules

Grid factors are validated against these constraints:

| Constraint | Value | Reason |
|------------|-------|--------|
| Minimum | 0.0 | No grid has negative emissions |
| Maximum | 2.0 | No grid exceeds 2 metric tons CO2e/kWh |
| Sweden (eu-north-1) | < 0.0001 | Historically very clean |
| India (ap-south-1) | > 0.0005 | Historically coal-heavy |

## Manual Updates

If the automatic tool fails, you can manually update `internal/carbon/grid_factors.go`:

```go
var GridEmissionFactors = map[string]float64{
    "us-east-1":      0.000379,    // Virginia (SERC)
    "us-east-2":      0.000411,    // Ohio (RFC)
    // ... other regions
}
```

Ensure you:

1. Use metric tons CO2e per kWh (not kg or grams)
2. Include a comment with the location name
3. Update the data vintage comment

## Troubleshooting

### Tool Cannot Fetch Data

If the CCF repository is unavailable:

1. Check if the URL has changed: `https://github.com/cloud-carbon-footprint/cloud-carbon-coefficients`
2. Use cached/fallback values (tool provides defaults)
3. Manually download and parse the data

### Validation Fails

If validation fails:

1. Check if a region has an unusually high value (possible data error)
2. Verify units are in metric tons (not kg or grams)
3. Check for typos in region names

### New AWS Region Not Listed

For new AWS regions:

1. Check CCF repository for the new region
2. If not available, use the closest geographic region as a proxy
3. Add a `// estimated` comment for transparency

## References

- [CCF Methodology](https://www.cloudcarbonfootprint.org/docs/methodology)
- [AWS Sustainability](https://aws.amazon.com/sustainability/)
- [EPA eGRID](https://www.epa.gov/egrid)
- [IEA Emission Factors](https://www.iea.org/data-and-statistics/data-tools/emissions-factors)
