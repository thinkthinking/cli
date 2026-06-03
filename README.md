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

npm 包是 Go 二进制的分发壳：通过 `optionalDependencies` 把各平台二进制拆成独立子包，npm 安装时**自动只下载匹配你系统的那一个**，装完即用——无 postinstall、无运行时下载，运行时**不依赖 Node**。

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

## 微信公众号配置流程

在使用 `wechat` 命令之前，需要先在微信公众平台获取凭证并配置到本地。

### 1. 获取 AppID 与 AppSecret

1. 登录 [微信公众平台](https://mp.weixin.qq.com/)，进入「设置与开发 → 基本配置」。
2. 获取或重置 `AppID`、`AppSecret`（**AppSecret 仅展示一次，请立即保存**）。
3. 在「基本配置」中配置 **IP 白名单**：将调用机器的公网出口 IP 加入白名单，否则 access_token 和后续接口会被拒绝。
4. 进入「设置与开发 → 接口权限」，确认以下接口可用：
   - 素材管理（上传/下载永久素材）
   - 草稿箱（新增/修改/查询草稿）
   - 发布能力（提交发布、查询发布状态）

### 2. 配置本地凭证

**方式一：通过 CLI 写入配置文件**

```bash
thinkthinking init
thinkthinking config set wechat.app_id     wx_your_appid
thinkthinking config set wechat.app_secret your_appsecret
thinkthinking config list                   # 验证（敏感字段已脱敏）
```

**方式二：环境变量（适合 CI/CD）**

```bash
export WECHAT_APP_ID=wx_your_appid
export WECHAT_APP_SECRET=your_appsecret
```

### 3. 验证配置

```bash
# 查看当前配置（敏感字段脱敏）
thinkthinking config list

# 查看配置文件路径与加载状态
thinkthinking config path
```

`wechat draft create` 执行前会自动进行凭证预检，缺失时返回结构化 `WECHAT_AUTH_ERROR` 并附带配置路径与环境变量提示，无需手动验证。

### 4. 发布流水线

一条完整的发布路径：

1. **上传素材** — `wechat draft create` 自动将正文本地图片与封面图上传到微信
2. **创建草稿** — 同上命令，将转换后的 HTML 写入草稿箱
3. **提交发布** — （待实现，目前需在微信公众平台网页端操作）
4. **查询状态** — （待实现）
5. **回填文章链接** — （待实现）

> **参考链接**
> - [微信公众平台](https://mp.weixin.qq.com/)
> - [微信公众号开发概述](https://developers.weixin.qq.com/doc/offiaccount/Getting_Started/Overview.html)
> - [接入指南](https://developers.weixin.qq.com/doc/offiaccount/Basic_Information/Access_Overview.html)
> - [获取 access_token](https://developers.weixin.qq.com/doc/offiaccount/Basic_Information/Get_access_token.html)
> - [新增草稿](https://developers.weixin.qq.com/doc/offiaccount/Draft_Box/Add_draft.html)
> - [发布接口](https://developers.weixin.qq.com/doc/offiaccount/Publish/Publish.html)
> - [微信公众号 API 整理文档](https://weixingongzhonghao.com.cn/zh/api/)

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
| `thinkthinking init [--local] [--force]` | 创建用户级（或项目级）配置 |
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

### 如何把内容正确放进公众号编辑器

> ⚠️ **不要把 `convert` 打印的 HTML 文本直接复制粘贴进公众号**——会显示成 HTML 源码，而非排版。

原因：微信编辑器只有在系统剪贴板携带 **`text/html` 富文本类型**时才会渲染粘贴内容。终端里 `convert | pbcopy` 或从 `.html` 文件复制，剪贴板只有纯文本（`text/plain`），微信便把标签当字面量插入。转换出的 HTML 本身是正确的、微信兼容的——问题只在「怎么投递」。三种正确方式：

| 方式 | 命令 | 适用场景 |
|------|------|----------|
| **剪贴板**（仅 macOS） | `convert --copy` | 终端最快：写入富文本剪贴板，到公众号 `Cmd+V` 直接渲染 |
| **浏览器预览页**（全平台） | `convert --preview` | 最稳：打开预览页肉眼校对排版，点「复制到公众号」按钮再粘贴 |
| **草稿 API**（推荐给 Agent） | `wechat draft create` | 全自动：直接把正文写进公众号草稿箱，无需剪贴板，见下节 |

```bash
# macOS：转换并写入剪贴板，然后去公众号 Cmd+V
thinkthinking wechat convert --input article.md --copy

# 任意平台：生成预览页并打开浏览器，页面内一键复制
thinkthinking wechat convert --input article.md --preview
```

- `--copy` 成功后 JSON 含 `"copied": true`；非 macOS 返回 `PLATFORM_NOT_SUPPORTED`，请改用 `--preview` 或 `--output`。
- `--preview` 成功后 JSON 含 `preview_path`（预览文件路径）与 `opened`（是否成功唤起浏览器；为 `false` 时按提示手动打开）。
- `--copy` / `--preview` / `--output` 可叠加使用。

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

错误码：`INVALID_INPUT` `CONFIG_ERROR` `FILE_NOT_FOUND` `MARKDOWN_CONVERT_ERROR` `WECHAT_AUTH_ERROR` `WECHAT_API_ERROR` `NETWORK_ERROR` `PLATFORM_NOT_SUPPORTED` `INTERNAL_ERROR`。

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

`@thinkthinking/cli` 不用 Node 实现 CLI，只作为 Go 二进制的分发壳。采用业界标准的 **`optionalDependencies` + 按平台拆分子包** 模式（esbuild / @openai/codex / @anthropic-ai/claude-code 同款）：

- 主包 `@thinkthinking/cli`（壳）的 `optionalDependencies` 声明 5 个平台子包：`@thinkthinking/cli-{darwin-arm64,darwin-x64,linux-x64,linux-arm64,win32-x64}`。
- 每个子包用 npm 的 `os` / `cpu` 字段限定平台，二进制直接打在子包里随 npm registry 分发。`npm install` 时 npm **自动只安装匹配当前系统的那一个子包**，无需 postinstall、无运行时下载。
- `bin/thinkthinking.js` 作为 wrapper，用 `require.resolve('@thinkthinking/cli-<platform>-<arch>/thinkthinking')` 定位子包二进制，再用 `spawnSync` 把参数原样转发，保留 stdout/stderr 与 exit code。

发布流程：`npm/scripts/build-packages.mjs` 从 GitHub Release 下载各平台二进制、组装出 5 个子包并同步主包版本号；CI（`.github/workflows/release.yml`）先发 5 个子包、再发主包（OIDC Trusted Publishing，无需 token）。

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
