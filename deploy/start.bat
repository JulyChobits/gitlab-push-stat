@echo off
chcp 65001 >nul 2>&1
title GitLab Push Stat

cd /d "%~dp0"

if not exist "data" mkdir data
if not exist "config" mkdir config

if not exist "config\config.yaml" (
    echo [错误] 配置文件 config\config.yaml 不存在
    echo 请修改 config\config.yaml 中的 GitLab 地址、Token 等配置
    pause
    exit /b 1
)

echo 正在启动 GitLab Push Stat...
echo 按 Ctrl+C 停止服务
echo.

gitlab-push-stat.exe

pause