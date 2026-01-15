#!/bin/bash
set -e

# Entrypoint script for finfocus-plugin-aws-public multi-region container
# Starts all 12 regional binaries and manages their lifecycle

# Define regions and ports
declare -a regions=("us-east-1" "us-west-2" "eu-west-1" "ap-southeast-1" "ap-southeast-2" "ap-northeast-1" "ap-south-1" "ca-central-1" "sa-east-1" "us-gov-west-1" "us-gov-east-1" "us-west-1")
declare -a ports=(8001 8002 8003 8004 8005 8006 8007 8008 8009 8010 8011 8012)

# start_binary starts the region-specific finfocus-plugin-aws-public binary in the background, prefixes non-JSON output with the region, injects a `"region"` field into JSON log lines, and writes the background PID to /tmp/pid_<region>.
start_binary() {
    local region=$1
    local port=$2
    local binary="/usr/local/bin/finfocus-plugin-aws-public-${region}"
    local fifo="/tmp/fifo_${region}"

    echo "Starting ${region} on port ${port}"

    # Create FIFO for binary output
    rm -f "$fifo"
    mkfifo "$fifo"

    # Start FIFO reader in background BEFORE starting the binary to avoid race condition.
    # This ensures the reader is waiting on the FIFO when the binary tries to write to it.
    (
        while IFS= read -r line; do
            # Try to inject region into JSON if it looks like JSON
            if [[ $line =~ ^\{.*\} ]]; then
                # Insert region field after opening brace
                # Use printf to safely handle special characters in the region string
                printf '{"region":"%s",%s\n' "$region" "${line:1}"
            else
                # Prefix non-JSON lines
                echo "[${region}] ${line}"
            fi
        done < "$fifo"
    ) &
    local reader_pid=$!

    # Now start the binary, which will connect to the waiting reader
    export REGION="${region}"
    export PORT="${port}"
    "${binary}" > "$fifo" 2>&1 &
    local binary_pid=$!
    echo "$binary_pid" > "/tmp/pid_${region}"

    # Clean up FIFO and wait for reader when binary exits
    (
        wait "$binary_pid" 2>/dev/null || true
        wait "$reader_pid" 2>/dev/null || true
        rm -f "$fifo"
    ) &
}

# stop_binaries stops all regional binaries and the metrics aggregator, sending SIGTERM, waiting up to TERMINATION_GRACE_SECONDS (defaults to 30) for graceful exit, then sending SIGKILL to any remaining processes and removing their PID files.
# 
# It reads per-region PID files at /tmp/pid_<region> and /tmp/pid_metrics, attempts graceful termination, polls for exit within the grace period, force-kills lingering processes after the timeout, and cleans up the corresponding PID files.
stop_binaries() {
    echo "Stopping all binaries..."
    local -a active_pids=()
    local grace_seconds="${TERMINATION_GRACE_SECONDS:-30}"

    for region in "${regions[@]}"; do
        pid_file="/tmp/pid_${region}"
        if [[ -f $pid_file ]]; then
            pid=$(cat "$pid_file")
            if kill -TERM "$pid" 2>/dev/null; then
                echo "Sent SIGTERM to ${region} (PID ${pid})"
                active_pids+=("${pid}:${pid_file}:${region}")
            fi
        fi
    done

    if [[ ${#active_pids[@]} -gt 0 ]]; then
        end_time=$((SECONDS + grace_seconds))
        while [[ ${#active_pids[@]} -gt 0 && $SECONDS -lt $end_time ]]; do
            for index in "${!active_pids[@]}"; do
                IFS=: read -r pid pid_file region <<< "${active_pids[$index]}"
                if ! kill -0 "$pid" 2>/dev/null; then
                    echo "${region} stopped gracefully"
                    unset "active_pids[$index]"
                    rm -f "$pid_file"
                fi
            done
            active_pids=("${active_pids[@]}")
            sleep 1
        done

        for entry in "${active_pids[@]}"; do
            IFS=: read -r pid pid_file region <<< "$entry"
            if kill -0 "$pid" 2>/dev/null; then
                echo "Force killing ${region}"
                kill -KILL "$pid" 2>/dev/null || true
            fi
            rm -f "$pid_file"
        done
    fi

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
        start_binary "$region" "$port"

        if timeout 5 bash -c "until curl -fsS http://localhost:${port}/healthz >/dev/null; do sleep 0.5; done"; then
            echo "Successfully started ${region}"
            break
        fi

        echo "Failed to start ${region} (attempt ${attempt})"
        stop_binaries
        if [[ $attempt -eq 3 ]]; then
            echo "Failed to start ${region} after 3 attempts. Shutting down container."
            exit 1
        fi
        sleep 2
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