#!/bin/bash

NAMESPACE="core-1-core"
CURL_POD="curl-test-runner"
K6_POD="k6-test-runner"
TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

FIXTURES_DIR="$TEST_DIR/../../fixtures"

get_rule_info() {
    local rule_name=$1
    kubectl exec -n $NAMESPACE deployment/ratelimit -- sh -c "
        curl -s -X POST http://localhost:8082/api/v1/ratelimit/check \
          -H 'Content-Type: application/json' \
          -d '{\"components\":{\"path\":\"/$rule_name\",\"user_id\":\"test\"}}' 2>/dev/null
    " 2>/dev/null || echo "{\"limit\":\"unknown\"}"
}

# Apply a fixture ConfigMap from the host (kubectl runs here, not inside the pod)
# and trigger immediate reconciliation via the operator API.
apply_config() {
    local yaml_file=$1
    echo -e "${YELLOW}  Applying ConfigMap from host: $(basename $yaml_file)${NC}"
    kubectl apply -f "$yaml_file" -n $NAMESPACE
    kubectl exec -n $NAMESPACE deployment/ratelimit -- \
        curl -s -X POST http://localhost:8082/api/v1/config/reload > /dev/null
    sleep 2
}

delete_config() {
    local cm_name=$1
    kubectl delete configmap "$cm_name" -n $NAMESPACE --ignore-not-found > /dev/null
    kubectl exec -n $NAMESPACE deployment/ratelimit -- \
        curl -s -X POST http://localhost:8082/api/v1/config/reload > /dev/null
}

run_test() {
    local test_name=$1
    local script=$2
    local args=$3
    
    echo ""
    echo -e "${BLUE}============================================================${NC}"
    echo -e "${GREEN}🚀 RUNNING: $test_name${NC}"
    echo -e "${BLUE}============================================================${NC}"
    
    kubectl exec -n $NAMESPACE $CURL_POD -- sh -c "sh /scripts/$script $args"
    
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✅ $test_name completed successfully${NC}"
    else
        echo -e "${RED}❌ $test_name failed${NC}"
    fi
    echo ""
}

run_k6_test() {
    local test_name=$1
    local script=$2
    
    echo ""
    echo -e "${BLUE}============================================================${NC}"
    echo -e "${GREEN}🚀 RUNNING K6: $test_name${NC}"
    echo -e "${BLUE}============================================================${NC}"
    
    kubectl exec -n $NAMESPACE $K6_POD -- sh -c "
        export K6_QUIET=1
        k6 run /scripts/$script 2>&1 | tail -40
    "
    
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✅ K6 $test_name completed successfully${NC}"
    else
        echo -e "${RED}❌ K6 $test_name failed${NC}"
    fi
    echo ""
}

# Clean up old pod
kubectl delete pod $CURL_POD -n $NAMESPACE --ignore-not-found
kubectl delete pod $K6_POD -n $NAMESPACE --ignore-not-found

# Apply ConfigMap and Pod
kubectl apply -f $TEST_DIR/demo-test-scripts.yaml -n $NAMESPACE
kubectl apply -f $TEST_DIR/curl-test-runner.yaml -n $NAMESPACE
kubectl apply -f $TEST_DIR/k6-test-runner.yaml -n $NAMESPACE

# Wait for pods
kubectl wait --for=condition=ready pod/$K6_POD -n $NAMESPACE --timeout=60s
kubectl wait --for=condition=ready pod/$CURL_POD -n $NAMESPACE --timeout=60s

# Menu
echo ""
echo -e "${BLUE}============================================================${NC}"
echo -e "${GREEN}           RATE LIMIT DEMO WITH PRIORITIES${NC}"
echo -e "${BLUE}============================================================${NC}"
echo "1) Show Current Rules with Priorities"
echo "2) Add Rules with Different Priorities"
echo "3) Priority Demo (Admin/VIP/Normal Users)"
echo "4) Gateway Distribution Test (200 requests)"
echo "5) Rate Limit Accuracy Test"
echo "6) Algorithm Comparison Test (200 requests)"
echo "7) K6 Load Test (constant 50 req/s, 30s)"
echo "8) K6 Burst Test (spike to 500 req/s)"
echo "9) Run ALL tests"
echo "10) Exit"
echo -e "${BLUE}============================================================${NC}"
read -p "Choose test (1-10): " choice

case $choice in
    1)
        run_test "Get Current Rules" "get-rules.sh" ""
        ;;
    2)
        apply_config "$FIXTURES_DIR/ratelimit-config-priority-demo.yaml"
        run_test "Add Rules with Priorities" "add-rules-with-priority.sh" ""
        ;;
    3)
        apply_config "$FIXTURES_DIR/ratelimit-config-priority-demo.yaml"
        run_test "Priority Demo" "priority-demo.sh" ""
        ;;
    4)
        run_test "Gateway Distribution Test" "gateway-distribution.sh" "200"
        ;;
    5)
        apply_config "$FIXTURES_DIR/ratelimit-config-accuracy.yaml"
        run_test "Rate Limit Accuracy Test" "accuracy-test.sh" ""
        delete_config "k6-accuracy-test"
        ;;
    6)
        apply_config "$FIXTURES_DIR/ratelimit-config-algo-compare.yaml"
        run_test "Algorithm Comparison Test" "algorithm-compare.sh" ""
        delete_config "k6-fixed-test"
        delete_config "k6-sliding-test"
        ;;
    7)
        apply_config "$FIXTURES_DIR/ratelimit-config-loadtest.yaml"
        run_k6_test "K6 Load Test" "k6-load-test.js"
        delete_config "ratelimit-config-loadtest"
        ;;
    8)
        run_k6_test "K6 Burst Test" "k6-burst-test.js"
        ;;
    9)
        run_test "Get Current Rules" "get-rules.sh" ""
        apply_config "$FIXTURES_DIR/ratelimit-config-priority-demo.yaml"
        run_test "Add Rules with Priorities" "add-rules-with-priority.sh" ""
        run_test "Priority Demo" "priority-demo.sh" ""
        run_test "Gateway Distribution Test" "gateway-distribution.sh" "200"
        apply_config "$FIXTURES_DIR/ratelimit-config-accuracy.yaml"
        run_test "Rate Limit Accuracy Test" "accuracy-test.sh" ""
        delete_config "k6-accuracy-test"
        apply_config "$FIXTURES_DIR/ratelimit-config-algo-compare.yaml"
        run_test "Algorithm Comparison Test" "algorithm-compare.sh" ""
        delete_config "k6-fixed-test"
        delete_config "k6-sliding-test"
        run_k6_test "K6 Load Test" "k6-load-test.js"
        run_k6_test "K6 Burst Test" "k6-burst-test.js"
        echo -e "${GREEN}✅ All tests completed${NC}"
        ;;
    10)
        echo "Exiting..."
        exit 0
        ;;
    *)
        echo -e "${RED}Invalid choice${NC}"
        exit 1
        ;;
esac

# Cleanup
kubectl delete pod $K6_POD -n $NAMESPACE --ignore-not-found
kubectl delete pod $CURL_POD -n $NAMESPACE --ignore-not-found