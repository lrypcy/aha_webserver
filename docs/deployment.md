# aha_webserver 部署手册（easy_deploy）

> 唯一事实源是仓库根目录 [deploy.yaml](../deploy.yaml)。本文是面向人的说明；agent 规则以 [AGENTS.md](../AGENTS.md) 为准。
> `easy_deploy` 位于同级目录 `../easy_deploy`，先在那边执行 `make build` 得到 `bin/easy_deploy`（下文简写为 `edp`）。

## 1. 部署契约

| 项 | 值 |
| --- | --- |
| 清单版本 | `easydeploy/v1` |
| 服务名 | `aha-webserver` |
| 监听端口 | `8080`（容器内 / 进程内），宿主 `hostPort: 8080` |
| 健康端点 | `GET /health`（存活）、`GET /ready`（含数据库连通性） |
| 裸机构建 | `go build -o bin/aha_webserver .` |
| 裸机启动 | `./bin/aha_webserver`（工作目录必须是仓库根，因为要读 `configs/config.yml`） |
| 容器镜像 | `aha_webserver:latest`，Dockerfile `build/package/Dockerfile` |
| 环境变量前缀 | `AHA_*`，优先级高于 `configs/config.yml` |
| 敏感变量 | `AHA_JWT_SECRET`、`AHA_DB_PASSWORD`（只写 `${VAR}`，不落明文） |

## 2. 标准流程

```bash
cd /path/to/easy_deploy && make build
cd /path/to/aha_webserver
alias edp=/path/to/easy_deploy/bin/easy_deploy

edp validate --json                        # 严格校验 + 归一化 Spec
edp doctor --target local --json           # 环境预检（工具/端口/目录）
edp plan --target local --json             # 预演 Step 序列，不执行
edp up --target local --yes --json         # 部署 + /health 轮询
edp status --json
edp logs --tail 100 --json
edp down --target local --yes --json       # 按 state 回收
```

`--json` 下 stdout 只有一行 JSON；退出码 `0/2/3/4/5/6/7` 分别对应 成功 / SPEC_INVALID / PRECONDITION_MISSING / STEP_FAILED / HEALTH_FAILED / STATE_CONFLICT / CONFIRMATION_REQUIRED。详见 `../easy_deploy/docs/agent-protocol.md`。

## 3. 六个目标

| target | 会做什么 | 前置 |
| --- | --- | --- |
| `local` | 本机编译二进制、写 pid+log、轮询 `/health` | Go 1.23.5+、C 编译器（SQLite CGO）、8080 空闲 |
| `systemd` | 生成 `edp-aha-webserver.service` 并 `systemctl enable --now` | systemctl + 免密 sudo |
| `ssh` | rsync 同步源码到 `profiles.ssh.host`，远端编译并 nohup 启动 | ssh/scp/rsync + 远端 Go 工具链 |
| `docker` | `docker build` + `docker run --env-file`，带 label 与 HEALTHCHECK | docker daemon |
| `compose` | 生成 `compose.yaml`（单服务）并 `docker compose up -d --build` | docker compose 插件 |
| `k8s` | 生成 Namespace/Deployment/Service/Secret，`kubectl apply` + `rollout status` | kubectl + 可用 context |

只产出清单、不执行：

```bash
edp render --target k8s --out deploy/k8s
edp render --target compose --out /tmp/aha-compose
```

仓库内 `deploy/k8s/` 是 `render --target k8s` 的快照，可直接交给 ArgoCD / GitOps 流水线。**含密钥的 Secret 文件不入库**（见 `.gitignore`）。

## 4. 环境变量

完整说明见 [configuration.md](configuration.md) 与 [.env.example](../.env.example)。部署时常用的键：

| 键名 | 用途 | 部署注意 |
| --- | --- | --- |
| `AHA_SERVER_PORT` | 监听端口 | 改这里必须同步 `deploy.yaml` 的 `runtime.port` / `hostPort` |
| `AHA_SERVER_ENV` | `dev` \| `production` | production 会强制要求 `AHA_JWT_SECRET` ≥ 32 字符、显式 CORS 白名单、DB 口令 |
| `AHA_DB_TYPE` | `sqlite` \| `mysql` \| `postgresql` | 本机默认 sqlite 零配置；compose 自带的是 postgres 变体（见 `docker-compose.yml`） |
| `AHA_DB_NAME` | 库名 / sqlite 文件名 | 默认 `aha.db` |
| `AHA_JWT_SECRET` | JWT 签名密钥 | 生产必填，`openssl rand -hex 32` 生成；只经环境变量注入 |
| `AHA_DB_PASSWORD` | 数据库口令 | 生产必填；清单里只写 `${AHA_DB_PASSWORD}` |
| `AHA_CORS_ORIGINS` | CORS 白名单 | 前端在 `http://localhost:3000` 时按清单设置 |

> 生产部署示例：`--set spec.runtime.env.AHA_SERVER_ENV=production` 并在环境中导出 `AHA_JWT_SECRET` / `AHA_DB_PASSWORD`。不要在 `deploy.yaml` 里写死任何值。

## 5. 与仓库原有部署资产的关系

| 资产 | 角色 |
| --- | --- |
| `deploy.yaml` | easy_deploy 的统一声明式入口（六种目标） |
| `build/package/Dockerfile` | 多阶段镜像，被 `docker` / `compose` 复用 |
| `docker-compose.yml` | 人工维护的 app + PostgreSQL 16 编排；easy_deploy 生成的是单服务 compose，两者不冲突 |
| `Makefile` 的 `docker-*` | 原有快捷命令，保留；新增 `deploy-*` 走 easy_deploy |
| `deploy/k8s/` | `render --target k8s` 的快照，供集群侧直接 apply |

## 6. 状态、清理与回滚

- 状态文件：`.easy_deploy/state.json`；pid `.easy_deploy/run/`，日志 `.easy_deploy/logs/`，生成物 `.easy_deploy/generated/<target>/`。均被 `.gitignore` 忽略。
- 所有资源带 `easydeploy.io/managed=true`、`easydeploy.io/service=aha-webserver`、`easydeploy.io/target=<target>` 标签；`down` 只回收 state 里登记的资源。
- `down` 必须显式 `--yes`；`--purge` 才会删卷。
- 回滚：容器目标用上一版镜像 tag，k8s 用 `kubectl -n aha rollout undo deployment/aha-webserver`；裸机用 `git restore` 回到上一版源码后重新 `up`。

## 7. 故障排查

| 现象 | 退出码 | 处理 |
| --- | --- | --- |
| 8080 被占用 | 3 | `doctor --target local` 会报；改 `profiles.local.hostPort` |
| `go build` 失败（CGO / sqlite） | 4 | 安装 C 工具链；macOS `xcode-select --install` |
| `/health` 不通 | 5 | `logs --json`，确认 `configs/config.yml` 是从仓库根目录读取的 |
| 生产模式启动即退出 | 5 | 检查 `AHA_JWT_SECRET` / `AHA_CORS_ORIGINS` / `AHA_DB_PASSWORD` |
| `kubectl cluster-info` 失败 / apiserver 起不来 | 3 | 旧 minikube profile 常见根因是 kubelet 客户端证书过期（`kubeadm certificates have expired` + `K8S_APISERVER_MISSING`）；用节点内有效 CA 重签 kubelet 客户端证书并 `systemctl restart kubelet`，不要 `minikube delete` |
| k8s Pod `ImagePullBackOff` / `rollout status` 超时 | 4（`rollout-*` Step） | `k8s` Driver 不构建镜像且清单为 `imagePullPolicy: IfNotPresent`；先 `minikube image load aha_webserver:latest` 把镜像送进节点，`minikube image ls` 确认 |
| `systemd` 无 systemctl / `ssh` 远端不通 | 3 | 只 `render --target <t>` 产出清单，并在交付说明标注「未实际部署」 |
| docker daemon 未启动 | 3 | `docker info` 确认；或改用 `--target local`。本机 daemon 当前已启动，此条仅为通用排查项 |
| `docker build` 报 `auth.docker.io ... Bad Gateway` / `failed to authorize` | 4（`build-*` Step） | Docker Hub 鉴权不通，用 `build.args` 换国内镜像源，见 §8 |
| `go mod download` 超时 | 4 | 用 `EDP_GOPROXY` 指向可达模块代理 |
| `docker run` / `docker build` 报 `read-only file system`、`no space left on device` | 4（容器 Step） | Docker VM 的 `/var/lib/docker` 被重挂只读：`docker info` 仍正常但 `docker system df` / `docker run` 失败。先腾宿主磁盘，再**重启 Docker Desktop** 让 VM 重新挂载；仅腾磁盘不会自动恢复 |
| 构建报错只看到 `progress line ...`、看不到 `ERROR` 尾部 | 4 | 旧版 CLI 的 `truncate` 只留前缀；已改为保留头尾，`error.message` 现在会带上 `ERROR: ...` 根因 |

## 8. 镜像构建与网络适配（build.args）

本机 Docker daemon 可达，但 Docker Desktop 自带代理 `http.docker.internal:3128` 对 `auth.docker.io` 返回 Bad Gateway，`FROM golang:1.23-alpine` 无法解析；直连 Docker Hub 也超时。适配方式是**声明式换源**，不改 Docker 全局配置、不改 Dockerfile 的默认行为：

| 层 | 改动 |
| --- | --- |
| `build/package/Dockerfile` | 顶部新增 `ARG GO_BASE_IMAGE` / `ARG RUN_BASE_IMAGE` / `ARG GOPROXY`，默认仍是官方 `golang:1.23-alpine` / `alpine:3.20` / `proxy.golang.org`；`FROM ${...}`、`ENV GOPROXY=${GOPROXY}` 引用 |
| `deploy.yaml` | `build.args` 用 `${EDP_*:-国内源}` 注入镜像源与模块代理，默认即可用；`docker` 目标转成 `--build-arg`，`compose` 目标写进 `build.args` |
| easy_deploy | `build.args` 的值支持 `${VAR}` / `${VAR:-default}` 展开；compose 渲染补齐 `build.args` |

可覆盖回官方源（或任意内网镜像仓库）：

```bash
EDP_GO_BASE_IMAGE=golang:1.23-alpine \
EDP_RUN_BASE_IMAGE=alpine:3.20 \
EDP_GOPROXY=https://proxy.golang.org,direct \
easy_deploy up --target docker --yes --json
```

容器内实测可达：`dl-cdn.alpinelinux.org`（apk）、`goproxy.cn`、`registry.npmmirror.com`、`fonts.googleapis.com`。`build.args` 未解析的变量会报 `SPEC_INVALID`，不会静默降级；同一份 Document 多次 `Resolve` 不会互相污染（本轮修复了 `Build.Args` map 被原地改写导致环境变量覆盖失效的问题）。`aha_react` 用同样的机制注入 `NODE_BASE_IMAGE` / `NPM_REGISTRY`。

**本机部署现状（2026-11-08 晚，最终）**：网络与换源已就绪，三个基础镜像已就绪。宿主数据卷曾写满（仅剩 144Mi）触发 Docker VM 的 `/var/lib/docker` 被重挂只读；经用户授权重启 Docker Desktop 后 VM 恢复可写（宿主现有约 28Gi）。重启期间两个 `unless-stopped` 用户容器因只读期间 `config.v2.json` 被写成全零被 daemon 跳过，已用同目录 `.tmp-config.v2.json*` 有效备份原位修复，按原始容器 ID 恢复且均 healthy —— **重启 Docker Desktop 会影响其他容器，必须先取得授权并预先记录基线**。

`local` / `docker` / `compose` / `k8s` 四个目标均已实际部署并验证：`up --target <t> --yes --json` 返回 `ok:true`、`health=ok`，`/health` 与 `/ready` 200。当前两应用**同时**运行在 `compose` 目标（宿主 8080/3000）与 `k8s`（namespace `aha`）上；k8s 为 `Deployment 2/2 Ready` + NodePort Service，`kubectl -n aha exec ... -- wget -qO- http://127.0.0.1:8080/health` 返回 `{"code":0,...,"status":"ok"}`，宿主侧 `kubectl -n aha port-forward svc/aha-webserver 18080:8080` 也验证了 `/health` / `/ready` 200。`status` / `logs` / `restart` / `down` + `up` 在 k8s 目标均实测通过。

**NodePort 端口是动态分配的**：`restart` 或 `down` + `up` 会重建 Service，端口每次不同（实测 31609 → 31989）。不要硬编码，用 `kubectl -n aha get svc aha-webserver -o jsonpath='{.spec.ports[0].nodePort}'` 读取，或直接用 ClusterIP / port-forward。

**minikube 资源规格**：profile 原为 `--cpus=2 --memory=4g`。加载大镜像并部署 4 个 Pod 后，容器 cgroup 被打满（`docker stats` 约 `3.9GiB / 3.9GiB`、CPU 200%+），apiserver / etcd 出现 TLS handshake timeout、Pod 反复重启。已用 `docker update --cpus 4 --memory 6g --memory-swap 6g minikube` 即时放宽（不重启、不删集群），并把 `~/.minikube/profiles/minikube/config.json` 的 `CPUs=4` / `Memory=6000` 同步持久化（原文件备份为 `.bak-20261108`）。放宽后 4 个 Pod 全部 `1/1 Ready`，`doctor --target k8s` 恢复通过。下次 `minikube stop/start` 会按新配额创建容器。

k8s 用 NodePort、不占宿主端口，因此可与 compose 并存；state 按 `<target>/<service>` 分键，**无需为部署 k8s 先 `down compose`**。切换到 `local` / `docker` / `compose` 这类占用宿主 8080/3000 的目标前，才需要先 `down --target <旧> --yes --json`。另外生成的 `compose.yaml` 带顶层 `name: edp-<stack>`、命令显式 `-p edp-<stack>`，后端项目 `edp-aha-webserver` 与前端 `edp-aha-react` 各自独立，`down` 不会误删对方（否则按目录名推导会共用项目名 `compose`）。

k8s 有两项前置条件：`k8s` Driver 只 `kubectl apply` + `rollout status`，**不构建镜像**，清单固定 `imagePullPolicy: IfNotPresent`，所以部署前必须先 `minikube image load aha_webserver:latest` 把宿主镜像送进节点；旧 profile 的 kubelet 客户端证书过期会让 `minikube start` 报 `K8S_APISERVER_MISSING`，本轮用节点内有效 CA 重签证书后恢复，未删集群（详见 `../easy_deploy/docs/driver-guide.md` §9）。`systemd` / `ssh` 仍返回 `PRECONDITION_MISSING`，只能 `plan` + `render`。

## 9. Agent 执行环境注意（本机实测）

| 环境变量 | 影响 | 处理 |
| --- | --- | --- |
| `NODE_OPTIONS=--require .../node-language-shim.cjs` | 会污染子进程环境；在同类 Node 项目里可让 `npm run build` 报 `write EPIPE` | 启动 easy_deploy 时加 `env -u NODE_OPTIONS`，避免把 shim 传给子进程 |
| `HTTP_PROXY` / `HTTPS_PROXY` 指向本地代理 | `curl http://127.0.0.1:8080/health` 会被代理拦截，返回 502 `upstream connect failed` | 用 `curl --noproxy '*' ...`；Go 的 `http.Client` 对 loopback 自动绕过代理，easy_deploy 的 `http_check` 不受影响 |

`up` 之后服务需在 CLI 退出后继续存活：easy_deploy 的 `exec_background` 已用 `Setsid` 把进程放进独立会话，`down` 按进程组回收。验证时用 `status --json` 的 `result.health == "ok"` 与实际 `curl` 结果双重确认，不要用 `plan` / `render` 成功冒充部署成功。
