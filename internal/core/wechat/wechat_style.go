package wechat

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/aymerick/douceur/css"
	"github.com/aymerick/douceur/parser"
	"gopkg.in/yaml.v3"
)

// Theme 是一个主题定义，移植自 wewrite theme.py。
//
// colors 用 map[string]any 而非专门 struct：主题 key 不固定（impeccable 有
// highlight_bg/summary_bg，其它没有），且 var(--xxx) 是动态解析，map 远比
// 每加字段就改 struct 灵活。darkmode 可能在 colors 内或顶层，两处都读。
type Theme struct {
	Name        string         `yaml:"name"`
	Description string         `yaml:"description"`
	BaseCSS     string         `yaml:"base_css"`
	Colors      map[string]any `yaml:"colors"`

	// 顶层可选字段（impeccable 风格主题用）。
	TopDarkmode  map[string]any `yaml:"darkmode"`
	CSSRandomize bool           `yaml:"css_randomize"`
	AIGCFooter   bool           `yaml:"aigc_footer"`
}

// Color 返回 colors 里的字符串值，缺失时返回 fallback。
func (t *Theme) Color(key, fallback string) string {
	if v, ok := t.Colors[key]; ok {
		if s, ok := v.(string); ok && s != "" {
			return s
		}
	}
	return fallback
}

// Darkmode 返回暗黑模式配色 map。优先 colors.darkmode（wewrite 主流写法），
// 回退顶层 darkmode（midnight/newspaper 写法）。两处都没有则返回 nil。
func (t *Theme) Darkmode() map[string]any {
	if v, ok := t.Colors["darkmode"]; ok {
		if m, ok := v.(map[string]any); ok {
			return m
		}
	}
	return t.TopDarkmode
}

// darkColor 从 darkmode map 取字符串值，带 fallback。
func darkColor(dm map[string]any, key, fallback string) string {
	if dm == nil {
		return fallback
	}
	if v, ok := dm[key]; ok {
		if s, ok := v.(string); ok && s != "" {
			return s
		}
	}
	return fallback
}

// cssRule 是一条解析后的 CSS 规则：选择器 + 有序属性。
// 用有序 slice 而非 map，复刻 Python dict 保序语义，保证输出确定性。
type cssRule struct {
	selector string
	props    []cssProp
}

type cssProp struct {
	name  string
	value string
}

// ThemeManager 负责加载主题（内嵌 + 用户/项目覆盖目录）。
type ThemeManager struct {
	// overrideDirs 是覆盖目录，优先级高于内嵌主题，按传入顺序后者覆盖前者。
	overrideDirs []string
}

// NewThemeManager 创建主题管理器。overrideDirs 通常为
// [~/.thinkthinking/themes, ./.thinkthinking/themes]。
func NewThemeManager(overrideDirs ...string) *ThemeManager {
	return &ThemeManager{overrideDirs: overrideDirs}
}

// Load 按名加载主题。先查覆盖目录（后者优先），再回退内嵌。
func (tm *ThemeManager) Load(name string) (*Theme, error) {
	if name == "" {
		name = "default"
	}

	// 覆盖目录优先（逆序遍历，让靠后的目录优先级更高）。
	for i := len(tm.overrideDirs) - 1; i >= 0; i-- {
		path := filepath.Join(tm.overrideDirs[i], name+".yaml")
		if data, err := os.ReadFile(path); err == nil {
			return parseTheme(data)
		}
	}

	// 回退内嵌主题。
	data, err := builtinThemesFS.ReadFile("themes/" + name + ".yaml")
	if err != nil {
		return nil, &ThemeNotFoundError{Name: name}
	}
	return parseTheme(data)
}

// List 返回所有可用主题名（内嵌 + 覆盖目录），去重排序。
func (tm *ThemeManager) List() []string {
	seen := map[string]bool{}

	entries, _ := fs.ReadDir(builtinThemesFS, "themes")
	for _, e := range entries {
		if name, ok := themeNameFromFile(e.Name()); ok {
			seen[name] = true
		}
	}
	for _, dir := range tm.overrideDirs {
		files, _ := os.ReadDir(dir)
		for _, f := range files {
			if name, ok := themeNameFromFile(f.Name()); ok {
				seen[name] = true
			}
		}
	}

	names := make([]string, 0, len(seen))
	for n := range seen {
		names = append(names, n)
	}
	sortStrings(names)
	return names
}

// themeNameFromFile 从文件名提取主题名（去 .yaml/.yml 后缀）。
func themeNameFromFile(filename string) (string, bool) {
	for _, ext := range []string{".yaml", ".yml"} {
		if strings.HasSuffix(filename, ext) {
			return strings.TrimSuffix(filename, ext), true
		}
	}
	return "", false
}

// parseTheme 解析主题 YAML。
func parseTheme(data []byte) (*Theme, error) {
	var t Theme
	if err := yaml.Unmarshal(data, &t); err != nil {
		return nil, fmt.Errorf("parse theme yaml: %w", err)
	}
	if t.Name == "" {
		return nil, fmt.Errorf("theme missing required field: name")
	}
	if t.Colors == nil {
		t.Colors = map[string]any{}
	}
	return &t, nil
}

// cssVarPattern 匹配 var(--xxx) 引用。
var cssVarPattern = regexp.MustCompile(`var\(\s*--([a-zA-Z0-9_-]+)\s*\)`)

// resolveCSSVariables 把 base_css 里的 var(--primary) 替换成 colors 里的值。
// 支持连字符转下划线的 key 匹配（var(--text-light) → colors.text_light）。
func (t *Theme) resolveCSSVariables(css string) string {
	return cssVarPattern.ReplaceAllStringFunc(css, func(match string) string {
		sub := cssVarPattern.FindStringSubmatch(match)
		if len(sub) < 2 {
			return match
		}
		key := strings.TrimPrefix(sub[1], "--")
		if v, ok := t.Colors[key]; ok {
			if s, ok := v.(string); ok {
				return s
			}
		}
		keyUnderscore := strings.ReplaceAll(key, "-", "_")
		if v, ok := t.Colors[keyUnderscore]; ok {
			if s, ok := v.(string); ok {
				return s
			}
		}
		return match // 未找到则保留原样
	})
}

// inlineCSSRules 把主题 base_css 解析成有序的 cssRule 列表。
//
// 先解析 var(--xxx) 变量，再用 douceur 解析 CSS，只保留简单选择器
// （拒绝伪类/媒体查询/复杂组合器），对齐 theme.py 的 _is_simple_selector。
func (t *Theme) inlineCSSRules() ([]cssRule, error) {
	resolved := t.resolveCSSVariables(t.BaseCSS)

	sheet, err := parser.Parse(resolved)
	if err != nil {
		return nil, fmt.Errorf("parse theme css: %w", err)
	}

	var rules []cssRule
	for _, rule := range sheet.Rules {
		// 只处理普通样式规则（跳过 @media 等 at-rule）。
		if rule.Kind != css.QualifiedRule {
			continue
		}
		// 收集属性（有序）。
		props := make([]cssProp, 0, len(rule.Declarations))
		for _, decl := range rule.Declarations {
			props = append(props, cssProp{name: decl.Property, value: decl.Value})
		}
		if len(props) == 0 {
			continue
		}
		// 一条规则可能有多个逗号分隔的选择器。
		for _, sel := range rule.Selectors {
			sel = strings.TrimSpace(sel)
			if !isSimpleSelector(sel) {
				continue
			}
			rules = append(rules, cssRule{selector: sel, props: props})
		}
	}
	return rules, nil
}

// isSimpleSelector 判断选择器是否足够简单可用于内联。
// 拒绝伪类(:)、at-rule(@)、子/兄弟组合器(> + ~)、属性([)、通配(*)。
// 后代选择器（空格，如 "blockquote p"）保留——goquery 支持且 wewrite 也保留。
func isSimpleSelector(sel string) bool {
	sel = strings.TrimSpace(sel)
	if sel == "" {
		return false
	}
	for _, ch := range []string{":", "@", ">", "+", "~", "[", "*"} {
		if strings.Contains(sel, ch) {
			return false
		}
	}
	return true
}
