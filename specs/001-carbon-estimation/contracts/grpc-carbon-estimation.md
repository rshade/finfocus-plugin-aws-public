# gRPC Service Contract: Carbon Estimation

**Feature**: 001-carbon-estimation  
**Date**: 2025-12-31  
**Protocol**: gRPC (PulumiCost CostSourceService v1)

This document defines the gRPC interface contract for carbon footprint estimation across AWS services.

---

## Protocol Overview

The plugin implements the `pulumicost.v1.CostSourceService` gRPC interface. Carbon estimation is delivered as an additional metric in the `GetProjectedCost` response.

**Key Protocol Requirements** (from constitution):
- NEVER log to stdout except PORT announcement
- Use zerolog for structured JSON logging to stderr
- Log entries MUST include `[pulumicost-plugin-aws-public]` component identifier
- Support LOG_LEVEL environment variable for log level configuration
- Use proto-defined ErrorCode enum for errors
- All gRPC method handlers MUST be thread-safe

---

## Service Definition

### Service: pulumicost.v1.CostSourceService

```protobuf
service CostSourceService {
  // Existing methods (unchanged)
  rpc Name(NameRequest) returns (NameResponse);
  rpc Supports(ResourceDescriptor) returns (SupportsResponse);
  rpc GetProjectedCost(ResourceDescriptor) returns (GetProjectedCostResponse);
  rpc GetActualCost(GetActualCostRequest) returns (GetActualCostResponse);
}
```

---

## Carbon Metric Extension

### Metric: METRIC_KIND_CARBON_FOOTPRINT

**Metric Kind**: String identifier for carbon footprint metrics  
**Unit**: gCO2e (grams CO2 equivalent)  
**Return Type**: `float64`

Carbon estimation is returned as a billing item in `GetProjectedCostResponse.billing_detail` with the following structure:

```json
{
  "carbon_footprint": {
    "operational": 12345.67,
    "embodied": 456.78,
    "total": 12802.45,
    "unit": "gCO2e",
    "calculation_breakdown": {
      "service": "ec2",
      "resource_type": "t3.micro",
      "region": "us-east-1",
      "utilization": 0.50,
      "hours": 730,
      "gpu_enabled": false,
      "gpu_power_watts": 0,
      "storage_type": null,
      "storage_size_gb": 0
    }
  }
}
```

### Billing Item Keys

| Key | Type | Description | Required |
|-----|------|-------------|----------|
| `carbon_footprint` | object | Carbon footprint breakdown | No (only if supported) |
| `carbon_footprint.operational` | float64 | Operational carbon (gCO2e) | Yes |
| `carbon_footprint.embodied` | float64 | Embodied carbon (gCO2e) | Yes (0 if not enabled) |
| `carbon_footprint.total` | float64 | Total carbon (gCO2e) | Yes |
| `carbon_footprint.unit` | string | Unit identifier ("gCO2e") | Yes |
| `carbon_footprint.calculation_breakdown` | object | Detailed breakdown | No (diagnostic) |

---

## Message Definitions

### Request: ResourceDescriptor

Carbon estimation uses the existing `ResourceDescriptor` proto message. No changes required.

```protobuf
message ResourceDescriptor {
  string region = 1;
  string resource_type = 2;  // e.g., "aws:ec2/instance", "aws:ebs/volume"
  map<string, string> properties = 3;  // Resource-specific attributes
}
```

**Carbon-Relevant Properties** (by service type):

#### EC2 Instance (`aws:ec2/instance`)
```json
{
  "instance_type": "t3.micro",
  "utilization_percentage": "50",  // Optional, default 50%
  "hours": "730",  // Optional, default monthly
  "include_embodied_carbon": "false"  // Optional, default false
}
```

#### EBS Volume (`aws:ebs/volume`)
```json
{
  "volume_type": "gp3",
  "size_gb": "100",
  "hours": "730"  // Optional, default monthly
}
```

#### S3 Storage (`aws:s3/bucket`)
```json
{
  "storage_class": "STANDARD",  // Optional, default STANDARD
  "size_gb": "100",
  "hours": "730"  // Optional, default monthly
}
```

#### Lambda Function (`aws:lambda/function`)
```json
{
  "memory_mb": "1792",
  "duration_ms": "500",  // Average invocation duration
  "invocations": "1000000",
  "architecture": "x86_64"  // Optional, default x86_64
}
```

#### RDS Instance (`aws:rds/instance`)
```json
{
  "instance_class": "db.m5.large",
  "multi_az": "false",  // Optional, default false
  "storage_type": "gp3",
  "storage_size_gb": "100",
  "utilization_percentage": "50",  // Optional, default 50%
  "hours": "730"  // Optional, default monthly
}
```

#### DynamoDB Table (`aws:dynamodb/table`)
```json
{
  "size_gb": "50",
  "hours": "730"  // Optional, default monthly
}
```

#### EKS Cluster (`aws:eks/cluster`)
```json
{
  "region": "us-east-1"
}
```

### Response: SupportsResponse

The `Supports` method must advertise carbon estimation capability.

```protobuf
message SupportsResponse {
  bool supported = 1;
  string reason = 2;  // If not supported
  repeated string supported_metrics = 3;  // NEW: Array of supported metric kinds
}
```

**Carbon-Enabled Response**:
```json
{
  "supported": true,
  "supported_metrics": [
    "METRIC_KIND_COST",
    "METRIC_KIND_CARBON_FOOTPRINT"
  ]
}
```

**Non-Carbon Service** (e.g., CloudWatch):
```json
{
  "supported": true,
  "reason": "Cost estimation supported, carbon not applicable",
  "supported_metrics": [
    "METRIC_KIND_COST"
  ]
}
```

### Response: GetProjectedCostResponse

Carbon data is added to the existing response via `billing_detail`.

```protobuf
message GetProjectedCostResponse {
  string unit_price = 1;
  string currency = 2;
  double cost_per_month = 3;
  map<string, string> billing_detail = 4;  // Carbon data added here
}
```

**Example Response with Carbon**:
```json
{
  "unit_price": "0.0208",
  "currency": "USD",
  "cost_per_month": 15.18,
  "billing_detail": {
    "carbon_footprint": {
      "operational": 12345.67,
      "embodied": 456.78,
      "total": 12802.45,
      "unit": "gCO2e"
    },
    "carbon_note": "Estimated using Cloud Carbon Footprint methodology. Grid factor: 0.000379 metric tons CO2e/kWh for us-east-1."
  }
}
```

**Example Response without Carbon** (unsupported service):
```json
{
  "unit_price": "0.0208",
  "currency": "USD",
  "cost_per_month": 15.18,
  "billing_detail": {}
}
```

---

## Error Handling

### Error Codes (Proto-Defined)

Use proto-defined ErrorCode enum from pulumicost.v1:

| Error Code | Value | Description | Use Case |
|------------|-------|-------------|-----------|
| ERROR_CODE_INVALID_RESOURCE | 6 | Missing required ResourceDescriptor fields | Missing instance_type, size_gb, etc. |
| ERROR_CODE_UNSUPPORTED_REGION | 9 | Region mismatch or unsupported | Unknown region in grid factors |
| ERROR_CODE_DATA_CORRUPTION | 11 | Embedded pricing data load failed | Malformed embedded CSV data |
| ERROR_CODE_UNSPECIFIED | 0 | Generic error | Unexpected calculation errors |

### Error Response Format

```protobuf
status: INVALID_ARGUMENT
details: [
  {
    "@type": "type.googleapis.com/pulumicost.v1.ErrorDetail",
    "code": 6,  // ERROR_CODE_INVALID_RESOURCE
    "message": "Missing required property: instance_type"
  }
]
```

---

## Service-Specific Contracts

### EC2 Instance Carbon Estimation

**Resource Type**: `aws:ec2/instance`  
**Supported Metrics**: `["METRIC_KIND_COST", "METRIC_KIND_CARBON_FOOTPRINT"]`

**Input Properties**:
| Property | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| instance_type | string | Yes | - | EC2 instance type (e.g., "t3.micro") |
| utilization_percentage | string | No | "50" | CPU utilization (0-100) |
| hours | string | No | "730" | Operating hours |
| include_embodied_carbon | string | No | "false" | Include embodied carbon |

**Output**:
```json
{
  "billing_detail": {
    "carbon_footprint": {
      "operational": 12345.67,
      "embodied": 456.78,
      "total": 12802.45,
      "unit": "gCO2e",
      "calculation_breakdown": {
        "service": "ec2",
        "resource_type": "t3.micro",
        "region": "us-east-1",
        "vcpu_count": 1,
        "min_watts": 2.12,
        "max_watts": 4.5,
        "utilization": 0.50,
        "hours": 730,
        "gpu_enabled": false,
        "gpu_power_watts": 0
      }
    }
  }
}
```

---

### EBS Volume Carbon Estimation

**Resource Type**: `aws:ebs/volume`  
**Supported Metrics**: `["METRIC_KIND_COST", "METRIC_KIND_CARBON_FOOTPRINT"]`

**Input Properties**:
| Property | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| volume_type | string | Yes | - | EBS volume type (gp2, gp3, io1, io2, st1, sc1) |
| size_gb | string | Yes | - | Volume size in gigabytes |
| hours | string | No | "730" | Storage duration (hours) |

**Output**:
```json
{
  "billing_detail": {
    "carbon_footprint": {
      "operational": 123.45,
      "embodied": 0,
      "total": 123.45,
      "unit": "gCO2e",
      "calculation_breakdown": {
        "service": "ebs",
        "resource_type": "gp3",
        "region": "us-east-1",
        "size_gb": 100,
        "size_tb": 0.0977,
        "technology": "SSD",
        "replication_factor": 2,
        "power_coefficient_wh_per_tbh": 1.2,
        "hours": 730
      }
    }
  }
}
```

---

### S3 Storage Carbon Estimation

**Resource Type**: `aws:s3/bucket`  
**Supported Metrics**: `["METRIC_KIND_COST", "METRIC_KIND_CARBON_FOOTPRINT"]`

**Input Properties**:
| Property | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| storage_class | string | No | "STANDARD" | S3 storage class (STANDARD, STANDARD_IA, ONEZONE_IA, GLACIER, DEEP_ARCHIVE) |
| size_gb | string | Yes | - | Storage size in gigabytes |
| hours | string | No | "730" | Storage duration (hours) |

**Output**:
```json
{
  "billing_detail": {
    "carbon_footprint": {
      "operational": 185.12,
      "embodied": 0,
      "total": 185.12,
      "unit": "gCO2e",
      "calculation_breakdown": {
        "service": "s3",
        "storage_class": "STANDARD",
        "region": "us-east-1",
        "size_gb": 100,
        "size_tb": 0.0977,
        "technology": "SSD",
        "replication_factor": 3,
        "power_coefficient_wh_per_tbh": 1.2,
        "hours": 730
      }
    }
  }
}
```

---

### Lambda Function Carbon Estimation

**Resource Type**: `aws:lambda/function`  
**Supported Metrics**: `["METRIC_KIND_COST", "METRIC_KIND_CARBON_FOOTPRINT"]`

**Input Properties**:
| Property | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| memory_mb | string | Yes | - | Allocated memory in megabytes |
| duration_ms | string | Yes | - | Average invocation duration in milliseconds |
| invocations | string | Yes | - | Total number of invocations |
| architecture | string | No | "x86_64" | CPU architecture (x86_64, arm64) |

**Output**:
```json
{
  "billing_detail": {
    "carbon_footprint": {
      "operational": 197.8,
      "embodied": 0,
      "total": 197.8,
      "unit": "gCO2e",
      "calculation_breakdown": {
        "service": "lambda",
        "resource_type": "function",
        "region": "us-east-1",
        "memory_mb": 1792,
        "vcpu_equivalent": 1.0,
        "duration_ms": 500,
        "invocations": 1000000,
        "running_time_hours": 138.89,
        "architecture": "x86_64",
        "utilization": 0.50
      }
    }
  }
}
```

---

### RDS Instance Carbon Estimation

**Resource Type**: `aws:rds/instance`  
**Supported Metrics**: `["METRIC_KIND_COST", "METRIC_KIND_CARBON_FOOTPRINT"]`

**Input Properties**:
| Property | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| instance_class | string | Yes | - | RDS instance class (EC2-equivalent) |
| multi_az | string | No | "false" | Multi-AZ deployment |
| storage_type | string | Yes | - | Storage type (gp3, io1, io2, etc.) |
| storage_size_gb | string | Yes | - | Storage size in gigabytes |
| utilization_percentage | string | No | "50" | CPU utilization (0-100) |
| hours | string | No | "730" | Operating hours |

**Output**:
```json
{
  "billing_detail": {
    "carbon_footprint": {
      "operational": 12568.9,
      "embodied": 456.78,
      "total": 13025.68,
      "unit": "gCO2e",
      "calculation_breakdown": {
        "service": "rds",
        "resource_type": "db.m5.large",
        "region": "us-east-1",
        "compute_carbon": 12345.67,
        "storage_carbon": 223.23,
        "multi_az": false,
        "storage_type": "gp3",
        "storage_size_gb": 100,
        "storage_replication_factor": 1,
        "utilization": 0.50,
        "hours": 730
      }
    }
  }
}
```

---

### DynamoDB Table Carbon Estimation

**Resource Type**: `aws:dynamodb/table`  
**Supported Metrics**: `["METRIC_KIND_COST", "METRIC_KIND_CARBON_FOOTPRINT"]`

**Input Properties**:
| Property | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| size_gb | string | Yes | - | Table storage size in gigabytes |
| hours | string | No | "730" | Storage duration (hours) |

**Output**:
```json
{
  "billing_detail": {
    "carbon_footprint": {
      "operational": 185.12,
      "embodied": 0,
      "total": 185.12,
      "unit": "gCO2e",
      "calculation_breakdown": {
        "service": "dynamodb",
        "resource_type": "table",
        "region": "us-east-1",
        "size_gb": 50,
        "size_tb": 0.0488,
        "technology": "SSD",
        "replication_factor": 3,
        "power_coefficient_wh_per_tbh": 1.2,
        "hours": 730
      }
    }
  }
}
```

---

### EKS Cluster Carbon Estimation

**Resource Type**: `aws:eks/cluster`  
**Supported Metrics**: `["METRIC_KIND_COST"]` (carbon not supported for control plane)

**Input Properties**:
| Property | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| region | string | Yes | - | AWS region |

**Output**:
```json
{
  "unit_price": "0.00",
  "currency": "USD",
  "cost_per_month": 73.0,
  "billing_detail": {
    "carbon_note": "EKS control plane carbon is shared across customers and not allocated. Estimate worker node carbon footprint as EC2 instances."
  }
}
```

**Supports Response**:
```json
{
  "supported": true,
  "supported_metrics": [
    "METRIC_KIND_COST"
  ],
  "reason": "Carbon estimation not supported for EKS control plane. Estimate worker nodes as EC2 instances."
}
```

---

## Validation Rules

### Input Validation

**Required Fields**:
- EC2: `instance_type` required
- EBS: `volume_type`, `size_gb` required
- S3: `size_gb` required
- Lambda: `memory_mb`, `duration_ms`, `invocations` required
- RDS: `instance_class`, `storage_type`, `storage_size_gb` required
- DynamoDB: `size_gb` required
- EKS: `region` required

**Value Validation**:
- `utilization_percentage`: Must be between 0 and 100
- `hours`: Must be >= 0
- `size_gb`: Must be >= 0
- `memory_mb`: Must be >= 128 (Lambda minimum)
- `duration_ms`: Must be >= 1
- `invocations`: Must be >= 0

**Type Validation**:
- Numeric properties must be parseable as float/int
- Storage classes must be valid for the service
- Instance types must exist in embedded data

### Output Validation

**Carbon Footprint Object** (if present):
- `operational`: Must be >= 0
- `embodied`: Must be >= 0
- `total`: Must equal `operational + embodied`
- `unit`: Must be "gCO2e"

**Supports Metrics**:
- Must always include `METRIC_KIND_COST` if service is supported
- Include `METRIC_KIND_CARBON_FOOTPRINT` only if carbon estimation is available

---

## Performance Requirements

**From Constitution**:
- GetProjectedCost() RPC: < 100ms per call
- Supports() RPC: < 10ms per call

**Implementation Strategy**:
- All lookup tables embedded at build time (no network I/O)
- Map lookups for O(1) access
- Lazy initialization with sync.Once
- Thread-safe concurrent access (mutex-free where possible)

---

## Backward Compatibility

**No Breaking Changes**:
- Carbon estimation is optional additive metric
- Existing cost estimation behavior unchanged
- Supports() method advertises carbon capability via new `supported_metrics` field
- Clients without carbon support ignore `carbon_footprint` in billing_detail

**Versioning**:
- Carbon estimation does not require protocol version change
- Clients can detect carbon support via `supported_metrics` array
- Default behavior (no carbon) for services where unsupported

---

## Thread Safety

**Requirements** (from constitution):
- All gRPC method handlers MUST be thread-safe
- Support at least 100 concurrent GetProjectedCost() calls
- Embed pricing data once using sync.Once

**Implementation**:
- Read-only map lookups are inherently thread-safe
- No mutable shared state during estimation
- sync.Once ensures single initialization
- No locking required for estimation path

---

## Logging Requirements

**From Constitution**:
- Use zerolog for structured JSON logging to stderr
- Log entries MUST include `[pulumicost-plugin-aws-public]` component identifier
- Never log to stdout except PORT announcement

**Example Log Entry**:
```json
{
  "level": "debug",
  "component": "[pulumicost-plugin-aws-public]",
  "msg": "Calculating carbon for EC2 instance",
  "service": "ec2",
  "instance_type": "t3.micro",
  "region": "us-east-1",
  "carbon_gco2e": 12345.67
}
```

---

## Testing Strategy

### Unit Tests
- Mock gRPC requests for each service type
- Validate carbon calculation against CCF reference values
- Test input validation error paths

### Integration Tests
- End-to-end gRPC calls with real embedded data
- Concurrent access tests (100+ goroutines)
- Performance benchmarks (<100ms target)

### Contract Tests
- Verify Supports() returns correct `supported_metrics`
- Validate billing_detail structure
- Test backward compatibility (clients without carbon support)
