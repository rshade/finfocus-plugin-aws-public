# Quickstart: Carbon Estimation Implementation

**Feature**: 001-carbon-estimation  
**Date**: 2025-12-31  
**Phase**: Design & Contracts

This quickstart guide provides step-by-step instructions for implementing carbon estimation across AWS services.

---

## Prerequisites

### Required Knowledge
- Go 1.25.5+ development
- gRPC protocol and PulumiCost CostSourceService interface
- Cloud Carbon Footprint (CCF) methodology
- Embedded data patterns in Go (`//go:embed`)
- Thread-safe concurrent programming with `sync.Once`

### Development Setup

```bash
# Install dependencies
go mod download

# Run linting
make lint

# Run tests
make test

# Build region-specific binary
make build-region REGION=us-east-1
```

---

## Implementation Roadmap

### Phase 1: GPU Power Consumption (P0)
1. Create `GPUSpec` embedded data structure
2. Implement GPU power lookup
3. Update EC2 estimator to include GPU power
4. Add GPU carbon estimation tests

### Phase 2: Storage Carbon (P0)
1. Create `StorageSpec` embedded data structure
2. Implement EBS carbon estimator
3. Implement S3 carbon estimator
4. Add storage carbon estimation tests

### Phase 3: Lambda Carbon (P1)
1. Implement Lambda carbon estimator
2. Add ARM64 efficiency factor
3. Add Lambda carbon estimation tests

### Phase 4: RDS & DynamoDB Carbon (P1)
1. Implement RDS carbon estimator (compute + storage)
2. Implement DynamoDB carbon estimator (storage only)
3. Add tests for RDS and DynamoDB

### Phase 5: Embodied Carbon (P2)
1. Create embodied carbon calculator
2. Add embodied carbon to EC2 estimator
3. Update proto response to include operational/embodied breakdown
4. Add embodied carbon tests

### Phase 6: EKS Control Plane (P1)
1. Implement EKS estimator (zero carbon, documentation)
2. Update Supports() response for EKS
3. Add EKS tests

### Phase 7: Grid Factor Update Process (P2)
1. Create grid factor update tool
2. Document update process
3. Add validation tests

---

## Step-by-Step Implementation

### Step 1: Create GPU Specifications (Phase 1)

**File**: `internal/carbon/gpu_specs.go`

```go
package carbon

import (
    _ "embed"
    "encoding/csv"
    "io"
    "strconv"
    "strings"
    "sync"

    "github.com/rs/zerolog"
)

// CSV column indices from GPU specs CSV
const (
    colInstanceType = 0
    colGPUModel     = 1
    colGPUCount     = 2
    colTDPPerGPU    = 3
)

//go:embed data/gpu_specs.csv
var gpuSpecsCSV string

type GPUSpec struct {
    InstanceType string
    GPUModel    string
    GPUCount    int
    TDPPerGPU   float64 // Watts per GPU
}

var (
    gpuSpecs     map[string]GPUSpec
    gpuSpecsOnce sync.Once
)

func parseGPUSpecs() {
    gpuSpecs = make(map[string]GPUSpec)

    reader := csv.NewReader(strings.NewReader(gpuSpecsCSV))
    _, err := reader.Read() // Skip header
    if err != nil {
        logger.Error().Err(err).Msg("failed to read GPU specs CSV header")
        return
    }

    for {
        record, err := reader.Read()
        if err == io.EOF {
            break
        }
        if err != nil {
            logger.Warn().Err(err).Msg("skipping malformed GPU specs CSV row")
            continue
        }

        if len(record) <= colTDPPerGPU {
            continue
        }

        instanceType := strings.TrimSpace(record[colInstanceType])
        if instanceType == "" {
            continue
        }

        gpuCount, err := strconv.Atoi(strings.TrimSpace(record[colGPUCount]))
        if err != nil || gpuCount < 0 {
            continue
        }

        tdpPerGPU, err := strconv.ParseFloat(strings.TrimSpace(record[colTDPPerGPU]), 64)
        if err != nil || tdpPerGPU < 0 {
            continue
        }

        gpuSpecs[instanceType] = GPUSpec{
            InstanceType: instanceType,
            GPUModel:     strings.TrimSpace(record[colGPUModel]),
            GPUCount:     gpuCount,
            TDPPerGPU:    tdpPerGPU,
        }
    }
}

func GetGPUSpec(instanceType string) (GPUSpec, bool) {
    gpuSpecsOnce.Do(parseGPUSpecs)
    spec, ok := gpuSpecs[instanceType]
    return spec, ok
}
```

**File**: `internal/carbon/data/gpu_specs.csv`

```csv
instance_type,gpu_model,gpu_count,tdp_per_gpu_watts
p4d.24xlarge,A100,8,400
p4de.24xlarge,A100,8,400
g5.xlarge,A10G,1,150
g5.4xlarge,A10G,1,150
g5.8xlarge,A10G,1,150
g5.12xlarge,A10G,4,150
g5.16xlarge,A10G,1,150
g5.24xlarge,A10G,4,150
inf2.xlarge,Inferentia2,1,175
inf2.8xlarge,Inferentia2,2,175
inf2.24xlarge,Inferentia2,6,175
inf2.48xlarge,Inferentia2,12,175
trn1.2xlarge,Trainium,1,175
trn1.32xlarge,Trainium,8,175
```

---

### Step 2: Create Storage Specifications (Phase 1)

**File**: `internal/carbon/storage_specs.go`

```go
package carbon

import (
    _ "embed"
    "encoding/csv"
    "io"
    "strconv"
    "strings"
    "sync"

    "github.com/rs/zerolog"
)

// CSV column indices from storage specs CSV
const (
    colServiceType       = 0
    colStorageClass      = 1
    colTechnology        = 2
    colReplicationFactor = 3
    colPowerCoeff       = 4
)

//go:embed data/storage_specs.csv
var storageSpecsCSV string

type StorageSpec struct {
    ServiceType       string
    StorageClass      string
    Technology        string // "SSD" or "HDD"
    ReplicationFactor int
    PowerCoefficient  float64 // Wh/TB-Hour
}

var (
    storageSpecs     map[string]map[string]StorageSpec // service_type -> storage_class -> spec
    storageSpecsOnce sync.Once
)

func parseStorageSpecs() {
    storageSpecs = make(map[string]map[string]StorageSpec)

    reader := csv.NewReader(strings.NewReader(storageSpecsCSV))
    _, err := reader.Read() // Skip header
    if err != nil {
        logger.Error().Err(err).Msg("failed to read storage specs CSV header")
        return
    }

    for {
        record, err := reader.Read()
        if err == io.EOF {
            break
        }
        if err != nil {
            logger.Warn().Err(err).Msg("skipping malformed storage specs CSV row")
            continue
        }

        if len(record) <= colPowerCoeff {
            continue
        }

        serviceType := strings.TrimSpace(record[colServiceType])
        storageClass := strings.TrimSpace(record[colStorageClass])
        if serviceType == "" || storageClass == "" {
            continue
        }

        replicationFactor, err := strconv.Atoi(strings.TrimSpace(record[colReplicationFactor]))
        if err != nil || replicationFactor < 1 || replicationFactor > 3 {
            continue
        }

        powerCoeff, err := strconv.ParseFloat(strings.TrimSpace(record[colPowerCoeff]), 64)
        if err != nil || powerCoeff <= 0 {
            continue
        }

        if storageSpecs[serviceType] == nil {
            storageSpecs[serviceType] = make(map[string]StorageSpec)
        }

        storageSpecs[serviceType][storageClass] = StorageSpec{
            ServiceType:       serviceType,
            StorageClass:      storageClass,
            Technology:        strings.TrimSpace(record[colTechnology]),
            ReplicationFactor: replicationFactor,
            PowerCoefficient:  powerCoeff,
        }
    }
}

func GetStorageSpec(serviceType, storageClass string) (StorageSpec, bool) {
    storageSpecsOnce.Do(parseStorageSpecs)
    if storageSpecs[serviceType] == nil {
        return StorageSpec{}, false
    }
    spec, ok := storageSpecs[serviceType][storageClass]
    return spec, ok
}
```

**File**: `internal/carbon/data/storage_specs.csv`

```csv
service_type,storage_class,technology,replication_factor,power_coefficient_wh_per_tbh
ebs,gp2,SSD,2,1.2
ebs,gp3,SSD,2,1.2
ebs,io1,SSD,2,1.2
ebs,io2,SSD,2,1.2
ebs,st1,HDD,2,0.65
ebs,sc1,HDD,2,0.65
s3,STANDARD,SSD,3,1.2
s3,STANDARD_IA,SSD,3,1.2
s3,INTELLIGENT_TIERING,SSD,3,1.2
s3,ONEZONE_IA,SSD,1,1.2
s3,GLACIER,HDD,3,0.65
s3,DEEP_ARCHIVE,HDD,3,0.65
```

---

### Step 3: Update EC2 Estimator for GPU Support (Phase 1)

**File**: `internal/carbon/estimator.go` (modify existing)

```go
func (e *Estimator) EstimateCarbonGrams(instanceType, region string, utilization, hours float64) (float64, bool) {
    spec, ok := GetInstanceSpec(instanceType)
    if !ok {
        return 0, false
    }

    gridFactor := GetGridFactor(region)

    // Calculate CPU power
    cpuAvgWatts := spec.MinWatts + (utilization * (spec.MaxWatts - spec.MinWatts))
    cpuEnergyKWh := (cpuAvgWatts * float64(spec.VCPUCount) * hours) / 1000.0

    // Check for GPU support
    gpuEnergyKWh := 0.0
    if gpuSpec, ok := GetGPUSpec(instanceType); ok && gpuSpec.GPUCount > 0 {
        // GPU power = TDP per GPU × GPU count × utilization
        gpuAvgWatts := gpuSpec.TDPPerGPU * float64(gpuSpec.GPUCount) * utilization
        gpuEnergyKWh = (gpuAvgWatts * hours) / 1000.0
    }

    // Total energy with PUE
    totalEnergyKWh := (cpuEnergyKWh + gpuEnergyKWh) * AWSPUE

    // Carbon in gCO2e
    carbonGrams := totalEnergyKWh * gridFactor * 1_000_000

    return carbonGrams, true
}
```

---

### Step 4: Implement EBS Carbon Estimator (Phase 1)

**File**: `internal/carbon/ebs_estimator.go`

```go
package carbon

import (
    "github.com/rs/zerolog"
)

type EBSEstimator struct{}

func NewEBSEstimator() *EBSEstimator {
    return &EBSEstimator{}
}

func (e *EBSEstimator) EstimateCarbonGrams(volumeType string, sizeGB float64, region string, hours float64) (float64, bool) {
    spec, ok := GetStorageSpec("ebs", volumeType)
    if !ok {
        return 0, false
    }

    gridFactor := GetGridFactor(region)

    // SizeTB = sizeGB / 1024
    sizeTB := sizeGB / 1024.0

    // Energy (kWh) = (SizeTB × Hours × PowerCoeff × ReplicationFactor) / 1000
    energyKWh := (sizeTB * hours * spec.PowerCoefficient * float64(spec.ReplicationFactor)) / 1000.0

    // Energy with PUE
    energyWithPUE := energyKWh * AWSPUE

    // Carbon (gCO2e)
    carbonGrams := energyWithPUE * gridFactor * 1_000_000

    return carbonGrams, true
}
```

---

### Step 5: Implement S3 Carbon Estimator (Phase 1)

**File**: `internal/carbon/s3_estimator.go`

```go
package carbon

import (
    "github.com/rs/zerolog"
)

type S3Estimator struct{}

func NewS3Estimator() *S3Estimator {
    return &S3Estimator{}
}

func (s *S3Estimator) EstimateCarbonGrams(storageClass string, sizeGB float64, region string, hours float64) (float64, bool) {
    spec, ok := GetStorageSpec("s3", storageClass)
    if !ok {
        return 0, false
    }

    gridFactor := GetGridFactor(region)

    // SizeTB = sizeGB / 1024
    sizeTB := sizeGB / 1024.0

    // Energy (kWh) = (SizeTB × Hours × PowerCoeff × ReplicationFactor) / 1000
    energyKWh := (sizeTB * hours * spec.PowerCoefficient * float64(spec.ReplicationFactor)) / 1000.0

    // Energy with PUE
    energyWithPUE := energyKWh * AWSPUE

    // Carbon (gCO2e)
    carbonGrams := energyWithPUE * gridFactor * 1_000_000

    return carbonGrams, true
}
```

---

### Step 6: Implement Lambda Carbon Estimator (Phase 2)

**File**: `internal/carbon/lambda_estimator.go`

```go
package carbon

const (
    // VCPUPer1792MB is the vCPU equivalent for Lambda memory allocation
    // 1792 MB = 1 vCPU in AWS Lambda
    VCPUPer1792MB = 1792
)

type LambdaEstimator struct{}

func NewLambdaEstimator() *LambdaEstimator {
    return &LambdaEstimator{}
}

func (l *LambdaEstimator) EstimateCarbonGrams(memoryMB int, durationMs int, invocations int64, architecture string, region string) (float64, bool) {
    gridFactor := GetGridFactor(region)

    // vCPU equivalent = MemoryMB / 1792
    vCPUEquivalent := float64(memoryMB) / VCPUPer1792MB

    // Running time (hours) = DurationMs × Invocations / 3,600,000
    runningTimeHours := (float64(durationMs) * float64(invocations)) / 3_600_000.0

    // Use average EC2 power for Lambda (no vCPU mapping)
    // Assume 50% utilization
    avgWatts := 2.12 + (0.50 * (4.5 - 2.12))

    // Energy (kWh) = (AvgWatts × vCPU Equivalent × RunningTime) / 1000
    energyKWh := (avgWatts * vCPUEquivalent * runningTimeHours) / 1000.0

    // Energy with PUE
    energyWithPUE := energyKWh * AWSPUE

    // Carbon (gCO2e)
    carbonGrams := energyWithPUE * gridFactor * 1_000_000

    // ARM64 efficiency factor (20% improvement)
    if architecture == "arm64" {
        carbonGrams *= 0.80
    }

    return carbonGrams, true
}
```

---

### Step 7: Implement RDS Carbon Estimator (Phase 2)

**File**: `internal/carbon/rds_estimator.go`

```go
package carbon

import (
    "github.com/rs/zerolog"
)

type RDSEstimator struct {
    ec2Estimator *Estimator
    ebsEstimator *EBSEstimator
}

func NewRDSEstimator() *RDSEstimator {
    return &RDSEstimator{
        ec2Estimator: NewEstimator(),
        ebsEstimator: NewEBSEstimator(),
    }
}

func (r *RDSEstimator) EstimateCarbonGrams(instanceType string, multiAZ bool, storageType string, sizeGB float64, region string, utilization, hours float64) (float64, bool) {
    // Compute carbon (EC2-equivalent)
    computeCarbon, ok := r.ec2Estimator.EstimateCarbonGrams(instanceType, region, utilization, hours)
    if !ok {
        return 0, false
    }

    // Multi-AZ multiplier (2× for synchronous replica)
    if multiAZ {
        computeCarbon *= 2.0
    }

    // Storage carbon (EBS-equivalent)
    storageCarbon, ok := r.ebsEstimator.EstimateCarbonGrams(storageType, sizeGB, region, hours)
    if !ok {
        return 0, false
    }

    // Multi-AZ storage replication (2×)
    if multiAZ {
        storageCarbon *= 2.0
    }

    // Total carbon = compute + storage
    totalCarbon := computeCarbon + storageCarbon

    return totalCarbon, true
}
```

---

### Step 8: Implement DynamoDB Carbon Estimator (Phase 2)

**File**: `internal/carbon/dynamodb_estimator.go`

```go
package carbon

import (
    "github.com/rs/zerolog"
)

type DynamoDBEstimator struct{}

func NewDynamoDBEstimator() *DynamoDBEstimator {
    return &DynamoDBEstimator{}
}

func (d *DynamoDBEstimator) EstimateCarbonGrams(sizeGB float64, region string, hours float64) (float64, bool) {
    gridFactor := GetGridFactor(region)

    // DynamoDB uses SSD with 3× replication (similar to S3 Standard)
    sizeTB := sizeGB / 1024.0

    // Power coefficient for SSD: 1.2 Wh/TB-Hour
    powerCoefficient := 1.2
    replicationFactor := 3

    // Energy (kWh) = (SizeTB × Hours × PowerCoeff × ReplicationFactor) / 1000
    energyKWh := (sizeTB * hours * powerCoefficient * float64(replicationFactor)) / 1000.0

    // Energy with PUE
    energyWithPUE := energyKWh * AWSPUE

    // Carbon (gCO2e)
    carbonGrams := energyWithPUE * gridFactor * 1_000_000

    return carbonGrams, true
}
```

---

### Step 9: Implement EKS Carbon Estimator (Phase 2)

**File**: `internal/carbon/eks_estimator.go`

```go
package carbon

type EKSEstimator struct{}

func NewEKSEstimator() *EKSEstimator {
    return &EKSEstimator{}
}

func (e *EKSEstimator) EstimateCarbonGrams(region string) (float64, string) {
    // EKS control plane carbon is shared across customers and not allocated
    // Direct users to estimate worker nodes as EC2 instances
    return 0.0, "EKS control plane carbon is shared across customers and not allocated. Estimate worker node carbon footprint as EC2 instances."
}
```

---

### Step 10: Add Embodied Carbon Calculator (Phase 3)

**File**: `internal/carbon/embodied_carbon.go`

```go
package carbon

const (
    // EmbodiedCarbonPerServer is the total embodied carbon per server in kgCO2e
    // Source: Cloud Carbon Footprint methodology
    EmbodiedCarbonPerServer = 1000.0 // kgCO2e

    // ServerLifespanMonths is the server lifespan for amortization
    // Source: CCF methodology (4-year standard)
    ServerLifespanMonths = 48 // months
)

// EmbodiedCarbonConfig contains configuration for embodied carbon calculation
type EmbodiedCarbonConfig struct {
    Enabled           bool
    ServerLifespanMonths int
    EmbodiedCarbonPerServer float64
}

// NewEmbodiedCarbonConfig creates a new embodied carbon configuration with defaults
func NewEmbodiedCarbonConfig() *EmbodiedCarbonConfig {
    return &EmbodiedCarbonConfig{
        Enabled:               false,
        ServerLifespanMonths:    ServerLifespanMonths,
        EmbodiedCarbonPerServer: EmbodiedCarbonPerServer,
    }
}

// CalculateEmbodiedCarbonGrams calculates monthly embodied carbon for an instance
func CalculateEmbodiedCarbonGrams(instanceType string, config *EmbodiedCarbonConfig) (float64, bool) {
    if !config.Enabled {
        return 0.0, true
    }

    spec, ok := GetInstanceSpec(instanceType)
    if !ok {
        return 0, false
    }

    // Find max vCPUs in the instance family (e.g., m5.24xlarge has 96 vCPUs)
    maxFamilyVCPU := float64(spec.VCPUCount) // Simplified: use instance vCPUs
    // TODO: Look up max vCPUs for instance family (requires family mapping)

    // Monthly embodied carbon = (EmbodiedCarbonPerServer / ServerLifespanMonths) × (InstanceVCPU / MaxFamilyVCPU)
    monthlyEmbodiedCarbon := (config.EmbodiedCarbonPerServer / float64(config.ServerLifespanMonths)) *
        (float64(spec.VCPUCount) / maxFamilyVCPU)

    // Convert kgCO2e to gCO2e
    return monthlyEmbodiedCarbon * 1000.0, true
}
```

---

## Testing

### Unit Tests

**File**: `internal/carbon/ebs_estimator_test.go`

```go
package carbon

import "testing"

func TestEBSEstimator_EstimateCarbonGrams(t *testing.T) {
    estimator := NewEBSEstimator()

    tests := []struct {
        name       string
        volumeType string
        sizeGB     float64
        region     string
        hours      float64
        wantCarbon float64
        wantOk     bool
    }{
        {
            name:       "100GB gp3 volume in us-east-1",
            volumeType: "gp3",
            sizeGB:     100,
            region:     "us-east-1",
            hours:      730,
            wantCarbon: 123.45, // Expected carbon in gCO2e
            wantOk:     true,
        },
        {
            name:       "Unknown volume type",
            volumeType: "unknown",
            sizeGB:     100,
            region:     "us-east-1",
            hours:      730,
            wantCarbon: 0,
            wantOk:     false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            gotCarbon, gotOk := estimator.EstimateCarbonGrams(tt.volumeType, tt.sizeGB, tt.region, tt.hours)
            if gotOk != tt.wantOk {
                t.Errorf("gotOk = %v, want %v", gotOk, tt.wantOk)
            }
            if gotOk && gotCarbon < 0 {
                t.Errorf("carbon must be >= 0, got %v", gotCarbon)
            }
        })
    }
}
```

### Integration Tests

**File**: `internal/carbon/integration_test.go`

```go
package carbon

import "testing"

func TestCarbonEstimators_ConcurrentAccess(t *testing.T) {
    // Test thread safety with concurrent gRPC calls
    done := make(chan bool)

    for i := 0; i < 100; i++ {
        go func() {
            _, _ := NewEstimator().EstimateCarbonGrams("t3.micro", "us-east-1", 0.5, 730)
            done <- true
        }()
    }

    for i := 0; i < 100; i++ {
        <-done
    }
}
```

---

## Building and Deployment

### Generate Embedded Data

```bash
# Copy GPU specs to embedded data directory
cp specs/001-carbon-estimation/data/gpu_specs.csv internal/carbon/data/gpu_specs.csv

# Copy storage specs to embedded data directory
cp specs/001-carbon-estimation/data/storage_specs.csv internal/carbon/data/storage_specs.csv

# Verify embedded data
go test ./internal/carbon -v
```

### Build Region-Specific Binaries

```bash
# Build us-east-1 binary
make build-region REGION=us-east-1

# Build all supported regions
make build
```

### Verify Installation

```bash
# Start the plugin
./pulumicost-plugin-aws-public-us-east-1

# Test with grpcurl (in another terminal)
grpcurl -plaintext localhost:PORT pulumicost.v1.CostSourceService/Supports
```

---

## Validation

### Manual Testing

1. **EC2 Instance with GPU**:
```bash
# Request carbon estimate for p4d.24xlarge
grpcurl -plaintext -d '{
  "region": "us-east-1",
  "resource_type": "aws:ec2/instance",
  "properties": {
    "instance_type": "p4d.24xlarge",
    "utilization_percentage": "50",
    "hours": "730"
  }
}' localhost:PORT pulumicost.v1.CostSourceService/GetProjectedCost
```

2. **EBS Volume**:
```bash
grpcurl -plaintext -d '{
  "region": "us-east-1",
  "resource_type": "aws:ebs/volume",
  "properties": {
    "volume_type": "gp3",
    "size_gb": "100",
    "hours": "730"
  }
}' localhost:PORT pulumicost.v1.CostSourceService/GetProjectedCost
```

3. **Lambda Function**:
```bash
grpcurl -plaintext -d '{
  "region": "us-east-1",
  "resource_type": "aws:lambda/function",
  "properties": {
    "memory_mb": "1792",
    "duration_ms": "500",
    "invocations": "1000000",
    "architecture": "x86_64"
  }
}' localhost:PORT pulumicost.v1.CostSourceService/GetProjectedCost
```

### Automated Validation

```bash
# Run all tests
make test

# Run linting
make lint

# Build all region binaries
make build

# Verify binary size (<250MB constraint)
ls -lh pulumicost-plugin-aws-public-*
```

---

## Common Issues

### Issue: Embedded CSV not found
**Error**: `panic: CCF instance specs not embedded. Run: make generate-carbon-data`
**Solution**: Ensure CSV files are in `internal/carbon/data/` and rebuild.

### Issue: Carbon estimate is zero
**Cause**: Instance type or volume type not found in embedded data.
**Solution**: Verify the resource type exists in embedded CSV files.

### Issue: High carbon estimate (unexpectedly)
**Cause**: Incorrect unit conversion (grams vs kilograms).
**Solution**: Verify grid factor is in metric tons/kWh and final result is multiplied by 1,000,000.

### Issue: Concurrent access panic
**Cause**: Mutable shared state without proper synchronization.
**Solution**: Ensure all lookups use read-only maps and initialization uses sync.Once.

---

## Next Steps

After implementing carbon estimation:

1. Update gRPC service router to dispatch to appropriate estimator
2. Add carbon data to `GetProjectedCostResponse.billing_detail`
3. Update `Supports()` response to advertise `METRIC_KIND_CARBON_FOOTPRINT`
4. Add integration tests for gRPC service methods
5. Update documentation with carbon estimation examples
6. Submit pull request with tests passing

---

## References

- [Cloud Carbon Footprint Methodology](https://cloudcarbonfootprint.org/docs/methodology)
- [Constitution](../.specify/memory/constitution.md)
- [Data Model](./data-model.md)
- [gRPC Contract](./contracts/grpc-carbon-estimation.md)
- [Research](./research.md)
