package wechat

import (
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/util"
	gmhtml "golang.org/x/net/html"
)

// ---- 步骤 6：代码块 data-lang（WrapperRenderer）----

// emptyPreWrapper 实现 chroma 的 PreWrapper 接口，Start/End 返回空，
// 从而抑制 chroma 自带的 <pre>，交由 wrapPreWithLang 输出带 data-lang 的 <pre>。
type emptyPreWrapper struct{}

func (emptyPreWrapper) Start(code bool, styleAttr string) string { return "" }
func (emptyPreWrapper) End(code bool) string                     { return "" }

// wrapPreWithLang 在高亮代码块外层写 <pre data-lang="lang"> ... </pre>。
// 从 ctx.Language() 取语言名（可能缺失，需保护）。比 Python 后处理从 class
// 提取更稳，也对齐 wewrite 给 <pre> 加 data-lang 的意图。
func wrapPreWithLang(w util.BufWriter, ctx highlighting.CodeBlockContext, entering bool) {
	if entering {
		lang, ok := ctx.Language()
		if ok && len(lang) > 0 {
			_, _ = w.WriteString(`<pre data-lang="`)
			_, _ = w.Write(lang)
			_, _ = w.WriteString(`">`)
		} else {
			_, _ = w.WriteString("<pre>")
		}
		return
	}
	_, _ = w.WriteString("</pre>")
}

// ---- 步骤 7：图片 ----

// processImages 收集 img src，并补响应式 style。
func processImages(htmlStr string) (string, []string) {
	doc, err := newFragmentDoc(htmlStr)
	if err != nil {
		return htmlStr, nil
	}
	var images []string
	doc.Find("img").Each(func(_ int, s *goquery.Selection) {
		if src, ok := s.Attr("src"); ok && src != "" {
			images = append(images, src)
		}
		existing, _ := s.Attr("style")
		if !strings.Contains(existing, "max-width") {
			additions := "max-width: 100%; height: auto; display: block; margin: 24px auto"
			if existing != "" {
				s.SetAttr("style", existing+"; "+additions)
			} else {
				s.SetAttr("style", additions)
			}
		}
	})
	out, rerr := renderFragment(doc)
	if rerr != nil {
		return htmlStr, images
	}
	return out, images
}

// ---- 步骤 9：列表转 section ----

// convertListsToSections 把 ul/ol 转成 section 结构（微信原生列表渲染不可靠）。
func convertListsToSections(htmlStr string, theme *Theme) (string, error) {
	doc, err := newFragmentDoc(htmlStr)
	if err != nil {
		return "", err
	}
	textColor := theme.Color("text", "#333333")
	primary := theme.Color("primary", "#2563eb")

	// ul → section（• bullet）。
	doc.Find("ul").Each(func(_ int, sel *goquery.Selection) {
		ul := sel.Nodes[0]
		section := newElement("section")
		for li := ul.FirstChild; li != nil; li = li.NextSibling {
			if li.Type != gmhtml.ElementNode || li.Data != "li" {
				continue
			}
			item := newStyledElement("section", fmt.Sprintf("display: flex; align-items: flex-start; margin-bottom: 8px; color: %s", textColor))
			bullet := newStyledElement("span", fmt.Sprintf("color: %s; margin-right: 8px; flex-shrink: 0; font-size: 18px; line-height: 1.6", primary))
			bullet.AppendChild(textNode("•"))
			content := newStyledElement("span", "flex: 1")
			moveChildren(li, content)
			item.AppendChild(bullet)
			item.AppendChild(content)
			section.AppendChild(item)
		}
		replaceNode(ul, section)
	})

	// ol → section（数字）。
	doc.Find("ol").Each(func(_ int, sel *goquery.Selection) {
		ol := sel.Nodes[0]
		section := newElement("section")
		num := 0
		for li := ol.FirstChild; li != nil; li = li.NextSibling {
			if li.Type != gmhtml.ElementNode || li.Data != "li" {
				continue
			}
			num++
			item := newStyledElement("section", fmt.Sprintf("display: flex; align-items: flex-start; margin-bottom: 8px; color: %s", textColor))
			number := newStyledElement("span", fmt.Sprintf("color: %s; margin-right: 8px; flex-shrink: 0; font-weight: 700; line-height: 1.8", primary))
			number.AppendChild(textNode(fmt.Sprintf("%d.", num)))
			content := newStyledElement("span", "flex: 1")
			moveChildren(li, content)
			item.AppendChild(number)
			item.AppendChild(content)
			section.AppendChild(item)
		}
		replaceNode(ol, section)
	})

	return renderFragment(doc)
}

// ---- 步骤 10：外链转脚注 ----

// convertLinksToFootnotes 把外链 a[href] 转成上标 [n]，文末追加参考链接列表。
func convertLinksToFootnotes(htmlStr string, theme *Theme) (string, error) {
	doc, err := newFragmentDoc(htmlStr)
	if err != nil {
		return "", err
	}
	primary := theme.Color("primary", "#2563eb")

	type footnote struct {
		num  int
		text string
		href string
	}
	var notes []footnote
	counter := 0

	doc.Find("a").Each(func(_ int, s *goquery.Selection) {
		href, ok := s.Attr("href")
		if !ok || href == "" || strings.HasPrefix(href, "#") {
			return // 跳过锚点
		}
		counter++
		linkText := s.Text()
		notes = append(notes, footnote{counter, linkText, href})

		// 用 "文本 + 上标[n]" 替换 <a>。
		a := s.Nodes[0]
		parent := a.Parent
		if parent == nil {
			return
		}
		txt := textNode(linkText)
		sup := newElement("sup")
		supSpan := newStyledElement("span", fmt.Sprintf("color: %s; font-size: 12px", primary))
		supSpan.AppendChild(textNode(fmt.Sprintf("[%d]", counter)))
		sup.AppendChild(supSpan)
		parent.InsertBefore(txt, a)
		parent.InsertBefore(sup, a)
		parent.RemoveChild(a)
	})

	if len(notes) > 0 {
		body := fragmentRoot(doc)
		hr := newStyledElement("hr", "border: none; border-top: 1px solid #e5e5e5; margin: 32px 0 16px")
		body.AppendChild(hr)
		refTitle := newStyledElement("p", "font-size: 13px; color: #999999; margin-bottom: 8px; font-weight: 700")
		refTitle.AppendChild(textNode("参考链接"))
		body.AppendChild(refTitle)
		for _, n := range notes {
			ref := newStyledElement("p", "font-size: 12px; color: #999999; margin: 2px 0; word-break: break-all")
			ref.AppendChild(textNode(fmt.Sprintf("[%d] %s: %s", n.num, n.text, n.href)))
			body.AppendChild(ref)
		}
	}

	return renderFragment(doc)
}

// ---- 步骤 11：主题 CSS 内联 ----

// applyInlineStyles 把主题规则按选择器匹配元素，合并进 inline style（已有优先）。
func applyInlineStyles(htmlStr string, rules []cssRule) (string, error) {
	doc, err := newFragmentDoc(htmlStr)
	if err != nil {
		return "", err
	}
	for _, rule := range rules {
		if rule.selector == "body" {
			continue // 不包 body
		}
		doc.Find(rule.selector).Each(func(_ int, s *goquery.Selection) {
			existing, _ := s.Attr("style")
			merged := mergeStyles(existing, rule.props)
			s.SetAttr("style", merged)
		})
	}
	return renderFragment(doc)
}

// mergeStyles 把已有 inline style 与主题属性合并。已有属性优先（不被覆盖），
// 顺序：先已有属性（保序），再补充缺失的主题属性。复刻 wewrite 的合并语义。
func mergeStyles(existing string, themeProps []cssProp) string {
	type kv struct{ k, v string }
	var ordered []kv
	seen := map[string]bool{}

	// 解析已有 inline style（手写切分，复刻 Python split 行为）。
	if existing != "" {
		for _, item := range strings.Split(existing, ";") {
			item = strings.TrimSpace(item)
			if item == "" {
				continue
			}
			idx := strings.Index(item, ":")
			if idx < 0 {
				continue
			}
			k := strings.TrimSpace(item[:idx])
			v := strings.TrimSpace(item[idx+1:])
			if !seen[k] {
				ordered = append(ordered, kv{k, v})
				seen[k] = true
			}
		}
	}
	// 补充缺失的主题属性。
	for _, p := range themeProps {
		if !seen[p.name] {
			ordered = append(ordered, kv{p.name, p.value})
			seen[p.name] = true
		}
	}

	parts := make([]string, 0, len(ordered))
	for _, e := range ordered {
		parts = append(parts, e.k+": "+e.v)
	}
	return strings.Join(parts, "; ")
}

// ---- 步骤 12：微信兼容修复 ----

// applyWeChatFixes 给每个 p 强制 color、每个 pre 强制 white-space。
func applyWeChatFixes(htmlStr string, theme *Theme) (string, error) {
	doc, err := newFragmentDoc(htmlStr)
	if err != nil {
		return "", err
	}
	textColor := theme.Color("text", "#333333")

	doc.Find("p").Each(func(_ int, s *goquery.Selection) {
		style, _ := s.Attr("style")
		if !strings.Contains(style, "color") {
			if style != "" {
				s.SetAttr("style", style+"; color: "+textColor)
			} else {
				s.SetAttr("style", "color: "+textColor)
			}
		}
	})
	doc.Find("pre").Each(func(_ int, s *goquery.Selection) {
		style, _ := s.Attr("style")
		if !strings.Contains(style, "white-space") {
			add := "white-space: pre-wrap; word-wrap: break-word"
			if style != "" {
				s.SetAttr("style", style+"; "+add)
			} else {
				s.SetAttr("style", add)
			}
		}
	})

	return renderFragment(doc)
}

// ---- 步骤 13：暗黑模式属性 ----

// injectDarkmode 注入 data-darkmode-* 属性。
func injectDarkmode(htmlStr string, theme *Theme) (string, error) {
	dm := theme.Darkmode()
	if dm == nil {
		return htmlStr, nil // 无暗黑配置则跳过
	}
	doc, err := newFragmentDoc(htmlStr)
	if err != nil {
		return "", err
	}

	dmText := darkColor(dm, "text", "#c8c8c8")
	dmPrimary := darkColor(dm, "primary", "#6aadff")

	// body 级元素：有 color 的才注入。
	doc.Find("p, span, section").Each(func(_ int, s *goquery.Selection) {
		style, _ := s.Attr("style")
		if strings.Contains(style, "color") {
			s.SetAttr("data-darkmode-color", dmText)
			s.SetAttr("data-darkmode-bgcolor", "transparent")
		}
	})
	// 标题。
	dmHeading := darkColor(dm, "text", "#e0e0e0")
	doc.Find("h1, h2, h3, h4").Each(func(_ int, s *goquery.Selection) {
		s.SetAttr("data-darkmode-color", dmHeading)
		s.SetAttr("data-darkmode-bgcolor", "transparent")
	})
	// 代码块。
	dmCodeBg := darkColor(dm, "code_bg", "#2d2d2d")
	dmCodeColor := darkColor(dm, "code_color", "#d4d4d4")
	doc.Find("pre").Each(func(_ int, s *goquery.Selection) {
		s.SetAttr("data-darkmode-bgcolor", dmCodeBg)
		s.SetAttr("data-darkmode-color", dmCodeColor)
	})
	doc.Find("code").Each(func(_ int, s *goquery.Selection) {
		s.SetAttr("data-darkmode-color", dmCodeColor)
	})
	// 引用块。
	dmQuoteBg := darkColor(dm, "quote_bg", "#2a2a2a")
	doc.Find("blockquote").Each(func(_ int, s *goquery.Selection) {
		s.SetAttr("data-darkmode-bgcolor", dmQuoteBg)
		s.SetAttr("data-darkmode-color", dmText)
	})
	// 粗体用主色。
	doc.Find("strong").Each(func(_ int, s *goquery.Selection) {
		s.SetAttr("data-darkmode-color", dmPrimary)
	})

	return renderFragment(doc)
}

// ---- 步骤 15：AIGC footer ----

// appendAIGCFooter 追加 AIGC 声明（微信平台合规要求）。
func appendAIGCFooter(htmlStr string) string {
	footer := `<p style="text-align: center; font-size: 13px; color: #9ca3af; margin-top: 48px; padding-top: 24px; border-top: 1px solid #e5e7eb;">本文由 AI 辅助创作，作者进行了实测验证和编辑修改。</p>`
	return htmlStr + "\n" + footer
}

// ---- 底层 *html.Node 操作（复刻 BeautifulSoup extract/append/replace）----

// newElement 创建一个元素节点。
func newElement(tag string) *gmhtml.Node {
	return &gmhtml.Node{Type: gmhtml.ElementNode, Data: tag}
}

// newStyledElement 创建一个带 style 属性的元素节点。
func newStyledElement(tag, style string) *gmhtml.Node {
	n := newElement(tag)
	n.Attr = append(n.Attr, gmhtml.Attribute{Key: "style", Val: style})
	return n
}

// textNode 创建一个文本节点。
func textNode(text string) *gmhtml.Node {
	return &gmhtml.Node{Type: gmhtml.TextNode, Data: text}
}

// moveChildren 把 src 的所有子节点搬到 dst（先 detach 再 append，避免迭代时改链表）。
func moveChildren(src, dst *gmhtml.Node) {
	var kids []*gmhtml.Node
	for c := src.FirstChild; c != nil; c = c.NextSibling {
		kids = append(kids, c)
	}
	for _, c := range kids {
		src.RemoveChild(c)
		dst.AppendChild(c)
	}
}

// replaceNode 用 replacement 替换 old（在其父节点中原位替换）。
func replaceNode(old, replacement *gmhtml.Node) {
	parent := old.Parent
	if parent == nil {
		return
	}
	parent.InsertBefore(replacement, old)
	parent.RemoveChild(old)
}
