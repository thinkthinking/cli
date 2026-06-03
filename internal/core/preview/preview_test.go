package preview

import (
	"runtime"
	"strings"
	"testing"
)

// TestRenderContainsEssentials 验证预览页包含关键结构：完整 HTML 文档、
// 原样注入的正文、ClipboardItem 复制逻辑、#preview 容器、标题。
func TestRenderContainsEssentials(t *testing.T) {
	r := New()
	body := `<h2 style="color:#2563eb">小标题</h2><p>正文 &amp; 内容</p>`
	out, err := r.Render(body, "我的文章")
	if err != nil {
		t.Fatalf("Render error: %v", err)
	}

	mustContain := []string{
		"<!DOCTYPE html>",
		`id="preview"`,
		"ClipboardItem",
		"text/html",
		"我的文章", // title 出现在 <title> 与工具条
		body,   // 正文原样注入（template.HTML 不转义）
		"复制到公众号",
	}
	for _, sub := range mustContain {
		if !strings.Contains(out, sub) {
			t.Errorf("rendered preview missing %q", sub)
		}
	}
}

// TestRenderEscapesTitleButNotBody 验证 title 被转义、body 不被转义。
func TestRenderEscapesTitleButNotBody(t *testing.T) {
	r := New()
	out, err := r.Render(`<p>hi</p>`, `a<script>x</script>`)
	if err != nil {
		t.Fatalf("Render error: %v", err)
	}
	// body 原样
	if !strings.Contains(out, "<p>hi</p>") {
		t.Error("body should be injected verbatim")
	}
	// title 中的 < 被转义，不应出现裸 <script>x</script>
	if strings.Contains(out, "<script>x</script>") {
		t.Error("title should be HTML-escaped to prevent injection")
	}
}

// TestRenderEmptyTitleFallback 验证空标题回退为「未命名」。
func TestRenderEmptyTitleFallback(t *testing.T) {
	r := New()
	out, err := r.Render(`<p>x</p>`, "")
	if err != nil {
		t.Fatalf("Render error: %v", err)
	}
	if !strings.Contains(out, "未命名") {
		t.Error("empty title should fall back to 未命名")
	}
}

// TestOpenCommand 验证各平台打开命令映射。
func TestOpenCommand(t *testing.T) {
	name, args := openCommand("/tmp/x.html")
	switch runtime.GOOS {
	case "darwin":
		if name != "open" || len(args) != 1 || args[0] != "/tmp/x.html" {
			t.Errorf("darwin openCommand = %q %v", name, args)
		}
	case "linux":
		if name != "xdg-open" {
			t.Errorf("linux openCommand = %q", name)
		}
	case "windows":
		if name != "cmd" {
			t.Errorf("windows openCommand = %q", name)
		}
	}
}
