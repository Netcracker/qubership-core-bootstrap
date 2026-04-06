package api

import (
    "bytes"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"

    "ratelimit-service/pkg/controller"
    "ratelimit-service/pkg/ratelimit"

    "github.com/gorilla/mux"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "k8s.io/client-go/kubernetes/fake"
)

func setupTestServer(t *testing.T) (*Server, *ratelimit.RateLimitManager, func()) {

    mockRateLimitManager := ratelimit.NewRateLimitManager(nil)

    mockRateLimitManager.AddRule(&ratelimit.Rule{
        Name:      "test_rule",
        Pattern:   "/test",
        Limit:     10,
        Window:    60,
        Algorithm: ratelimit.AlgorithmSlidingWindow,
    })

    clientset := fake.NewSimpleClientset()
    mockController := controller.NewConfigMapController(clientset, nil, mockRateLimitManager)

    server := NewServer(nil, mockController, mockRateLimitManager)

    cleanup := func() {

        mockRateLimitManager.ClearRules()
    }

    return server, mockRateLimitManager, cleanup
}

func TestHealthCheck(t *testing.T) {
    server, _, cleanup := setupTestServer(t)
    defer cleanup()

    req, err := http.NewRequest("GET", "/health", nil)
    require.NoError(t, err)

    rr := httptest.NewRecorder()
    server.router.ServeHTTP(rr, req)

    assert.Equal(t, http.StatusOK, rr.Code)

    var response map[string]string
    err = json.Unmarshal(rr.Body.Bytes(), &response)
    require.NoError(t, err)
    assert.Equal(t, "healthy", response["status"])
}

func TestReadinessCheck(t *testing.T) {
    server, _, cleanup := setupTestServer(t)
    defer cleanup()

    req, err := http.NewRequest("GET", "/ready", nil)
    require.NoError(t, err)

    rr := httptest.NewRecorder()
    server.router.ServeHTTP(rr, req)

    assert.Equal(t, http.StatusOK, rr.Code)

    var response map[string]string
    err = json.Unmarshal(rr.Body.Bytes(), &response)
    require.NoError(t, err)
    assert.Equal(t, "ready", response["status"])
}

func TestAuthenticate(t *testing.T) {
    tests := []struct {
        name           string
        apiKey         string
        requestKey     string
        expectedStatus int
    }{
        {
            name:           "no auth required when api key empty",
            apiKey:         "",
            requestKey:     "",
            expectedStatus: http.StatusOK,
        },
        {
            name:           "valid api key",
            apiKey:         "test-key-123",
            requestKey:     "test-key-123",
            expectedStatus: http.StatusOK,
        },
        {
            name:           "invalid api key",
            apiKey:         "test-key-123",
            requestKey:     "wrong-key",
            expectedStatus: http.StatusUnauthorized,
        },
        {
            name:           "missing api key",
            apiKey:         "test-key-123",
            requestKey:     "",
            expectedStatus: http.StatusUnauthorized,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            server := &Server{
                router: mux.NewRouter(),
                apiKey: tt.apiKey,
            }

            server.router.HandleFunc("/test", server.authenticate(func(w http.ResponseWriter, r *http.Request) {
                w.WriteHeader(http.StatusOK)
            })).Methods("GET")

            req, err := http.NewRequest("GET", "/test", nil)
            require.NoError(t, err)

            if tt.requestKey != "" {
                req.Header.Set("X-API-Key", tt.requestKey)
            }

            rr := httptest.NewRecorder()
            server.router.ServeHTTP(rr, req)

            assert.Equal(t, tt.expectedStatus, rr.Code)
        })
    }
}

func TestRespondWithJSON(t *testing.T) {
    rr := httptest.NewRecorder()
    data := map[string]string{"test": "value"}

    respondWithJSON(rr, http.StatusCreated, data)

    assert.Equal(t, http.StatusCreated, rr.Code)
    assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

    var response map[string]string
    err := json.Unmarshal(rr.Body.Bytes(), &response)
    require.NoError(t, err)
    assert.Equal(t, "value", response["test"])
}

func TestNotFoundRoute(t *testing.T) {
    server, _, cleanup := setupTestServer(t)
    defer cleanup()

    req, err := http.NewRequest("GET", "/nonexistent", nil)
    require.NoError(t, err)

    rr := httptest.NewRecorder()
    server.router.ServeHTTP(rr, req)

    assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestRateLimitRulesAPI(t *testing.T) {
    server, _, cleanup := setupTestServer(t)
    defer cleanup()

    rule := map[string]interface{}{
        "name":       "new_test_rule",
        "pattern":    "/api/test",
        "limit":      10,
        "window_sec": 60,
        "algorithm":  "sliding_window",
    }

    body, _ := json.Marshal(rule)
    req, err := http.NewRequest("POST", "/api/v1/ratelimit/rules", bytes.NewBuffer(body))
    require.NoError(t, err)

    rr := httptest.NewRecorder()
    server.router.ServeHTTP(rr, req)

    assert.Equal(t, http.StatusCreated, rr.Code)

    var response map[string]string
    err = json.Unmarshal(rr.Body.Bytes(), &response)
    require.NoError(t, err)
    assert.Equal(t, "success", response["status"])

    req, err = http.NewRequest("GET", "/api/v1/ratelimit/rules", nil)
    require.NoError(t, err)

    rr = httptest.NewRecorder()
    server.router.ServeHTTP(rr, req)

    assert.Equal(t, http.StatusOK, rr.Code)

    req, err = http.NewRequest("DELETE", "/api/v1/ratelimit/rules/new_test_rule", nil)
    require.NoError(t, err)

    rr = httptest.NewRecorder()
    server.router.ServeHTTP(rr, req)

    assert.Equal(t, http.StatusOK, rr.Code)
}

func TestCheckRateLimit(t *testing.T) {
    server, _, cleanup := setupTestServer(t)
    defer cleanup()

    reqBody := map[string]interface{}{
        "components": map[string]string{
            "path":    "/test",
            "user_id": "alice",
        },
    }

    body, _ := json.Marshal(reqBody)
    req, err := http.NewRequest("POST", "/api/v1/ratelimit/check", bytes.NewBuffer(body))
    require.NoError(t, err)

    rr := httptest.NewRecorder()
    server.router.ServeHTTP(rr, req)

    assert.Equal(t, http.StatusOK, rr.Code)

    var result map[string]interface{}
    err = json.Unmarshal(rr.Body.Bytes(), &result)
    require.NoError(t, err)

    hasAllowed := false
    if _, ok := result["Allowed"]; ok {
        hasAllowed = true
    }
    if _, ok := result["allowed"]; ok {
        hasAllowed = true
    }

    assert.True(t, hasAllowed, "Result should contain 'Allowed' or 'allowed' field")
    t.Logf("Rate limit check result: %+v", result)
}

func TestReloadConfig(t *testing.T) {
    server, _, cleanup := setupTestServer(t)
    defer cleanup()

    req, err := http.NewRequest("POST", "/api/v1/config/reload", nil)
    require.NoError(t, err)

    rr := httptest.NewRecorder()
    server.router.ServeHTTP(rr, req)

    assert.Contains(t, []int{http.StatusOK, http.StatusInternalServerError}, rr.Code)
}
