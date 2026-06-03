package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestUserConfigPath 断言用户配置路径固定为 ~/.thinkthinking/config.yaml，
// 而不是 XDG / Library / AppData（init.md 的明确要求）。
func TestUserConfigPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	path, err := UserConfigPath()
	if err != nil {
		t.Fatalf("UserConfigPath error: %v", err)
	}

	want := filepath.Join(home, ".thinkthinking", "config.yaml")
	if path != want {
		t.Errorf("config path = %q, want %q", path, want)
	}
	// 反向确认：不应包含任何平台特定目录段。
	for _, bad := range []string{".config", "Library", "AppData"} {
		if strings.Contains(path, bad) {
			t.Errorf("config path should not contain %q, got %q", bad, path)
		}
	}
}

// TestLoadDefaults 验证无配置文件时返回内置默认值。
func TestLoadDefaults(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	// 隔离 env，避免宿主机的 WECHAT_* 干扰。
	for _, e := range []string{"WECHAT_APP_ID", "WECHAT_APP_SECRET", "WECHAT_ACCESS_TOKEN", "WECHAT_AUTHOR", "WECHAT_THEME"} {
		t.Setenv(e, "")
	}

	res, err := Load(LoadOptions{})
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if res.Config.WeChat.DefaultAuthor != "thinkthinking" {
		t.Errorf("default_author = %q, want thinkthinking", res.Config.WeChat.DefaultAuthor)
	}
	if res.Config.WeChat.DefaultTheme != "default" {
		t.Errorf("default_theme = %q, want default", res.Config.WeChat.DefaultTheme)
	}
}

// TestEnvOverridesFile 验证环境变量优先级高于配置文件。
func TestEnvOverridesFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	// 写一个用户配置文件，app_id = from_file。
	dir := filepath.Join(home, ".thinkthinking")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	content := "wechat:\n  app_id: from_file\n"
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	// 设置 env override。
	t.Setenv("WECHAT_APP_ID", "from_env")

	res, err := Load(LoadOptions{})
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if got := res.Config.WeChat.AppID; got != "from_env" {
		t.Errorf("app_id = %q, want from_env (env should override file)", got)
	}
}

// TestSetValuePreservesOthers 验证 config set 不破坏文件中其它字段。
func TestSetValuePreservesOthers(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	path := filepath.Join(home, ".thinkthinking", "config.yaml")

	if err := SetValue(path, "wechat.app_id", "id1"); err != nil {
		t.Fatalf("SetValue error: %v", err)
	}
	if err := SetValue(path, "wechat.app_secret", "secret1"); err != nil {
		t.Fatalf("SetValue error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	if !strings.Contains(s, "id1") || !strings.Contains(s, "secret1") {
		t.Errorf("second set lost first value, file:\n%s", s)
	}
}

// TestMaskSensitive 验证脱敏逻辑：长值首尾可见、短值固定掩码、空值不变。
func TestMaskSensitive(t *testing.T) {
	cases := map[string]string{
		"":                      "",
		"short":                 "********",
		"super_secret_abcdefgh": "supe********efgh",
	}
	for in, want := range cases {
		if got := MaskSensitive(in); got != want {
			t.Errorf("MaskSensitive(%q) = %q, want %q", in, got, want)
		}
	}
}
