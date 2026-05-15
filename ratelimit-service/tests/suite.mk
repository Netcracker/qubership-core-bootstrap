# tests/suite.mk
# Общие переменные и хелперы для всех test-целей.

# Параметры окружения (можно переопределить из CI: NAMESPACE=... make ...).
NAMESPACE      ?= core-1-core
TEST_TIMEOUT   ?= 10m
PF_SCRIPT      := bash tests/scripts/port-forward.sh

# ANSI-цвета для вывода.
GREEN  := \033[0;32m
RED    := \033[0;31m
YELLOW := \033[1;33m
BLUE   := \033[0;34m
NC     := \033[0m

# Re-export для подскриптов.
export NAMESPACE

# Helper: запуск go test без интеграционных тегов.
define go_test_unit
	@go clean -testcache
	@go test -v -race -short -timeout $(TEST_TIMEOUT) $(1)
endef

# Helper: запуск go test с тегом integration (без port-forward — миниредис).
define go_test_integration
	@go clean -testcache
	@go test -v -tags=integration -timeout $(TEST_TIMEOUT) -run "TestIntegration" $(1)
endef
