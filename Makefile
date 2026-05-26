# ═══════════════════════════════════════════════════════════════
#  top-search — Makefile
#  github.com/shamhi/top-search
# ═══════════════════════════════════════════════════════════════

RESET   := $(shell tput sgr0 2>/dev/null || echo "")
RED     := $(shell tput setaf 1 2>/dev/null || echo "")
GREEN   := $(shell tput setaf 2 2>/dev/null || echo "")
YELLOW  := $(shell tput setaf 3 2>/dev/null || echo "")
BLUE    := $(shell tput setaf 4 2>/dev/null || echo "")
MAGENTA := $(shell tput setaf 5 2>/dev/null || echo "")
CYAN    := $(shell tput setaf 6 2>/dev/null || echo "")
BOLD    := $(shell tput bold 2>/dev/null || echo "")

GO_VERSION   := 1.26
GO_MODULE    := github.com/shamhi/top-search

GOLANGCI_LINT_VERSION := v2.12.2
GCI_VERSION           := v0.14.0
GOFUMPT_VERSION       := v0.10.0
BUF_VERSION           := v1.69.0

BIN_DIR      := $(PWD)/bin

GOLANGCI_LINT := $(BIN_DIR)/golangci-lint
GCI           := $(BIN_DIR)/gci
GOFUMPT       := $(BIN_DIR)/gofumpt
BUF           := $(BIN_DIR)/buf

DOCKER_COMPOSE_FILE := deploy/docker-compose.yml
DOCKER_ENV_FILE     := config/.env

SERVER_CONTAINER := top-search
NATS_CONTAINER   := nats
REDIS_CONTAINER  := redis

.DEFAULT_GOAL := help

# ═══════════════════════════════════════════════════════════════
#  📋 HELP
# ═══════════════════════════════════════════════════════════════

.PHONY: help
help: ## Показать все команды
	@printf "$(GREEN)$(BOLD)🏔  top-search $(RESET)\n"
	@printf "$(CYAN)Команды:$(RESET)\n"
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  $(YELLOW)%-20s$(RESET) %s\n", $$1, $$2}'

# ═══════════════════════════════════════════════════════════════
#  🛠 TOOLS
# ═══════════════════════════════════════════════════════════════

.PHONY: install-tools
install-tools: ## Установить все dev-инструменты в bin/
	@mkdir -p $(BIN_DIR)
	@if [ ! -f $(GOFUMPT) ]; then \
		printf '$(GREEN)$(BOLD)  ↓$(RESET) Installing gofumpt $(GOFUMPT_VERSION)...\n'; \
		GOBIN=$(BIN_DIR) go install mvdan.cc/gofumpt@$(GOFUMPT_VERSION); \
	fi
	@if [ ! -f $(GCI) ]; then \
		printf '$(GREEN)$(BOLD)  ↓$(RESET) Installing gci $(GCI_VERSION)...\n'; \
		GOBIN=$(BIN_DIR) go install github.com/daixiang0/gci@$(GCI_VERSION); \
	fi
	@if [ ! -f $(GOLANGCI_LINT) ]; then \
		printf '$(GREEN)$(BOLD)  ↓$(RESET) Installing golangci-lint $(GOLANGCI_LINT_VERSION)...\n'; \
		GOBIN=$(BIN_DIR) go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION); \
	fi
	@if [ ! -f $(BUF) ]; then \
		printf '$(GREEN)$(BOLD)  ↓$(RESET) Installing buf $(BUF_VERSION)...\n'; \
		GOBIN=$(BIN_DIR) go install github.com/bufbuild/buf/cmd/buf@$(BUF_VERSION); \
	fi
	@printf '$(GREEN)$(BOLD)✓$(RESET) All tools installed\n'

# ═══════════════════════════════════════════════════════════════
#  🎨 FORMAT
# ═══════════════════════════════════════════════════════════════

.PHONY: fmt
fmt: install-tools ## Форматировать код (gofumpt + gci)
	@printf '$(BLUE)$(BOLD)  ♻$(RESET) gofumpt...\n'
	@find . -type f -name '*.go' \
		! -path '*/vendor/*' ! -path '*/gen/*' ! -path '*/mocks/*' \
		-exec $(GOFUMPT) -extra -w {} +
	@printf '$(GREEN)$(BOLD)  ✓$(RESET) gofumpt done\n'
	@printf '$(BLUE)$(BOLD)  ♻$(RESET) gci...\n'
	@find . -type f -name '*.go' \
		! -path '*/vendor/*' ! -path '*/gen/*' ! -path '*/mocks/*' \
		-exec $(GCI) write -s standard -s default -s "Prefix($(GO_MODULE))" {} +
	@printf '$(GREEN)$(BOLD)  ✓$(RESET) gci done\n'

.PHONY: lint
lint: install-tools ## Запустить golangci-lint
	@printf '$(BLUE)$(BOLD)  🔍$(RESET) golangci-lint...\n'
	@$(GOLANGCI_LINT) run --timeout=5m ./...
	@printf '$(GREEN)$(BOLD)  ✓$(RESET) Lint passed\n'

.PHONY: lint-fix
lint-fix: install-tools ## Запустить golangci-lint с автофиксом
	@printf '$(BLUE)$(BOLD)  🔧$(RESET) golangci-lint --fix...\n'
	@$(GOLANGCI_LINT) run --timeout=5m --fix ./...
	@printf '$(GREEN)$(BOLD)  ✓$(RESET) Lint + fix done\n'

.PHONY: check
check: fmt lint ## Полная проверка кода (fmt + lint)

# ═══════════════════════════════════════════════════════════════
#  🐳 DOCKER
# ═══════════════════════════════════════════════════════════════

.PHONY: dc-up
dc-up: ## Запустить контейнеры в foreground
	@printf '$(GREEN)$(BOLD)  ▶$(RESET) Starting containers...\n'
	docker compose --env-file $(DOCKER_ENV_FILE) -f $(DOCKER_COMPOSE_FILE) up

.PHONY: dc-upd
dc-upd: ## Запустить контейнеры в фоне
	@printf '$(GREEN)$(BOLD)  ▶$(RESET) Starting containers (detached)...\n'
	docker compose --env-file $(DOCKER_ENV_FILE) -f $(DOCKER_COMPOSE_FILE) up -d

.PHONY: dc-down
dc-down: ## Остановить контейнеры
	@printf '$(RED)$(BOLD)  ⏹$(RESET) Stopping containers...\n'
	docker compose --env-file $(DOCKER_ENV_FILE) -f $(DOCKER_COMPOSE_FILE) down

.PHONY: dc-build
dc-build: ## Собрать образы
	@printf '$(BLUE)$(BOLD)  🔨$(RESET) Building images...\n'
	docker compose --env-file $(DOCKER_ENV_FILE) -f $(DOCKER_COMPOSE_FILE) build

.PHONY: dc-rebuild
dc-rebuild: ## Пересобрать с нуля (--no-cache)
	@printf '$(BLUE)$(BOLD)  🔨$(RESET) Rebuilding (--no-cache)...\n'
	docker compose --env-file $(DOCKER_ENV_FILE) -f $(DOCKER_COMPOSE_FILE) build --no-cache

.PHONY: dc-restart
dc-restart: dc-down dc-upd ## Перезапустить все контейнеры

.PHONY: dc-restart-server
dc-restart-server: ## Перезапустить сервер
	@printf '$(YELLOW)$(BOLD)  🔄$(RESET) Restarting $(SERVER_CONTAINER)...\n'
	docker compose --env-file $(DOCKER_ENV_FILE) -f $(DOCKER_COMPOSE_FILE) restart $(SERVER_CONTAINER)

.PHONY: dc-status
dc-status: ## Статус контейнеров
	@printf '$(CYAN)$(BOLD)  ℹ$(RESET) Container status:\n'
	@docker compose --env-file $(DOCKER_ENV_FILE) -f $(DOCKER_COMPOSE_FILE) ps

.PHONY: dc-logs
dc-logs: ## Логи всех контейнеров
	docker compose --env-file $(DOCKER_ENV_FILE) -f $(DOCKER_COMPOSE_FILE) logs -f

.PHONY: dc-logs-server
dc-logs-server: ## Логи сервера
	docker compose --env-file $(DOCKER_ENV_FILE) -f $(DOCKER_COMPOSE_FILE) logs -f $(SERVER_CONTAINER)

.PHONY: dc-logs-nats
dc-logs-nats: ## Логи NATS
	docker compose --env-file $(DOCKER_ENV_FILE) -f $(DOCKER_COMPOSE_FILE) logs -f $(NATS_CONTAINER)

.PHONY: dc-logs-redis
dc-logs-redis: ## Логи Redis
	docker compose --env-file $(DOCKER_ENV_FILE) -f $(DOCKER_COMPOSE_FILE) logs -f $(REDIS_CONTAINER)

.PHONY: dc-shell-server
dc-shell-server: ## Shell в контейнере сервера
	docker compose --env-file $(DOCKER_ENV_FILE) -f $(DOCKER_COMPOSE_FILE) exec $(SERVER_CONTAINER) /bin/sh

.PHONY: dc-stats
dc-stats: ## Ресурсы контейнеров
	docker compose --env-file $(DOCKER_ENV_FILE) -f $(DOCKER_COMPOSE_FILE) stats --no-stream

.PHONY: dc-clean
dc-clean: ## Остановить + удалить volumes
	@printf '$(RED)$(BOLD)  🧹$(RESET) Removing containers + volumes...\n'
	docker compose --env-file $(DOCKER_ENV_FILE) -f $(DOCKER_COMPOSE_FILE) down -v

# ═══════════════════════════════════════════════════════════════
#  🧪 GO DEV
# ═══════════════════════════════════════════════════════════════

.PHONY: test
test: ## Запустить тесты
	@printf '$(BLUE)$(BOLD)  🧪$(RESET) Running tests...\n'
	@go test -race -shuffle=on -count=1 ./...
	@printf '$(GREEN)$(BOLD)  ✓$(RESET) All tests passed\n'

.PHONY: test-cover
test-cover: ## Тесты + coverage report
	@printf '$(BLUE)$(BOLD)  📊$(RESET) Running tests with coverage...\n'
	@go test -race -coverprofile=coverage.out ./...
	@go tool cover -func=coverage.out
	@go tool cover -html=coverage.out -o coverage.html
	@printf '$(GREEN)$(BOLD)  ✓$(RESET) Coverage → coverage.html\n'

.PHONY: vet
vet: ## Запустить go vet
	@printf '$(BLUE)$(BOLD)  🔍$(RESET) Running go vet...\n'
	@go vet ./...
	@printf '$(GREEN)$(BOLD)  ✓$(RESET) Vet passed\n'

.PHONY: tidy
tidy: ## go mod tidy
	@printf '$(BLUE)$(BOLD)  📦$(RESET) go mod tidy...\n'
	@go mod tidy
	@printf '$(GREEN)$(BOLD)  ✓$(RESET) Modules tidied\n'

.PHONY: build
build: ## Собрать Go приложение
	@printf '$(BLUE)$(BOLD)  🔨$(RESET) Building...\n'
	@go build -v ./...
	@printf '$(GREEN)$(BOLD)  ✓$(RESET) Build succeeded\n'

.PHONY: run
run: ## Запустить сервер локально
	@printf '$(GREEN)$(BOLD)  ▶$(RESET) Starting server...\n'
	@go run ./cmd/server

.PHONY: produce
produce: ## Запустить highload-продьюсер поисковых запросов
	@go run ./cmd/producer

.PHONY: produce-load
produce-load: ## Highload: 10K rps, 4 workers, 30s
	@go run ./cmd/producer -rate 10000 -workers 8 -duration 30s -batch 10

# ═══════════════════════════════════════════════════════════════
#  📐 PROTO
# ═══════════════════════════════════════════════════════════════

.PHONY: proto
proto: install-tools ## Сгенерировать proto код
	@printf '$(BLUE)$(BOLD)  📐$(RESET) Generating proto code...\n'
	@$(BUF) generate
	@printf '$(GREEN)$(BOLD)  ✓$(RESET) Proto generated\n'

# ═══════════════════════════════════════════════════════════════
#  🏃 QUICK START
# ═══════════════════════════════════════════════════════════════

.PHONY: dev
dev: install-tools proto dc-upd ## Полный dev-запуск (tools + proto + infra)
	@printf '$(GREEN)$(BOLD)  ✓$(RESET) Dev environment ready.\n'
	@printf '  gRPC:   localhost:50051\n'
	@printf '  NATS:   nats://localhost:4222\n'
	@printf '  Redis:  redis://localhost:6379\n'

.PHONY: metrics
metrics: ## Метрики проекта
	@printf '$(CYAN)$(BOLD)  📊 Project Metrics$(RESET)\n'
	@printf '  Go files:  %s\n' $$(find . -name '*.go' ! -path '*/gen/*' ! -path '*/vendor/*' | wc -l)
	@printf '  Go LOC:    %s\n' $$(find . -name '*.go' ! -path '*/gen/*' ! -path '*/vendor/*' -exec wc -l {} + | tail -1 | awk '{print $$1}')
	@printf '  Packages:  %s\n' $$(go list ./... | wc -l)
	@printf '  Deps:      %s\n' $$(go list -m all | wc -l)
