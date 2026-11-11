---
name: debug-build-failure
description: "go build / go vet / go test 失败时按症状定位根因并修复的可执行排查表，含预先建库与生成物缺失等新约定。"
---

# 技能：排查构建与静态检查失败

适用：`go build ./...`、`go vet ./...`、`go test ./...` 报错，或运行时数据库连接失败。开始前读 `.agents/context/architecture.md` 与 `.agents/rules/go-style.md`。

## 步骤

1. **完整复现并记录报错**：`go build ./...`，保留原始输出，不要只看最后一行。
2. **按症状查表**：

| 症状 | 根因 | 修复 |
| --- | --- | --- |
| `cannot find package ".../api"` / `no Go files in api` | `api/` 被 `make swag_clean` 清空或未生成 | `swag init --output api --parseInternal --parseDependency` |
| `undefined: model.Xxx` / `undefined: dto.Xxx` | 模型 / DTO 未建、改名或漏提交 | 检查 `internal/model`（含 `sys_config.go`）与 `internal/dto`，补齐类型后重新生成 swag |
| `import cycle not allowed` / `undefined: app.New` | 依赖方向反了（controller 反向 import router / app） | 按 `.agents/context/architecture.md` 的依赖方向调整；装配统一放 `internal/app/app.go` |
| `fmt.Errorf format %w has arg ... of wrong type string`（go vet） | 把 `err.Error()` 传给了 `%w` | 改成 `fmt.Errorf("上下文: %w", err)`；判断用 `errors.Is` / `errors.As` |
| `Unknown database` / `database does not exist`（运行时） | 数据库层已移除自动 `CREATE DATABASE` | 启动前预先建好 `AHA_DB_NAME` 指定的库，见 `.agents/skills/run-and-verify/SKILL.md` 第 2 步 |
| `no such file or directory`（sqlite 运行时） | `AHA_DB_NAME` 路径父目录不存在 | `mkdir -p` 该目录 |
| 建表报错含 `jsonb` / `::jsonb`（sqlite / mysql 运行时） | 模型硬编码了 PostgreSQL 专属类型 | 改成 `gorm:"serializer:json"`，去掉 `type:jsonb` 与 `::jsonb` 默认值 |
| 必填字段不生效、`binding:"required"` 没反应 | Fiber v2 不认 `binding` 标签（那是 Gin 的） | 换成 `validate` 标签 + `go-playground/validator` 或手写校验 |
| `missing go.sum entry` / 版本冲突 | 依赖未同步 | `go mod tidy` 然后 `go mod download` |
| 环境变量读不到、`viper.Get*` 返回零值 | 缺 `SetEnvPrefix("AHA")` / `SetEnvKeyReplacer(".", "_")` / `AutomaticEnv()`；或 `Unmarshal` 前没 `SetDefault` | 在 `internal/config` 补齐；注意 `AutomaticEnv` 对 `Unmarshal` 无效 |
| 路由 404、路径出现 `/api/v1/api/v1` | 分组后又写了 `/api/v1` 前缀 | 分组下只写 `/jobs`、`/tasks` 等相对路径 |

3. **缩小范围**：`go build ./internal/...` 定位到具体包；`go test ./internal/controller/job -run TestXxx` 定位到具体用例；`go vet ./internal/...` 定位静态问题。
4. **最小修复**：只改根因，不顺手重构；改完回到第 1 步复现确认消失。
5. **跑完整检查清单**：`gofmt -w .` → `go vet ./...` → `go test ./...` → `go build ./...`，必要时 `go test -race ./...`。
6. **同步文档**：若根因属于 `AGENTS.md` 第 12 节常见坑，确认文档已覆盖；新坑补进 `AGENTS.md` 与 `.agents/rules/`。

## 禁止的绕过方式

- 删测试、`-short` 跳过、加 `//nolint` 压掉 vet 报错
- 改 `go.mod` 版本或注释掉 import 让编译通过
- 把 error 吞掉（`_ = ...`）或把 `panic` 换成静默返回
- 在 `internal/database` 里加回 `CREATE DATABASE` 来绕过预先建库约定

## 完成定义（DoD）

- [ ] 原始报错已消失，且是修根因而非绕过
- [ ] 四条检查命令全绿，`go test -race ./...`（涉及并发时）通过
- [ ] 新发现的坑已补进 `AGENTS.md` 常见坑或对应规则
