#!/bin/bash
set -e

# Entrypoint script for finfocus-plugin-aws-public multi-region container
# Starts all 12 regional binaries and manages their lifecycle

# Define regions and ports
declare -a regions=("us-east-1" "us-west-2" "eu-west-1" "ap-southeast-1" "ap-southeast-2" "ap-northeast-1" "ap-south-1" "ca-central-1" "sa-east-1" "us-gov-west-1" "us-gov-east-1" "us-west-1")
declare -a ports=(8001 8002 8003 8004 8005 8006 8007 8008 8009 8010 8011 8012)

# start_binary starts the regional finfocus binary in the background, injects a "region" field into JSON output lines or prefixes non-JSON lines with the region tag, and records the child's PID to /tmp/pid_<region>.
start_binary() {
    local region=$1
    local port=$2
    local binary="/usr/local/bin/finfocus-plugin-aws-public-${region}"

    echo "Starting ${region} on port ${port}"

    # Start binary in background, capturing both stdout and stderr
    # Use sed to inject region field into JSON lines
    (
        export REGION="${region}"
        export PORT="${port}"
        exec "${binary}" 2>&1 | while IFS= read -r line; do
            # Try to inject region into JSON if it looks like JSON
            if [[ $line =~ ^\{.*\} ]]; then
                # Insert region field after opening brace
                echo "$line" | sed 's/{/{"region":"'"${region}"'",/'
            else
                # Prefix non-JSON lines
                echo "[${region}] ${line}"
            fi
        done
    ) &
    echo $! > "/tmp/pid_${region}"
}

# stop_binaries stops all regional binaries and the metrics aggregator by sending SIGTERM and waiting up to 10 seconds for each to exit, sending SIGKILL if still running, and removing their PID files.
stop_binaries() {
    echo "Stopping all binaries..."
    for region in "${regions[@]}"; do
        pid_file="/tmp/pid_${region}"
        if [[ -f $pid_file ]]; then
            pid=$(cat "$pid_file")
            if kill -TERM "$pid" 2>/dev/null; then
                echo "Sent SIGTERM to ${region} (PID ${pid})"
                # Wait up to 10 seconds for graceful shutdown
                for i in {1..10}; do
                    if ! kill -0 "$pid" 2>/dev/null; then
                        echo "${region} stopped gracefully"
                        break
                    fi
                    sleep 1
                done
                # Force kill if still running
                if kill -0 "$pid" 2>/dev/null; then
                    echo "Force killing ${region}"
                    kill -KILL "$pid" 2>/dev/null || true
                fi
            fi
            rm -f "$pid_file"
        fi
    done

    if [[ -f /tmp/pid_metrics ]]; then
        metrics_pid=$(cat /tmp/pid_metrics)
        if kill -TERM "$metrics_pid" 2>/dev/null; then
            echo "Sent SIGTERM to metrics aggregator (PID ${metrics_pid})"
        fi
        rm -f /tmp/pid_metrics
    fi
}

# Set default environment variables if not set
export FINFOCUS_PLUGIN_WEB_ENABLED="${FINFOCUS_PLUGIN_WEB_ENABLED:-true}"
export FINFOCUS_PLUGIN_HEALTH_ENDPOINT="${FINFOCUS_PLUGIN_HEALTH_ENDPOINT:-true}"
export FINFOCUS_LOG_LEVEL="${FINFOCUS_LOG_LEVEL:-info}"

# Trap signals for graceful shutdown
trap stop_binaries TERM INT

echo "Starting finfocus-plugin-aws-public multi-region container"

# Start all binaries
for i in "${!regions[@]}"; do
    region="${regions[$i]}"
    port="${ports[$i]}"

    # Retry logic: try up to 3 times
    for attempt in {1..3}; do
        echo "Attempting to start ${region} (attempt ${attempt})"
        if start_binary "$region" "$port"; then
            echo "Successfully started ${region}"
            break
        else
            echo "Failed to start ${region} (attempt ${attempt})"
            if [[ $attempt -eq 3 ]]; then
                echo "Failed to start ${region} after 3 attempts. Shutting down container."
                stop_binaries
                exit 1
            fi
            sleep 2
        fi
    done
done

# Start the metrics aggregator in background
echo "Starting metrics aggregator"
/usr/local/bin/metrics-aggregator &
echo $! > /tmp/pid_metrics

echo "All binaries started successfully. Container is ready."

# Wait for all processes
wait

echo "All processes exited. Container shutting down."