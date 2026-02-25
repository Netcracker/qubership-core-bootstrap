# cr-synchronizer

cr-synchronizer is a lightweight controller component that watches and synchronizes Custom Resources (CRs) used by the qubership platform. It ensures desired CR state is propagated, reconciled, and recorded with structured events and labels. 

## Istio gateway handling
For correct routing through Istio Gateway and ensure backward compatibility were kept, fallback routes have been included. After installing, we need to update existing services to switch to these fallback routes accordingly. So helm weight-based approach is used, CR syncronizer waits fallback routes have parents and correct status.conditions

### Local development

Create configuration for HTTPRoute with the proper weight (before postintall job) and lables watched by cr-synchronizer, eg:
```
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: '{{ .Values.PRIVATE_GW_ROUTE_NAME }}'
  annotations:
    helm.sh/hook: "pre-install, pre-upgrade"
    helm.sh/hook-weight: "-60"
    helm.sh/hook-delete-policy: "before-hook-creation"
  labels:
    app.kubernetes.io/name: '{{ .Values.SERVICE_NAME }}'
    deployment.netcracker.com/sessionId: '{{ .Values.DEPLOYMENT_SESSION_ID }}'
spec:
  parentRefs:
  - name: '{{ .Values.ISTIO_PRIVATE_GATEWAY_NAME }}'
  rules:
  - matches:
    - path:
        type: PathPrefix
        value: /
    backendRefs:
    - name: '{{ .Values.PRIVATE_FB_SERVICE_NAME }}'
      port: 8080
---
```
Services are pointing to the gateway where this route is configured should use post-install hook and have a weight more then "100" to be applied after cr-synchronizer postinstall job, eg:
```
kind: Service
apiVersion: v1
metadata:
    name: '{{ .Values.PRIVATE_SERVICE_NAME }}'
    annotations:
      helm.sh/hook: "post-install, post-upgrade"
      helm.sh/hook-weight: "110"
      helm.sh/hook-delete-policy: "before-hook-creation"
...      
```
Set or pass the values used by the template: `CR_SYNCHRONIZER_IMAGE`, `SERVICE_MESH_TYPE`, `DEPLOYMENT_SESSION_ID`, `CHECK_DECLARATION_PLURALS` (if needed), `SERVICE_NAME`, etc.

Example `values.yaml` (minimum for startup):

```yaml
CR_SYNCHRONIZER_IMAGE: "your-registry/cr-synchronizer:latest"

# Enable Istio-mode inside synchronizer job
SERVICE_MESH_TYPE: ISTIO

# Session id / service name used by the template
DEPLOYMENT_SESSION_ID: "postdeploy-{{ .Release.Revision }}"
SERVICE_NAME: "test-service"
APPLICATION_NAME: "test-app"

# Optional: list of plurals to process
CHECK_DECLARATION_PLURALS: "httproutes"
RESOURCE_POLLING_TIMEOUT: 300
```


**Installing the chart (example):**

```bash
helm upgrade --install test-app ./test-chart \
  --set CR_SYNCHRONIZER_IMAGE="cr-synchronizer:latest" \
  --set SERVICE_MESH_TYPE="ISTIO" \
  --set DEPLOYMENT_SESSION_ID="session-123" \
  --set SERVICE_NAME="test-service" \
  --namespace controller-namespace
```

**Verification after installation:**

Make sure the desired Gateway and HTTPRoute exist in the namespace:

```bash
# adjust resource names accordingly
kubectl -n controller-namespace get gateway,myroute    
kubectl -n controller-namespace get httproute
```

Check logs of the Job/Pod for debugging:

```bash
kubectl -n controller-namespace logs job/synchronizer-postinstall-job-name
# or, if it's a Deployment/Pod:
kubectl -n controller-namespace logs deploy/cr-synchronizer
kubectl -n controller-namespace logs finalyzer-postinstall-job-core-istio-mesh-7mmtl
```

Verify that the service has been applied:

```bash
kubectl -n controller-namespace get svc test-service
```
