#!/bin/bash
# deploy-cloud.sh - Deploy operator to cloud

set -e

NAMESPACE="core-1-core"

echo "=== Deploying RateLimit Operator to Cloud ==="

# Build Docker image
echo "Building Docker image..."
docker build -t ratelimit:latest .

# Push to registry (adjust for your registry)
# docker tag ratelimit:latest your-registry/ratelimit:latest
# docker push your-registry/ratelimit:latest

# Deploy with Helm
echo "Deploying with Helm..."
helm upgrade --install ratelimit ./helm/ratelimit \
    --namespace $NAMESPACE \
    --create-namespace \
    --set image.repository=ratelimit \
    --set image.tag=latest \
    --set config.redis.addr=redis.$NAMESPACE.svc.cluster.local:6379

# Wait for deployment
echo "Waiting for deployment..."
kubectl rollout status deployment ratelimit -n $NAMESPACE --timeout=120s

# Apply EnvoyFilter
echo "Applying EnvoyFilter..."
kubectl apply -f deploy/envoyfilter.yaml

# Restart gateway
echo "Restarting gateway..."
kubectl rollout restart deployment public-gateway-istio -n $NAMESPACE

echo "✓ RateLimit Operator deployed successfully"