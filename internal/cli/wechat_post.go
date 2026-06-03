package cli

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/thinkthinking/cli/internal/core/output"
	"github.com/thinkthinking/cli/internal/core/wechat"
)

// newWeChatPostCmd 实现 `thinkthinking wechat post`：把 Markdown/HTML 发布为
// 微信公众号草稿（自动转换、上传正文本地图片、上传封面）。
//
// 设计取舍：post 是面向「发一篇文章」的高频命令，刻意收敛参数——
// --title / --author / --cover 三者必填，--markdown-file 与 --html-file 二选一，
// 由 cobra 的内置校验给出清晰错误（统一包装成 INVALID_INPUT JSON）。
func newWeChatPostCmd() *cobra.Command {
	var (
		title        string
		markdownFile string
		htmlFile     string
		theme        string
		author       string
		digest       string
		cover        string
		noUpload     bool
	)

	cmd := &cobra.Command{
		Use:   "post",
		Short: "把 Markdown/HTML 发布为微信公众号草稿（自动转换 + 上传图片与封面）",
		Long: `把一篇文章发布为微信公众号草稿。

自动完成：Markdown → 微信兼容 HTML → 上传正文本地图片并回填 URL → 上传封面 → 写入草稿箱。
--title / --author / --cover 必填；正文来源 --markdown-file 与 --html-file 二选一。`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			// 凭证预检：缺失则返回结构化 WECHAT_AUTH_ERROR（不 panic）。
			if aerr := checkWeChatCredentials(); aerr != nil {
				return aerr
			}

			input := wechat.CreateDraftInput{
				Title:        title,
				Author:       author,
				Digest:       digest,
				CoverImage:   cover,
				UploadImages: !noUpload,
			}

			if markdownFile != "" {
				data, rerr := os.ReadFile(markdownFile)
				if rerr != nil {
					if os.IsNotExist(rerr) {
						return output.Newf(output.CodeFileNotFound, "markdown file not found: %s", markdownFile)
					}
					return output.Wrap(rerr, output.CodeInvalidInput)
				}
				input.Markdown = string(data)
				input.MarkdownDir = filepath.Dir(markdownFile)
				input.ThemeName = firstNonEmpty(theme, container.Config.WeChat.DefaultTheme)
				input.ConvertOptions = wechat.DefaultConvertOptions()
			} else {
				data, rerr := os.ReadFile(htmlFile)
				if rerr != nil {
					if os.IsNotExist(rerr) {
						return output.Newf(output.CodeFileNotFound, "html file not found: %s", htmlFile)
					}
					return output.Wrap(rerr, output.CodeInvalidInput)
				}
				input.HTML = string(data)
			}

			result, err := container.DraftService.CreateDraft(context.Background(), input)
			if err != nil {
				return mapWeChatError(err)
			}

			return container.Output.Success(map[string]any{
				"media_id":        result.MediaID,
				"title":           result.Title,
				"digest":          result.Digest,
				"thumb_media_id":  result.ThumbMediaID,
				"uploaded_images": result.UploadedImages,
				"created":         true,
			})
		},
	}

	f := cmd.Flags()
	f.StringVar(&title, "title", "", "文章标题（必填）")
	f.StringVar(&author, "author", "", "作者（必填）")
	f.StringVar(&cover, "cover", "", "本地封面图路径，上传为 thumb_media_id（必填）")
	f.StringVar(&markdownFile, "markdown-file", "", "Markdown 文件路径（与 --html-file 二选一）")
	f.StringVar(&htmlFile, "html-file", "", "HTML 文件路径（与 --markdown-file 二选一）")
	f.StringVarP(&theme, "theme", "t", "", "主题名（仅 markdown 路径，默认读配置 wechat.default_theme）")
	f.StringVar(&digest, "digest", "", "摘要（默认用转换生成的摘要）")
	f.BoolVar(&noUpload, "no-upload-images", false, "禁用正文本地图片自动上传")

	// 必填与互斥约束交给 cobra：校验失败返回的 error 由 Execute() 统一包装成
	// INVALID_INPUT 的 JSON envelope，保持 Agent 契约。
	_ = cmd.MarkFlagRequired("title")
	_ = cmd.MarkFlagRequired("author")
	_ = cmd.MarkFlagRequired("cover")
	cmd.MarkFlagsOneRequired("markdown-file", "html-file")
	cmd.MarkFlagsMutuallyExclusive("markdown-file", "html-file")
	return cmd
}

// checkWeChatCredentials 预检凭证：app_id+app_secret 或 access_token 至少一组齐全。
// 返回结构化 WECHAT_AUTH_ERROR，带配置路径与环境变量提示。
func checkWeChatCredentials() *output.AppError {
	wc := container.Config.WeChat
	if wc.AccessToken != "" {
		return nil
	}
	if wc.AppID != "" && wc.AppSecret != "" {
		return nil
	}
	configPath := ""
	if container.ConfigLoad != nil {
		configPath = container.ConfigLoad.UserPath
	}
	return output.New(output.CodeWeChatAuth, "missing wechat app_id or app_secret").
		WithDetails(map[string]any{
			"config_path": configPath,
			"env":         []string{"WECHAT_APP_ID", "WECHAT_APP_SECRET", "WECHAT_ACCESS_TOKEN"},
			"hint":        "运行 thinkthinking config set wechat.app_id <id> 与 wechat.app_secret <secret>，或设置对应环境变量。",
		})
}

// mapWeChatError 把 core 层的 wechat 错误类型映射到统一错误码。
func mapWeChatError(err error) *output.AppError {
	var authErr *wechat.AuthError
	var netErr *wechat.NetworkError
	var fileErr *wechat.FileError
	var apiErr *wechat.APIError

	switch {
	case errors.As(err, &authErr):
		return output.New(output.CodeWeChatAuth, err.Error())
	case errors.As(err, &netErr):
		return output.New(output.CodeNetworkError, err.Error())
	case errors.As(err, &fileErr):
		return output.Newf(output.CodeFileNotFound, "%s", err.Error())
	case errors.As(err, &apiErr):
		ae := output.New(output.CodeWeChatAPI, err.Error())
		if hint := apiErr.Hint(); hint != "" {
			ae = ae.WithDetails(map[string]any{
				"errcode": apiErr.ErrCode,
				"errmsg":  apiErr.ErrMsg,
				"hint":    hint,
			})
		}
		return ae
	default:
		return output.Wrap(err, output.CodeInternalError)
	}
}

// firstNonEmpty 返回第一个非空（去空白后非空）字符串。
func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
