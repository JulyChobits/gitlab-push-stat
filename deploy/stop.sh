多方-http#!/bin/bash
# GitLab Push Stat 停止脚本 (Linux)

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$SCRIPT_DIR"

PID_FILE="./data/server.pid"

if [ ! -f "$PID_FILE" ]; then
    echo "服务未在运行 (PID文件不存在)"
    exit 0
fi

PID=$(cat "$PID_FILE")

if kill -0 "$PID" 2>/dev/null; then
    echo "正在停止服务 (PID: $PID)..."
    kill "$PID"
    sleep 2
    if kill -0 "$PID" 2>/dev/null; then
        echo "服务未响应，强制终止..."
        kill -9 "$PID"
    fi
    echo "服务已停止"
else
    echo "服务未在运行 (PID: $PID 已不存在)"
fi

rm -f "$PID_FILE"