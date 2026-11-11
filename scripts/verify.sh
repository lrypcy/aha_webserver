#!/usr/bin/env bash
# 一键本地校验：gofmt 检查 -> go vet -> go test -> go build，任一步失败立即退出
# 用法：./scripts/verify.sh（或 bash scripts/verify.sh）
# 首次使用请赋可执行权限：chmod +x scripts/verify.sh
set -euo pipefail

cd "$(dirname "$0")/.."

echo "==> 1/4 检查 gofmt"
unformatted=$(gofmt -l .)
if [[ -n "${unformatted}" ]]; then
  echo "以下文件未通过 gofmt，请先执行 make fmt："
  echo "${unformatted}"
  exit 1
fi
echo "    gofmt OK"

echo "==> 2/4 go vet ./..."
go vet ./...

echo "==> 3/4 go test ./..."
go test ./...

echo "==> 4/4 go build ./..."
go build ./...

echo "==> 全部校验通过"
