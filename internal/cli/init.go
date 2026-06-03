package cli

import (
	"github.com/spf13/cobra"
	"github.com/thinkthinking/cli/internal/config"
	"github.com/thinkthinking/cli/internal/core/output"
)

// newInitCmd 实现 `thinkthinking init [--local]`，创建配置文件。
func newInitCmd() *cobra.Command {
	var local bool
	var force bool

	cmd := &cobra.Command{
		Use:   "init",
		Short: "初始化配置文件（默认用户级 ~/.thinkthinking/config.yaml）",
		Long: `创建配置文件。

默认创建用户级配置：~/.thinkthinking/config.yaml
使用 --local 创建项目级配置：./.thinkthinking/config.yaml`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := resolveInitPath(local)
			if err != nil {
				return output.New(output.CodeConfigError, err.Error())
			}

			created, err := config.CreateFile(path, force)
			if err != nil {
				return output.Wrap(err, output.CodeConfigError)
			}

			data := map[string]any{
				"path":    path,
				"created": created,
			}
			if created {
				data["created_files"] = []string{path}
			} else {
				data["created_files"] = []string{}
				data["note"] = "config file already exists; use --force to overwrite"
			}
			return container.Output.Success(data)
		},
	}

	cmd.Flags().BoolVar(&local, "local", false, "创建项目级配置 ./.thinkthinking/config.yaml")
	cmd.Flags().BoolVar(&force, "force", false, "若已存在则覆盖")
	return cmd
}

// resolveInitPath 根据 --local 返回应创建的配置文件路径。
func resolveInitPath(local bool) (string, error) {
	if local {
		return config.ProjectConfigPath(), nil
	}
	return config.UserConfigPath()
}
