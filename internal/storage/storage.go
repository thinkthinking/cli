// Package storage 预留持久化层。
//
// 第一期不使用数据库。此包仅定义最小接口骨架，未来可接 SQLite 等实现，
// 用于缓存 access_token、记录草稿历史、保存素材 media_id 映射等。
// 现在留空是有意为之——避免过度设计（见 init.md）。
package storage

import "context"

// Store 是未来持久化层的最小抽象。当前无实现。
type Store interface {
	// Get 按 key 读取值。
	Get(ctx context.Context, key string) ([]byte, error)
	// Set 写入键值。
	Set(ctx context.Context, key string, value []byte) error
	// Close 释放底层资源。
	Close() error
}

// TODO(thinkthinking): 第二期可在此实现 SQLite-backed Store，
// 用于跨进程缓存 access_token 与草稿/素材记录。
