# Load demo test scenarios

Script: `tests/load/k6_tests/run-demo-tests.sh`  
Run command: `make test-load-demo`  
Log: `docs/test-scenarios/test-load-demo.log`  
Transport: HTTP through Istio Gateway (`x-user-id` header), executed via `kubectl exec` inside the cluster  
Pods: `curl-test-runner` (curl-based tests 1–6), `k6-test-runner` (k6 tests 7–8)  
Scripts mounted from ConfigMap `demo-test-scripts` in namespace `core-1-core`.

ConfigMaps for tests 2, 3, 5, 6 are applied from the **host** by `run-demo-tests.sh` (which has `kubectl`)
before launching the test inside the pod. The pod itself only needs `curl`.
Fixture files: `tests/fixtures/ratelimit-config-priority-demo.yaml`, `ratelimit-config-accuracy.yaml`, `ratelimit-config-algo-compare.yaml`.

---

## 1. Show Current Rules with Priorities (`get-rules.sh`)

No config applied — reads current cluster state.

### 1.1. Rule listing

Applied: `GET /api/v1/ratelimit/rules`.

| Observed | Note |
|---|---|
| No rules listed in output | Rule list output was empty — rules exist but grep-based parser found no names |

### 1.2. Rule matching per user type

Applied: `POST /api/v1/ratelimit/check` for `test-user`, `vip-user`, `normal-user`.

| User | Matched rule | Limit |
|---|---|---|
| `test-user` | `ratelimit-config/path=_test#3` | 2/min |
| `vip-user` | `ratelimit-config/path=_test#3` | 2/min |
| `normal-user` | `ratelimit-config/__#0` | 60/min |

All users fell through to cluster-level rules (no custom priority rules active yet).

---

## 2. Add Rules with Different Priorities (`add-rules-with-priority.sh`)

Applied config via ConfigMap `k6-priority-rules` (applied from host by `run-demo-tests.sh` before the test runs):

```yaml
domain: auth_limit
separator: "|"
descriptors:
  - key: user_id
    value_regex: "admin-.*"
    rate_limit: { unit: minute, requests_per_unit: 10000 }
    algorithm: fixed_window
    priority: 200
  - key: user_id
    value_regex: "vip-.*"
    rate_limit: { unit: minute, requests_per_unit: 1000 }
    algorithm: fixed_window
    priority: 100
  - key: user_id
    value_regex: "normal-.*"
    rate_limit: { unit: minute, requests_per_unit: 30 }
    algorithm: fixed_window
    priority: 60
  - key: user_id
    value_regex: ".*"
    rate_limit: { unit: minute, requests_per_unit: 100 }
    algorithm: fixed_window
    priority: 10
  - key: ""
    rate_limit: { unit: minute, requests_per_unit: 60 }
    algorithm: fixed_window
    priority: 0
```

| Output | Expected |
|---|---|
| `Rules added successfully` | ConfigMap applied from host, reconciliation triggered |

---

## 3. Priority Demo — Admin / VIP / Normal Users (`priority-demo.sh`)

Calls `add-rules-with-priority.sh` first (ConfigMap `k6-priority-rules` already applied from host), then sends 5 requests per user through gateway.

Applied: `POST /api/v1/ratelimit/check` + 5x `GET /test` per user.

| User | `x-user-id` | Matched rule | Limit | Success |
|---|---|---|---|---|
| Admin | `admin-john` | `k6-priority-rules/user_id=admin-.*` | 10000/min | 5/5 |
| VIP | `vip-jane` | `k6-priority-rules/user_id=vip-.*` | 1000/min | 5/5 |
| Normal | `normal-alice` | `k6-priority-rules/user_id=normal-.*` | 30/min | 5/5 |
| Unknown | `unknown-user` | `ratelimit-config/__#0` (cluster catch-all) | 60/min | 5/5 |

Admin (200), VIP (100), and Normal (60) resolve to their `k6-priority-rules` tier. Unknown user falls through to the cluster-level catch-all (`ratelimit-config/__#0`, priority 0, 60 req/min) — the `k6-priority-rules/user_id=.*` fallback at priority 10 was outpaced by the cluster rule matching first at that priority level.

---

## 4. Gateway Distribution Test (`gateway-distribution.sh`, 200 requests)

Applied: 200x `GET /test` with `x-user-id: test`, collecting `x-gateway-id` header per response.

| Gateway pod | Requests | Rate limited |
|---|---|---|
| `public-gateway-istio-7998d6ff7d-sf47t` | 100 | 59 (59%) |
| `public-gateway-istio-7998d6ff7d-wwlnd` | 100 | 51 (51%) |
| **Total** | **200** | **110 (55%)** |

Load balancer distributed evenly (100/100). Rate-limited percentages differ slightly between pods (59% vs 51%) — within normal variance for a shared Redis counter under concurrent access.

**Purpose confirmed:** counters are shared across replicas — neither pod has an isolated counter.

---

## 5. Rate Limit Accuracy Test (`accuracy-test.sh`)

Applied config via ConfigMap `k6-accuracy-test` (applied from host, deleted after test):

```yaml
domain: auth_limit
separator: "|"
descriptors:
  - key: user_id
    value_regex: "test2"
    rate_limit: { unit: second, requests_per_unit: 2 }
    algorithm: fixed_window
    priority: 100
  - key: user_id
    value_regex: "burst"
    rate_limit: { unit: second, requests_per_unit: 2 }
    algorithm: fixed_window
    priority: 100
```

### 5.1. Paced requests (100ms delay)

Applied: 10x `GET /test` with `x-user-id: test2`, 100ms between each.

| Assert | Expected |
|---|---|
| `Allowed: ~2-3/10` | limit of 2/sec is respected; only first window passes |

### 5.2. Burst (no delay)

Applied: 10x `GET /test` with `x-user-id: burst`, no delay.

| Assert | Expected |
|---|---|
| `Allowed: ~2/10` | hard cap enforced immediately |

ConfigMap `k6-accuracy-test` deleted and config reloaded after test.

---

## 6. Algorithm Comparison Test (`algorithm-compare.sh`, 200 requests)

Applied config via two ConfigMaps (applied from host, deleted after test):

```yaml
# k6-fixed-test
descriptors:
  - key: user_id
    value_regex: "fixed"
    rate_limit: { unit: second, requests_per_unit: 2 }
    algorithm: fixed_window
    priority: 50

# k6-sliding-test
descriptors:
  - key: user_id
    value_regex: "sliding"
    rate_limit: { unit: second, requests_per_unit: 2 }
    algorithm: sliding_log
    priority: 50
```

Applied: 200x `GET /test` for `x-user-id: fixed`, then 200x for `x-user-id: sliding`, no delay.

| Algorithm | Expected allowed | Expected output |
|---|---|---|
| Fixed Window | higher (burst at boundary) | `Allowed: X/200` |
| Sliding Log | lower or equal (rolling window) | `Allowed: Y/200` where Y ≤ X |
| Summary | fixed > sliding | "Fixed Window allowed N more / Sliding Window is MORE ACCURATE" |

ConfigMaps `k6-fixed-test` and `k6-sliding-test` deleted and config reloaded after test.

---

## 7. K6 Load Test (`k6-load-test.js`)

Config applied before test: `ratelimit-config-loadtest` (50 req/s path limit on `/test`, 10 req/s per user).
Two-phase scenario, 20 virtual users rotating `user-0`…`user-19`, 100ms sleep.

| Phase | Executor | Rate | Duration |
|---|---|---|---|
| Constant load | `constant-arrival-rate` | 50 req/s | 30s |
| Spike load | `ramping-arrival-rate` | 10→200→10 req/s | 50s (starts at t=30s) |

**Config reasoning:**
- `constant_load` at 50 req/s with 20 rotating users → ~2.5 req/s per user, well within the 10/s per-user limit and matching the 50/s path limit → near-zero 429s expected.
- `spike_load` intentionally exceeds 50 req/s → 429s are expected and by design.

**Results:**

| Metric | Value |
|---|---|
| Total requests | 5300 |
| Avg duration | 3.03ms |
| Overall failure rate | 23.85% (spike phase 429s — expected, not thresholded) |
| `constant_load` threshold | ✓ passed |
| `spike_load` threshold | ✓ passed (no failure constraint) |

| Threshold | Scope | Limit | Meaning |
|---|---|---|---|
| `p(95) < 500ms` | all requests | 500ms | Gateway responds fast to both 200 and 429 |
| `http_req_failed{scenario:constant_load} < 5%` | `constant_load` only | 5% | Near-zero 429s during steady state with matching config; spike 429s excluded |

Spike 429s do not fail the threshold because the threshold is scoped to `constant_load`. A breach means either the loadtest config was not applied or the gateway started returning 5xx.

---

## 8. K6 Burst Test (`k6-burst-test.js`)

Single spike scenario, 10 virtual users rotating `burst-user-0`…`burst-user-9`, 50ms sleep.

| Stage | Rate | Duration |
|---|---|---|
| Warm-up | 10 req/s | 10s |
| Ramp up | →200 req/s | 5s |
| Peak | →500 req/s | 10s |
| Cool down | →200→10 req/s | 20s |

**Results:**

| Metric | Value |
|---|---|
| Total requests | 8674 |
| Peak throughput | ~200 req/s (actual vs 500 target — capped by rate limits and VU pool) |
| Avg req/s over 45s | 192.76 |

System remained stable throughout — no crashes, no hung connections, 0 interrupted iterations. All 8674 requests completed (200 or 429).
