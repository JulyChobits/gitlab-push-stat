#!/bin/bash
# GitLab Push Stat 重启脚本 (Linux)

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

echo "正在重启 GitLab Push Stat..."
"$SCRIPT_DIR/stop.sh"
sleep 1
"$SCRIPT_DIR/start.sh"