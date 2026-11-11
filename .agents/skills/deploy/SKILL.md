# 技能：用 easy_deploy 部署 aha_webserver

**触发**：要启动 / 更新 / 停止本服务，或校验部署资产、排查部署失败。

## 前置

读 `AGENTS.md` 第 6 节、`.agents/rules/deployment.md`、`docs/deployment.md`；确认 `../easy_deploy/bin/easy_deploy` 已构建。

## 标准步骤

1. **校验**：`easy_deploy validate --json` → `ok:true`；否则按 `error.code=SPEC_INVALID` 修 `deploy.yaml`。
2. **预检**：`easy_deploy doctor --target <t> --json`。
   - 失败（`PRECONDITION_MISSING`）→ 不硬上：换可用 target，或只执行 `render` 产出清单，并在汇报中标注「未实际部署」。
3. **预演**：`easy_deploy plan --target <t> --json`，检查 Step 序列（build → write env → start → http_check），确认没有意外 destructive Step。
4. **部署**：`easy_deploy up --target <t> --yes --json`。
5. **验证**：`easy_deploy status --json` 的 `result.health == "ok"`，并手工 `curl -fsS http://127.0.0.1:8080/health`。
6. **清理**：`easy_deploy down --target <t> --yes --json`。

## 按退出码分流

| 退出码 | 动作 |
| --- | --- |
| 2 | `validate --json` 读 message，改 `deploy.yaml` 未知字段 / 未解析 `${VAR}` |
| 3 | `doctor --json` 看 fail 项；常见是 8080 被占、systemd/ssh 不可用。注意本机 docker daemon 与 minikube 均已启动，`doctor --target docker\|compose\|k8s` 均通过；若 `kubectl cluster-info` 失败，检查旧 profile 的 kubelet 证书是否过期（见 `../easy_deploy/docs/driver-guide.md` §9） |
| 4 | 读 `error.step`；`build-aha-webserver` 失败多为 CGO / sqlite 缺 C 工具链；容器目标还可能是 registry 鉴权不通（`build.args` 换源已适配）、Docker VM 只读（见 `docs/deployment.md` §8）或 8080 端口被另一目标占用（先 `down` 旧目标） |
| 5 | `logs --json`；确认工作目录是仓库根（要读 `configs/config.yml`）、生产模式密钥是否齐全 |
| 6 | `down --yes` 清理残留 state 后重试 |
| 7 | 向用户复述 `result.resources` 影响范围并请求确认，补 `--yes` |

## 改端口 / 健康路径 / 镜像后

1. 改 `deploy.yaml`；
2. 同步 `Dockerfile`（EXPOSE/HEALTHCHECK）、`docker-compose.yml`、`.env.example`、`docs/deployment.md`、`AGENTS.md` 第 6 节、`.agents/config.json`；
3. `easy_deploy render --target k8s --out deploy/k8s` 刷新快照（Secret 文件不入库）；
4. 六个目标都跑一次 `plan --json` 确认无回归。

## 完成定义

- 至少一个目标真实部署且 `/health` 返回 2xx；其余目标有 `plan` + `render` 产物。
- `status --json` 的 health 为 `ok`；清理命令与 state 位置已写进交付说明。
- 无明文密钥入库；所有改动文件与验证证据可核对。
