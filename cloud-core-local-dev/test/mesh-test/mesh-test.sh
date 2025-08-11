echo 'Testing mesh service through public gateway...';
echo 'Attempting to call mesh test service...';

TIMEOUT=60;
RETRY_INTERVAL=5;
START_TIME=\$(date +%s);

while true; do
    CURRENT_TIME=\$(date +%s);
    ELAPSED_TIME=\$((CURRENT_TIME - START_TIME));
    
    if [ \$ELAPSED_TIME -ge \$TIMEOUT ]; then
        echo 'Timeout reached after 60 seconds. Mesh test failed.';
        exit 1;
    fi;

    echo 'Attempt ' \$((ELAPSED_TIME/ \$RETRY_INTERVAL + 1))' - Testing mesh service... ' \$((TIMEOUT - ELAPSED_TIME)) ' s remaining)';

    if curl -s -f -m 10 http://mesh-test-service:8080/health >/dev/null 2>&1; then
        echo '✓ Mesh service is responding internally';

        if curl -s -f -m 10 http://public-gateway-service:8080/mesh-test/health >/dev/null 2>&1; then
            echo '✓ Mesh service is accessible through public gateway';
            echo '✓ Mesh smoke test successful!';
            echo 'Response from public gateway:';
            curl -s -m 10 http://public-gateway-service:8080/mesh-test/health;
            echo '';
            exit 0;
        else
            echo '✗ Mesh service not accessible through public gateway';
            echo 'Retrying in '\$RETRY_INTERVAL' seconds...';
            sleep \$RETRY_INTERVAL;
        fi;
    else
        echo '✗ Mesh service not responding internally';
        echo 'Retrying in '\$RETRY_INTERVAL' seconds...';
        sleep \$RETRY_INTERVAL;
    fi;
done; 