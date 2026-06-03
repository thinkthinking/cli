package output

import (
	"encoding/json"
	"io"
)

// jsonWriter 是默认的 OutputWriter 实现，输出统一 JSON envelope 到给定 writer
// （生产环境为 os.Stdout，测试时可替换为 bytes.Buffer）。
type jsonWriter struct {
	w      io.Writer
	pretty bool
}

// NewJSONWriter 创建一个 JSON OutputWriter。
//
// pretty=true 时缩进美化（对应 --pretty）；否则单行紧凑输出，便于 Agent 逐行 parse。
func NewJSONWriter(w io.Writer, pretty bool) OutputWriter {
	return &jsonWriter{w: w, pretty: pretty}
}

// Success 输出成功 envelope。
func (j *jsonWriter) Success(data any) error {
	return j.encode(Envelope{OK: true, Data: data, Error: nil})
}

// Error 输出失败 envelope。nil 时归一化为 INTERNAL_ERROR，保证永远有结构化错误。
func (j *jsonWriter) Error(appErr *AppError) error {
	if appErr == nil {
		appErr = New(CodeInternalError, "unknown error")
	}
	return j.encode(Envelope{OK: false, Data: nil, Error: appErr})
}

// encode 执行实际的 JSON 编码。
//
// 关键：SetEscapeHTML(false)。Go encoding/json 默认会把 <、>、& 转义成
// < 等，这会破坏我们输出的微信 HTML 字段，让 Agent 拿到不可读的转义串
// （也是 wewrite ensure_ascii=False、md2wechat-lite SetEscapeHTML(false) 同源的坑）。
// 中文本身 encoding/json 不会转 \uXXXX，但 HTML 符号必须靠这个关掉。
func (j *jsonWriter) encode(v any) error {
	enc := json.NewEncoder(j.w)
	enc.SetEscapeHTML(false)
	if j.pretty {
		enc.SetIndent("", "  ")
	}
	// json.Encoder.Encode 会自动追加换行符，便于 Agent 按行读取。
	return enc.Encode(v)
}
