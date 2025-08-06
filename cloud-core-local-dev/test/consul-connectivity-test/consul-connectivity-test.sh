echo 'Starting Consul connectivity check with retry loop...';

TIMEOUT=300;
RETRY_INTERVAL=10;
START_TIME=\$(date +%s);

while true; do
    CURRENT_TIME=\$(date +%s);
    ELAPSED_TIME=\$((CURRENT_TIME - START_TIME));
    
    if [ \$ELAPSED_TIME -ge \$TIMEOUT ]; then
        echo 'Timeout reached after 5 minutes. Consul connectivity check failed.';
        exit 1;
    fi;

    echo 'Attempt '\$((ELAPSED_TIME / \$RETRY_INTERVAL + 1)) '- Checking Consul connectivity... ('\$((TIMEOUT - ELAPSED_TIME))'s remaining)';

    if curl -s -f http://\$CONSUL_SERVICE_NAME.\$CONSUL_NAMESPACE.\$INGRESS_GATEWAY_CLOUD_PRIVATE_HOST:8500/v1/status/leader >/dev/null 2>&1; then
        echo '✓ Consul API is responding';
        
        if curl -s -f http://\$CONSUL_SERVICE_NAME.\$CONSUL_NAMESPACE.\$INGRESS_GATEWAY_CLOUD_PRIVATE_HOST:8500/v1/status/peers >/dev/null 2>&1; then
            echo '✓ Consul cluster is healthy';
            echo '✓ Consul connectivity check successful!';
            exit 0;
        else
            echo '✗ Consul cluster health check failed';
            echo 'Retrying in '\$RETRY_INTERVAL' seconds...';
            sleep \$RETRY_INTERVAL;
        fi;
    else
        echo '✗ Consul API is not responding';
        echo 'Retrying in '\$RETRY_INTERVAL' seconds...';
        sleep \$RETRY_INTERVAL;
    fi;
done; 