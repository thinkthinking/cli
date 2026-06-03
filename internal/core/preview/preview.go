// Package preview 把转换后的微信 HTML 包成一个自包含的浏览器预览页，
// 并提供在默认浏览器中打开的能力。
//
// 存在意义：终端直接复制 HTML 文本无法携带 text/html flavor，粘贴进微信会
// 变成源码。预览页里的「复制到公众号」按钮用浏览器原生 ClipboardItem 写入
// text/html flavor（与 doocs/md 同款、已被微信验证），用户可肉眼校对排版后
// 一键复制粘贴——这是跨平台最稳的投递方式。
package preview

import (
	"bytes"
	"fmt"
	"html/template"
	"os/exec"
	"runtime"
)

// Renderer 生成预览页 HTML 并能在浏览器打开。
type Renderer interface {
	// Render 把正文 HTML（body 片段）包成一个完整的预览页 HTML 文档。
	// title 用于页面标题与工具条展示。
	Render(body, title string) (string, error)
	// Open 在默认浏览器打开给定文件/URL。best-effort：返回 error 不代表预览页
	// 不可用（文件已生成），调用方可据此提示用户手动打开。
	Open(target string) error
}

// New 返回默认 Renderer。
func New() Renderer {
	return &renderer{}
}

type renderer struct{}

// templateData 是预览页模板的数据。Body 用 template.HTML 以原样注入受信任的
// 转换结果（不转义）；Title 走默认转义，防止标题里的特殊字符破坏页面。
type templateData struct {
	Title string
	Body  template.HTML
}

func (renderer) Render(body, title string) (string, error) {
	raw, err := templateFS.ReadFile("templates/preview.html")
	if err != nil {
		return "", fmt.Errorf("read preview template: %w", err)
	}
	tmpl, err := template.New("preview").Parse(string(raw))
	if err != nil {
		return "", fmt.Errorf("parse preview template: %w", err)
	}

	if title == "" {
		title = "未命名"
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, templateData{
		Title: title,
		Body:  template.HTML(body), //nolint:gosec // body 是本工具转换的受信任 HTML
	}); err != nil {
		return "", fmt.Errorf("render preview template: %w", err)
	}
	return buf.String(), nil
}

func (renderer) Open(target string) error {
	name, args := openCommand(target)
	if name == "" {
		return fmt.Errorf("open browser: unsupported platform %s", runtime.GOOS)
	}
	if err := exec.Command(name, args...).Start(); err != nil {
		return fmt.Errorf("open browser via %s: %w", name, err)
	}
	return nil
}

// openCommand 按平台返回打开浏览器的命令与参数。空 name 表示不支持。
func openCommand(target string) (string, []string) {
	switch runtime.GOOS {
	case "darwin":
		return "open", []string{target}
	case "linux":
		return "xdg-open", []string{target}
	case "windows":
		// start 是 cmd 内建命令；空字符串是 start 的「窗口标题」占位参数。
		return "cmd", []string{"/c", "start", "", target}
	default:
		return "", nil
	}
}
