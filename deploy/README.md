# GitLab Push Stat - 部署说明

## 目录结构

```
gitlab-push-stat/
├── gitlab-push-stat          # Linux 可执行文件
├── gitlab-push-stat.exe      # Windows 可执行文件
├── start.sh                  # Linux 启动脚本
├── stop.sh                   # Linux 停止脚本
├── start.bat                 # Windows 启动脚本
├── config/
│   └── config.yaml           # 配置文件（需修改）
├── data/                     # 数据库目录（自动创建）
└── web/                      # 前端资源（已嵌入二进制）
```

## 配置

编辑 `config/config.yaml`，主要需要修改的配置项：

```yaml
# GitLab 配置
gitlab:
  url: "http://your-gitlab-url"     # GitLab 服务器地址
  token: "your-access-token"        # GitLab Personal Access Token (需要 api 权限)

# 服务器配置
server:
  port: 8080                        # 服务端口
  mode: "release"                   # 运行模式

# 数据库配置
database:
  path: "./data/gitlab-stat.db"     # SQLite 数据库路径

# 定时任务配置
schedule:
  sync: "*/30 * * * *"              # 数据同步频率（默认每30分钟）
  daily: "0 2 * * *"                # 日报表生成时间（默认凌晨2点）
  weekly: "0 3 * * 1"               # 周报表生成时间（默认周一凌晨3点）
  monthly: "0 4 1 * *"              # 月报表生成时间（默认每月1号凌晨4点）

# AI 审核配置（可选）
ai:
  enabled: true                     # 是否启用 AI 审核
  api_base: "https://api.openai.com/v1"  # OpenAI API 地址
  api_key: "your-api-key"          # API Key
  model: "gpt-4"                    # 模型名称
```

## Linux 部署

### 1. 赋予执行权限
```bash
chmod +x gitlab-push-stat start.sh stop.sh
```

### 2. 修改配置
```bash
vim config/config.yaml
# 修改 GitLab 地址、Token 等配置
```

### 3. 启动服务
```bash
./start.sh
```

### 4. 查看日志
```bash
tail -f data/server.log
```

### 5. 停止服务
```bash
./stop.sh
```

### 6. 设置 systemd 自启动（可选）

创建 `/etc/systemd/system/gitlab-push-stat.service`：

```ini
[Unit]
Description=GitLab Push Stat
After=network.target

[Service]
Type=simple
WorkingDirectory=/path/to/gitlab-push-stat
ExecStart=/path/to/gitlab-push-stat/gitlab-push-stat
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

然后启用并启动：
```bash
sudo systemctl daemon-reload
sudo systemctl enable gitlab-push-stat
sudo systemctl start gitlab-push-stat
sudo systemctl status gitlab-push-stat
```

## Windows 部署

### 1. 修改配置
用文本编辑器打开 `config\config.yaml`，修改 GitLab 地址、Token 等配置。

### 2. 启动服务
双击 `start.bat` 或在命令行运行：
```cmd
gitlab-push-stat.exe
```

### 3. 注册为 Windows 服务（可选）

使用 [NSSM](https://nssm.cc/) 注册为 Windows 服务：
```cmd
nssm install GitLabPushStat "C:\path\to\gitlab-push-stat.exe"
nssm set GitLabPushStat AppDirectory "C:\path\to"
nssm start GitLabPushStat
```

## 访问

启动后访问：`http://服务器IP:8080`

## 功能说明

- **看板** - 总体统计、提交趋势、项目分布、贡献排行
- **日报表** - 按日查看每个人的提交详情、代码变更
- **周报/月报** - 周期性汇总报表
- **项目管理** - 管理监控的 GitLab 项目
- **过滤规则** - 配置提交过滤（如忽略 merge commit）
- **AI 审核** - AI 自动评估代码质量、难度和工时

## 注意事项

1. GitLab Token 需要 `api` 权限
2. 首次启动会全量同步所有项目的提交记录，可能需要较长时间
3. 数据库使用 SQLite，数据文件在 `data/gitlab-stat.db`
4. 建议定期备份 `data` 目录