package config

import (
	"os"
	"path/filepath"
)

// 配置目录与文件名常量。第一期统一跨平台使用 ~/.thinkthinking/，
// 刻意不使用 XDG / AppData / Library 等平台特定路径（见 init.md）。
const (
	// DirName 是配置目录名（用户级在 home 下，项目级在 cwd 下）。
	DirName = ".thinkthinking"
	// FileName 是配置文件名。
	FileName = "config.yaml"
	// ThemesSubDir 是用户可放置自定义主题的子目录。
	ThemesSubDir = "themes"
)

// UserConfigDir 返回用户级配置目录：~/.thinkthinking 。
func UserConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, DirName), nil
}

// UserConfigPath 返回用户级配置文件路径：~/.thinkthinking/config.yaml 。
func UserConfigPath() (string, error) {
	dir, err := UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, FileName), nil
}

// ProjectConfigDir 返回项目级配置目录：<cwd>/.thinkthinking 。
func ProjectConfigDir() string {
	return DirName
}

// ProjectConfigPath 返回项目级配置文件路径：./.thinkthinking/config.yaml 。
func ProjectConfigPath() string {
	return filepath.Join(DirName, FileName)
}

// UserThemesDir 返回用户自定义主题目录：~/.thinkthinking/themes 。
func UserThemesDir() (string, error) {
	dir, err := UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, ThemesSubDir), nil
}

// FileExists 判断路径是否为存在的普通文件。
func FileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
