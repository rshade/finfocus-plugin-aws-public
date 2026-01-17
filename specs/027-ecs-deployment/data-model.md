# Data Model: Amazon ECS Deployment

**Feature**: Add Amazon ECS Deployment Example

## Configuration Entities

### 1. ECS Task Definition Structure
The JSON structure required to deploy the plugin.

| Field | Value / Type | Description |
|-------|--------------|-------------|
| `family` | `finfocus-plugin-aws-public` | Task family name |
| `networkMode` | `awsvpc` | Required for Fargate |
| `cpu` | `1024` (1 vCPU) or `2048` (2 vCPU) | Recommended: 2 vCPU |
| `memory` | `4096` (4 GB) | Recommended: 4 GB |
| `executionRoleArn` | String (ARN) | Role for pulling image/logging |
| `taskRoleArn` | String (ARN) | Role for plugin permissions (optional) |

### 2. Container Definition
Inside the Task Definition.

| Field | Value / Type | Description |
|-------|--------------|-------------|
| `image` | `ghcr.io/rshade/finfocus-plugin-aws-public:latest` | Multi-region image |
| `portMappings` | Array of 12 ports | 8001-8012 (TCP) |
| `healthCheck` | Command: `["CMD-SHELL", "/healthcheck.sh"]` | Uses image's built-in check |

### 3. Environment Variables
Configuration passed to the container.

| Variable | Default | Description |
|----------|---------|-------------|
| `FINFOCUS_PLUGIN_WEB_ENABLED` | `true` | Enables HTTP server (Required for ECS) |
| `FINFOCUS_PLUGIN_HEALTH_ENDPOINT` | `true` | Enables /healthz (Required for healthcheck) |
| `FINFOCUS_LOG_LEVEL` | `info` | Logging verbosity |
| `FINFOCUS_CORS_ALLOWED_ORIGINS` | `*` | CORS configuration |

## Network Model

### 1. Service Discovery (Cloud Map)
| Entity | Type | Configuration |
|--------|------|---------------|
| Namespace | Private DNS | e.g., `finfocus.local` |
| Service | Service | e.g., `aws-public` |
| DNS Record | A | Resolves to Task IP |
| TTL | 60 | Standard TTL |

### 2. Security Group
| Type | Protocol | Port Range | Source |
|------|----------|------------|--------|
| Inbound | TCP | 8001-8012 | VPC CIDR or Client SG |
| Inbound | TCP | 9090 | Monitoring SG (Optional) |
