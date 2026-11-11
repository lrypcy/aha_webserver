# API 使用指南

## 基础信息

- Base URL：`http://localhost:8080`（端口来自 `AHA_SERVER_PORT`，默认 `8080`）
- 业务 API 前缀：`/api/v1`
- 非版本化端点：`GET /health`、`GET /ready`、`GET /swagger/*`
- 认证：`Authorization: Bearer <token>`。公开端点只有 `/register`、`/login`、`/health`、`/ready`、`/swagger/*`；其余 `/api/v1` 端点都要 Token，`DELETE /api/v1/jobs/:id` 还要 `admin` 角色。

统一响应体（`/api/v1` 所有响应）：

```json
{"code":0,"message":"ok","data":{},"request_id":"..."}
```

`code` 为业务码（成功 `0`），`request_id` 由服务端生成、回传用于日志检索；HTTP 状态码仍用标准值 `400/401/403/404/409/429/500`。列表接口用 `page`（默认 1）与 `page_size`（默认 20、上限 100），`data` 形如 `{"items":[...],"total":123,"page":1,"page_size":20}`。

## 1. 健康检查与文档

```bash
curl -s http://localhost:8080/health
curl -s http://localhost:8080/ready
curl -s -o /dev/null -w "%{http_code}
" http://localhost:8080/swagger/index.html
```

Swagger UI：`http://localhost:8080/swagger/index.html`，文档数据 `http://localhost:8080/swagger/doc.json`（来自 `api/` 生成物，改注解后需 `swag init --output api --parseInternal --parseDependency`）。

## 2. 注册 → 登录 → 拿 Token

```bash
curl -s -X POST http://localhost:8080/api/v1/register   -H "Content-Type: application/json"   -d '{"username":"alice","email":"alice@example.com","password":"{{SECRET_uhac3dzi7}}"}'
```

响应 `201`，`data` 为用户信息（**不含密码 / 哈希**）。用户名或邮箱已存在返回 `409` + code `5001`。

```bash
curl -s -X POST http://localhost:8080/api/v1/login   -H "Content-Type: application/json"   -d '{"username":"alice","password":"{{SECRET_uhac3dzi7}}"}'
```

把返回的 `data.token` 存成 shell 变量（需要 `jq`；没有就手动复制）：

```bash
TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/login   -H "Content-Type: application/json"   -d '{"username":"alice","password":"{{SECRET_uhac3dzi7}}"}' | jq -r '.data.token')
echo "$TOKEN"
```

Token 为 JWT HS256，默认有效期 `24h`（`AHA_JWT_EXPIRATION`），claims 含 `user_id` / `role`。

## 3. 当前用户

```bash
curl -s http://localhost:8080/api/v1/me -H "Authorization: Bearer $TOKEN"
```

不带 `Authorization` 返回 `401` + code `2001`。

## 4. 创建 Job（带 Bearer）

```bash
curl -s -X POST http://localhost:8080/api/v1/jobs   -H "Authorization: Bearer $TOKEN"   -H "Content-Type: application/json"   -d '{"title":"demo-job"}'
```

返回 `201`，`data` 为创建的 Job。

## 5. Job 列表与详情（分页）

```bash
curl -s "http://localhost:8080/api/v1/jobs?page=1&page_size=20"   -H "Authorization: Bearer $TOKEN"

curl -s http://localhost:8080/api/v1/jobs/1   -H "Authorization: Bearer $TOKEN"
```

`page` 非整数或 `page_size` 超 100 → `400` + code `1002`；不存在的 id → `404` + code `4001`。

## 6. 创建 Task（带 Bearer）

```bash
curl -s -X POST http://localhost:8080/api/v1/tasks   -H "Authorization: Bearer $TOKEN"   -H "Content-Type: application/json"   -d '{"job_id":1,"title":"demo-task"}'
```

## 7. Task 列表（过滤 + 分页）与详情

```bash
curl -s "http://localhost:8080/api/v1/tasks?job_id=1&status=0&page=1&page_size=20"   -H "Authorization: Bearer $TOKEN"

curl -s http://localhost:8080/api/v1/tasks/1   -H "Authorization: Bearer $TOKEN"
```

`GET /api/v1/tasks` 支持 `job_id` 与 `status` 过滤；过滤字段走白名单，不接受任意列名。

## 8. 更新 Task 状态

```bash
curl -s -X PATCH http://localhost:8080/api/v1/tasks/1/status   -H "Authorization: Bearer $TOKEN"   -H "Content-Type: application/json"   -d '{"status":1}'
```

非法状态流转 → `409` + code `5001`。Status 取值语义见 `.agents/context/glossary.md`，以 `internal/model` 定义为准。

## 9. 删除 Job（仅 admin）

```bash
curl -s -X DELETE http://localhost:8080/api/v1/jobs/1   -H "Authorization: Bearer $ADMIN_TOKEN"
```

普通 `user` 调用 → `403` + code `3001`。角色只有 `user|admin` 两种。

## 10. 错误响应示例

```json
{"code":2001,"message":"未认证或 Token 无效","data":null,"request_id":"..."}
```

错误码对照见 `.agents/rules/api-contract.md`：`1001` 参数错误、`1002` 分页非法、`2001` 未认证、`3001` 权限不足、`4001` 资源不存在、`5001` 冲突、`6001` 限流、`9999` 内部错误。

## 附：完整端点

| 方法 | 路径 | 鉴权 |
| --- | --- | --- |
| POST | `/api/v1/register` | 公开 |
| POST | `/api/v1/login` | 公开 |
| GET | `/api/v1/me` | Bearer |
| POST | `/api/v1/jobs` | Bearer |
| GET | `/api/v1/jobs` | Bearer |
| GET | `/api/v1/jobs/:id` | Bearer |
| DELETE | `/api/v1/jobs/:id` | Bearer + admin |
| POST | `/api/v1/tasks` | Bearer |
| GET | `/api/v1/tasks` | Bearer |
| GET | `/api/v1/tasks/:id` | Bearer |
| PATCH | `/api/v1/tasks/:id/status` | Bearer |
| GET | `/health` | 公开 |
| GET | `/ready` | 公开 |
| GET | `/swagger/*` | 公开 |

配置与环境变量见 `docs/configuration.md`。
