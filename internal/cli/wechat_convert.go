package cli

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"

	"github.com/spf13/cobra"
	"github.com/thinkthinking/cli/internal/core/clipboard"
	"github.com/thinkthinking/cli/internal/core/output"
	"github.com/thinkthinking/cli/internal/core/wechat"
)

// newWeChatConvertCmd 实现 `thinkthinking wechat convert`。
func newWeChatConvertCmd() *cobra.Command {
	var (
		input     string
		out       string
		stdin     bool
		theme     string
		noDark    bool
		noFoot    bool
		noCont    bool
		doCopy    bool
		doPreview bool
	)

	cmd := &cobra.Command{
		Use:   "convert",
		Short: "将 Markdown 转换为微信公众号兼容 HTML",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			// 读取 Markdown 来源：--stdin 或 --input 二选一。
			markdown, srcLabel, aerr := readMarkdownSource(cmd, input, stdin)
			if aerr != nil {
				return aerr
			}

			// 主题：flag > 配置默认。
			if theme == "" {
				theme = container.Config.WeChat.DefaultTheme
			}

			opts := wechat.DefaultConvertOptions()
			opts.DarkMode = !noDark
			opts.Footnotes = !noFoot
			opts.Containers = !noCont

			result, err := container.Converter.Convert(context.Background(), wechat.ConvertInput{
				Markdown:  markdown,
				ThemeName: theme,
				Options:   opts,
			})
			if err != nil {
				// 主题不存在属于用户输入问题，归类为 INVALID_INPUT。
				if errors.Is(err, wechat.ErrThemeNotFound) {
					return output.New(output.CodeInvalidInput, err.Error())
				}
				return output.Wrap(err, output.CodeMarkdownConvert)
			}

			// 基础 data 字段，所有路径共用。
			data := map[string]any{
				"input":  srcLabel,
				"title":  result.Title,
				"digest": result.Digest,
				"theme":  result.ThemeName,
				"images": result.Images,
			}
			warnings := result.Warnings

			// html 字段策略：指定 --output 时写文件并省略完整 html（防止 JSON 过大），
			// 否则把 html 带在返回里。--copy / --preview 只新增字段，不改变既有取值。
			if out != "" {
				if werr := writeOutputFile(out, result.HTML); werr != nil {
					return output.Wrap(werr, output.CodeInternalError)
				}
				data["output"] = out
				data["html_path"] = out
			} else {
				data["html"] = result.HTML
			}

			// --copy：把 HTML 以 text/html flavor 写入系统剪贴板（仅 macOS）。
			if doCopy {
				plain := wechat.HTMLToPlainText(result.HTML)
				if cerr := container.Clipboard.WriteHTML(context.Background(), result.HTML, plain); cerr != nil {
					if errors.Is(cerr, clipboard.ErrUnsupported) {
						return output.New(output.CodePlatformNotSupported, "当前平台不支持 --copy（仅 macOS）").
							WithDetails(map[string]any{
								"platform": runtime.GOOS,
								"hint":     "非 macOS 请改用 --preview（浏览器一键复制）或 --output 写文件后手动导入",
							})
					}
					return output.Wrap(cerr, output.CodeInternalError)
				}
				data["copied"] = true
			}

			// --preview：生成预览页写临时文件并打开浏览器（打开失败仅告警，不致命）。
			if doPreview {
				page, perr := container.Preview.Render(result.HTML, result.Title)
				if perr != nil {
					return output.Wrap(perr, output.CodeInternalError)
				}
				previewPath, perr := writeTempPreview(page)
				if perr != nil {
					return output.Wrap(perr, output.CodeInternalError)
				}
				data["preview_path"] = previewPath
				opened := true
				if oerr := container.Preview.Open(previewPath); oerr != nil {
					opened = false
					warnings = append(warnings, "无法自动打开浏览器，请手动打开: "+previewPath+"（"+oerr.Error()+"）")
				}
				data["opened"] = opened
			}

			data["warnings"] = warnings
			return container.Output.Success(data)
		},
	}

	f := cmd.Flags()
	f.StringVarP(&input, "input", "i", "", "Markdown 输入文件路径")
	f.StringVarP(&out, "output", "o", "", "输出 HTML 文件路径（指定后 JSON 不返回完整 html）")
	f.BoolVar(&stdin, "stdin", false, "从 stdin 读取 Markdown")
	f.StringVarP(&theme, "theme", "t", "", "主题名（默认读配置 wechat.default_theme）")
	f.BoolVar(&noDark, "no-darkmode", false, "禁用暗黑模式属性注入")
	f.BoolVar(&noFoot, "no-footnotes", false, "禁用外链转脚注")
	f.BoolVar(&noCont, "no-containers", false, "禁用 ::: 容器块语法")
	f.BoolVar(&doCopy, "copy", false, "把 HTML 以富文本写入系统剪贴板（仅 macOS），公众号 Cmd+V 即渲染")
	f.BoolVar(&doPreview, "preview", false, "生成浏览器预览页并打开，页面内可一键复制到公众号")
	return cmd
}

// readMarkdownSource 按 --stdin / --input 读取 Markdown，返回内容与来源标签。
func readMarkdownSource(cmd *cobra.Command, input string, stdin bool) (string, string, *output.AppError) {
	if stdin {
		data, err := io.ReadAll(cmd.InOrStdin())
		if err != nil {
			return "", "", output.Wrap(err, output.CodeInvalidInput)
		}
		return string(data), "<stdin>", nil
	}
	if input == "" {
		return "", "", output.New(output.CodeInvalidInput, "either --input or --stdin is required")
	}
	data, err := os.ReadFile(input)
	if err != nil {
		if os.IsNotExist(err) {
			return "", "", output.Newf(output.CodeFileNotFound, "input file not found: %s", input)
		}
		return "", "", output.Wrap(err, output.CodeInvalidInput)
	}
	return string(data), input, nil
}

// writeOutputFile 写 HTML 到指定路径，自动创建父目录。
func writeOutputFile(path, content string) error {
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

// writeTempPreview 把预览页 HTML 写入系统临时目录，返回文件路径（供浏览器打开）。
func writeTempPreview(content string) (string, error) {
	f, err := os.CreateTemp("", "thinkthinking-preview-*.html")
	if err != nil {
		return "", err
	}
	if _, err := f.WriteString(content); err != nil {
		f.Close()
		return "", err
	}
	if err := f.Close(); err != nil {
		return "", err
	}
	return f.Name(), nil
}
