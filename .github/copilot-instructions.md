# GitHub Copilot 指引（薄封装）

请遵循项目根目录 `../AGENTS.md` 中的全部规范：技术栈与版本、目录地图、Go 编码规范、API 契约、数据库与迁移规范、安全红线、新增接口 / 修改模型的标准步骤、提交前检查清单、常见坑。

补充说明：

- 规则细则在 `.agents/rules/`，可执行技能在 `.agents/skills/`，工作流在 `.agents/workflows/`，背景知识在 `.agents/context/`，索引见 `.agents/README.md`。
- 统一响应体 `{"code":0,"message":"ok","data":...,"request_id":"..."}` 与 `/api/v1` 前缀是硬契约，不要生成 `fiber.Map{"error": ...}` 这类临时响应。
- 部署事实源是根目录 `deploy.yaml`（easy_deploy，六个目标）；改端口 / 健康路径 / 镜像必须同步 Dockerfile、docker-compose.yml、`docs/deployment.md` 与 `deploy/k8s/`；密钥只走 `${VAR}`。
- 提交前必须 `gofmt -w .` / `go vet ./...` / `go test ./...` / `go build ./...` 全绿。

本文件只指向 `AGENTS.md`，避免规则多处维护产生分歧。
