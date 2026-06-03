package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// DefaultFileContent 返回一个带注释的默认配置文件内容，用于 init。
//
// 注释帮助用户（和 Agent）理解每个字段，敏感字段留空等待填写。
func DefaultFileContent() string {
	return `# thinkthinking 配置文件
# 路径优先级（高→低）：CLI flags > 环境变量 > 项目配置 > 用户配置 > 默认值
#
# 微信公众号凭证也可通过环境变量提供（优先级高于本文件）：
#   WECHAT_APP_ID / WECHAT_APP_SECRET / WECHAT_ACCESS_TOKEN

wechat:
  app_id: ""              # 公众号 AppID
  app_secret: ""          # 公众号 AppSecret（敏感）
  access_token: ""        # 可选：直接提供 access_token（敏感）
  default_author: "thinkthinking"
  default_theme: "default"

output:
  pretty: false           # 默认是否美化 JSON 输出
  quiet: false            # 默认是否抑制非必要 stderr 输出
`
}

// CreateFile 在 path 写入默认配置文件。
//
//   - 自动创建父目录（0700，配置含敏感信息）。
//   - 若文件已存在且 overwrite=false，返回 (created=false, nil)，不覆盖。
//
// 返回 created 表示是否实际写入了新文件。
func CreateFile(path string, overwrite bool) (bool, error) {
	if FileExists(path) && !overwrite {
		return false, nil
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return false, fmt.Errorf("create config dir %s: %w", dir, err)
	}

	if err := os.WriteFile(path, []byte(DefaultFileContent()), 0o600); err != nil {
		return false, fmt.Errorf("write config file %s: %w", path, err)
	}
	return true, nil
}

// SetValue 在指定配置文件中设置单个点路径键的值，保留文件其余内容。
//
// 实现策略：读现有 YAML 到 map → 按点路径设置 → 写回。若文件不存在则先创建。
// 这样 config set 不会丢失用户已有的其它配置或破坏结构。
func SetValue(path, key, value string) error {
	root := map[string]any{}

	if FileExists(path) {
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read config %s: %w", path, err)
		}
		if err := yaml.Unmarshal(data, &root); err != nil {
			return fmt.Errorf("parse config %s: %w", path, err)
		}
		if root == nil {
			root = map[string]any{}
		}
	} else {
		// 确保父目录存在。
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			return fmt.Errorf("create config dir: %w", err)
		}
	}

	setNested(root, strings.Split(key, "."), value)

	out, err := yaml.Marshal(root)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	if err := os.WriteFile(path, out, 0o600); err != nil {
		return fmt.Errorf("write config %s: %w", path, err)
	}
	return nil
}

// setNested 按 keys 路径在嵌套 map 中设置 value，沿途创建缺失的子 map。
func setNested(m map[string]any, keys []string, value string) {
	for i, k := range keys {
		if i == len(keys)-1 {
			m[k] = value
			return
		}
		child, ok := m[k].(map[string]any)
		if !ok {
			child = map[string]any{}
			m[k] = child
		}
		m = child
	}
}

// MaskSensitive 对敏感值脱敏：长度 >8 显示首尾各 4 位，否则统一 "********"。
// 借鉴 md2wechat-lite 的 maskSensitive，但短值返回固定长度掩码，避免泄露长度。
func MaskSensitive(s string) string {
	if s == "" {
		return ""
	}
	if len(s) <= 8 {
		return "********"
	}
	return s[:4] + "********" + s[len(s)-4:]
}
