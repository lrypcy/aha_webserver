# 规则：API 契约

> 细则，权威规范见 `AGENTS.md` 第 6 节。

## 1. 统一响应体

`/api/v1` 下所有成功与失败都返回同一形状：

```json
{"code":0,"message":"ok","data":{},"request_id":"..."}
```

- `code`：业务码，成功固定 `0`。
- `message`：成功为 `"ok"`，失败为可读提示，不含堆栈、不含 SQL、不含密钥。
- `data`：业务数据；列表为 `{"items":[...],"total":123,"page":1,"page_size":20}`；失败可为 `null`。
- `request_id`：requestid 中间件生成，原样回传。
- 响应封装只由 `internal/response` 提供；禁止 `c.JSON(model.User)`、禁止 `fiber.Map{"error": ...}`。

## 2. 错误码表（定义在 `internal/response`）

| code | HTTP | 含义 | 典型场景 |
| --- | --- | --- | --- |
| 0 | 200 / 201 | 成功 | — |
| 1001 | 400 | 请求参数 / 请求体错误 | BodyParser 失败、校验失败、id 非整数 |
| 1002 | 400 | 分页参数非法 | page / page_size 越界或非整数 |
| 2001 | 401 | 未认证 | 缺失、无效、过期的 Bearer Token |
| 3001 | 403 | 权限不足 | `RequireRole("admin")` 拒绝非 admin |
| 4001 | 404 | 资源不存在 | `gorm.ErrRecordNotFound` |
| 5001 | 409 | 资源冲突 | 用户名 / 邮箱重复注册、非法状态流转 |
| 6001 | 429 | 请求过于频繁 | limiter 拒绝 |
| 9999 | 500 | 服务器内部错误 | 数据库错误、recover 兜底 |

规则：新增错误码只在 `internal/response` 扩展，不要散落到 controller；HTTP 状态码仍由 `c.Status(...)` 设置，不要全返回 200。

## 3. 分页

- 参数：`page`（从 1 开始，默认 1）、`page_size`（默认 20，上限 100）。
- 非整数 / 小于 1 / 超过上限 → `400` + code `1002`。
- 列表响应 `data` 必须含 `items` / `total` / `page` / `page_size`。
- `GET /api/v1/tasks` 额外支持 `job_id`、`status` 过滤；过滤字段走白名单，不拼 SQL。

## 4. 端点清单

| 方法 | 路径 | 鉴权 | 说明 |
| --- | --- | --- | --- |
| POST | `/api/v1/register` | 公开 | 注册，返回用户（不含密码） |
| POST | `/api/v1/login` | 公开 | 登录，返回 Token |
| GET | `/api/v1/me` | Bearer | 当前登录用户 |
| POST | `/api/v1/jobs` | Bearer | 创建 Job |
| GET | `/api/v1/jobs` | Bearer | Job 列表，分页 |
| GET | `/api/v1/jobs/:id` | Bearer | Job 详情 |
| DELETE | `/api/v1/jobs/:id` | Bearer + admin | 删除 Job |
| POST | `/api/v1/tasks` | Bearer | 创建 Task |
| GET | `/api/v1/tasks` | Bearer | Task 列表，分页 + `job_id` / `status` 过滤 |
| GET | `/api/v1/tasks/:id` | Bearer | Task 详情 |
| PATCH | `/api/v1/tasks/:id/status` | Bearer | 更新 Task 状态 |
| GET | `/health` | 公开 | 存活探针 |
| GET | `/ready` | 公开 | 就绪探针（含 DB 连通性） |
| GET | `/swagger/*` | 公开 | Swagger UI |

公开端点只有 `/register`、`/login`、`/health`、`/ready`、`/swagger/*`；其余 `/api/v1` 端点都要 Bearer Token，`DELETE /api/v1/jobs/:id` 额外要 admin。

## 5. 路由分组规则

```go
g := app.Group("/api/v1")
g.Post("/jobs", ctrl.Create)          // 对：/api/v1/jobs
g.Post("/api/v1/jobs", ctrl.Create)   // 错：/api/v1/api/v1/jobs
```

- 分组前缀只在 `internal/router` 建一次；各 controller 不自己建 `/api/v1` 分组。
- 全局中间件（requestid / logger / recover / cors / limiter）在 `internal/app/app.go` 注册一次；jwt 与 RequireRole 在 `internal/router` 按路由挂。
- 非版本化端点 `/health`、`/ready`、`/swagger/*` 不进 `/api/v1` 分组、不挂 jwt。

## 6. swag 注解模板

```go
// CreateJob 创建 Job
// @Summary 创建 Job
// @Tags Job
// @Accept json
// @Produce json
// @Param body body dto.CreateJobRequest true "请求体"
// @Success 201 {object} response.Envelope{data=dto.JobResponse}
// @Failure 400 {object} response.Envelope
// @Failure 401 {object} response.Envelope
// @Failure 500 {object} response.Envelope
// @Security BearerAuth
// @Router /api/v1/jobs [post]
func CreateJob(c *fiber.Ctx) error { ... }
```

- `@Router` 路径写完整 `/api/v1/...`；`@Param` / `@Success` 引用 `internal/dto` 与 `internal/response` 类型，不引用 model。
- 通用注解（`@title` `@version` `@BasePath /api/v1` `@host localhost:8080` `@securityDefinitions.apikey BearerAuth`）在 `main.go`；Swagger UI 路由 `/swagger/*` 在 `internal/app/app.go` 挂载。
- 注解变更后必须 `swag init --output api --parseInternal --parseDependency`。

## 7. 自检清单

- [ ] 所有响应经 `internal/response`，形状一致且带 `request_id`
- [ ] HTTP 状态码是标准值，未全返回 200
- [ ] 列表接口实现了 `page` / `page_size` 与 `total`
- [ ] 路由前缀无重复，公开 / 鉴权 / admin 边界正确
- [ ] swag 注解齐全且已重新生成 `api/`
