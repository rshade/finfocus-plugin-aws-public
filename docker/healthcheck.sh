#!/bin/bash

# Health check script for finfocus-plugin-aws-public multi-region container
# Verifies that all 12 regional HTTP endpoints are responding
#
# Optional environment variable:
#   QUIET_HEALTHCHECK - When set to 1/true, suppresses verbose output
#                       (failures and final status are always printed)

set -e

# Define ports
declare -a ports=(8001 8002 8003 8004 8005 8006 8007 8008 8009 8010 8011 8012)

# Check quiet mode
quiet="${QUIET_HEALTHCHECK:-0}"
if [[ "$quiet" == "true" ]]; then
    quiet=1
fi

if [[ "$quiet" != "1" ]]; then
    echo "Running health check for all regional endpoints"
fi

for port in "${ports[@]}"; do
    if [[ "$quiet" != "1" ]]; then
        echo "Checking port ${port}..."
    fi

    # Try to connect to health endpoint
    if curl -f -s --max-time 5 "http://localhost:${port}/healthz" > /dev/null 2>&1; then
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