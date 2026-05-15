# tests/helpers/

Переиспользуемый Go-код для тестов: фабрики miniredis, загрузка YAML-фикстур, выделение свободных портов, создание тестовых ConfigMap-ов.

В фазе 2a (текущей) каталог создан пустым.
В фазе 2b сюда будут добавлены:

- `miniredis.go` — стандартный setup miniredis + RedisClient + RateLimitManager.
- `fixtures.go` — загрузка YAML из `tests/fixtures/`, парсинг в `[]*ratelimit.Rule`.
- `ports.go` — выделение свободного TCP-порта для тестов.
- `configmap.go` — создание тестового ConfigMap через `fake.Clientset` или реальный k8s API; `SetRules(t, clientset, namespace, rules)` с `t.Cleanup`.

Хелперы используются из `tests/integration/`, `tests/e2e/`, `tests/cloud-e2e/`.
