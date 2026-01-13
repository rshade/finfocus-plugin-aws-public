# Feature Specification: FinFocus AWS Public Plugin

**Feature Branch**: `001-finfocus-aws-plugin`
**Created**: 2025-11-16
**Status**: Draft - Revised for gRPC Protocol
**Input**: User description: "FinFocus AWS Public Plugin - A fallback cost plugin for FinFocus that estimates AWS resource costs using public AWS on-demand pricing, without needing CUR/Cost Explorer/Vantage data"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Basic Cost Estimation via gRPC (Priority: P1)

A FinFocus user needs to estimate the monthly costs of their AWS infrastructure stack that includes EC2 instances and EBS volumes. The FinFocus core calls the plugin's gRPC service for each resource, and the plugin returns cost estimates using publicly available pricing data.

**Why this priority**: This is the core value proposition - providing immediate cost estimates for the most common AWS compute and storage resources via the standard gRPC protocol. Without this, the plugin has no value.

**Independent Test**: Can be fully tested by starting the plugin gRPC service, calling GetProjectedCost RPC with EC2 and EBS ResourceDescriptors, and validating the GetProjectedCostResponse messages contain accurate cost estimates.

**Acceptance Scenarios**:

1. **Given** the plugin gRPC service is running, **When** core calls GetProjectedCost with a t3.micro EC2 ResourceDescriptor in us-east-1, **Then** it returns a GetProjectedCostResponse with cost_per_month based on 730 hours of on-demand usage
2. **Given** the plugin gRPC service is running, **When** core calls GetProjectedCost with a 100GB gp3 EBS ResourceDescriptor, **Then** it returns a GetProjectedCostResponse with cost_per_month based on volume size and type
3. **Given** the plugin processes multiple sequential RPC calls, **When** core calls GetProjectedCost for 10 different resources, **Then** each call returns independently with correct per-resource costs

---

### User Story 2 - gRPC Service Lifecycle Management (Priority: P1)

The FinFocus core needs to start the plugin, discover its gRPC port, communicate via gRPC, and shut down gracefully. The plugin must announce its port and handle lifecycle signals properly.

**Why this priority**: This is foundational infrastructure. Without proper lifecycle management, the plugin cannot integrate with FinFocus core at all.

**Independent Test**: Can be fully tested by starting the plugin process, capturing PORT announcement on stdout, connecting a gRPC client, invoking methods, and verifying graceful shutdown on context cancellation.

**Acceptance Scenarios**:

1. **Given** the plugin is started with PORT environment variable set, **When** the plugin initializes, **Then** it writes "PORT=<port>" to stdout and begins serving gRPC on that port
2. **Given** the plugin is started without PORT environment variable, **When** the plugin initializes, **Then** it selects an ephemeral port, announces it, and serves on 127.0.0.1
3. **Given** the plugin is serving gRPC requests, **When** the context is cancelled, **Then** it performs graceful shutdown and stops accepting new requests
4. **Given** the plugin is running, **When** core connects a gRPC client to the announced port, **Then** all CostSourceService methods are available

---

### User Story 3 - Resource Support Detection (Priority: P2)

The FinFocus core needs to determine whether the plugin can provide cost estimates for a given resource type before attempting to call GetProjectedCost. The plugin implements the Supports() RPC to advertise its capabilities.

**Why this priority**: This enables efficient routing and prevents unnecessary RPC calls for unsupported resources. Critical for multi-plugin environments.

**Independent Test**: Can be fully tested by calling the Supports RPC with various ResourceDescriptors and validating the SupportsResponse indicates correct support status and reasons.

**Acceptance Scenarios**:

1. **Given** the plugin supports EC2, **When** core calls Supports with resource_type "ec2" and region "us-east-1", **Then** it returns supported=true
2. **Given** the plugin does not fully implement S3, **When** core calls Supports with resource_type "s3", **Then** it returns supported=true with reason "Limited support - returns $0 estimate"
3. **Given** a plugin binary for us-east-1, **When** core calls Supports with region "eu-west-1", **Then** it returns supported=false with reason "Region not supported by this binary"
4. **Given** core calls Supports for an unknown resource_type, **When** the resource is not in EC2/EBS/S3/Lambda/RDS/DynamoDB, **Then** it returns supported=false

---

### User Story 4 - Region-Specific Cost Estimation (Priority: P2)

A FinFocus user deploys resources across different AWS regions and needs accurate region-specific pricing. Each plugin binary embeds pricing for exactly one region and handles region mismatches gracefully via gRPC error responses.

**Why this priority**: AWS pricing varies significantly by region. This ensures cost estimates are accurate by distributing region-specific binaries.

**Independent Test**: Can be fully tested by starting region-specific plugin binaries and calling GetProjectedCost with matching and mismatching regions, validating correct pricing or ERROR_CODE_UNSUPPORTED_REGION responses.

**Acceptance Scenarios**:

1. **Given** a us-east-1 plugin binary is running, **When** core calls GetProjectedCost for a resource in us-east-1, **Then** it returns cost estimates using us-east-1 pricing
2. **Given** a us-east-1 plugin binary is running, **When** core calls GetProjectedCost for a resource in us-west-2, **Then** it returns a gRPC error with ErrorCode ERROR_CODE_UNSUPPORTED_REGION and details map containing pluginRegion="us-east-1" and requiredRegion="us-west-2"
3. **Given** a us-west-2 plugin binary, **When** Supports is called for us-west-2 resources, **Then** it returns supported=true

---

### User Story 5 - Stub Support for Additional AWS Services (Priority: P3)

A FinFocus user has a stack that includes S3 buckets, Lambda functions, RDS databases, or DynamoDB tables alongside EC2/EBS resources. The plugin acknowledges these resources via Supports() and returns $0 estimates via GetProjectedCost.

**Why this priority**: This provides a complete picture of the stack and sets user expectations about what is and isn't fully estimated. It prevents silent failures.

**Independent Test**: Can be fully tested by calling GetProjectedCost for S3, Lambda, RDS, and DynamoDB ResourceDescriptors and validating $0 cost_per_month with appropriate billing_detail messages.

**Acceptance Scenarios**:

1. **Given** the plugin receives a GetProjectedCost call for S3, **When** processing the request, **Then** it returns cost_per_month=0 with billing_detail="S3 cost estimation not implemented - returning $0"
2. **Given** GetProjectedCost calls for Lambda, RDS, and DynamoDB, **When** processed, **Then** each returns $0 with service-specific not-implemented billing_detail messages
3. **Given** a Supports call for S3/Lambda/RDS/DynamoDB, **When** checked, **Then** it returns supported=true with reason indicating limited support

---

### User Story 6 - Transparent Cost Breakdown via Pricing Spec (Priority: P2 - Optional for MVP)

A FinFocus user wants to understand how costs are calculated for each resource, including pricing rates, billing modes, and assumptions. The plugin optionally implements GetPricingSpec() to provide detailed pricing information.

**Why this priority**: Transparency builds trust and allows users to validate estimates against their expected usage patterns. Critical for adoption. **Note**: This is an enhancement feature that can be deferred to v2 if needed - the core cost estimation (US1) provides billing_detail which covers basic transparency.

**Independent Test**: Can be fully tested by calling GetPricingSpec RPC with EC2 and EBS ResourceDescriptors and validating the PricingSpec message includes rate_per_unit, billing_mode, and metric_hints.

**Acceptance Scenarios**:

1. **Given** the plugin implements GetPricingSpec, **When** core calls it for a t3.micro EC2 instance, **Then** it returns PricingSpec with billing_mode="per_hour", rate_per_unit=hourly_rate, and metric_hints for vcpu_hours
2. **Given** a GetPricingSpec call for EBS gp3, **When** processed, **Then** it returns PricingSpec with billing_mode="per_gb_month", rate_per_unit=GB_rate, and description of assumptions (Linux, shared tenancy)
3. **Given** GetPricingSpec is called for an unimplemented service, **When** processed, **Then** it returns PricingSpec with rate_per_unit=0 and billing_detail explaining not implemented

---

### Edge Cases

- What happens when a ResourceDescriptor lacks required fields (e.g., EC2 without sku/instance type)?
- How does the plugin handle ResourceDescriptors with unknown resource_type values?
- What happens when embedded pricing data is corrupted or fails to parse at initialization?
- How does the plugin handle extremely high concurrent RPC call volumes?
- What happens when an EBS ResourceDescriptor specifies no size in tags?
- How does the plugin handle unknown or new AWS instance types not in the pricing data?
- What happens if gRPC client calls GetProjectedCost for a resource the plugin returned supported=false for?
- How does the plugin handle context cancellation during an in-flight RPC call?

## Requirements *(mandatory)*

### Functional Requirements

**gRPC Service Implementation:**
- **FR-001**: Plugin MUST implement the CostSourceService gRPC interface from finfocus.v1 proto
- **FR-002**: Plugin MUST implement Name() RPC returning NameResponse with name="aws-public"
- **FR-003**: Plugin MUST implement Supports() RPC to indicate resource support based on resource_type and region
- **FR-004**: Plugin MUST implement GetProjectedCost() RPC to return cost estimates for a single resource
- **FR-005**: Plugin MAY implement GetPricingSpec() RPC to provide detailed pricing specifications
- **FR-006**: Plugin MUST NOT implement GetActualCost() RPC (not applicable for public pricing)

**Service Lifecycle:**
- **FR-007**: Plugin MUST announce its gRPC port by writing "PORT=<port>" to stdout on startup
- **FR-008**: Plugin MUST serve gRPC on loopback address (127.0.0.1) only
- **FR-009**: Plugin MUST use PORT environment variable when set, otherwise select ephemeral port
- **FR-010**: Plugin MUST perform graceful shutdown when context is cancelled
- **FR-011**: Plugin MUST use the pluginsdk.Serve() function from finfocus-core/pkg/pluginsdk
- **FR-012**: Plugin MUST register CostSourceService with the gRPC server

**Resource Support:**
- **FR-013**: Supports() MUST return supported=true for resource_type "ec2" in the plugin's region
- **FR-014**: Supports() MUST return supported=true for resource_type "ebs" in the plugin's region
- **FR-015**: Supports() MUST return supported=true for resource_type "s3", "lambda", "rds", "dynamodb" with reason indicating limited support
- **FR-016**: Supports() MUST return supported=false for resources in different regions than the plugin binary, with reason explaining region mismatch
- **FR-017**: Supports() MUST return supported=false for unknown resource types not in the supported list

**Cost Estimation (EC2 & EBS):**
- **FR-018**: GetProjectedCost() MUST calculate EC2 costs using ResourceDescriptor.sku (instance type) and region
- **FR-019**: GetProjectedCost() MUST calculate EBS costs using ResourceDescriptor.sku (volume type) and tags for volume size
- **FR-020**: GetProjectedCost() MUST use embedded region-specific pricing data for lookups
- **FR-021**: GetProjectedCost() MUST assume 730 hours/month for EC2 monthly cost calculation
- **FR-022**: GetProjectedCost() MUST return cost_per_month and unit_price in GetProjectedCostResponse
- **FR-023**: GetProjectedCost() MUST set currency="USD" for all responses
- **FR-024**: GetProjectedCost() MUST set billing_detail to describe assumptions (e.g., "On-demand Linux, shared tenancy, 730 hrs/month")

**Cost Estimation (Stub Services):**
- **FR-025**: GetProjectedCost() MUST return cost_per_month=0 for S3, Lambda, RDS, DynamoDB resource types
- **FR-026**: GetProjectedCost() MUST set billing_detail for stub services to explain not implemented (e.g., "S3 cost estimation not implemented")

**Error Handling:**
- **FR-027**: Plugin MUST return gRPC error with code ERROR_CODE_UNSUPPORTED_REGION when ResourceDescriptor.region does not match plugin region
- **FR-028**: ERROR_CODE_UNSUPPORTED_REGION errors MUST include ErrorDetail.details map with pluginRegion and requiredRegion keys
- **FR-029**: Plugin MUST return ERROR_CODE_INVALID_RESOURCE when ResourceDescriptor lacks required fields (provider, resource_type, sku, region)
- **FR-030**: Plugin MUST return ERROR_CODE_DATA_CORRUPTION when embedded pricing data is corrupted or unreadable
- **FR-031**: Plugin MUST use ErrorCode enum values from finfocus.v1.ErrorCode proto definition
- **FR-032**: Plugin MUST NOT define custom error codes outside the proto enum

**Pricing Data Management:**
- **FR-033**: Plugin MUST embed region-specific AWS pricing data at build time using go:embed
- **FR-034**: Plugin MUST be compiled as region-specific binaries (one binary per AWS region)
- **FR-035**: Plugin MUST use build tags to embed only the relevant region's pricing data in each binary
- **FR-036**: Plugin MUST parse and index embedded pricing data during initialization (before serving)
- **FR-037**: Plugin MUST support a build-time tool for fetching and trimming AWS pricing data from public APIs
- **FR-038**: Build-time pricing tool MUST support a dummy mode for development without AWS API access
- **FR-039**: Plugin MUST use GoReleaser to build region-specific binaries with embedded pricing data
- **FR-040**: Plugin MUST follow binary naming convention: finfocus-plugin-aws-public-{region}

**EBS Volume Defaults:**
- **FR-041**: GetProjectedCost() MUST check ResourceDescriptor.tags for "size" or "volume_size" when estimating EBS
- **FR-042**: GetProjectedCost() MUST default to 8GB when EBS size is not specified in tags
- **FR-043**: GetProjectedCost() MUST include the default size assumption in billing_detail when defaulted

**Observability:**
- **FR-044**: Plugin MAY implement ObservabilityService for health checks and metrics
- **FR-045**: Plugin SHOULD log diagnostic messages to stderr (never stdout except PORT announcement)
- **FR-046**: Plugin SHOULD log startup, pricing data load, and shutdown events

**Performance & Concurrency:**
- **FR-047**: Plugin MUST handle concurrent GetProjectedCost RPC calls safely (thread-safe pricing lookups)
- **FR-048**: Plugin MUST be stateless - each RPC call is independent with no persistent state between calls

### Key Entities

**Proto-Defined Types (from finfocus/v1/costsource.proto):**
- **ResourceDescriptor**: Input message containing provider, resource_type, sku, region, and tags for identifying a resource
- **GetProjectedCostRequest**: RPC request containing a ResourceDescriptor
- **GetProjectedCostResponse**: RPC response containing unit_price, currency, cost_per_month, and billing_detail
- **SupportsRequest**: RPC request with ResourceDescriptor to check support
- **SupportsResponse**: RPC response with supported boolean and reason string
- **PricingSpec**: Detailed pricing specification with billing_mode, rate_per_unit, metric_hints, and description
- **ErrorCode**: Enum of standard error codes including ERROR_CODE_UNSUPPORTED_REGION, ERROR_CODE_INVALID_RESOURCE, ERROR_CODE_DATA_CORRUPTION
- **ErrorDetail**: Structured error information with code, category, message, and details map

**Plugin-Specific Internal Types:**
- **PricingData**: Internal representation of embedded AWS pricing information per region, loaded at initialization
- **PricingIndex**: In-memory lookup structures for fast price queries by instance type and volume type

## Success Criteria *(mandatory)*

### Measurable Outcomes

**RPC Performance:**
- **SC-001**: GetProjectedCost RPC responds in under 100 milliseconds per call for EC2 and EBS resources
- **SC-002**: Supports RPC responds in under 10 milliseconds per call
- **SC-003**: Plugin initialization (pricing data load and parse) completes in under 500 milliseconds
- **SC-004**: Plugin handles at least 100 concurrent GetProjectedCost RPC calls without errors or degradation

**Cost Accuracy:**
- **SC-005**: EC2 cost estimates are within 5% of actual AWS on-demand pricing for standard instance types
- **SC-006**: EBS cost estimates are within 2% of actual AWS on-demand pricing for gp2, gp3, io1, io2 volume types

**Protocol Compliance:**
- **SC-007**: Plugin successfully passes gRPC health checks via ObservabilityService (if implemented)
- **SC-008**: All GetProjectedCost responses include non-empty billing_detail explaining assumptions
- **SC-009**: All UNSUPPORTED_REGION errors include pluginRegion and requiredRegion in details map
- **SC-010**: Plugin announces PORT within 1 second of startup and begins serving within 2 seconds total

**Build & Distribution:**
- **SC-011**: Plugin binary sizes remain under 10MB per region when including embedded pricing data
- **SC-012**: GoReleaser successfully builds binaries for at least 3 regions (us-east-1, us-west-2, eu-west-1)
- **SC-013**: Each region binary correctly rejects resources from other regions with ERROR_CODE_UNSUPPORTED_REGION

**Stub Services:**
- **SC-014**: All stub services (S3, Lambda, RDS, DynamoDB) return consistent $0 cost_per_month with explanatory billing_detail

## Assumptions

- FinFocus core uses the gRPC CostSourceService protocol defined in finfocus-spec
- FinFocus core invokes plugins as separate processes and connects via announced gRPC port
- FinFocus core handles orchestration of multiple region-specific binaries for multi-region stacks
- AWS provides public pricing API endpoints that are accessible without authentication (build-time only)
- AWS pricing data structure is consistent enough to be trimmed and parsed reliably
- Standard on-demand pricing is sufficient for v1 (no spot, reserved, or savings plan pricing)
- 730 hours per month (24x7 usage) is an acceptable default for EC2 cost estimation
- Linux operating system is an acceptable default for EC2 instances
- Shared tenancy is an acceptable default for EC2 instances
- 8GB is an acceptable default for EBS volume size when not specified in tags
- GoReleaser can handle build tags and multiple binary outputs for region-specific builds
- Embedded JSON pricing data can be efficiently parsed and indexed in memory
- ResourceDescriptor.sku contains the AWS instance type for EC2 (e.g., "t3.micro")
- ResourceDescriptor.sku contains the AWS volume type for EBS (e.g., "gp3")
- ResourceDescriptor.tags may contain "size" or "volume_size" for EBS volumes
- Binary size under 10MB per region is acceptable for distribution
- A single binary per region approach is acceptable for v1 (multi-region handled by core)
- The pluginsdk package from finfocus-core provides Serve() for lifecycle management

## Dependencies

- Go programming language and toolchain (version 1.25 or later)
- finfocus-spec repository for proto definitions
- finfocus-core/pkg/pluginsdk for plugin SDK and Serve() function
- gRPC and protobuf Go packages
- GoReleaser for multi-binary build orchestration
- AWS public pricing API access (build-time only, not runtime)
- Standard AWS resource type naming conventions (e.g., "ec2", "ebs", "s3")

## Out of Scope

- Real-time pricing updates (pricing is embedded at build time)
- AWS Spot instance pricing
- AWS Reserved instance pricing
- AWS Savings Plan pricing
- Custom discount rates or negotiated pricing (planned for future versions)
- Non-AWS cloud providers
- Historical cost data (GetActualCost RPC not implemented)
- Cost optimization recommendations
- Multi-region handling within a single binary (handled by FinFocus core calling region-specific binaries)
- Authentication or authorization (plugin serves on loopback only)
- Detailed S3 cost estimation including storage classes, requests, and data transfer
- Detailed Lambda cost estimation including invocations, duration, and memory
- Detailed RDS cost estimation including instance types, storage, IOPS, and backups
- Detailed DynamoDB cost estimation including read/write capacity units and storage
- Data transfer costs between regions, AZs, or to internet
- AWS support plan costs
- Taxes and fees
- TLS/mTLS for gRPC (serves on loopback)
- Plugin authentication to core (local process trust model)
- Batch processing of multiple resources in a single RPC call
