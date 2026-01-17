# Research: Amazon ECS Deployment

**Feature**: Add Amazon ECS Deployment Example
**Status**: Research Complete

## 1. Multi-Region Docker Image Analysis

**Findings**:
- The image `ghcr.io/rshade/finfocus-plugin-aws-public:latest` runs **all 12 regional binaries** simultaneously + 1 metrics aggregator.
- **Ports Exposed**: 8001-8012 (Regions) and 9090 (Metrics).
- **Architecture**: Single container, multiple processes managed by `entrypoint.sh`.
- **Health Check**: Built-in `/healthcheck.sh` verifies all 12 regions.

**Resource Sizing Decision**:
- **Recommendation**: 2 vCPU, 4 GB Memory.
- **Rationale**:
  - Base overhead: ~1.8 GB (12 regions × ~150MB static embedded data).
  - Runtime overhead: Active gRPC/HTTP handling requires buffer.
  - Metrics aggregator: Low overhead.
  - 4 GB provides ~100% headroom over static data for runtime allocations.
  - 2 vCPU ensures parallel startup and handling of concurrent requests without throttling.

## 2. Networking Strategy

**Decision**: Use **AWS Cloud Map (Service Discovery)** with Private DNS Namespace and **A Records**.
- **Rationale**:
  - The "Multi-Region" nature exposes 12 ports on a single IP.
  - A Load Balancer (ALB/NLB) would require 12 Listeners and 12 Target Groups (one per port), which is complex to manage and cost-inefficient.
  - **A Records** resolve to the Task IP. Clients can simply address `service.namespace:8001`, `service.namespace:8002`, etc.
  - This is the simplest, most cost-effective approach for internal VPC traffic.

**Bind Address Verification (Assumption)**:
- **Assumption**: The Docker image binaries bind to `0.0.0.0` or the container environment allows external access to mapped ports.
- **Evidence**: `specs/032-multi-region-docker` success criteria requires external reachability.
- **Action**: Documentation will assume standard port mapping works.

## 3. ECS Task Definition Configuration

**Key Settings**:
- **Launch Type**: Fargate.
- **Network Mode**: `awsvpc` (Required for Fargate).
- **Health Check**: Delegate to Docker `HEALTHCHECK` (CMD `/healthcheck.sh`) or replicate in Task Definition.
- **Log Driver**: `awslogs` (CloudWatch Logs) with `awslogs-stream-prefix=plugin`.
- **Environment Variables**:
  - `FINFOCUS_PLUGIN_WEB_ENABLED=true` (Default in image, but good to be explicit).
  - `FINFOCUS_LOG_LEVEL=info`.
  - `FINFOCUS_CORS_ALLOWED_ORIGINS=*` (or specific).

## 4. Documentation Structure

**Decision**: Create a single comprehensive guide `docs/ecs-deployment.md` linked from `README.md`.
- **Sections**:
  1.  **Architecture**: Diagram of Service Discovery + Multi-port container.
  2.  **Prerequisites**: VPC, Security Groups.
  3.  **Quick Start**: Terraform/CloudFormation snippet or CLI commands.
  4.  **Reference**: Full Task Definition JSON.
  5.  **Troubleshooting**: Common errors (Security Groups, IAM).

**Alternatives Considered**:
- **Separate file per strategy (ALB vs Service Discovery)**: Rejected. Service Discovery is the clear winner; others should be mentioned as "Alternatives" but not fully documented to avoid confusion.
