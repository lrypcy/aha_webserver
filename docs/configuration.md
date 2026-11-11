# 配置指南

## 配置来源与优先级

- `configs/config.yml`：非敏感默认值，随项目提交。
- `configs/config.example.yml`：可提交的示例配置，**不含任何真实密钥**。
- 环境变量：前缀 `AHA_`，由 Viper 的 `SetEnvPrefix("AHA")` + `SetEnvKeyReplacer(".", "_")` + `AutomaticEnv()` 生效（`internal/config` 统一加载）。

优先级从高到低：`viper.Set`（显式代码设置） > **环境变量 `AHA_`** > `configs/config.yml` > `viper.SetDefault`。配置键里的 `.` 映射成环境变量里的 `_`，例如 `server.port` → `AHA_SERVER_PORT`、`db.type` → `AHA_DB_TYPE`。

Viper 注意：`AutomaticEnv()` 只对 `viper.Get*` 生效；若用 `Unmarshal(&cfg)` 把配置读进结构体，必须先对每个键 `SetDefault`，否则环境变量不会落进结构体。

## 配置项与环境变量对照表

| 配置键（config.yml） | 环境变量 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `server.port` | `AHA_SERVER_PORT` | `8080` | HTTP 监听端口 |
| `server.env` | `AHA_SERVER_ENV` | 见 `configs/config.yml` | 运行环境（如 dev / prod）；prod 时 `AHA_JWT_SECRET` 必填 |
| `db.type` | `AHA_DB_TYPE` | `sqlite` | `sqlite|mysql|postgresql` |
| `db.host` | `AHA_DB_HOST` | 见 `configs/config.yml` | 数据库主机；sqlite 时忽略 |
| `db.port` | `AHA_DB_PORT` | 见 `configs/config.yml` | 数据库端口；sqlite 时忽略 |
| `db.user` | `AHA_DB_USER` | 无默认（mysql / postgresql 必填） | 数据库用户名 |
| `db.password` | `AHA_DB_PASSWORD` | 无默认（敏感，**不写进 config.yml**） | 数据库密码 |
| `db.name` | `AHA_DB_NAME` | 见 `configs/config.yml` | 库名；sqlite 时为数据库文件路径 |
| `jwt.secret` | `AHA_JWT_SECRET` | **无默认，生产必填** | JWT HS256 签名密钥；为空且 `AHA_SERVER_ENV=prod` 时拒绝启动 |
| `jwt.expiration` | `AHA_JWT_EXPIRATION` | `24h` | Token 有效期，Go duration 格式 |
| `log.level` | `AHA_LOG_LEVEL` | 见 `configs/config.yml` | 日志级别 |
| `cors.origins` | `AHA_CORS_ORIGINS` | 见 `configs/config.yml` | 允许的来源，逗号分隔；生产不要用 `*` |
| `rate_limit.max` | `AHA_RATE_LIMIT_MAX` | 见 `configs/config.yml` | limiter 窗口内最大请求数，超限返回 `429` + code `6001` |

密钥与密码只允许来自 `AHA_JWT_SECRET` / `AHA_DB_PASSWORD` 等环境变量，禁止写入 `configs/config.yml`、`configs/config.example.yml`、代码常量或注释。

## 三种数据库 DSN 示例

DSN 由 `internal/database` 根据上述环境变量拼装；下面给出拼装后的**形态**，不要把整串写进 `configs/config.yml`，也不要在代码里硬编码账号密码。

### 1. sqlite（默认）

```bash
AHA_DB_TYPE=sqlite
AHA_DB_NAME=./data/aha.db
```

DSN 形态：`./data/aha.db`（文件路径本身即 DSN）。前置动作：保证父目录存在，服务不会自动创建目录：

```bash
mkdir -p ./data
```

### 2. mysql

```bash
AHA_DB_TYPE=mysql
AHA_DB_HOST=127.0.0.1
AHA_DB_PORT=3306
AHA_DB_USER=aha
AHA_DB_PASSWORD=<只走环境变量，勿写入任何文件>
AHA_DB_NAME=aha_webserver
```

DSN 形态：

```text
aha:<password>@tcp(127.0.0.1:3306)/aha_webserver?charset=utf8mb4&parseTime=True&loc=Local
```

前置动作 —— **数据库层已移除自动 `CREATE DATABASE`，库必须预先创建**：

```sql
CREATE DATABASE IF NOT EXISTS aha_webserver CHARACTER SET utf8mb4;
```

### 3. postgresql

```bash
AHA_DB_TYPE=postgresql
AHA_DB_HOST=127.0.0.1
AHA_DB_PORT=5432
AHA_DB_USER=aha
AHA_DB_PASSWORD=<只走环境变量，勿写入任何文件>
AHA_DB_NAME=aha_webserver
```

DSN 形态：

```text
host=127.0.0.1 user=aha password=<password> dbname=aha_webserver port=5432 sslmode=disable TimeZone=Asia/Shanghai
```

前置动作 —— 同样需要**预先建库**：

```bash
createdb -h 127.0.0.1 -p 5432 -U aha aha_webserver
```

未预先建库时 `gorm.Open` 会直接失败（`Unknown database` / `database does not exist`）；服务不会自动建库，也不会自动降级到 sqlite。迁移脚本放 `scripts/`，初始化 SQL 见 `init/casbin.sql`。

连接成功后 `internal/database` 执行 `AutoMigrate(Job, JobResult, Task, User, SysConfig)`，顺序固定（`Job` 必须在 `JobResult` / `Task` 之前）。`AutoMigrate` 只增不删，删列 / 改类型 / 改表名需要 `scripts/` 下的显式迁移。

## 启动示例

```bash
# 默认 sqlite，端口 8080
AHA_SERVER_PORT=8080 go run .

# mysql（先执行上面的 CREATE DATABASE）
AHA_DB_TYPE=mysql AHA_DB_HOST=127.0.0.1 AHA_DB_PORT=3306 AHA_DB_USER=aha AHA_DB_PASSWORD=... AHA_DB_NAME=aha_webserver AHA_JWT_SECRET=... AHA_JWT_EXPIRATION=24h go run .
```

## 生产检查清单

- [ ] `AHA_JWT_SECRET` 已设置且足够随机；`AHA_SERVER_ENV=prod` 时缺失必须启动失败
- [ ] `AHA_DB_PASSWORD` 只走环境变量，未出现在 `configs/config.yml` / `configs/config.example.yml` / 代码 / 日志
- [ ] mysql / postgresql 已**预先建库**；sqlite 的 `AHA_DB_NAME` 父目录已存在
- [ ] `AHA_CORS_ORIGINS` 未配置为 `*`
- [ ] `AHA_RATE_LIMIT_MAX` 与 limiter、`AHA_LOG_LEVEL` 符合生产要求
- [ ] 服务监听 `AHA_SERVER_PORT`（默认 `8080`），`/health` 与 `/ready` 可访问
- [ ] `api/` 生成物是 `swag init --output api --parseInternal --parseDependency` 产出的最新版本

接口用法见 `docs/api-guide.md`。
