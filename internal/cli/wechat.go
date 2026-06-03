package cli

import (
	"github.com/spf13/cobra"
)

// newWeChatCmd 是 `thinkthinking wechat` 父命令，聚合微信公众号相关子命令。
func newWeChatCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "wechat",
		Short: "微信公众号能力：convert（转 HTML） / post（发布草稿）",
		Long: `微信公众号能力。两个子命令：

  convert  Markdown → 微信公众号兼容 HTML（不联网，纯转换）
  post     把一篇文章发布为草稿（联网：转换 → 上传图片 → 上传封面 → 写草稿箱）

前置：需先配置凭证（仅 post 联网时需要）：
  thinkthinking config set wechat.app_id     <appid>
  thinkthinking config set wechat.app_secret <secret>
  # 或用环境变量 WECHAT_APP_ID / WECHAT_APP_SECRET

------------------------------------------------------------------
convert —— Markdown 转 HTML
------------------------------------------------------------------
  thinkthinking wechat convert <file>              # 转换，HTML 在 data.html
  thinkthinking wechat convert <file> -t midnight  # 指定主题
  thinkthinking wechat convert <file> --copy       # 写富文本剪贴板（macOS）
  thinkthinking wechat convert <file> --preview    # 打开浏览器预览页
  thinkthinking wechat convert <file> -o out.html  # 写文件
  thinkthinking wechat convert --stdin < file.md   # 从 stdin 读
  内置主题：default / minimal / midnight / newspaper / tech-modern
  关闭增强：--no-darkmode / --no-footnotes / --no-containers

------------------------------------------------------------------
post —— 发布草稿（必填 --title / --author / --cover）
------------------------------------------------------------------
  thinkthinking wechat post article.md \
    --title "标题" --author "作者" --cover cover.jpg
  thinkthinking wechat post article.html \          # .html 直传，不转换
    --title "标题" --author "作者" --cover cover.jpg
  正文文件按后缀自动判断：.md/.markdown 转换；.html/.htm 直传。
  本地图片自动上传并回填 URL（--no-upload-images 可关）。

各子命令更多细节见 ` + "`thinkthinking wechat <cmd> -h`" + `。`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SetOut(cmd.ErrOrStderr())
			return cmd.Help()
		},
	}
	cmd.AddCommand(newWeChatConvertCmd())
	cmd.AddCommand(newWeChatPostCmd())
	return cmd
}
