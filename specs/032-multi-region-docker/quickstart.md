# Quickstart: Multi-Region Docker Image

## Prerequisites
- Docker
- Make (optional, for build commands)

## Build

To build the image locally, you need a valid release version (e.g., `v0.1.0`) that exists in GitHub Releases with artifacts.

```bash
docker build \
  --build-arg VERSION=v0.1.0 \
  -t finfocus-plugin-aws-public:latest \
  -f build/Dockerfile .
```

## Run

Run the container exposing all ports:

```bash
docker run -d \
  --name finfocus-aws \
  -p 8001-8012:8001-8012 \
  -p 9090:9090 \
  finfocus-plugin-aws-public:latest
```

## Verify

### Check Health
Access the us-east-1 health endpoint:
```bash
curl http://localhost:8001/healthz
```

### Check Metrics
Access the aggregated metrics:
```bash
curl http://localhost:9090/metrics
```

### Check Logs
Verify region prefixes in logs:
```bash
docker logs finfocus-aws
# Output should show: [us-east-1] ...
```

## Kubernetes Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: finfocus-aws-plugin
spec:
  replicas: 1
  selector:
    matchLabels:
      app: finfocus-aws-plugin
  template:
    metadata:
      labels:
        app: finfocus-aws-plugin
    spec:
      containers:
      - name: plugin
        image: ghcr.io/rshade/finfocus-plugin-aws-public:latest
        ports:
        - containerPort: 8001
          name: us-east-1
        # ... other ports
        - containerPort: 9090
          name: metrics
        readinessProbe:
          httpGet:
            path: /healthz
            port: 8001 # Check primary region
```
