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

// --- New 错误路径测试 ---

func TestNew_GetwdError(t *testing.T) {
	// 空字符串 + Getwd 失败：New 内部调用 os.Getwd() 时应返回错误
	tmpDir, err := os.MkdirTemp("", "configresolver-test-getwd-*")
	if err != nil {
		t.Fatal(err)
	}
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(origDir) })

	if err := os.Remove(tmpDir); err != nil {
		t.Skipf("无法删除当前工作目录: %v", err)
	}

	if _, err := os.Getwd(); err != nil {
		_, err = New("")
		if err == nil {
			t.Error("New(\"\") 应在 Getwd 失败时返回错误")
		}
	}
}

func TestNew_AbsError(t *testing.T) {
	// 相对路径 + Getwd 失败：filepath.Abs 应返回错误
	tmpDir, err := os.MkdirTemp("", "configresolver-test-abs-*")
	if err != nil {
		t.Fatal(err)
	}
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(origDir) })

	if err := os.Remove(tmpDir); err != nil {
		t.Skipf("无法删除当前工作目录: %v", err)
	}

	if _, err := os.Getwd(); err != nil {
		_, err = New("./my-config")
		if err == nil {
			t.Error("New(\"./my-config\") 应在 Getwd 失败时返回错误")
		}
	}
}

// --- AgentConfigDir ---

func TestResolver_AgentConfigDir_EmptyName(t *testing.T) {
	// agentName 为空时应返回 <config-dir>/agents
	r, err := New("/test-config")
	if err != nil {
		t.Fatalf("New(\"/test-config\") 返回错误: %v", err)
	}

	want := "/test-config/agents"
	if got := r.AgentConfigDir(""); got != want {
		t.Errorf("AgentConfigDir(\"\") = %q, want %q", got, want)
	}
}

func TestResolver_AgentConfigDir_TableDriven(t *testing.T) {
	tests := []struct {
		name      string
		agentName string
		want      string
	}{
		{name: "指定 agent 名称", agentName: "my-agent", want: "/base/agents/my-agent"},
		{name: "空 agent 名称返回 agents 目录", agentName: "", want: "/base/agents"},
		{name: "带横线的 agent 名称", agentName: "code-assistant-v2", want: "/base/agents/code-assistant-v2"},
	}

	r, err := New("/base")
	if err != nil {
		t.Fatalf("New(\"/base\") 返回错误: %v", err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := r.AgentConfigDir(tt.agentName)
			if got != tt.want {
				t.Errorf("AgentConfigDir(%q) = %q, want %q", tt.agentName, got, tt.want)
			}
		})
	}
}

// --- EndpointsDir ---

func TestResolver_EndpointsDir(t *testing.T) {
	tests := []struct {
		name       string
		configDir  string
		wantSuffix string
	}{
		{name: "绝对路径", configDir: "/app/config", wantSuffix: "/app/config/endpoints"},
		{name: "相对路径", configDir: "./my-config", wantSuffix: "/endpoints"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := New(tt.configDir)
			if err != nil {
				t.Fatalf("New(%q) 返回错误: %v", tt.configDir, err)
			}
			got := r.EndpointsDir()
			if !strings.HasSuffix(got, tt.wantSuffix) {
				t.Errorf("EndpointsDir() = %q, want suffix %q", got, tt.wantSuffix)
			}
			if !filepath.IsAbs(got) {
				t.Errorf("EndpointsDir() 应返回绝对路径, got %q", got)
			}
		})
	}
}

// --- EnsureEndpointsDir ---

func TestResolver_EnsureEndpointsDir_CreatesDirectory(t *testing.T) {
	// 不存在的目录应被创建
	tmpDir := t.TempDir()

	r, err := New(tmpDir)
	if err != nil {
		t.Fatalf("New(%q) 返回错误: %v", tmpDir, err)
	}

	path, err := r.EnsureEndpointsDir()
	if err != nil {
		t.Fatalf("EnsureEndpointsDir() 返回错误: %v", err)
	}

	want := filepath.Join(tmpDir, "endpoints")
	if path != want {
		t.Errorf("EnsureEndpointsDir() = %q, want %q", path, want)
	}

	// 验证目录确实被创建
	info, err := os.Stat(want)
	if err != nil {
		t.Fatalf("os.Stat(%q) 返回错误: %v", want, err)
	}
	if !info.IsDir() {
		t.Errorf("%q 应是目录", want)
	}
}

func TestResolver_EnsureEndpointsDir_AlreadyExists(t *testing.T) {
	// 目录已存在时不应返回错误
	tmpDir := t.TempDir()
	endpointsDir := filepath.Join(tmpDir, "endpoints")
	if err := os.MkdirAll(endpointsDir, 0755); err != nil {
		t.Fatalf("os.MkdirAll(%q) 返回错误: %v", endpointsDir, err)
	}

	r, err := New(tmpDir)
	if err != nil {
		t.Fatalf("New(%q) 返回错误: %v", tmpDir, err)
	}

	path, err := r.EnsureEndpointsDir()
	if err != nil {
		t.Fatalf("EnsureEndpointsDir() 返回错误: %v", err)
	}
	if path != endpointsDir {
		t.Errorf("EnsureEndpointsDir() = %q, want %q", path, endpointsDir)
	}
}

func TestResolver_EnsureEndpointsDir_Error(t *testing.T) {
	// 父路径为文件时 MkdirAll 应失败
	tmpDir := t.TempDir()
	barrier := filepath.Join(tmpDir, "barrier")
	if err := os.WriteFile(barrier, []byte(""), 0644); err != nil {
		t.Fatalf("写入文件失败: %v", err)
	}

	r, err := New(barrier)
	if err != nil {
		t.Fatalf("New(%q) 返回错误: %v", barrier, err)
	}

	_, err = r.EnsureEndpointsDir()
	if err == nil {
		t.Error("EnsureEndpointsDir 应在父路径为文件时返回错误")
	}
}

// --- EnsureConfigDir ---

func TestResolver_EnsureConfigDir_CreatesDirectory(t *testing.T) {
	// 不存在的配置目录应被创建
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, "my-config")

	r, err := New(configDir)
	if err != nil {
		t.Fatalf("New(%q) 返回错误: %v", configDir, err)
	}

	path, err := r.EnsureConfigDir()
	if err != nil {
		t.Fatalf("EnsureConfigDir() 返回错误: %v", err)
	}
	if path != configDir {
		t.Errorf("EnsureConfigDir() = %q, want %q", path, configDir)
	}

	// 验证目录确实被创建
	info, err := os.Stat(configDir)
	if err != nil {
		t.Fatalf("os.Stat(%q) 返回错误: %v", configDir, err)
	}
	if !info.IsDir() {
		t.Errorf("%q 应是目录", configDir)
	}
}

func TestResolver_EnsureConfigDir_AlreadyExists(t *testing.T) {
	// 目录已存在时不应返回错误
	tmpDir := t.TempDir()

	r, err := New(tmpDir)
	if err != nil {
		t.Fatalf("New(%q) 返回错误: %v", tmpDir, err)
	}

	path, err := r.EnsureConfigDir()
	if err != nil {
		t.Fatalf("EnsureConfigDir() 返回错误: %v", err)
	}
	if path != tmpDir {
		t.Errorf("EnsureConfigDir() = %q, want %q", path, tmpDir)
	}
}

func TestResolver_EnsureConfigDir_Error(t *testing.T) {
	// 路径为文件时 MkdirAll 应失败
	tmpDir := t.TempDir()
	barrier := filepath.Join(tmpDir, "barrier")
	if err := os.WriteFile(barrier, []byte(""), 0644); err != nil {
		t.Fatalf("写入文件失败: %v", err)
	}

	r, err := New(barrier)
	if err != nil {
		t.Fatalf("New(%q) 返回错误: %v", barrier, err)
	}

	_, err = r.EnsureConfigDir()
	if err == nil {
		t.Error("EnsureConfigDir 应在路径为文件时返回错误")
	}
}

// --- IsDefaultConfigDir ---

func TestResolver_IsDefaultConfigDir(t *testing.T) {
	tmpDir := changeToTempDir(t)

	tests := []struct {
		name      string
		configDir string
		want      bool
	}{
		{name: "默认路径应返回 true", configDir: "", want: true},
		{name: "自定义绝对路径应返回 false", configDir: "/tmp/custom-config", want: false},
		{name: "自定义相对路径应返回 false", configDir: "./my-config", want: false},
		{name: "显式指定默认路径应返回 true", configDir: filepath.Join(tmpDir, DefaultConfigDirName), want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := New(tt.configDir)
			if err != nil {
				t.Fatalf("New(%q) 返回错误: %v", tt.configDir, err)
			}
			if got := r.IsDefaultConfigDir(); got != tt.want {
				t.Errorf("IsDefaultConfigDir() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestResolver_IsDefaultConfigDir_GetwdError(t *testing.T) {
	// 先获取有效 cwd 创建默认路径 resolver，再破坏 cwd，验证 IsDefaultConfigDir 返回 false
	tmpDir, err := os.MkdirTemp("", "configresolver-test-isdefault-*")
	if err != nil {
		t.Fatal(err)
	}
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(origDir) })

	// 在 cwd 有效时创建 Resolver（使用默认路径）
	r, err := New("")
	if err != nil {
		t.Fatalf("New(\"\") 返回错误: %v", err)
	}

	// 现在破坏 cwd
	if err := os.Remove(tmpDir); err != nil {
		t.Skipf("无法删除当前工作目录: %v", err)
	}

	// 验证 Getwd 确实失败了，然后检查 IsDefaultConfigDir 返回 false
	if _, err := os.Getwd(); err != nil {
		if got := r.IsDefaultConfigDir(); got != false {
			t.Errorf("IsDefaultConfigDir() = %v, want false (Getwd 失败)", got)
		}
	}
}

// --- Resolve 错误传播 ---

func TestResolve_ErrorFromNew(t *testing.T) {
	// Resolve 在 New 失败时应传播错误
	tmpDir, err := os.MkdirTemp("", "configresolver-test-resolve-*")
	if err != nil {
		t.Fatal(err)
	}
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(origDir) })

	if err := os.Remove(tmpDir); err != nil {
		t.Skipf("无法删除当前工作目录: %v", err)
	}

	if _, err := os.Getwd(); err != nil {
		_, err = Resolve("")
		if err == nil {
			t.Error("Resolve(\"\") 应在 New 失败时返回错误")
		}
	}
}

// --- StandardErrors ---

func TestStandardErrors_AreDefined(t *testing.T) {
	// 验证所有标准错误 sentinel 已定义且非空
	if ErrGetwdFailed == nil {
		t.Error("ErrGetwdFailed 不应为 nil")
	}
	if ErrAbsPathFailed == nil {
		t.Error("ErrAbsPathFailed 不应为 nil")
	}

	// 验证错误消息符合预期
	if ErrGetwdFailed.Error() != "无法获取当前工作目录" {
		t.Errorf("ErrGetwdFailed.Error() = %q, want %q", ErrGetwdFailed.Error(), "无法获取当前工作目录")
	}
	if ErrAbsPathFailed.Error() != "无法将路径转换为绝对路径" {
		t.Errorf("ErrAbsPathFailed.Error() = %q, want %q", ErrAbsPathFailed.Error(), "无法将路径转换为绝对路径")
	}
}

// --- ConfigDir getter ---

func TestResolver_ConfigDir_Getter(t *testing.T) {
	tests := []struct {
		name      string
		configDir string
	}{
		{name: "根目录", configDir: "/"},
		{name: "深层路径", configDir: "/a/b/c/d"},
		{name: "带点的路径", configDir: "/home/user/.config"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := New(tt.configDir)
			if err != nil {
				t.Fatalf("New(%q) 返回错误: %v", tt.configDir, err)
			}
			got := r.ConfigDir()
			if got != tt.configDir {
				t.Errorf("ConfigDir() = %q, want %q", got, tt.configDir)
			}
		})
	}
}

// --- Resolve 边界情况 ---

func TestResolve_NonExistentPath_DoesNotCreate(t *testing.T) {
	// Resolve 不应在文件系统上创建任何目录
	tmpDir := t.TempDir()
	nonExistent := filepath.Join(tmpDir, "does-not-exist")

	got, err := Resolve(nonExistent)
	if err != nil {
		t.Fatalf("Resolve(%q) 返回错误: %v", nonExistent, err)
	}
	if got != nonExistent {
		t.Errorf("Resolve(%q) = %q, want %q", nonExistent, got, nonExistent)
	}

	// 验证目录未被创建
	if _, err := os.Stat(nonExistent); !os.IsNotExist(err) {
		t.Errorf("目录不应被创建，但 %s 存在", nonExistent)
	}
}

// --- Resolve 默认路径不依赖于先前 cwd 状态 ---

func TestResolve_DefaultPath_RelativeToCwd(t *testing.T) {
	// 默认路径应相对于当前工作目录
	tmpDir := changeToTempDir(t)

	got, err := Resolve("")
	if err != nil {
		t.Fatalf("Resolve(\"\") 返回错误: %v", err)
	}

	want := filepath.Join(tmpDir, DefaultConfigDirName)
	if got != want {
		t.Errorf("Resolve(\"\") = %q, want %q", got, want)
	}
	if !filepath.IsAbs(got) {
		t.Errorf("Resolve(\"\") 应返回绝对路径, got %q", got)
	}
}
