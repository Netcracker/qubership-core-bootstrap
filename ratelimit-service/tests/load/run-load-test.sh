#!/bin/bash
# run-load-test.sh - Запуск нагрузочного тестирования

set -e

NAMESPACE="core-1-core"
GATEWAY_PORT="8080"
DURATION="60s"
CONCURRENCY="50"

# Получить JWT токен
JWT_TOKEN=$(kubectl get secret e2e-jwt-token -n $NAMESPACE -o jsonpath='{.data.token}' | base64 -d)

echo "=== Load Testing RateLimit Operator ==="
echo "Gateway: localhost:$GATEWAY_PORT"
echo "Duration: $DURATION"
echo "Concurrency: $CONCURRENCY"
echo ""

# Использование wrk
echo "Running wrk test..."
wrk -t4 -c$CONCURRENCY -d$DURATION \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -H "x-user-id: load-user" \
  "http://localhost:$GATEWAY_PORT/test"

# Сбор метрик во время теста
echo ""
echo "Collecting metrics during test..."
kubectl port-forward -n $NAMESPACE svc/ratelimit-service-api 8082:8082 &
PF_PID=$!
sleep 2

curl -s http://localhost:8082/metrics | grep -E "ratelimit_(checks|violating|active)" || true

kill $PF_PID 2>/dev/null

echo ""
echo "=== Load test completed ==="