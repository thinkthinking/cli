# @thinkthinking/cli

面向 **LLM Agent、自动化脚本和开发者** 的本地 CLI 工具箱。第一期聚焦微信公众号能力：将 Markdown 转换为微信公众号兼容 HTML，并上传草稿。所有命令默认输出统一 JSON envelope。

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
thinkthinking wechat convert --input article.md
thinkthinking wechat draft create --markdown-file article.md --title "标题"
```

> 若在极少数环境下安装时带了 `--no-optional` / `--omit=optional`，平台子包会被跳过、命令会报「找不到平台二进制」。此时去掉该参数重装即可，或用 [GitHub Releases](https://github.com/thinkthinking/cli/releases) 的一键脚本安装。

完整文档见 [GitHub 仓库](https://github.com/thinkthinking/cli)。

## License

MIT
