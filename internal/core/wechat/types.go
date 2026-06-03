// Package wechat 实现 Markdown → 微信公众号兼容 HTML 的转换，以及草稿上传。
//
// 转换 pipeline 移植自经过实战验证的 wewrite（Python）converter.py，
// 用 goldmark + goquery + douceur 在 Go 侧 1:1 复刻其行为（语义/视觉等价，
// 非逐字节相同——代码高亮 chroma≠pygments、HTML 规范化存在差异）。
package wechat

import (
	"context"
	"encoding/json"
)

// ConvertInput 是一次转换的输入。
type ConvertInput struct {
	// Markdown 是源文本。与 InputPath 二选一，由调用方解析后填入。
	Markdown string
	// ThemeName 是主题名（如 "default"）。空则用 "default"。
	ThemeName string
	// Options 控制可选兼容增强特性。
	Options ConvertOptions
}

// ConvertOptions 控制可选的兼容增强特性。
//
// 基础兼容（CJK 空格 / 列表转 section / p 强制 color / pre 换行）无条件执行，
// 不在此开关。这里只放可选项。零值即合理默认：暗黑模式 + 脚注 + 容器块默认开，
// 反指纹默认关（随机序列无法复现），AIGC footer 由主题决定。
type ConvertOptions struct {
	// DarkMode 注入 data-darkmode-* 属性（默认开）。
	DarkMode bool
	// Footnotes 把外链转成上标脚注（默认开）。
	Footnotes bool
	// Containers 处理 :::callout 等容器块（默认开）。
	Containers bool
	// AntiFingerprint 启用 CSS 随机微扰（默认关）。
	AntiFingerprint bool
	// AIGCFooter 追加 AIGC 声明（默认跟随主题；此字段为显式覆盖）。
	AIGCFooter bool
	// aigcFooterSet 标记 AIGCFooter 是否被显式设置（区分"未设置"与"设为 false"）。
	aigcFooterSet bool
}

// DefaultConvertOptions 返回推荐的默认开关组合。
func DefaultConvertOptions() ConvertOptions {
	return ConvertOptions{
		DarkMode:        true,
		Footnotes:       true,
		Containers:      true,
		AntiFingerprint: false,
	}
}

// ConvertResult 是转换结果。
type ConvertResult struct {
	// HTML 是微信兼容的 inline-style HTML（仅 body 内容，无 html/head 包裹）。
	HTML string `json:"html"`
	// Title 是提取出的 H1 标题（微信有独立 title 字段，正文已剔除 H1）。
	Title string `json:"title"`
	// Digest 是自动生成的摘要（≤120 UTF-8 字节）。
	Digest string `json:"digest"`
	// Images 是正文中引用的图片 src 列表（供后续上传回填）。
	Images []string `json:"images"`
	// ThemeName 是实际使用的主题名。
	ThemeName string `json:"theme"`
	// Warnings 收集转换期的非致命问题（如未知容器类型、缺失主题字段）。
	Warnings []string `json:"warnings"`
}

// MarkdownConverter 是转换器接口。CLI / draft service 都依赖此接口而非具体实现，
// 便于测试时 mock。
type MarkdownConverter interface {
	Convert(ctx context.Context, input ConvertInput) (*ConvertResult, error)
}

// ---- 草稿相关 ----

// DraftArticle 是微信草稿 API 的单篇文章结构（/cgi-bin/draft/add）。
// 字段名严格对齐微信官方 API。
type DraftArticle struct {
	Title        string `json:"title"`
	Author       string `json:"author,omitempty"`
	Digest       string `json:"digest,omitempty"`
	Content      string `json:"content"`
	ThumbMediaID string `json:"thumb_media_id,omitempty"`
	ShowCoverPic int    `json:"show_cover_pic"`
	// ContentSourceURL 原文链接（可选）。
	ContentSourceURL string `json:"content_source_url,omitempty"`
}

// draftAddRequest 是 draft/add 的请求体。
type draftAddRequest struct {
	Articles []DraftArticle `json:"articles"`
}

// draftAddResponse 是 draft/add 的响应体。
type draftAddResponse struct {
	MediaID string `json:"media_id"`
	ErrCode int    `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
}

// CreateDraftInput 是创建草稿的输入。
type CreateDraftInput struct {
	// Title 必填。
	Title string
	// Markdown 与 HTML 二选一：Markdown 非空则先转换；否则用 HTML。
	Markdown string
	HTML     string
	// ThemeName 当走 Markdown 转换时使用。
	ThemeName string
	// Author 可选，默认从配置 wechat.default_author 读取（由 CLI 注入）。
	Author string
	// Digest 可选，空则用转换生成的摘要（仅 Markdown 路径有）。
	Digest string
	// CoverImage 可选：本地封面图路径，上传为 thumb_media_id。
	CoverImage string
	// CoverMediaID 可选：已有的封面 media_id（与 CoverImage 二选一）。
	CoverMediaID string
	// MarkdownDir 是 Markdown 文件所在目录，用于解析正文相对图片路径。
	MarkdownDir string
	// UploadImages 控制是否自动上传正文本地图片并回填 url。
	UploadImages bool
	// ConvertOptions 透传给转换器。
	ConvertOptions ConvertOptions
}

// UploadedImage 记录一次正文图片上传结果。
type UploadedImage struct {
	Source    string `json:"source"`
	WeChatURL string `json:"wechat_url,omitempty"`
	Skipped   bool   `json:"skipped"`
	Reason    string `json:"reason,omitempty"`
}

// CreateDraftResult 是创建草稿的结果。
type CreateDraftResult struct {
	MediaID        string          `json:"media_id"`
	Title          string          `json:"title"`
	Digest         string          `json:"digest"`
	ThumbMediaID   string          `json:"thumb_media_id,omitempty"`
	UploadedImages []UploadedImage `json:"uploaded_images"`
	RawResponse    json.RawMessage `json:"raw_response,omitempty"`
}

// DraftService 是草稿创建接口，便于 CLI 依赖抽象、测试时注入 mock client。
type DraftService interface {
	CreateDraft(ctx context.Context, input CreateDraftInput) (*CreateDraftResult, error)
}
