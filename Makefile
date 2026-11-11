# aha_webserver 工程化构建入口
# 默认目标：直接执行 make 或 make help 查看所有可用目标

.DEFAULT_GOAL := help

BIN_DIR    := bin
BINARY     := $(BIN_DIR)/aha_webserver
SWAG_DIR   := api
IMAGE      := aha_webserver:latest
COMPOSE    := docker-compose.yml
DOCKERFILE := build/package/Dockerfile
EDP        := ../easy_deploy/bin/easy_deploy
DEPLOY_TARGET ?= local

.PHONY: help fmt tidy lint build run test test-cover swag swag_clean docker-build docker-up docker-down clean \
        deploy-check deploy-plan deploy-up deploy-local deploy-down deploy-status deploy-logs deploy-render

help: ## 显示所有可用目标（默认目标）
	@echo "aha_webserver —— 用法: make <target>"
	@echo ""
	@echo "  help         显示本帮助（默认目标）"
	@echo "  fmt          gofmt -w . 格式化全部 Go 源码"
	@echo "  tidy         go mod tidy 整理依赖"
	@echo "  lint         go vet ./... 静态检查"
	@echo "  build        go build -o bin/aha_webserver ."
	@echo "  run          go run .（默认 sqlite 零配置，端口 8080）"
	@echo "  test         go test ./..."
	@echo "  test-cover   -race + coverage.out 并打印函数覆盖率"
	@echo "  swag         swag init 生成 api/ 下的接口文档"
	@echo "  swag_clean   清理 swag 临时残留（保留生成物，不影响 go build）"
	@echo "  docker-build 构建镜像 aha_webserver:latest"
	@echo "  docker-up    docker compose up -d --build"
	@echo "  docker-down  docker compose down"
	@echo "  deploy-check   easy_deploy validate + doctor + plan（DEPLOY_TARGET=xxx）"
	@echo "  deploy-plan    easy_deploy plan --target DEPLOY_TARGET"
	@echo "  deploy-local   本机裸机部署（等价于 DEPLOY_TARGET=local deploy-up）"
	@echo "  deploy-status / deploy-logs / deploy-down"
	@echo "  deploy-render  render --target k8s 刷新 deploy/k8s/ 快照"
	@echo "  clean        删除 bin/、coverage.out、dist/"

fmt: ## gofmt -w .
	gofmt -w .

tidy: ## go mod tidy
	go mod tidy

lint: ## go vet ./...
	go vet ./...

build: ## go build -o bin/aha_webserver .
	@mkdir -p $(BIN_DIR)
	go build -o $(BINARY) .

run: ## go run .
	go run .

test: ## go test ./...
	go test ./...

test-cover: ## go test -race -coverprofile=coverage.out ./... 并打印函数级覆盖率
	go test -race -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

swag: ## swag init --output api --parseInternal --parseDependency
	@command -v swag >/dev/null 2>&1 || { echo "未找到 swag，请先执行: go install github.com/swaggo/swag/cmd/swag@latest"; exit 1; }
	swag init --output $(SWAG_DIR) --parseInternal --parseDependency

swag_clean: ## 只清理临时残留，保留 api/ 三个生成物，确保 go build 不受影响
	@echo "说明：api/docs.go、api/swagger.json、api/swagger.yaml 是 swag 生成物；"
	@echo "      docs.go 被 main.go 导入，删除会导致 go build 失败，因此本目标不删除它们。"
	@echo "      如需更新文档，请直接执行 make swag（同名文件会被覆盖）。"
	@echo "      下面仅清理明显的临时残留（*.tmp / *.bak / *~）："
	@find $(SWAG_DIR) -maxdepth 1 -type f -name '*.tmp' -print -delete 2>/dev/null || true
	@find $(SWAG_DIR) -maxdepth 1 -type f -name '*.bak' -print -delete 2>/dev/null || true
	@find $(SWAG_DIR) -maxdepth 1 -type f -name '*~' -print -delete 2>/dev/null || true

docker-build: ## docker build 多阶段镜像
	docker build -f $(DOCKERFILE) -t $(IMAGE) .

docker-up: ## docker compose up -d --build
	docker compose -f $(COMPOSE) up -d --build

docker-down: ## docker compose down
	docker compose -f $(COMPOSE) down

# ---------- easy_deploy 统一部署（DEPLOY_TARGET=local|systemd|ssh|docker|compose|k8s） ----------
# 先在 ../easy_deploy 执行 make build 生成 $(EDP)。
deploy-check: ## validate + doctor + plan（只预演，不执行）
	@test -x $(EDP) || { echo "缺少 $(EDP)，请先在 ../easy_deploy 执行 make build"; exit 1; }
	$(EDP) validate --json
	$(EDP) doctor --target $(DEPLOY_TARGET) --json
	$(EDP) plan --target $(DEPLOY_TARGET) --json

deploy-plan: ## 只输出 Plan（隐含 dry-run）
	@test -x $(EDP) || { echo "缺少 $(EDP)"; exit 1; }
	$(EDP) plan --target $(DEPLOY_TARGET) --json

deploy-up: ## 部署到 DEPLOY_TARGET
	@test -x $(EDP) || { echo "缺少 $(EDP)"; exit 1; }
	$(EDP) up --target $(DEPLOY_TARGET) --yes --json

deploy-local: DEPLOY_TARGET = local
deploy-local: deploy-up ## 本机裸机部署

deploy-down: ## 按 state 回收资源
	@test -x $(EDP) || { echo "缺少 $(EDP)"; exit 1; }
	$(EDP) down --target $(DEPLOY_TARGET) --yes --json

deploy-status: ## 查询状态与健康
	@test -x $(EDP) || { echo "缺少 $(EDP)"; exit 1; }
	$(EDP) status --target $(DEPLOY_TARGET) --json

deploy-logs: ## 查看最近 100 行日志
	@test -x $(EDP) || { echo "缺少 $(EDP)"; exit 1; }
	$(EDP) logs --target $(DEPLOY_TARGET) --tail 100 --json

deploy-render: ## 刷新 deploy/k8s/ 快照（Secret 文件不入库）
	@test -x $(EDP) || { echo "缺少 $(EDP)"; exit 1; }
	$(EDP) render --target k8s --out deploy/k8s
	@rm -f deploy/k8s/*-secret.yaml
	@echo "已刷新 deploy/k8s/（含密钥的 Secret 已排除）。"

clean: ## 清理 bin/、coverage.out、dist/
	go clean
	rm -rf $(BIN_DIR) coverage.out dist
	@echo "已清理构建产物。"
