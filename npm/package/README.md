# @thinkthinking/cli

面向 **LLM Agent、自动化脚本和开发者** 的本地 CLI 工具箱。第一期聚焦微信公众号能力：将 Markdown 转换为微信公众号兼容 HTML，并上传草稿。所有命令默认输出统一 JSON envelope。

## 安装

```bash
npm install -g @thinkthinking/cli
```

本 npm 包只是 Go 二进制的分发壳：`postinstall` 会根据你的平台/架构从 [GitHub Releases](https://github.com/thinkthinking/cli/releases) 下载对应二进制，运行时**不依赖 Node**。

## 快速开始

```bash
thinkthinking init
thinkthinking config set wechat.app_id     wx_your_appid
thinkthinking config set wechat.app_secret your_appsecret
thinkthinking wechat convert --input article.md
thinkthinking wechat draft create --markdown-file article.md --title "标题"
```

## 环境变量

- `THINKTHINKING_SKIP_DOWNLOAD=1` 跳过二进制下载（离线 / 自行构建）
- `THINKTHINKING_VERSION=x.y.z` 指定下载版本

完整文档见 [GitHub 仓库](https://github.com/thinkthinking/cli)。

## License

MIT
