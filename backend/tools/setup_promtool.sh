#!/usr/bin/env bash
set -euo pipefail

# 设置 promtool：先检查 apt 已安装的，如果没有则下载，最后复制到 tools/promtool 目录
# 用法：
#   cd backend
#   bash ./tools/setup_promtool.sh

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
DEST_DIR="${SCRIPT_DIR}/promtool"
PROMTOOL_BIN=""

mkdir -p "$DEST_DIR"

echo "========================================="
echo "设置 Promtool"
echo "========================================="
echo ""

# Step 1: 检查系统是否已安装 promtool
echo "[1/3] 检查系统是否已安装 promtool..."
if command -v promtool &> /dev/null; then
  PROMTOOL_BIN=$(which promtool)
  echo "✓ 检测到系统已安装: $PROMTOOL_BIN"
else
  echo "✗ 系统未安装 promtool"
  
  # Step 2: 检查本地 tools/promtool 是否已存在
  if [ -f "${DEST_DIR}/promtool" ]; then
    echo "✓ 本地 ${DEST_DIR}/promtool 已存在，使用本地版本"
    PROMTOOL_BIN="${DEST_DIR}/promtool"
  else
    echo "✗ 本地版本不存在，需要下载"
    echo ""
    echo "[2/3] 使用 apt 下载 prometheus..."
    
    # 检查是否是 Ubuntu/Debian 系统
    if ! command -v apt-get &> /dev/null; then
      echo "❌ 错误：此脚本仅支持 apt 包管理器（Ubuntu/Debian）"
      echo "请手动运行："
      echo "  - macOS: brew install prometheus"
      echo "  - Fedora/RHEL: sudo dnf install prometheus"
      exit 1
    fi
    
    echo "更新包列表..."
    sudo apt-get update -qq
    
    echo "安装 prometheus..."
    if ! sudo apt-get install -y prometheus > /dev/null 2>&1; then
      echo "❌ apt 安装失败，请检查网络或手动运行: sudo apt-get install prometheus"
      exit 1
    fi
    
    echo "✓ prometheus 安装完成"
    
    # 验证安装
    if command -v promtool &> /dev/null; then
      PROMTOOL_BIN=$(which promtool)
      echo "✓ promtool 位置: $PROMTOOL_BIN"
    else
      echo "❌ 安装后仍未找到 promtool，请检查 prometheus 包"
      exit 1
    fi
  fi
fi

# Step 3: 复制 promtool 到 tools/promtool 目录
echo ""
echo "[3/3] 复制 promtool 到本地目录..."
if [ ! -f "$PROMTOOL_BIN" ]; then
  echo "❌ 错误：未找到 promtool: $PROMTOOL_BIN"
  exit 1
fi

cp "$PROMTOOL_BIN" "${DEST_DIR}/promtool"
chmod +x "${DEST_DIR}/promtool"
echo "✓ 已复制到 ${DEST_DIR}/promtool"

# 验证
echo ""
echo "========================================="
echo "✓ 设置完成"
echo "========================================="
echo ""
"${DEST_DIR}/promtool" --version
