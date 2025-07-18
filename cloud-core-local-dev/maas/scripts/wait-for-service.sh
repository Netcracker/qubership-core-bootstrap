#!/bin/bash

# Usage: wait-for-service.sh '<check_command>' <service_name> [timeout_sec]
# Example: "maas-service" ./wait-for-service.sh 'kubectl logs -n maas -l name=maas-service --tail=100 | grep -q "Starting server on"' 300

set -e

SERVICE_NAME="$1"
CHECK_CMD="$2"
TIMEOUT="${3:-300}"

if [ -z "$CHECK_CMD" ] || [ -z "$SERVICE_NAME" ]; then
  echo "Usage: $0 '<check_command>' <service_name> [timeout_sec]"
  exit 1
fi

START_TIME=$(date +%s)
while true; do
  CURRENT_TIME=$(date +%s)
  ELAPSED_TIME=$((CURRENT_TIME - START_TIME))
  if [ $ELAPSED_TIME -ge $TIMEOUT ]; then
    echo "Timeout reached after $((TIMEOUT/60)) minutes. Started message not found in logs."
    exit 1
  fi
  if eval "$CHECK_CMD"; then
    echo "=== $SERVICE_NAME started ==="
    break
  fi
  echo "Waiting for $SERVICE_NAME start... - $((TIMEOUT - ELAPSED_TIME))s remaining"
  sleep 10
done 