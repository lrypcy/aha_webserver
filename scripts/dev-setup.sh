#!/usr/bin/env bash
# 开发环境初始化：准备本地配置 + 提示设置环境变量
# 用法：./scripts/dev-setup.sh（或 bash scripts/dev-setup.sh）
# 首次使用请赋可执行权限：chmod +x scripts/dev-setup.sh
set -euo pipefail

cd "$(dirname "$0")/.."

# 1) 从示例配置生成本地默认配置；已存在时保留，避免覆盖你的改动
if [[ ! -f configs/config.yml ]]; then
  if [[ ! -f configs/config.example.yml ]]; then
    echo "错误：缺少 configs/config.example.yml，请检查仓库是否完整。" >&2
    exit 1
  fi
  cp configs/config.example.yml configs/config.yml
  echo "==> 已从 configs/config.example.yml 生成 configs/config.yml"
else
  echo "==> configs/config.yml 已存在，保留现有内容（如需重置请对照 config.example.yml）"
fi

# 2) 生成 .env（已加入 .gitignore，切勿提交），敏感值只放这里
if [[ ! -f .env ]]; then
  cp .env.example .env
  echo "==> 已从 .env.example 生成 .env"
else
  echo "==> .env 已存在，保留现有内容"
fi

echo ""
echo "下一步："
echo "  1) 生成强随机 JWT 密钥并填入 .env 的 AHA_JWT_SECRET（生产必填）："
echo "       openssl rand -hex 32"
echo "  2) 本地零配置启动（默认 sqlite、端口 8080）：make run"
echo "     如需加载 .env：set -a; source .env; set +a; make run"
echo "  3) 健康检查：curl http://localhost:8080/health"
echo "  4) 完整校验：./scripts/verify.sh（或 make test）"
echo ""
echo "提示：真实密钥不要写进 configs/ 或提交到 Git；.env 与 *.db 均已被忽略。"
