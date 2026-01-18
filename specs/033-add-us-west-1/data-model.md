# Data Model - Add us-west-1 (N. California) Region Support

## Overview

This feature primarily extends existing enumerations and configuration files rather than introducing new complex data structures. The core data model remains the AWS Pricing JSON structure, which is consistent across all regions.

## 1. Configuration Entities

### Regions Configuration (`internal/pricing/regions.yaml`)

Extended enumeration to include `us-west-1`.

| Field | Type | New Value | Description |
| :--- | :--- | :--- | :--- |
| `id` | `string` | `usw1` | Short identifier for internal mapping. |
| `name` | `string` | `us-west-1` | Standard AWS region code. |
| `tag` | `string` | `region_usw1` | Go build tag for conditional compilation. |

## 2. Embedded Data Artifacts

### Regional Binary (`finfocus-plugin-aws-public-us-west-1`)

A new build artifact containing:

1.  **Pricing Data**: `data/aws_pricing_us-west-1.json` (Embedded)
    *   **Format**: JSON (Standard Pricing API Schema)
    *   **Services**: EC2, EBS, RDS, S3, Lambda, DynamoDB, ELB, etc.
    *   **Constraint**: Full dataset required (no filtering).

2.  **Carbon Data**: `internal/carbon/grid_factors.go`
    *   **Key**: `us-west-1`
    *   **Value**: `0.000322` (WECC grid factor)

## 3. Runtime Contracts (gRPC)

No changes to the `finfocus.v1.CostSourceService` protocol definition. The behavior changes are limited to:

*   **`Supports(ResourceDescriptor)`**: Returns `true` when `region="us-west-1"`.
*   **`GetProjectedCost(ResourceDescriptor)`**: Returns valid costs for `us-west-1` resources.
*   **Error Handling**:
    *   Returns `ERROR_CODE_UNSUPPORTED_REGION` if called with non-`us-west-1` region (standard behavior).
    *   Returns `ERROR_CODE_INVALID_RESOURCE` (mapped to `UnsupportedResource`) if resource type exists in AWS but not in `us-west-1`.

## 4. Docker Environment

### Environment Variables

| Variable | Scope | Change |
| :--- | :--- | :--- |
| `REGIONS` | `Dockerfile` | Append `us-west-1` to list. |
| `ports` | `entrypoint.sh` | Append `8010` to array. |
