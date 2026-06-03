package cli

import (
	"github.com/spf13/cobra"
	"github.com/thinkthinking/cli/internal/core/output"
	"github.com/thinkthinking/cli/internal/version"
)

// newVersionCmd 实现 `thinkthinking version`，输出版本信息 JSON。
func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "输出版本信息（JSON）",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			// 演示标准模式：从 container 取 OutputWriter，调 core 拿数据，输出。
			if err := container.Output.Success(version.Get()); err != nil {
				return output.Wrap(err, output.CodeInternalError)
			}
			return nil
		},
	}
}
