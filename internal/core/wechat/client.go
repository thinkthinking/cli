package wechat

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// apiBase 是微信公众号 API 根地址。抽成变量便于测试时指向 mock server。
var apiBase = "https://api.weixin.qq.com"

// Credentials 是调用微信 API 所需的凭证。
type Credentials struct {
	AppID       string
	AppSecret   string
	AccessToken string // 可选：直接提供，跳过 token 获取
}

// WeChatClient 抽象微信 API 调用，便于 draft service 测试时 mock。
type WeChatClient interface {
	// AccessToken 返回有效的 access_token（带缓存）。
	AccessToken(ctx context.Context) (string, error)
	// UploadContentImage 上传正文图片，返回可直接用于 content 的 url。
	UploadContentImage(ctx context.Context, imagePath string) (string, error)
	// UploadMaterial 上传永久素材（用作封面），返回 media_id 和 url。
	UploadMaterial(ctx context.Context, imagePath string) (mediaID string, url string, err error)
	// AddDraft 创建草稿，返回 media_id 与原始响应体（保留供 debug）。
	AddDraft(ctx context.Context, articles []DraftArticle) (mediaID string, raw json.RawMessage, err error)
}

// httpClient 是 WeChatClient 的真实实现。
type httpClient struct {
	creds Credentials
	hc    *http.Client

	mu          sync.Mutex
	cachedToken string
	tokenExpiry time.Time
}

// NewClient 创建一个真实的微信 API 客户端。
func NewClient(creds Credentials) WeChatClient {
	return &httpClient{
		creds: creds,
		hc:    &http.Client{Timeout: 30 * time.Second},
	}
}

// tokenResponse 是 /cgi-bin/token 的响应。
type tokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	ErrCode     int    `json:"errcode"`
	ErrMsg      string `json:"errmsg"`
}

// AccessToken 返回有效 token。优先用预置 token；否则用 appid/secret 获取并缓存
// 到 expires_in - 300s（5 分钟 buffer，移植自 wewrite wechat_api.py）。
func (c *httpClient) AccessToken(ctx context.Context) (string, error) {
	// 预置 token 直接用。
	if c.creds.AccessToken != "" {
		return c.creds.AccessToken, nil
	}
	if c.creds.AppID == "" || c.creds.AppSecret == "" {
		return "", &AuthError{Reason: "missing app_id or app_secret"}
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.cachedToken != "" && time.Now().Before(c.tokenExpiry) {
		return c.cachedToken, nil
	}

	u := fmt.Sprintf("%s/cgi-bin/token?grant_type=client_credential&appid=%s&secret=%s",
		apiBase, c.creds.AppID, c.creds.AppSecret)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return "", &NetworkError{Op: "build token request", Err: err}
	}
	resp, err := c.hc.Do(req)
	if err != nil {
		return "", &NetworkError{Op: "get access_token", Err: err}
	}
	defer resp.Body.Close()

	var tr tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tr); err != nil {
		return "", &APIError{Op: "decode token response", Err: err}
	}
	if tr.AccessToken == "" {
		return "", &APIError{ErrCode: tr.ErrCode, ErrMsg: tr.ErrMsg, Op: "get access_token"}
	}

	expiresIn := tr.ExpiresIn
	if expiresIn <= 0 {
		expiresIn = 7200
	}
	c.cachedToken = tr.AccessToken
	c.tokenExpiry = time.Now().Add(time.Duration(expiresIn-300) * time.Second)
	return c.cachedToken, nil
}

// postMultipartFile 向给定路径 POST 一个 multipart 文件（字段名 media）。
func (c *httpClient) postMultipartFile(ctx context.Context, urlPath string, imagePath string) ([]byte, error) {
	f, err := os.Open(imagePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, &FileError{Path: imagePath, Err: err}
		}
		return nil, &FileError{Path: imagePath, Err: err}
	}
	defer f.Close()

	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	part, err := w.CreateFormFile("media", filepath.Base(imagePath))
	if err != nil {
		return nil, &APIError{Op: "create multipart", Err: err}
	}
	if _, err := io.Copy(part, f); err != nil {
		return nil, &APIError{Op: "copy file to multipart", Err: err}
	}
	if err := w.Close(); err != nil {
		return nil, &APIError{Op: "close multipart", Err: err}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, urlPath, &body)
	if err != nil {
		return nil, &NetworkError{Op: "build upload request", Err: err}
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, &NetworkError{Op: "upload", Err: err}
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

// AddDraft 创建草稿。关键：JSON body 必须不转义中文与 HTML 符号，
// 否则微信会把转义字面量存进去（标题溢出、乱码）——对应 wewrite ensure_ascii=False。
func (c *httpClient) AddDraft(ctx context.Context, articles []DraftArticle) (string, json.RawMessage, error) {
	token, err := c.AccessToken(ctx)
	if err != nil {
		return "", nil, err
	}

	payload := draftAddRequest{Articles: articles}
	bodyBytes, err := marshalNoEscape(payload)
	if err != nil {
		return "", nil, &APIError{Op: "marshal draft", Err: err}
	}

	u := fmt.Sprintf("%s/cgi-bin/draft/add?access_token=%s", apiBase, token)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", nil, &NetworkError{Op: "build draft request", Err: err}
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	resp, err := c.hc.Do(req)
	if err != nil {
		return "", nil, &NetworkError{Op: "create draft", Err: err}
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	var dr draftAddResponse
	if err := json.Unmarshal(raw, &dr); err != nil {
		return "", raw, &APIError{Op: "decode draft response", Err: err}
	}
	if dr.ErrCode != 0 {
		return "", raw, &APIError{ErrCode: dr.ErrCode, ErrMsg: dr.ErrMsg, Op: "create draft"}
	}
	if dr.MediaID == "" {
		return "", raw, &APIError{Op: "create draft", ErrMsg: "missing media_id in response"}
	}
	return dr.MediaID, raw, nil
}

// marshalNoEscape 序列化为不转义 HTML 符号的 JSON（对应 ensure_ascii=False 的关键部分）。
func marshalNoEscape(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
