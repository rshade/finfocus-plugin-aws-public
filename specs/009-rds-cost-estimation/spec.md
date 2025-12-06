# Feature Specification: RDS Instance Cost Estimation

**Feature Branch**: `009-rds-cost-estimation`
**Created**: 2025-12-02
**Status**: Draft
**Input**: GitHub Issue #52 - feat(rds): implement RDS instance cost estimation

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Basic RDS Instance Cost Query (Priority: P1)

As a cloud cost analyst, I want to get accurate cost estimates for RDS database instances based on instance type and database engine, so that I can understand the compute cost component of my database infrastructure.

**Why this priority**: Instance compute costs represent the primary cost driver for RDS. Without accurate instance pricing, cost estimates are fundamentally incomplete. This is the core value proposition.

**Independent Test**: Can be fully tested by sending a cost query for a db.t3.medium MySQL instance and verifying a non-zero hourly rate is returned with correct monthly calculation (hourly × 730 hours).

**Acceptance Scenarios**:

1. **Given** an RDS resource with instance type "db.t3.medium" and engine "mysql", **When** GetProjectedCost is called, **Then** return accurate hourly rate and monthly cost (hourly × 730)
2. **Given** an RDS resource with instance type "db.m5.large" and engine "postgres", **When** GetProjectedCost is called, **Then** return accurate hourly rate specific to PostgreSQL engine pricing
3. **Given** an RDS resource with no engine specified, **When** GetProjectedCost is called, **Then** default to MySQL engine pricing

---

### User Story 2 - RDS Storage Cost Estimation (Priority: P2)

As a cloud cost analyst, I want storage costs included in RDS estimates based on volume type and size, so that I can see the complete database infrastructure cost.

**Why this priority**: Storage is the second major cost component for RDS. Combined with instance costs, this provides a complete basic cost picture.

**Independent Test**: Can be tested by sending a cost query with storage_type and storage_size tags, verifying storage cost is calculated and added to the total monthly cost.

**Acceptance Scenarios**:

1. **Given** an RDS resource with storage_type "gp3" and storage_size "100", **When** GetProjectedCost is called, **Then** include storage cost (100 GB × per-GB rate) in total monthly cost
2. **Given** an RDS resource with no storage_type specified, **When** GetProjectedCost is called, **Then** default to gp2 storage type
3. **Given** an RDS resource with no storage_size specified, **When** GetProjectedCost is called, **Then** default to 20 GB storage

---

### User Story 3 - Multi-Engine Support (Priority: P3)

As a cloud cost analyst, I want cost estimates for different database engines (MySQL, PostgreSQL, MariaDB, Oracle, SQL Server), so that I can estimate costs across my diverse database portfolio.

**Why this priority**: Enterprise environments often use multiple database engines. Supporting all major engines ensures broad applicability.

**Independent Test**: Can be tested by querying costs for each supported engine type and verifying engine-specific pricing is returned.

**Acceptance Scenarios**:

1. **Given** an RDS resource with engine "postgres", **When** GetProjectedCost is called, **Then** return PostgreSQL-specific pricing
2. **Given** an RDS resource with engine "mariadb", **When** GetProjectedCost is called, **Then** return MariaDB-specific pricing
3. **Given** an RDS resource with engine "oracle-se2", **When** GetProjectedCost is called, **Then** return Oracle Standard Edition 2 pricing
4. **Given** an RDS resource with engine "sqlserver-ex", **When** GetProjectedCost is called, **Then** return SQL Server Express pricing

---

### User Story 4 - Supports Query for RDS (Priority: P1)

As a PulumiCost core system, I need to query whether this plugin supports RDS resources, so that I can route cost requests to the appropriate plugin.

**Why this priority**: Equal priority to P1 as Supports() is prerequisite for any cost estimation to occur.

**Independent Test**: Can be tested by calling Supports() with resource_type "rds" and verifying supported=true without "Limited support" caveat.

**Acceptance Scenarios**:

1. **Given** a resource descriptor with resource_type "rds" and matching region, **When** Supports is called, **Then** return supported=true with reason indicating full support
2. **Given** a resource descriptor with resource_type "rds" and non-matching region, **When** Supports is called, **Then** return supported=false with region mismatch reason

---

### Edge Cases

- What happens when an unknown instance type is provided?
  - Return $0 cost with explanatory billing_detail message
- What happens when an unsupported database engine is specified?
  - Default to MySQL pricing with note in billing_detail
- How does system handle invalid storage_size values (non-numeric, negative)?
  - Default to 20 GB with note in billing_detail
- What happens when storage_type is not recognized?
  - Default to gp2 with note in billing_detail
- How does system handle concurrent RDS pricing lookups?
  - Thread-safe access to pricing indexes ensures correct results

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST return accurate hourly rates for RDS instance types (db.t3.*, db.m5.*, db.r5.*, etc.)
- **FR-002**: System MUST support MySQL, PostgreSQL, MariaDB, Oracle Standard Edition 2, and SQL Server Express engines
- **FR-003**: System MUST calculate monthly instance cost as hourly_rate × 730 hours
- **FR-004**: System MUST calculate storage cost based on volume type (gp2, gp3, io1, magnetic) and size in GB
- **FR-005**: System MUST combine instance cost and storage cost for total monthly estimate
- **FR-006**: System MUST default to MySQL engine when engine tag is not provided
- **FR-007**: System MUST default to gp2 storage type when storage_type tag is not provided
- **FR-008**: System MUST default to 20 GB storage when storage_size tag is not provided or invalid
- **FR-009**: System MUST return $0 with explanatory message for unknown instance types
- **FR-010**: System MUST provide thread-safe pricing lookups for concurrent requests
- **FR-011**: Supports() MUST return supported=true without "Limited support" reason for RDS resources
- **FR-012**: System MUST include RDS pricing data in all 9 regional binaries
- **FR-013**: System MUST use Single-AZ deployment pricing only (Multi-AZ is out of scope)
- **FR-014**: System MUST return billing_detail describing all assumptions (instance type, engine, hours, storage)

### Key Entities

- **RDS Instance Price**: Represents hourly compute cost for a specific instance type and engine combination
- **RDS Storage Price**: Represents per-GB-month cost for a specific storage volume type
- **Resource Descriptor**: Input containing instance type (Sku), region, and tags (engine, storage_type, storage_size)
- **Projected Cost Response**: Output containing unit_price (hourly), currency, cost_per_month (total), and billing_detail

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Cost queries for common instance types (db.t3.micro, db.t3.medium, db.m5.large) return non-zero results within 100ms
- **SC-002**: Cost estimates are within 5% accuracy compared to AWS Pricing Calculator for the same configuration
- **SC-003**: All 9 regional binaries return accurate region-specific RDS pricing
- **SC-004**: System handles 100 concurrent RDS cost queries without errors or data corruption
- **SC-005**: 100% of unit tests pass covering instance types, engines, storage types, and default value scenarios
- **SC-006**: billing_detail message clearly communicates all pricing assumptions to users

## Scope Boundaries

### In Scope

- Single-AZ RDS instance pricing
- Instance compute hours (on-demand)
- General purpose and provisioned IOPS storage (per GB)
- MySQL, PostgreSQL, MariaDB, Oracle SE2, SQL Server Express engines
- All 9 supported AWS regions

### Out of Scope

- Multi-AZ deployment pricing
- Read replica pricing
- Reserved instance pricing
- Aurora and Aurora Serverless
- Provisioned IOPS (per IOPS) pricing
- Backup storage costs
- Data transfer costs
- License-included vs BYOL pricing variations beyond defaults

## Assumptions

- AWS public pricing API provides accurate, up-to-date RDS pricing data
- 730 hours per month is an acceptable standard for cost projection
- Default engine (MySQL) is the most common use case when unspecified
- Default storage type (gp2) and size (20 GB) represent typical minimum configurations
- Single-AZ deployment is sufficient for v1 cost estimation
- License-included pricing is used for Oracle and SQL Server (standard default)
