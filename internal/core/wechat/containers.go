package wechat

import (
	"fmt"
	"regexp"
	"strings"
)

// 容器块语法移植自 wewrite converter.py 的 _preprocess_containers。
// 在 Markdown 解析前，把 :::xxx ... ::: 块替换成内联样式 HTML。
// 顺序与 Python 一致：dialogue → timeline → callout → quote → highlight → summary。

var (
	reDialogue  = regexp.MustCompile(`(?s):::dialogue\n(.*?)\n:::`)
	reTimeline  = regexp.MustCompile(`(?s):::timeline\n(.*?)\n:::`)
	reCallout   = regexp.MustCompile(`(?s):::callout\s+(\w+)\n(.*?)\n:::`)
	reQuoteBlk  = regexp.MustCompile(`(?s):::quote\n(.*?)\n:::`)
	reHighlight = regexp.MustCompile(`(?s):::highlight\n(.*?)\n:::`)
	reSummary   = regexp.MustCompile(`(?s):::summary\n(.*?)\n:::`)
)

// preprocessContainers 依次处理 6 种容器块。
func (c *converter) preprocessContainers(theme *Theme, text string) string {
	text = c.processDialogue(theme, text)
	text = c.processTimeline(theme, text)
	text = c.processCallout(theme, text)
	text = c.processQuoteBlock(theme, text)
	text = c.processHighlight(theme, text)
	text = c.processSummary(theme, text)
	return text
}

// processDialogue 把 :::dialogue 块转成聊天气泡布局。
// 行内以 "> " 开头表示右侧（回复）气泡，否则左侧气泡。
func (c *converter) processDialogue(theme *Theme, text string) string {
	primary := theme.Color("primary", "#2563eb")
	return reDialogue.ReplaceAllStringFunc(text, func(m string) string {
		content := reDialogue.FindStringSubmatch(m)[1]
		var bubbles []string
		for _, line := range strings.Split(strings.TrimSpace(content), "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			if strings.HasPrefix(line, "> ") {
				msg := strings.TrimSpace(line[2:])
				bubbles = append(bubbles, fmt.Sprintf(
					`<section style="display: flex; justify-content: flex-end; margin-bottom: 12px">`+
						`<section style="background: %s; color: white; padding: 10px 14px; border-radius: 12px 12px 2px 12px; max-width: 80%%; font-size: 15px; line-height: 1.6">%s</section></section>`,
					primary, msg))
			} else {
				bubbles = append(bubbles, fmt.Sprintf(
					`<section style="display: flex; justify-content: flex-start; margin-bottom: 12px">`+
						`<section style="background: #f3f4f6; color: #333; padding: 10px 14px; border-radius: 12px 12px 12px 2px; max-width: 80%%; font-size: 15px; line-height: 1.6">%s</section></section>`,
					line))
			}
		}
		return strings.Join(bubbles, "\n")
	})
}

// processTimeline 把 :::timeline 块转成竖向时间线。
func (c *converter) processTimeline(theme *Theme, text string) string {
	primary := theme.Color("primary", "#2563eb")
	return reTimeline.ReplaceAllStringFunc(text, func(m string) string {
		content := reTimeline.FindStringSubmatch(m)[1]
		var items []string
		for _, line := range strings.Split(strings.TrimSpace(content), "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			items = append(items, fmt.Sprintf(
				`<section style="display: flex; margin-bottom: 16px">`+
					`<section style="flex-shrink: 0; width: 12px; display: flex; flex-direction: column; align-items: center">`+
					`<section style="width: 10px; height: 10px; border-radius: 50%%; background: %s; margin-top: 6px"></section>`+
					`<section style="width: 2px; flex: 1; background: #e5e7eb; margin-top: 4px"></section>`+
					`</section>`+
					`<section style="flex: 1; padding-left: 12px; padding-bottom: 8px; font-size: 15px; line-height: 1.7">%s</section>`+
					`</section>`,
				primary, line))
		}
		return strings.Join(items, "\n")
	})
}

// calloutStyle 是 callout 类型 → (主色, 背景, 图标) 的映射。
var calloutStyle = map[string][3]string{
	"tip":     {"#059669", "#ecfdf5", "💡"},
	"warning": {"#d97706", "#fffbeb", "⚠️"},
	"info":    {"#2563eb", "#eff6ff", "ℹ️"},
	"danger":  {"#dc2626", "#fef2f2", "🚨"},
}

// processCallout 把 :::callout tip/warning/info/danger 块转成提示框。
func (c *converter) processCallout(theme *Theme, text string) string {
	return reCallout.ReplaceAllStringFunc(text, func(m string) string {
		sub := reCallout.FindStringSubmatch(m)
		ctype := strings.ToLower(strings.TrimSpace(sub[1]))
		content := strings.TrimSpace(sub[2])
		style, ok := calloutStyle[ctype]
		if !ok {
			style = calloutStyle["info"]
		}
		color, bg, icon := style[0], style[1], style[2]
		return fmt.Sprintf(
			`<section style="background: %s; border-left: 4px solid %s; padding: 14px 16px; border-radius: 4px; margin: 16px 0; font-size: 15px; line-height: 1.7">`+
				`<section style="font-weight: 700; color: %s; margin-bottom: 6px">%s %s</section>%s</section>`,
			bg, color, color, icon, strings.ToUpper(ctype), content)
	})
}

// processQuoteBlock 把 :::quote 块转成样式化引言。
func (c *converter) processQuoteBlock(theme *Theme, text string) string {
	primary := theme.Color("primary", "#2563eb")
	return reQuoteBlk.ReplaceAllStringFunc(text, func(m string) string {
		content := strings.TrimSpace(reQuoteBlk.FindStringSubmatch(m)[1])
		return fmt.Sprintf(
			`<section style="margin: 24px 0; padding: 20px 24px; border-left: 4px solid %s; background: linear-gradient(135deg, #f8f9fa 0%%, #ffffff 100%%); border-radius: 0 8px 8px 0">`+
				`<section style="font-size: 18px; line-height: 1.8; color: #333; font-style: italic">"%s"</section></section>`,
			primary, content)
	})
}

// processHighlight 把 :::highlight 块转成琥珀色信息框（首行作标题）。
func (c *converter) processHighlight(theme *Theme, text string) string {
	secondary := theme.Color("secondary", "#c4820e")
	bg := theme.Color("highlight_bg", "#fef7e8")
	border := theme.Color("highlight_border", "rgba(196,130,14,0.2)")
	return reHighlight.ReplaceAllStringFunc(text, func(m string) string {
		content := strings.TrimSpace(reHighlight.FindStringSubmatch(m)[1])
		title, body := splitTitleBody(content, "")
		var sb strings.Builder
		fmt.Fprintf(&sb, `<section style="margin: 24px 0; padding: 20px 24px; background: %s; border: 1px solid %s; border-radius: 6px;">`, bg, border)
		if title != "" {
			fmt.Fprintf(&sb, `<p style="margin: 0;"><strong style="color: %s;">%s</strong></p>`, secondary, title)
		}
		if body != "" {
			fmt.Fprintf(&sb, `<p style="margin: 8px 0 0 0;">%s</p>`, body)
		}
		sb.WriteString(`</section>`)
		return sb.String()
	})
}

// processSummary 把 :::summary 块转成青色总结框（首行作标题，默认"总结"）。
func (c *converter) processSummary(theme *Theme, text string) string {
	primary := theme.Color("primary", "#1a6b5a")
	bg := theme.Color("summary_bg", "#e8f5f0")
	border := theme.Color("summary_border", "rgba(26,107,90,0.15)")
	return reSummary.ReplaceAllStringFunc(text, func(m string) string {
		content := strings.TrimSpace(reSummary.FindStringSubmatch(m)[1])
		title, body := splitTitleBody(content, "总结")
		var sb strings.Builder
		fmt.Fprintf(&sb, `<section style="margin: 24px 0; padding: 20px 24px; background: %s; border: 1px solid %s; border-radius: 6px;">`, bg, border)
		fmt.Fprintf(&sb, `<p style="margin: 0;"><strong style="color: %s;">%s</strong></p>`, primary, title)
		if body != "" {
			fmt.Fprintf(&sb, `<p style="margin: 8px 0 0 0;">%s</p>`, body)
		}
		sb.WriteString(`</section>`)
		return sb.String()
	})
}

// splitTitleBody 把内容按首个换行拆成 (标题, 正文)。
// 标题为空时用 defaultTitle。复刻 Python 的 split('\n', 1)。
func splitTitleBody(content, defaultTitle string) (string, string) {
	parts := strings.SplitN(content, "\n", 2)
	title := strings.TrimSpace(parts[0])
	if title == "" {
		title = defaultTitle
	}
	body := ""
	if len(parts) > 1 {
		body = strings.TrimSpace(parts[1])
	}
	return title, body
}
