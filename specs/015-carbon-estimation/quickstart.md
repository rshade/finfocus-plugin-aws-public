# Quickstart: Carbon Emission Estimation

**Feature**: 015-carbon-estimation
**Date**: 2025-12-19

## Prerequisites

1. Go 1.25+
2. pulumicost-spec v0.4.10+ (update go.mod)
3. CCF instance data CSV downloaded

## Quick Setup

### 1. Update Dependencies

```bash
go get github.com/rshade/pulumicost-spec@v0.4.10
go mod tidy
```

### 2. Download CCF Data

```bash
# Create data directory
mkdir -p data

# Download CCF instance specifications
curl -L -o data/ccf_instance_specs.csv \
  "https://raw.githubusercontent.com/cloud-carbon-footprint/cloud-carbon-coefficients/main/data/aws-instances.csv"
```

### 3. Verify Build

```bash
# Build with region tag to verify embedded data works
go build -tags region_use1 ./cmd/pulumicost-plugin-aws-public

# Run tests
make test
```

## Implementation Order

### Step 1: Create Carbon Package

```bash
mkdir -p internal/carbon
```

Create files in order:

1. `internal/carbon/constants.go` - PUE, default utilization, grid factors
2. `internal/carbon/instance_specs.go` - CSV parsing and lookup
3. `internal/carbon/estimator.go` - Carbon calculation logic
4. `internal/carbon/estimator_test.go` - Unit tests

### Step 2: Update Plugin

Modify in order:

1. `internal/plugin/plugin.go` - Add CarbonEstimator to AWSPublicPlugin struct
2. `internal/plugin/projected.go` - Call carbon estimator in estimateEC2
3. `internal/plugin/supports.go` - Return supported_metrics for EC2

### Step 3: Add Tests

1. Add carbon metric assertions to `internal/plugin/projected_test.go`
2. Add supports_metrics assertions to `internal/plugin/supports_test.go`

## Key Code Snippets

### Grid Emission Factors

```go
// internal/carbon/constants.go
package carbon

const (
    AWS_PUE            = 1.135
    DefaultUtilization = 0.50
    HoursPerMonth      = 730.0
)

var GridEmissionFactors = map[string]float64{
    "us-east-1":      0.000379,
    "us-west-2":      0.000322,
    "eu-north-1":     0.0000088,
    // ... other regions
    "_default":       0.00039278,
}
```

### Instance Spec Lookup

```go
// internal/carbon/instance_specs.go
package carbon

import (
    _ "embed"
    "encoding/csv"
    "strings"
    "sync"
)

//go:embed data/ccf_instance_specs.csv
var instanceSpecsCSV []byte

type InstanceSpec struct {
    InstanceType string
    VCPUCount    int
    MinWatts     float64
    MaxWatts     float64
}

var (
    instanceSpecs map[string]InstanceSpec
    specsOnce     sync.Once
)

func GetInstanceSpec(instanceType string) (InstanceSpec, bool) {
    specsOnce.Do(parseInstanceSpecs)
    spec, ok := instanceSpecs[instanceType]
    return spec, ok
}
```

### Carbon Estimator

```go
// internal/carbon/estimator.go
package carbon

type CarbonEstimator interface {
    EstimateCarbonGrams(instanceType, region string, utilization, hours float64) (float64, bool)
}

type Estimator struct{}

func (e *Estimator) EstimateCarbonGrams(instanceType, region string, utilization, hours float64) (float64, bool) {
    spec, ok := GetInstanceSpec(instanceType)
    if !ok {
        return 0, false
    }

    gridFactor := GridEmissionFactors[region]
    if gridFactor == 0 {
        gridFactor = GridEmissionFactors["_default"]
    }

    avgWatts := spec.MinWatts + (utilization * (spec.MaxWatts - spec.MinWatts))
    energyKWh := (avgWatts * float64(spec.VCPUCount) * hours) / 1000.0
    energyPUE := energyKWh * AWS_PUE
    carbonGrams := energyPUE * gridFactor * 1_000_000

    return carbonGrams, true
}
```

### Plugin Integration

```go
// internal/plugin/projected.go (modification)
func (p *AWSPublicPlugin) estimateEC2(traceID string, resource *pbc.ResourceDescriptor) (*pbc.GetProjectedCostResponse, error) {
    // ... existing financial calculation ...

    resp := &pbc.GetProjectedCostResponse{
        CostPerMonth:  costPerMonth,
        UnitPrice:     hourlyRate,
        Currency:      "USD",
        BillingDetail: billingDetail,
    }

    // Add carbon estimation
    utilization := getUtilization(p.currentRequest, resource)
    carbonGrams, ok := p.carbonEstimator.EstimateCarbonGrams(
        instanceType, resource.Region, utilization, hoursPerMonth,
    )
    if ok {
        resp.ImpactMetrics = []*pbc.ImpactMetric{
            {
                Kind:  pbc.MetricKind_METRIC_KIND_CARBON_FOOTPRINT,
                Value: carbonGrams,
                Unit:  "gCO2e",
            },
        }
    }

    return resp, nil
}
```

## Validation

### Unit Test

```go
func TestCarbonEstimation_T3Micro_USEast1(t *testing.T) {
    e := &carbon.Estimator{}

    carbonGrams, ok := e.EstimateCarbonGrams("t3.micro", "us-east-1", 0.5, 730)

    assert.True(t, ok)
    assert.InDelta(t, 500, carbonGrams, 200) // Approximate expected range
}
```

### Integration Test

```bash
# Start plugin
./pulumicost-plugin-aws-public-us-east-1 &
PORT=$(head -1) # Capture PORT=XXXXX

# Test with grpcurl
grpcurl -plaintext -d '{
  "resource": {
    "provider": "aws",
    "resource_type": "aws:ec2/instance:Instance",
    "sku": "t3.micro",
    "region": "us-east-1"
  }
}' localhost:$PORT pulumicost.v1.CostSourceService/GetProjectedCost

# Verify response contains impact_metrics with METRIC_KIND_CARBON_FOOTPRINT
```

## Attribution

Add to NOTICE or README.md:

```text
Cloud Carbon Footprint
https://www.cloudcarbonfootprint.org/
Copyright 2021 Thoughtworks, Inc.
Licensed under the Apache License, Version 2.0
```
