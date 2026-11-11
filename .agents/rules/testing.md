# 规则：测试

> 细则，权威规范见 `AGENTS.md` 第 11 节。CI 见 `.github/workflows/ci.yml`。

## 1. 命令

```bash
go test ./...                 # 必跑
go test -race ./...           # 改了并发 / 全局状态时加跑
go vet ./...                  # 必跑（能抓 %w 误用）
gofmt -l .                    # 必须无输出
go build ./...                # 必跑
```

## 2. 文件与命名

- 测试文件放被测包同目录，命名 `<name>_test.go`，`package` 与被测包一致（需要黑盒时用 `package xxx_test`）。
- 测试函数 `TestXxx`，表驱动优先，子测试用 `t.Run(c.name, ...)`。

```go
func TestCreateJob(t *testing.T) {
    cases := []struct{
        name string
        body string
        wantStatus int
        wantCode int
    }{
        {"正常创建", `{"title":"demo"}`, fiber.StatusCreated, 0},
        {"缺少必填字段", `{}`, fiber.StatusBadRequest, 1001},
    }
    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            // ...
        })
    }
}
```

- 表驱动用例里的 JSON 请求体用 Go 反引号原始字符串，避免转义；字段校验规则与 DTO 的 «B»validate«B» 标签保持一致。

## 3. Fiber handler 测试

用 `app.Test`，不要起真实端口：

```go
app := fiber.New()
// 按 internal/app 的装配方式注册被测路由
req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs", strings.NewReader(body))
req.Header.Set("Content-Type", "application/json")
req.Header.Set("Authorization", "Bearer "+token)
resp, err := app.Test(req, -1)
// 断言 resp.StatusCode 与响应体 code / data / request_id
```

- 断言三件事：HTTP 状态码、响应体 `code`、响应体 `request_id` 非空。
- 数据库用临时 sqlite 文件或 `sqlmock`，每个用例独立，不复用全局 `*gorm.DB`。
- 测试里不要写真实 `AHA_JWT_SECRET`，用测试专用常量并在 `t.Cleanup` 恢复环境。

## 4. 必须覆盖的场景

| 场景 | 断言 |
| --- | --- |
| 每个写接口 | 正常 `201` + code `0`；缺字段 `400` + code `1001` |
| 每个查详情 | `200`；`gorm.ErrRecordNotFound` → `404` + code `4001` |
| 每个列表 | `page` / `page_size` 默认值、越界 `400` + code `1002`、`total` 正确 |
| 鉴权 | 无 `Authorization` → `401` + code `2001`；无效 / 过期 Token → `401` |
| 角色 | 非 admin 调 `DELETE /api/v1/jobs/:id` → `403` + code `3001` |
| 限流 | 超 `AHA_RATE_LIMIT_MAX` → `429` + code `6001`（可用独立低阈值配置） |
| 健康 | `GET /health` → `200`；`GET /ready` 在 DB 不可用时非 `200` |
| 模型 | `serializer:json` 字段往返读写一致；三种 `AHA_DB_TYPE` 均能 AutoMigrate |

## 5. 迁移与 Sqlite 注意

- mysql / postgresql 的集成测试**必须预先建库**（数据库层已移除自动 `CREATE DATABASE`），否则连接直接失败。
- sqlite 测试文件路径放 `t.TempDir()`，避免污染工作区。

## 6. 自检清单

- [ ] `go test ./...` 通过，无 `-short` 跳过关键用例
- [ ] 新增接口有正常 + 参数错误 + 鉴权失败三类用例
- [ ] 改了模型有往返读写测试，且三种数据库类型都考虑过
- [ ] `go vet ./...` / `gofmt -l .` / `go build ./...` 全绿
