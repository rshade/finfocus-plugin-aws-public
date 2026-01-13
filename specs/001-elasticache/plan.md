# Implementation Plan: ElastiCache Cost Estimation

**Branch**: `001-elasticache` | **Date**: 2025-12-31 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/001-elasticache/spec.md`

## Summary

Add Amazon ElastiCache cost estimation to the FinFocus AWS Public plugin. ElastiCache is a managed in-memory caching service supporting Redis, Memcached, and Valkey engines with node-based hourly pricing. The implementation follows the established service pattern (similar to RDS/EKS) with pricing data generation, embedding, parsing, and estimation.

**Technical Approach**: Index-based pricing lookup keyed by `{nodeType}:{engine}` (e.g., `cache.m5.large:redis`), consistent with the RDS pattern for multi-variant services.

## Technical Context

**Language/Version**: Go 1.25+
**Primary Dependencies**: gRPC (finfocus-spec/sdk/go/pluginsdk), zerolog, sync.WaitGroup
**Storage**: Embedded JSON pricing data (via `//go:embed`)
**Testing**: Go testing with table-driven tests, integration tests with region tags
**Target Platform**: Linux server (gRPC service on 127.0.0.1)
**Project Type**: Single project (existing plugin)
**Performance Goals**: GetProjectedCost() < 100ms, startup with ElastiCache parsing < 500ms total
**Constraints**: Binary size < 250MB (ElastiCache adds ~5-10MB per region), memory < 400MB
**Scale/Scope**: 12 supported regions, ~3 engines (Redis, Memcached, Valkey), ~50+ instance types

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Evidence |
|-----------|--------|----------|
| I. Code Quality & Simplicity | PASS | Follows existing RDS/EKS pattern - no new abstractions |
| II. Testing Discipline | PASS | Unit tests for estimator, integration tests planned |
| III. Protocol & Interface Consistency | PASS | Uses existing gRPC methods, proto types |
| IV. Performance & Reliability | PASS | Indexed lookup < 100ms, parallel init, thread-safe |
| V. Build & Release Quality | PASS | Region build tags, goreleaser integration |
| Security Requirements | PASS | No credentials, input validation via proto types |
| Development Workflow | PASS | Feature branch, conventional commits |

**Constitution Version**: 2.2.0

## Project Structure

### Documentation (this feature)

```text
specs/001-elasticache/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output (N/A - uses existing gRPC)
└── tasks.md             # Phase 2 output (/speckit.tasks command)
```

### Source Code (repository root)

```text
# Existing structure - files to MODIFY:
tools/
└── generate-pricing/
    └── main.go                    # Add "AmazonElastiCache": "elasticache"

internal/
├── pricing/
│   ├── types.go                   # Add elasticacheInstancePrice struct
│   ├── client.go                  # Add parsing + lookup methods
│   ├── embed_use1.go              # Add rawElastiCacheJSON
│   ├── embed_usw2.go              # Add rawElastiCacheJSON
│   ├── embed_euw1.go              # Add rawElastiCacheJSON
│   ├── embed_apse1.go             # Add rawElastiCacheJSON
│   ├── embed_apse2.go             # Add rawElastiCacheJSON
│   ├── embed_apne1.go             # Add rawElastiCacheJSON
│   ├── embed_aps1.go              # Add rawElastiCacheJSON
│   ├── embed_cac1.go              # Add rawElastiCacheJSON
│   ├── embed_sae1.go              # Add rawElastiCacheJSON
│   ├── embed_govw1.go             # Add rawElastiCacheJSON
│   ├── embed_gove1.go             # Add rawElastiCacheJSON
│   ├── embed_usw1.go              # Add rawElastiCacheJSON
│   └── embed_fallback.go          # Add fallback test data
└── plugin/
    ├── projected.go               # Add estimateElastiCache + dispatcher case
    ├── projected_test.go          # Add ElastiCache tests
    └── supports.go                # Add "elasticache" to supported list

# Files to CREATE:
internal/
└── plugin/
    └── integration_elasticache_test.go  # ElastiCache-specific integration tests
```

**Structure Decision**: This is an extension to an existing single-project plugin. No new directories needed - all changes fit existing structure.

## Complexity Tracking

> No constitution violations. All changes follow established patterns.

| Aspect | Assessment |
|--------|-----------|
| New abstractions | 0 - Uses existing pricing client pattern |
| New packages | 0 - All changes in existing packages |
| Build complexity | Minimal - Adds one service to existing goreleaser flow |
| Testing burden | Moderate - 4 new test files, follows existing patterns |

## Phase 0: Research Findings

### Research Task 1: AWS ElastiCache Pricing API Structure

**Decision**: Use `AmazonElastiCache` service code with `Cache Instance` product family filter

**Rationale**: AWS Price List API uses `AmazonElastiCache` as the official service code. The `Cache Instance` product family contains node-based hourly pricing. Serverless uses a different product family (`ElastiCache Serverless`) which is out of scope.

**Key Attributes Identified**:
- `instanceType`: e.g., "cache.m5.large", "cache.r6g.xlarge"
- `cacheEngine`: "Redis", "Memcached", "Valkey"
- `regionCode`: AWS region
- `usagetype`: Contains node usage identifier
- `productFamily`: "Cache Instance" (filter criterion)

**Sample Pricing Structure**:

```json
{
  "products": {
    "SKU123": {
      "productFamily": "Cache Instance",
      "attributes": {
        "instanceType": "cache.m5.large",
        "cacheEngine": "Redis",
        "regionCode": "us-east-1"
      }
    }
  },
  "terms": {
    "OnDemand": {
      "SKU123": {
        "SKU123.JRTCKXETXF": {
          "priceDimensions": {
            "SKU123.JRTCKXETXF.6YS6EN2CT7": {
              "unit": "Hrs",
              "pricePerUnit": {"USD": "0.156"}
            }
          }
        }
      }
    }
  }
}
```

### Research Task 2: Index Key Strategy

**Decision**: Use `{instanceType}:{engine}` composite key (e.g., `cache.m5.large:Redis`)

**Rationale**: Different engines may have different prices for the same instance type. This mirrors the RDS pattern where instance type alone is insufficient (RDS uses `{instanceType}:{engine}` as well).

**Alternatives Considered**:
- Instance type only: Rejected - pricing varies by engine
- Three-level index (region/instance/engine): Rejected - region is already handled by binary selection

### Research Task 3: Engine Name Normalization

**Decision**: Normalize to AWS canonical form (Redis, Memcached, Valkey)

**Normalization Map**:

```go
var engineNormalization = map[string]string{
    "redis":     "Redis",
    "memcached": "Memcached",
    "valkey":    "Valkey",
}
```

**Rationale**: AWS API uses title case. User input may be any case. Normalization at lookup time allows flexible input while ensuring consistent index keys.

### Research Task 4: Pulumi Resource Type Mapping

**Decision**: Support three Pulumi resource types plus legacy format

**Mapping**:

| Pulumi Type | Normalized |
|-------------|------------|
| `aws:elasticache/cluster:Cluster` | elasticache |
| `aws:elasticache/replicationGroup:ReplicationGroup` | elasticache |
| `aws:elasticache/serverlessCache:ServerlessCache` | elasticache (returns $0 - out of scope) |
| `elasticache` | elasticache |

**Rationale**: All three Pulumi types should route to the same estimator. Serverless returns $0 with explanation since ECPU pricing is out of scope for v1.

### Research Task 5: Expected File Sizes

**Decision**: ElastiCache pricing data expected to be 5-10MB per region

**Rationale**: Based on existing services:
- EC2: ~154MB (largest, many instance types)
- RDS: ~7MB (moderate, multi-engine)
- EKS: ~772KB (small, simple pricing)

ElastiCache has ~50 instance types × 3 engines = ~150 SKUs. Estimated 5-10MB is well within binary size budget.
