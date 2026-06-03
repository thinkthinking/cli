package config

// Defaults 返回内置默认配置。它是优先级链的最底层。
//
// default_author / default_theme 的默认值对应 init.md 的配置示例。
func Defaults() Config {
	return Config{
		WeChat: WeChatConfig{
			DefaultAuthor: "thinkthinking",
			DefaultTheme:  "default",
		},
		Output: OutputConfig{
			Pretty: false,
			Quiet:  false,
		},
	}
}

// defaultsMap 以 koanf 点路径形式返回默认值，供 loader 作为最底层 provider 注入。
func defaultsMap() map[string]any {
	d := Defaults()
	return map[string]any{
		"wechat.default_author": d.WeChat.DefaultAuthor,
		"wechat.default_theme":  d.WeChat.DefaultTheme,
		"output.pretty":         d.Output.Pretty,
		"output.quiet":          d.Output.Quiet,
	}
}
