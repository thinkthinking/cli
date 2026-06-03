// Package config 负责加载、合并、读写 thinkthinking 配置。
//
// 配置文件固定为 ~/.thinkthinking/config.yaml（用户级）与
// ./.thinkthinking/config.yaml（项目级）。优先级（高→低）：
//
//	CLI flags > 环境变量 > 项目配置 > 用户配置 > 内置默认值
//
// 敏感字段（app_secret / access_token）在 List 输出时脱敏。
package config

// WeChatConfig 是微信公众号相关配置。
type WeChatConfig struct {
	AppID         string `koanf:"app_id" yaml:"app_id" json:"app_id"`
	AppSecret     string `koanf:"app_secret" yaml:"app_secret" json:"app_secret"`
	AccessToken   string `koanf:"access_token" yaml:"access_token" json:"access_token"`
	DefaultAuthor string `koanf:"default_author" yaml:"default_author" json:"default_author"`
	DefaultTheme  string `koanf:"default_theme" yaml:"default_theme" json:"default_theme"`
}

// OutputConfig 控制默认输出行为（可被对应 flag 覆盖）。
type OutputConfig struct {
	Pretty bool `koanf:"pretty" yaml:"pretty" json:"pretty"`
	Quiet  bool `koanf:"quiet" yaml:"quiet" json:"quiet"`
}

// Config 是完整配置树。
type Config struct {
	WeChat WeChatConfig `koanf:"wechat" yaml:"wechat" json:"wechat"`
	Output OutputConfig `koanf:"output" yaml:"output" json:"output"`
}

// sensitiveKeys 是 config list 时需要脱敏的点路径键集合。
var sensitiveKeys = map[string]bool{
	"wechat.app_secret":   true,
	"wechat.access_token": true,
}

// IsSensitiveKey 报告给定点路径键是否为敏感字段。
func IsSensitiveKey(key string) bool {
	return sensitiveKeys[key]
}
