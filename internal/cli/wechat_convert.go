package cli

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/thinkthinking/cli/internal/core/output"
	"github.com/thinkthinking/cli/internal/core/wechat"
)

// newWeChatConvertCmd 实现 `thinkthinking wechat convert`。
func newWeChatConvertCmd() *cobra.Command {
	var (
		input  string
		out    string
		stdin  bool
		theme  string
		noDark bool
		noFoot bool
		noCont bool
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

			// 若指定 --output，写文件并避免在 JSON 里塞完整 html（防止过大）。
			if out != "" {
				if werr := writeOutputFile(out, result.HTML); werr != nil {
					return output.Wrap(werr, output.CodeInternalError)
				}
				return container.Output.Success(map[string]any{
					"input":     srcLabel,
					"output":    out,
					"html_path": out,
					"title":     result.Title,
					"digest":    result.Digest,
					"theme":     result.ThemeName,
					"images":    result.Images,
					"warnings":  result.Warnings,
				})
			}

			return container.Output.Success(map[string]any{
				"input":    srcLabel,
				"html":     result.HTML,
				"title":    result.Title,
				"digest":   result.Digest,
				"theme":    result.ThemeName,
				"images":   result.Images,
				"warnings": result.Warnings,
			})
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
