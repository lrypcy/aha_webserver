# 规则：安全红线

> 细则，权威规范见 `AGENTS.md` 第 8 节。以下 8 条均为硬红线，评审时逐条核对。

## 1. 禁止打印敏感信息

- 禁止 `fmt.Println(u.Password)`、`fmt.Println(password)`、打印 Token / 密码哈希 / `AHA_JWT_SECRET`。
- logger 中间件打请求体必须先脱敏 `password`、`token`、`secret` 字段；错误响应不含堆栈、不含 SQL 原文。
- 反例（旧代码，已删）：`VerifyPassword` 里两行 `fmt.Println` 明文密码与哈希。

## 2. 密钥只走环境变量

- `AHA_JWT_SECRET`（生产必填，空则拒绝启动）、`AHA_DB_PASSWORD` 只能来自环境变量。
- 禁止写进 `configs/config.yml`、`configs/config.example.yml`、代码常量、注释、Swagger 示例、`init/casbin.sql`。
- 生产环境 `AHA_SERVER_ENV=prod` 缺少 `AHA_JWT_SECRET` 必须启动失败，禁止回退到默认密钥。
- 旧代码从 `sys_config` 表读 `jwt_secret` 的做法已废弃，不要恢复。

## 3. 禁止字符串拼接 SQL

```go
db.Where("id = ?", id)                       // 对：参数绑定
db.Where(field+" = ?", value)                // 危险：字段名来自请求时需白名单
db.Raw("SELECT * FROM users WHERE name='"+n+"'")  // 禁止
```

- 动态排序 / 过滤字段（如 `status`、`job_id`）先过白名单常量再拼列名。
- `AutoMigrate` 之外不手写 `CREATE DATABASE` / `CREATE TABLE`；数据库层已移除自动建库，见第 6 节。

## 4. DTO 必须与 model 分离

- `internal/dto` 定义出入参，`internal/model` 定义表结构。
- 禁止 `var u model.User; c.BodyParser(&u)`（会把 `Password`、`IsActive` 等内部字段暴露给客户端）。
- 禁止 `c.JSON(user)` 直接返回 model（含 `Password` 字段）；用 `dto.UserResponse`。
- 注册 / 登录响应里不得出现密码哈希；`User.Password` 的 `json:"-"` 不是唯一防线。

## 5. 密码与认证

- 密码 bcrypt 哈希存储；登录用 `bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))`。
- 禁止明文比较、禁止自己实现哈希、禁止 `BeforeSave` 重复加密（只在 `BeforeCreate` 或显式 `UpdatePassword` 加密一次）。
- JWT：HS256，`Authorization: Bearer <token>`，claims 含 `user_id` / `role`；校验签名算法，拒绝 `none` 与算法混淆。
- 角色只有 `user|admin`；`DELETE /api/v1/jobs/:id` 必须 `RequireRole("admin")`。

## 6. 数据库前置条件（新约定）

- 重构后的 `internal/database` **不再执行 `CREATE DATABASE`**：mysql / postgresql 必须先手工或用 `scripts/` 建好 `AHA_DB_NAME` 指定的库，再启动服务。
- sqlite 需保证 `AHA_DB_NAME` 路径的父目录存在（服务不会自动创建目录）。
- DSN 只含连接信息，不含账号密码硬编码；连接失败即启动失败，不要自动降级到 sqlite。

## 7. 限流与 CORS

- limiter（`AHA_RATE_LIMIT_MAX`）与 cors（`AHA_CORS_ORIGINS`）在 `internal/app/app.go` 全局注册，保持开启。
- 生产 `AHA_CORS_ORIGINS` 不要配 `*`；不要为调试全局关闭中间件。
- recover 中间件兜底 panic → `500` + code `9999`，不把 panic 内容回传客户端。

## 8. 依赖与生成物

- 不手改 `api/` 生成物；改注解后走 `swag init --output api --parseInternal --parseDependency`。
- 不引入来路不明的依赖；加依赖后 `go mod tidy`，并检查 `go.sum` 变化。

## 自检清单

- [ ] 无密码 / Token / 密钥日志，日志已脱敏
- [ ] 密钥只来自 `AHA_JWT_SECRET` / `AHA_DB_PASSWORD`
- [ ] 无字符串拼接 SQL，动态字段已白名单
- [ ] 出入参全走 `internal/dto`，无 model 直接进出
- [ ] mysql / postgresql 已预先建库（不再自动 `CREATE DATABASE`）
- [ ] 角色边界、限流、CORS、recover 均已生效
