package wechat

import (
	"errors"
	"fmt"
)

// ErrThemeNotFound 表示请求的主题不存在。CLI 层据此映射为 INVALID_INPUT。
var ErrThemeNotFound = errors.New("theme not found")

// ThemeNotFoundError 包装主题名，便于错误信息携带上下文同时可被 errors.Is 识别。
type ThemeNotFoundError struct {
	Name string
}

func (e *ThemeNotFoundError) Error() string {
	return fmt.Sprintf("theme not found: %s", e.Name)
}

// Is 让 errors.Is(err, ErrThemeNotFound) 成立。
func (e *ThemeNotFoundError) Is(target error) bool {
	return target == ErrThemeNotFound
}

// AuthError 表示凭证缺失或无效（CLI 映射为 WECHAT_AUTH_ERROR）。
type AuthError struct {
	Reason string
}

func (e *AuthError) Error() string { return "wechat auth error: " + e.Reason }

// NetworkError 表示网络层失败（CLI 映射为 NETWORK_ERROR）。
type NetworkError struct {
	Op  string
	Err error
}

func (e *NetworkError) Error() string {
	return fmt.Sprintf("network error during %s: %v", e.Op, e.Err)
}
func (e *NetworkError) Unwrap() error { return e.Err }

// FileError 表示本地文件问题（CLI 映射为 FILE_NOT_FOUND）。
type FileError struct {
	Path string
	Err  error
}

func (e *FileError) Error() string {
	return fmt.Sprintf("file error: %s: %v", e.Path, e.Err)
}
func (e *FileError) Unwrap() error { return e.Err }

// APIError 表示微信 API 返回的业务错误（CLI 映射为 WECHAT_API_ERROR）。
// 携带 errcode/errmsg，并在已知错误码时附带可操作的中文提示（吸收 md2wechat-skill）。
type APIError struct {
	Op      string
	ErrCode int
	ErrMsg  string
	Err     error
}

func (e *APIError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("wechat api error during %s: %v", e.Op, e.Err)
	}
	base := fmt.Sprintf("wechat api error during %s: errcode=%d errmsg=%s", e.Op, e.ErrCode, e.ErrMsg)
	if hint := draftErrorHint(e.ErrCode); hint != "" {
		return base + "; hint: " + hint
	}
	return base
}
func (e *APIError) Unwrap() error { return e.Err }

// Hint 返回该错误码的可操作提示（供 CLI 放进 error.details）。
func (e *APIError) Hint() string {
	return draftErrorHint(e.ErrCode)
}

// draftErrorHint 把常见微信草稿错误码翻译成可操作提示（吸收 md2wechat-skill errors.go）。
func draftErrorHint(code int) string {
	switch code {
	case 40001:
		return "access_token 无效或已过期，请检查 app_id/app_secret 或重新获取 token。"
	case 40164, 45028:
		return "调用方 IP 不在公众号 IP 白名单中，请在公众号后台「开发-基本配置」添加服务器 IP。"
	case 41001:
		return "缺少 access_token 参数。"
	case 45002:
		return "草稿内容超出微信限制，请精简正文或减小内嵌 HTML 体积。"
	case 45003:
		return "草稿标题超出限制，请将标题缩短到 64 字节以内。"
	case 45004:
		return "草稿摘要超出限制，请将 digest 缩短到 120 字节以内。"
	case 53404:
		return "账号已被限制操作权限。"
	case 53500, 53501:
		return "触发发布频率限制或内容安全校验，请稍后重试或检查内容合规性。"
	default:
		return ""
	}
}
