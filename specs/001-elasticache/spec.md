# Feature Specification: ElastiCache Cost Estimation

**Feature Branch**: `001-elasticache`
**Created**: 2025-12-31
**Status**: Draft
**Input**: User description: "Implement cost estimation for Amazon ElastiCache, a managed in-memory caching service. ElastiCache supports Redis, Memcached, and Valkey engines with node-based pricing."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Basic ElastiCache Cost Estimation (Priority: P1)

As a platform engineer, I want to get cost estimates for my ElastiCache clusters so that I can understand the monthly cost of my caching infrastructure.

**Why this priority**: This is the core value proposition - without basic cost estimation, the feature provides no value. Platform engineers need accurate monthly cost projections to plan budgets and compare infrastructure options.

**Independent Test**: Can be fully tested by submitting an ElastiCache resource descriptor with instance type and engine, verifying a non-zero monthly cost is returned.

**Acceptance Scenarios**:

1. **Given** a resource descriptor with `resource_type: "elasticache"`, `sku: "cache.m5.large"`, and `tags: {"engine": "redis"}`, **When** GetProjectedCost is called, **Then** return a positive monthly cost with billing detail explaining the calculation.

2. **Given** a resource descriptor with `resource_type: "aws:elasticache/cluster:Cluster"` (Pulumi format), **When** GetProjectedCost is called, **Then** the system recognizes the resource type and returns cost estimation.

3. **Given** a resource descriptor with `resource_type: "aws:elasticache/replicationGroup:ReplicationGroup"` (Pulumi format), **When** GetProjectedCost is called, **Then** the system recognizes the resource type and returns cost estimation.

---

### User Story 2 - Multi-Engine Support (Priority: P1)

As a platform engineer, I want cost estimates for all ElastiCache engine types (Redis, Memcached, Valkey) so that I can accurately estimate costs regardless of which cache engine I choose.

**Why this priority**: Engine support is essential for accuracy - different engines may have different pricing. Users should not get $0 costs because they chose Memcached instead of Redis.

**Independent Test**: Can be tested by submitting requests with each engine type and verifying non-zero costs are returned for all supported engines.

**Acceptance Scenarios**:

1. **Given** a resource descriptor with `tags: {"engine": "redis"}`, **When** GetProjectedCost is called, **Then** return pricing specific to Redis engine.

2. **Given** a resource descriptor with `tags: {"engine": "memcached"}`, **When** GetProjectedCost is called, **Then** return pricing specific to Memcached engine.

3. **Given** a resource descriptor with `tags: {"engine": "valkey"}`, **When** GetProjectedCost is called, **Then** return pricing specific to Valkey engine.

4. **Given** a resource descriptor with engine in different cases ("REDIS", "Redis", "redis"), **When** GetProjectedCost is called, **Then** normalize the engine name and return correct pricing.

---

### User Story 3 - Multi-Node Cluster Estimation (Priority: P2)

As a platform engineer, I want cost estimates that account for the number of nodes in my ElastiCache cluster so that I can accurately estimate costs for production clusters with multiple replicas.

**Why this priority**: Production clusters typically have multiple nodes for high availability. Single-node estimation would significantly underestimate real costs.

**Independent Test**: Can be tested by submitting requests with different node counts and verifying the cost scales proportionally.

**Acceptance Scenarios**:

1. **Given** a resource descriptor with `tags: {"num_nodes": "3"}`, **When** GetProjectedCost is called, **Then** return cost equal to (hourly_rate x 3 nodes x 730 hours).

2. **Given** a resource descriptor with `tags: {"num_cache_clusters": "5"}` (Pulumi replication group format), **When** GetProjectedCost is called, **Then** recognize this as node count and calculate accordingly.

3. **Given** a resource descriptor without node count specified, **When** GetProjectedCost is called, **Then** default to 1 node and note this assumption in billing detail.

---

### User Story 4 - Sensible Defaults (Priority: P2)

As a platform engineer, I want the system to use sensible defaults when optional parameters are missing so that I still get useful estimates even with minimal input.

**Why this priority**: Users may not always specify all parameters. Reasonable defaults improve usability while maintaining accuracy for common use cases.

**Independent Test**: Can be tested by submitting minimal resource descriptors and verifying defaults are applied with clear documentation in billing detail.

**Acceptance Scenarios**:

1. **Given** a resource descriptor without engine specified, **When** GetProjectedCost is called, **Then** default to Redis and include "engine defaulted to redis" in billing detail.

2. **Given** a resource descriptor without node count, **When** GetProjectedCost is called, **Then** default to 1 node and include "node count defaulted to 1" in billing detail.

3. **Given** a resource descriptor with only instance type (sku), **When** GetProjectedCost is called, **Then** apply all defaults and return a valid cost estimate.

---

### User Story 5 - Support Check (Priority: P3)

As a PulumiCost core service, I want to query whether the plugin supports ElastiCache resources so that I can route requests appropriately.

**Why this priority**: Support checking is infrastructure-level functionality. The core estimation features are more critical for end users.

**Independent Test**: Can be tested by calling Supports() with ElastiCache resource types and verifying correct responses.

**Acceptance Scenarios**:

1. **Given** a resource with `resource_type: "elasticache"`, **When** Supports is called, **Then** return `supported: true`.

2. **Given** a resource with `resource_type: "aws:elasticache/cluster:Cluster"`, **When** Supports is called, **Then** return `supported: true`.

3. **Given** a resource with `resource_type: "aws:elasticache/replicationGroup:ReplicationGroup"`, **When** Supports is called, **Then** return `supported: true`.

---

### Edge Cases

- What happens when an unknown instance type is requested? System returns $0 with billing detail explaining the instance type was not found in pricing data.
- What happens when an invalid engine type is specified? System attempts lookup, returns $0 with explanation if not found.
- What happens when node count is zero or negative? System ignores invalid values and defaults to 1 node.
- What happens when node count is extremely large (e.g., 1000)? System calculates the cost mathematically without arbitrary limits.
- What happens when the region in the request doesn't match the plugin's embedded region? System returns ERROR_CODE_UNSUPPORTED_REGION with appropriate details.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST accept ElastiCache resource descriptors with `resource_type` identifying ElastiCache resources
- **FR-002**: System MUST support instance types using the `cache.*` naming convention (e.g., cache.m5.large, cache.r6g.xlarge)
- **FR-003**: System MUST support Redis, Memcached, and Valkey cache engines
- **FR-004**: System MUST normalize engine names to handle case variations (redis, Redis, REDIS)
- **FR-005**: System MUST calculate monthly cost as (hourly_rate x num_nodes x 730 hours)
- **FR-006**: System MUST support node count via `num_nodes`, `num_cache_clusters`, or `nodes` tags
- **FR-007**: System MUST default engine to Redis when not specified
- **FR-008**: System MUST default node count to 1 when not specified
- **FR-009**: System MUST include assumption notes in billing detail when defaults are applied
- **FR-010**: System MUST return $0 with explanatory message when instance type is not found in pricing data
- **FR-011**: System MUST recognize Pulumi resource type formats (`aws:elasticache/cluster:Cluster`, `aws:elasticache/replicationGroup:ReplicationGroup`)
- **FR-012**: System MUST return ERROR_CODE_INVALID_RESOURCE when instance type is missing entirely
- **FR-013**: System MUST embed ElastiCache pricing data for all supported regions
- **FR-014**: System MUST filter pricing data to only include "Cache Instance" product family (not Serverless)

### Key Entities

- **Cache Instance**: The billable unit for ElastiCache - represents a single cache node with specific instance type and engine
- **Cache Engine**: The software running on cache nodes - Redis, Memcached, or Valkey - affects pricing
- **Replication Group**: A collection of cache nodes (clusters) for high availability - cost is sum of all nodes
- **Instance Type**: The hardware configuration (e.g., cache.m5.large) - determines hourly rate

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Users receive cost estimates for common ElastiCache instance types (cache.t3.micro, cache.m5.large, cache.r6g.xlarge) with non-zero values
- **SC-002**: Cost estimates for 3-node clusters are within 1% of (3 x single-node cost)
- **SC-003**: All three supported engines (Redis, Memcached, Valkey) return valid pricing for the same instance type
- **SC-004**: Billing detail clearly explains all assumptions when defaults are applied
- **SC-005**: Response time for cost estimation is under 100ms for cached pricing data lookups
- **SC-006**: ElastiCache pricing data files are under 10MB per region (reasonable size for embedded data)
- **SC-007**: 100% of Pulumi-format resource types are correctly recognized and estimated

## Assumptions

- On-demand pricing only; Reserved Instances and Savings Plans are out of scope for v1
- Serverless ElastiCache is out of scope for v1 (uses different pricing model with ECPUs)
- Global Datastore data transfer costs are out of scope for v1
- Backup storage costs are not included in estimates
- Data transfer costs are not included in estimates
- Hours per month is standardized at 730 (consistent with other services in the plugin)

## Out of Scope

- Reserved Instance pricing
- Serverless ElastiCache (ECPU-based pricing)
- Global Datastore data transfer
- Backup storage costs
- Data transfer costs
- Carbon footprint estimation for ElastiCache
