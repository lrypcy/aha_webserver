# 规则：部署契约（easy_deploy）

## 1. 唯一事实源

`deploy.yaml`（`apiVersion: easydeploy/v1`，`kind: Service`）描述本服务在六种目标上的部署需求。任何关于端口、镜像、构建命令、启动命令、健康路径、环境变量的改动，**先改 `deploy.yaml`**，再同步下游资产。禁止在 `.agents/` 或 `docs/` 里另写一份互相矛盾的参数。

## 2. 必须同步的资产

| 改动 | 必须同步 |
| --- | --- |
| `runtime.port` / `hostPort` | `Dockerfile` 的 `EXPOSE` 与 `HEALTHCHECK`、`docker-compose.yml` 的 `ports`、`.env.example` 的 `AHA_SERVER_PORT`、`docs/deployment.md`、`deploy/k8s/` 快照 |
| `runtime.health.path` | `Dockerfile` 的 `HEALTHCHECK`、`docker-compose.yml` 的 `healthcheck`、`docs/deployment.md`、`deploy/k8s/` 的 liveness/readiness 探针 |
| `spec.image` / `build.dockerfile` | `Makefile` 的 `IMAGE` / `DOCKERFILE`、`docker-compose.yml`、`README.md` |
| `runtime.env` 新增键 | `.env.example`（只列键名）、`configs/config.example.yml`、`docs/configuration.md`、`AGENTS.md` 第 6 节、`.agents/config.json` |
| `build.command` / `runtime.command` | `Makefile` 的 `build` / `run`、`docs/deployment.md` |

改完执行 `easy_deploy render --target k8s --out deploy/k8s` 刷新快照；**含密钥的 Secret 文件不入库**。

## 3. 环境变量与密钥

- `deploy.yaml` 的 `env` 只放**非敏感**值；`AHA_JWT_SECRET` / `AHA_DB_PASSWORD` 一律写 `${VAR}`，由进程环境注入。
- `.env.example` 只列键名与占位说明，禁止写入真实值。
- 生产模式（`AHA_SERVER_ENV=production`）会强制要求 ≥32 字符 JWT 密钥、显式 CORS 白名单、非空 DB 口令；缺失时服务启动即退出，属于 `HEALTH_FAILED`，不是 CLI 的 bug。
- 生成的 `.easy_deploy/generated/<target>/*.env`、`*-secret.yaml` 权限 0600 且被忽略，不能提交。

## 4. 探针约定

- `/health` 是存活探针（不含依赖检查），`/ready` 含数据库连通性。
- 两者挂在根路径，不受 `/api/v1`、鉴权、限流影响。
- easy_deploy 用它生成 `http_check` Step、Docker HEALTHCHECK、compose healthcheck、k8s liveness/readinessProbe。
- 禁止把探针改成依赖外部服务的接口，否则 k8s 会因短暂依赖抖动反复重启 Pod。

## 5. 标准流程（agent 必须照做）

```
validate --json → doctor --target <t> --json → plan --target <t> --json
  → up --target <t> --yes --json → status --json → (失败) logs --json → down --yes --json
```

- `plan` 与 `--dry-run` 只预演，绝不执行；`render` 只产出原生清单。
- `down` 必须显式 `--yes`，`--purge` 才删卷；agent 不得静默替用户确认破坏性操作。
- `doctor` 报 `PRECONDITION_MISSING` 时**不要硬上**：换可用 target 或只 render，并在交付说明标注「仅生成清单，未实际部署」。
- `doctor --target docker` 通过**不等于**镜像可构建：Docker VM 只读会让 `docker build` 在提交层时失败（退出码 4）。诊断 `docker system df` / `docker run --rm alpine true`，恢复需先腾宿主磁盘再重启 Docker Desktop（须用户授权；只读期间可能写坏运行中容器的 `config.v2.json`，先记录基线并准备从 `.tmp-config.v2.json*` 恢复）。
- `compose` 目标生成的 `compose.yaml` 必须带顶层 `name: edp-<stack>`、命令显式 `-p edp-<stack>`，否则共用项目名会让 `down --remove-orphans` 跨应用误删。切换 `docker`/`compose`/`local` 前先 `down` 旧目标释放端口。
- 退出码分流见 `../easy_deploy/docs/agent-protocol.md`；失败先读 `error.hint`。

## 6. 状态与清理

- `.easy_deploy/` 被 `.gitignore` 忽略，含 state / pid / log / generated。
- 所有资源带 `easydeploy.io/managed=true`、`easydeploy.io/service=aha-webserver`、`easydeploy.io/target=<target>` 标签。
- `down` 只回收 state 中登记的资源；手工 `docker rm` / `kill` 会导致 state 失联，需按标签清理。
- 不要提交 `.env`、`.easy_deploy/`、任何含密钥的生成物。

## 7. 与人工资产的关系

`build/package/Dockerfile` 与 `docker-compose.yml` 继续由人维护（`docker-compose.yml` 是 app+PostgreSQL 编排）；easy_deploy 生成的单服务 compose 是另一份产物，二者不冲突，但端口/镜像/健康路径必须一致。

## 8. 禁止事项

- 禁止在 `deploy.yaml` 里写死任何口令 / token。
- 禁止为了「让部署通过」而关闭生产模式的密钥校验或把 CORS 改成 `*`。
- 禁止在未跑 `doctor` 的情况下声称 docker / k8s 部署成功。
- 禁止用 `render` 成功冒充实际部署成功。
- 禁止在环境不可达时硬上 `up`；只 `render` 时必须在交付说明标注「仅生成清单，未实际部署」。

## 9. Agent 执行环境注意

- agent shell 会设置 `NODE_OPTIONS=--require .../node-language-shim.cjs`；启动 easy_deploy 时用 `env -u NODE_OPTIONS`，避免把 shim 传给子进程。
- agent shell 会设置 `HTTP_PROXY` / `HTTPS_PROXY`。`curl` 验证 `127.0.0.1` 必须加 `--noproxy '*'`，否则 502 `upstream connect failed` 是代理拦截、不是服务挂了。Go 的 `http.Client` 对 loopback 自动绕过代理，easy_deploy 的 `http_check` 不受影响。
- 判断服务是否真的运行，以 `status --json` 的 `result.health` 与手工 `curl /health` 双重确认。
- `exec_background` 已用 `Setsid` 脱离会话、`down` 按进程组回收；不要手工只 kill pid 文件的父进程。
