# Data Model: FinFocus AWS Public Plugin

**Phase**: 1 - Design
**Date**: 2025-11-16
**Status**: Complete

## Overview

This document defines the internal data structures used by the plugin. The plugin uses **proto-defined types** for all external communication and **internal Go types** for pricing data management.

## Proto-Defined Types (External Interface)

**Source**: `github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1`

These types are NEVER modified by the plugin - they are defined by the proto contract.

### ResourceDescriptor (Input)

```protobuf
message ResourceDescriptor {
  string provider = 1;       // Always "aws" for this plugin
  string resource_type = 2;  // "ec2", "ebs", "s3", "lambda", "rds", "dynamodb"
  string sku = 3;            // Instance type or volume type
  string region = 4;         // "us-east-1", "us-west-2", "eu-west-1"
  map<string, string> tags = 5;  // Additional metadata
}
```

**Usage in Plugin**:
- `provider`: Validated to be "aws", rejected otherwise
- `resource_type`: Routed to appropriate estimator (estimateEC2, estimateEBS, estimateStub)
- `sku`: Used as lookup key for pricing (instance type for EC2, volume type for EBS)
- `region`: Validated against plugin's compiled region
- `tags`: Used for EBS size lookup (keys: "size" or "volume_size")

---

### GetProjectedCostResponse (Output)

```protobuf
message GetProjectedCostResponse {
  double unit_price = 1;      // Per-unit rate ($/hour or $/GB-month)
  string currency = 2;        // Always "USD" for v1
  double cost_per_month = 3;  // Calculated monthly cost
  string billing_detail = 4;  // Human-readable assumptions
}
```

**Example Values**:

**EC2 t3.micro in us-east-1**:
```json
{
  "unit_price": 0.0104,
  "currency": "USD",
  "cost_per_month": 7.592,
  "billing_detail": "On-demand Linux/Shared instance, t3.micro, 730 hrs/month"
}
```

**EBS gp3 100GB in us-east-1**:
```json
{
  "unit_price": 0.08,
  "currency": "USD",
  "cost_per_month": 8.0,
  "billing_detail": "EBS gp3 storage, 100GB"
}
```

**S3 (stub service)**:
```json
{
  "unit_price": 0,
  "currency": "USD",
  "cost_per_month": 0,
  "billing_detail": "S3 cost estimation not implemented - returning $0"
}
```

---

### SupportsResponse (Output)

```protobuf
message SupportsResponse {
  bool supported = 1;
  string reason = 2;
}
```

**Example Values**:

**Fully supported (EC2 in correct region)**:
```json
{
  "supported": true,
  "reason": "Fully supported"
}
```

**Limited support (S3 stub)**:
```json
{
  "supported": true,
  "reason": "Limited support - returns $0 estimate"
}
```

**Unsupported (wrong region)**:
```json
{
  "supported": false,
  "reason": "Region us-west-2 not supported by this binary (compiled for us-east-1)"
}
```

---

## Internal Plugin Types

**Source**: Plugin implementation in `internal/plugin/` and `internal/pricing/`

### AWSPublicPlugin

**File**: `internal/plugin/plugin.go`

```go
package plugin

import (
    pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

// AWSPublicPlugin implements CostSourceService gRPC interface
type AWSPublicPlugin struct {
    pbc.UnimplementedCostSourceServiceServer  // Embed for forward compatibility

    region  string         // Compiled region (e.g., "us-east-1")
    pricing PricingClient  // Interface to pricing data
}

// NewAWSPublicPlugin creates a new plugin instance
func NewAWSPublicPlugin(region string, pricing PricingClient) *AWSPublicPlugin {
    return &AWSPublicPlugin{
        region:  region,
        pricing: pricing,
    }
}
```

**Fields**:
- `region`: The AWS region this binary is compiled for (embedded at build time)
- `pricing`: Abstraction for pricing data access (allows mocking in tests)

**Immutability**: Plugin is stateless - all fields are set at construction and never modified. Safe for concurrent gRPC calls.

---

### PricingClient (Interface)

**File**: `internal/pricing/client.go`

```go
package pricing

// PricingClient provides pricing data lookups
type PricingClient interface {
    // Region returns the AWS region for this pricing data
    Region() string

    // Currency returns the currency code (always "USD" for v1)
    Currency() string

    // EC2OnDemandPricePerHour returns hourly rate for an EC2 instance
    // Returns (price, true) if found, (0, false) if not found
    EC2OnDemandPricePerHour(instanceType, os, tenancy string) (float64, bool)

    // EBSPricePerGBMonth returns monthly rate per GB for an EBS volume
    // Returns (price, true) if found, (0, false) if not found
    EBSPricePerGBMonth(volumeType string) (float64, bool)
}
```

**Why Interface**: Allows unit tests to use mockPricingClient without embedding real pricing data.

---

### Client (PricingClient Implementation)

**File**: `internal/pricing/client.go`

```go
package pricing

import (
    "encoding/json"
    "sync"
)

// Client implements PricingClient with embedded JSON data
type Client struct {
    region   string
    currency string

    // Thread-safe initialization
    once sync.Once
    err  error

    // In-memory pricing indexes (built on first access)
    ec2Index map[string]ec2OnDemandPrice
    ebsIndex map[string]ebsVolumePrice
}

// NewClient creates a Client from embedded rawPricingJSON
func NewClient() (*Client, error) {
    c := &Client{}
    if err := c.init(); err != nil {
        return nil, err
    }
    return c, nil
}

// init parses embedded pricing data exactly once
func (c *Client) init() error {
    c.once.Do(func() {
        var data pricingData
        if err := json.Unmarshal(rawPricingJSON, &data); err != nil {
            c.err = err
            return
        }

        c.region = data.Region
        c.currency = data.Currency
        c.ec2Index = buildEC2Index(data.EC2)
        c.ebsIndex = buildEBSIndex(data.EBS)
    })
    return c.err
}
```

**Thread Safety**:
- `sync.Once` ensures parsing happens exactly once, even under concurrent access
- Indexes (maps) are built during initialization and then read-only
- Safe for concurrent `EC2OnDemandPricePerHour()` and `EBSPricePerGBMonth()` calls

**Error Handling**:
- If embedded JSON is corrupted, `init()` sets `c.err` and returns it on every call
- This triggers ERROR_CODE_DATA_CORRUPTION at the gRPC layer

---

## Embedded Pricing Data Format

**Files**: `data/aws_pricing_<region>.json` (generated at build time)

**Schema**:
```json
{
  "region": "us-east-1",
  "currency": "USD",
  "ec2": {
    "t3.micro": {
      "instance_type": "t3.micro",
      "operating_system": "Linux",
      "tenancy": "Shared",
      "hourly_rate": 0.0104
    },
    "t3.small": {
      "instance_type": "t3.small",
      "operating_system": "Linux",
      "tenancy": "Shared",
      "hourly_rate": 0.0208
    }
  },
  "ebs": {
    "gp3": {
      "volume_type": "gp3",
      "rate_per_gb_month": 0.08
    },
    "gp2": {
      "volume_type": "gp2",
      "rate_per_gb_month": 0.10
    },
    "io1": {
      "volume_type": "io1",
      "rate_per_gb_month": 0.125
    },
    "io2": {
      "volume_type": "io2",
      "rate_per_gb_month": 0.125
    }
  }
}
```

**Constraints**:
- File size: <500KB per region (trimmed from full AWS pricing API)
- Only on-demand pricing (no spot, reserved, savings plans)
- EC2: Only Linux/Shared instances (most common use case)
- EBS: Only standard volume types (gp2, gp3, io1, io2)

---

### pricingData (Internal Parse Target)

**File**: `internal/pricing/types.go`

```go
package pricing

// pricingData is the top-level structure for unmarshaling embedded JSON
type pricingData struct {
    Region   string                      `json:"region"`
    Currency string                      `json:"currency"`
    EC2      map[string]ec2OnDemandPrice `json:"ec2"`
    EBS      map[string]ebsVolumePrice   `json:"ebs"`
}

// ec2OnDemandPrice represents a single EC2 instance pricing entry
type ec2OnDemandPrice struct {
    InstanceType    string  `json:"instance_type"`
    OperatingSystem string  `json:"operating_system"`
    Tenancy         string  `json:"tenancy"`
    HourlyRate      float64 `json:"hourly_rate"`
}

// ebsVolumePrice represents a single EBS volume type pricing entry
type ebsVolumePrice struct {
    VolumeType      string  `json:"volume_type"`
    RatePerGBMonth  float64 `json:"rate_per_gb_month"`
}
```

---

### Pricing Indexes (Internal)

**File**: `internal/pricing/client.go`

```go
// ec2Index key format: "{instanceType}/{os}/{tenancy}"
// Example: "t3.micro/Linux/Shared"
type ec2Index map[string]ec2OnDemandPrice

// ebsIndex key format: "{volumeType}"
// Example: "gp3"
type ebsIndex map[string]ebsVolumePrice

// buildEC2Index creates O(1) lookup map from pricing data
func buildEC2Index(ec2Data map[string]ec2OnDemandPrice) map[string]ec2OnDemandPrice {
    index := make(map[string]ec2OnDemandPrice, len(ec2Data))
    for instanceType, price := range ec2Data {
        key := fmt.Sprintf("%s/%s/%s", instanceType, price.OperatingSystem, price.Tenancy)
        index[key] = price
    }
    return index
}

// buildEBSIndex creates O(1) lookup map from pricing data
func buildEBSIndex(ebsData map[string]ebsVolumePrice) map[string]ebsVolumePrice {
    // Already keyed by volume type, just copy
    return ebsData
}
```

**Performance**: O(1) lookups for pricing queries, critical for <100ms RPC latency target.

---

## Data Flow

### GetProjectedCost for EC2

```
1. gRPC Request
   → GetProjectedCostRequest{resource: ResourceDescriptor}

2. Plugin Validation
   → Check provider == "aws"
   → Check region == plugin.region
   → Check resource_type == "ec2"
   → Extract sku (instance type)

3. Pricing Lookup
   → pricing.EC2OnDemandPricePerHour("t3.micro", "Linux", "Shared")
   → O(1) map lookup: ec2Index["t3.micro/Linux/Shared"]
   → Returns (0.0104, true)

4. Cost Calculation
   → monthlyCost = hourlyRate * 730
   → monthlyCost = 0.0104 * 730 = 7.592

5. gRPC Response
   → GetProjectedCostResponse{
       unit_price: 0.0104,
       currency: "USD",
       cost_per_month: 7.592,
       billing_detail: "On-demand Linux/Shared instance, t3.micro, 730 hrs/month"
     }
```

---

### GetProjectedCost for EBS

```
1. gRPC Request
   → GetProjectedCostRequest{resource: ResourceDescriptor{
       resource_type: "ebs",
       sku: "gp3",
       tags: {"size": "100"}
     }}

2. Plugin Validation
   → Check provider == "aws"
   → Check region == plugin.region
   → Check resource_type == "ebs"
   → Extract sku (volume type)
   → Extract size from tags["size"] or tags["volume_size"]
   → Default to 8GB if not found

3. Pricing Lookup
   → pricing.EBSPricePerGBMonth("gp3")
   → O(1) map lookup: ebsIndex["gp3"]
   → Returns (0.08, true)

4. Cost Calculation
   → monthlyCost = sizeGB * ratePerGBMonth
   → monthlyCost = 100 * 0.08 = 8.0

5. gRPC Response
   → GetProjectedCostResponse{
       unit_price: 0.08,
       currency: "USD",
       cost_per_month: 8.0,
       billing_detail: "EBS gp3 storage, 100GB"
     }
```

---

## Assumptions and Defaults

### EC2 Assumptions

| Assumption | Value | Rationale |
|------------|-------|-----------|
| Operating System | Linux | Most common AWS use case |
| Tenancy | Shared | Standard multi-tenant instances |
| Hours per Month | 730 | 24×7 on-demand usage (365 days × 24 hours ÷ 12 months) |
| Pricing Model | On-demand | No spot, reserved, or savings plan support in v1 |

**Documented in**: `billing_detail` field of GetProjectedCostResponse

---

### EBS Assumptions

| Assumption | Value | Rationale |
|------------|-------|-----------|
| Default Size | 8 GB | AWS default for most EBS volumes |
| Billing Model | Per GB-month | Standard EBS pricing unit |
| IOPS/Throughput | Standard | No provisioned IOPS or throughput pricing in v1 |

**Documented in**: `billing_detail` field when size is defaulted

---

## Edge Cases

### Missing or Invalid Data

| Scenario | Behavior |
|----------|----------|
| ResourceDescriptor lacks `sku` | Return gRPC InvalidArgument error |
| EC2 instance type not in pricing data | Return cost_per_month=0, billing_detail explains not found |
| EBS volume type not in pricing data | Return cost_per_month=0, billing_detail explains not found |
| EBS tags lack size | Default to 8GB, document in billing_detail |
| ResourceDescriptor.region mismatch | Return gRPC FailedPrecondition with ERROR_CODE_UNSUPPORTED_REGION |
| Embedded pricing JSON corrupted | Return gRPC Internal error with ERROR_CODE_DATA_CORRUPTION |

---

## Type Safety

**Go Strong Typing**:
- All proto messages are strongly typed (no `interface{}` or `map[string]any` in proto types)
- Pricing lookups return `(float64, bool)` to distinguish "not found" from "zero cost"
- No string-to-number conversions in hot path (pricing is pre-parsed)

**Proto Compatibility**:
- Plugin uses generated proto code from finfocus-spec
- No manual proto message construction (use proto builders)
- Proto version compatibility handled by gRPC runtime

---

## Memory Usage

**Estimated Memory Footprint** (per region binary):

| Component | Size Estimate |
|-----------|---------------|
| Embedded pricing JSON (raw) | ~100-200 KB |
| Parsed pricing indexes (ec2 + ebs) | ~500 KB - 1 MB |
| gRPC server overhead | ~5-10 MB |
| Plugin code and dependencies | ~10-20 MB |
| **Total** | **~15-30 MB** |

**Well under** the 50 MB constraint from constitution and success criteria (SC-003).

---

## Validation Rules

### ResourceDescriptor Validation

```go
func validateResourceDescriptor(rd *pbc.ResourceDescriptor) error {
    if rd == nil {
        return errors.New("missing resource descriptor")
    }
    if rd.Provider == "" {
        return errors.New("missing provider")
    }
    if rd.ResourceType == "" {
        return errors.New("missing resource_type")
    }
    if rd.Region == "" {
        return errors.New("missing region")
    }
    // sku validation is resource-type specific (done in estimators)
    return nil
}
```

**Applied in**: All gRPC RPC handlers before processing.

---

## Conclusion

The data model is designed for:
- **Simplicity**: Proto types for external interface, minimal internal types
- **Performance**: O(1) pricing lookups via indexed maps
- **Thread Safety**: sync.Once initialization, immutable indexes
- **Testability**: PricingClient interface allows mocking

**Next**: Generate API contracts in `contracts/` directory.
