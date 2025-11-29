# Quickstart: Zerolog Structured Logging

**Date**: 2025-11-26
**Feature**: 005-zerolog-logging

## Overview

This guide explains how to integrate zerolog structured logging with trace
propagation into the pulumicost-plugin-aws-public plugin.

## Prerequisites

- Go 1.25+
- pulumicost-spec v0.3.0+ with SDK logging utilities
- zerolog v1.34.0+

## Step 1: Add Dependencies

Update `go.mod`:

```go
require (
    github.com/rs/zerolog v1.34.0
    github.com/google/uuid v1.6.0
)
```

Run:

```bash
go mod tidy
```

## Step 2: Initialize Logger in main.go

```go
package main

import (
    "os"

    "github.com/rs/zerolog"
    "github.com/rshade/pulumicost-spec/sdk/go/pluginsdk"
)

const version = "0.0.3"

func main() {
    // Parse log level from environment
    level := zerolog.InfoLevel
    if lvl := os.Getenv("LOG_LEVEL"); lvl != "" {
        if parsed, err := zerolog.ParseLevel(lvl); err == nil {
            level = parsed
        }
    }

    // Create logger using SDK utility
    logger := pluginsdk.NewPluginLogger("aws-public", version, level, nil)

    // Log startup
    logger.Info().
        Str("aws_region", region).
        Msg("plugin started")

    // Pass logger to plugin
    awsPlugin := plugin.NewAWSPublicPlugin(region, pricingClient, logger)
    // ...
}
```

## Step 3: Add Logger Field to Plugin

```go
package plugin

import (
    "github.com/rs/zerolog"
    "github.com/rshade/pulumicost-spec/sdk/go/pluginsdk"
)

type AWSPublicPlugin struct {
    region  string
    pricing pricing.PricingClient
    logger  zerolog.Logger
}

func NewAWSPublicPlugin(region string, pc pricing.PricingClient,
    logger zerolog.Logger) *AWSPublicPlugin {
    return &AWSPublicPlugin{
        region:  region,
        pricing: pc,
        logger:  logger,
    }
}
```

## Step 4: Instrument Handlers

### GetProjectedCost

```go
func (p *AWSPublicPlugin) GetProjectedCost(ctx context.Context,
    req *pbc.GetProjectedCostRequest) (*pbc.GetProjectedCostResponse, error) {

    start := time.Now()
    traceID := p.getTraceID(ctx)

    p.logger.Debug().
        Str(pluginsdk.FieldTraceID, traceID).
        Str(pluginsdk.FieldOperation, "GetProjectedCost").
        Str(pluginsdk.FieldResourceType, req.Resource.ResourceType).
        Msg("processing request")

    // ... existing logic ...

    p.logger.Info().
        Str(pluginsdk.FieldTraceID, traceID).
        Str(pluginsdk.FieldOperation, "GetProjectedCost").
        Str(pluginsdk.FieldResourceType, req.Resource.ResourceType).
        Float64(pluginsdk.FieldCostMonthly, resp.CostPerMonth).
        Int64(pluginsdk.FieldDurationMs, time.Since(start).Milliseconds()).
        Msg("cost calculated")

    return resp, nil
}
```

### Error Logging

```go
func (p *AWSPublicPlugin) logError(ctx context.Context, op string,
    err error, code pbc.ErrorCode) {

    traceID := p.getTraceID(ctx)
    p.logger.Error().
        Str(pluginsdk.FieldTraceID, traceID).
        Str(pluginsdk.FieldOperation, op).
        Str(pluginsdk.FieldErrorCode, code.String()).
        Err(err).
        Msg("request failed")
}
```

### trace_id Helper

```go
func (p *AWSPublicPlugin) getTraceID(ctx context.Context) string {
    traceID := pluginsdk.TraceIDFromContext(ctx)
    if traceID == "" {
        traceID = uuid.New().String()
    }
    return traceID
}
```

## Step 5: Register Interceptor

**Note**: This requires pluginsdk.ServeConfig to support interceptors. If not
available, see research.md for alternatives.

```go
config := pluginsdk.ServeConfig{
    Plugin: awsPlugin,
    Port:   0,
    UnaryInterceptors: []grpc.UnaryServerInterceptor{
        pluginsdk.TracingUnaryServerInterceptor(),
    },
}
```

## Step 6: Run Tests

```bash
# Run all tests
make test

# Run with verbose logging
LOG_LEVEL=debug go test ./internal/plugin/... -v

# Run benchmarks
go test -bench=. ./internal/plugin/...
```

## Step 7: Verify Output

Start plugin with debug logging:

```bash
LOG_LEVEL=debug ./pulumicost-plugin-aws-public-us-east-1 2>&1 | jq .
```

Expected output format:

```json
{"time":"...","level":"info","plugin_name":"aws-public","plugin_version":"0.0.3",
 "aws_region":"us-east-1","message":"plugin started"}
```

## Environment Variables

| Variable    | Default  | Description                                         |
| ----------- | -------- | --------------------------------------------------- |
| `LOG_LEVEL` | `info`   | Minimum log level (trace, debug, info, warn, error) |
| `PORT`      | random   | gRPC server port                                    |

## Troubleshooting

### Logs not appearing

- Check stderr is not redirected
- Verify LOG_LEVEL is not set to a higher level than expected

### Missing trace_id

- Ensure TracingUnaryServerInterceptor is registered
- Check pulumicost-core is sending x-pulumicost-trace-id header

### Performance issues

- Use info level in production (debug is verbose)
- Verify benchmarks show <1ms overhead
