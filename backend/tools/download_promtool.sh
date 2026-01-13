#!/usr/bin/env bash
set -euo pipefail

# 下载 promtool 或使用系统已安装的版本
# 用法：
#   cd backend
#   ./tools/download_promtool.sh
# 或使用已安装的 promtool（如果系统已有）：
#   which promtool
#   export PROMTOOL_PATH=$(which promtool)

DEST_DIR="$(cd "$(dirname "$0")" && pwd)"

# 先检查系统是否已有 promtool
if command -v promtool &> /dev/null; then
  echo "✓ 检测到系统已安装 promtool: $(which promtool)"
  echo "无需下载，直接使用系统版本"
  exit 0
fi

# 检查本地是否已有
if [ -f "${DEST_DIR}/promtool" ]; then
  echo "✓ 本地已存在 promtool: ${DEST_DIR}/promtool"
  exit 0
fi

# 如果都没有，提示用户手动安装或下载
echo "❌ 未找到 promtool，请手动安装或下载："
echo ""
echo "方式 1 - 使用包管理器安装（推荐）："
echo "  Ubuntu/Debian: sudo apt-get install prometheus-client"
echo "  macOS: brew install prometheus"
echo "  Fedora/RHEL: sudo dnf install prometheus"
echo ""
echo "方式 2 - 手动下载（国内镜像）："
echo "  访问 https://prometheus.ac.cn/download/ 下载最新版本"
echo "  解压后将 promtool 放到 ${DEST_DIR}/"
echo ""
echo "方式 3 - 设置环境变量指向已有 promtool："
echo "  export PROMTOOL_PATH=/path/to/promtool"
echo ""
exit 1

echo "Extracting promtool..."
tar -xzf "${TMP_DIR}/prometheus.tar.gz" -C "$TMP_DIR"
cp "${TMP_DIR}/prometheus-${VERSION}.${TARGET_OS}-${TARGET_ARCH}/promtool" "${DEST_DIR}/promtool"
chmod +x "${DEST_DIR}/promtool"

echo "promtool saved to ${DEST_DIR}/promtool"
