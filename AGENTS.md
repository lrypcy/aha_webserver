# AGENTS.md — aha_webserver 权威规则入口

本文件是 `aha_webserver`（Go / Fiber 的 Job→Task 任务管理 HTTP 服务）面向 AI coding agent 的**唯一权威规范**。`.agents/` 是它的展开与可执行化，冲突以本文件为准。

> 本轮修正：此前 `AGENTS.md` 被误放成了「小写别名入口」的内容（自引用、无规则正文），现已恢复为主规范。
> macOS 默认大小写不敏感，`agents.md` 与 `AGENTS.md` 无法共存，因此本仓库**不提供**小写别名文件；只识别小写名的工具请直接读 `AGENTS.md`。

## 1. 必读顺序

1. 本文件（定位、技术栈、命令、硬约束、部署契约、检查清单）
2. [.agents/README.md](.agents/README.md)（知识库索引）与 [.agents/config.json](.agents/config.json)（机器可读）
3. `.agents/rules/`：[go-style.md](.agents/rules/go-style.md)、[api-contract.md](.agents/rules/api-contract.md)、[security.md](.agents/rules/security.md)、[testing.md](.agents/rules/testing.md)、[deployment.md](.agents/rules/deployment.md)
4. `.agents/context/`：[architecture.md](.agents/context/architecture.md)、[glossary.md](.agents/context/glossary.md)、[deployment.md](.agents/context/deployment.md)
5. 动手时打开对应 `.agents/skills/*/SKILL.md`，收尾走 `.agents/workflows/`

## 2. 技术栈与版本

| 类别 | 技术 |
| --- | --- |
| 语言 | Go 1.23.5，module `github.com/lrypcy/aha_webserver` |
| Web | Fiber v2 |
| ORM / DB | GORM，sqlite（CGO）/ mysql / postgresql，可切换 |
| 配置 | Viper，`AHA_*` 环境变量 > `configs/config.yml` > 内置默认值 |
| 鉴权 | JWT HS256，角色 `user` / `admin` |
| 文档 | swaggo 生成 `api/docs.go`、`api/swagger.json`、`api/swagger.yaml` |
| 部署 | easy_deploy（`deploy.yaml`，`apiVersion: easydeploy/v1`） |

## 3. 命令

| 用途 | 命令 |
| --- | --- |
| 格式化 / 静态检查 / 测试 / 构建 | `make fmt` / `make lint` / `make test` / `make build` |
| 本地运行（sqlite 零配置） | `make run`（`go run .`，端口 8080，工作目录必须是仓库根） |
| 接口文档 | `make swag`（`api/docs.go` 被 main.go 导入，不可删） |
| 原生 compose | `make docker-up` / `make docker-down` |
| **统一部署** | `make deploy-check` / `make deploy-local` / `make deploy-down`（见第 6 节） |

## 4. 目录地图

```
aha_webserver/
├── AGENTS.md              # 本文件（权威）
├── CLAUDE.md / .github/copilot-instructions.md / .cursor/rules/*.mdc  # 薄封装
├── deploy.yaml            # easy_deploy 唯一部署事实源
├── deploy/k8s/            # render --target k8s 的快照（不含 Secret）
├── docs/deployment.md     # 面向人的部署手册
├── Makefile / README.md / .env.example / .gitignore
├── configs/               # config.yml（非敏感默认值）+ config.example.yml
├── build/package/Dockerfile
├── docker-compose.yml     # 人工维护的 app + PostgreSQL 编排
├── api/                   # swag 生成物（docs.go 被导入，勿删）
├── main.go
├── internal/
│   ├── app/ config/ router/ controller/ middleware/ dto/ model/ response/ database/
│   └── controller/health/   # /health、/ready
└── .agents/               # 规则、技能、工作流、背景知识、机器可读配置
```

## 5. 硬性约束（与 `.agents/rules/` 一致）

1. **API 契约**：统一响应体 `{"code":0,"message":"ok","data":...,"request_id":"..."}`；业务接口一律 `/api/v1` 前缀；不得生成 `fiber.Map{"error": ...}` 这类临时响应。
2. **分层与依赖方向**：`controller → service/dto → model → database`；DTO 与 GORM Model 分离；不在 controller 里直接拼 SQL 或返回模型。
3. **配置优先级**：`AHA_*` 环境变量 > `configs/config.yml` > 默认值；`config.yml` 只放非敏感默认值。
4. **安全**：密钥/口令不落盘到仓库；生产强制 `AHA_JWT_SECRET` ≥ 32 字符、显式 CORS 白名单、DB 口令；密码哈希、参数化查询、JWT 角色校验不可跳过。
5. **探针**：`/health`、`/ready` 挂在根路径（不受 `/api/v1` 与鉴权、限流影响），是 Docker / compose / k8s 探针的唯一来源。
6. **生成物**：`api/` 三件套是 swag 生成物，`docs.go` 被导入；更新用 `make swag`，不要手改或删除。
7. **最小改动**：不改无关代码、不擅自升级依赖、不顺手重构；改前先读现状，越界文件不动。
8. **不打印密钥**：文档、日志、输出只列环境变量键名，绝不回显值 / token。
9. **验证后汇报证据**：给出命令、退出码、变更清单与文件行号；不声称未验证的结论。

## 6. 部署契约（本轮新增，与 easy_deploy 对齐）

| 项 | 值 |
| --- | --- |
| 清单 | [deploy.yaml](deploy.yaml)，`apiVersion: easydeploy/v1`，`kind: Service` |
| 六个目标 | `local` / `systemd` / `ssh` / `docker` / `compose` / `k8s` |
| 端口 | `runtime.port=8080`、`hostPort=8080` |
| 健康 | `runtime.health.path=/health`，`timeout=60s` |
| 裸机构建 / 启动 | `go build -o bin/aha_webserver .` / `./bin/aha_webserver` |
| 容器 | `image=aha_webserver:latest`，Dockerfile `build/package/Dockerfile` |
| 状态目录 | `.easy_deploy/`（已忽略），`down` 只回收其中登记的资源 |

标准 agent 流程（全程 `--json`、无 TTY）：

```bash
edp validate --json
edp doctor --target <t> --json
edp plan --target <t> --json          # 隐含 dry-run，可审查 Step 列表
edp up --target <t> --yes --json
edp status --json / edp logs --tail 100 --json
edp down --target <t> --yes --json
```

退出码：`2` SPEC_INVALID / `3` PRECONDITION_MISSING / `4` STEP_FAILED / `5` HEALTH_FAILED / `6` STATE_CONFLICT / `7` CONFIRMATION_REQUIRED。完整协议见 `../easy_deploy/docs/agent-protocol.md`。

**不可违反**：
- 端口或健康路径改动必须同步 `deploy.yaml`、`Dockerfile` 的 `EXPOSE`/`HEALTHCHECK`、`docker-compose.yml`、`docs/deployment.md`、`deploy/k8s/` 快照。
- 新增环境变量必须同步 `.env.example`（只列键名）、`configs/config.example.yml` 与 `deploy.yaml` 的 `runtime.env`。
- 生产密钥只能在进程环境注入，清单里只能写 `${VAR}`；`render` 出的 Secret 文件不得入库。
- 环境不具备（如 `systemd` 无 systemctl、`ssh` 远端不通、旧集群证书过期）时不允许硬上：用 `doctor` 判定，改目标或只 `render`，并在交付说明里明确标注「仅生成清单，未实际部署」。注意 **`doctor` 通过 ≠ 可实际部署**：`docker` 的 VM 只读会让 `docker build` 在提交层失败；`k8s` 的集群可达不代表镜像在节点内，`k8s` Driver 不构建镜像，需先 `minikube image load`。

细则见 [.agents/rules/deployment.md](.agents/rules/deployment.md)，背景见 [.agents/context/deployment.md](.agents/context/deployment.md)，技能见 [.agents/skills/deploy/SKILL.md](.agents/skills/deploy/SKILL.md)。

## 7. 提交前检查清单

- [ ] `gofmt -w .` / `go vet ./...` / `go test ./...` / `go build ./...` 全绿
- [ ] 改了接口注解或 DTO 后跑过 `make swag`
- [ ] `easy_deploy validate --json` 对 `deploy.yaml` 返回 `ok:true`
- [ ] 六个目标至少 `plan --json` 能产出 Step（不执行）
- [ ] `deploy/k8s/` 快照与 `render --target k8s` 一致（改过端口/镜像/探针时）
- [ ] 无明文密钥；`.easy_deploy/`、`.env` 未被提交
- [ ] 文档四处同步：`AGENTS.md`、`docs/deployment.md`、`README.md`、`.agents/config.json`

## 8. 已知边界与既有状况

- `go test ./...` 目前有 2 个既有失败用例（`internal/app` 的 `TestAdminCanDeleteJob`、`TestSwaggerDocumentIsServed`），属于工作树中未重构完的部分；本轮部署工作未改动业务代码，不擅自修复。
- easy_deploy v1 不自动回滚、不推镜像仓库、不打包 Helm chart；需要时按 `docs/deployment.md` 第 6 节手工执行。
- 本机 Docker daemon 24.0.6 / compose v2.21 已启动、`doctor --target docker|compose` 通过；Docker Hub 鉴权不通已由 `build.args` 声明式换国内源适配（见 `docs/deployment.md` §8）。宿主磁盘曾写满导致 Docker VM 的 `/var/lib/docker` 被重挂只读，经用户授权重启 Docker Desktop 后已恢复可写（期间两个 `unless-stopped` 用户容器损坏的 `config.v2.json` 已用 `.tmp-config.v2.json*` 备份原位修复并按原始 ID 恢复）。
- `local` / `docker` / `compose` / `k8s` 四个目标均已**实际部署并验证**：`up --yes --json` ok、`health=ok`，`/health` 与 `/ready` 200。两应用当前**同时**运行在 `compose` 目标（宿主 8080/3000）与 `k8s`（namespace `aha`）上。compose 项目名 `edp-aha-webserver` / `edp-aha-react` 各自独立，`down` 互不干扰；k8s 部署为 NodePort，不占宿主端口，state 按 `<target>/<service>` 分键互不覆盖（实测 `down --target k8s` 只删本应用 Deployment/Service/Secret）。`status` / `logs` / `restart` 在 k8s 目标同样实测通过；`restart` 会重建 Service，**NodePort 动态端口会变化**，不要硬编码。切换到 `local` / `docker` / `compose` 这类占用宿主端口的目标前，仍需先 `down --target <旧> --yes --json` 释放 8080/3000。
- minikube 旧 profile 曾因 kubelet 客户端证书过期导致 `K8S_APISERVER_MISSING`；已用节点内有效 CA 重签证书并重启 kubelet 恢复（**未执行 `minikube delete`**，保留了两年前的既有负载）。`doctor --target k8s` 现通过。`k8s` Driver 不构建镜像，清单为 `imagePullPolicy: IfNotPresent`，部署前必须先把 `aha_webserver:latest` 送进节点（`minikube image load`），否则 Pod 会 `ImagePullBackOff`。
- minikube profile 原 `--cpus=2 --memory=4g` 在部署后曾被打满导致 apiserver 超时、Pod 反复重启；已用 `docker update --cpus 4 --memory 6g --memory-swap 6g minikube` 放宽并同步 `~/.minikube/profiles/minikube/config.json`（`CPUs=4` / `Memory=6000`，原文件备份 `.bak-20261108`）。若 `doctor --target k8s` 间歇性返回 3，先查 `docker stats minikube`，不要误判为清单或证书问题。
- `systemd`（无 systemctl）、`ssh`（远端不可达）仍返回 `PRECONDITION_MISSING`，只做 `plan` + `render`。
- `docker-compose.yml` 是 app + PostgreSQL 的人工编排；easy_deploy 生成的是单服务 compose，两者并存不冲突。

## 9. 权威声明

`AGENTS.md` 是唯一规则正文；`CLAUDE.md`、`.github/copilot-instructions.md`、`.cursor/rules/aha-webserver.mdc` 均为薄封装，不得复制规则正文，避免多处漂移。
