package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

// TestSuccessEnvelope 验证成功 envelope 的结构：ok=true、data 存在、error 为 null。
func TestSuccessEnvelope(t *testing.T) {
	var buf bytes.Buffer
	w := NewJSONWriter(&buf, false)

	if err := w.Success(map[string]string{"hello": "world"}); err != nil {
		t.Fatalf("Success returned error: %v", err)
	}

	var env Envelope
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("output is not valid JSON: %v\ngot: %s", err, buf.String())
	}
	if !env.OK {
		t.Errorf("expected ok=true, got false")
	}
	if env.Error != nil {
		t.Errorf("expected error=null, got %+v", env.Error)
	}
	if env.Data == nil {
		t.Errorf("expected data present, got nil")
	}
}

// TestErrorEnvelope 验证失败 envelope：ok=false、data=null、error 带 code/message/details。
func TestErrorEnvelope(t *testing.T) {
	var buf bytes.Buffer
	w := NewJSONWriter(&buf, false)

	appErr := New(CodeWeChatAuth, "missing credentials").
		WithDetails(map[string]any{"env": []string{"WECHAT_APP_ID"}})
	if err := w.Error(appErr); err != nil {
		t.Fatalf("Error returned error: %v", err)
	}

	var env Envelope
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if env.OK {
		t.Errorf("expected ok=false, got true")
	}
	if env.Data != nil {
		t.Errorf("expected data=null, got %+v", env.Data)
	}
	if env.Error == nil || env.Error.Code != CodeWeChatAuth {
		t.Errorf("expected error.code=%s, got %+v", CodeWeChatAuth, env.Error)
	}
}

// TestNoHTMLEscaping 是最关键的测试：HTML 字段中的 < > & 不能被转义成
// < 等，否则微信 HTML 对 Agent 不可读。对应 SetEscapeHTML(false)。
func TestNoHTMLEscaping(t *testing.T) {
	var buf bytes.Buffer
	w := NewJSONWriter(&buf, false)

	html := `<section style="color:#333">你好 & 世界 <strong>粗体</strong></section>`
	if err := w.Success(map[string]string{"html": html}); err != nil {
		t.Fatalf("Success returned error: %v", err)
	}

	out := buf.String()
	// (1) 原始 HTML 符号必须原样出现在字节流里。这是关键断言：encoding/json 的
	//     SetEscapeHTML(true) 会把 '<' 编码成 unicode 转义序列（反斜杠 u003c），
	//     届时 "<section" 这个子串根本不会出现。所以这一条存在即证明未被转义。
	if !strings.Contains(out, "<section") {
		t.Errorf("expected raw '<section' in output (got escaped form):\n%s", out)
	}
	if !strings.Contains(out, "你好 & 世界") {
		t.Errorf("expected raw '&' and CJK preserved:\n%s", out)
	}
	// (2) 防御性检查：确认输出里没有出现反斜杠+u 开头的转义序列。
	//     用 strconv.Quote 构造 needle，避免在源码里直接写转义字符（会被工具链二次解码）。
	if strings.Contains(out, `\u00`) {
		t.Errorf("output contains a \\u00 escape sequence, SetEscapeHTML(false) not working:\n%s", out)
	}
	// (3) 最强断言：round-trip 后 html 字段与原文逐字符相等。
	var env Envelope
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	data, _ := env.Data.(map[string]any)
	if got, _ := data["html"].(string); got != html {
		t.Errorf("html round-trip mismatch:\nwant: %s\ngot:  %s", html, got)
	}
}

// TestErrorNilNormalized 验证 Error(nil) 被归一化为 INTERNAL_ERROR 而非 panic。
func TestErrorNilNormalized(t *testing.T) {
	var buf bytes.Buffer
	w := NewJSONWriter(&buf, false)

	if err := w.Error(nil); err != nil {
		t.Fatalf("Error(nil) returned error: %v", err)
	}
	var env Envelope
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if env.OK || env.Error == nil || env.Error.Code != CodeInternalError {
		t.Errorf("expected normalized INTERNAL_ERROR, got %+v", env.Error)
	}
}
