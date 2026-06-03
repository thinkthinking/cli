package config

import (
	"fmt"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

// envMap 把已知环境变量名映射到 koanf 点路径键。
//
// 这是刻意写成显式白名单（而非通用前缀转换）：对 Agent 友好的工具，
// 环境变量的语义应当确定、可预测，而不是隐式地把任意 env 灌进配置。
var envMap = map[string]string{
	"WECHAT_APP_ID":       "wechat.app_id",
	"WECHAT_APP_SECRET":   "wechat.app_secret",
	"WECHAT_ACCESS_TOKEN": "wechat.access_token",
	"WECHAT_AUTHOR":       "wechat.default_author",
	"WECHAT_THEME":        "wechat.default_theme",
}

// LoadResult 描述一次配置加载的结果，包含最终配置与诊断信息（实际加载了哪些文件）。
type LoadResult struct {
	Config      *Config
	K           *koanf.Koanf
	LoadedFiles []string // 实际存在并被加载的配置文件路径（按优先级低→高）
	UserPath    string   // 用户配置文件路径（无论是否存在）
	ProjectPath string   // 项目配置文件路径（无论是否存在）
}

// LoadOptions 控制加载行为。
type LoadOptions struct {
	// ExplicitPath 对应 --config：若非空，只加载这个文件（跳过默认搜索）。
	ExplicitPath string
}

// Load 按优先级链加载配置：defaults → 用户配置 → 项目配置 → 环境变量。
//
// 注意：CLI flags 是最高优先级，但 flags 与命令强相关，由各命令在读取配置后
// 自行覆盖（flag 值非空则优先），不在此处理。
func Load(opts LoadOptions) (*LoadResult, error) {
	k := koanf.New(".")

	// 1. 最底层：内置默认值。
	if err := k.Load(confmap.Provider(defaultsMap(), "."), nil); err != nil {
		return nil, fmt.Errorf("load defaults: %w", err)
	}

	result := &LoadResult{K: k}

	// 解析候选文件路径。
	var candidates []string
	if opts.ExplicitPath != "" {
		// --config 指定：只用这一个文件，且必须存在（否则报错，避免静默忽略用户意图）。
		if !FileExists(opts.ExplicitPath) {
			return nil, fmt.Errorf("config file not found: %s", opts.ExplicitPath)
		}
		candidates = []string{opts.ExplicitPath}
		result.UserPath = opts.ExplicitPath
	} else {
		userPath, err := UserConfigPath()
		if err != nil {
			return nil, fmt.Errorf("resolve user config path: %w", err)
		}
		result.UserPath = userPath
		result.ProjectPath = ProjectConfigPath()
		// 优先级低→高：用户配置 → 项目配置。后加载的覆盖先加载的。
		candidates = []string{userPath, result.ProjectPath}
	}

	// 2. 加载文件层。
	for _, path := range candidates {
		if !FileExists(path) {
			continue
		}
		if err := k.Load(file.Provider(path), yaml.Parser()); err != nil {
			return nil, fmt.Errorf("load config %s: %w", path, err)
		}
		result.LoadedFiles = append(result.LoadedFiles, path)
	}

	// 3. 最高层（文件之上）：环境变量。用 ProviderWithValue 以便同时按
	//    键映射 + 按值过滤：只接受白名单里的变量，且空值不覆盖下层配置
	//    （否则 WECHAT_APP_ID="" 会把默认值/文件值清空）。
	//    回调返回空 key 表示丢弃该变量。
	envProvider := env.ProviderWithValue("", ".", func(name, value string) (string, any) {
		key, ok := envMap[name]
		if !ok || value == "" {
			return "", nil
		}
		return key, value
	})
	if err := k.Load(envProvider, nil); err != nil {
		return nil, fmt.Errorf("load env: %w", err)
	}

	// 解组到结构体。
	var cfg Config
	if err := k.UnmarshalWithConf("", &cfg, koanf.UnmarshalConf{Tag: "koanf"}); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}
	result.Config = &cfg

	return result, nil
}

// FlatMap 返回当前 koanf 实例的扁平点路径键值对，供 config list 使用。
// 敏感字段由调用方负责脱敏。
func (r *LoadResult) FlatMap() map[string]any {
	return r.K.All()
}
