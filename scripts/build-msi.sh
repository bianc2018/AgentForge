#!/usr/bin/env bash
# build-msi.sh — 从已编译的 Windows 二进制生成 MSI 安装包
#
# 用法：bash scripts/build-msi.sh <version>
# 示例：bash scripts/build-msi.sh v1.0.0
#
# 前置条件：
#   - wixl（sudo apt install wixl）
#   - msitools（sudo apt install msitools），提供 msibuild 用于 PATH 注册后处理
# 输入：dist/ 目录下已编译的 Windows 二进制（由 goreleaser 构建）
# 输出：dist/ 目录下的 .msi 安装包

set -euo pipefail

VERSION="${1:-unknown}"
WXS_TEMPLATE="builds/windows/installer.wxs"

# 去掉版本号前的 v 前缀（WiX 版本字段只接受数字和点）
CLEAN_VERSION="${VERSION#v}"

echo "==> 构建 MSI 安装包（版本: ${CLEAN_VERSION})"

# 检查 wixl 可用性
if ! command -v wixl &>/dev/null; then
  echo "⚠️  wixl 未安装，跳过 MSI 生成。"
  echo "   安装方法：sudo apt install wixl  (Debian/Ubuntu)"
  echo "            sudo dnf install msitools  (Fedora/RHEL)"
  exit 0
fi

# 检查模板文件
if [[ ! -f "$WXS_TEMPLATE" ]]; then
  echo "❌ MSI 模板文件不存在: $WXS_TEMPLATE"
  exit 1
fi

# 为每个 Windows 二进制生成 MSI
for BINARY in dist/agent-forge_windows_*/agent-forge.exe; do
  if [[ ! -f "$BINARY" ]]; then
    echo "⚠️  未找到 Windows 二进制文件，跳过 MSI 生成。"
    exit 0
  fi

  # 从路径推断架构：dist/agent-forge_windows_amd64_v1/agent-forge.exe → amd64
  ARCH_DIR=$(basename "$(dirname "$BINARY")")
  GOARCH=$(echo "$ARCH_DIR" | sed 's/agent-forge_windows_//' | sed 's/_v[0-9].*//')

  case "$GOARCH" in
    amd64|arm64)
      WIN64="yes"
      ;;
    *)
      echo "⚠️  未知架构: $GOARCH，跳过"
      continue
      ;;
  esac

  OUTPUT="dist/agent-forge_${CLEAN_VERSION}_windows_${GOARCH}.msi"
  WORK_WXS=$(mktemp /tmp/agent-forge-installer-XXXXXX.wxs)

  # 替换 wxs 模板中的占位符
  sed -e "s|__PRODUCT_NAME__|AgentForge|g" \
      -e "s|__PRODUCT_VERSION__|${CLEAN_VERSION}|g" \
      -e "s|__WIN64__|${WIN64}|g" \
      -e "s|__BINARY__|${BINARY}|g" \
      "$WXS_TEMPLATE" > "$WORK_WXS"

  echo "  → 生成 $OUTPUT ($GOARCH)"
  if wixl -o "$OUTPUT" "$WORK_WXS" 2>&1; then
    echo "  ✓ $OUTPUT 生成成功 ($(du -h "$OUTPUT" | cut -f1))"

    # 注入 Environment 表（PATH 注册）
    # wixl 不支持 <Environment> 元素，通过 msibuild 后处理写入 MSI 数据库
    if command -v msibuild &>/dev/null; then
      ENV_IDT=$(mktemp /tmp/agent-forge-env-XXXXXX.idt)
      printf "Environment\tName\tValue\tComponent_\n"  > "$ENV_IDT"
      printf "s72\tS255\tS255\ts72\n"                >> "$ENV_IDT"
      printf "Environment\tEnvironment\n"             >> "$ENV_IDT"
      printf "PathEnv\tPATH\t[INSTALLDIR]\tMainExecutable\n" >> "$ENV_IDT"
      if msibuild "$OUTPUT" -i "$ENV_IDT" 2>/dev/null; then
        echo "  ✓ PATH 注册已注入"
      else
        echo "  ⚠ PATH 注册注入失败"
      fi
      rm -f "$ENV_IDT"
    else
      echo "  ⚠ msitools 未安装，跳过 PATH 注册"
      echo "    安装: sudo apt install msitools  (Debian/Ubuntu)"
      echo "          sudo dnf install msitools   (Fedora/RHEL)"
    fi
  else
    echo "  ❌ MSI 生成失败"
    rm -f "$WORK_WXS"
    exit 1
  fi

  rm -f "$WORK_WXS"
done

echo "==> MSI 构建完成"
