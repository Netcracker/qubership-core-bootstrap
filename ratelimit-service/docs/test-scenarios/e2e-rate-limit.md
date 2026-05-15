# E2E test scenarios — rate limit through operator API

Test file: `tests/e2e/rate_limit_test.go`  
Run command: `make test-e2e`  
Entry point: `TestE2E_RateLimitThroughOperatorAPI`

---

## Setup

A `LocalOperator` is started (Redis at `localhost:6379`, API at `:8083`).  
A ConfigMap is created in the cluster with the following rate limit config:

```yaml
domain: auth_limit
separator: "|"
descriptors:
  - key: user_id
    value_regex: "e2e-test-user"
    rate_limit:
      unit: minute
      requests_per_unit: 2
    algorithm: fixed_window
    priority: 100
  - key: user_id
    value_regex: "other-test-user"
    rate_limit:
      unit: minute
      requests_per_unit: 100
    algorithm: fixed_window
    priority: 100
```

Both rules use `priority: 100` to ensure they always override any cluster-level `ratelimit-config` rules (typically `priority ≤ 50`).  
`POST /api/v1/config/reload` is called immediately after to force reconciliation without waiting for the watcher debounce.  
Redis keys for the test user are cleared before the run.

---

## Scenarios

### 1. First two requests are allowed

Applied: 2x `POST /api/v1/ratelimit/check` with `user_id=e2e-test-user`, `path=/test`.

| Assert | Expected |
|---|---|
| `allowed == true` | both requests pass |
| `limit == 2` | the per-user rule is matched |

---

### 2. Third request is rejected (limit exhausted)

Applied: a third `POST /api/v1/ratelimit/check` with the same components.

| Assert | Expected |
|---|---|
| `allowed == false` | 2/minute limit is exhausted |

---

### 3. Another user is not affected

Applied: `POST /api/v1/ratelimit/check` with `user_id=other-test-user`, `path=/test` — after the first user has exhausted their limit.

| Assert | Expected |
|---|---|
| `allowed == true` | counters are per-user, not shared |
| `limit == 100` | the `other-test-user` rule (priority=100) is matched |

---

### 4. Rate limit reset

Applied: `POST /api/v1/users/e2e-test-user/reset`, then one more check request.

| Assert | Expected |
|---|---|
| `allowed == true` | counter is zeroed after reset |

---

## Helper test — `TestHelperFunctions`

No external dependencies. Validates internal utility correctness.

### ValidatePattern

Applied: `regexp.Compile` on a set of patterns.

| Pattern | Expected |
|---|---|
| `.*user_id=test.*`, `user_id=test`, `^.*user_id=test.*$` | `err == nil` |
| `*user_id=test*`, `[invalid`, `?invalid` | `err != nil` |

### BuildKey

Applied: `buildKeyForTest(components, separator)` with `{"user_id": "test", "path": "/api"}`.

| Assert | Expected |
|---|---|
| `Contains("user_id=test")` | key-value pair is present |
| `Contains("path=/api")` | both components are included |
| `Contains("\|")` | separator is applied |
| `len(key1) == len(key2)` | output is stable across calls |
| `Contains("&")` with `separator="&"` | separator is configurable |

### BuildKeyStable

Applied: `buildKeyForTest` with `{"z_key": "last", "a_key": "first", "m_key": "middle"}`.

| Assert | Expected |
|---|---|
| `Contains("a_key=first")` | all components present |
| `Contains("m_key=middle")` | all components present |
| `Contains("z_key=last")` | all components present |
