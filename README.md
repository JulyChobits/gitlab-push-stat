# 🔍 GitLab Push Stat — 管理杀手锏，效率神器

> **"当老板问'你们团队最近干了啥'的时候，你终于可以甩出一份漂亮报表了。"**

![GitHub stars](https://img.shields.io/github/stars/YOUR_USERNAME/GitlabPushStat?style=social)
![GitHub forks](https://img.shields.io/github/forks/YOUR_USERNAME/GitlabPushStat?style=social)
![GitHub issues](https://img.shields.io/github/issues/YOUR_USERNAME/GitlabPushStat)
![License](https://img.shields.io/badge/license-MIT-blue.svg)
![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go&logoColor=white)
![Vue](https://img.shields.io/badge/Vue-3.x-4FC08D?logo=vue.js&logoColor=white)

---

## 🌟 如果这个项目帮你省下了一场无聊的汇报会议，请点亮右上角的 ⭐ Star

*（你的 Star 是我继续肝代码的动力，也是我向领导邀功的资本 😭）*

---

## 😤 你是不是也遇到过这些问题？

| 场景 | 痛苦程度 |
|------|---------|
| 领导问："小王，你们组上周写了多少行代码？" | 💀💀💀💀💀 |
| 周会上每个人说"我修了好多 bug"，但谁也说不清楚 | 💀💀💀💀 |
| 新人入职三个月，你根本不知道他的产出如何 | 💀💀💀💀💀 |
| 月底写绩效，只能凭"感觉"打分 | 💀💀💀💀💀💀 |
| 有人一天提交 50 次，全是改注释和空格 | 💀💀💀 |

**别慌，GitLab Push Stat 就是来拯救你的。** 🦸

---

## ✨ 它能做什么？

### 🤖 AI 智能审核 —— 让 AI 帮你写日报
接入大模型（支持 OpenAI 兼容 API），自动分析每天的代码提交，生成：
- **难度评估**：这段代码是"改了个文案"还是"重构了核心模块"
- **工时估算**：科学评估工作量，告别"感觉他今天很闲"
- **代码质量**：优秀 / 良好 / 一般 / 待改进，有据可查
- **亮点与建议**：AI 甚至能指出代码写得好的地方（比你的 Code Review 还细）

> 💡 *团队成员甚至不知道自己被"审"了。等他们发现的时候，你已经拿着报表在周会上侃侃而谈了。*

### 📊 多维报表 —— 日报 / 周报 / 月报自动生成
- **日报**：每天凌晨自动生成，固化保存，再也不怕"忘了写日报"
- **周报**：周一凌晨自动汇总，开会直接投屏
- **月报**：月度产出清晰可见，绩效评估有据可依

### 🔍 智能过滤 —— 虚假工作量，无所遁形
自动剔除那些"看起来很努力"的提交：
- `node_modules/`、`vendor/`、`dist/` 等依赖目录
- `.lock`、`.min.js`、`.map` 等自动生成文件
- `package-lock.json`、`go.sum` 等包管理文件
- `Merge branch`、`Auto merge` 等无意义提交

> 🎯 *终于没人能靠 `npm install` 凑代码量了。*

### 📈 可视化看板 —— 数据会说话
- **提交趋势**：团队是越来越卷，还是越来越躺，一图看懂
- **贡献排行榜**：谁是团队的真·大腿，一目了然
- **项目分布**：各项目投入的人力占比，资源分配有据可依

> ⚠️ *排行榜功能请谨慎使用，可能引发内卷。（但说实话，这就是它的价值）*

### ⏰ 定时任务 —— 全自动运转
| 任务 | 默认频率 | 说明 |
|------|---------|------|
| 数据同步 | 每 30 分钟 | 自动拉取 GitLab 最新提交 |
| 日报生成 | 每天凌晨 2:00 | AI 审核后自动生成 |
| 周报生成 | 每周一凌晨 3:00 | 自动汇总本周数据 |
| 月报生成 | 每月 1 号凌晨 4:00 | 月度产出报告 |

### 🖥️ Web 管理界面 —— 不用写一行配置
- 项目管理：添加/启用/禁用 GitLab 项目
- 过滤规则：可视化管理过滤策略
- 报表查看：日报/周报/月报在线浏览
- AI 审核：一键触发，结果实时展示

---

## 🛠️ 技术栈

| 后端 | 前端 |
|------|------|
| Go 1.21+ | Vue 3 + Vite |
| Gin (HTTP 框架) | Element Plus (UI 组件库) |
| GORM (ORM) | ECharts (图表可视化) |
| SQLite (嵌入式数据库) | Axios (HTTP 客户端) |
| robfig/cron (定时任务) | |
| Viper (配置管理) | |

> 🧩 *Go + Vue 双剑合璧，单二进制部署，不依赖任何外部数据库。领导问"运维成本高吗"的时候，你可以自信地说：几乎为零。*

---

## 🚀 开箱即用，秒出数据

### 第一步：编辑配置

编辑 `config/config.yaml`，填入你的 GitLab 信息和 AI 配置：

```yaml
gitlab:
  url: "https://your-gitlab-server.com"    # 你的 GitLab 地址
  token: "your-personal-access-token"       # Personal Access Token（需要 read_api 权限）

ai:
  enabled: true                             # 开启 AI 审核
  base_url: "https://api.openai.com/v1"     # 兼容 OpenAI 的 API 地址
  api_key: "your-api-key"                   # API Key
  model: "gpt-4o"                           # 模型名称
```

### 第二步：启动服务

下载发布包解压后，根据你的操作系统选择启动方式：

**🪟 Windows**

```bash
# 双击运行，或在命令行执行：
start.bat
```

**🐧 Linux**

```bash
# 启动（后台运行，日志写入 data/server.log）
bash start.sh

# 停止
bash stop.sh

# 重启
bash restart.sh
```

启动后访问 `http://localhost:8080` 即可进入管理界面。

### 第三步：坐等数据

数据自动拉取，报表自动生成。你可以先去喝杯咖啡 ☕。

> 💡 *也可以在 Web 界面手动点击"同步"按钮，立刻拉取数据。*

---

## 📂 项目结构

```
GitlabPushStat/
├── main.go                     # 程序入口
├── go.mod / go.sum             # Go 模块
├── config/
│   └── config.yaml             # 配置文件（核心）
├── internal/
│   ├── ai/                     # 🤖 AI 审核引擎
│   ├── collector/              # 📥 数据采集
│   ├── config/                 # ⚙️ 配置管理
│   ├── database/               # 🗄️ 数据库初始化
│   ├── filter/                 # 🔍 智能过滤引擎
│   ├── gitlab/                 # 🦊 GitLab API 客户端
│   ├── handler/                # 🌐 HTTP 处理器
│   ├── model/                  # 📋 数据模型
│   ├── report/                 # 📊 报表生成
│   └── scheduler/              # ⏰ 定时任务调度
├── web/
│   ├── src/                    # Vue 源码
│   └── dist/                   # 前端构建产物
├── data/                       # SQLite 数据库文件
└── deploy/                     # 部署包模板
```

---

## 📡 API 接口一览

| 方法 | 路径 | 说明 |
|------|------|------|
| `GET` | `/api/dashboard/summary` | 看板汇总数据 |
| `GET` | `/api/dashboard/trends` | 提交趋势数据 |
| `GET` | `/api/projects` | 项目列表 |
| `POST` | `/api/projects` | 添加项目 |
| `GET` | `/api/reports/daily` | 日报表 |
| `GET` | `/api/reports/weekly` | 周报表 |
| `GET` | `/api/reports/monthly` | 月报表 |
| `GET` | `/api/filters` | 过滤规则 |
| `POST` | `/api/tasks/sync` | 手动触发同步 |
| `POST` | `/api/tasks/ai-review` | 手动触发 AI 审核 |

---

## 🔧 配置说明

### 环境变量

| 变量名 | 默认值 | 说明 |
|--------|--------|------|
| `CONFIG_PATH` | `config/config.yaml` | 配置文件路径 |

### 过滤规则类型

| 类型 | 说明 | 示例 |
|------|------|------|
| 文件路径 | 正则匹配路径 | `vendor/.*`、`node_modules/.*` |
| 文件扩展名 | 过滤特定后缀 | `.min.js`、`.lock` |
| 文件名 | 精确匹配文件名 | `package-lock.json` |
| 提交信息 | 正则匹配 commit message | `^Merge branch` |

---

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

不管是修一个 bug、加一个功能、还是改一个错别字，都是对这个项目的贡献 ❤️

---

## ⭐ Star 一下，让更多人看到

如果你正在为团队管理的数据可视化发愁，或者你的领导又在催周报了——

```
  ____  _              _    ____  _
 / ___|| |_ __ _ _ __ | |  / ___|| |_ __ _ _ __
 \___ \| __/ _` | '_ \| |  \___ \| __/ _` | '_ \
  ___) | || (_| | |_) |_|  ___) | || (_| | |_) |
 |____/ \__\__,_| .__/(_)  |____/ \__\__,_| .__/
                 |_|                       |_|
```

**请点亮右上角的 ⭐ Star！**

- 你的每一个 Star，都让我觉得熬夜是值得的 🌙
- 你的每一个 Fork，都让我离升职加薪更近了一步（大概）
- 你的每一个 Issue，都是我发现 bug 的线索（求轻喷）

> 📢 *据统计，Star 过这个项目的人，团队代码质量都提升了 200%（数据来源：我编的）*

---

## 📜 License

[MIT License](LICENSE) — 随便用，不用谢。但 Star 一下总可以吧？😉

---

*Made with ❤️ and ☕ by a developer who was tired of writing daily reports manually.*