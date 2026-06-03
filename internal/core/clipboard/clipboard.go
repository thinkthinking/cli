// Package clipboard 把 HTML 以富文本（text/html flavor）写入系统剪贴板。
//
// 存在意义：微信公众号编辑器是 contenteditable，仅当剪贴板携带 text/html
// flavor 时才会把粘贴内容渲染成排版；只有 text/plain 时会把标签当字面量插入
// （表现为「粘贴出来全是 HTML 源码」）。终端的 `convert | pbcopy` 只能写
// text/plain，因此必须由本包显式写入 html flavor。
//
// 第一期仅实现 macOS（用户环境）。其它平台返回 ErrUnsupported，由 CLI 层
// 翻译成结构化错误并引导用户改用 --preview / --output。
package clipboard

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

// ErrUnsupported 表示当前操作系统暂不支持富文本剪贴板写入。
var ErrUnsupported = errors.New("clipboard: rich-text copy not supported on this platform")

// Writer 把 HTML 写入系统剪贴板。html 用于 text/html flavor（微信据此渲染），
// plain 用于 text/plain flavor（粘贴到纯文本环境时的回退）。
type Writer interface {
	WriteHTML(ctx context.Context, html, plain string) error
}

// New 按当前平台返回合适的 Writer。非 macOS 返回的实现其 WriteHTML 恒为
// ErrUnsupported。
func New() Writer {
	if runtime.GOOS == "darwin" {
		return &macWriter{}
	}
	return &unsupportedWriter{}
}

// unsupportedWriter 是非 macOS 平台的占位实现。
type unsupportedWriter struct{}

func (unsupportedWriter) WriteHTML(context.Context, string, string) error {
	return ErrUnsupported
}

// jxaScript 是写剪贴板的 JavaScript for Automation 脚本。
//
// 走 NSPasteboard 而非 pbcopy：pbcopy 只能写 text/plain。html/plain 内容经
// 临时文件 + 环境变量传入（见 macWriter.WriteHTML），脚本本身不含任何用户数据，
// 因此无注入风险，也规避了 ARG_MAX 与命令行转义问题。
const jxaScript = `ObjC.import('AppKit'); ObjC.import('Foundation');
var env = $.NSProcessInfo.processInfo.environment;
var html = $.NSString.stringWithContentsOfFileEncodingError(env.objectForKey('TT_HTML_FILE').js, $.NSUTF8StringEncoding, null);
if (!html) { throw new Error('cannot read html file'); }
var plain = $.NSString.stringWithContentsOfFileEncodingError(env.objectForKey('TT_PLAIN_FILE').js, $.NSUTF8StringEncoding, null);
var pb = $.NSPasteboard.generalPasteboard;
pb.clearContents;
var ok = pb.setStringForType(html, 'public.html');
if (plain) { pb.setStringForType(plain, 'public.utf8-plain-text'); }
if (!ok) { throw new Error('failed to set public.html flavor'); }`

// macWriter 通过 osascript(JXA) + NSPasteboard 写入 public.html flavor。
type macWriter struct{}

func (macWriter) WriteHTML(ctx context.Context, html, plain string) error {
	htmlFile, err := writeTempFile("thinkthinking-clip-*.html", html)
	if err != nil {
		return err
	}
	defer os.Remove(htmlFile)

	plainFile, err := writeTempFile("thinkthinking-clip-*.txt", plain)
	if err != nil {
		return err
	}
	defer os.Remove(plainFile)

	cmd := exec.CommandContext(ctx, "osascript", "-l", "JavaScript", "-e", jxaScript)
	cmd.Env = append(os.Environ(),
		"TT_HTML_FILE="+htmlFile,
		"TT_PLAIN_FILE="+plainFile,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		// osascript 失败时 stderr 带原因，一并带出便于诊断。
		msg := string(out)
		if msg == "" {
			return fmt.Errorf("osascript clipboard write: %w", err)
		}
		return fmt.Errorf("osascript clipboard write: %w: %s", err, msg)
	}
	return nil
}

// writeTempFile 把 content 写入一个新临时文件，返回其路径。调用方负责删除。
func writeTempFile(pattern, content string) (string, error) {
	f, err := os.CreateTemp("", pattern)
	if err != nil {
		return "", fmt.Errorf("create temp file: %w", err)
	}
	if _, err := f.WriteString(content); err != nil {
		f.Close()
		os.Remove(f.Name())
		return "", fmt.Errorf("write temp file: %w", err)
	}
	if err := f.Close(); err != nil {
		os.Remove(f.Name())
		return "", fmt.Errorf("close temp file: %w", err)
	}
	return f.Name(), nil
}
