# aha_webserver 多阶段构建
#
# CGO 与 SQLite 的取舍：
#   gorm.io/driver/sqlite 依赖 github.com/mattn/go-sqlite3，该驱动通过 cgo 把 SQLite 的 C 源码编进二进制，
#   因此不能设置 CGO_ENABLED=0（否则编译失败，或运行期报 "Binary was compiled with CGO_ENABLED=0"）。
#   构建阶段选用 golang:1.23-alpine 并安装 gcc/musl-dev 提供 C 工具链；也可换成官方 golang:1.23（Debian 系自带 gcc）。
#   运行阶段用 alpine:3.20 + ca-certificates + tzdata，体积小且与 musl 动态链接兼容；以非 root 用户运行。

# ---------- 可覆盖的构建期参数 ----------
# 默认值保持官方源；受限网络下由 deploy.yaml 的 build.args 注入镜像源 / Go 模块代理，
# 例如 --set 或环境变量 EDP_GO_BASE_IMAGE / EDP_RUN_BASE_IMAGE / EDP_GOPROXY。
ARG GO_BASE_IMAGE=golang:1.23-alpine
ARG RUN_BASE_IMAGE=alpine:3.20
ARG GOPROXY=https://proxy.golang.org,direct

# ---------- 构建阶段 ----------
FROM ${GO_BASE_IMAGE} AS builder
ARG GOPROXY
ENV GOPROXY=${GOPROXY}

RUN apk add --no-cache gcc musl-dev

WORKDIR /src

# 先复制依赖清单，充分利用 Docker 层缓存
COPY go.mod go.sum ./
RUN go mod download

# 再复制源码；api/ 为 swag 生成物且被 main.go 导入，必须一并复制
COPY . .

ENV CGO_ENABLED=1 GOOS=linux
RUN mkdir -p /out && go build -trimpath -ldflags="-s -w" -o /out/aha_webserver .

# ---------- 运行阶段 ----------
FROM ${RUN_BASE_IMAGE}

RUN apk add --no-cache ca-certificates tzdata && addgroup -S app && adduser -S -G app app

WORKDIR /app
COPY --from=builder /out/aha_webserver /app/aha_webserver
# 非敏感默认配置随镜像提供；运行时用 AHA_* 环境变量覆盖
COPY configs/ /app/configs/
RUN mkdir -p /app/data && chown -R app:app /app

USER app
EXPOSE 8080

# /health 是非版本化存活探针，供 Docker 与 compose 复用
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 CMD wget -qO- http://127.0.0.1:8080/health >/dev/null || exit 1

ENTRYPOINT ["/app/aha_webserver"]
