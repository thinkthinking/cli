// Package cli 实现命令行入口层。
//
// 职责边界（来自 init.md）：CLI command 只负责 (1) 解析参数、(2) 从 app.Container
// 取 service 并调用、(3) 通过 OutputWriter 输出 JSON。不在此层写业务逻辑。
package cli

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/thinkthinking/cli/internal/app"
	"github.com/thinkthinking/cli/internal/core/output"
)

// 全局 flag 的值容器。cobra 在 parse 后填充，PersistentPreRun 据此装配 container。
type globalFlags struct {
	configPath string
	pretty     bool
	quiet      bool
	noColor    bool
	traceID    string
	verbose    bool
}

// container 是装配好的服务集合，供各子命令通过 currentContainer() 获取。
// 使用包级变量是 cobra 的常见模式：PersistentPreRun 设置，子命令 Run 读取。
var (
	gFlags    globalFlags
	container *app.Container
)

// NewRootCmd 构建根命令并挂载全局 flags。
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "thinkthinking",
		Short: "thinkthinking — 面向 Agent 的个人 CLI 工具箱",
		Long: `thinkthinking 是一个面向大语言模型 Agent、自动化脚本和开发者的本地 CLI 工具箱。

第一期聚焦微信公众号能力：将 Markdown 转换为微信公众号兼容 HTML，并上传草稿。
所有命令默认输出统一 JSON envelope，便于 Agent 稳定解析。`,
		// 我们自行用 JSON envelope 输出错误，禁用 cobra 的默认错误/用法打印，
		// 避免非 JSON 文本污染 stdout。
		SilenceErrors: true,
		SilenceUsage:  true,
		// 根命令无子命令时打印帮助。关键：强制把帮助写到 stderr，保持 stdout
		// 纯净（Agent 契约：stdout 只允许 JSON envelope）。cobra 默认 Help 走 stdout，
		// 这里显式重定向。
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SetOut(cmd.ErrOrStderr())
			return cmd.Help()
		},
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			container = app.New(app.Options{
				ConfigPath: gFlags.configPath,
				Pretty:     gFlags.pretty,
				Quiet:      gFlags.quiet,
				Verbose:    gFlags.verbose,
				Stdout:     cmd.OutOrStdout(),
				Stderr:     cmd.ErrOrStderr(),
			})
		},
	}

	// 全局持久化 flags（对所有子命令生效）。
	pf := root.PersistentFlags()
	pf.StringVar(&gFlags.configPath, "config", "", "指定配置文件路径（覆盖默认搜索）")
	pf.BoolVar(&gFlags.pretty, "pretty", false, "美化 JSON 输出")
	pf.BoolVar(&gFlags.quiet, "quiet", false, "抑制非必要的 stderr 输出")
	pf.BoolVar(&gFlags.noColor, "no-color", false, "禁用彩色输出（预留，第一期默认无颜色）")
	pf.StringVar(&gFlags.traceID, "trace-id", "", "调用方追踪 ID，透传到日志便于排查")
	pf.BoolVarP(&gFlags.verbose, "verbose", "v", false, "输出 debug 级别日志到 stderr")

	// 挂载子命令。
	root.AddCommand(newVersionCmd())
	root.AddCommand(newInitCmd())
	root.AddCommand(newConfigCmd())
	root.AddCommand(newWeChatCmd())

	return root
}

// Execute 是 main 的唯一入口。它构建根命令并执行，把错误统一转成 JSON envelope。
func Execute() int {
	root := NewRootCmd()
	if err := root.Execute(); err != nil {
		// 走到这里说明子命令返回了未处理的 error。容器可能尚未装配
		// （例如 flag 解析阶段就失败），因此兜底直接写一个 JSON 错误到 stdout。
		return handleExecuteError(err)
	}
	return 0
}

// handleExecuteError 把执行期错误输出为统一 JSON envelope。
func handleExecuteError(err error) int {
	w := output.NewJSONWriter(os.Stdout, gFlags.pretty)
	_ = w.Error(output.Wrap(err, output.CodeInvalidInput))
	return 1
}
