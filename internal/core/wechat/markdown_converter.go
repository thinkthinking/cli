package wechat

import (
	"bytes"
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/PuerkitoBio/goquery"
	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer/html"
	gmhtml "golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// converter 是 MarkdownConverter 的实现，移植自 wewrite converter.py。
type converter struct {
	themes *ThemeManager
	md     goldmark.Markdown
}

// NewConverter 创建一个转换器。overrideDirs 为主题覆盖目录。
func NewConverter(overrideDirs ...string) MarkdownConverter {
	md := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM, // 表格、删除线、任务列表、自动链接
			highlighting.NewHighlighting(
				highlighting.WithStyle("github"),
				// data-lang：用 WrapperRenderer 从语言名写 <pre data-lang>。
				highlighting.WithWrapperRenderer(wrapPreWithLang),
				highlighting.WithFormatOptions(
					// 关键：不输出 CSS class，改为 inline style 高亮，
					// 对应 wewrite codehilite noclasses=True。
					chromahtml.WithClasses(false),
					// 抑制 chroma 自带的 <pre>，由 WrapperRenderer 接管，
					// 否则会出现双层 <pre>。
					chromahtml.WithPreWrapper(emptyPreWrapper{}),
				),
			),
		),
		goldmark.WithRendererOptions(
			html.WithHardWraps(), // 对应 nl2br：单换行渲染成 <br>
			html.WithUnsafe(),    // 让容器块预处理生成的 raw HTML 能渲染
		),
	)

	return &converter{
		themes: NewThemeManager(overrideDirs...),
		md:     md,
	}
}

// Convert 执行 15 步 pipeline，顺序严格对齐 wewrite converter.py。
func (c *converter) Convert(ctx context.Context, input ConvertInput) (*ConvertResult, error) {
	theme, err := c.themes.Load(input.ThemeName)
	if err != nil {
		return nil, err
	}
	opts := input.Options
	warnings := []string{}

	text := input.Markdown

	// 1-2. 提取并剔除 H1（微信有独立 title 字段）。
	title := extractTitle(text)
	text = stripH1(text)

	// 3. 容器块预处理（在 Markdown 解析前）。
	if opts.Containers {
		text = c.preprocessContainers(theme, text)
	}

	// 4. CJK 与拉丁字符间插空格。
	text = fixCJKSpacing(text)

	// 5. Markdown → HTML。
	htmlStr, err := c.markdownToHTML(text)
	if err != nil {
		return nil, err
	}

	// 6. data-lang 已由 WrapperRenderer 处理，此处 no-op（保留步骤注释对齐）。

	// 7. 图片：收集 src + 补响应式 style。
	htmlStr, images := processImages(htmlStr)

	// 8. 粗体内中文标点外移。
	htmlStr = fixCJKBoldPunctuation(htmlStr)

	// 9. 列表转 section。
	htmlStr, err = convertListsToSections(htmlStr, theme)
	if err != nil {
		return nil, err
	}

	// 10. 外链转脚注。
	if opts.Footnotes {
		htmlStr, err = convertLinksToFootnotes(htmlStr, theme)
		if err != nil {
			return nil, err
		}
	}

	// 11. 主题 CSS 内联。
	rules, rerr := theme.inlineCSSRules()
	if rerr != nil {
		warnings = append(warnings, "theme css parse: "+rerr.Error())
	} else {
		htmlStr, err = applyInlineStyles(htmlStr, rules)
		if err != nil {
			return nil, err
		}
	}

	// 12. 微信兼容修复（p 强制 color、pre 强制 white-space）。
	htmlStr, err = applyWeChatFixes(htmlStr, theme)
	if err != nil {
		return nil, err
	}

	// 13. 暗黑模式属性注入。
	if opts.DarkMode {
		htmlStr, err = injectDarkmode(htmlStr, theme)
		if err != nil {
			return nil, err
		}
	}

	// 14. 反指纹随机微扰（默认关）。
	if opts.AntiFingerprint || theme.CSSRandomize {
		htmlStr = randomizeCSS(htmlStr)
	}

	// 15. AIGC footer（主题开关或显式覆盖）。
	if theme.AIGCFooter || (opts.aigcFooterSet && opts.AIGCFooter) {
		htmlStr = appendAIGCFooter(htmlStr)
	}

	// 16. 生成 digest（120 UTF-8 字节）。
	digest := generateDigest(htmlStr, 120)

	// images/warnings 用非 nil 空切片，保证 JSON 输出为 [] 而非 null，
	// 让 Agent 可以无条件 len()/遍历。
	if images == nil {
		images = []string{}
	}

	return &ConvertResult{
		HTML:      htmlStr,
		Title:     title,
		Digest:    digest,
		Images:    images,
		ThemeName: theme.Name,
		Warnings:  warnings,
	}, nil
}

// markdownToHTML 用 goldmark 把 Markdown 转成 HTML 字符串。
func (c *converter) markdownToHTML(text string) (string, error) {
	var buf bytes.Buffer
	if err := c.md.Convert([]byte(text), &buf); err != nil {
		return "", fmt.Errorf("markdown convert: %w", err)
	}
	return buf.String(), nil
}

// ---- 步骤 1-2：标题 ----

var h1LinePattern = regexp.MustCompile(`(?m)^#\s+(.+?)\s*$`)

// extractTitle 提取第一个 H1（# 开头但非 ##）。
func extractTitle(text string) string {
	for _, line := range strings.Split(text, "\n") {
		s := strings.TrimSpace(line)
		if strings.HasPrefix(s, "# ") && !strings.HasPrefix(s, "## ") {
			return strings.TrimSpace(s[2:])
		}
	}
	return ""
}

// stripH1 删除所有 H1 行。
func stripH1(text string) string {
	var out []string
	for _, line := range strings.Split(text, "\n") {
		s := strings.TrimSpace(line)
		if strings.HasPrefix(s, "# ") && !strings.HasPrefix(s, "## ") {
			continue
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

// ---- 步骤 4：CJK 空格 ----

var (
	// CJK 字符范围（汉字、扩展、标点、全角）。
	cjkClass   = `[\x{4e00}-\x{9fff}\x{3400}-\x{4dbf}\x{3000}-\x{303f}\x{ff00}-\x{ffef}]`
	latinClass = `[A-Za-z0-9]`
	reCJKLatin = regexp.MustCompile(`(` + cjkClass + `)(` + latinClass + `)`)
	reLatinCJK = regexp.MustCompile(`(` + latinClass + `)(` + cjkClass + `)`)
)

// fixCJKSpacing 在 CJK↔拉丁边界插入普通空格，逐行处理，跳过 ``` 代码块。
// 注：wewrite docstring 说插 U+200A，但其代码实为普通空格，以代码为准。
func fixCJKSpacing(text string) string {
	lines := strings.Split(text, "\n")
	inCode := false
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			inCode = !inCode
			continue
		}
		if inCode {
			continue
		}
		line = reCJKLatin.ReplaceAllString(line, "$1 $2")
		line = reLatinCJK.ReplaceAllString(line, "$1 $2")
		lines[i] = line
	}
	return strings.Join(lines, "\n")
}

// ---- 步骤 8：粗体标点外移 ----

var reBoldPunct = regexp.MustCompile(`(<strong>)(.*?)([，。！？；：、]+)(</strong>)`)

// fixCJKBoldPunctuation 把粗体内尾随的中文标点移到 </strong> 外。
func fixCJKBoldPunctuation(htmlStr string) string {
	return reBoldPunct.ReplaceAllString(htmlStr, "$1$2$4$3")
}

// ---- digest ----

// generateDigest 从 HTML 提取纯文本，截断到 maxBytes 个 UTF-8 字节。
func generateDigest(htmlStr string, maxBytes int) string {
	text := htmlToPlainText(htmlStr)
	text = strings.TrimSpace(regexp.MustCompile(`\s+`).ReplaceAllString(text, " "))

	if len(text) <= maxBytes {
		return text
	}

	const ellipsis = "..."
	target := maxBytes - len(ellipsis)
	if target < 0 {
		target = 0
	}
	// 回退到合法 UTF-8 边界。
	cut := target
	for cut > 0 && !utf8.RuneStart(text[cut]) {
		cut--
	}
	return strings.TrimRight(text[:cut], " ") + ellipsis
}

// htmlToPlainText 提取 HTML 的纯文本（goquery 的 .Text()）。
func htmlToPlainText(htmlStr string) string {
	doc, err := newFragmentDoc(htmlStr)
	if err != nil {
		return ""
	}
	return doc.Text()
}

// sortStrings 是 sort.Strings 的薄封装，避免在 style 文件重复 import sort。
func sortStrings(s []string) { sort.Strings(s) }

// ---- fragment 解析/渲染：所有 goquery 步骤的公共底座 ----

// newFragmentDoc 把 HTML 片段解析为 goquery.Document，以 body 为 context，
// 避免 goquery.NewDocumentFromReader 强行包 html/head/body。
func newFragmentDoc(htmlStr string) (*goquery.Document, error) {
	// DataAtom 必须与 Data 一致，否则 x/net/html 会报 "inconsistent Node"。
	bodyCtx := &gmhtml.Node{Type: gmhtml.ElementNode, DataAtom: atom.Body, Data: "body"}
	nodes, err := gmhtml.ParseFragment(strings.NewReader(htmlStr), bodyCtx)
	if err != nil {
		return nil, fmt.Errorf("parse html fragment: %w", err)
	}
	for _, n := range nodes {
		bodyCtx.AppendChild(n)
	}
	return goquery.NewDocumentFromNode(bodyCtx), nil
}

// renderFragment 渲染 body context 的所有子节点为 HTML 字符串，
// 不渲染 body 标签本身（输出仅 body 内容，对齐 wewrite str(soup)）。
//
// 注意：newFragmentDoc 把 body context 节点本身作为文档根（doc.Nodes[0]），
// 所以这里直接遍历根的子节点，而不是 doc.Find("body")（那会找后代，返回空）。
func renderFragment(doc *goquery.Document) (string, error) {
	root := fragmentRoot(doc)
	if root == nil {
		return doc.Html()
	}
	var buf bytes.Buffer
	for child := root.FirstChild; child != nil; child = child.NextSibling {
		if err := gmhtml.Render(&buf, child); err != nil {
			return "", fmt.Errorf("render html fragment: %w", err)
		}
	}
	return buf.String(), nil
}

// fragmentRoot 返回 newFragmentDoc 创建的 body context 根节点。
func fragmentRoot(doc *goquery.Document) *gmhtml.Node {
	if len(doc.Nodes) > 0 {
		return doc.Nodes[0]
	}
	return nil
}
