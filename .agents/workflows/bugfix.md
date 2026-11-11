# 工作流：修复 Bug

适用：接口报错、行为不符契约、构建失败、启动失败。构建类问题直接用 `.agents/skills/debug-build-failure/SKILL.md`。

## 流程

1. **复现**：写下一个能稳定复现的最小输入（curl / 测试用例 / `go build ./...` 输出），保留原始报错全文。
2. **定位**：按 `.agents/context/architecture.md` 的请求链路逐层缩小范围：全局中间件 → `/api/v1` 分组 → jwt / RequireRole → controller → DTO 校验 → `internal/database` → model；用 `go build ./internal/...`、`go test ./internal/xxx -run TestXxx` 定位包与用例。
3. **判根因，不判表象**：常见根因见 `AGENTS.md` 第 12 节（Fiber `binding` 标签、`%w` 误用、分组重复前缀、`type:jsonb`、swag 未重新生成、全局可变状态、打印密码、动态列名拼接、Viper `AutomaticEnv` 坑）以及**数据库层已移除自动 `CREATE DATABASE`** 导致的 `Unknown database`。
4. **先加回归测试**：在改动前写一个能暴露该 bug 的 `_test.go` 用例，确认它当前失败。
5. **最小修复**：只改根因，不顺手重构、不改无关契约、不绕过（删测试、`//nolint`、吞 error、把 `CREATE DATABASE` 加回 `internal/database` 都禁止）。
6. **确认回归**：新测试通过，`go test ./...` 全绿；按原复现步骤再走一遍确认现象消失。
7. **补文档**：若属于新坑，补进 `AGENTS.md` 第 12 节与对应 `.agents/rules/`；若契约有调整，同步 `docs/api-guide.md`。
8. **检查清单与提交**。

## 检查清单

- [ ] 有稳定复现步骤与原始报错记录
- [ ] 有先失败后通过的回归测试
- [ ] 修复是最小改动，未引入新全局状态 / 新 `type:jsonb` / 新拼接 SQL
- [ ] 未通过删测试、`//nolint`、改 `go.mod`、吞 error 绕过
- [ ] `go vet ./...` / `go test ./...` / `go build ./...` 全绿，必要时 `go test -race ./...`
- [ ] 相关文档已同步（`AGENTS.md` 常见坑 / `.agents/rules/` / `docs/`）

## 完成定义（DoD）

原 bug 不再复现；回归测试纳入 `go test ./...`；无回归；四条检查命令全绿；根因与修复方式写进 PR 描述。
