// Command thinkthinking 是 CLI 的可执行入口。
//
// 它只做一件事：把控制权交给 internal/cli，并用其返回的退出码退出。
// 所有逻辑都在 internal 包内，便于测试与未来复用。
package main

import (
	"os"

	"github.com/thinkthinking/cli/internal/cli"
)

func main() {
	os.Exit(cli.Execute())
}
