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

// newWeChatDraftCmd 是 `thinkthinking wechat draft` 父命令。
func newWeChatDraftCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "draft",
		Short: "微信公众号草稿：create",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SetOut(cmd.ErrOrStderr())
			return cmd.Help()
		},
	}
	cmd.AddCommand(newWeChatDraftCreateCmd())
	return cmd
}

// newWeChatDraftCreateCmd 实现 `thinkthinking wechat draft create`。
func newWeChatDraftCreateCmd() *cobra.Command {
	var (
		title        string
		markdownFile string
		htmlFile     string
		theme        string
		author       string
		digest       string
		cover        string
		coverMediaID string
		noUpload     bool
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "创建微信公众号草稿（支持 Markdown 自动转换与本地图片上传）",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			// title 必填。
			if strings.TrimSpace(title) == "" {
				return output.New(output.CodeInvalidInput, "--title is required")
			}
			// markdown-file 与 html-file 二选一。
			if (markdownFile == "") == (htmlFile == "") {
				return output.New(output.CodeInvalidInput, "exactly one of --markdown-file or --html-file is required")
			}

			// 凭证预检：缺失则返回结构化 WECHAT_AUTH_ERROR（不 panic）。
			if aerr := checkWeChatCredentials(); aerr != nil {
				return aerr
			}

			input := wechat.CreateDraftInput{
				Title:        title,
				Author:       firstNonEmpty(author, container.Config.WeChat.DefaultAuthor),
				Digest:       digest,
				CoverImage:   cover,
				CoverMediaID: coverMediaID,
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
	f.StringVar(&markdownFile, "markdown-file", "", "Markdown 文件路径（与 --html-file 二选一）")
	f.StringVar(&htmlFile, "html-file", "", "HTML 文件路径（与 --markdown-file 二选一）")
	f.StringVarP(&theme, "theme", "t", "", "主题名（仅 markdown 路径，默认读配置）")
	f.StringVar(&author, "author", "", "作者（默认读配置 wechat.default_author）")
	f.StringVar(&digest, "digest", "", "摘要（默认用转换生成的摘要）")
	f.StringVar(&cover, "cover", "", "本地封面图路径（上传为 thumb_media_id）")
	f.StringVar(&coverMediaID, "cover-media-id", "", "已有的封面 media_id（与 --cover 二选一）")
	f.BoolVar(&noUpload, "no-upload-images", false, "禁用正文本地图片自动上传")
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

// firstNonEmpty 返回第一个非空字符串。
func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
