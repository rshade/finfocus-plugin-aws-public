# Research: Embed Raw AWS Pricing JSON Per Service

**Feature**: 018-raw-pricing-embed
**Date**: 2025-12-20

## Overview

This document consolidates research findings for the per-service pricing embed refactor.
No NEEDS CLARIFICATION items existed in the technical context - all decisions were
pre-determined by the GitHub issue and existing codebase patterns.

## Research Findings

### 1. AWS Price List API Structure

**Decision**: Use individual service endpoints without any processing.

**Rationale**: Each AWS service has a dedicated endpoint that returns complete,
self-contained pricing data:

```text
https://pricing.us-east-1.amazonaws.com/offers/v1.0/aws/{SERVICE}/current/{REGION}/index.json
```

The response includes:

- `offerCode` - Service identifier (AmazonEC2, AWSELB, etc.)
- `version` - AWS-assigned version string
- `publicationDate` - When AWS published this data
- `products` - Map of SKU → product details
- `terms` - Map of OnDemand/Reserved → pricing terms

**Alternatives Considered**:

- Combined blob (current approach): Introduces synthetic metadata and combining bugs
- Filtered/trimmed data: Caused v0.0.10 regression (85% data loss)

### 2. Embed File Pattern

**Decision**: Declare multiple `//go:embed` variables per region file.

**Rationale**: Go's embed directive supports multiple declarations in a single file:

```go
//go:build region_use1

package pricing

import _ "embed"

//go:embed data/ec2_us-east-1.json
var rawEC2JSON []byte

//go:embed data/s3_us-east-1.json
var rawS3JSON []byte

// ... etc for each service
```

**Alternatives Considered**:

- Single combined file (current): Loses service isolation and metadata
- Separate embed files per service: Increases file count 7x, more complexity

### 3. Client Initialization Strategy

**Decision**: Parse each service file independently in `Client.init()`.

**Rationale**: The existing `awsPricing` struct in `types.go` already matches
the AWS API response format exactly. Each service file can be parsed with
the same unmarshaling logic, then indexed into the appropriate maps.

```go
func (c *Client) init() error {
    c.once.Do(func() {
        // Parse EC2 pricing
        if err := c.parseEC2(rawEC2JSON); err != nil { ... }
        // Parse S3 pricing
        if err := c.parseS3(rawS3JSON); err != nil { ... }
        // ... etc
    })
}
```

**Alternatives Considered**:

- Parse all at once into combined struct: Loses service isolation for error handling
- Lazy parsing per service: Adds complexity, unpredictable latency

### 4. Service File Naming Convention

**Decision**: `{service}_{region}.json` (lowercase service abbreviation).

**Rationale**: Matches AWS service codes but in lowercase for consistency:

| AWS Service Code | File Name Pattern |
|------------------|-------------------|
| AmazonEC2 | `ec2_{region}.json` |
| AmazonS3 | `s3_{region}.json` |
| AmazonRDS | `rds_{region}.json` |
| AmazonEKS | `eks_{region}.json` |
| AWSLambda | `lambda_{region}.json` |
| AmazonDynamoDB | `dynamodb_{region}.json` |
| AWSELB | `elb_{region}.json` |

**Alternatives Considered**:

- Full AWS service codes: Verbose, inconsistent casing
- Numeric prefixes: Less readable, no benefit

### 5. Per-Service Size Thresholds

**Decision**: Define minimum file sizes based on current AWS API response sizes.

**Rationale**: Prevents v0.0.10-style regressions where data was silently truncated.
Thresholds based on actual AWS API responses (December 2025):

| Service | us-east-1 Size | Minimum Threshold |
|---------|----------------|-------------------|
| EC2 | ~120MB | 100MB |
| RDS | ~15MB | 10MB |
| EKS | ~3MB | 2MB |
| Lambda | ~2MB | 1MB |
| S3 | ~1MB | 500KB |
| DynamoDB | ~500KB | 400KB |
| ELB | ~500KB | 400KB |

**Alternatives Considered**:

- Single combined threshold: Doesn't catch per-service data loss
- Product count thresholds: Less stable (AWS adds/removes products)

### 6. Generation Tool Behavior

**Decision**: Fail fast if any service fetch fails; write files atomically.

**Rationale**: Partial data is worse than no data. If EC2 fetch succeeds but ELB
fails, we don't want a release with missing ELB pricing.

```go
for _, service := range services {
    data, err := fetchServicePricing(region, service)
    if err != nil {
        return fmt.Errorf("failed to fetch %s for %s: %w", service, region, err)
    }
    if err := writeServiceFile(data, service, region, outDir); err != nil {
        return fmt.Errorf("failed to write %s: %w", service, err)
    }
}
```

**Alternatives Considered**:

- Continue on error: Produces incomplete pricing sets
- Retry logic: Adds complexity; transient failures can just re-run tool

## Validation

All research items resolved:

- [x] AWS API structure documented
- [x] Embed pattern determined
- [x] Client init strategy defined
- [x] File naming convention established
- [x] Size thresholds calculated
- [x] Generation tool behavior specified

## Next Steps

Proceed to Phase 1 design artifacts:

- `data-model.md` - Entity definitions for service pricing files
- `quickstart.md` - Developer setup and usage guide
