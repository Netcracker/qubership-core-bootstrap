#!/bin/bash

set -e

MAAS_NAMESPACE=${MAAS_NAMESPACE:-maas}
SERVICE=maas-service
LOCAL_PORT=8080
REMOTE_PORT=8080
TIMEOUT=180  # 3 минуты
INTERVAL=5  # секунд между попытками

echo "--- Start port-forward ${SERVICE} for port ${LOCAL_PORT}:${REMOTE_PORT}..."
kubectl port-forward svc/${SERVICE} ${LOCAL_PORT}:${REMOTE_PORT} -n "${MAAS_NAMESPACE}" > /dev/null 2>&1 &
PF_PID=$!

cleanup() {
    echo "--- Close port-forward (PID=${PF_PID})"
    kill $PF_PID 2>/dev/null || true
}
trap cleanup EXIT

for i in $(seq 1 10); do
    nc -z localhost ${LOCAL_PORT} && break
    echo "--- Waiting for port ${LOCAL_PORT}... (${i}/10)"
    sleep 1
done

START_TIME=$(date +%s)
while true; do
    echo "--- Perform health check..."
    RESPONSE=$(curl -s http://localhost:${LOCAL_PORT}/health)

    STATUS=$(echo "$RESPONSE" | jq -r '.status')
    POSTGRES_STATUS=$(echo "$RESPONSE" | jq -r '.postgres.status')
    KAFKA_STATUS=$(echo "$RESPONSE" | jq -r '.kafka.status')
    RABBIT_STATUS=$(echo "$RESPONSE" | jq -r '.rabbit.status')

    ERROR=0

    if [ "$STATUS" != "UP" ]; then
        echo "❌ MaaS status is not UP: status=$STATUS"
        ERROR=1
    fi
    if [ "$POSTGRES_STATUS" != "UP" ]; then
        echo "❌ Postgres status is not UP: postgres.status=$POSTGRES_STATUS"
        ERROR=1
    fi
    if [ "$KAFKA_STATUS" != "UP" ]; then
        echo "❌ Kafka status is not UP: kafka.status=$KAFKA_STATUS"
        ERROR=1
    fi
    if [ "$RABBIT_STATUS" != "UP" ]; then
        echo "❌ Rabbit status is not UP: rabbit.status=$RABBIT_STATUS"
        ERROR=1
    fi

    if [ $ERROR -eq 0 ]; then
        echo "✅ MaaS health is OK"
        exit 0
    fi

    CURRENT_TIME=$(date +%s)
    ELAPSED=$((CURRENT_TIME - START_TIME))
    if [ $ELAPSED -ge $TIMEOUT ]; then
        echo "❌ MaaS health check failed! Таймаут ${TIMEOUT} секунд."
        echo "RESPONSE = $RESPONSE"
        exit 1
    fi
    echo "--- Not healthy yet, retrying in ${INTERVAL}s (elapsed: ${ELAPSED}s/${TIMEOUT}s)"
    sleep $INTERVAL
done 