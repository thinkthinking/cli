package wechat

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

// mockClient 是 WeChatClient 的测试替身，记录调用并返回预设结果。
type mockClient struct {
	uploadedImages []string // 记录上传的正文图路径
	uploadedCover  string   // 记录上传的封面路径
	draftArticles  []DraftArticle
	contentImgURL  string // UploadContentImage 返回的 url
	coverMediaID   string // UploadMaterial 返回的 media_id
	draftMediaID   string // AddDraft 返回的 media_id
	uploadErr      error  // 若非 nil，上传时返回此错误
}

func (m *mockClient) AccessToken(ctx context.Context) (string, error) {
	return "mock-token", nil
}

func (m *mockClient) UploadContentImage(ctx context.Context, imagePath string) (string, error) {
	if m.uploadErr != nil {
		return "", m.uploadErr
	}
	m.uploadedImages = append(m.uploadedImages, imagePath)
	return m.contentImgURL, nil
}

func (m *mockClient) UploadMaterial(ctx context.Context, imagePath string) (string, string, error) {
	m.uploadedCover = imagePath
	return m.coverMediaID, "http://mmbiz.qpic.cn/cover.jpg", nil
}

func (m *mockClient) AddDraft(ctx context.Context, articles []DraftArticle) (string, json.RawMessage, error) {
	m.draftArticles = articles
	return m.draftMediaID, json.RawMessage(`{"media_id":"` + m.draftMediaID + `"}`), nil
}

// TestCreateDraftFromMarkdown 验证完整流程：转换 → 创建草稿。
func TestCreateDraftFromMarkdown(t *testing.T) {
	mc := &mockClient{draftMediaID: "DRAFT_123"}
	svc := NewDraftService(NewConverter(), mc)

	res, err := svc.CreateDraft(context.Background(), CreateDraftInput{
		Title:          "我的文章",
		Markdown:       "# 我的文章\n\n正文 content 内容。",
		ThemeName:      "default",
		Author:         "tester",
		ConvertOptions: DefaultConvertOptions(),
	})
	if err != nil {
		t.Fatalf("CreateDraft error: %v", err)
	}
	if res.MediaID != "DRAFT_123" {
		t.Errorf("media_id = %q, want DRAFT_123", res.MediaID)
	}
	// 草稿文章应携带转换后的 HTML 与作者。
	if len(mc.draftArticles) != 1 {
		t.Fatalf("expected 1 article, got %d", len(mc.draftArticles))
	}
	art := mc.draftArticles[0]
	if art.Title != "我的文章" || art.Author != "tester" {
		t.Errorf("article title/author wrong: %+v", art)
	}
	if !strings.Contains(art.Content, "<p style=") {
		t.Errorf("article content should be converted HTML, got: %s", art.Content)
	}
	if art.Digest == "" {
		t.Errorf("digest should be auto-generated from markdown")
	}
}

// TestCreateDraftFromHTML 验证 HTML 直接路径（不转换）。
func TestCreateDraftFromHTML(t *testing.T) {
	mc := &mockClient{draftMediaID: "D2"}
	svc := NewDraftService(NewConverter(), mc)

	res, err := svc.CreateDraft(context.Background(), CreateDraftInput{
		Title:  "标题",
		HTML:   "<section>已有 HTML</section>",
		Digest: "自定义摘要",
	})
	if err != nil {
		t.Fatalf("CreateDraft error: %v", err)
	}
	if res.MediaID != "D2" {
		t.Errorf("media_id = %q, want D2", res.MediaID)
	}
	if mc.draftArticles[0].Content != "<section>已有 HTML</section>" {
		t.Errorf("HTML path should pass content through unchanged, got: %s", mc.draftArticles[0].Content)
	}
	if mc.draftArticles[0].Digest != "自定义摘要" {
		t.Errorf("explicit digest should be used, got: %s", mc.draftArticles[0].Digest)
	}
}

// TestCreateDraftRemoteImageSkipped 远程图片应跳过上传。
func TestCreateDraftRemoteImageSkipped(t *testing.T) {
	mc := &mockClient{draftMediaID: "D3", contentImgURL: "http://wx/img.png"}
	svc := NewDraftService(NewConverter(), mc)

	res, err := svc.CreateDraft(context.Background(), CreateDraftInput{
		Title:          "图片测试",
		Markdown:       "![远程](https://example.com/a.png)",
		ThemeName:      "default",
		UploadImages:   true,
		ConvertOptions: DefaultConvertOptions(),
	})
	if err != nil {
		t.Fatalf("CreateDraft error: %v", err)
	}
	// 远程图不应触发上传。
	if len(mc.uploadedImages) != 0 {
		t.Errorf("remote image should not be uploaded, got uploads: %v", mc.uploadedImages)
	}
	if len(res.UploadedImages) != 1 || !res.UploadedImages[0].Skipped {
		t.Errorf("expected 1 skipped remote image, got: %+v", res.UploadedImages)
	}
}

// TestCreateDraftTitleRequired 缺 title 报错。
func TestCreateDraftTitleRequired(t *testing.T) {
	svc := NewDraftService(NewConverter(), &mockClient{})
	_, err := svc.CreateDraft(context.Background(), CreateDraftInput{
		HTML: "<p>x</p>",
	})
	if err == nil {
		t.Fatal("expected error for missing title")
	}
}

// TestCoverMediaIDPreferred 已有 cover-media-id 时不上传封面。
func TestCoverMediaIDPreferred(t *testing.T) {
	mc := &mockClient{draftMediaID: "D4"}
	svc := NewDraftService(NewConverter(), mc)

	res, err := svc.CreateDraft(context.Background(), CreateDraftInput{
		Title:        "标题",
		HTML:         "<p>x</p>",
		CoverMediaID: "EXISTING_MEDIA",
	})
	if err != nil {
		t.Fatalf("CreateDraft error: %v", err)
	}
	if mc.uploadedCover != "" {
		t.Errorf("should not upload cover when cover-media-id given, uploaded: %s", mc.uploadedCover)
	}
	if res.ThumbMediaID != "EXISTING_MEDIA" {
		t.Errorf("thumb_media_id = %q, want EXISTING_MEDIA", res.ThumbMediaID)
	}
}
