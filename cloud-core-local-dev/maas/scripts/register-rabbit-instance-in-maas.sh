#!/bin/bash

set -e

if [ -z "$1" ]; then
  echo "Please specify RABBIT_INSTANCE parameter"
  exit 1
fi

RABBIT_INSTANCE="$1"

MAAS_ACCOUNT_MANAGER_USERNAME=${MAAS_ACCOUNT_MANAGER_USERNAME:-manager}
MAAS_ACCOUNT_MANAGER_PASSWORD=${MAAS_ACCOUNT_MANAGER_PASSWORD:-manager}
MAAS_NAMESPACE=${MAAS_NAMESPACE:-maas}
SERVICE=maas-service
LOCAL_PORT=8080
REMOTE_PORT=8080

echo "--- Start port-forward ${SERVICE} for port ${LOCAL_PORT}:${REMOTE_PORT}..."
kubectl port-forward svc/${SERVICE} ${LOCAL_PORT}:${REMOTE_PORT} -n ${MAAS_NAMESPACE} > /dev/null 2>&1 &
PF_PID=$!

cleanup() {
    echo "--- Close port-forward (PID=${PF_PID})"
    kill $PF_PID 2>/dev/null || true
}
trap cleanup EXIT

for i in $(seq 1 10); do
    nc -z localhost ${LOCAL_PORT} && break
    echo "Waiting for port ${LOCAL_PORT}... (${i}/10)"
    sleep 1
done

echo "--- Prepare body for POST-request..."
JSON_BODY=$(cat <<EOF
{
  "id": "${RABBIT_INSTANCE}",
  "apiUrl": "http://${RABBIT_INSTANCE}.${RABBIT_NAMESPACE}:15672/api",
  "amqpUrl": "amqp://${RABBIT_INSTANCE}.${RABBIT_NAMESPACE}:5672",
  "user": "admin",
  "password": "admin"
}
EOF
)

echo "--- Send POST-request..."
RESPONSE=$(mktemp)
HTTP_CODE=$(curl -s -w "%{http_code}" -o "$RESPONSE" \
  -X POST "http://localhost:${LOCAL_PORT}/api/v2/rabbit/instance" \
  -H "Content-Type: application/json" \
  -u "${MAAS_ACCOUNT_MANAGER_USERNAME}:${MAAS_ACCOUNT_MANAGER_PASSWORD}" \
  -d "${JSON_BODY}")

echo "--- Response HTTP-code: ${HTTP_CODE}"
cat "$RESPONSE" | jq . || cat "$RESPONSE"

if [[ "$HTTP_CODE" == "200" ]]; then
  echo "✅ Instance successfully registered"
elif [[ "$HTTP_CODE" == "400" ]]; then
  CODE=$(jq -r '.code // empty' "$RESPONSE" 2>/dev/null)
  if [[ "$CODE" == "MAAS-0600" ]]; then
    echo "⚠️ Instance already registered"
  else
    echo "❌ Error 400: $CODE"
    exit 1
  fi
else
  echo "❌ Unexpected response: HTTP $HTTP_CODE"
  exit 1
fi
