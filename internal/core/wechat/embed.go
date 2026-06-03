package wechat

import "embed"

// builtinThemesFS 内嵌 5 个精选主题。这样 npm 分发的单 binary 自带主题，
// 无需额外资源文件。用户可在 ~/.thinkthinking/themes/ 或项目 .thinkthinking/themes/
// 放置同名 yaml 覆盖内置主题。
//
//go:embed themes/*.yaml
var builtinThemesFS embed.FS
