package wechat

import (
	"os"
	"strings"
	"testing"
)

// TestMarshalNoEscape 验证草稿 JSON body 不转义中文与 HTML 符号。
//
// 这是移植 wewrite ensure_ascii=False 的关键：若用默认 json.Marshal，
// 微信会把 < > & 的转义字面量存进去，导致标题溢出与正文乱码。
func TestMarshalNoEscape(t *testing.T) {
	article := DraftArticle{
		Title:   "中文标题 & 测试",
		Content: `<section style="color:#333">你好 <strong>世界</strong></section>`,
	}
	data, err := marshalNoEscape(draftAddRequest{Articles: []DraftArticle{article}})
	if err != nil {
		t.Fatalf("marshalNoEscape error: %v", err)
	}
	out := string(data)

	// HTML 标签必须原样保留（不被转成 unicode 转义）。
	if !strings.Contains(out, "<section") || !strings.Contains(out, "<strong>") {
		t.Errorf("HTML tags should be preserved raw, got:\n%s", out)
	}
	// 中文必须原样（encoding/json 本就不转中文，但确认一下）。
	if !strings.Contains(out, "中文标题") || !strings.Contains(out, "你好") {
		t.Errorf("CJK should be preserved, got:\n%s", out)
	}
	// 不应出现 \u00 开头的转义序列。
	if strings.Contains(out, `\u00`) {
		t.Errorf("should not contain \\u00 escapes, got:\n%s", out)
	}
}

// TestResolveImagePath 验证正文图片路径解析的三级回退逻辑。
func TestResolveImagePath(t *testing.T) {
	// 绝对路径不存在 → 空。
	if got := resolveImagePath("/no/such/abs.png", ""); got != "" {
		t.Errorf("nonexistent abs path should resolve to empty, got %q", got)
	}
	// 创建一个临时文件，验证相对 mdDir 解析。
	dir := t.TempDir()
	imgPath := dir + "/pic.png"
	if err := os.WriteFile(imgPath, []byte("fake"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := resolveImagePath("pic.png", dir); got != imgPath {
		t.Errorf("relative-to-mdDir resolve = %q, want %q", got, imgPath)
	}
}
