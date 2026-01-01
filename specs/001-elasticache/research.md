# Research: ElastiCache Cost Estimation

**Feature**: 001-elasticache
**Date**: 2025-12-31
**Status**: Complete

## Research Summary

All technical unknowns have been resolved. The implementation follows established patterns from existing services (RDS, EKS).

## Decision Log

### D1: AWS Pricing API Service Code

**Decision**: Use `AmazonElastiCache` service code

**Rationale**: This is the official AWS Price List API service code for ElastiCache.

**Evidence**: AWS pricing endpoint format:

```text
https://pricing.us-east-1.amazonaws.com/offers/v1.0/aws/AmazonElastiCache/current/{region}/index.json
```

---

### D2: Product Family Filter

**Decision**: Filter on `productFamily == "Cache Instance"`

**Rationale**: ElastiCache pricing data contains multiple product families:
- `Cache Instance` - Node-based hourly pricing (IN SCOPE)
- `ElastiCache Serverless` - ECPU-based pricing (OUT OF SCOPE)
- `Global Datastore` - Cross-region replication (OUT OF SCOPE)

Only "Cache Instance" is relevant for v1.

---

### D3: Pricing Index Key Structure

**Decision**: Use composite key `{instanceType}:{engine}`

**Example Keys**:
- `cache.m5.large:Redis`
- `cache.r6g.xlarge:Memcached`
- `cache.t3.micro:Valkey`

**Rationale**: The same instance type may have different prices for different engines. This matches the RDS pattern where engine affects pricing.

**Alternatives Rejected**:
- Instance type only: Pricing varies by engine
- SKU-based: SKUs are opaque and not user-facing

---

### D4: Engine Normalization

**Decision**: Normalize user input to AWS canonical form at lookup time

**Normalization Map**:

```go
var elasticacheEngineNormalization = map[string]string{
    "redis":     "Redis",
    "memcached": "Memcached",
    "valkey":    "Valkey",
}
```

**Rationale**:
- AWS API uses title case (Redis, Memcached, Valkey)
- Users may provide any case variation
- Normalizing at lookup time (not storage) keeps the index consistent with AWS data

---

### D5: Pulumi Resource Type Handling

**Decision**: Map all ElastiCache Pulumi types to single estimator

**Mapping Table**:

| Input | Normalized | Notes |
|-------|------------|-------|
| `aws:elasticache/cluster:Cluster` | elasticache | Single-node clusters |
| `aws:elasticache/replicationGroup:ReplicationGroup` | elasticache | Multi-node with replicas |
| `aws:elasticache/serverlessCache:ServerlessCache` | elasticache | Returns $0 (out of scope) |
| `elasticache` | elasticache | Legacy format |

**Implementation**: Add to `detectService()` in projected.go:

```go
case "elasticache", "cache":
    return "elasticache"
```

---

### D6: Node Count Tag Keys

**Decision**: Check multiple tag keys for node count

**Priority Order**:
1. `num_cache_clusters` (Pulumi aws:elasticache/replicationGroup)
2. `num_nodes` (Generic)
3. `nodes` (Short form)

**Default**: 1 node if none specified

**Rationale**: Pulumi uses `num_cache_clusters` for replication groups. Supporting multiple keys improves compatibility.

---

### D7: Default Engine

**Decision**: Default to Redis when engine not specified

**Rationale**: Redis is the most common ElastiCache engine choice. This matches user expectations and AWS console defaults.

---

### D8: Pricing Data Size Estimate

**Decision**: Expected 5-10MB per region

**Calculation**:
- ~50 instance types (cache.t3.*, cache.m5.*, cache.r6g.*, etc.)
- ~3 engines (Redis, Memcached, Valkey)
- ~150 unique SKUs
- Each SKU ~50KB with terms → ~7.5MB

**Comparison**:
- EC2: ~154MB (5000+ instance types)
- RDS: ~7MB (100+ instance types × engines)
- EKS: ~772KB (simple pricing)

**Impact**: Well within 250MB binary limit. No constitution concerns.

---

### D9: Error Handling Strategy

**Decision**: Return $0 with explanation for unknown instance types (graceful degradation)

**Rationale**: Consistent with other services. One unknown resource shouldn't fail the entire stack estimation.

**Error Cases**:

| Scenario | Response |
|----------|----------|
| Unknown instance type | $0, "ElastiCache instance type not found" |
| Missing instance type | ERROR_CODE_INVALID_RESOURCE |
| Region mismatch | ERROR_CODE_UNSUPPORTED_REGION |
| Unknown engine | Attempt lookup, $0 if not found |

---

### D10: Integration Points

**Decision**: Reuse all existing infrastructure

**Components Used**:
- `tools/generate-pricing`: Add to serviceConfig map
- `internal/pricing/client.go`: Add parser and lookup methods
- `internal/pricing/embed_*.go`: Add embed directive to all 12 files
- `internal/plugin/projected.go`: Add estimator function
- `internal/plugin/supports.go`: Add to supported services list

**No New Components Required**: Implementation follows established patterns exactly.

## Codebase Pattern Conformance

### Pricing Client Pattern

```go
// Type definition (types.go)
type elasticacheInstancePrice struct {
    Unit       string
    HourlyRate float64
    Currency   string
}

// Index field (client.go)
elasticacheIndex map[string]elasticacheInstancePrice

// Parser (client.go)
func (c *Client) parseElastiCachePricing(data []byte) (string, error)

// Lookup (client.go)
func (c *Client) ElastiCacheOnDemandPricePerHour(instanceType, engine string) (float64, bool)
```

### Estimator Pattern

```go
// Estimator (projected.go)
func (p *AWSPublicPlugin) estimateElastiCache(traceID string, resource *pbc.ResourceDescriptor) (*pbc.GetProjectedCostResponse, error)

// Dispatcher case (projected.go)
case "elasticache":
    resp, err = p.estimateElastiCache(traceID, resource)
```

### Support Registration Pattern

```go
// In Supports() switch (supports.go)
case "ebs", "rds", "eks", "s3", "lambda", "dynamodb", "elb", "natgw", "cloudwatch", "elasticache":
    return &pbc.SupportsResponse{Supported: true}, nil
```

## Open Questions

None. All technical unknowns have been resolved.

## References

- [AWS ElastiCache Pricing](https://aws.amazon.com/elasticache/pricing/)
- [AWS Price List API](https://docs.aws.amazon.com/awsaccountbilling/latest/aboutv2/price-list-api.html)
- Existing RDS implementation: `internal/plugin/projected.go:727`
- Existing EKS implementation: `internal/plugin/projected.go:1050`
