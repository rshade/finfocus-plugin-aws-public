# Data Model: SDK Migration and Code Consolidation

**Feature**: 013-sdk-migration
**Date**: 2025-12-16

## Entity Definitions

### EC2Attributes

Extracted EC2 configuration for pricing lookups.

| Field | Type | Values | Default |
|-------|------|--------|---------|
| OS | string | "Linux", "Windows" | "Linux" |
| Tenancy | string | "Shared", "Dedicated", "Host" | "Shared" |

**Extraction Rules**:

- `platform` tag/attr: "windows" (case-insensitive) → Windows, else Linux
- `tenancy` tag/attr: "dedicated"/"host" (case-insensitive) → Dedicated/Host,
  else Shared

**Source Compatibility**:

- `map[string]string` (ResourceDescriptor.Tags)
- `*structpb.Struct` (Pulumi EstimateCost attributes)

### RegionConfig

AWS region configuration for build and embed generation.

| Field | Type | Example | Validation |
|-------|------|---------|------------|
| ID | string | "us-east-1" | Required, safe chars `[a-zA-Z0-9_-]+` |
| Name | string | "US East (N. Virginia)" | Required, safe chars |
| Tag | string | "region_use1" | Required, must equal `region_` + ID variant |

**Validation Rules**:

1. All fields required (non-empty)
2. Safe character pattern: `/^[a-zA-Z0-9_-]+$/`
3. Tag format: `region_` prefix + abbreviated ID
4. No duplicate IDs within config

**File Format** (regions.yaml):

```yaml
regions:
  - id: us-east-1
    name: US_East_N_Virginia
    tag: region_use1
```

### ARNComponents

Parsed components from AWS ARN string.

| Field | Type | Example | Notes |
|-------|------|---------|-------|
| Partition | string | "aws", "aws-cn", "aws-us-gov" | Required |
| Service | string | "ec2", "rds", "s3" | Required |
| Region | string | "us-east-1" | May be empty for global services |
| AccountID | string | "123456789012" | 12 digits or empty |
| ResourceType | string | "instance", "volume", "db" | Extracted from resource part |
| ResourceID | string | "i-1234567890abcdef0" | After resource type |

**ARN Format**:

```text
arn:partition:service:region:account:resource-type/resource-id
arn:partition:service:region:account:resource-type:resource-id
```

**Service Mapping** (ARN service → Pulumi resource_type):

| ARN Pattern | Pulumi ResourceType |
|-------------|---------------------|
| ec2:...:instance/* | aws:ec2/instance:Instance |
| ec2:...:volume/* | aws:ebs/volume:Volume |
| rds:...:db:* | aws:rds/instance:Instance |
| s3:::* | aws:s3/bucket:Bucket |
| lambda:...:function:* | aws:lambda/function:Function |
| dynamodb:...:table/* | aws:dynamodb/table:Table |
| eks:...:cluster/* | aws:eks/cluster:Cluster |

**Special Cases**:

- S3: Region may be empty (global namespace)
- EBS: Uses `ec2` service in ARN, not `ebs`

## State Transitions

### Request Validation Flow

```text
Request Received
     │
     ▼
┌────────────────┐
│ SDK Validation │ ─── Error ──► InvalidArgument + ErrorDetail
└────────────────┘
     │ Pass
     ▼
┌────────────────┐
│ Region Check   │ ─── Mismatch ──► UNSUPPORTED_REGION + details
└────────────────┘
     │ Pass
     ▼
┌────────────────┐
│ Service Check  │ ─── Unknown ──► UNSUPPORTED_RESOURCE
└────────────────┘
     │ Pass
     ▼
   Estimation
```

### ARN Resolution Flow

```text
GetActualCost Request
     │
     ▼
┌────────────────┐
│ ARN Present?   │ ─── No ──────────────────────────────────┐
└────────────────┘                                           │
     │ Yes                                                   │
     ▼                                                       │
┌────────────────┐                                           │
│ Parse ARN      │ ─── Invalid ──┐                           │
└────────────────┘                │                          │
     │ Valid                      │                          │
     ▼                            ▼                          │
┌────────────────┐        ┌────────────────┐                 │
│ Extract Region │        │ JSON ResourceId│ ─── Invalid ───┤
│ Map Service    │        │ Parsing        │                 │
│ Need SKU: Tags │        └────────────────┘                 │
└────────────────┘                │ Valid                    │
     │                            ▼                          │
     │                    ┌────────────────┐                 │
     │                    │ Use Parsed     │                 │
     │                    │ ResourceDescr  │                 │
     │                    └────────────────┘                 │
     │                            │                          │
     │◄───────────────────────────┘                          │
     │                                                       │
     │◄──────────────────────────────────────────────────────┤
     ▼                                                       │
┌────────────────┐                                           │
│ Tags Fallback  │◄──────────────────────────────────────────┘
│ Extraction     │
└────────────────┘
     │
     ▼
  ResourceDescriptor
```

## Relationships

```text
┌─────────────────────────────────────────────────────────────┐
│                      Plugin Runtime                         │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌─────────────┐       uses       ┌─────────────────────┐  │
│  │ projected.go│─────────────────►│ EC2Attributes       │  │
│  │ estimate.go │                  │ (ec2_attrs.go)      │  │
│  └─────────────┘                  └─────────────────────┘  │
│         │                                                   │
│         │ validates via                                     │
│         ▼                                                   │
│  ┌─────────────────────┐         ┌─────────────────────┐   │
│  │ validation.go       │◄────────│ pluginsdk           │   │
│  │ - SDK wrappers      │  wraps  │ - Validate*Request  │   │
│  │ - Region check      │         └─────────────────────┘   │
│  └─────────────────────┘                                   │
│         │                                                   │
│         │ creates errors                                    │
│         ▼                                                   │
│  ┌─────────────────────┐                                   │
│  │ ErrorDetail         │                                   │
│  │ (from proto)        │                                   │
│  └─────────────────────┘                                   │
│                                                             │
│  ┌─────────────┐       parses     ┌─────────────────────┐  │
│  │ actual.go   │─────────────────►│ ARNComponents       │  │
│  │             │                  │ (arn.go)            │  │
│  └─────────────┘                  └─────────────────────┘  │
│                                                             │
└─────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────┐
│                      Build Tools                            │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌─────────────────┐     imports   ┌─────────────────────┐ │
│  │ generate-embeds │──────────────►│ regionsconfig       │ │
│  └─────────────────┘               │ - RegionConfig      │ │
│                                    │ - Load()            │ │
│  ┌─────────────────┐     imports   │ - Validate()        │ │
│  │ generate-       │──────────────►│                     │ │
│  │ goreleaser      │               └─────────────────────┘ │
│  └─────────────────┘                         │             │
│                                              │ reads       │
│                                              ▼             │
│                                    ┌─────────────────────┐ │
│                                    │ regions.yaml        │ │
│                                    └─────────────────────┘ │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

## Validation Rules Summary

| Entity | Rule | Error |
|--------|------|-------|
| EC2Attributes | Platform normalization | N/A (always defaults) |
| EC2Attributes | Tenancy normalization | N/A (always defaults) |
| RegionConfig | ID required | "region missing id" |
| RegionConfig | Name required | "region {ID} missing name" |
| RegionConfig | Tag required | "region {ID} missing tag" |
| RegionConfig | Safe characters | "contains invalid characters" |
| RegionConfig | Tag format | "tag mismatch: expected X, got Y" |
| RegionConfig | Unique IDs | "duplicate region id: X" |
| ARNComponents | Minimum 6 parts | "invalid ARN format" |
| ARNComponents | Valid partition | "unknown partition" |
| ARNComponents | Service mapping exists | Falls back to raw service |
