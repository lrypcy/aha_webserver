# 背景：aha_webserver 部署拓扑

## 六种目标的映射

```
                      deploy.yaml (runtime.port=8080, health=/health)
                                   │
        ┌──────────────┬───────────┴────────┬──────────────┬─────────────┐
        ▼              ▼                    ▼              ▼             ▼
     local          systemd               ssh           docker        compose / k8s
  go build →      go build →          rsync →        docker build →  image →
  ./bin/...       unit + systemctl   远端 go build   docker run     Deployment/Service
  pid + log       journalctl         nohup + pid    env-file       Secret + probes
        └──────────────┴────────────────────┴──────────────┴─────────────┘
                                   ▼
                     http_check GET http://<host>:8080/health
```

## 资产映射表

| 目标 | 构建 | 运行 | 健康 | 日志 | 状态 |
| --- | --- | --- | --- | --- | --- |
| local | `go build -o bin/aha_webserver .` | `./bin/aha_webserver` | HTTP 轮询 | `.easy_deploy/logs/aha-webserver.log` | `.easy_deploy/run/*.pid` |
| systemd | 同上 | `edp-aha-webserver.service` | HTTP 轮询 | journalctl | unit 文件 |
| ssh | 远端 `go build` | 远端 nohup | 远端 HTTP | 远端 `*.log` | 远端 `*.pid` |
| docker | `build/package/Dockerfile` | `edp-aha-webserver` | HEALTHCHECK + HTTP | `docker logs` | 容器 label |
| compose | 生成的 `compose.yaml` | compose service | healthcheck + HTTP | `docker compose logs` | compose 文件 |
| k8s | 现成镜像 | Deployment + Service | liveness/readiness | `kubectl logs` | `kubectl rollout status` |

## 配置加载链

```
进程环境 AHA_*  ──► viper.AutomaticEnv (prefix=AHA, "."→"_")
configs/config.yml ──► 非敏感默认值（server.env=dev, database.type=sqlite, jwt.secret=""）
内置默认值 ──► 兜底
        │
        ▼
 config.Load() → Validate()：production 强制 JWT/CORS/DB 口令；dev 生成临时 JWT 密钥
        │
        ▼
 Fiber 监听 :8080，注册 /health、/ready、/api/v1/*
```

**工作目录敏感**：`main.go` 用相对路径 `configs/config.yml` 读配置，因此裸机启动命令必须在仓库根目录执行（`runtime.workDir` 默认就是 `spec.dir`）。

## 仓库内两套编排

| 文件 | 服务 | 用途 |
| --- | --- | --- |
| `docker-compose.yml` | app + PostgreSQL 16 | 人工维护，带 DB 的完整本地环境 |
| easy_deploy 生成的 `compose.yaml` | 仅 app（sqlite） | 由 `deploy.yaml` 自动产出，用于 CI / 快速验证 |

两者端口都是 8080、镜像都是 `aha_webserver:latest`、健康路径都是 `/health`，保持一致。

## 端口与依赖

- 前端 `aha_react` 默认通过 `http://127.0.0.1:8080` 访问本服务；CORS 白名单需包含 `http://localhost:3000`。
- 生产切换 PostgreSQL/MySQL 时，还要提供 `AHA_DB_HOST`、`AHA_DB_PORT`、`AHA_DB_USER`、`AHA_DB_PASSWORD`，且数据库需先建好库（本服务不会自动 CREATE DATABASE）。
