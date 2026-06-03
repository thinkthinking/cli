// Package output 定义 thinkthinking 所有命令的统一输出契约。
//
// 设计目标：面向 LLM Agent / 自动化脚本。所有命令默认输出一个稳定的
// JSON envelope，Agent 可以无歧义地 parse。约定：
//
//   - 成功: {"ok": true,  "data": {...}, "error": null}
//   - 失败: {"ok": false, "data": null, "error": {"code","message","details"}}
//
// JSON 永远走 stdout；日志 / warning / debug 永远走 stderr，绝不污染 stdout。
package output

// Envelope 是所有命令输出的统一信封。
//
// 字段顺序与 init.md 约定一致：ok / data / error。即便为 null 也显式输出
// （error 用指针 + 非 omitempty），让 Agent 可以稳定依赖字段存在性。
type Envelope struct {
	OK    bool      `json:"ok"`
	Data  any       `json:"data"`
	Error *AppError `json:"error"`
}

// AppError 是结构化错误，带机器可读的 code、人类可读的 message，以及
// 任意补充 details（例如缺失的配置项、环境变量名、配置文件路径）。
//
// 它同时实现了 error 接口，因此可以在 core 层当作普通 error 传递，
// 最终由 OutputWriter.Error 序列化。
type AppError struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}

// Error 实现 error 接口。
func (e *AppError) Error() string {
	if e == nil {
		return ""
	}
	return e.Code + ": " + e.Message
}

// WithDetails 返回带补充信息的副本，便于链式构造。
func (e *AppError) WithDetails(details map[string]any) *AppError {
	if e == nil {
		return nil
	}
	clone := *e
	clone.Details = details
	return &clone
}

// OutputWriter 抽象输出目标，便于测试时替换为 buffer。
//
// 与 init.md 接口对齐：Success(data any) error / Error(err AppError) error。
type OutputWriter interface {
	Success(data any) error
	Error(appErr *AppError) error
}
