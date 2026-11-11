---
name: add-model-migration
description: "新增或修改 GORM 模型与 AutoMigrate 的可执行步骤：模型 tag、DTO 同步、迁移顺序、预先建库、显式迁移脚本与检查清单。"
---

# 技能：新增 / 修改模型与迁移

适用：改动 `internal/model` 下任意结构体（含 `internal/model/sys_config.go` 的 SysConfig）。开始前读 `AGENTS.md` 第 7、10 节。

## 步骤

1. **判断变更类型**：加列 / 加表 → 交给 `AutoMigrate`；删列 / 改列类型 / 改表名 / 改外键 → **破坏性变更**，必须走第 6 步显式迁移。
2. **改模型**：在 `internal/model/` 对应文件改结构体。结构化字段（map / slice / struct）用 GORM `serializer:json`：
   ```go
   type Job struct {
       gorm.Model
       Title  string            ``gorm:"size:255;not null"``
       Config map[string]any    ``gorm:"serializer:json"``
       Label  []string          ``gorm:"serializer:json"``
   }
   ```
   **禁止 `gorm:"type:jsonb"`、`default:'[]'::jsonb`、`SERIAL` 等数据库专属写法**；用 `primaryKey` / `autoIncrement` / `index` / `uniqueIndex` 抽象。新增表要放到 `internal/model` 对应文件（如 SysConfig 放 `sys_config.go`）。
3. **同步 DTO**：在 `internal/dto/` 只暴露应对外可见的字段。新增列不要自动出现在响应里；`User.Password` 永远不进 DTO。
4. **确认 AutoMigrate**：在 `internal/database/` 检查列表包含该模型，顺序固定 **`Job → JobResult → Task → User → SysConfig`**（`Job` 必须在 `JobResult` / `Task` 之前，因为外键依赖）。只在 `internal/database` 调一次 `AutoMigrate`。
5. **预先建库（新约定，必做）**：数据库层**已移除自动 `CREATE DATABASE`**，连接前库必须已存在：
   - `AHA_DB_TYPE=mysql` / `postgresql`：启动前手工或用 `scripts/` 建好 `AHA_DB_NAME` 指定的库（例如 `CREATE DATABASE aha_webserver;`），否则 `gorm.Open` 直接失败；
   - `AHA_DB_TYPE=sqlite`（默认）：保证 `AHA_DB_NAME` 路径的父目录存在（`mkdir -p`），服务不会自动建目录；
   - 集成测试同理，不要把 `CREATE DATABASE` 写回 `internal/database`。
   三种 DSN 形态见 `docs/configuration.md`。
6. **破坏性变更写显式迁移**：在 `scripts/` 加迁移脚本，PR 里写明执行顺序、数据回填方式与回滚方式；不要指望 `AutoMigrate` 删列或改类型。
7. **响应结构变了就重生成文档**：改 `@Success` / `@Param` 注解后跑 `swag init --output api --parseInternal --parseDependency`。
8. **写测试**：模型字段往返读写一致（尤其 `serializer:json`）；三种 `AHA_DB_TYPE` 都能 `AutoMigrate` 成功；sqlite 测试文件放 `t.TempDir()`。
9. **跑检查清单**：`gofmt -w .` → `go vet ./...` → `go test ./...` → `go build ./...`。

## 完成定义（DoD）

- [ ] 无 `type:jsonb` / 数据库专属类型，结构化字段用 `serializer:json`
- [ ] `AutoMigrate` 列表与顺序正确，`Job` 在 `JobResult` / `Task` 之前
- [ ] mysql / postgresql 已预先建库，sqlite 父目录已存在；数据库层无 `CREATE DATABASE`
- [ ] DTO 未泄露内部字段；破坏性变更有 `scripts/` 迁移脚本与 PR 说明
- [ ] `api/` 生成物（如响应结构变化）已重新生成
- [ ] 四种检查命令全绿
