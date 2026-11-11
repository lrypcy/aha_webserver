---
name: add-endpoint
description: "在 aha_webserver 新增 /api/v1 接口的可执行步骤：DTO、controller、router、swag 注解、测试、文档与提交前检查。"
---

# 技能：新增接口

适用：新增 `/api/v1` 下任意端点。开始前读 `AGENTS.md` 第 6、9 节与 `.agents/rules/api-contract.md`。

## 步骤

1. **定契约并写下来**：方法、完整路径（带 `/api/v1` 前缀）、是否 Bearer、是否 admin、请求/响应字段、是否分页、涉及的错误码。契约定完再动手，不要边写边改。
2. **写 DTO**：在 `internal/dto/` 建 `xxx.go`，定义 `XxxRequest` / `XxxResponse`。字段带 `json` 与 `validate` 标签；**不要写 `binding`**（Fiber v2 不认）。只放对外字段，禁止 `Password` / 哈希 / 内部字段，禁止直接复用 `internal/model` 的结构体。
3. **写 handler**：在 `internal/controller/<domain>/` 建 `xxx.go`，签名 `func Xxx(c *fiber.Ctx) error`：
   - `c.BodyParser(&req)` 失败 → `internal/response` 的 `400` + code `1001`；
   - 再跑 `validate` 校验，失败同样 `400` / `1001`；
   - 查库用 `db.WithContext(c.UserContext())`，`errors.Is(err, gorm.ErrRecordNotFound)` → `404` / `4001`；
   - 成功走 `internal/response` 的统一响应体，不要自己拼 `fiber.Map`。
4. **挂路由**：在 `internal/router/` 的 `/api/v1` 分组下注册。公开路由不挂 jwt；需认证挂 jwt 中间件；需 admin 再挂 `RequireRole("admin")`。**子路由不要再写 `/api/v1` 前缀**（会变成 `/api/v1/api/v1/...`）。
   ```go
   g := app.Group("/api/v1")
   g.Post("/jobs", jobCtrl.Create)                       // 对
   g.Delete("/jobs/:id", jwt.Middleware(), auth.RequireRole("admin"), jobCtrl.Delete)
   ```
5. **写 swag 注解**（贴在 handler 上方）：`@Summary` `@Tags` `@Accept json` `@Produce json` `@Param` `@Success` `@Failure` `@Router /api/v1/xxx [method]`，需认证补 `@Security BearerAuth`。`@Router` 路径写完整前缀；`@Param` / `@Success` 引用 `internal/dto` 与 `internal/response` 类型。
6. **生成文档**：`swag init --output api --parseInternal --parseDependency`（或 `make swag`）。确认 `api/docs.go`、`api/swagger.json`、`api/swagger.yaml` 三个文件都更新了。
7. **写测试**：同包 `xxx_test.go`，表驱动，至少三类用例：正常 `201`/`200`、参数错误 `400`、无 Token `401`；admin 接口再加非 admin `403`。用 `app.Test` 不起真实端口，断言 `request_id` 非空。
8. **更新文档**：在 `docs/api-guide.md` 加对应 curl 示例（含 `Authorization: Bearer`）。
9. **跑检查清单**：`gofmt -w .` → `go vet ./...` → `go test ./...` → `go build ./...`。

## 完成定义（DoD）

- [ ] 端点出现在 `/api/v1` 下，前缀无重复，鉴权边界正确
- [ ] 出入参全走 `internal/dto`，响应经 `internal/response` 且带 `request_id`
- [ ] 错误码只在 `internal/response` 扩展，HTTP 状态码是标准值
- [ ] swag 注解齐全，`api/` 三个生成物已重新生成并一并提交
- [ ] 测试覆盖正常 / 参数错误 / 鉴权失败三类，`go test ./...` 通过
- [ ] `docs/api-guide.md` 已同步，四条检查命令全绿
