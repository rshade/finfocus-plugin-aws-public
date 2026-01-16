# Data Model: Multi-Region Docker Image

## Entities

### Docker Image
The primary artifact delivered by this feature.

| Attribute | Type | Description | Constraint |
|-----------|------|-------------|------------|
| Base Image | String | `alpine:3.19` | FR-001 |
| User | User | `plugin` (UID 65532) | FR-007 |
| Size | Size | ~2.0 GB | SC-001 |
| Entrypoint | Script | `/entrypoint.sh` | FR-005 |

### Regional Binary Process
12 instances running inside the container.

| Attribute | Type | Description |
|-----------|------|-------------|
| Name | String | `finfocus-plugin-aws-public-<region>` |
| Region | String | AWS Region (e.g., `us-east-1`) |
| Port | Int | 8001 - 8012 |
| Protocol | String | HTTP (Web Enabled) |

### Environment Variables
Variables passed to the container and propagated to all binaries.

| Variable | Default | Description |
|----------|---------|-------------|
| `FINFOCUS_PLUGIN_WEB_ENABLED` | `true` | Enables HTTP server |
| `FINFOCUS_PLUGIN_HEALTH_ENDPOINT` | `true` | Enables health check |
| `FINFOCUS_LOG_LEVEL` | `info` | Logging verbosity |

## Ports Mapping

| Region | Port |
|--------|------|
| us-east-1 | 8001 |
| us-west-2 | 8002 |
| eu-west-1 | 8003 |
| ap-southeast-1 | 8004 |
| ap-southeast-2 | 8005 |
| ap-northeast-1 | 8006 |
| ap-south-1 | 8007 |
| ca-central-1 | 8008 |
| sa-east-1 | 8009 |
| us-gov-west-1 | 8010 |
| us-gov-east-1 | 8011 |
| us-west-1 | 8012 |
| **Metrics** | **9090** |
