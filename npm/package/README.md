# @thinkthinking/cli

面向 **LLM Agent、自动化脚本与开发者** 的本地 CLI 工具箱。第一期聚焦微信公众号：把 **Markdown 一键转成微信公众号兼容 HTML** 并直接**发布草稿**。所有命令输出统一 JSON envelope，便于 Claude Code、Codex 等 Agent 或脚本稳定解析。

## 安装

```bash
npm install -g @thinkthinking/cli
```

本 npm 包是 Go 二进制的分发壳：通过 `optionalDependencies` 把各平台二进制拆成独立子包（`@thinkthinking/cli-darwin-arm64` 等），npm 安装时**自动只下载匹配你系统的那一个**，装完即用——无 postinstall、无运行时下载，运行时**不依赖 Node**。

## 快速开始

```bash
thinkthinking init
thinkthinking config set wechat.app_id     wx_your_appid
thinkthinking config set wechat.app_secret your_appsecret

# Markdown → 微信公众号 HTML
thinkthinking wechat convert --input article.md

# 一步发布为草稿（自动转换 + 上传正文图片与封面）
thinkthinking wechat post \
  --markdown-file article.md \
  --title "标题" --author "你的名字" --cover cover.jpg
```

> 若在极少数环境下安装时带了 `--no-optional` / `--omit=optional`，平台子包会被跳过、命令会报「找不到平台二进制」。此时去掉该参数重装即可，或用 [GitHub Releases](https://github.com/thinkthinking/cli/releases) 的一键脚本安装。

完整文档见 [GitHub 仓库](https://github.com/thinkthinking/cli)。

## License

MIT
