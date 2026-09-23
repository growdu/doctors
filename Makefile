# 陪诊师平台 · 顶层 Makefile
# 用法：make help

SHELL := /bin/bash
GO    ?= go
DOCKER_COMPOSE ?= docker compose

SERVICES := auth order match

.PHONY: help
help: ## 显示所有命令
	@echo "Targets:"
	@echo "  tidy                 go mod tidy"
	@echo "  build                编译所有服务二进制到 bin/"
	@echo "  test                 单元测试"
	@echo "  test-integration     集成测试（需 docker compose up）"
	@echo "  lint                 golangci-lint"
	@echo "  fmt                  go fmt"
	@echo "  vet                  go vet"
	@echo "  docker-up            启动 PG + Redis + Kafka"
	@echo "  docker-down          停止并清理容器"
	@echo "  docker-logs          查看日志"
	@echo "  migrate              数据库迁移（占位）"
	@echo "  run-auth             启动 auth-service"
	@echo "  run-order            启动 order-service"
	@echo "  run-match            启动 match-service"
	@echo "  clean                清理 bin 和临时文件"

.PHONY: tidy
tidy: ## go mod tidy
	$(GO) mod tidy

.PHONY: build
build: ## 编译所有服务二进制到 bin/
	@mkdir -p bin
	@for svc in $(SERVICES); do \
		echo ">>> building $$svc"; \
		$(GO) build -o bin/$$svc ./services/$$svc/cmd; \
	done

.PHONY: test
test: ## 单元测试
	$(GO) test -race -count=1 ./...

.PHONY: test-integration
test-integration: ## 集成测试（需要 docker compose up）
	$(GO) test -race -count=1 -tags=integration ./...

.PHONY: lint
lint: ## golangci-lint
	golangci-lint run ./...

.PHONY: fmt
fmt: ## go fmt
	$(GO) fmt ./...

.PHONY: vet
vet: ## go vet
	$(GO) vet ./...

.PHONY: docker-up
docker-up: ## 启动 PG + Redis + Kafka
	$(DOCKER_COMPOSE) up -d
	@echo "等待服务就绪..."
	@$(DOCKER_COMPOSE) ps

.PHONY: docker-down
docker-down: ## 停止并清理容器
	$(DOCKER_COMPOSE) down -v

.PHONY: docker-logs
docker-logs: ## 查看日志
	$(DOCKER_COMPOSE) logs -f --tail=100

.PHONY: migrate
migrate: ## 数据库迁移（占位，下个阶段实装）
	@echo "TODO: integrate golang-migrate"

.PHONY: run-auth
run-auth: ## 启动 auth-service
	$(GO) run ./services/auth/cmd

.PHONY: run-order
run-order: ## 启动 order-service
	$(GO) run ./services/order/cmd

.PHONY: run-match
run-match: ## 启动 match-service
	$(GO) run ./services/match/cmd

.PHONY: clean
clean: ## 清理 bin 和临时文件
	rm -rf bin/ .data/ coverage.html