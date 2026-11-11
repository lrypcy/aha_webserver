# 工作流：代码评审

适用：评审任何 PR / 改动。按维度逐项过，不靠肉眼扫一遍就通过。权威规范 `AGENTS.md`。

## 审查维度

| 维度 | 看什么 | 依据 |
| --- | --- | --- |
| 契约 | 路径带 `/api/v1` 且无重复前缀；统一响应体含 `code`/`message`/`data`/`request_id`；HTTP 状态码标准 | `.agents/rules/api-contract.md` |
| 分层 | `main.go` → `internal/app/app.go` 装配；controller 不反向 import router / app；model 不依赖 dto | `.agents/context/architecture.md` |
| DTO | 出入参全走 `internal/dto`；无 `model.User` 直接 `c.JSON`；无 `c.BodyParser(&model.Xxx)` | `.agents/rules/security.md` 第 4 条 |
| 错误处理 | `%w` 后是 error 值而非 `err.Error()`；`errors.Is` / `errors.As`；不吞错误；错误统一由 `internal/response` 转换 | `.agents/rules/go-style.md` 第 3 节 |
| 全局状态 | 无 `var db *gorm.DB` / `var secretKey string`；`init()` 无 IO 与 `panic` | `.agents/rules/go-style.md` 第 5 节 |
| 安全 | 无密码 / Token / 密钥日志；密钥只走 `AHA_JWT_SECRET` / `AHA_DB_PASSWORD`；无拼接 SQL；bcrypt；HS256 | `.agents/rules/security.md` |
| 数据库 | `AutoMigrate` 顺序 `Job → JobResult → Task → User → SysConfig`；`serializer:json`；无 `type:jsonb`；**未加回 `CREATE DATABASE`**（库需预先创建） | `AGENTS.md` 第 7 节 |
| 中间件 | requestid / logger / recover / cors / limiter 在 `internal/app/app.go` 按序注册；jwt 与 RequireRole 按路由挂；`/health` / `/ready` / `/swagger/*` 不挂 jwt | `.agents/context/architecture.md` |
| 文档 | `api/` 三个生成物随注解变更更新；`docs/api-guide.md` / `docs/configuration.md` 同步 | `.agents/skills/swagger-update/SKILL.md` |
| 测试 | 新增接口有正常 / 参数错误 / 鉴权失败用例；admin 接口有 `403` 用例；`go test ./...` 通过 | `.agents/rules/testing.md` |

## 检查清单

- [ ] `gofmt -l .` 无输出
- [ ] `go vet ./...` 通过（尤其 `%w` 类型报错）
- [ ] `go test ./...` 通过，关键分支有测试
- [ ] `go build ./...` 通过
- [ ] 无 `binding:` 标签、无 `type:jsonb`、无 `::jsonb`、无字符串拼接 SQL
- [ ] 无 `fmt.Println` 密码 / 哈希 / Token / 密钥
- [ ] 无新增包级可变状态；依赖方向单向
- [ ] `api/` 生成物是 `swag init` 产出，无手工编辑
- [ ] mysql / postgresql 部署说明里写明**预先建库**，sqlite 写明父目录前置
- [ ] `docs/` 与 `.agents/` 相关文档已同步

## 常见否决项（出现即打回）

1. 直接 `c.JSON(model.User)` 或 `c.BodyParser(&model.Job)`。
2. `fmt.Errorf("%w ...", err.Error())` 或 `go vet` 报错被 `//nolint` 压掉。
3. `gorm:"type:jsonb"` / `::jsonb` 硬编码。
4. `app.Group("/api/v1")` 下再写 `/api/v1` 前缀。
5. 把 `CREATE DATABASE` 加回 `internal/database` 来「顺手」解决连接失败。
6. `main.go` 里散落中间件 / swagger / health 注册，未集中在 `internal/app/app.go`。
7. 改了注解但 `api/` 没重新生成。

## 完成定义（DoD）

上表 10 个维度均无阻塞问题；四条检查命令全绿；无常见否决项；文档与 `api/` 生成物同步；PR 描述包含数据库前置动作与（如有）迁移脚本执行顺序。
