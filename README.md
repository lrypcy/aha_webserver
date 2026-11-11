# aha_webserver

面向 **Job → Task** 的任务管理型 HTTP 服务。基于 Go 1.23.5 重构，使用 Fiber v2 提供 REST API、GORM 负责持久化、Viper 管理配置、JWT（HS256）鉴权、swaggo 生成接口文档。

## 项目简介

服务把一次「作业（Job）」拆分为多个「任务（Task）」管理，提供作业/任务的创建、查询、状态流转，以及用户注册、登录与鉴权（`user` / `admin` 两种角色）。所有业务接口统一挂在 `/api/v1` 前缀下，响应统一封装并附带 `request_id` 便于链路排查。

## 特性清单

- **零配置启动**：默认 SQLite 文件库 + GORM AutoMigrate，无需外部数据库即可 `go run .`。
- **多数据库**：通过 `AHA_DB_TYPE` 切换 `sqlite` / `mysql` / `postgresql`。
- **统一响应体**：`{"code":0,"message":"ok","data":...,"request_id":"..."}`，HTTP 状态码仍遵循标准语义。
- **JWT 鉴权**：HS256 Bearer Token，claims 携带 `user_id` / `role`；`RequireRole` 中间件控制 admin 专属接口。
- **完整中间件**：requestid、logger、recover、cors、limiter、jwt、RequireRole。
- **可观测端点**：`/health`、`/ready`、`/swagger/*`。
- **工程化**：Makefile、多阶段 Dockerfile、docker-compose（app + PostgreSQL 16）、GitHub Actions CI、一键校验脚本。
- **环境变量优先**：`configs/config.yml` 只放非敏感默认值，`AHA_*` 环境变量覆盖，密钥不落盘到仓库。

## 快速开始（SQLite 零配置）

> 前置：Go 1.23.5+，且本机有 C 编译器（SQLite 驱动 `mattn/go-sqlite3` 依赖 CGO；macOS 执行 `xcode-select --install`，Debian/Ubuntu 执行 `apt install build-essential`）。

```bash
# 1) 进入项目
cd aha_webserver

# 2) （可选）准备本地配置与环境文件
cp configs/config.example.yml configs/config.yml   # config.yml 已存在则可跳过
cp .env.example .env                               # 按需修改，切勿提交
# 或直接跑一键脚本：./scripts/dev-setup.sh

# 3) 启动：默认 sqlite、端口 8080，首次启动自动 AutoMigrate 建表
go run .
# 或： make run

# 4) 验证
curl http://localhost:8080/health
curl http://localhost:8080/ready
```

浏览器打开 <http://localhost:8080/swagger/index.html> 查看接口文档（需先执行 `make swag` 生成 `api/` 产物）。

> 不想建 `.env` 时可直接导出密钥：`export AHA_JWT_SECRET=$(openssl rand -hex 32)` 后再 `make run`。

## 配置与环境变量

配置优先级：**`AHA_*` 环境变量 > `configs/config.yml` > 内置默认值**。
`configs/config.yml` 只放**非敏感默认值**；`configs/config.example.yml` 是带注释的示例；含密钥的本地私有配置可放 `configs/config.local.yml`（已被 `.gitignore` 忽略）。

Viper 通过 `AutomaticEnv` + `AHA_` 前缀读取环境变量，并把配置键的点号映射为下划线，例如 `server.port` ↔ `AHA_SERVER_PORT`、`db.type` ↔ `AHA_DB_TYPE`。

| 环境变量 | 说明 | 默认值 |
| --- | --- | --- |
| `AHA_SERVER_PORT` | HTTP 监听端口 | `8080` |
| `AHA_SERVER_ENV` | 运行环境：`development` / `production` | 见 `configs/config.yml` |
| `AHA_DB_TYPE` | 数据库类型：`sqlite` / `mysql` / `postgresql` | `sqlite` |
| `AHA_DB_HOST` | 数据库主机（sqlite 忽略） | 见 `configs/config.yml` |
| `AHA_DB_PORT` | 数据库端口（sqlite 忽略） | 见 `configs/config.yml` |
| `AHA_DB_USER` | 数据库用户名（sqlite 忽略） | 见 `configs/config.yml` |
| `AHA_DB_PASSWORD` | 数据库密码（sqlite 忽略） | 见 `configs/config.yml` |
| `AHA_DB_NAME` | 库名；sqlite 时为数据文件名 | 见 `configs/config.yml` |
| `AHA_JWT_SECRET` | JWT HS256 签名密钥，**生产必填** | 空（生产禁止为空） |
| `AHA_JWT_EXPIRATION` | Token 有效期，Go duration 格式 | `24h` |
| `AHA_LOG_LEVEL` | 日志级别 | 见 `configs/config.yml` |
| `AHA_CORS_ORIGINS` | 允许的跨域来源，逗号分隔 | 见 `configs/config.yml` |
| `AHA_RATE_LIMIT_MAX` | 限流窗口内最大请求数 | 见 `configs/config.yml` |

生成强密钥：

```bash
openssl rand -hex 32
```

把输出写入 `.env` 的 `AHA_JWT_SECRET`。**不要把真实密钥提交到 Git**（`.env`、`configs/config.local.yml` 均已被忽略）。

## API 速查表

统一前缀 `/api/v1`；除注册/登录外均需 `Authorization: Bearer <token>`。

| 方法 | 路径 | 说明 | 角色 |
| --- | --- | --- | --- |
| POST | `/api/v1/register` | 用户注册 | 公开 |
| POST | `/api/v1/login` | 登录，返回 JWT | 公开 |
| GET | `/api/v1/me` | 当前登录用户信息 | user / admin |
| POST | `/api/v1/jobs` | 创建 Job | user / admin |
| GET | `/api/v1/jobs` | Job 分页列表 | user / admin |
| GET | `/api/v1/jobs/:id` | Job 详情 | user / admin |
| DELETE | `/api/v1/jobs/:id` | 删除 Job | **admin** |
| POST | `/api/v1/tasks` | 创建 Task | user / admin |
| GET | `/api/v1/tasks` | Task 分页列表，可按 `job_id` / `status` 过滤 | user / admin |
| GET | `/api/v1/tasks/:id` | Task 详情 | user / admin |
| PATCH | `/api/v1/tasks/:id/status` | 更新 Task 状态 | user / admin |

非版本化端点：

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | `/health` | 存活探针（Docker HEALTHCHECK 使用） |
| GET | `/ready` | 就绪探针（数据库等依赖可用） |
| GET | `/swagger/*` | Swagger UI |

统一响应体（成功）：

```json
{
  "code": 0,
  "message": "ok",
  "data": { "id": 1, "status": "pending" },
  "request_id": "8f3c1e..."
}
```

失败时 `code` 为非 0、`message` 为错误描述、`data` 通常为 `null`，并同样携带 `request_id`。HTTP 状态码仍使用标准值：`400`（参数错误）、`401`（未认证）、`403`（无权限）、`404`（不存在）、`409`（冲突）、`429`（限流）、`500`（服务内部错误）。

分页约定：`page`（从 1 开始，默认 1）、`page_size`（默认 20，上限 100），非法分页返回 400；列表接口 `data` 形如 `{"items":[...],"total":123,"page":1,"page_size":20}`。

业务错误码（与 HTTP 状态码配套）：

| code | HTTP | 含义 |
| --- | --- | --- |
| 1001 | 400 | 请求参数 / 请求体错误 |
| 1002 | 400 | 分页参数非法 |
| 2001 | 401 | 未认证、Token 无效或过期 |
| 3001 | 403 | 权限不足 |
| 4001 | 404 | 资源不存在 |
| 5001 | 409 | 资源冲突（如用户名 / 邮箱重复注册） |
| 6001 | 429 | 请求过于频繁 |
| 9999 | 500 | 服务器内部错误 |

## 认证示例

```bash
BASE=http://localhost:8080/api/v1

# 1) 注册
curl -X POST "$BASE/register" -H 'Content-Type: application/json' -d '{"username":"alice","email":"alice@example.com","password":"Passw0rd!"}'

# 2) 登录拿 Token（响应 data 中的 token 即为 JWT）
curl -X POST "$BASE/login" -H 'Content-Type: application/json' -d '{"username":"alice","password":"Passw0rd!"}'

export TOKEN='<粘贴上一步返回的 token>'

# 3) 携带 Token 访问受保护接口
curl -H "Authorization: Bearer $TOKEN" "$BASE/me"
curl -H "Authorization: Bearer $TOKEN" "$BASE/jobs?page=1&page_size=10"

# 4) admin 专属接口（需要 role=admin 的账号）
curl -X DELETE -H "Authorization: Bearer $TOKEN" "$BASE/jobs/1"
```

> 请求/响应字段名以 `internal/dto` 中的定义为准：注册需 `username` / `email` / `password`（密码至少 8 字节），登录需 `username` / `password`；登录成功的 `data` 含 `token`、`token_type`、`expires_in`、`user`。示例用户名与密码仅用于本地演示，生产环境请使用强密码，并只通过环境变量注入密钥。

## 从旧路由迁移到 `/api/v1`

**旧版裸路由 `/login`、`/register`、`/task`、`/job` 已全部下线，统一收敛到 `/api/v1/*`**（复数资源名 + 统一前缀），不再保留旧地址。

| 旧路由（重构前） | 新路由（重构后） |
| --- | --- |
| `POST /login` | `POST /api/v1/login` |
| `POST /register` | `POST /api/v1/register` |
| `POST /job` | `POST /api/v1/jobs` |
| `GET /job/:id` | `GET /api/v1/jobs/:id` |
| `GET /job`（列表） | `GET /api/v1/jobs`（分页） |
| — | `DELETE /api/v1/jobs/:id`（新增，仅 admin） |
| `POST /task` | `POST /api/v1/tasks` |
| `GET /task`（列表） | `GET /api/v1/tasks`（分页，支持 `job_id` / `status` 过滤） |
| `GET /task/:id` | `GET /api/v1/tasks/:id` |
| — | `PATCH /api/v1/tasks/:id/status`（新增） |

迁移要点：

1. **路径**：全部加 `/api/v1`，`/job` → `/jobs`、`/task` → `/tasks`；`/login`、`/register` 同样加前缀。
2. **认证**：旧版裸接口无鉴权；现在除注册/登录外都要 `Authorization: Bearer <token>`，删除 Job 还需 `role=admin`。
3. **响应**：旧版直接返回裸对象，现在统一包一层 `code/message/data/request_id`，前端取业务数据改读 `data`。
4. **分页/过滤**：列表接口改为分页返回，并支持 `job_id` / `status` 过滤。
5. **探针与文档**：`/health`、`/ready`、`/swagger/*` 保持不带版本前缀。

## 测试

```bash
make test          # go test ./...
make test-cover    # -race + coverage.out，并打印函数级覆盖率
make lint          # go vet ./...
./scripts/verify.sh   # 一键：gofmt 检查 → vet → test → build，失败即退出
```

CI（`.github/workflows/ci.yml`）在 Go 1.23.x 上执行 gofmt 检查、`go vet ./...`、`go test -race ./...`、`go build ./...`，并缓存 Go modules。

## Docker 运行

```bash
cp .env.example .env     # 务必修改 AHA_JWT_SECRET（openssl rand -hex 32）
make docker-build        # 仅构建镜像
make docker-up           # 等价于 docker compose up -d --build（app + postgres:16-alpine）
make docker-down         # 停止并移除容器
```

- 端口映射 `8080:8080`；app 通过内部网络以 `AHA_DB_HOST=postgres` 连接 PostgreSQL。
- **数据库需预先建好**：compose 通过 `POSTGRES_DB` / `POSTGRES_USER` / `POSTGRES_PASSWORD` 初始化 PostgreSQL 实例，app 的 `AHA_DB_NAME` 指向同一个库；数据库层本身不会自动 `CREATE DATABASE`，仅做 GORM AutoMigrate 建表。
- PostgreSQL 数据落在命名卷 `pgdata`，`docker compose down` 不会删除数据。
- app 与 postgres 均配置健康检查，app 依赖 postgres `service_healthy` 后才启动。
- 镜像为多阶段构建：构建阶段 `golang:1.23-alpine` + `gcc/musl-dev`（CGO，供 `go-sqlite3` 编译），运行阶段 `alpine` + `ca-certificates` + `tzdata`，以非 root 用户运行，`HEALTHCHECK` 指向 `/health`。

```bash
curl http://localhost:8080/health
docker compose ps        # 查看 STATUS (healthy)
```

## 目录结构

```text
aha_webserver/
├── main.go                          # 程序入口
├── internal/
│   ├── app/                         # 应用装配与启动流程
│   ├── config/                      # 配置加载（viper + AHA_ 环境变量）
│   ├── response/                    # 统一响应体
│   ├── database/                    # GORM 初始化 / AutoMigrate
│   ├── dto/                         # 请求与响应 DTO
│   ├── model/                       # Job / JobResult / Task / User / SysConfig
│   ├── middleware/
│   │   ├── jwt/                     # JWT 签发与校验
│   │   └── auth/                    # RequireRole 等鉴权中间件
│   ├── controller/
│   │   ├── auth/                    # register / login / me
│   │   ├── job/                     # Job 接口
│   │   ├── task/                    # Task 接口
│   │   └── health/                  # health / ready
│   └── router/                      # 路由注册（/api/v1）
├── configs/
│   ├── config.yml                   # 非敏感默认配置
│   └── config.example.yml           # 配置示例
├── api/                             # swag 生成物（docs.go / swagger.json / swagger.yaml）
├── scripts/
│   ├── verify.sh                    # 一键校验
│   ├── dev-setup.sh                 # 开发环境初始化
│   └── casbin.sh                    # 既有：初始化 SQL 脚本
├── init/casbin.sql                  # 初始化 SQL
├── build/package/Dockerfile         # 多阶段 Docker 镜像
├── docker-compose.yml               # app + postgres:16-alpine
├── .github/workflows/ci.yml         # CI 流水线
├── .env.example                     # 环境变量示例
├── .dockerignore
├── .gitignore
├── .agents/                         # 智能体协作说明
├── docs/                            # 设计文档
├── Makefile
└── README.md
```

> 另：`go.mod` / `go.sum` / `LICENSE` / `.vscode/`（tasks.json、launch.json、settings.json）等为仓库既有文件。

常用 Make 目标：`make help` 查看全部；`make fmt`（`gofmt -w .`）、`make tidy`、`make lint`、`make build`（输出 `bin/aha_webserver`）、`make run`、`make test`、`make test-cover`、`make swag`、`make swag_clean`、`make docker-build`、`make docker-up`、`make docker-down`、`make clean`。

## 常见问题（FAQ）

**Q：`go run .` 报 `Binary was compiled with CGO_ENABLED=0` 或找不到 `gcc`？**
A：`mattn/go-sqlite3` 需要 CGO 与 C 编译器。设置 `CGO_ENABLED=1` 并安装编译器（macOS：`xcode-select --install`；Debian/Ubuntu：`sudo apt install build-essential`；Alpine：`apk add gcc musl-dev`）。Dockerfile 已内置处理。

**Q：生产环境启动失败，提示缺少 JWT 密钥？**
A：`AHA_JWT_SECRET` 生产必填。用 `openssl rand -hex 32` 生成并通过环境变量注入，不要写进配置文件或提交到仓库。

**Q：端口 8080 被占用？**
A：`AHA_SERVER_PORT=9090 make run`；compose 场景同时调整 `docker-compose.yml` 的端口映射。

**Q：`/swagger/*` 404 或文档是旧的？**
A：执行 `make swag` 重新生成 `api/docs.go`、`api/swagger.json`、`api/swagger.yaml`（`docs.go` 被 `main.go` 导入，缺失会导致 `go build` 失败，请勿手动删除）。`make swag_clean` 只清理临时残留，不会删这三个生成物。

**Q：VS Code 里 `make fmt` 任务报「No rule to make target」？**
A：旧 Makefile 缺少 `fmt` 目标，现已补上 `gofmt -w .`；`tasks.json` 的 build / tidy / fmt 三个任务均可直接使用。

**Q：返回 401 / 403？**
A：401 表示未带或 Token 失效（检查 `Authorization: Bearer` 与 `AHA_JWT_EXPIRATION`）；403 表示角色不足，例如 `DELETE /api/v1/jobs/:id` 需要 `role=admin`。

**Q：返回 429？**
A：触发限流，检查 `AHA_RATE_LIMIT_MAX`，或稍后重试。

**Q：如何切换到 MySQL / PostgreSQL？**
A：设置 `AHA_DB_TYPE=mysql|postgresql` 及 `AHA_DB_HOST/PORT/USER/PASSWORD/NAME`。注意数据库与库需**预先创建**（服务只做 AutoMigrate 建表，不自动 `CREATE DATABASE`）；compose 已用 `POSTGRES_DB` 预建库。

**Q：`.env` / `*.db` 会不会被提交？**
A：不会，已在 `.gitignore` 中忽略。注意 `.vscode/` 目录已入库（含 tasks.json 等），忽略规则对已跟踪文件不生效。

**Q：脚本没有执行权限？**
A：`chmod +x scripts/verify.sh scripts/dev-setup.sh`，或用 `bash scripts/verify.sh` 运行。

## 统一部署（easy_deploy）

本仓库用根目录 [deploy.yaml](deploy.yaml) 作为**唯一部署事实源**，通过同级目录的 `easy_deploy` CLI 部署到本机裸机、systemd、SSH 远端、Docker、Compose、Kubernetes 六种目标。完整手册见 [docs/deployment.md](docs/deployment.md)，agent 规则见 [AGENTS.md](AGENTS.md) 第 6 节。

```bash
cd ../easy_deploy && make build      # 生成 bin/easy_deploy
cd ../aha_webserver
make deploy-check                    # validate + doctor + plan（只预演）
make deploy-local                    # 本机裸机部署，/health 轮询验证
make deploy-status
make deploy-down                     # 按 state 回收
DEPLOY_TARGET=k8s make deploy-plan   # 预演 k8s 的 Step 序列
make deploy-render                   # 刷新 deploy/k8s/ 快照（Secret 不入库）
```

所有命令支持 `--json`（结构化 envelope）与稳定退出码，适合 AI agent 无交互调用；`down` 必须显式 `--yes`。`.easy_deploy/` 存放 state / pid / log / 生成物，已被 `.gitignore` 忽略。

