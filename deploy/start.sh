#!/bin/bash
# GitLab Push Stat 启动脚本 (Linux)

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$SCRIPT_DIR"

BINARY="./gitlab-push-stat"
PID_FILE="./data/server.pid"
LOG_FILE="./data/server.log"

# 确保目录存在
mkdir -p ./data
mkdir -p ./config
mkdir -p ./web

# 检查是否已在运行
if [ -f "$PID_FILE" ]; then
    OLD_PID=$(cat "$PID_FILE")
    if kill -0 "$OLD_PID" 2>/dev/null; then
        echo "服务已在运行 (PID: $OLD_PID)"
        exit 1
    fi
fi

# 检查配置文件
if [ ! -f "./config/config.yaml" ]; then
    echo "错误: 配置文件 ./config/config.yaml 不存在"
    echo "请从模板复制并修改: cp config/config.yaml.example config/config.yaml"
    exit 1
fi

# 启动服务
echo "启动 GitLab Push Stat..."
nohup "$BINARY" >> "$LOG_FILE" 2>&1 &
PID=$!
echo "$PID" > "$PID_FILE"
echo "服务已启动 (PID: $PID)"
echo "日志文件: $LOG_FILE"
echo "访问地址: http://localhost:8080"