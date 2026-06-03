package cli

import (
	"github.com/spf13/cobra"
)

// newWeChatCmd 是 `thinkthinking wechat` 父命令，聚合微信公众号相关子命令。
func newWeChatCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "wechat",
		Short: "微信公众号能力：convert / post",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SetOut(cmd.ErrOrStderr())
			return cmd.Help()
		},
	}
	cmd.AddCommand(newWeChatConvertCmd())
	cmd.AddCommand(newWeChatPostCmd())
	return cmd
}
