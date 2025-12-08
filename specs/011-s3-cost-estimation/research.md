# Research Findings: S3 Storage Cost Estimation

**Feature**: 011-s3-cost-estimation
**Date**: 2025-12-07
**Researcher**: speckit.plan

## Decision: AWS S3 Pricing API Integration

**What**: Use AWS Public Pricing API endpoint for S3 storage pricing data.

**Rationale**: The API provides official, up-to-date pricing data for all AWS services including S3. Using the public pricing API ensures accuracy and eliminates manual maintenance of pricing tables.

**Alternatives Considered**:
- Manual pricing tables: Rejected due to maintenance overhead and risk of stale data
- AWS Cost Explorer API: Rejected due to authentication requirements and runtime dependencies

## Decision: Storage Class Mapping

**What**: Map user-friendly SKUs to AWS API storageClass values as follows:
- "STANDARD" → "Standard"
- "STANDARD_IA" → "Standard - Infrequent Access"
- "ONEZONE_IA" → "One Zone - Infrequent Access"
- "GLACIER" → "Glacier Flexible Retrieval"
- "DEEP_ARCHIVE" → "Glacier Deep Archive"

**Rationale**: Provides intuitive SKU names for users while mapping to official AWS pricing API values. Underscore-separated format follows existing patterns in the codebase.

**Alternatives Considered**:
- Use AWS API values directly: Rejected due to complexity for users
- Custom naming scheme: Rejected to maintain consistency

## Decision: Pricing Data Filtering

**What**: Filter pricing products by:
- ProductFamily == "Storage"
- servicecode == "AmazonS3"
- Extract storageClass attribute for indexing

**Rationale**: S3 pricing API includes multiple product families (Storage, Data Transfer, etc.). Filtering ensures only storage pricing is indexed, avoiding confusion with request-based charges.

**Alternatives Considered**:
- Include all S3 products: Rejected due to scope (out of scope: request charges)
- Manual filtering by product name: Rejected as less reliable than structured attributes

## Decision: Thread Safety Pattern

**What**: Use sync.Once for pricing data initialization and sync.RWMutex for concurrent map access.

**Rationale**: Follows existing codebase patterns (EBS implementation) and constitution requirements for thread-safe concurrent RPC calls.

**Alternatives Considered**:
- Channel-based synchronization: Rejected due to complexity for read-heavy workload
- No synchronization: Rejected violates constitution thread safety requirement

## Decision: Error Handling for Unknown Storage Classes

**What**: Return $0 cost with explanatory BillingDetail for unknown storage classes.

**Rationale**: Graceful degradation allows the system to continue functioning while providing clear feedback. Follows existing patterns in the codebase.

**Alternatives Considered**:
- Return error: Rejected as it would break cost estimation flow
- Default to STANDARD pricing: Rejected as it could mislead users

## Decision: Size Tag Extraction

**What**: Extract size from tags["size"] as string, parse to float64 GB, default to 1.0 if missing or invalid.

**Rationale**: Flexible input handling, sensible default prevents zero-cost results, follows existing tag-based patterns in codebase.

**Alternatives Considered**:
- Require size tag: Rejected as too restrictive
- Use different tag name: Rejected to maintain consistency

## Decision: Logging Requirements

**What**: Debug log successful pricing lookups with trace_id, storage_class, region, unit_price.

**Rationale**: Constitution requires zerolog structured logging, performance monitoring via warnings if >50ms. Debug logs enable troubleshooting without verbosity in production.

**Alternatives Considered**:
- Info level logging: Rejected due to log volume
- No logging: Rejected violates constitution performance monitoring requirement

## Decision: Build Integration

**What**: Extend generate-pricing tool to fetch S3 pricing data alongside existing services.

**Rationale**: Consistent with existing build process, ensures all regions include S3 pricing data.

**Alternatives Considered**:
- Separate S3 pricing tool: Rejected due to duplication
- Manual pricing files: Rejected due to maintenance burden