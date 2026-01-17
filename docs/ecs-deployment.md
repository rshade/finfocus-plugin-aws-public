# Amazon ECS Deployment Guide

## Introduction

This guide provides a reference configuration for deploying the multi-region **finfocus-plugin-aws-public** Docker image to Amazon ECS using Fargate. The multi-region image (`ghcr.io/rshade/finfocus-plugin-aws-public:latest`) runs 12 separate regional plugin processes (plus a metrics aggregator) within a single container, exposing ports 8001-8012.

## Architecture

Deploying this plugin requires a **Service Discovery** approach rather than a traditional Load Balancer, because the container exposes 12 distinct ports (one for each supported AWS region).

**Key Components:**
- **ECS Service (Fargate)**: Runs the multi-region container.
- **AWS Cloud Map (Service Discovery)**: Creates `A` records in a private VPC DNS namespace (e.g., `finfocus.local`), allowing other services to reach the plugin via `plugin-name.namespace:8001`.
- **Security Group**: Allows inbound TCP traffic on ports 8001-8012 from your VPC CIDR.

**Architecture Flow:**

```text
Request -> Service Discovery (DNS) -> Task IP:8001 (us-east-1)
Request -> Service Discovery (DNS) -> Task IP:8002 (us-west-2)
...
```

> **Note:** A visual architecture diagram is planned for a future update.

## ECS Task Definition Reference

Use the following JSON structure to register your ECS Task Definition. This configuration enables all 12 regional ports and the metrics port.

### Resource Sizing

**Recommended Configuration:**
- **CPU**: 2048 (2 vCPU)
- **Memory**: 4096 (4 GB)

**Rationale:**
The container runs 12 concurrent processes (one per region). Each process loads approximately 150MB of embedded pricing data into memory.
- **Base Memory Overhead**: ~1.8 GB (12 regions × 150MB)
- **Runtime Buffer**: 4 GB provides sufficient headroom for gRPC request handling and safe garbage collection without OOM kills.
- **CPU**: 2 vCPU ensures that multiple regions can start up in parallel and handle concurrent cost estimation requests without throttling.

### Task Definition JSON

```json
{
  "family": "finfocus-plugin-aws-public",
  "networkMode": "awsvpc",
  "requiresCompatibilities": ["FARGATE"],
  "cpu": "2048",
  "memory": "4096",
  "executionRoleArn": "arn:aws:iam::ACCOUNT_ID:role/ecsTaskExecutionRole",
  "containerDefinitions": [
    {
      "name": "plugin",
      "image": "ghcr.io/rshade/finfocus-plugin-aws-public:latest",
      "essential": true,
      "portMappings": [
        {"containerPort": 8001, "protocol": "tcp"},
        {"containerPort": 8002, "protocol": "tcp"},
        {"containerPort": 8003, "protocol": "tcp"},
        {"containerPort": 8004, "protocol": "tcp"},
        {"containerPort": 8005, "protocol": "tcp"},
        {"containerPort": 8006, "protocol": "tcp"},
        {"containerPort": 8007, "protocol": "tcp"},
        {"containerPort": 8008, "protocol": "tcp"},
        {"containerPort": 8009, "protocol": "tcp"},
        {"containerPort": 8010, "protocol": "tcp"},
        {"containerPort": 8011, "protocol": "tcp"},
        {"containerPort": 8012, "protocol": "tcp"},
        {"containerPort": 9090, "protocol": "tcp"}
      ],
      "healthCheck": {
        "command": ["CMD-SHELL", "/healthcheck.sh"],
        "interval": 30,
        "timeout": 5,
        "retries": 3,
        "startPeriod": 30
      },
      "logConfiguration": {
        "logDriver": "awslogs",
        "options": {
          "awslogs-group": "/ecs/finfocus-plugin",
          "awslogs-region": "us-east-1",
          "awslogs-stream-prefix": "ecs"
        }
      },
      "environment": [
        {"name": "FINFOCUS_PLUGIN_WEB_ENABLED", "value": "true"},
        {"name": "FINFOCUS_LOG_LEVEL", "value": "info"}
      ]
    }
  ]
}
```

## Networking Strategy

### Why Service Discovery?

Using **AWS Cloud Map (Service Discovery)** is recommended over Application Load Balancers (ALB) or Network Load Balancers (NLB) for this specific use case.

| Feature | Service Discovery (Recommended) | Application Load Balancer (ALB) | Network Load Balancer (NLB) |
|---------|---------------------------------|---------------------------------|-----------------------------|
| **Cost** | Low (Route53 pricing) | High (Hourly + Data Processing) | High (Hourly + LCU) |
| **Complexity** | Low (Single A record) | High (12 Listeners + 12 Target Groups) | High (12 Listeners + 12 Target Groups) |
| **Limits** | High scaling limits | 50 Listeners per ALB (consumes 12) | 50 Listeners per NLB (consumes 12) |
| **Resolution** | Direct Task IP | Via Load Balancer DNS | Via Load Balancer DNS |

Since the plugin exposes 12 distinct ports on a single IP, an ALB/NLB setup would require creating and managing **12 separate listeners and target groups**, which significantly increases infrastructure complexity and cost. Service Discovery allows clients to resolve the Task IP directly via DNS and connect to any of the exposed ports.

### Accessing the Plugin

Once deployed with Service Discovery enabled (e.g., namespace `finfocus.local`, service `plugin`), the plugin is accessible to other services in your VPC via the following internal DNS addresses:

- **us-east-1**: `http://plugin.finfocus.local:8001`
- **us-west-2**: `http://plugin.finfocus.local:8002`
- **eu-west-1**: `http://plugin.finfocus.local:8003`
- ...and so on for all 12 regions.

You do not need to configure separate DNS records for each port; the single `A` record resolves to the container's private IP, and all ports are available on that IP.

## Environment Variables

The container supports the following environment variables. These are applied globally to all 12 regional processes.

| Variable | Default | Description |
|----------|---------|-------------|
| `FINFOCUS_PLUGIN_WEB_ENABLED` | `true` | Enables the HTTP server for web UI and health checks. **Required** for ECS deployments. |
| `FINFOCUS_PLUGIN_HEALTH_ENDPOINT` | `true` | Enables the `/healthz` endpoint. **Required** for the container health check script. |
| `FINFOCUS_LOG_LEVEL` | `info` | Controls logging verbosity. Options: `debug`, `info`, `warn`, `error`. |
| `FINFOCUS_CORS_ALLOWED_ORIGINS` | `*` | Configures CORS for the web server. Set to specific domains (e.g., `https://app.finfocus.io`) in production. |

## Prerequisites

Before deploying, ensure you have the following AWS resources ready:

- [ ] **VPC**: A VPC with private subnets (public subnets are not recommended for internal services).
- [ ] **Security Group**: Allow inbound traffic on TCP ports **8001-8012** from your VPC CIDR (or specific client security groups).
- [ ] **IAM Execution Role**: An IAM role with `AmazonECSTaskExecutionRolePolicy` attached (allows ECS to pull images and write logs).
- [ ] **Cloud Map Namespace**: A Private DNS Namespace (e.g., `finfocus.local`) created in AWS Cloud Map (Service Discovery).

## Troubleshooting

### Service Discovery Not Resolving

**Symptom**: `Could not resolve host: plugin.finfocus.local`
- **Check**: Ensure the ECS Service has successfully registered the task. Look at the "Service Discovery" tab in the ECS Console.
- **Check**: Verify your VPC has `enableDnsHostnames` and `enableDnsSupport` set to `true`.
- **Check**: Ensure the client service is in the same VPC or has VPC Peering/Transit Gateway access.

### Ports Not Accessible

**Symptom**: Connection timed out to port 8001.
- **Check**: Verify Security Group rules allow inbound TCP 8001-8012.
- **Check**: Ensure the task is running in a subnet with routes to/from the client.

### Container Fails to Start

**Symptom**: Task stops immediately.
- **Inspect**: CloudWatch Logs for error messages and startup failures.
- **Increase**: Task memory allocation to 4 GB or more if logs show OOM (Out of Memory) errors.
- **Confirm**: IAM Execution Role has permissions to pull from GHCR (if using a private repo, though this image is public).

## Terraform Example

This snippet demonstrates how to deploy the plugin with Service Discovery using Terraform.

```hcl
resource "aws_service_discovery_private_dns_namespace" "main" {
  name        = "finfocus.local"
  description = "Internal service discovery namespace"
  vpc         = aws_vpc.main.id
}

resource "aws_service_discovery_service" "plugin" {
  name = "aws-public"

  dns_config {
    namespace_id = aws_service_discovery_private_dns_namespace.main.id

    dns_records {
      ttl  = 60
      type = "A"
    }

    routing_policy = "MULTIVALUE"
  }

  health_check_custom_config {
    failure_threshold = 1
  }
}

resource "aws_ecs_task_definition" "plugin" {
  family                   = "finfocus-plugin-aws-public"
  network_mode             = "awsvpc"
  requires_compatibilities = ["FARGATE"]
  cpu                      = 2048
  memory                   = 4096
  execution_role_arn       = aws_iam_role.execution.arn

  container_definitions = jsonencode([
    {
      name      = "plugin"
      image     = "ghcr.io/rshade/finfocus-plugin-aws-public:latest"
      essential = true
      portMappings = [
        for port in range(8001, 8013) : {
          containerPort = port
          protocol      = "tcp"
        }
      ]
      healthCheck = {
        command     = ["CMD-SHELL", "/healthcheck.sh"]
        interval    = 30
        timeout     = 5
        retries     = 3
        startPeriod = 30
      }
      logConfiguration = {
        logDriver = "awslogs"
        options = {
          "awslogs-group"         = "/ecs/finfocus-plugin"
          "awslogs-region"        = "us-east-1"
          "awslogs-stream-prefix" = "ecs"
        }
      }
      environment = [
        { name = "FINFOCUS_PLUGIN_WEB_ENABLED", value = "true" }
      ]
    }
  ])
}

resource "aws_ecs_service" "plugin" {
  name            = "finfocus-plugin-aws-public"
  cluster         = aws_ecs_cluster.main.id
  task_definition = aws_ecs_task_definition.plugin.arn
  desired_count   = 1
  launch_type     = "FARGATE"

  network_configuration {
    subnets         = aws_subnet.private[*].id
    security_groups = [aws_security_group.plugin.id]
  }

  service_registries {
    registry_arn = aws_service_discovery_service.plugin.arn
  }
}
```

## Pulumi YAML Example

This example uses **Pulumi YAML** for a declarative deployment. It is often faster to iterate on and easier to test with `pulumi preview`.

```yaml
name: finfocus-plugin-aws-public-ecs
runtime: yaml
description: Deploy FinFocus AWS Public Plugin to ECS Fargate using Pulumi YAML

variables:
  # Replace these with your actual VPC and Subnet IDs
  vpcId:
    fn::invoke:
      function: aws:ec2:getVpc
      arguments:
        default: true
  subnetIds:
    fn::invoke:
      function: aws:ec2:getSubnets
      arguments:
        filters:
          - name: vpc-id
            values: [${vpcId.id}]

resources:
  # 1. Cloud Map Private DNS Namespace
  namespace:
    type: aws:servicediscovery:PrivateDnsNamespace
    properties:
      name: finfocus.local
      description: Internal service discovery for FinFocus plugins
      vpc: ${vpcId.id}

  # 2. Cloud Map Service
  discoveryService:
    type: aws:servicediscovery:Service
    properties:
      name: aws-public
      dnsConfig:
        namespaceId: ${namespace.id}
        dnsRecords:
          - ttl: 60
            type: A
        routingPolicy: MULTIVALUE
      healthCheckCustomConfig:
        failureThreshold: 1

  # 3. Security Group
  pluginSg:
    type: aws:ec2:SecurityGroup
    properties:
      description: Allow inbound traffic to FinFocus plugin regional ports
      vpcId: ${vpcId.id}
      ingress:
        - protocol: tcp
          fromPort: 8001
          toPort: 8012
          cidrBlocks: ["0.0.0.0/0"]
        - protocol: tcp
          fromPort: 9090
          toPort: 9090
          cidrBlocks: ["0.0.0.0/0"]

  # 4. ECS Cluster
  cluster:
    type: aws:ecs:Cluster
    properties:
      name: finfocus-cluster

  # 5. IAM Execution Role
  executionRole:
    type: aws:iam:Role
    properties:
      assumeRolePolicy:
        fn::toJSON:
          Version: "2012-10-17"
          Statement:
            - Action: "sts:AssumeRole"
              Effect: "Allow"
              Principal:
                Service: "ecs-tasks.amazonaws.com"
      managedPolicyArns:
        - arn:aws:iam::aws:policy/service-role/AmazonECSTaskExecutionRolePolicy

  # 6. CloudWatch Log Group
  logGroup:
    type: aws:cloudwatch:LogGroup
    properties:
      name: /ecs/finfocus-plugin
      retentionInDays: 7

  # 7. ECS Task Definition
  taskDefinition:
    type: aws:ecs:TaskDefinition
    properties:
      family: finfocus-plugin-aws-public
      cpu: "2048"
      memory: "4096"
      networkMode: awsvpc
      requiresCompatibilities: ["FARGATE"]
      executionRoleArn: ${executionRole.arn}
      containerDefinitions:
        fn::toJSON:
          - name: plugin
            image: ghcr.io/rshade/finfocus-plugin-aws-public:latest
            essential: true
            portMappings:
              - { containerPort: 8001, protocol: tcp }
              - { containerPort: 8002, protocol: tcp }
              - { containerPort: 8003, protocol: tcp }
              - { containerPort: 8004, protocol: tcp }
              - { containerPort: 8005, protocol: tcp }
              - { containerPort: 8006, protocol: tcp }
              - { containerPort: 8007, protocol: tcp }
              - { containerPort: 8008, protocol: tcp }
              - { containerPort: 8009, protocol: tcp }
              - { containerPort: 8010, protocol: tcp }
              - { containerPort: 8011, protocol: tcp }
              - { containerPort: 8012, protocol: tcp }
              - { containerPort: 9090, protocol: tcp }
            healthCheck:
              command: ["CMD-SHELL", "/healthcheck.sh"]
              interval: 30
              timeout: 5
              retries: 3
              startPeriod: 30
            logConfiguration:
              logDriver: awslogs
              options:
                awslogs-group: /ecs/finfocus-plugin
                awslogs-region: us-east-1
                awslogs-stream-prefix: ecs
            environment:
              - { name: FINFOCUS_PLUGIN_WEB_ENABLED, value: "true" }

  # 8. ECS Service
  service:
    type: aws:ecs:Service
    properties:
      name: finfocus-plugin-aws-public
      cluster: ${cluster.arn}
      taskDefinition: ${taskDefinition.arn}
      desiredCount: 1
      launchType: FARGATE
      networkConfiguration:
        assignPublicIp: true
        subnets: ${subnetIds.ids}
        securityGroups: [${pluginSg.id}]
      serviceRegistries:
        registryArn: ${discoveryService.arn}

outputs:
  endpoint: http://aws-public.finfocus.local:8001
```
