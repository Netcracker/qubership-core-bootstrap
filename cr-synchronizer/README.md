# cr-synchronizer

cr-synchronizer is a lightweight controller component that watches and synchronizes Custom Resources (CRs) used by the qubership platform. It ensures desired CR state is propagated, reconciled, and recorded with structured events and labels. 

## Istio gateway handling
For correct routing through Istio Gateway and ensure backward compatibility were kept, fallback routes have been included. After installing, we need to update existing services to switch to these fallback routes accordingly. So the gateway_service_generator was added to check preconditions before services switching

### Local development

Place the files with the desired Service manifests into the chart's `declarations/` so that the Helm expression `.Files.Glob "declarations/*"` in `_synchronizer.yaml` includes them in a ConfigMap.  
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
CHECK_DECLARATION_PLURALS: "services,gateways"
RESOURCE_POLLING_TIMEOUT: 300
```

**How to organize declaratives (chart structure):**

Create a `declarations/` folder in your chart, and add YAML files there (each file can contain one or multiple objects separated by `---`).  
The `_synchronizer.yaml` template already does:
- `{{ $filesExist := (.Files.Glob "declarations/*") }}` — if files exist, it creates a ConfigMap named `synchronizer.transport.configmap` and includes all files as data entries.
- The `synchronizer.postinstall.job` mounts this ConfigMap into the container: the volume `declarations-{{ .Values.SERVICE_NAME }}` is mounted at `/mnt/declaratives`.

**Example declaration file (`declarations/test-gateway-services.yaml`):**

```yaml
apiVersion: v1
kind: Service
metadata:
  name: test-service
  annotations:
    gateway.target: "test-gateway-name"
    gateway.route: "test-route-name"
spec:
  selector:
    app: test-app
  ports:
    - protocol: TCP
      port: 8080
      targetPort: 8080
---
```
where gateway.target and gateway.route are preconditions to process services declarations. These resources must be deployed first

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
kubectl -n controller-namespace logs deploy/my-cr-synchronizer
```

Verify that the service has been applied:

```bash
kubectl -n controller-namespace get svc test-service
```
