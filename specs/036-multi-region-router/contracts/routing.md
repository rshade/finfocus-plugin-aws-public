# Routing Contract: Multi-Region Router

**Branch**: `036-multi-region-router` | **Date**: 2026-02-12

## Overview

The router implements the same `CostSourceService` gRPC/Connect interface as
region binaries. Clients cannot distinguish the router from a region binary —
the protocol is identical.

## RPC Routing Table

| RPC | Routing Strategy | Region Source |
|-----|-----------------|---------------|
| Name | **Local** — returns `"finfocus-plugin-aws-public"` | N/A |
| GetPluginInfo | **Local** — returns router metadata | N/A |
| Supports | **Delegate** to region child | `request.resource.region` |
| GetProjectedCost | **Delegate** to region child | `request.resource.region` |
| GetActualCost | **Delegate** to region child | Extracted from request tags/resource_id |
| EstimateCost | **Delegate** to region child | Extracted from request attributes |
| GetPricingSpec | **Delegate** to region child | `request.resource.region` |
| GetRecommendations | **Fan-out** by region, merge results | `resource.region` per item |
| DismissRecommendation | **Local** — returns Unimplemented | N/A |
| GetBudgets | **Local** — returns Unimplemented | N/A |
| DryRun | **Delegate** to first available child | Any available region |

## Delegation Flow

```text
1. Router receives RPC request
2. Extract region from ResourceDescriptor.Region
3. If region empty → return InvalidArgument
4. registry.GetOrLaunch(region):
   a. If child ready → return existing client
   b. If child idle → launch child process
      i.  exec.Command with FINFOCUS_PLUGIN_WEB_ENABLED=true
      ii. Read stdout for PORT=<port> (30s timeout)
      iii. Create pluginsdk.NewConnectClient("http://localhost:<port>")
      iv. Mark child ready
   c. If child unhealthy → restart (up to 3 retries)
   d. If no binary → download (if online) or return error (if offline)
5. Delegate RPC via child's pluginsdk.Client
6. Return response to caller
```

## Error Codes

| Condition | gRPC Code | Error Code | Details |
|-----------|-----------|------------|---------|
| Missing region in request | InvalidArgument | ERROR_CODE_INVALID_RESOURCE | "region is required" |
| Region binary not found (offline) | Unavailable | ERROR_CODE_UNSUPPORTED_REGION | Install command suggestion |
| Download failed | Unavailable | ERROR_CODE_UNSUPPORTED_REGION | "Failed to download region binary" |
| Checksum verification failed | Internal | ERROR_CODE_DATA_CORRUPTION | "SHA256 mismatch" |
| Child startup timeout | Unavailable | ERROR_CODE_UNSUPPORTED_REGION | "Child failed to start within 30s" |
| Child startup retries exhausted | Unavailable | ERROR_CODE_UNSUPPORTED_REGION | "Failed after 3 attempts" |
| Child RPC error | Pass-through | Pass-through | Original child error |

## GetRecommendations Fan-Out

```text
1. Group target_resources by resource.region
2. For each region group (in parallel):
   a. Get or launch child for region
   b. Send GetRecommendations with region-filtered resources
3. Merge all responses:
   a. Concatenate recommendations from all children
   b. If a region child fails: log warning, continue with others
4. Return merged response
```

## Trace ID Propagation

```text
Incoming request → Extract trace_id from gRPC metadata
                 → Set trace_id in Connect request header
                 → Child receives trace_id in incoming metadata
                 → Child logs with trace_id (standard behavior)
```

## Child Environment

Children are launched with these environment variables:

| Variable | Value | Purpose |
|----------|-------|---------|
| `FINFOCUS_PLUGIN_WEB_ENABLED` | `true` | Enable Connect protocol |
| `FINFOCUS_LOG_LEVEL` | Inherited | Match router log level |
| `PORT` | `0` | Ephemeral port (child announces its own) |
