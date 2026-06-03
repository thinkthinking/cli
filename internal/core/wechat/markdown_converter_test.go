package wechat

import (
	"context"
	"strings"
	"testing"
)

// convert 是测试辅助：用默认选项转换给定 Markdown。
func convert(t *testing.T, md string) *ConvertResult {
	t.Helper()
	c := NewConverter()
	res, err := c.Convert(context.Background(), ConvertInput{
		Markdown:  md,
		ThemeName: "default",
		Options:   DefaultConvertOptions(),
	})
	if err != nil {
		t.Fatalf("Convert error: %v", err)
	}
	return res
}

// TestExtractTitleAndStrip 验证 H1 被提取为 title 且从正文剔除。
func TestExtractTitleAndStrip(t *testing.T) {
	res := convert(t, "# 我的标题\n\n正文内容。")
	if res.Title != "我的标题" {
		t.Errorf("title = %q, want 我的标题", res.Title)
	}
	if strings.Contains(res.HTML, "我的标题") {
		t.Errorf("H1 should be stripped from body, but found in HTML:\n%s", res.HTML)
	}
}

// TestHeadingAndParagraph 基础元素：H2 和段落带 inline style。
func TestHeadingAndParagraph(t *testing.T) {
	res := convert(t, "## 小标题\n\n一段文字。")
	if !strings.Contains(res.HTML, "<h2 style=") {
		t.Errorf("expected styled h2, got:\n%s", res.HTML)
	}
	if !strings.Contains(res.HTML, "<p style=") {
		t.Errorf("expected styled p, got:\n%s", res.HTML)
	}
}

// TestBlockquote 引用块带微信风格 inline style。
func TestBlockquote(t *testing.T) {
	res := convert(t, "> 这是一段引用。")
	if !strings.Contains(res.HTML, "<blockquote style=") {
		t.Errorf("expected styled blockquote, got:\n%s", res.HTML)
	}
	if !strings.Contains(res.HTML, "border-left") {
		t.Errorf("blockquote should have border-left style, got:\n%s", res.HTML)
	}
}

// TestCodeBlock 代码块用 <pre data-lang> 且内联高亮（无 CSS class）。
func TestCodeBlock(t *testing.T) {
	res := convert(t, "```go\nfunc main() {}\n```")
	if !strings.Contains(res.HTML, `data-lang="go"`) {
		t.Errorf("expected pre with data-lang=go, got:\n%s", res.HTML)
	}
	if !strings.Contains(res.HTML, "white-space: pre-wrap") {
		t.Errorf("pre should have white-space:pre-wrap, got:\n%s", res.HTML)
	}
	// 不应残留 chroma 的 class（noclasses 模式）。
	if strings.Contains(res.HTML, `class="chroma"`) {
		t.Errorf("should not contain chroma class in inline mode, got:\n%s", res.HTML)
	}
}

// TestListsToSections ul/ol 转 section，不应残留原生 <ul>/<li>。
func TestListsToSections(t *testing.T) {
	res := convert(t, "- 甲\n- 乙\n\n1. 一\n2. 二")
	if strings.Contains(res.HTML, "<ul") || strings.Contains(res.HTML, "<li") || strings.Contains(res.HTML, "<ol") {
		t.Errorf("native list tags should be converted to sections, got:\n%s", res.HTML)
	}
	if !strings.Contains(res.HTML, "•") {
		t.Errorf("unordered list should use bullet •, got:\n%s", res.HTML)
	}
	if !strings.Contains(res.HTML, "1.") || !strings.Contains(res.HTML, "2.") {
		t.Errorf("ordered list should have numbers, got:\n%s", res.HTML)
	}
}

// TestCJKSpacing 中英文之间自动加空格。
func TestCJKSpacing(t *testing.T) {
	res := convert(t, "这是 Go 语言")
	// "这是Go语言" 应变成 "这是 Go 语言"。
	if !strings.Contains(res.HTML, "这是 Go 语言") {
		t.Errorf("expected CJK-latin spacing '这是 Go 语言', got:\n%s", res.HTML)
	}
}

// TestCJKSpacingSkipsCodeBlock 代码块内不加 CJK 空格。
func TestCJKSpacingSkipsCodeBlock(t *testing.T) {
	res := convert(t, "```\nvar 变量x = 1\n```")
	// 代码块内 "变量x" 不应被插入空格变成 "变量 x"。
	if strings.Contains(res.HTML, "变量 x") {
		t.Errorf("code block content should not get CJK spacing, got:\n%s", res.HTML)
	}
}

// TestFootnotes 外链转上标脚注 + 文末参考链接。
func TestFootnotes(t *testing.T) {
	res := convert(t, "见 [示例](https://example.com) 链接。")
	if !strings.Contains(res.HTML, "[1]") {
		t.Errorf("expected superscript [1], got:\n%s", res.HTML)
	}
	if !strings.Contains(res.HTML, "参考链接") {
		t.Errorf("expected 参考链接 section, got:\n%s", res.HTML)
	}
	if !strings.Contains(res.HTML, "https://example.com") {
		t.Errorf("expected URL in references, got:\n%s", res.HTML)
	}
	// 原始 <a href> 不应保留。
	if strings.Contains(res.HTML, "<a href") {
		t.Errorf("external <a> should be converted to footnote, got:\n%s", res.HTML)
	}
}

// TestCallout :::callout 容器块。
func TestCallout(t *testing.T) {
	res := convert(t, ":::callout tip\n这是提示\n:::")
	if !strings.Contains(res.HTML, "💡") || !strings.Contains(res.HTML, "TIP") {
		t.Errorf("expected tip callout with 💡 TIP, got:\n%s", res.HTML)
	}
}

// TestDigestByteLimit digest 不超过 120 UTF-8 字节。
func TestDigestByteLimit(t *testing.T) {
	long := "# 标题\n\n" + strings.Repeat("这是一段很长的中文内容用来测试摘要截断。", 20)
	res := convert(t, long)
	if len(res.Digest) > 120 {
		t.Errorf("digest = %d bytes, want <= 120: %q", len(res.Digest), res.Digest)
	}
	if !strings.HasSuffix(res.Digest, "...") {
		t.Errorf("truncated digest should end with ..., got: %q", res.Digest)
	}
}

// TestImagesNonNil 无图片时 Images 为非 nil 空切片（Agent 友好）。
func TestImagesNonNil(t *testing.T) {
	res := convert(t, "纯文本，无图片。")
	if res.Images == nil {
		t.Errorf("Images should be non-nil empty slice, got nil")
	}
}

// TestImageCollected 图片被收集并加响应式 style。
func TestImageCollected(t *testing.T) {
	res := convert(t, "![alt](pic.png)")
	if len(res.Images) != 1 || res.Images[0] != "pic.png" {
		t.Errorf("expected images=[pic.png], got %v", res.Images)
	}
	if !strings.Contains(res.HTML, "max-width: 100%") {
		t.Errorf("image should have responsive style, got:\n%s", res.HTML)
	}
}

// TestThemeNotFound 未知主题返回 ErrThemeNotFound。
func TestThemeNotFound(t *testing.T) {
	c := NewConverter()
	_, err := c.Convert(context.Background(), ConvertInput{
		Markdown:  "test",
		ThemeName: "no-such-theme",
		Options:   DefaultConvertOptions(),
	})
	if err == nil {
		t.Fatal("expected error for unknown theme")
	}
}
