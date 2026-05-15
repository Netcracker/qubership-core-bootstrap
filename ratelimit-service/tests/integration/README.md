# tests/integration/

Цель каталога — integration-тесты, использующие miniredis (in-process Redis). Они не требуют kubectl/port-forward и запускаются изолированно.

## Файлы

| Файл | Что тестирует |
|------|---------------|
| `ratelimit_test.go` | RateLimitManager: Check, GetViolatingUsers |
| `controller_test.go` | ConfigMapController + RateLimitManager |
| `api_test.go` | HTTP-сервер (api.Server.CheckRateLimit) через httptest |

## Запуск

```bash
make test-integration
```

## Примечания

- **metrics_test.go is in tests/e2e/** (as `metrics_collector_test.go`, moved in phase 2c-I) — `metrics.NewMetricsCollectorService` требует Redis `INFO`-команд, которых нет в miniredis. Тест скипает себя, если реальный Redis недоступен.
- Все тесты здесь используют `helpers.NewEnv(t)` для получения miniredis-окружения.
- Build-tag: `//go:build integration`
