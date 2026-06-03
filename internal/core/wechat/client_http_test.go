package wechat

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestHTTPClientDraftFlow 用 mock HTTP server 验证真实 httpClient 的端到端流程：
// 获取 token → 创建草稿，确认打到正确的 endpoint 且 token 被携带。
func TestHTTPClientDraftFlow(t *testing.T) {
	var gotTokenReq, gotDraftReq bool
	var draftBody string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/cgi-bin/token"):
			gotTokenReq = true
			if r.URL.Query().Get("appid") != "app123" {
				t.Errorf("token request missing appid")
			}
			_, _ = w.Write([]byte(`{"access_token":"TOK","expires_in":7200}`))
		case strings.HasPrefix(r.URL.Path, "/cgi-bin/draft/add"):
			gotDraftReq = true
			if r.URL.Query().Get("access_token") != "TOK" {
				t.Errorf("draft request missing access_token, got %q", r.URL.Query().Get("access_token"))
			}
			buf := make([]byte, r.ContentLength)
			_, _ = r.Body.Read(buf)
			draftBody = string(buf)
			_, _ = w.Write([]byte(`{"media_id":"M999"}`))
		default:
			t.Errorf("unexpected request to %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	// 临时指向 mock server。
	orig := apiBase
	apiBase = srv.URL
	defer func() { apiBase = orig }()

	client := NewClient(Credentials{AppID: "app123", AppSecret: "secret456"})
	mediaID, raw, err := client.AddDraft(context.Background(), []DraftArticle{
		{Title: "中文标题", Content: `<p>你好 & 世界</p>`},
	})
	if err != nil {
		t.Fatalf("AddDraft error: %v", err)
	}
	if !gotTokenReq {
		t.Error("token endpoint was not called")
	}
	if !gotDraftReq {
		t.Error("draft endpoint was not called")
	}
	if mediaID != "M999" {
		t.Errorf("media_id = %q, want M999", mediaID)
	}
	if len(raw) == 0 {
		t.Error("raw response should be preserved")
	}
	// 验证草稿 body 里中文与 HTML 未被转义。
	if !strings.Contains(draftBody, "中文标题") || !strings.Contains(draftBody, "<p>") {
		t.Errorf("draft body should preserve raw CJK/HTML, got: %s", draftBody)
	}
}

// TestHTTPClientTokenError 验证 token 接口返回错误码时被正确包装。
func TestHTTPClientTokenError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"errcode":40013,"errmsg":"invalid appid"}`))
	}))
	defer srv.Close()

	orig := apiBase
	apiBase = srv.URL
	defer func() { apiBase = orig }()

	client := NewClient(Credentials{AppID: "bad", AppSecret: "bad"})
	_, err := client.AccessToken(context.Background())
	if err == nil {
		t.Fatal("expected error for bad appid")
	}
	var apiErr *APIError
	if !asAPIError(err, &apiErr) || apiErr.ErrCode != 40013 {
		t.Errorf("expected APIError with errcode 40013, got %v", err)
	}
}

// asAPIError 是 errors.As 的薄封装（避免在测试里 import errors 仅为一处）。
func asAPIError(err error, target **APIError) bool {
	if e, ok := err.(*APIError); ok {
		*target = e
		return true
	}
	return false
}
