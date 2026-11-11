---
name: swagger-update
description: "改动 swag 注解、DTO 或模型后重新生成 api/ 下 Swagger 文档并验证的可执行步骤。"
---

# 技能：更新 Swagger 文档

适用：改了 `@Param` / `@Success` / `@Router` 注解、`internal/dto` 或 `internal/model` 之后。开始前读 `AGENTS.md` 第 4 节与 `.agents/rules/api-contract.md` 第 6 节。

## 步骤

1. **确认 swag 可用**：
   ```bash
   swag --version
   # 未安装时：
   go install github.com/swaggo/swag/cmd/swag@latest
   ```
2. **核对 `main.go` 的通用注解**：`@title`、`@version`、`@BasePath /api/v1`、`@host localhost:8080`、`@securityDefinitions.apikey BearerAuth`（`in: header`、`name: Authorization`）。`@BasePath` 必须是 `/api/v1`，不要留旧值 `/v2`。
3. **核对 UI 路由**：`GET /swagger/*` 在 `internal/app/app.go` 里通过 fiber-swagger 挂载（应用装配已集中到 `internal/app/app.go`，不再散在 `main.go`）。`/swagger/*` 不挂 jwt。
4. **逐个 handler 查注解**：`@Router` 写完整 `/api/v1/...` 路径；`@Param` / `@Success` 引用 `internal/dto` 与 `internal/response` 类型，不直接引用 `internal/model`；需认证的路由有 `@Security BearerAuth`。
5. **重新生成**：
   ```bash
   swag init --output api --parseInternal --parseDependency
   # 等价封装
   make swag
   ```
6. **确认三个生成物都变了**：`api/docs.go`、`api/swagger.json`、`api/swagger.yaml`。`api/` 是生成物，**禁止手改**；要改就改注解后重新生成。
7. **编译验证**：`go build ./...`。`main.go` 空导入 `github.com/lrypcy/aha_webserver/api`，若 `api/` 被 `make swag_clean` 清空且未重新生成，这里会直接失败。
8. **启动验证**：`AHA_SERVER_PORT=8080 go run .` 后访问 `http://localhost:8080/swagger/index.html` 与 `http://localhost:8080/swagger/doc.json`，确认新端点、新字段、`@BasePath /api/v1` 都已生效。
9. **跑检查清单**：`gofmt -w .` → `go vet ./...` → `go test ./...` → `go build ./...`。

## 完成定义（DoD）

- [ ] `api/` 三个生成物已重新生成并一并提交，无手工编辑痕迹
- [ ] `@BasePath` 为 `/api/v1`，所有 `@Router` 带完整前缀
- [ ] `go build ./...` 通过，`/swagger/index.html` 展示的是最新文档
- [ ] 四条检查命令全绿
