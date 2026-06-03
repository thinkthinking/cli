// Package version 持有构建期注入的版本信息。
//
// 这些变量由 GoReleaser 通过 -ldflags "-X ..." 注入（见 .goreleaser.yaml）。
// 本地 go run / go build 时保持默认占位值。
package version

// 构建期可注入的变量。注意：必须是 var 而非 const，ldflags 才能覆盖。
var (
	// Name 是 CLI 名称。
	Name = "thinkthinking"
	// Version 是语义化版本号。
	Version = "0.1.0"
	// Commit 是构建时的 git commit short hash。
	Commit = "dev"
	// Date 是构建时间（RFC3339）。
	Date = "unknown"
)

// Info 是 version 命令输出的结构。
type Info struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Commit  string `json:"commit"`
	Date    string `json:"date"`
}

// Get 返回当前构建的版本信息。
func Get() Info {
	return Info{
		Name:    Name,
		Version: Version,
		Commit:  Commit,
		Date:    Date,
	}
}
