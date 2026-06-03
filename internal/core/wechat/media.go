package wechat

import (
	"context"
	"encoding/json"
	"fmt"
)

// 微信有两个不同的图片上传接口（移植自 wewrite wechat_api.py）：
//   - media/uploadimg     ：正文内图片，返回可直接嵌入 content 的 url
//   - material/add_material：永久素材（用作封面），返回 media_id

// uploadImgResponse 是 /cgi-bin/media/uploadimg 的响应。
type uploadImgResponse struct {
	URL     string `json:"url"`
	ErrCode int    `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
}

// UploadContentImage 上传正文图片，返回 url。
func (c *httpClient) UploadContentImage(ctx context.Context, imagePath string) (string, error) {
	token, err := c.AccessToken(ctx)
	if err != nil {
		return "", err
	}
	u := fmt.Sprintf("%s/cgi-bin/media/uploadimg?access_token=%s", apiBase, token)
	raw, err := c.postMultipartFile(ctx, u, imagePath)
	if err != nil {
		return "", err
	}
	var r uploadImgResponse
	if err := json.Unmarshal(raw, &r); err != nil {
		return "", &APIError{Op: "decode uploadimg response", Err: err}
	}
	if r.URL == "" {
		return "", &APIError{ErrCode: r.ErrCode, ErrMsg: r.ErrMsg, Op: "upload content image"}
	}
	return r.URL, nil
}

// addMaterialResponse 是 /cgi-bin/material/add_material 的响应。
type addMaterialResponse struct {
	MediaID string `json:"media_id"`
	URL     string `json:"url"`
	ErrCode int    `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
}

// UploadMaterial 上传永久图片素材（用作封面），返回 media_id 与 url。
func (c *httpClient) UploadMaterial(ctx context.Context, imagePath string) (string, string, error) {
	token, err := c.AccessToken(ctx)
	if err != nil {
		return "", "", err
	}
	u := fmt.Sprintf("%s/cgi-bin/material/add_material?access_token=%s&type=image", apiBase, token)
	raw, err := c.postMultipartFile(ctx, u, imagePath)
	if err != nil {
		return "", "", err
	}
	var r addMaterialResponse
	if err := json.Unmarshal(raw, &r); err != nil {
		return "", "", &APIError{Op: "decode add_material response", Err: err}
	}
	if r.MediaID == "" {
		return "", "", &APIError{ErrCode: r.ErrCode, ErrMsg: r.ErrMsg, Op: "upload material"}
	}
	return r.MediaID, r.URL, nil
}
