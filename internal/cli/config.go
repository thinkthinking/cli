package cli

import (
	"strings"

	"github.com/spf13/cobra"
	"github.com/thinkthinking/cli/internal/config"
	"github.com/thinkthinking/cli/internal/core/output"
)

// newConfigCmd 实现 `thinkthinking config path|get|set|list`。
func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "管理配置：path / get / set / list",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SetOut(cmd.ErrOrStderr())
			return cmd.Help()
		},
	}
	cmd.AddCommand(newConfigPathCmd(), newConfigGetCmd(), newConfigSetCmd(), newConfigListCmd())
	return cmd
}

// newConfigPathCmd 输出用户配置文件路径（无论是否存在）。
func newConfigPathCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "path",
		Short: "输出用户配置文件路径",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := config.UserConfigPath()
			if err != nil {
				return output.Wrap(err, output.CodeConfigError)
			}
			data := map[string]any{
				"path":   path,
				"exists": config.FileExists(path),
			}
			if container.ConfigLoad != nil && len(container.ConfigLoad.LoadedFiles) > 0 {
				data["loaded_files"] = container.ConfigLoad.LoadedFiles
			}
			return container.Output.Success(data)
		},
	}
}

// newConfigGetCmd 读取单个配置项。
func newConfigGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <key>",
		Short: "读取配置项，例如 wechat.app_id",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := strings.TrimSpace(args[0])
			if key == "" {
				return output.New(output.CodeInvalidInput, "key is required")
			}
			if container.ConfigErr != nil {
				return output.Wrap(container.ConfigErr, output.CodeConfigError)
			}

			val := container.ConfigLoad.K.Get(key)
			return container.Output.Success(map[string]any{
				"key":   key,
				"value": val,
			})
		},
	}
}

// newConfigSetCmd 写入单个配置项到目标文件。
func newConfigSetCmd() *cobra.Command {
	var local bool
	cmd := &cobra.Command{
		Use:   "set <key> <value>",
		Short: "设置配置项，例如 config set wechat.app_id wx123",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := strings.TrimSpace(args[0])
			value := args[1]
			if key == "" {
				return output.New(output.CodeInvalidInput, "key is required")
			}

			path, err := resolveInitPath(local)
			if err != nil {
				return output.New(output.CodeConfigError, err.Error())
			}
			if err := config.SetValue(path, key, value); err != nil {
				return output.Wrap(err, output.CodeConfigError)
			}
			return container.Output.Success(map[string]any{
				"key":     key,
				"updated": true,
				"path":    path,
			})
		},
	}
	cmd.Flags().BoolVar(&local, "local", false, "写入项目级配置而非用户级")
	return cmd
}

// newConfigListCmd 输出完整配置，敏感字段脱敏。
func newConfigListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "输出完整配置（敏感字段脱敏）",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if container.ConfigErr != nil {
				return output.Wrap(container.ConfigErr, output.CodeConfigError)
			}

			// 以结构化（嵌套）形式输出，敏感字段脱敏。
			cfg := container.Config
			data := map[string]any{
				"wechat": map[string]any{
					"app_id":         cfg.WeChat.AppID,
					"app_secret":     config.MaskSensitive(cfg.WeChat.AppSecret),
					"access_token":   config.MaskSensitive(cfg.WeChat.AccessToken),
					"default_author": cfg.WeChat.DefaultAuthor,
					"default_theme":  cfg.WeChat.DefaultTheme,
				},
				"output": map[string]any{
					"pretty": cfg.Output.Pretty,
					"quiet":  cfg.Output.Quiet,
				},
			}
			return container.Output.Success(data)
		},
	}
}
