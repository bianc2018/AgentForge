#!/usr/bin/env bash
# build-release.sh — AgentForge 一键编译发布脚本（Linux/macOS）
#
# 用法：
#   ./scripts/build-release.sh                  # snapshot 构建
#   ./scripts/build-release.sh --version v1.2.3  # 指定版本号
#   ./scripts/build-release.sh --release         # 正式发布模式（需 tag）
#   ./scripts/build-release.sh --help             # 显示帮助
#
# 前置条件：
#   - Go ≥1.21（go 在 PATH 中）
#   - goreleaser（go install github.com/goreleaser/goreleaser/v2@latest）

set -euo pipefail

# ─── 确保 Go 工具链在 PATH 中 ─────────────────────────────────────
# ~/go/bin 是 go install 的默认安装目标，/usr/local/go/bin 可能是手动安装的 Go
for dir in "$HOME/go/bin" "/usr/local/go/bin"; do
  if [[ -d "$dir" ]]; then
    case ":$PATH:" in
      *":$dir:"*) ;;
      *) export PATH="$dir:$PATH" ;;
    esac
  fi
done

# ─── 颜色输出 ───────────────────────────────────────────────────
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

info()  { printf "${GREEN}[INFO]${NC} %s\n" "$*"; }
warn()  { printf "${YELLOW}[WARN]${NC} %s\n" "$*" >&2; }
error() { printf "${RED}[ERROR]${NC} %s\n" "$*" >&2; }

# ─── 参数解析 ───────────────────────────────────────────────────
VERSION=""
RELEASE=false
SKIP_TESTS=""

print_help() {
  cat <<EOF
用法: $0 [选项]

选项:
  --version <ver>   指定版本号（如 v1.2.3），用于正式发布模式
  --release         正式发布模式（需 git tag），默认使用 snapshot
  --skip-tests      跳过单元测试
  --help            显示此帮助信息

示例:
  $0                         # snapshot 构建（日常开发）
  $0 --version v1.0.0        # 以 v1.0.0 版本 snapshot 构建
  $0 --release --version v1.0.0  # 正式发布 v1.0.0
EOF
  exit 0
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --version)
      VERSION="$2"
      shift 2
      ;;
    --release)
      RELEASE=true
      shift
      ;;
    --skip-tests)
      SKIP_TESTS="--skip-tests"
      shift
      ;;
    --help|-h)
      print_help
      ;;
    *)
      error "未知参数: $1"
      echo "使用 --help 查看可用选项"
      exit 2
      ;;
  esac
done

# ─── 环境校验 ───────────────────────────────────────────────────
info "校验构建环境..."

# 检查 Go
if ! command -v go &>/dev/null; then
  error "Go 未安装，请从 https://go.dev/dl/ 下载安装"
  exit 1
fi

GO_VERSION=$(go version | grep -oP 'go\K[0-9]+\.[0-9]+' | head -1)
GO_MAJOR=$(echo "$GO_VERSION" | cut -d. -f1)
GO_MINOR=$(echo "$GO_VERSION" | cut -d. -f2)

if [[ "$GO_MAJOR" -lt 1 ]] || [[ "$GO_MAJOR" -eq 1 && "$GO_MINOR" -lt 21 ]]; then
  error "Go 版本过低，需要 Go ≥1.21，当前版本: $(go version)"
  echo "  安装指引: https://go.dev/dl/"
  exit 1
fi
info "Go $(go version)"

# 检查 goreleaser
if ! command -v goreleaser &>/dev/null && [[ ! -x "$HOME/go/bin/goreleaser" ]]; then
  error "goreleaser 未安装"
  echo "  安装命令: go install github.com/goreleaser/goreleaser/v2@latest"
  echo "  安装后请确保 ~/go/bin 在 PATH 中: export PATH=\$HOME/go/bin:\$PATH"
  exit 1
fi
# 如果 command -v 找不到但 ~/go/bin/goreleaser 存在，补上 PATH
if ! command -v goreleaser &>/dev/null; then
  export PATH="$HOME/go/bin:$PATH"
fi
info "goreleaser $(goreleaser --version 2>&1 | head -1)"

# ─── 构建发布 ───────────────────────────────────────────────────
info "开始构建..."

RELEASE_FLAGS=()

if [[ "$RELEASE" == "true" ]]; then
  RELEASE_FLAGS+=("release")
  if [[ -n "$VERSION" ]]; then
    export GORELEASER_CURRENT_TAG="$VERSION"
    info "正式发布模式，版本: $VERSION"
  else
    info "正式发布模式（使用当前 git tag）"
  fi
else
  RELEASE_FLAGS+=("release" "--snapshot" "--clean")
  info "Snapshot 模式"
fi

if [[ -n "$SKIP_TESTS" ]]; then
  RELEASE_FLAGS+=("--skip=validate")
fi

goreleaser "${RELEASE_FLAGS[@]}"

info "构建完成，产物在 dist/ 目录"
ls -lh dist/ 2>/dev/null | grep -v "^total" | grep -v "^d" || true
