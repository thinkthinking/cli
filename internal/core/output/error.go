package output

import "fmt"

// 错误码常量。所有 core / cli 层产生的错误都应归类到这些稳定的 code，
// 方便 Agent 基于 code 分支处理，而不是 parse message 文本。
const (
	CodeInvalidInput    = "INVALID_INPUT"
	CodeConfigError     = "CONFIG_ERROR"
	CodeFileNotFound    = "FILE_NOT_FOUND"
	CodeMarkdownConvert = "MARKDOWN_CONVERT_ERROR"
	CodeWeChatAuth      = "WECHAT_AUTH_ERROR"
	CodeWeChatAPI       = "WECHAT_API_ERROR"
	CodeNetworkError    = "NETWORK_ERROR"
	CodeInternalError   = "INTERNAL_ERROR"
)

// New 构造一个 AppError。
func New(code, message string) *AppError {
	return &AppError{Code: code, Message: message}
}

// Newf 同 New，但 message 支持格式化（避免调用方手动 fmt.Sprintf）。
func Newf(code, format string, args ...any) *AppError {
	return &AppError{Code: code, Message: fmt.Sprintf(format, args...)}
}

// Wrap 把任意 error 归一化为 AppError。
//
//   - 已是 *AppError：原样返回（保留原始 code/details）。
//   - 其它 error：包成给定 code 的 AppError。
//   - nil：返回 nil。
//
// 这样 core 层可以自由返回 *AppError 或普通 error，CLI 层统一 Wrap 后输出。
func Wrap(err error, fallbackCode string) *AppError {
	if err == nil {
		return nil
	}
	if appErr, ok := err.(*AppError); ok {
		return appErr
	}
	return &AppError{Code: fallbackCode, Message: err.Error()}
}
