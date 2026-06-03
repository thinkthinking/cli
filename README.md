# thinkthinking

面向 **LLM Agent、自动化脚本和开发者** 的本地 CLI 工具箱。

第一期聚焦微信公众号能力：将 Markdown 转换为微信公众号兼容 HTML，并上传草稿。所有命令默认输出统一 JSON envelope，便于 Claude Code、Codex 等 Agent 或脚本稳定解析。

> 转换逻辑移植自经过实战验证的 [wewrite](https://github.com/) pipeline：CJK 自动空格、列表转 section、外链转脚注、暗黑模式属性、`:::` 容器块语法、120 字节摘要等微信兼容修复一应俱全。

---

## 安装

### npm（推荐）

```bash
npm install -g @thinkthinking/cli
```

npm 包只是 Go 二进制的分发壳：`postinstall` 会根据你的平台/架构从 GitHub Releases 下载对应二进制，运行时**不依赖 Node**。

### 一键脚本

```bash
curl -fsSL https://raw.githubusercontent.com/thinkthinking/cli/master/scripts/install.sh | bash
```

### 从源码构建

```bash
git clone https://github.com/thinkthinking/cli.git
cd cli
make build      # 产物在 bin/thinkthinking
```

---

## 快速开始

```bash
# 1. 初始化用户配置 ~/.thinkthinking/config.yaml
thinkthinking init

# 2. 配置微信公众号凭证
thinkthinking config set wechat.app_id     wx_your_appid
thinkthinking config set wechat.app_secret your_appsecret

# 3. 把 Markdown 转成微信兼容 HTML
thinkthinking wechat convert --input article.md

# 4. 转换并上传草稿（自动转换 + 上传正文本地图片）
thinkthinking wechat draft create --markdown-file article.md --title "文章标题"
```

---

## 命令

| 命令 | 说明 |
|------|------|
| `thinkthinking version` | 输出版本信息 |
| `thinkthinking init [--local]` | 创建用户级（或项目级）配置 |
| `thinkthinking config path` | 输出配置文件路径 |
| `thinkthinking config get <key>` | 读取配置项 |
| `thinkthinking config set <key> <value>` | 设置配置项 |
| `thinkthinking config list` | 输出完整配置（敏感字段脱敏） |
| `thinkthinking wechat convert` | Markdown → 微信公众号 HTML |
| `thinkthinking wechat draft create` | 创建微信公众号草稿 |

### wechat convert

```bash
thinkthinking wechat convert --input article.md
thinkthinking wechat convert --input article.md --output dist/article.html
thinkthinking wechat convert --stdin < article.md
thinkthinking wechat convert --input article.md --theme midnight
```

内置主题：`default`、`minimal`、`midnight`、`newspaper`、`tech-modern`。
可在 `~/.thinkthinking/themes/` 或 `./.thinkthinking/themes/` 放置同名 YAML 覆盖内置主题。

支持的兼容增强（可用 flag 关闭）：

- `--no-darkmode` 关闭暗黑模式属性注入
- `--no-footnotes` 关闭外链转脚注
- `--no-containers` 关闭 `:::callout` / `:::timeline` / `:::dialogue` / `:::quote` / `:::highlight` / `:::summary` 容器块

### wechat draft create

```bash
# 从 Markdown（自动转换 + 上传正文本地图片）
thinkthinking wechat draft create --markdown-file article.md --title "标题"

# 从已有 HTML
thinkthinking wechat draft create --html-file article.html --title "标题"

# 指定作者与封面
thinkthinking wechat draft create --markdown-file article.md --title "标题" \
  --author "thinkthinking" --cover cover.jpg
```

- `--title` 必填；`--markdown-file` 与 `--html-file` 二选一
- 正文中的**本地图片**会自动上传到微信并回填 URL；远程 `http(s)` 图片跳过
- `--cover` 上传本地封面图；`--cover-media-id` 使用已有封面 media_id
- `--no-upload-images` 禁用正文图片自动上传

---

## JSON 输出规范

所有命令默认输出统一 envelope。**JSON 走 stdout，日志/警告走 stderr**，Agent 可稳定 parse stdout。

成功：

```json
{ "ok": true, "data": { }, "error": null }
```

失败：

```json
{
  "ok": false,
  "data": null,
  "error": { "code": "WECHAT_AUTH_ERROR", "message": "...", "details": { } }
}
```

错误码：`INVALID_INPUT` `CONFIG_ERROR` `FILE_NOT_FOUND` `MARKDOWN_CONVERT_ERROR` `WECHAT_AUTH_ERROR` `WECHAT_API_ERROR` `NETWORK_ERROR` `INTERNAL_ERROR`。

全局 flags：`--config` `--pretty` `--quiet` `--no-color` `--trace-id` `--verbose`。

---

## 配置

配置文件固定路径（跨平台一致）：

- 用户级：`~/.thinkthinking/config.yaml`
- 项目级：`./.thinkthinking/config.yaml`

优先级（高 → 低）：

```
CLI flags > 环境变量 > 项目配置 > 用户配置 > 默认值
```

配置示例：

```yaml
wechat:
  app_id: ""
  app_secret: ""
  access_token: ""
  default_author: "thinkthinking"
  default_theme: "default"

output:
  pretty: false
  quiet: false
```

### 环境变量

凭证可通过环境变量提供（优先级高于配置文件）：

| 环境变量 | 对应配置 |
|----------|----------|
| `WECHAT_APP_ID` | `wechat.app_id` |
| `WECHAT_APP_SECRET` | `wechat.app_secret` |
| `WECHAT_ACCESS_TOKEN` | `wechat.access_token` |
| `WECHAT_AUTHOR` | `wechat.default_author` |
| `WECHAT_THEME` | `wechat.default_theme` |

---

## npm 分发原理

`@thinkthinking/cli` 不用 Node 实现 CLI，只作为 Go 二进制的分发壳：

1. `npm install -g @thinkthinking/cli` 触发 `postinstall` → `install.js`
2. `install.js` 按 `process.platform` / `process.arch` 拼出 GitHub Releases 归档名并下载
3. 解压到 `bin/native/`，给 macOS/Linux 二进制加可执行权限
4. `bin/thinkthinking.js` 作为 wrapper，用 `spawnSync` 把参数原样转发给 native 二进制，保留 stdout/stderr 与 exit code

环境变量：

- `THINKTHINKING_SKIP_DOWNLOAD=1` 跳过下载（离线 / 自行构建）
- `THINKTHINKING_VERSION=x.y.z` 指定下载版本

---

## 架构

```
cmd/thinkthinking        # 入口
internal/
  cli/                   # 参数解析 → 调 service → 输出 JSON
  app/                   # Application Container（装配所有 service）
  core/
    wechat/              # 转换 pipeline、草稿服务、微信 client（含内嵌主题）
    output/              # 统一 JSON envelope
  config/                # 配置加载/读写（koanf 多源合并）
  logging/ storage/ server/ tui/ version/
```

核心能力沉淀在 `internal/core`，CLI 只是入口之一。未来的 TUI / GUI / Local API 复用同一套 core，不推倒重来。

---

## Roadmap

- [ ] TUI（Bubble Tea 交互式界面）
- [ ] GUI（通过 core service / local API）
- [ ] Local API server
- [ ] MCP server（让 Agent 直接调用 thinkthinking tools）
- [ ] 小绿书（图片消息）草稿
- [ ] 更多个人工具：文章处理、内容发布、文件转换、AI 辅助写作、知识库整理

---

## 开发

```bash
make build      # 构建
make test       # 测试
make lint       # gofmt 检查 + go vet
make fmt        # 格式化
make snapshot   # 本地试构建跨平台产物
```

## License

MIT
