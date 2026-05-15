# Cloud E2E test scenarios — rate limit through gateway

Test files: `tests/cloud-e2e/rate_limit_test.go`, `tests/cloud-e2e/composite_limits_test.go`  
Run commands: `make test-cloud-e2e` · `make test-cloud-e2e-layered`  
Transport: HTTP requests through Istio Gateway with `x-user-id` header, real Redis

---

## TestCloudE2E_RateLimitThroughGateway

### Setup

Applied config via ConfigMap `cloud-e2e-rttg`:

```yaml
domain: auth_limit
separator: "|"
descriptors:
  - key: user_id
    value_regex: "cloud-e2e-user"
    rate_limit:
      unit: minute
      requests_per_unit: 2
    algorithm: fixed_window
    priority: 100
```

`POST /api/v1/config/reload` is called immediately after to force reconciliation.

---

### 1. First two requests are allowed

Applied: 2x `GET /test` through gateway with `x-user-id: cloud-e2e-user`.

| Assert | Expected |
|---|---|
| `HTTP status == 200` | both requests pass |

---

### 2. Third request is rate limited

Applied: 3rd `GET /test` with the same user.

| Assert | Expected |
|---|---|
| `HTTP status == 429` | limit of 2/min is exhausted |

---

### 3. Violating users appear in the API

Applied: `GET /api/v1/users/violating` on the operator.

| Assert | Expected |
|---|---|
| response contains `cloud-e2e-user` | user is tracked as violating |

---

### 4. Rate limit reset restores access

Applied: `POST /api/v1/users/cloud-e2e-user/reset`, then one more `GET /test`.

| Assert | Expected |
|---|---|
| `HTTP status == 200` | counter is zeroed, request passes |

---

### 5. Redis key is created for the rate-limited user

Applied: `KEYS *` on Redis after the test run.

| Assert | Expected |
|---|---|
| at least 1 key matching `ratelimit:*cloud-e2e-user*` | Redis counter exists |

---

## TestCloudE2E_TwoUsersRateLimit

### Setup

Applied config via ConfigMap `cloud-e2e-two-users`:

```yaml
domain: auth_limit
separator: "|"
descriptors:
  - key: user_id
    value_regex: "bad-user"
    rate_limit:
      unit: minute
      requests_per_unit: 30
    algorithm: fixed_window
    priority: 100
  - key: user_id
    value_regex: "good-user"
    rate_limit:
      unit: minute
      requests_per_unit: 1000
    algorithm: fixed_window
    priority: 100
```

Both rules use `priority: 100` to override any cluster-level rules (typically `priority ≤ 50`).

---

### 1. Good user is rarely rate limited

Applied: 50x `GET /test` through gateway with `x-user-id: good-user` (50ms delay between requests).

| Assert | Expected |
|---|---|
| `rate-limited count ≤ 5` | 1000/min limit is high enough that almost all 50 requests pass |

---

### 2. Bad user is rate limited after ~30 requests

Applied: 50x `GET /test` through gateway with `x-user-id: bad-user`.

| Assert | Expected |
|---|---|
| `rate-limited count ≈ 20 (±10)` | limit of 30/min exhausted; ~20 out of 50 are blocked |

---

### 3. Rate limits are isolated per user

Applied: compare `good-user` and `bad-user` counters after both runs.

| Assert | Expected |
|---|---|
| good-user blocked ≤ 5 AND bad-user blocked > 15 | counters are independent, not shared |

---

### 4. Reset restores access for bad user

Applied: `POST /api/v1/users/bad-user/reset`, then 10x `GET /test` for bad-user.

| Assert | Expected |
|---|---|
| `success count > 8` | counter is zeroed, requests pass again |

---

## TestCloudE2E_CompositeLimits

**Build tag:** `cloud_e2e_layered` · **Runtime:** ~2 minutes  
**Run command:** `make test-cloud-e2e-layered`

### Setup

Applied config via ConfigMap `cloud-e2e-composite` (layered / nested descriptors):

```yaml
domain: auth_limit
separator: "|"
descriptors:
  - key: path
    value: /api/v1/orders
    rate_limit:
      unit: minute
      requests_per_unit: 1000
    algorithm: fixed_window
    priority: 50
    descriptors:
      - key: user_id
        rate_limit:
          unit: minute
          requests_per_unit: 10
        algorithm: fixed_window
        priority: 100
```

Two-level rule: outer cap = 1000 requests/min across all users on `path=/api/v1/orders`, inner cap = 10 requests/min per individual `user_id`.

---

### Step 1: 100 users × 10 requests = 1000 all allowed

Applied: `GET /api/v1/orders` for each of 100 unique user IDs (`layered-user-000` … `layered-user-099`), 10 requests each.

| Assert | Expected |
|---|---|
| `allowed ≈ 1000 (±20)` | inner per-user limit (10) not exceeded; outer cap (1000) exactly filled |

---

### Step 2: exhausted user is blocked by inner rule

Applied: 11th `GET /api/v1/orders` for `layered-user-000` (already used all 10).

| Assert | Expected |
|---|---|
| `HTTP status == 429` | inner rule (10/min per user_id) triggers |

---

### Step 3: new user is blocked by outer rule

Applied: first `GET /api/v1/orders` for `layered-eve` (101st unique user, outer cap already full).

| Assert | Expected |
|---|---|
| `HTTP status == 429` | outer rule (1000/min for path) triggers even for a fresh user |

---

### Step 4: window rollover restores access

Applied: wait 65 seconds for the minute window to expire, then one `GET /api/v1/orders` for `layered-user-000`.

| Assert | Expected |
|---|---|
| `HTTP status == 200` | counters reset after window rollover |
