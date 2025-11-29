# Data Model: Zerolog Structured Logging

**Date**: 2025-11-26
**Feature**: 005-zerolog-logging

## Log Entry Schema

### Base Fields (all log entries)

| Field            | Type     | Source          |
| ---------------- | -------- | --------------- |
| `time`           | ISO8601  | zerolog         |
| `level`          | string   | zerolog         |
| `message`        | string   | handler         |
| `plugin_name`    | string   | NewPluginLogger |
| `plugin_version` | string   | NewPluginLogger |

### Tracing Fields (gRPC requests)

| Field         | Type   | Source             |
| ------------- | ------ | ------------------ |
| `trace_id`    | string | TraceIDFromContext |
| `operation`   | string | handler            |
| `duration_ms` | int64  | handler            |

### Resource Fields (cost operations)

| Field           | Type   | Source             |
| --------------- | ------ | ------------------ |
| `resource_type` | string | ResourceDescriptor |
| `aws_service`   | string | handler            |
| `aws_region`    | string | ResourceDescriptor |
| `instance_type` | string | Sku field          |
| `storage_type`  | string | Sku field          |

### Cost Fields (GetProjectedCost, GetActualCost)

| Field          | Type    | Source   |
| -------------- | ------- | -------- |
| `cost_monthly` | float64 | response |
| `unit_price`   | float64 | response |
| `currency`     | string  | response |

### Error Fields (error responses)

| Field        | Type   | Source  |
| ------------ | ------ | ------- |
| `error`      | string | handler |
| `error_code` | string | handler |

## Log Entry Examples

### Startup Log

```json
{
  "time": "2025-11-26T10:00:00Z",
  "level": "info",
  "plugin_name": "aws-public",
  "plugin_version": "0.0.3",
  "aws_region": "us-east-1",
  "message": "plugin started"
}
```

### GetProjectedCost Success

```json
{
  "time": "2025-11-26T10:00:01Z",
  "level": "info",
  "plugin_name": "aws-public",
  "plugin_version": "0.0.3",
  "trace_id": "abc123-def456-ghi789",
  "operation": "GetProjectedCost",
  "resource_type": "ec2",
  "aws_service": "ec2",
  "aws_region": "us-east-1",
  "instance_type": "t3.micro",
  "cost_monthly": 7.59,
  "duration_ms": 2,
  "message": "cost calculated"
}
```

### Supports Request

```json
{
  "time": "2025-11-26T10:00:02Z",
  "level": "info",
  "plugin_name": "aws-public",
  "plugin_version": "0.0.3",
  "trace_id": "abc123-def456-ghi789",
  "operation": "Supports",
  "resource_type": "ec2",
  "aws_region": "us-east-1",
  "supported": true,
  "duration_ms": 0,
  "message": "resource support check"
}
```

### Debug: SKU Lookup

```json
{
  "time": "2025-11-26T10:00:01Z",
  "level": "debug",
  "plugin_name": "aws-public",
  "plugin_version": "0.0.3",
  "trace_id": "abc123-def456-ghi789",
  "operation": "GetProjectedCost",
  "resource_type": "ec2",
  "instance_type": "t3.micro",
  "unit_price": 0.0104,
  "message": "pricing lookup successful"
}
```

### Error: Region Mismatch

```json
{
  "time": "2025-11-26T10:00:03Z",
  "level": "error",
  "plugin_name": "aws-public",
  "plugin_version": "0.0.3",
  "trace_id": "abc123-def456-ghi789",
  "operation": "GetProjectedCost",
  "resource_type": "ec2",
  "aws_region": "eu-west-1",
  "error": "region not supported by this binary",
  "error_code": "ERROR_CODE_UNSUPPORTED_REGION",
  "duration_ms": 0,
  "message": "request failed"
}
```

## State Transitions

Not applicable - log entries are immutable records with no state machine.

## Validation Rules

1. **trace_id**: If empty from context, generate UUID before logging
2. **operation**: Must match gRPC method name exactly
3. **duration_ms**: Must be non-negative
4. **cost_monthly**: Must be non-negative (0 for stub services)
5. **level**: Must be valid zerolog level

## Relationships

```text
Log Entry
├── always has: time, level, message, plugin_name, plugin_version
├── gRPC requests add: trace_id, operation, duration_ms
├── cost operations add: resource_type, aws_*, cost fields
└── errors add: error, error_code
```
