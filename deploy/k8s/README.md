# deploy/k8s — k8s 清单快照

由 `easy_deploy render --target k8s --out deploy/k8s` 生成（`make deploy-render`），清单版本与仓库根目录 `deploy.yaml` 一致：namespace `aha`、`serviceType: NodePort`、`replicas: 2`、探针 `/health`、端口 `8080`。

| 文件 | 说明 |
| --- | --- |
| `00-namespace.yaml` | Namespace `aha`（kubectl apply 会先应用，文件名以 00 开头保证顺序） |
| `aha-webserver-deployment.yaml` | Deployment，含 liveness/readiness 探针与 resources |
| `aha-webserver-service.yaml` | NodePort Service，`port=8080`，`targetPort=8080` |
| `aha-webserver-secret.example.yaml` | **Secret 模板，只含占位值，供 GitOps 参考** |

## 密钥处理

`render` 还会生成 `aha-webserver-secret.yaml`（`stringData` 含展开后的真实环境变量值），该文件权限 0600 且被 `.gitignore` 的 `deploy/k8s/*-secret.yaml` 忽略，**绝不入库**。`easy_deploy up --target k8s` 会把它写到 `.easy_deploy/generated/k8s/` 并随其他清单一起 `kubectl apply -f`。

走 GitOps / ArgoCD 时不要直接 apply 这个快照，应先在集群侧创建 Secret：

```bash
kubectl -n aha create secret generic aha-webserver-env \
  --from-literal=AHA_SERVER_PORT=8080 \
  --from-literal=AHA_SERVER_ENV=production \
  --from-literal=AHA_JWT_SECRET="$AHA_JWT_SECRET" \
  --from-literal=AHA_DB_PASSWORD="$AHA_DB_PASSWORD"
kubectl apply -f deploy/k8s/00-namespace.yaml
kubectl apply -f deploy/k8s/aha-webserver-deployment.yaml
kubectl apply -f deploy/k8s/aha-webserver-service.yaml
```

生产环境请改用 Sealed Secrets / External Secrets / KMS，不要把明文 Secret 提交到仓库。完整说明见 [../../docs/deployment.md](../../docs/deployment.md)。
