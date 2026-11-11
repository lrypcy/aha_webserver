# 工作流：功能实现

适用：新增接口、改模型、加中间件等任何功能改动。权威规范见 `AGENTS.md`，细则见 `.agents/rules/`，背景见 `.agents/context/architecture.md`。

## 流程

1. **读规范**：先读 `AGENTS.md`，再读对应 `.agents/rules/` 与 `.agents/context/architecture.md`。不要跳过。
2. **写清契约**：把方法、完整路径（带 `/api/v1`）、鉴权、请求/响应字段、分页、错误码写成清单；涉及模型变更先确认 `AutoMigrate` 顺序与是否破坏性。
3. **选技能**：新增接口用 `.agents/skills/add-endpoint/SKILL.md`；改模型用 `.agents/skills/add-model-migration/SKILL.md`；两者都做就按顺序都跑。
4. **实现**：按 DTO → controller → router → swag 注解的顺序落地；装配相关的改动（中间件、swagger、health、路由）集中在 `internal/app/app.go` 与 `internal/router`，不要散到 `main.go`。
5. **数据库前置**：若用 mysql / postgresql，启动前**预先建库**（数据库层已移除 `CREATE DATABASE`）；sqlite 保证 `AHA_DB_NAME` 父目录存在。见 `.agents/skills/run-and-verify/SKILL.md`。
6. **生成文档**：改过注解 / DTO / model 就跑 `swag init --output api --parseInternal --parseDependency`；同步 `docs/api-guide.md` 与 `docs/configuration.md`。
7. **测试**：表驱动 `_test.go`，覆盖正常 / 参数错误 / 鉴权失败 / 角色越权；`go test ./...`。
8. **验证**：按 `.agents/skills/run-and-verify/SKILL.md` 起服务跑通注册 → 登录 → `Bearer` 创建 Job / Task 链路。
9. **检查清单与提交**：跑 `AGENTS.md` 第 11 节四条命令，全绿后提交。

## 检查清单

- [ ] 契约已明确，路径带 `/api/v1` 且无重复前缀
- [ ] 出入参走 `internal/dto`，响应走 `internal/response`，未直接暴露 model
- [ ] 错误码只在 `internal/response` 扩展，HTTP 状态码为标准值
- [ ] 无 `binding` 标签、无 `type:jsonb`、无字符串拼接 SQL、无密码 / Token 日志
- [ ] 无包级可变状态，依赖经 `main.go` → `internal/app/app.go` 注入
- [ ] `AutoMigrate` 顺序为 `Job → JobResult → Task → User → SysConfig`；mysql / postgresql 已预先建库
- [ ] `api/` 三个生成物已重新生成，`docs/` 已同步
- [ ] `gofmt -w .` / `go vet ./...` / `go test ./...` / `go build ./...` 全绿

## 完成定义（DoD）

功能按契约可用；冒烟链路通过；测试覆盖关键分支；四条检查命令全绿；`api/` 与 `docs/` 同步；PR 里写明数据库前置动作与（如有）显式迁移脚本的执行顺序。
