# 规则：Go 编码风格

> 细则，权威规范见 `AGENTS.md` 第 5 节。

## 1. 命名

| 对象 | 规则 | 正例 | 反例 |
| --- | --- | --- | --- |
| 包名 | 小写单词，无下划线、无驼峰 | `package dto` | `package my_dto` |
| 导出标识符 | 驼峰；缩写保持常见写法 | `UserID`、`HTTPPort`、`DSN`、`JWTSecret` | `UserId`、`Dsn` |
| 文件名 | 小写 + 下划线 | `job_model.go`、`auth_controller.go` | `JobModel.go` |
| 测试文件 | 同包 `_test.go` | `job_test.go` | — |
| 枚举 | 类型化常量，不散落裸 int | `type Status int` + `const StatusPending Status = iota` | `const Pending = 0` |
| 接收器 | 单字母，同类型保持一致 | `func (j *Job) ...` | 同一类型一会儿 `j` 一会儿 `job` |

## 2. 包与依赖方向

```
main.go → internal/app → internal/router → internal/middleware/* → internal/controller/*
                                                            ↓
                                    internal/dto / internal/response / internal/database → internal/model
```

- `internal/app/app.go` 负责装配（fiber 中间件 + swagger + health + 路由），不写业务逻辑。
- controller 不得反向 import `internal/router` / `internal/app`；`internal/model` 不得依赖 dto / controller。
- `internal/config` 只负责加载配置，通过构造参数把配置传给 database / jwt / app，不要在各处直接读 `viper.Get*`（除 config 包内部）。

## 3. 错误处理

- error 一律返回并向上传递，禁止 `fmt.Println(err)` 后继续执行。

```go
// 错：%w 收到 string，渲染成 %!w(string=...)，错误链断裂
return fmt.Errorf("%w, failed to update password", err.Error())

// 对：%w 后面必须是 error 值
return fmt.Errorf("创建用户失败: %w", err)
```

- 判断用 `errors.Is(err, gorm.ErrRecordNotFound)` / `errors.As`，禁止 `==` 或 `strings.Contains(err.Error(), ...)`。
- 业务错误统一交给 `internal/response` 转响应体；controller 禁止自己拼 `fiber.Map{"error": ...}`。
- 禁止吞错误：`_ = db.AutoMigrate(...)` 不允许；`db.AutoMigrate(...)` 裸调用同样不允许。
- 只加一层上下文，不要层层重复 `failed to ...`。
- `go vet ./...` 会报 `fmt.Errorf format %w has arg ... of wrong type`，看到必须修，不能 `//nolint` 绕过。

## 4. context

- handler 签名固定 `func(c *fiber.Ctx) error`。
- 需要超时/取消时 `ctx := c.UserContext()`，并 `db.WithContext(ctx).Where(...)`。
- 不要在结构体里缓存 `context.Context`，不要跨请求复用 ctx。
- 启动与优雅关闭用 `context.Context` 传递取消信号；`init()` 禁止阻塞 IO。

## 5. 禁止全局可变状态

```go
// 反例（旧代码里的写法，重构必须去掉）
var db *gorm.DB
var secretKey string
var expiration time.Duration

func Init(secret string, exp time.Duration) {
    secretKey = secret
    expiration = exp
}
```

- 改为构造注入：`main.go` 读 `internal/config` → 构造 `*gorm.DB` 与 JWT 参数 → 传给 `internal/app` → 传给 router / controller / middleware。
- 请求级数据只放 `c.Locals(key, value)`，不要包级 map。
- 只读 `const` 可以包级；可变状态不行。
- `init()` 里禁止连库、读文件、`panic`（`panic("fail to connect database")` 这类写法要换成返回 error 由 main 处理）。

## 6. 格式化与静态检查

```bash
gofmt -w .
go vet ./...
go test ./...
go build ./...
```

- import 分组：标准库 / 第三方 / 本项目 `github.com/lrypcy/aha_webserver/...`，三组之间空行。
- 不要手动对齐、不要留尾随空格；`gofmt -l .` 必须无输出。
- 导出标识符写注释（`// Xxx ...`）；包注释可选但推荐。

## 7. 注释与 swag 注解

- 每个对外 handler 必须带 `@Summary` `@Tags` `@Accept json` `@Produce json` `@Param` `@Success` `@Failure` `@Router /api/v1/xxx [method]`。
- `@Router` 要写完整前缀 `/api/v1`；需要认证的加 `@Security BearerAuth`。
- 注解里引用的是 `internal/dto` 类型，不要直接引用 `internal/model`。
- 注解改了必须重跑 `swag init --output api --parseInternal --parseDependency`。

## 8. 自检清单

- [ ] 命名符合上表，枚举是类型化常量
- [ ] 依赖方向单向，无反向 import
- [ ] 所有 error 都返回 / 包装正确，`%w` 后是 error 值
- [ ] 无包级可变状态，`init()` 无 IO
- [ ] `gofmt -l .` 无输出，`go vet ./...` 通过
