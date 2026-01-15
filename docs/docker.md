# Multi-Region Docker Image

The FinFocus AWS Public Plugin is available as a single Docker image that bundles
all 12 supported regional binaries. This simplifies deployment to Kubernetes
and other container orchestration platforms.

## Architecture

The Docker image uses a multi-process architecture managed by an entrypoint script.
It runs 12 instances of the plugin (one per region) plus a metrics aggregator.

### Port Mapping

| Port | Region |
|------|--------|
| 8001 | us-east-1 |
| 8002 | us-west-2 |
| 8003 | eu-west-1 |
| 8004 | ap-southeast-1 |
| 8005 | ap-southeast-2 |
| 8006 | ap-northeast-1 |
| 8007 | ap-south-1 |
| 8008 | ca-central-1 |
| 8009 | sa-east-1 |
| 8010 | us-gov-west-1 |
| 8011 | us-gov-east-1 |
| 8012 | us-west-1 |
| 9090 | Prometheus Metrics |

## Getting Started

### Run Locally

```bash
docker run -d \
  --name finfocus-aws \
  -p 8001-8012:8001-8012 \
  -p 9090:9090 \
  ghcr.io/rshade/finfocus-plugin-aws-public:${VERSION}
```

> **Note:** Replace `${VERSION}` with an actual release tag (e.g., `v0.1.0`) or use `:latest` for the newest release.

### Kubernetes Deployment

See [test/k8s/deployment.yaml](../test/k8s/deployment.yaml) for a complete
deployment example.

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: finfocus-aws-plugin
spec:
  replicas: 1
  template:
    spec:
      containers:
        - name: plugin
          image: ghcr.io/rshade/finfocus-plugin-aws-public:${VERSION}
          ports:
            - containerPort: 8001
              name: us-east-1
            # ... other ports
            - containerPort: 9090
              name: metrics
```

## Features

### Graceful Shutdown

The entrypoint script traps `SIGTERM` and ensures all 12 regional binaries
shut down cleanly.

### Log Aggregation

Logs from all 12 binaries are multiplexed to the container's stdout.
Each log entry is injected with a `region` field for easy filtering:

```json
{"level":"info","region":"us-east-1","message":"cost calculated",...}
```

### Metrics Aggregation

A built-in metrics aggregator scrapes all 12 regional `/metrics` endpoints
and provides a unified view on port `9090`.

### Health Checks

The image includes a health check script that verifies all 12 regional
binaries are responding before marking the container as healthy.

## Building from Source

To build the image locally, you must provide the `VERSION` build argument
matching an existing GitHub release:

```bash
docker build \
  --build-arg VERSION=v0.1.0 \
  -t finfocus-plugin-aws-public:latest \
  -f docker/Dockerfile .
```

## Security

- **Non-root user**: Runs as `plugin` (UID 65532).
- **Alpine base**: Built on `alpine:3.19` for a minimal attack surface.
- **Read-only**: Compatible with read-only filesystems.
