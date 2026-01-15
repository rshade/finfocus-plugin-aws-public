# Multi-Region Docker Image for finfocus-plugin-aws-public
# Stage 1: Download and prepare regional binaries
FROM alpine:3.19 AS downloader

# Install dependencies
RUN apk add --no-cache \
    ca-certificates=20241121-r1 \
    curl=8.5.0-r0

# Set working directory
WORKDIR /tmp

# Define build arguments
ARG VERSION=v0.1.0

# Define regions
ENV REGIONS="us-east-1 us-west-2 eu-west-1 ap-southeast-1 ap-southeast-2 ap-northeast-1 ap-south-1 ca-central-1 sa-east-1 us-gov-west-1 us-gov-east-1 us-west-1"

# Download and extract regional binaries
RUN set -ex; \
    arch=$(uname -m); \
    case "$arch" in \
        x86_64) release_arch="x86_64" ;; \
        aarch64) release_arch="arm64" ;; \
        *) echo "Unsupported architecture: ${arch}"; exit 1 ;; \
    esac; \
    for region in $REGIONS; do \
        tarball="finfocus-plugin-aws-public_${VERSION}_Linux_${release_arch}_${region}.tar.gz"; \
        echo "Downloading ${region} binary..."; \
        curl -L -o "/tmp/${tarball}" \
            "https://github.com/rshade/finfocus-plugin-aws-public/releases/download/${VERSION}/${tarball}"; \
        tar -xzf "/tmp/${tarball}" -C /tmp; \
        rm "/tmp/${tarball}"; \
        binary="/tmp/finfocus-plugin-aws-public-${region}"; \
        if [ ! -f "$binary" ]; then \
            echo "Error: Binary for ${region} not found after extraction"; \
            exit 1; \
        fi; \
        chmod +x "$binary"; \
    done; \
    echo "Successfully downloaded all regional binaries"

# Stage 2: Build metrics-aggregator
FROM golang:1.25.5-alpine AS builder

WORKDIR /workspace

# Copy go.mod and go.sum for dependency resolution
COPY go.mod go.sum ./

# Copy metrics-aggregator source
COPY cmd/metrics-aggregator/ cmd/metrics-aggregator/

# Build the metrics-aggregator binary
RUN go build -o /tmp/metrics-aggregator ./cmd/metrics-aggregator

# Stage 3: Final runtime image
FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates=20241121-r1 \
    curl=8.5.0-r0 \
    tini=0.19.0-r2

WORKDIR /app

# Copy regional binaries from downloader stage
COPY --from=downloader /tmp/finfocus-plugin-aws-public-* /usr/local/bin/

# Copy metrics-aggregator from builder stage
COPY --from=builder /tmp/metrics-aggregator /usr/local/bin/metrics-aggregator

# Copy scripts
COPY docker/entrypoint.sh /entrypoint.sh
COPY docker/healthcheck.sh /healthcheck.sh

# Make scripts executable
RUN chmod +x /entrypoint.sh /healthcheck.sh

# Verify all regional binaries exist
RUN set -ex; \
    for region in us-east-1 us-west-2 eu-west-1 ap-southeast-1 ap-southeast-2 ap-northeast-1 ap-south-1 ca-central-1 sa-east-1 us-gov-west-1 us-gov-east-1 us-west-1; do \
        if [ ! -f "/usr/local/bin/finfocus-plugin-aws-public-${region}" ]; then \
            echo "ERROR: Binary for ${region} not found in final image"; \
            exit 1; \
        fi; \
    done; \
    echo "All regional binaries verified in final image"

# Create non-root user
RUN addgroup -g 65532 -S plugin && \
    adduser -u 65532 -S plugin -G plugin

# Expose ports
EXPOSE 8001 8002 8003 8004 8005 8006 8007 8008 8009 8010 8011 8012 9090

# Set default environment variables
ENV FINFOCUS_PLUGIN_WEB_ENABLED=true
ENV FINFOCUS_PLUGIN_HEALTH_ENDPOINT=true

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=30s --retries=3 \
    CMD /healthcheck.sh

# Use tini as init process
ENTRYPOINT ["/sbin/tini", "--", "/entrypoint.sh"]

# Switch to non-root user
USER plugin