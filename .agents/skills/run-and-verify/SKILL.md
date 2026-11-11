---
name: run-and-verify
description: "在本地启动 aha_webserver 并跑通冒烟验证：预先建库、生成文档、注册登录、Bearer 创建 Job/Task、健康检查与检查清单。"
---

# 技能：启动并验证

适用：改完代码后本地验证。开始前读 `AGENTS.md` 第 4、6 节与 `docs/configuration.md`。

## 步骤

1. **准备配置**：确认 `configs/config.yml`（非敏感默认值）与 `configs/config.example.yml` 存在；生产环境必须设 `AHA_JWT_SECRET`（空则拒绝启动）。环境变量前缀是 `AHA_`，`AHA_SERVER_PORT` 默认 `8080`。
2. **预先建库（mysql / postgresql 必做）**：数据库层已移除自动 `CREATE DATABASE`。用 `AHA_DB_NAME` 指定的名字先建库：
   ```bash
   # mysql
   mysql -h $AHA_DB_HOST -P $AHA_DB_PORT -u $AHA_DB_USER -p -e "CREATE DATABASE IF NOT EXISTS $AHA_DB_NAME;"
   # postgresql
   createdb -h $AHA_DB_HOST -p $AHA_DB_PORT -U $AHA_DB_USER $AHA_DB_NAME
   # sqlite（默认）：只需保证目录存在
   mkdir -p "$(dirname "$AHA_DB_NAME")"
   ```
   注意：这一步是**部署前置动作**，不要写回 `internal/database`。
3. **依赖与静态检查**：
   ```bash
   go mod download
   gofmt -w .
   go vet ./...
   go test ./...
   go build ./...
   ```
4. **确认 Swagger 生成物**：若 `api/` 为空或刚改过注解，跑 `swag init --output api --parseInternal --parseDependency`（或 `make swag`）。`main.go` 空导入了 `api` 包，生成物缺失会编译失败。
5. **启动服务**（默认 `8080`）：
   ```bash
   AHA_SERVER_PORT=8080 go run .
   ```
   用后台作业启动，确认进程在 `http://localhost:8080` 监听后再继续。
6. **冒烟 curl**（同一 shell 里先拿 Token）：
   ```bash
   curl -s http://localhost:8080/health
   curl -s http://localhost:8080/ready
   curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/swagger/index.html

   curl -s -X POST http://localhost:8080/api/v1/register      -H "Content-Type: application/json"      -d '{"username":"alice","email":"alice@example.com","password":"Passw0rd!"}'

   TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/login      -H "Content-Type: application/json"      -d '{"username":"alice","password":"Passw0rd!"}' | jq -r '.data.token')
   echo "$TOKEN"

   curl -s http://localhost:8080/api/v1/me -H "Authorization: Bearer $TOKEN"
   curl -s -X POST http://localhost:8080/api/v1/jobs      -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json"      -d '{"title":"demo-job"}'
   curl -s "http://localhost:8080/api/v1/jobs?page=1&page_size=20" -H "Authorization: Bearer $TOKEN"
   curl -s -X POST http://localhost:8080/api/v1/tasks      -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json"      -d '{"job_id":1,"title":"demo-task"}'
   curl -s "http://localhost:8080/api/v1/tasks?job_id=1&status=0&page=1&page_size=20" -H "Authorization: Bearer $TOKEN"
   curl -s -X PATCH http://localhost:8080/api/v1/tasks/1/status      -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -d '{"status":1}'
   ```
   若无 `jq`，手动复制 `/api/v1/login` 响应 `data.token` 的值填到 `$TOKEN`。
7. **核对响应形状**：每个 `/api/v1` 响应都应是 `{"code":0,"message":"ok","data":...,"request_id":"..."}`；确认 `request_id` 非空；无 Token 请求 `/api/v1/jobs` 应得 `401` + code `2001`；非 admin 调 `DELETE /api/v1/jobs/:id` 应得 `403` + code `3001`。
8. **停止服务**：结束后台作业，不要留残留进程。
9. **再次跑检查清单**：`go vet ./...` → `go test ./...` → `go build ./...`。

## 完成定义（DoD）

- [ ] mysql / postgresql 已预先建库；sqlite 目录存在；服务未自动建库
- [ ] `/health`、`/ready`、`/swagger/index.html` 均返回 `200`
- [ ] 注册 → 登录 → `Bearer` 创建 Job / Task → 列表 / 详情 / 状态更新 全链路通过
- [ ] 统一响应体与 `request_id` 正确，401 / 403 边界正确
- [ ] 四条检查命令全绿，后台进程已停止
