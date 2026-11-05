package configserver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/netcracker/core-bootstrap/v2/scripts/consul"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type consulRequest struct {
	path  string
	token string
}

type fakeConsul struct {
	status   int
	body     string
	requests []consulRequest
}

func newFakeConsul(t *testing.T, status int, body string) (*fakeConsul, string) {
	fake := &fakeConsul{status: status, body: body}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fake.requests = append(fake.requests, consulRequest{path: r.URL.Path, token: r.Header.Get("X-Consul-Token")})
		w.WriteHeader(fake.status)
		_, _ = w.Write([]byte(fake.body))
	}))
	t.Cleanup(server.Close)
	return fake, server.URL
}

type taskActions struct {
	issuedTokens   int
	deletedSecrets []string
}

func newTask(t *testing.T, consulEnabled bool, consulAddress string, env map[string]string) (*Configurer, *taskActions) {
	actions := &taskActions{}
	task := New(&consul.Configurer{Enabled: consulEnabled, Address: consulAddress})
	task.issueToken = func(context.Context) error {
		actions.issuedTokens++
		return nil
	}
	task.deleteSecret = func(_ context.Context, namespace string, secretName string) error {
		actions.deletedSecrets = append(actions.deletedSecrets, namespace+"/"+secretName)
		return nil
	}
	values := map[string]string{"NAMESPACE": "core-ns", "CONSUL_ADMIN_TOKEN": "admin-token"}
	for name, value := range env {
		values[name] = value
	}
	require.NoError(t, task.Configure(func(name string) string { return values[name] }))
	return task, actions
}

const authMethodWithNetcracker = `{"Name":"applications-k8s-m2m","Type":"jwt","Config":{"BoundAudiences":["https://kubernetes.default.svc.cluster.local","netcracker"]}}`

func TestExecute_ChoosesTheActionByAuthMode(t *testing.T) {
	cases := []struct {
		name           string
		consulAuthMode string
		m2mAuthMode    string
		wantIssued     int
		wantDeleted    []string
	}{
		{"legacy issues the token", "legacy", "", 1, nil},
		{"k8s deletes the secret", "k8s", "", 0, []string{"core-ns/config-server-consul-token"}},
		{"an unset consul auth mode follows M2M_AUTH_MODE", "", "k8s", 0, []string{"core-ns/config-server-consul-token"}},
		{"consul auth mode overrides M2M_AUTH_MODE", "legacy", "k8s", 1, nil},
		{"M2M_AUTH_MODE is matched case-insensitively", "", " K8S ", 0, []string{"core-ns/config-server-consul-token"}},
		{"no mode set at all is legacy", "", "", 1, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			consulServer, address := newFakeConsul(t, http.StatusOK, authMethodWithNetcracker)
			task, actions := newTask(t, true, address, map[string]string{
				"CONSUL_AUTH_MODE": tc.consulAuthMode,
				"M2M_AUTH_MODE":    tc.m2mAuthMode,
			})

			require.NoError(t, task.Execute(context.Background()))

			assert.Equal(t, tc.wantIssued, actions.issuedTokens)
			assert.Equal(t, tc.wantDeleted, actions.deletedSecrets)
			assert.Empty(t, consulServer.requests)
		})
	}
}

func TestExecute_HybridChoosesTheActionByTheConsulAuthMethod(t *testing.T) {
	cases := []struct {
		name        string
		status      int
		body        string
		wantIssued  int
		wantDeleted []string
	}{
		{"the auth method accepts the audience", http.StatusOK, authMethodWithNetcracker, 0, []string{"core-ns/config-server-consul-token"}},
		{"the auth method does not accept the audience", http.StatusOK,
			`{"Name":"applications-k8s-m2m","Type":"jwt","Config":{"BoundAudiences":["https://kubernetes.default.svc.cluster.local"]}}`, 1, nil},
		{"the auth method has no bound audiences", http.StatusOK, `{"Name":"applications-k8s-m2m","Type":"jwt","Config":{}}`, 1, nil},
		{"Consul has no such auth method", http.StatusNotFound, "ACL auth method not found", 1, nil},
		{"Consul ACLs are disabled", http.StatusUnauthorized, "ACL support disabled", 0, []string{"core-ns/config-server-consul-token"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			consulServer, address := newFakeConsul(t, tc.status, tc.body)
			task, actions := newTask(t, true, address, map[string]string{"CONSUL_AUTH_MODE": "hybrid"})

			require.NoError(t, task.Execute(context.Background()))

			assert.Equal(t, tc.wantIssued, actions.issuedTokens)
			assert.Equal(t, tc.wantDeleted, actions.deletedSecrets)
			assert.Equal(t, []consulRequest{{path: "/v1/acl/auth-method/applications-k8s-m2m", token: "admin-token"}},
				consulServer.requests)
		})
	}
}

func TestExecute_HybridFailsWhenTheAuthMethodCannotBeRead(t *testing.T) {
	cases := []struct {
		name   string
		status int
		body   string
	}{
		{"the admin token is refused", http.StatusForbidden, "Permission denied"},
		{"unauthorized for another reason", http.StatusUnauthorized, "ACL not found"},
		{"401 whose body only contains the ACL disabled text", http.StatusUnauthorized, "ACL support disabled for this token"},
		{"Consul fails", http.StatusInternalServerError, "rpc error"},
		{"the auth method is not JSON", http.StatusOK, "not json"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, address := newFakeConsul(t, tc.status, tc.body)
			task, actions := newTask(t, true, address, map[string]string{"CONSUL_AUTH_MODE": "hybrid"})

			err := task.Execute(context.Background())

			assert.Error(t, err)
			assert.Equal(t, 0, actions.issuedTokens)
			assert.Empty(t, actions.deletedSecrets)
		})
	}
}

func TestExecute_HybridFailsWhenConsulIsUnreachable(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	address := server.URL
	server.Close()
	task, actions := newTask(t, true, address, map[string]string{"CONSUL_AUTH_MODE": "hybrid"})

	err := task.Execute(context.Background())

	assert.Error(t, err)
	assert.Equal(t, 0, actions.issuedTokens)
	assert.Empty(t, actions.deletedSecrets)
}

func TestExecute_HybridAsksForTheConfiguredAuthMethodWithTheAdminToken(t *testing.T) {
	consulServer, address := newFakeConsul(t, http.StatusOK,
		`{"Name":"custom-method","Type":"jwt","Config":{"BoundAudiences":["custom-audience"]}}`)
	task, actions := newTask(t, true, address, map[string]string{
		"CONSUL_AUTH_MODE":     "hybrid",
		"CONSUL_AUTH_METHOD":   "custom-method",
		"CONSUL_AUTH_AUDIENCE": "custom-audience",
	})

	require.NoError(t, task.Execute(context.Background()))

	assert.Equal(t, []consulRequest{{path: "/v1/acl/auth-method/custom-method", token: "admin-token"}}, consulServer.requests)
	assert.Equal(t, []string{"core-ns/config-server-consul-token"}, actions.deletedSecrets)
}

func TestExecute_DoesNothingWhenConsulIsDisabled(t *testing.T) {
	for _, mode := range []string{"legacy", "hybrid", "k8s"} {
		t.Run(mode, func(t *testing.T) {
			consulServer, address := newFakeConsul(t, http.StatusOK, authMethodWithNetcracker)
			task, actions := newTask(t, false, address, map[string]string{"CONSUL_AUTH_MODE": mode})

			require.NoError(t, task.Execute(context.Background()))

			assert.Equal(t, 0, actions.issuedTokens)
			assert.Empty(t, actions.deletedSecrets)
			assert.Empty(t, consulServer.requests)
		})
	}
}

func TestConfigure_RejectsAnUnknownMode(t *testing.T) {
	cases := []struct {
		name           string
		consulAuthMode string
		m2mAuthMode    string
	}{
		{"unknown consul auth mode", "static", "legacy"},
		{"unknown M2M_AUTH_MODE with consul auth mode unset", "", "static"},
		{"consul auth mode in upper case", "K8S", "legacy"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			values := map[string]string{
				"NAMESPACE":        "core-ns",
				"CONSUL_AUTH_MODE": tc.consulAuthMode,
				"M2M_AUTH_MODE":    tc.m2mAuthMode,
			}
			task := New(&consul.Configurer{Enabled: true})

			err := task.Configure(func(name string) string { return values[name] })

			assert.Error(t, err)
		})
	}
}
