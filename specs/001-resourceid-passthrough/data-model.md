# Data Model: Resource ID Passthrough

**Feature**: 001-resourceid-passthrough
**Date**: 2025-12-26

## Overview

This feature modifies how data flows through the `GetRecommendations` handler, specifically the correlation information between input resources and output recommendations.

## Entities

### ResourceDescriptor (Input - from proto)

The input resource descriptor from pulumicost-spec v0.4.11+.

| Field | Type | Description | New/Existing |
|-------|------|-------------|--------------|
| `id` | string | Unique resource identifier (e.g., Pulumi URN) | **NEW in v0.4.11** |
| `provider` | string | Cloud provider (e.g., "aws") | Existing |
| `resource_type` | string | Resource type (e.g., "ec2", "ebs") | Existing |
| `sku` | string | SKU/instance type (e.g., "t3.micro") | Existing |
| `region` | string | AWS region (e.g., "us-east-1") | Existing |
| `tags` | map[string]string | Resource tags including `resource_id`, `name` | Existing |

### ResourceRecommendationInfo (Output - from proto)

The resource info embedded in each recommendation response.

| Field | Type | Description | Usage |
|-------|------|-------------|-------|
| `id` | string | Populated from input `ResourceDescriptor.id` or `tags["resource_id"]` | **Modified logic** |
| `name` | string | Populated from `tags["name"]` | Unchanged |
| `provider` | string | Always "aws" for this plugin | Unchanged |
| `resource_type` | string | Service type (ec2, ebs, etc.) | Unchanged |
| `region` | string | AWS region | Unchanged |
| `sku` | string | Instance/volume type | Unchanged |

## Data Flow

```text
┌─────────────────────────────────────────────────────────────────┐
│                    GetRecommendationsRequest                     │
├─────────────────────────────────────────────────────────────────┤
│  TargetResources: []*ResourceDescriptor                         │
│    ├── Id: "urn:pulumi:stack::project::aws:ec2:myserver"  ◄─NEW │
│    ├── Provider: "aws"                                          │
│    ├── ResourceType: "ec2"                                      │
│    ├── Sku: "m5.large"                                          │
│    ├── Region: "us-east-1"                                      │
│    └── Tags: {"resource_id": "legacy-id", "name": "MyServer"}   │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                    ID Resolution Logic                           │
├─────────────────────────────────────────────────────────────────┤
│  1. Check ResourceDescriptor.Id (trim whitespace)               │
│     - If non-empty: use as Resource.Id                          │
│  2. Else check tags["resource_id"]                              │
│     - If present: use as Resource.Id                            │
│  3. Else: Resource.Id remains empty                             │
│                                                                  │
│  Always: Resource.Name = tags["name"] (unchanged)               │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                    GetRecommendationsResponse                    │
├─────────────────────────────────────────────────────────────────┤
│  Recommendations: []*Recommendation                              │
│    └── Resource: *ResourceRecommendationInfo                     │
│          ├── Id: "urn:pulumi:stack::project::aws:ec2:myserver"  │
│          ├── Name: "MyServer"                                    │
│          ├── Provider: "aws"                                     │
│          ├── ResourceType: "ec2"                                 │
│          ├── Region: "us-east-1"                                 │
│          └── Sku: "m5.large"                                     │
└─────────────────────────────────────────────────────────────────┘
```

## Validation Rules

| Rule | Implementation |
|------|----------------|
| Empty native ID | Fall back to `tags["resource_id"]` |
| Whitespace-only native ID | Treat as empty, fall back to tag |
| Both ID sources present | Native ID takes priority |
| Neither ID source present | `Resource.Id` remains empty string |
| Multiple recommendations per resource | Each recommendation gets same ID |

## State Transitions

N/A - This is a stateless transformation. No persistent state changes.

## Backward Compatibility

| Scenario | Before (v0.4.10) | After (v0.4.11+) |
|----------|------------------|------------------|
| Native ID populated | N/A (field didn't exist) | Uses native ID |
| Only tag present | Uses `tags["resource_id"]` | Uses `tags["resource_id"]` (unchanged) |
| Neither present | `Resource.Id` empty | `Resource.Id` empty (unchanged) |
