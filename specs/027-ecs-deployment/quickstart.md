# Quick Start: ECS Deployment

## Prerequisites
- AWS Account
- VPC with private subnets
- ECS Cluster (Fargate capable)

## Deployment Steps

1.  **Create Security Group**: Allow inbound TCP 8001-8012 from your VPC.
2.  **Create Cloud Map Namespace**: `finfocus.local` (Private DNS).
3.  **Register Task Definition**:

```json
{
  "family": "finfocus-plugin-aws-public",
  "networkMode": "awsvpc",
  "cpu": "2048",
  "memory": "4096",
  "containerDefinitions": [
    {
      "name": "plugin",
      "image": "ghcr.io/rshade/finfocus-plugin-aws-public:latest",
      "portMappings": [
        {"containerPort": 8001}, {"containerPort": 8002}, {"containerPort": 8003},
        {"containerPort": 8004}, {"containerPort": 8005}, {"containerPort": 8006},
        {"containerPort": 8007}, {"containerPort": 8008}, {"containerPort": 8009},
        {"containerPort": 8010}, {"containerPort": 8011}, {"containerPort": 8012},
        {"containerPort": 9090}
      ],
      "healthCheck": {
        "command": ["CMD-SHELL", "/healthcheck.sh"],
        "interval": 30,
        "timeout": 5,
        "retries": 3
      },
      "logConfiguration": {
        "logDriver": "awslogs",
        "options": {
          "awslogs-group": "/ecs/finfocus-plugin",
          "awslogs-region": "us-east-1",
          "awslogs-stream-prefix": "ecs"
        }
      }
    }
  ]
}
```

4.  **Create Service**:
    - Launch Type: Fargate
    - Service Connect / Discovery: Enable Service Discovery (Type A record).
    - Desired Count: 1

## Accessing the Plugin
- **us-east-1**: `http://finfocus-plugin-aws-public.finfocus.local:8001`
- **us-west-2**: `http://finfocus-plugin-aws-public.finfocus.local:8002`
...
