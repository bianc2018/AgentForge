package configresolver

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// changeToTempDir 切换到临时目录并返回恢复函数。
func changeToTempDir(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("获取当前工作目录失败: %v", err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("切换到临时目录失败: %v", err)
	}
	t.Cleanup(func() {
		os.Chdir(origDir) // 测试结束后恢复原目录
	})
	return tmpDir
}

// --- UT-13: ConfigResolver.Resolve ---

func TestResolve_DefaultPath(t *testing.T) {
	// 默认路径：未指定 -c 时返回 $(pwd)/coding-config
	tmpDir := changeToTempDir(t)

	got, err := Resolve("")
	if err != nil {
		t.Fatalf("Resolve(\"\") 返回错误: %v", err)
	}

	want := filepath.Join(tmpDir, DefaultConfigDirName)
	if got != want {
		t.Errorf("Resolve(\"\") = %q, want %q", got, want)
	}
}

func TestResolve_CustomPath(t *testing.T) {
	// 自定义路径：-c /path/to/config 返回 /path/to/config
	got, err := Resolve("/path/to/config")
	if err != nil {
		t.Fatalf("Resolve(\"/path/to/config\") 返回错误: %v", err)
	}

	if got != "/path/to/config" {
		t.Errorf("Resolve(\"/path/to/config\") = %q, want %q", got, "/path/to/config")
	}
}

func TestResolve_RelativePath(t *testing.T) {
	// 相对路径：-c ./config 解析为绝对路径
	tmpDir := changeToTempDir(t)

	got, err := Resolve("./config")
	if err != nil {
		t.Fatalf("Resolve(\"./config\") 返回错误: %v", err)
	}

	want := filepath.Join(tmpDir, "config")
	if got != want {
		t.Errorf("Resolve(\"./config\") = %q, want %q", got, want)
	}

	// 验证是绝对路径
	if !filepath.IsAbs(got) {
		t.Errorf("Resolve(\"./config\") 返回非绝对路径: %q", got)
	}
}

func TestResolve_PathDoesNotCreateDirectory(t *testing.T) {
	// 路径不存在时仅返回路径，不创建目录
	nonexistentPath := "/tmp/agent-forge-test-nonexistent-dir"

	got, err := Resolve(nonexistentPath)
	if err != nil {
		t.Fatalf("Resolve(%q) 返回错误: %v", nonexistentPath, err)
	}

	if got != nonexistentPath {
		t.Errorf("Resolve(%q) = %q, want %q", nonexistentPath, got, nonexistentPath)
	}

	// 验证目录未被创建
	if _, err := os.Stat(nonexistentPath); !os.IsNotExist(err) {
		t.Errorf("目录不应被创建，但 %s 存在", nonexistentPath)
	}
}

// TestNew_EndpointsDirDerivedPath 验证基于配置目录的派生路径正确
func TestNew_EndpointsDirDerivedPath(t *testing.T) {
	tmpDir := changeToTempDir(t)

	r, err := New("")
	if err != nil {
		t.Fatalf("New(\"\") 返回错误: %v", err)
	}

	want := filepath.Join(tmpDir, DefaultConfigDirName, "endpoints")
	if got := r.EndpointsDir(); got != want {
		t.Errorf("EndpointsDir() = %q, want %q", got, want)
	}
}

// TestNew_AgentConfigDirDerivedPath 验证 agent 配置目录派生路径正确
func TestNew_AgentConfigDirDerivedPath(t *testing.T) {
	r, err := New("/test-config")
	if err != nil {
		t.Fatalf("New(\"/test-config\") 返回错误: %v", err)
	}

	want := "/test-config/agents/claude"
	if got := r.AgentConfigDir("claude"); got != want {
		t.Errorf("AgentConfigDir(\"claude\") = %q, want %q", got, want)
	}
}

// TestResolve_NonExistentPath 验证不存在的路径也能被解析（不创建目录）
func TestNew_NonExistentPathDoesNotCreateDir(t *testing.T) {
	r, err := New("/nonexistent/path")
	if err != nil {
		t.Fatalf("New(\"/nonexistent/path\") 返回错误: %v", err)
	}
	if r.ConfigDir() != "/nonexistent/path" {
		t.Errorf("ConfigDir() = %q, want %q", r.ConfigDir(), "/nonexistent/path")
	}
}

// TestNew_NewWithEmptyString 验证空字符串使用默认路径
func TestNew_DefaultPath(t *testing.T) {
	tmpDir := changeToTempDir(t)

	r, err := New("")
	if err != nil {
		t.Fatalf("New(\"\") 返回错误: %v", err)
	}

	want := filepath.Join(tmpDir, DefaultConfigDirName)
	if r.ConfigDir() != want {
		t.Errorf("New(\"\").ConfigDir() = %q, want %q", r.ConfigDir(), want)
	}
}

// TestNew_RelativePathConvertedToAbsolute 验证相对路径被正确转换为绝对路径
func TestNew_RelativePathConvertedToAbsolute(t *testing.T) {
	tmpDir := changeToTempDir(t)

	r, err := New("./my-config")
	if err != nil {
		t.Fatalf("New(\"./my-config\") 返回错误: %v", err)
	}

	if !filepath.IsAbs(r.ConfigDir()) {
		t.Errorf("相对路径应被转换为绝对路径，got %q", r.ConfigDir())
	}
	if !strings.HasPrefix(r.ConfigDir(), tmpDir) {
		t.Errorf("绝对路径应以临时目录开头，got %q", r.ConfigDir())
	}
}
