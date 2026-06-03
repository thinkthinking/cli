package clipboard

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"
)

// TestNewReturnsPlatformWriter 验证 New 按平台返回正确实现类型。
func TestNewReturnsPlatformWriter(t *testing.T) {
	w := New()
	switch runtime.GOOS {
	case "darwin":
		if _, ok := w.(*macWriter); !ok {
			t.Errorf("darwin should get *macWriter, got %T", w)
		}
	default:
		if _, ok := w.(*unsupportedWriter); !ok {
			t.Errorf("non-darwin should get *unsupportedWriter, got %T", w)
		}
	}
}

// TestUnsupportedWriterReturnsErr 验证非支持平台实现恒返回 ErrUnsupported。
func TestUnsupportedWriterReturnsErr(t *testing.T) {
	var w Writer = &unsupportedWriter{}
	err := w.WriteHTML(context.Background(), "<p>hi</p>", "hi")
	if !errors.Is(err, ErrUnsupported) {
		t.Errorf("WriteHTML err = %v, want ErrUnsupported", err)
	}
}

// TestWriteTempFile 验证临时文件写入与内容正确。
func TestWriteTempFile(t *testing.T) {
	const content = "<p>临时文件 & <b>内容</b></p>"
	path, err := writeTempFile("thinkthinking-test-*.html", content)
	if err != nil {
		t.Fatalf("writeTempFile error: %v", err)
	}
	defer os.Remove(path)

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if string(got) != content {
		t.Errorf("content = %q, want %q", got, content)
	}
}

// TestMacWriterRealClipboard 在 macOS 上做真实写入并用 `osascript -e 'clipboard info'`
// 校验剪贴板出现 «class HTML» flavor。会改写系统剪贴板，故用 -short 跳过，
// 在非 darwin / 无 osascript 环境自动跳过。
func TestMacWriterRealClipboard(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping real clipboard write in -short mode")
	}
	if runtime.GOOS != "darwin" {
		t.Skip("clipboard real write only verifiable on darwin")
	}
	if _, err := exec.LookPath("osascript"); err != nil {
		t.Skip("osascript not available")
	}

	w := &macWriter{}
	if err := w.WriteHTML(context.Background(), `<h2 style="color:#2563eb">标题</h2>`, "标题"); err != nil {
		t.Fatalf("WriteHTML error: %v", err)
	}

	out, err := exec.Command("osascript", "-e", "clipboard info").CombinedOutput()
	if err != nil {
		t.Fatalf("clipboard info: %v (%s)", err, out)
	}
	if !strings.Contains(string(out), "HTML") {
		t.Errorf("clipboard info missing HTML flavor: %s", out)
	}
}
