# Quickstart: ElastiCache Cost Estimation

**Feature**: 001-elasticache
**Date**: 2025-12-31

## Overview

This guide covers how to estimate costs for Amazon ElastiCache clusters using the FinFocus AWS Public plugin.

## Basic Usage

### Single Redis Node

**Request**:

```json
{
  "resource": {
    "provider": "aws",
    "resource_type": "elasticache",
    "sku": "cache.m5.large",
    "region": "us-east-1",
    "tags": {
      "engine": "redis"
    }
  }
}
```

**Response**:

```json
{
  "cost_per_month": 113.88,
  "unit_price": 0.156,
  "currency": "USD",
  "billing_detail": "ElastiCache cache.m5.large (redis), 1 node(s), 730 hrs/month"
}
```

### Multi-Node Redis Cluster

**Request**:

```json
{
  "resource": {
    "provider": "aws",
    "resource_type": "aws:elasticache/replicationGroup:ReplicationGroup",
    "sku": "cache.r6g.large",
    "region": "us-east-1",
    "tags": {
      "engine": "redis",
      "num_cache_clusters": "3"
    }
  }
}
```

**Response**:

```json
{
  "cost_per_month": 328.50,
  "unit_price": 0.15,
  "currency": "USD",
  "billing_detail": "ElastiCache cache.r6g.large (redis), 3 node(s), 730 hrs/month"
}
```

### Memcached Cluster

**Request**:

```json
{
  "resource": {
    "provider": "aws",
    "resource_type": "elasticache",
    "sku": "cache.m5.large",
    "region": "us-east-1",
    "tags": {
      "engine": "memcached",
      "num_nodes": "2"
    }
  }
}
```

**Response**:

```json
{
  "cost_per_month": 227.76,
  "unit_price": 0.156,
  "currency": "USD",
  "billing_detail": "ElastiCache cache.m5.large (memcached), 2 node(s), 730 hrs/month"
}
```

## Supported Resource Types

| Resource Type | Description |
|---------------|-------------|
| `elasticache` | Legacy simple format |
| `aws:elasticache/cluster:Cluster` | Pulumi single-node cluster |
| `aws:elasticache/replicationGroup:ReplicationGroup` | Pulumi multi-node replication group |

## Supported Engines

| Engine | Tag Value | Notes |
|--------|-----------|-------|
| Redis | `redis` | Default if not specified |
| Memcached | `memcached` | |
| Valkey | `valkey` | AWS-managed Redis alternative |

Engine names are case-insensitive (`redis`, `Redis`, `REDIS` all work).

## Tag Reference

| Tag | Purpose | Default |
|-----|---------|---------|
| `engine` | Cache engine type | `redis` |
| `num_cache_clusters` | Node count (Pulumi) | 1 |
| `num_nodes` | Node count (generic) | 1 |
| `nodes` | Node count (short) | 1 |

## Default Behavior

When parameters are omitted:
- Engine defaults to **Redis**
- Node count defaults to **1**
- Defaults are noted in `billing_detail`

**Example with defaults**:

```json
{
  "resource": {
    "provider": "aws",
    "resource_type": "elasticache",
    "sku": "cache.t3.micro",
    "region": "us-east-1"
  }
}
```

**Response**:

```json
{
  "cost_per_month": 12.41,
  "unit_price": 0.017,
  "currency": "USD",
  "billing_detail": "ElastiCache cache.t3.micro (redis), 1 node(s), 730 hrs/month, engine defaulted to redis, node count defaulted to 1"
}
```

## Error Handling

### Unknown Instance Type

**Request**:

```json
{
  "resource": {
    "resource_type": "elasticache",
    "sku": "cache.unknown.type",
    "region": "us-east-1"
  }
}
```

**Response** (graceful degradation):

```json
{
  "cost_per_month": 0,
  "unit_price": 0,
  "currency": "USD",
  "billing_detail": "ElastiCache instance type 'cache.unknown.type' not found in pricing data"
}
```

### Missing Instance Type

**Request**:

```json
{
  "resource": {
    "resource_type": "elasticache",
    "region": "us-east-1"
  }
}
```

**Response** (gRPC error):

```text
Error: ERROR_CODE_INVALID_RESOURCE
Message: "ElastiCache requires instance type in Sku or tags"
```

### Region Mismatch

If the request region doesn't match the plugin binary's region:

**Response** (gRPC error):

```text
Error: ERROR_CODE_UNSUPPORTED_REGION
Message: "Region 'eu-west-1' not supported by this binary (us-east-1)"
```

## Cost Calculation

**Formula**:

```text
cost_per_month = hourly_rate × num_nodes × 730 hours
```

**Example**:
- Instance: cache.m5.large
- Hourly rate: $0.156
- Nodes: 3
- Monthly cost: $0.156 × 3 × 730 = $341.64

## What's Included

- On-demand node hours
- All supported engines (Redis, Memcached, Valkey)
- Multi-node cluster calculations

## What's NOT Included

- Reserved Instance pricing
- Serverless ElastiCache (ECPU-based)
- Global Datastore data transfer
- Backup storage costs
- Data transfer costs

## Testing Locally

```bash
# Build us-east-1 binary
make build-region REGION=us-east-1

# Start plugin (captures PORT)
./bin/finfocus-plugin-aws-public-us-east-1

# In another terminal, use grpcurl:
grpcurl -plaintext -d '{
  "resource": {
    "provider": "aws",
    "resource_type": "elasticache",
    "sku": "cache.m5.large",
    "region": "us-east-1",
    "tags": {"engine": "redis"}
  }
}' localhost:<PORT> finfocus.v1.CostSourceService/GetProjectedCost
```
