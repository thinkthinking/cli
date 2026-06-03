package wechat

import (
	"math/rand"
	"regexp"
	"strconv"
)

// randomizeCSS 对 HTML 做轻微 CSS 随机微扰（反指纹），移植自 wewrite _randomize_css。
//
// 默认关闭：随机序列无法与 Python 对齐，且对 Agent 可复现性不利。仅当主题
// css_randomize=true 或显式开启 AntiFingerprint 时调用。
//
// 微扰 p 的 margin-bottom / font-size / line-height，以及 h2 的 margin / font-size。
func randomizeCSS(htmlStr string) string {
	fontSizeDelta := []int{0, -1}[rand.Intn(2)]

	// p: margin: 0 0 Npx 0 → 抖动 N。
	htmlStr = reMarginBottom.ReplaceAllStringFunc(htmlStr, func(m string) string {
		sub := reMarginBottom.FindStringSubmatch(m)
		n, _ := strconv.Atoi(sub[1])
		return "margin: 0 0 " + strconv.Itoa(n+rand.Intn(7)-3) + "px 0"
	})
	// p: font-size: 17px → 17+delta。
	htmlStr = reFont17.ReplaceAllString(htmlStr, "font-size: "+strconv.Itoa(17+fontSizeDelta)+"px")
	// h2: font-size: 22px → 22±1。
	htmlStr = reFont22.ReplaceAllStringFunc(htmlStr, func(string) string {
		return "font-size: " + strconv.Itoa(22+[]int{-1, 0, 1}[rand.Intn(3)]) + "px"
	})

	return htmlStr
}

var (
	reMarginBottom = regexp.MustCompile(`margin:\s*0\s+0\s+(\d+)px\s+0`)
	reFont17       = regexp.MustCompile(`font-size:\s*17px`)
	reFont22       = regexp.MustCompile(`font-size:\s*22px`)
)
