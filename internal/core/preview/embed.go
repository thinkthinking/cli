package preview

import "embed"

// templateFS 内嵌浏览器预览页模板。仿 wechat/embed.go 的内嵌主题做法，
// 让单 binary 自带预览页资源，npm 分发无需额外文件。
//
//go:embed templates/preview.html
var templateFS embed.FS
