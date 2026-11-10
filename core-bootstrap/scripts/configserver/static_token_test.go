package configserver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/netcracker/core-bootstrap/v2/scripts/consul"
	"github.com/netcracker/qubership-core-lib-go/v3/configloader"
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

func setAuthEnv(t *testing.T, env map[string]string) {
	for _, name := range []string{"CONSUL_AUTH_MODE", "M2M_AUTH_MODE", "CONSUL_AUTH_METHOD", "CONSUL_AUTH_AUDIENCE"} {
		t.Setenv(name, env[name])
	}
	configloader.Init(configloader.EnvPropertySource())
	t.Cleanup(func() { configloader.Init() })
}

func newTask(t *testing.T, consulAddress string, env map[string]string) *Configurer {
	setAuthEnv(t, env)
	values := map[string]string{
		"NAMESPACE":          "core-ns",
		"CONSUL_ENABLED":     "true",
		"CONSUL_PUBLIC_URL":  consulAddress,
		"CONSUL_ADMIN_TOKEN": "admin-token",
	}
	accessor := func(name string) string { return values[name] }
	consulConfigurer := consul.New()
	require.NoError(t, consulConfigurer.Configure(accessor))
	task := New(consulConfigurer)
	require.NoError(t, task.Configure(accessor))
	return task
}

const authMethodWithNetcracker = `{"Name":"applications-k8s-m2m","Type":"jwt","Config":{"BoundAudiences":["https://kubernetes.default.svc.cluster.local","netcracker"]}}`

func TestStaticTokenNeeded_ByMode(t *testing.T) {
	cases := []struct {
		name string
		mode string
		want bool
	}{
		{"legacy needs the token", "legacy", true},
		{"k8s needs no token", "k8s", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			consulServer, address := newFakeConsul(t, http.StatusOK, authMethodWithNetcracker)
			task := newTask(t, address, map[string]string{"CONSUL_AUTH_MODE": tc.mode})

			needed, err := task.staticTokenNeeded(context.Background())

			require.NoError(t, err)
			assert.Equal(t, tc.want, needed)
			assert.Empty(t, consulServer.requests)
		})
	}
}

func TestStaticTokenNeeded_HybridDecidesByTheConsulAuthMethod(t *testing.T) {
	cases := []struct {
		name   string
		status int
		body   string
		want   bool
	}{
		{"the auth method accepts the audience", http.StatusOK, authMethodWithNetcracker, false},
		{"the auth method does not accept the audience", http.StatusOK,
			`{"Name":"applications-k8s-m2m","Type":"jwt","Config":{"BoundAudiences":["https://kubernetes.default.svc.cluster.local"]}}`, true},
		{"Consul has no such auth method", http.StatusNotFound, "ACL auth method not found", true},
		{"Consul ACLs are disabled", http.StatusUnauthorized, "ACL support disabled", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			consulServer, address := newFakeConsul(t, tc.status, tc.body)
			task := newTask(t, address, map[string]string{"CONSUL_AUTH_MODE": "hybrid"})

			needed, err := task.staticTokenNeeded(context.Background())

			require.NoError(t, err)
			assert.Equal(t, tc.want, needed)
			assert.Equal(t, []consulRequest{{path: "/v1/acl/auth-method/applications-k8s-m2m", token: "admin-token"}},
				consulServer.requests)
		})
	}
}

func TestStaticTokenNeeded_HybridFailsWhenTheAuthMethodCannotBeRead(t *testing.T) {
	cases := []struct {
		name   string
		status int
		body   string
	}{
		{"the admin token is refused", http.StatusForbidden, "Permission denied"},
		{"401 whose body adds words to the ACL disabled text", http.StatusUnauthorized, "ACL support disabled for this token"},
		{"the auth method is not JSON", http.StatusOK, "not json"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, address := newFakeConsul(t, tc.status, tc.body)
			task := newTask(t, address, map[string]string{"CONSUL_AUTH_MODE": "hybrid"})

			_, err := task.staticTokenNeeded(context.Background())

			assert.Error(t, err)
		})
	}
}

func TestStaticTokenNeeded_HybridFailsWhenConsulIsUnreachable(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	address := server.URL
	server.Close()
	task := newTask(t, address, map[string]string{"CONSUL_AUTH_MODE": "hybrid"})

	_, err := task.staticTokenNeeded(context.Background())

	assert.Error(t, err)
}

func TestStaticTokenNeeded_HybridAsksForTheConfiguredAuthMethodAndAudience(t *testing.T) {
	consulServer, address := newFakeConsul(t, http.StatusOK,
		`{"Name":"custom-method","Type":"jwt","Config":{"BoundAudiences":["custom-audience"]}}`)
	task := newTask(t, address, map[string]string{
		"CONSUL_AUTH_MODE":     "hybrid",
		"CONSUL_AUTH_METHOD":   "custom-method",
		"CONSUL_AUTH_AUDIENCE": "custom-audience",
	})

	needed, err := task.staticTokenNeeded(context.Background())

	require.NoError(t, err)
	assert.False(t, needed)
	assert.Equal(t, []consulRequest{{path: "/v1/acl/auth-method/custom-method", token: "admin-token"}}, consulServer.requests)
}

func TestConfigure_RequiresTheAdminTokenOnlyOutsideK8s(t *testing.T) {
	values := map[string]string{"NAMESPACE": "core-ns", "CONSUL_ENABLED": "true"}
	accessor := func(name string) string { return values[name] }
	for mode, wantErr := range map[string]bool{"hybrid": true, "k8s": false} {
		t.Run(mode, func(t *testing.T) {
			setAuthEnv(t, map[string]string{"CONSUL_AUTH_MODE": mode})
			consulConfigurer := consul.New()
			require.NoError(t, consulConfigurer.Configure(accessor))

			err := New(consulConfigurer).Configure(accessor)

			assert.Equal(t, wantErr, err != nil, "error: %v", err)
		})
	}
}

func TestConfigure_RejectsAnUnknownMode(t *testing.T) {
	setAuthEnv(t, map[string]string{"CONSUL_AUTH_MODE": "static"})
	values := map[string]string{"NAMESPACE": "core-ns"}
	task := New(consul.New())

	err := task.Configure(func(name string) string { return values[name] })

	assert.ErrorContains(t, err, "static")
}
