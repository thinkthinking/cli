// Package logging 提供全局 slog logger。
//
// 铁律：日志只写 stderr，绝不写 stdout。stdout 专属于 JSON envelope，
// 任何污染都会破坏 Agent 对输出的解析。
package logging

import (
	"io"
	"log/slog"
	"os"
)

// Options 控制 logger 行为。
type Options struct {
	// Quiet 为 true 时抬高日志级别到 Error，抑制非必要输出（对应 --quiet）。
	Quiet bool
	// Verbose 为 true 时输出 Debug 级别日志。
	Verbose bool
	// Writer 为日志目标，默认 os.Stderr。仅测试时覆盖。
	Writer io.Writer
}

// New 创建一个写入 stderr 的文本 logger。
func New(opts Options) *slog.Logger {
	w := opts.Writer
	if w == nil {
		w = os.Stderr
	}

	level := slog.LevelInfo
	switch {
	case opts.Quiet:
		level = slog.LevelError
	case opts.Verbose:
		level = slog.LevelDebug
	}

	handler := slog.NewTextHandler(w, &slog.HandlerOptions{Level: level})
	return slog.New(handler)
}
