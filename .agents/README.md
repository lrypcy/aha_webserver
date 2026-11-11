# .agents/ — AI agent 协作知识库

本目录是 `AGENTS.md` 的配套资料，供 AI coding agent 在改代码时按需查阅。`AGENTS.md` 是唯一权威规范，本目录是它的展开与可执行化，不与之冲突；如有冲突以 `AGENTS.md` 为准。

## 目录用途

| 目录 / 文件 | 用途 |
| --- | --- |
| `rules/` | 按主题拆开的硬规则：Go 风格、API 契约、安全、测试、部署 |
| `skills/` | 可照着直接执行的技能，每个技能一个目录 + `SKILL.md` |
| `workflows/` | 端到端流程：功能实现、修 bug、代码评审、部署，含检查清单与完成定义 |
| `context/` | 只读背景：分层架构与请求链路、术语表、部署拓扑 |
| `config.json` | 机器可读索引：技术栈、命令、部署契约、端口、环境变量键名、约束 |

## 索引

### 规则 rules/
| 文件 | 看什么 |
| --- | --- |
| `.agents/rules/go-style.md` | 命名、错误处理、context、禁止全局可变状态、格式化 |
| `.agents/rules/api-contract.md` | 统一响应体、错误码表、分页、端点清单、swag 注解 |
| `.agents/rules/security.md` | 密钥、密码、SQL 注入、DTO 分离、JWT 与角色 |
| `.agents/rules/testing.md` | 表驱动测试、`app.Test`、命令与 CI |
| `.agents/rules/deployment.md` | easy_deploy 六目标、deploy.yaml 契约、端口/探针/密钥同步规则 |

### 技能 skills/
| 技能 | 什么时候用 |
| --- | --- |
| `.agents/skills/add-endpoint/SKILL.md` | 新增 `/api/v1` 接口 |
| `.agents/skills/add-model-migration/SKILL.md` | 新增 / 修改 GORM 模型与迁移 |
| `.agents/skills/run-and-verify/SKILL.md` | 启动服务并跑冒烟验证 |
| `.agents/skills/swagger-update/SKILL.md` | 改了注解 / DTO 后重新生成 `api/` |
| `.agents/skills/debug-build-failure/SKILL.md` | `go build` / `go vet` 失败时排查 |
| `.agents/skills/deploy/SKILL.md` | 用 easy_deploy 校验 / 预演 / 部署 / 排障 |

### 工作流 workflows/
| 文件 | 用途 |
| --- | --- |
| `.agents/workflows/feature-implementation.md` | 新功能从契约到提交的流程 |
| `.agents/workflows/bugfix.md` | 复现 → 定位 → 最小修复 → 回归 |
| `.agents/workflows/code-review.md` | 评审维度 + 检查清单 + 完成定义 |
| `.agents/workflows/deployment.md` | 部署：校验 → 预检 → 预演 → 执行 → 验证 → 汇报 |

### 背景 context/
| 文件 | 用途 |
| --- | --- |
| `.agents/context/architecture.md` | 分层、依赖方向、完整请求链路、数据关系 |
| `.agents/context/glossary.md` | Job / Task / JobResult / Status / ErrorCode / 角色等术语 |
| `.agents/context/deployment.md` | 六目标部署拓扑、资产映射、状态与回滚 |

### 其他入口
- `AGENTS.md`（主规范）、`CLAUDE.md`、`.github/copilot-instructions.md`、`.cursor/rules/aha-webserver.mdc`（均为薄封装）
  - macOS 默认大小写不敏感，`agents.md` 与 `AGENTS.md` 是同一路径，故不建立小写别名文件。
- `docs/api-guide.md`、`docs/configuration.md`、`docs/deployment.md`（面向人的文档）
- `deploy.yaml`（唯一部署事实源）、`deploy/k8s/`（render 快照）

## 使用约定

1. 先读 `AGENTS.md`，再按任务类型读对应 `rules/` 与 `context/`。
2. 具体动手时打开对应 `skills/` 照步骤执行，不要凭记忆跳步。
3. 跨多个步骤的任务用 `workflows/` 里的检查清单与完成定义收尾。
4. 本目录内的命令、路径、端口、环境变量名必须与 `AGENTS.md` / `config.json` 完全一致；发现不一致以 `AGENTS.md` 为准并修正文档。
