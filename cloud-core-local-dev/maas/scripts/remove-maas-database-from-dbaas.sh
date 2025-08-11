#!/bin/bash

set -e

USERNAME=${DBAAS_CLUSTER_DBA_CREDENTIALS_USERNAME:-cluster-dba}
PASSWORD=${DBAAS_CLUSTER_DBA_CREDENTIALS_PASSWORD:-password}
DBAAS_NAMESPACE=${DBAAS_NAMESPACE:-dbaas}
MAAS_NAMESPACE=${MAAS_NAMESPACE:-maas}
SERVICE=dbaas-aggregator
LOCAL_PORT=8080
REMOTE_PORT=8080


echo "--- Check service availability..."
if ! kubectl get svc/${SERVICE} -n ${DBAAS_NAMESPACE} >/dev/null 2>&1; then
    echo "Service ${SERVICE} not found in namespace ${DBAAS_NAMESPACE} - skip maas database removal"
    exit 0
fi

echo "--- Check if service has endpoints..."
if ! kubectl get endpoints/${SERVICE} -n ${DBAAS_NAMESPACE} -o jsonpath='{.subsets[*].addresses}' | grep -q .; then
    echo "Service ${SERVICE} in namespace ${DBAAS_NAMESPACE} has no endpoints - skip maas database removal"
    exit 0
fi

echo "--- Start port-forward ${SERVICE} for port ${LOCAL_PORT}:${REMOTE_PORT}..."
kubectl port-forward svc/${SERVICE} ${LOCAL_PORT}:${REMOTE_PORT} -n ${DBAAS_NAMESPACE} > /dev/null 2>&1 &
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
     "classifier": {
            "microserviceName": "maas-service",
            "namespace": "${MAAS_NAMESPACE}",
            "scope": "service"
        },
    "originService":"maas-service"
}
EOF
)

echo "--- Send DELETE-request..."
RESPONSE=$(mktemp)

HTTP_CODE=$(curl -s -w "%{http_code}" -o "$RESPONSE" \
  -X DELETE "http://localhost:${LOCAL_PORT}/api/v3/dbaas/${MAAS_NAMESPACE}/databases/postgresql" \
  -H "Content-Type: application/json" \
  -u "${USERNAME}:${PASSWORD}" \
  -d "${JSON_BODY}")

echo "--- Response HTTP-code: ${HTTP_CODE}"
cat "$RESPONSE" | jq . || cat "$RESPONSE"

if [[ "$HTTP_CODE" == "200" ]]; then
  echo "✅ MaaS database successfully deleted"
else
  echo "❌ Unexpected response: HTTP $HTTP_CODE"
  exit 1
fi
