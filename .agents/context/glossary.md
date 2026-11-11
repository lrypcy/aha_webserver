# 背景：术语表

> 只读背景文档。命令 / 路径 / 环境变量名以 `AGENTS.md` 为准；枚举取值以 `internal/model` 实际定义为准。

## 领域对象

| 术语 | 含义 |
| --- | --- |
| `Job`（作业） | 聚合根，一个作业下挂多个 Task，完成后产出一份 JobResult。字段含 Title / Creator / Status / ErrorCode / Config / Label。 |
| `Task`（子任务） | 属于某个 Job（`JobID`），有自己的 Status / ErrorCode / Config / Label。`PATCH /api/v1/tasks/:id/status` 改的就是它的状态。 |
| `JobResult`（作业结果） | 与 Job 一对一，通过 `JobID` 关联；存 Output（主要输出）、Metrics（结构化指标）、Logs（执行日志）、Artifacts（产物路径列表）。 |
| `User` | 注册用户。`Username` / `Email` 唯一，`Password` 存 bcrypt 哈希，`LastLogin` / `IsActive` 为辅助字段。绝不整体对外返回。 |
| `SysConfig` | 系统配置表，`key` / `value` 形式，定义在 `internal/model/sys_config.go`。旧代码曾从该表读 `jwt_secret`，现已废弃（密钥只走 `AHA_JWT_SECRET`）。 |
| `Config`（字段） | Job / Task 上的结构化配置字段，类型化后用 `gorm:"serializer:json"` 存 JSON 文本。 |
| `Label`（字段） | 标签列表，字符串切片，同样用 `serializer:json` 存。禁止 `default:'[]'::jsonb` 这类 PostgreSQL 专有写法。 |

## 状态与错误

| 术语 | 含义 |
| --- | --- |
| `Status` | 类型化枚举 `type Status int`，描述 Job / Task 生命周期。现有取值语义为 Pending / Started / Running / Failure / Success（数值以 `internal/model` 为准），不要散落裸 int。 |
| `ErrorCode` | 类型化枚举 `type ErrorCode int`，表示作业 / 任务失败时的内部错误分类；与 HTTP 响应的 `code` 不是同一套东西。 |
| `code` | 统一响应体中的**业务码**，成功 `0`，错误见 `.agents/rules/api-contract.md` 错误码表（1001 / 1002 / 2001 / 3001 / 4001 / 5001 / 6001 / 9999）。定义在 `internal/response`。 |
| `message` | 响应体中的可读提示，成功 `"ok"`，不含堆栈 / SQL / 密钥。 |
| `data` | 响应体业务数据；列表含 `items` / `total` / `page` / `page_size`。 |
| `request_id` | requestid 中间件生成的请求标识，写入 `c.Locals` 并原样回传，用于日志检索。 |

## 认证与角色

| 术语 | 含义 |
| --- | --- |
| `JWT` | HS256 签发的 Token，`Authorization: Bearer <token>` 传递。 |
| `claims` | Token 载荷，含 `user_id` 与 `role`。 |
| `role` | 角色，只有 `user|admin` 两种。admin 才能 `DELETE /api/v1/jobs/:id`。 |
| `RequireRole` | `internal/middleware/auth` 提供的角色鉴权中间件，如 `RequireRole("admin")`。 |
| `Bearer Token` | `Authorization` 头的取值格式；jwt 中间件负责解析与校验。 |

## 接口与契约

| 术语 | 含义 |
| --- | --- |
| `/api/v1` | 全部业务 API 的统一前缀，在 `internal/router` 建一次分组，不要在子路由里重复写。 |
| `非版本化端点` | `GET /health`、`GET /ready`、`GET /swagger/*`，不带 `/api/v1` 前缀、不挂 jwt。 |
| `page` / `page_size` | 列表分页参数；`page` 默认 1，`page_size` 默认 20、上限 100。 |
| `job_id` / `status` | `GET /api/v1/tasks` 的过滤参数，字段需白名单校验。 |
| `DTO` | `internal/dto` 里的出入参结构，与 `internal/model` 分离，不暴露 Password 等内部字段。 |
| `统一响应体` | `{"code":0,"message":"ok","data":...,"request_id":"..."}`，由 `internal/response` 构造。 |

## 基础设施

| 术语 | 含义 |
| --- | --- |
| `AHA_` | Viper 环境变量前缀（`SetEnvPrefix("AHA")` + `AutomaticEnv()`）。配置项 `server.port` 对应 `AHA_SERVER_PORT`，分隔符 `.` 映射为 `_`。 |
| `AHA_DB_TYPE` | 数据库类型 `sqlite|mysql|postgresql`，默认 `sqlite`。 |
| `DSN` | 数据库连接串，由 `internal/database` 按环境变量拼装；三种写法见 `docs/configuration.md`。 |
| `AutoMigrate` | GORM 自动迁移，顺序 `Job → JobResult → Task → User → SysConfig`；只增不删。**不建库**：库需预先创建。 |
| `serializer:json` | GORM tag，把 map / slice / struct 存成 JSON 文本，sqlite / mysql / postgresql 通用。 |
| `预先建库` | 新约定：数据库层已移除 `CREATE DATABASE`，mysql / postgresql 启动前必须手工或用 `scripts/` 建好 `AHA_DB_NAME` 指定的库。 |
| `requestid / logger / recover / cors / limiter / jwt` | 全局中间件，在 `internal/app/app.go` 按序注册；jwt 与 RequireRole 在 `internal/router` 按路由挂。 |
| `swag / swagger` | swaggo 注解与 `swag init --output api --parseInternal --parseDependency` 生成的 `api/` 文档，UI 在 `/swagger/*`。 |
