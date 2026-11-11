# 背景：分层架构与请求链路

> 只读背景文档，硬规则以 `AGENTS.md` 为准。

## 1. 分层与包职责

| 层 | 包 / 文件 | 职责 |
| --- | --- | --- |
| 入口 | `main.go` | 加载配置、初始化数据库、调用 `internal/app` 装配、`app.Listen`；swag 通用注解也在此 |
| 装配 | `internal/app/app.go` | 创建 `fiber.App`，按序注册全局中间件、挂载 `/swagger/*`、`/health` / `/ready`，再把 `/api/v1` 路由挂上 |
| 配置 | `internal/config/` | 读 `configs/config.yml` + `AHA_` 环境变量，输出强类型配置 |
| 路由 | `internal/router/` | 建 `/api/v1` 分组，按路由挂 jwt / RequireRole，绑定各 controller |
| 中间件 | `internal/middleware/jwt/` | 解析 Bearer Token、HS256 校验，写 `user_id` / `role` 到 `c.Locals` |
| 中间件 | `internal/middleware/auth/` | `RequireRole("admin")` |
| 控制层 | `internal/controller/auth` / `job` / `task` / `health` | handler：绑定 DTO、校验、调库、回响应 |
| 契约 | `internal/dto/` | 出入参 DTO（与 model 分离） |
| 契约 | `internal/response/` | 统一响应体与业务错误码 |
| 数据 | `internal/database/` | 拼 DSN、`gorm.Open`、`AutoMigrate`、提供 `*gorm.DB` |
| 模型 | `internal/model/`（含 `sys_config.go`） | Job / JobResult / Task / User / SysConfig，Status / ErrorCode 枚举 |
| 生成物 | `api/` | swag 输出的 docs.go / swagger.json / swagger.yaml，禁止手改 |

## 2. 依赖方向（单向，禁止反向 import）

```
main.go
  → internal/config（配置结构体）
  → internal/database（构造 *gorm.DB）
  → internal/app/app.go（装配）
        → 全局中间件 requestid / logger / recover / cors / limiter
        → /swagger/*（fiber-swagger + api/ 生成物）
        → controller/health（/health、/ready）
        → internal/router（/api/v1 分组）
              → middleware/jwt → middleware/auth
              → controller/auth、controller/job、controller/task
                    → internal/dto、internal/response、internal/database、internal/model
```

- controller 不得反向 import `internal/router` / `internal/app`；`internal/model` 不得依赖 dto / controller。
- 配置、`*gorm.DB`、JWT 参数由 `main.go` 构造后**注入**，不通过包级可变变量共享（见 `.agents/rules/go-style.md` 第 5 节）。

## 3. 全局中间件顺序（`internal/app/app.go` 注册，顺序不可随意调换）

```
Client
  → requestid   生成 request_id 写入 c.Locals
  → logger      记录 method / path / status / 耗时 / request_id（脱敏）
  → recover     panic 兜底 → 500 + code 9999
  → cors        AHA_CORS_ORIGINS
  → limiter     AHA_RATE_LIMIT_MAX → 429 + code 6001
  → 路由分发
```

## 4. 完整请求链路

### 版本化 API（`/api/v1/...`）

```
请求 → 上述全局中间件
     → internal/router 的 /api/v1 分组
     → jwt 中间件（公开路由 /register、/login 跳过；其余全部经过）
          解析 Authorization: Bearer <token>，HS256 校验，写 c.Locals("user_id" / "role")
     → RequireRole("admin")（仅 DELETE /api/v1/jobs/:id）
     → controller/<domain> handler
          c.BodyParser(&dto.XxxRequest) + validate 校验
          db.WithContext(c.UserContext()) 查库 → model
     → internal/response 统一封装 {"code":0,"message":"ok","data":...,"request_id":"..."}
     → c.Status(标准 HTTP 码).JSON(...)
```

### 非版本化端点

- `GET /health`：只过全局中间件，直接返回存活状态，不查库、不挂 jwt。
- `GET /ready`：过全局中间件，检查 `*gorm.DB` 连通性，失败返回 `503`。
- `GET /swagger/*`：过全局中间件，由 fiber-swagger 提供 UI，读 `api/` 生成物，不挂 jwt。

## 5. 启动链路

```
main.go
 1. internal/config：读 configs/config.yml → SetDefault
    → SetEnvPrefix("AHA") → SetEnvKeyReplacer(".", "_") → AutomaticEnv()
    （configs/config.example.yml 是可提交示例，不含真实密钥）
 2. 校验 AHA_JWT_SECRET（生产必填）、AHA_DB_* 齐全
 3. internal/database：按 AHA_DB_TYPE（sqlite|mysql|postgresql，默认 sqlite）拼 DSN → gorm.Open
    ※ 新约定：数据库层已移除自动 CREATE DATABASE
       mysql / postgresql 必须预先建好 AHA_DB_NAME 指定的库
       sqlite 需保证 AHA_DB_NAME 路径的父目录存在
 4. AutoMigrate(Job, JobResult, Task, User, SysConfig) —— 顺序固定，Job 在 JobResult / Task 之前
 5. internal/app/app.go：全局中间件 → /swagger/* → /health /ready → /api/v1 路由
 6. app.Listen(":8080")，端口来自 AHA_SERVER_PORT（默认 8080）
```

## 6. 数据关系

```
User ──1:N──> Job ──1:N──> Task
                │
                └──1:1──> JobResult

SysConfig：key / value 形式的系统配置表（internal/model/sys_config.go）
```

- `Job` 是聚合根，`Task` 与 `JobResult` 通过 `JobID` 关联；这正是 `AutoMigrate` 必须 `Job` 在前的原因。
- 结构化字段（Config / Label / Metrics / Artifacts）用 `gorm:"serializer:json"` 存储，禁止 `type:jsonb` 硬编码。
- `Status` 为类型化枚举、`ErrorCode` 为业务错误枚举，具体取值以 `internal/model` 定义为准（见 `glossary.md`）。

## 7. 跨层约定

- 只有 `internal/response` 决定响应形状；controller 不自己拼 `fiber.Map`。
- 只有 `internal/database` 调 `AutoMigrate`；**建库是部署前置动作**，服务不再自动 `CREATE DATABASE`（见 `docs/configuration.md`）。
- 认证信息只经 `c.Locals` 传递，不落全局变量。
- 改完按 `.agents/workflows/` 收尾，提交前跑 `AGENTS.md` 第 11 节清单。
