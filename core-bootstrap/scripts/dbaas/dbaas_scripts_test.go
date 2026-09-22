package dbaas

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func accessorFor(values map[string]string) func(string) string {
	return func(name string) string { return values[name] }
}

// configuration returns the mandatory Configure parameters, overridden by extra.
func configuration(extra map[string]string) map[string]string {
	values := map[string]string{
		"NAMESPACE":                              "core-ns",
		"API_DBAAS_ADDRESS":                      "http://dbaas-aggregator.dbaas:8080",
		"DBAAS_CLUSTER_DBA_CREDENTIALS_USERNAME": "cluster-dba",
		"DBAAS_CLUSTER_DBA_CREDENTIALS_PASSWORD": "secret",
	}
	for key, value := range extra {
		values[key] = value
	}
	return values
}

func TestConfigure_ReadsTheOperatorFlag(t *testing.T) {
	for _, tc := range []struct {
		value string
		want  bool
	}{
		{"true", true},
		{"TRUE", true},
		{"false", false},
		{"", false},
		{"yes", false},
	} {
		t.Run(tc.value, func(t *testing.T) {
			c := New()

			err := c.Configure(accessorFor(configuration(map[string]string{"DBAAS_OPERATOR_ENABLED": tc.value})))

			assert.NoError(t, err)
			assert.Equal(t, tc.want, c.dbaasOperatorEnabled)
		})
	}
}

// With the operator enabled, the database comes from an InternalDatabase resource, so nothing may
// be sent to dbaas-aggregator from here.
func TestCreateDatabase_SkipsRestWhenTheOperatorIsEnabled(t *testing.T) {
	aggregator := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request to dbaas-aggregator: %s %s", r.Method, r.URL.Path)
	}))
	defer aggregator.Close()
	c := &Configurer{Namespace: "core-ns", ApiDbaasAddress: aggregator.URL, dbaasOperatorEnabled: true}

	err := c.CreateDatabase(context.Background(), "control-plane", "control-plane-db-credentials", nil)

	assert.NoError(t, err)
}

// The classifier sent here identifies the database control-plane already uses. An InternalDatabase
// that replaces this call must declare the same classifier, so that an upgrade adopts the existing
// database rather than provisioning an empty one.
func TestGetOrCreateDb_RegistersTheServiceDatabase(t *testing.T) {
	var got struct {
		method, path, user, password string
		body                         map[string]interface{}
	}
	aggregator := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got.method, got.path = r.Method, r.URL.Path
		got.user, got.password, _ = r.BasicAuth()
		assert.NoError(t, json.NewDecoder(r.Body).Decode(&got.body))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"name":"dbaas_cp","connectionProperties":{"host":"pg","port":5432,"name":"dbaas_cp","username":"u","password":"p","role":"admin"}}`))
	}))
	defer aggregator.Close()
	c := &Configurer{Namespace: "core-ns", ApiDbaasAddress: aggregator.URL, Username: "cluster-dba", password: "secret"}

	props, err := c.getOrCreateDb(context.Background(), "control-plane")

	assert.NoError(t, err)
	assert.Equal(t, http.MethodPut, got.method)
	assert.Equal(t, "/api/v3/dbaas/core-ns/databases", got.path)
	assert.Equal(t, "cluster-dba", got.user)
	assert.Equal(t, "secret", got.password)
	assert.Equal(t, map[string]interface{}{"namespace": "core-ns", "microserviceName": "control-plane", "scope": "service"}, got.body["classifier"])
	assert.Equal(t, "postgresql", got.body["type"])
	assert.Equal(t, "control-plane", got.body["originService"])
	assert.Equal(t, "pg", props.Host)
	assert.Equal(t, 5432, props.Port)
	assert.Equal(t, "u", props.Username)
}

func TestGetOrCreateDb_RejectsAnErrorStatus(t *testing.T) {
	aggregator := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer aggregator.Close()
	c := &Configurer{Namespace: "core-ns", ApiDbaasAddress: aggregator.URL}

	_, err := c.getOrCreateDb(context.Background(), "control-plane")

	assert.ErrorContains(t, err, "Wrong dbaas response status")
}
