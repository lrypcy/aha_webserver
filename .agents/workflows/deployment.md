# 工作流：部署 aha_webserver

## 阶段 → 产出 → 验证

| 阶段 | 动作 | 产出 | 验证 |
| --- | --- | --- | --- |
| 1 盘点 | 确认 `deploy.yaml` 端口/镜像/健康路径/环境变量与源码一致 | 事实表 | 与 `configs/config.yml`、`main.go`、Dockerfile 核对 |
| 2 校验 | `validate --json`、`plan --json` | 归一化 Spec + Step 列表 | `ok:true`，Step 序列符合预期 |
| 3 预检 | `doctor --target <t> --json` | 逐项检查 | 不可用目标明确记录，不硬上 |
| 4 生成清单 | `render --target k8s --out deploy/k8s` | k8s / compose 原生清单 | YAML 可解析、端口与探针一致；Secret 不入库 |
| 5 部署 | `up --target <t> --yes --json` | 运行中的服务 + state | `result.health == "ok"`，HTTP 2xx |
| 6 观测 | `status --json`、`logs --tail 100 --json` | 状态与日志 | 日志无敏感值回显 |
| 7 汇报 | 汇总 endpoint / pid / state / 清理命令 / 跳过目标 | 部署报告 | 明确区分「已验证运行」与「仅生成清单」 |

## 回滚

- `easy_deploy down --target <t> --yes --json`
- k8s：`kubectl -n aha rollout undo deployment/aha-webserver`
- 代码改动 `git restore <file>`；生成物在 `.easy_deploy/`，删除不影响源码。

## 完成定义

- 至少一个目标真实部署成功且健康端点通过。
- 六个目标的 `plan` 可执行；目标不具备时仅 `render` 且已标注。
- 文档四处同步：`AGENTS.md`、`docs/deployment.md`、`README.md`、`.agents/config.json`。
- 无明文密钥入库。
