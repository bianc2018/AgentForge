package engine

import (
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// mockHTTPClient 实现可 mock 的 HTTP 客户端。
type mockHTTPClient struct {
	statusCode int
	body       string
	err        error
}

func (m *mockHTTPClient) Get(url string) (*http.Response, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &http.Response{
		StatusCode: m.statusCode,
		Body:       io.NopCloser(strings.NewReader(m.body)),
	}, nil
}

// --- UT-14: SelfUpdateEngine 自更新单元测试 ---

// TestUpdate_Success 验证正常更新流程。
func TestUpdate_Success(t *testing.T) {
	tmpDir := t.TempDir()
	currentPath := filepath.Join(tmpDir, "agent-forge")
	backupPath := currentPath + ".bak"

	// 创建当前二进制
	if err := os.WriteFile(currentPath, []byte("current binary content"), 0755); err != nil {
		t.Fatal(err)
	}

	engine := New(
		WithCurrentPath(currentPath),
		WithHTTPClient(&mockHTTPClient{
			statusCode: http.StatusOK,
			body:       "new binary content",
		}),
	)

	if err := engine.Update(); err != nil {
		t.Fatalf("Update() 返回错误: %v", err)
	}

	// 验证新内容已写入
	data, err := os.ReadFile(currentPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "new binary content" {
		t.Errorf("当前二进制内容应为 'new binary content', 实际: %s", data)
	}

	// 验证备份已删除
	if _, err := os.Stat(backupPath); !os.IsNotExist(err) {
		t.Error("备份文件应已被删除")
	}
}

// TestUpdate_DownloadFailed 验证下载失败时回滚。
func TestUpdate_DownloadFailed(t *testing.T) {
	tmpDir := t.TempDir()
	currentPath := filepath.Join(tmpDir, "agent-forge")
	originalContent := "original binary content"

	if err := os.WriteFile(currentPath, []byte(originalContent), 0755); err != nil {
		t.Fatal(err)
	}

	engine := New(
		WithCurrentPath(currentPath),
		WithHTTPClient(&mockHTTPClient{
			err: errors.New("network error"),
		}),
	)

	err := engine.Update()
	if err == nil {
		t.Fatal("Update() 应返回错误")
	}

	// 验证已回滚到原始内容
	data, err := os.ReadFile(currentPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != originalContent {
		t.Errorf("下载失败后应回滚到原始内容, 实际: %s", data)
	}
}

// TestUpdate_HTTPError 验证 HTTP 非 200 状态码时回滚。
func TestUpdate_HTTPError(t *testing.T) {
	tmpDir := t.TempDir()
	currentPath := filepath.Join(tmpDir, "agent-forge")
	originalContent := "original"

	if err := os.WriteFile(currentPath, []byte(originalContent), 0755); err != nil {
		t.Fatal(err)
	}

	engine := New(
		WithCurrentPath(currentPath),
		WithHTTPClient(&mockHTTPClient{
			statusCode: http.StatusNotFound,
			body:       "not found",
		}),
	)

	err := engine.Update()
	if err == nil {
		t.Fatal("Update() 应返回错误")
	}

	// 验证已回滚
	data, _ := os.ReadFile(currentPath)
	if string(data) != originalContent {
		t.Errorf("HTTP 错误时应回滚, 实际: %s", data)
	}
}

// TestUpdate_EmptyDownload 验证下载内容为空时回滚。
func TestUpdate_EmptyDownload(t *testing.T) {
	tmpDir := t.TempDir()
	currentPath := filepath.Join(tmpDir, "agent-forge")
	originalContent := "original"

	if err := os.WriteFile(currentPath, []byte(originalContent), 0755); err != nil {
		t.Fatal(err)
	}

	engine := New(
		WithCurrentPath(currentPath),
		WithHTTPClient(&mockHTTPClient{
			statusCode: http.StatusOK,
			body:       "",
		}),
	)

	err := engine.Update()
	if err == nil {
		t.Fatal("Update() 应返回错误")
	}

	// 验证已回滚
	data, _ := os.ReadFile(currentPath)
	if string(data) != originalContent {
		t.Errorf("空下载内容时应回滚, 实际: %s", data)
	}
}

// TestUpdate_RenameFailed 验证替换二进制失败时回滚。
func TestUpdate_RenameFailed(t *testing.T) {
	tmpDir := t.TempDir()
	currentPath := filepath.Join(tmpDir, "agent-forge")
	originalContent := "original"

	if err := os.WriteFile(currentPath, []byte(originalContent), 0755); err != nil {
		t.Fatal(err)
	}

	engine := New(
		WithCurrentPath(currentPath),
		WithHTTPClient(&mockHTTPClient{
			statusCode: http.StatusOK,
			body:       "new content",
		}),
		WithRename(func(oldpath, newpath string) error {
			return errors.New("rename failed")
		}),
	)

	err := engine.Update()
	if err == nil {
		t.Fatal("Update() 应返回错误")
	}

	// 验证已回滚到原始内容
	data, _ := os.ReadFile(currentPath)
	if string(data) != originalContent {
		t.Errorf("重命名失败时应回滚, 实际: %s", data)
	}
}

// TestUpdate_DetectCurrentPath 验证自动检测当前路径。
func TestUpdate_DetectCurrentPath(t *testing.T) {
	engine := New()
	if engine.currentPath == "" {
		t.Error("不指定 currentPath 时应自动检测可执行文件路径")
	}
}

// TestUpdate_CustomUpdateURL 验证自定义更新 URL。
func TestUpdate_CustomUpdateURL(t *testing.T) {
	engine := New(
		WithUpdateURL("https://example.com/custom-update"),
	)
	if engine.updateURL != "https://example.com/custom-update" {
		t.Errorf("自定义 URL 应为 https://example.com/custom-update, 实际: %s", engine.updateURL)
	}
}

// TestNew_WithOptions 验证各种选项设置。
func TestNew_WithOptions(t *testing.T) {
	tmpDir := t.TempDir()
	currentPath := filepath.Join(tmpDir, "test-binary")
	engine := New(
		WithCurrentPath(currentPath),
		WithHTTPClient(&mockHTTPClient{statusCode: http.StatusOK, body: "test"}),
		WithUpdateURL("https://test.url/update"),
		WithRename(func(old, new string) error { return nil }),
	)

	if engine.currentPath != currentPath {
		t.Error("currentPath 未正确设置")
	}
	if engine.updateURL != "https://test.url/update" {
		t.Errorf("updateURL 未正确设置, 实际: %s", engine.updateURL)
	}
}
