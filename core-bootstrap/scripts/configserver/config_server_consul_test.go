package configserver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/netcracker/core-bootstrap/v2/scripts/consul"
	"github.com/netcracker/core-bootstrap/v2/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const secretPath = "/api/v1/namespaces/core-ns/secrets/config-server-consul-token"

type fakeKubernetes struct {
	status   int
	requests []string
}

func newFakeKubernetes(t *testing.T, status int) *fakeKubernetes {
	fake := &fakeKubernetes{status: status}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fake.requests = append(fake.requests, r.Method+" "+r.URL.Path)
		w.WriteHeader(fake.status)
	}))
	t.Cleanup(server.Close)
	previous := utils.K8sClient
	utils.K8sClient = kubernetes.NewForConfigOrDie(&rest.Config{Host: server.URL})
	t.Cleanup(func() { utils.K8sClient = previous })
	return fake
}

func TestExecute_K8sModeDeletesTheStaticTokenSecret(t *testing.T) {
	cases := []struct {
		name    string
		status  int
		wantErr bool
	}{
		{"the secret is deleted", http.StatusOK, false},
		{"a secret that is already gone is not an error", http.StatusNotFound, false},
		{"a refused delete fails the task", http.StatusForbidden, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			consulServer, address := newFakeConsul(t, http.StatusOK, authMethod)
			k8s := newFakeKubernetes(t, tc.status)
			task := newTask(t, address, map[string]string{"CONSUL_AUTH_MODE": "k8s"})

			err := task.Execute(context.Background())

			assert.Equal(t, tc.wantErr, err != nil, "Execute returned %v", err)
			assert.Equal(t, []string{"DELETE " + secretPath}, k8s.requests)
			assert.Empty(t, consulServer.requests)
		})
	}
}

func TestExecute_HybridFailsWithoutTouchingTheSecretWhenTheAuthMethodCannotBeRead(t *testing.T) {
	_, address := newFakeConsul(t, http.StatusForbidden, "Permission denied")
	k8s := newFakeKubernetes(t, http.StatusOK)
	task := newTask(t, address, map[string]string{"CONSUL_AUTH_MODE": "hybrid"})

	err := task.Execute(context.Background())

	assert.Error(t, err)
	assert.Empty(t, k8s.requests)
}

func TestExecute_LegacyGoesOnToTheStaticTokenAndKeepsTheSecret(t *testing.T) {
	consulServer, address := newFakeConsul(t, http.StatusOK, authMethod)
	k8s := newFakeKubernetes(t, http.StatusInternalServerError)
	task := newTask(t, address, map[string]string{"CONSUL_AUTH_MODE": "legacy"})

	err := task.Execute(context.Background())

	assert.Error(t, err)
	assert.Equal(t, []string{"GET " + secretPath}, k8s.requests)
	assert.Empty(t, consulServer.requests)
}

func TestExecute_DoesNothingWhenConsulIsDisabled(t *testing.T) {
	setAuthEnv(t, map[string]string{"CONSUL_AUTH_MODE": "k8s"})
	k8s := newFakeKubernetes(t, http.StatusOK)
	values := map[string]string{"NAMESPACE": "core-ns", "CONSUL_ENABLED": "false"}
	accessor := func(name string) string { return values[name] }
	consulConfigurer := consul.New()
	require.NoError(t, consulConfigurer.Configure(accessor))
	task := New(consulConfigurer)
	require.NoError(t, task.Configure(accessor))

	err := task.Execute(context.Background())

	assert.NoError(t, err)
	assert.Empty(t, k8s.requests)
}
