#!/bin/bash

# Health check script for finfocus-plugin-aws-public multi-region container
# Verifies that all 9 regional HTTP endpoints are responding
#
# Optional environment variables:
#   QUIET_HEALTHCHECK - When set to 1/true, suppresses verbose output
#                       (failures and final status are always printed)
#   HEALTH_CHECK_TIMEOUT - HTTP request timeout per region (default: 5 seconds)
#                          Increase if regions take longer to start

set -e

# Define ports (must match available release assets: 9 regions)
declare -a ports=(8001 8002 8003 8004 8005 8006 8007 8008 8009)

# Check quiet mode
quiet="${QUIET_HEALTHCHECK:-0}"
if [[ "$quiet" == "true" ]]; then
    quiet=1
fi

# Get HTTP timeout (default: 5 seconds)
timeout="${HEALTH_CHECK_TIMEOUT:-5}"

if [[ "$quiet" != "1" ]]; then
    echo "Running health check for all regional endpoints"
fi

for port in "${ports[@]}"; do
    if [[ "$quiet" != "1" ]]; then
        echo "Checking port ${port}..."
    fi

    # Try to connect to health endpoint
    if curl -f -s --max-time "$timeout" "http://localhost:${port}/healthz" > /dev/null 2>&1; then
        if [[ "$quiet" != "1" ]]; then
            echo "✓ Port ${port} is healthy"
        fi
    else
        echo "✗ Port ${port} is unhealthy"
        exit 1
    fi
done

echo "All endpoints are healthy"
exit 0