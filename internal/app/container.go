// Package app 提供 Application Container：集中装配并持有所有跨命令复用的
// 服务（Config / Logger / OutputWriter / WeChat Converter / Draft Service）。
//
// 设计意图（来自 init.md）：CLI command 不直接 new 具体实现，一律从 Container
// 取 service。这样未来 TUI / GUI / Local API 可以复用同一套 core 能力，而不必
// 重写业务逻辑。Container 随实现阶段逐步填充字段，是有意的渐进式装配。
package app

import (
	"io"
	"log/slog"
	"path/filepath"

	"github.com/thinkthinking/cli/internal/config"
	"github.com/thinkthinking/cli/internal/core/output"
	"github.com/thinkthinking/cli/internal/core/wechat"
	"github.com/thinkthinking/cli/internal/logging"
)

// Options 是构建 Container 所需的输入，来自已解析的全局 flags。
type Options struct {
	// ConfigPath 是 --config 指定的配置文件路径（可空，空则走默认搜索）。
	ConfigPath string
	// Pretty 对应 --pretty，控制 JSON 是否缩进。
	Pretty bool
	// Quiet 对应 --quiet。
	Quiet bool
	// Verbose 控制 debug 日志。
	Verbose bool
	// Stdout / Stderr 为输出目标；为空时由调用方（main）注入 os.Stdout/os.Stderr。
	Stdout io.Writer
	Stderr io.Writer
}

// Container 持有所有装配好的服务。
type Container struct {
	Logger *slog.Logger
	Output output.OutputWriter

	// Config 与 ConfigLoad 在装配时加载。ConfigErr 记录加载错误，由命令决定
	// 是否致命（例如 version 不依赖配置，convert/draft 依赖）。
	Config     *config.Config
	ConfigLoad *config.LoadResult
	ConfigErr  error

	// Converter 是 Markdown→微信 HTML 转换器，带用户/项目主题覆盖目录。
	Converter wechat.MarkdownConverter

	// DraftService 创建微信草稿（编排转换 + 上传 + 草稿 API）。
	DraftService wechat.DraftService
}

// New 装配一个 Container。
func New(opts Options) *Container {
	logger := logging.New(logging.Options{
		Quiet:   opts.Quiet,
		Verbose: opts.Verbose,
		Writer:  opts.Stderr,
	})

	c := &Container{
		Logger: logger,
		Output: output.NewJSONWriter(opts.Stdout, opts.Pretty),
	}

	// 加载配置（失败不 panic，记录到 ConfigErr 供命令按需处理）。
	loaded, err := config.Load(config.LoadOptions{ExplicitPath: opts.ConfigPath})
	if err != nil {
		c.ConfigErr = err
		// 即便加载失败，也给出一份默认配置，避免下游空指针。
		def := config.Defaults()
		c.Config = &def
	} else {
		c.Config = loaded.Config
		c.ConfigLoad = loaded
	}

	// 装配转换器，带主题覆盖目录（用户级 + 项目级）。
	c.Converter = wechat.NewConverter(themeOverrideDirs()...)

	// 装配草稿服务，凭证来自配置（已合并 env 覆盖）。
	client := wechat.NewClient(wechat.Credentials{
		AppID:       c.Config.WeChat.AppID,
		AppSecret:   c.Config.WeChat.AppSecret,
		AccessToken: c.Config.WeChat.AccessToken,
	})
	c.DraftService = wechat.NewDraftService(c.Converter, client)

	return c
}

// themeOverrideDirs 返回主题覆盖目录：~/.thinkthinking/themes 与
// ./.thinkthinking/themes。加载时后者优先级更高，均可覆盖内嵌主题。
func themeOverrideDirs() []string {
	var dirs []string
	if userDir, err := config.UserThemesDir(); err == nil {
		dirs = append(dirs, userDir)
	}
	dirs = append(dirs, filepath.Join(config.ProjectConfigDir(), config.ThemesSubDir))
	return dirs
}
