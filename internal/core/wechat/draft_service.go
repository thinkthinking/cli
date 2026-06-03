package wechat

import (
	"context"
	"os"
	"path/filepath"
	"strings"
)

// draftService 是 DraftService 的实现，编排：转换 → 上传正文图 → 回填 url
// → 上传封面 → 创建草稿。移植自 wewrite cli.py 的 cmd_publish。
type draftService struct {
	converter MarkdownConverter
	client    WeChatClient
}

// NewDraftService 创建草稿服务。converter 用于 Markdown→HTML，client 调微信 API。
func NewDraftService(converter MarkdownConverter, client WeChatClient) DraftService {
	return &draftService{converter: converter, client: client}
}

// CreateDraft 执行草稿创建全流程。
func (s *draftService) CreateDraft(ctx context.Context, input CreateDraftInput) (*CreateDraftResult, error) {
	if strings.TrimSpace(input.Title) == "" {
		return nil, &AuthError{Reason: "title is required"} // title 缺失属输入错误
	}

	// 1. 取得 HTML 与 digest。
	html := input.HTML
	digest := input.Digest
	var images []string

	if input.Markdown != "" {
		res, err := s.converter.Convert(ctx, ConvertInput{
			Markdown:  input.Markdown,
			ThemeName: input.ThemeName,
			Options:   input.ConvertOptions,
		})
		if err != nil {
			return nil, err
		}
		html = res.HTML
		images = res.Images
		if digest == "" {
			digest = res.Digest
		}
	}

	result := &CreateDraftResult{
		Title:          input.Title,
		Digest:         digest,
		UploadedImages: []UploadedImage{},
	}

	// 2. 上传正文本地图片并回填 url（远程 http(s) 跳过）。
	if input.UploadImages && len(images) > 0 {
		var uerr error
		html, result.UploadedImages, uerr = s.uploadAndReplaceImages(ctx, html, images, input.MarkdownDir)
		if uerr != nil {
			return nil, uerr
		}
	}

	// 3. 处理封面：优先已有 media_id，否则上传本地封面图。
	thumbMediaID := input.CoverMediaID
	if thumbMediaID == "" && input.CoverImage != "" {
		mediaID, _, err := s.client.UploadMaterial(ctx, input.CoverImage)
		if err != nil {
			return nil, err
		}
		thumbMediaID = mediaID
	}
	result.ThumbMediaID = thumbMediaID

	// 4. 创建草稿。
	article := DraftArticle{
		Title:        input.Title,
		Author:       input.Author,
		Digest:       digest,
		Content:      html,
		ThumbMediaID: thumbMediaID,
		ShowCoverPic: 0,
	}
	mediaID, raw, err := s.client.AddDraft(ctx, []DraftArticle{article})
	if err != nil {
		return nil, err
	}
	result.MediaID = mediaID
	result.RawResponse = raw
	return result, nil
}

// uploadAndReplaceImages 上传正文本地图片并把 html 里的原始 src 替换成微信 url。
// 远程 http(s) 图片跳过；本地图片按「绝对 → 相对 cwd → 相对 md 目录」解析。
// 移植自 wewrite cmd_publish 的图片处理。
func (s *draftService) uploadAndReplaceImages(ctx context.Context, html string, images []string, mdDir string) (string, []UploadedImage, error) {
	var uploaded []UploadedImage
	for _, src := range images {
		if strings.HasPrefix(src, "http://") || strings.HasPrefix(src, "https://") {
			uploaded = append(uploaded, UploadedImage{Source: src, Skipped: true, Reason: "remote image"})
			continue
		}

		localPath := resolveImagePath(src, mdDir)
		if localPath == "" {
			uploaded = append(uploaded, UploadedImage{Source: src, Skipped: true, Reason: "file not found"})
			continue
		}

		url, err := s.client.UploadContentImage(ctx, localPath)
		if err != nil {
			return html, uploaded, err
		}
		html = strings.ReplaceAll(html, src, url)
		uploaded = append(uploaded, UploadedImage{Source: src, WeChatURL: url})
	}
	return html, uploaded, nil
}

// resolveImagePath 解析正文图片的本地路径：绝对路径 → 相对 cwd → 相对 md 目录。
// 找不到返回空字符串。
func resolveImagePath(src, mdDir string) string {
	if filepath.IsAbs(src) {
		if fileExists(src) {
			return src
		}
		return ""
	}
	if fileExists(src) {
		return src
	}
	if mdDir != "" {
		candidate := filepath.Join(mdDir, src)
		if fileExists(candidate) {
			return candidate
		}
	}
	return ""
}

// fileExists 判断普通文件是否存在（本地小工具，不依赖 config 包）。
func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
