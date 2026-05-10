// Package argspersistence 提供 ArgsPersistence 的单元测试。
//
// 本文件覆盖 UT-5 (ArgsPersistence.Save) 和 UT-6 (ArgsPersistence.Load)，
// 使用 t.TempDir() 模拟文件系统，验证持久化和恢复逻辑的正确性。
package argspersistence

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/agent-forge/cli/internal/shared/argsparser"
)

// setupPersistence 创建一个使用临时目录的 Persistence 实例用于测试。
func setupPersistence(t *testing.T) (*Persistence, string) {
	t.Helper()
	configDir := t.TempDir()
	p := New(configDir)
	return p, configDir
}

// lastArgsPath 返回指定配置目录下 .last_args 文件的路径。
func lastArgsPath(configDir string) string {
	return filepath.Join(configDir, LastArgsFileName)
}

// assertFileContent 验证文件内容包含指定的 key=value 对。
func assertFileContains(t *testing.T, content, key, value string) {
	t.Helper()
	expected := key + "=" + value
	if !strings.Contains(content, expected) {
		t.Errorf("文件内容应包含 %q, 实际内容: %s", expected, content)
	}
}

// --- UT-5: ArgsPersistence.Save() ---

// TestSave_Normal 验证正常保存所有参数字段。
//
// 覆盖案例：正常保存 — 所有参数字段准确写入文件
func TestSave_Normal(t *testing.T) {
	p, configDir := setupPersistence(t)
	params := argsparser.RunParams{
		Agent:   "claude",
		Ports:   []string{"3000:3000"},
		Mounts:  []string{"/host/data"},
		Workdir: "/workspace",
		Envs:    []string{"OPENAI_KEY=sk-xxx"},
		Docker:  false,
		RunCmd:  "",
	}

	if err := p.Save(params); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// 验证文件存在
	if _, err := os.Stat(lastArgsPath(configDir)); os.IsNotExist(err) {
		t.Fatal(".last_args 文件未创建")
	}

	// 读取并验证文件内容
	data, err := os.ReadFile(lastArgsPath(configDir))
	if err != nil {
		t.Fatalf("读取 .last_args 文件失败: %v", err)
	}
	content := string(data)

	assertFileContains(t, content, "AGENT", "claude")
	assertFileContains(t, content, "PORTS", "3000:3000")
	assertFileContains(t, content, "MOUNTS", "/host/data")
	assertFileContains(t, content, "WORKDIR", "/workspace")
	assertFileContains(t, content, "ENVS", "OPENAI_KEY=sk-xxx")
	assertFileContains(t, content, "MODE", "normal")
	assertFileContains(t, content, "RUN_CMD", "")
	assertFileContains(t, content, "DIND", "false")
}

// TestSave_MultiPorts 验证多端口映射的正确序列化。
//
// 覆盖案例：端口映射多值 — -p 3000:3000 -p 8080:8080 正确序列化
func TestSave_MultiPorts(t *testing.T) {
	p, configDir := setupPersistence(t)
	params := argsparser.RunParams{
		Ports: []string{"3000:3000", "8080:8080", "9090:9090"},
	}

	if err := p.Save(params); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	data, err := os.ReadFile(lastArgsPath(configDir))
	if err != nil {
		t.Fatalf("读取 .last_args 文件失败: %v", err)
	}
	content := string(data)

	expected := "PORTS=3000:3000 8080:8080 9090:9090"
	if !strings.Contains(content, expected) {
		t.Errorf("多端口映射序列化错误:\n期望包含: %q\n实际内容: %s", expected, content)
	}
}

// TestSave_MultiEnvs 验证多环境变量的正确序列化。
//
// 覆盖案例：环境变量多值 — -e KEY1=VAL1 -e KEY2=VAL2 正确序列化
func TestSave_MultiEnvs(t *testing.T) {
	p, configDir := setupPersistence(t)
	params := argsparser.RunParams{
		Envs: []string{"KEY1=VAL1", "KEY2=VAL2", "KEY3=VAL3"},
	}

	if err := p.Save(params); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	data, err := os.ReadFile(lastArgsPath(configDir))
	if err != nil {
		t.Fatalf("读取 .last_args 文件失败: %v", err)
	}
	content := string(data)

	expected := "ENVS=KEY1=VAL1 KEY2=VAL2 KEY3=VAL3"
	if !strings.Contains(content, expected) {
		t.Errorf("多环境变量序列化错误:\n期望包含: %q\n实际内容: %s", expected, content)
	}
}

// TestSave_MultiMounts 验证多挂载路径的正确序列化。
//
// 覆盖案例：挂载路径多值 — -m /a -m /b 正确序列化
func TestSave_MultiMounts(t *testing.T) {
	p, configDir := setupPersistence(t)
	params := argsparser.RunParams{
		Mounts: []string{"/data", "/config", "/logs"},
	}

	if err := p.Save(params); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	data, err := os.ReadFile(lastArgsPath(configDir))
	if err != nil {
		t.Fatalf("读取 .last_args 文件失败: %v", err)
	}
	content := string(data)

	expected := "MOUNTS=/data /config /logs"
	if !strings.Contains(content, expected) {
		t.Errorf("多挂载路径序列化错误:\n期望包含: %q\n实际内容: %s", expected, content)
	}
}

// TestSave_EmptyFields 验证未提供的参数保存为空值。
//
// 覆盖案例：空值字段 — 未提供的参数保存为空
func TestSave_EmptyFields(t *testing.T) {
	p, configDir := setupPersistence(t)
	params := argsparser.RunParams{}

	if err := p.Save(params); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	data, err := os.ReadFile(lastArgsPath(configDir))
	if err != nil {
		t.Fatalf("读取 .last_args 文件失败: %v", err)
	}
	content := string(data)

	assertFileContains(t, content, "AGENT", "")
	assertFileContains(t, content, "PORTS", "")
	assertFileContains(t, content, "MOUNTS", "")
	assertFileContains(t, content, "WORKDIR", "")
	assertFileContains(t, content, "ENVS", "")
	assertFileContains(t, content, "RUN_CMD", "")
	assertFileContains(t, content, "DIND", "false")
}

// TestSave_FilePermission 验证写入后文件权限为 0600。
//
// 覆盖案例：写入权限 — 文件写入后权限为 0600
func TestSave_FilePermission(t *testing.T) {
	p, configDir := setupPersistence(t)
	params := argsparser.RunParams{
		Agent: "claude",
	}

	if err := p.Save(params); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	info, err := os.Stat(lastArgsPath(configDir))
	if err != nil {
		t.Fatalf("Stat .last_args 失败: %v", err)
	}

	// 验证文件权限为 0600
	got := info.Mode().Perm()
	want := os.FileMode(0600)
	if got != want {
		t.Errorf("文件权限 = %o, 期望 %o", got, want)
	}
}

// TestSave_DockerMode 验证 Docker 模式下 MODE 和 DIND 字段。
func TestSave_DockerMode(t *testing.T) {
	p, configDir := setupPersistence(t)
	params := argsparser.RunParams{
		Agent:  "claude",
		Docker: true,
	}

	if err := p.Save(params); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	data, err := os.ReadFile(lastArgsPath(configDir))
	if err != nil {
		t.Fatalf("读取 .last_args 文件失败: %v", err)
	}
	content := string(data)

	assertFileContains(t, content, "MODE", "docker")
	assertFileContains(t, content, "DIND", "true")
}

// TestSave_RunCommandMode 验证 --run 模式下 MODE 字段。
func TestSave_RunCommandMode(t *testing.T) {
	p, configDir := setupPersistence(t)
	params := argsparser.RunParams{
		RunCmd: "npm test",
	}

	if err := p.Save(params); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	data, err := os.ReadFile(lastArgsPath(configDir))
	if err != nil {
		t.Fatalf("读取 .last_args 文件失败: %v", err)
	}
	content := string(data)

	assertFileContains(t, content, "MODE", "run")
	assertFileContains(t, content, "RUN_CMD", "npm test")
}

// --- UT-6: ArgsPersistence.Load() ---

// TestLoad_Normal 验证正常加载所有字段。
//
// 覆盖案例：正常加载 — 正确解析所有字段返回结构化参数
func TestLoad_Normal(t *testing.T) {
	p, _ := setupPersistence(t)
	// 先保存再加载，验证往返完整性
	original := argsparser.RunParams{
		Agent:   "claude",
		Ports:   []string{"3000:3000", "8080:8080"},
		Mounts:  []string{"/host/data", "/host/config"},
		Workdir: "/workspace",
		Envs:    []string{"OPENAI_KEY=sk-xxx", "DEBUG=true"},
		Docker:  false,
		RunCmd:  "",
	}

	if err := p.Save(original); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	loaded, err := p.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// 验证所有字段
	if loaded.Agent != original.Agent {
		t.Errorf("Agent = %q, want %q", loaded.Agent, original.Agent)
	}
	if len(loaded.Ports) != len(original.Ports) {
		t.Errorf("len(Ports) = %d, want %d", len(loaded.Ports), len(original.Ports))
	}
	for i := range original.Ports {
		if loaded.Ports[i] != original.Ports[i] {
			t.Errorf("Ports[%d] = %q, want %q", i, loaded.Ports[i], original.Ports[i])
		}
	}
	if len(loaded.Mounts) != len(original.Mounts) {
		t.Errorf("len(Mounts) = %d, want %d", len(loaded.Mounts), len(original.Mounts))
	}
	for i := range original.Mounts {
		if loaded.Mounts[i] != original.Mounts[i] {
			t.Errorf("Mounts[%d] = %q, want %q", i, loaded.Mounts[i], original.Mounts[i])
		}
	}
	if loaded.Workdir != original.Workdir {
		t.Errorf("Workdir = %q, want %q", loaded.Workdir, original.Workdir)
	}
	if len(loaded.Envs) != len(original.Envs) {
		t.Errorf("len(Envs) = %d, want %d", len(loaded.Envs), len(original.Envs))
	}
	for i := range original.Envs {
		if loaded.Envs[i] != original.Envs[i] {
			t.Errorf("Envs[%d] = %q, want %q", i, loaded.Envs[i], original.Envs[i])
		}
	}
	if loaded.Docker != original.Docker {
		t.Errorf("Docker = %v, want %v", loaded.Docker, original.Docker)
	}
	if loaded.RunCmd != original.RunCmd {
		t.Errorf("RunCmd = %q, want %q", loaded.RunCmd, original.RunCmd)
	}
}

// TestLoad_FileNotFound 验证 .last_args 不存在时返回 ErrFileNotFound。
//
// 覆盖案例：文件不存在 — 返回 ErrFileNotFound
func TestLoad_FileNotFound(t *testing.T) {
	p, _ := setupPersistence(t)

	_, err := p.Load()
	if err == nil {
		t.Fatal("Load() 应返回错误，但返回 nil")
	}
	if err != ErrFileNotFound {
		t.Errorf("Load() error = %v, want %v", err, ErrFileNotFound)
	}
}
//
// TestLoad_MalformedFile 验证格式错误的文件不会导致崩溃。
//
// 覆盖案例：文件格式错误 — 部分字段缺失时以空值填充，不崩溃
func TestLoad_MalformedFile(t *testing.T) {
	p, configDir := setupPersistence(t)

	// 创建一个包含各种异常行的 .last_args 文件
	content := `AGENT=claude
INVALID_LINE_NO_EQUALS
=
MALFORMED=missing_newline`
	if err := os.WriteFile(lastArgsPath(configDir), []byte(content), 0600); err != nil {
		t.Fatalf("写入测试文件失败: %v", err)
	}

	params, err := p.Load()
	if err != nil {
		t.Fatalf("Load() 即使文件格式错误也不应返回错误, got: %v", err)
	}

	// Agent 应被正确解析
	if params.Agent != "claude" {
		t.Errorf("Agent = %q, want %q", params.Agent, "claude")
	}
	// 其他字段应使用默认值（空值）
	if params.Ports != nil {
		t.Errorf("Ports = %v, want nil", params.Ports)
	}
	if params.Mounts != nil {
		t.Errorf("Mounts = %v, want nil", params.Mounts)
	}
}

// TestLoad_EmptyFile 验证空文件不会导致崩溃，返回空参数集。
//
// 覆盖案例：空文件 — 返回空参数集，不崩溃
func TestLoad_EmptyFile(t *testing.T) {
	p, configDir := setupPersistence(t)

	// 创建空文件
	if err := os.WriteFile(lastArgsPath(configDir), []byte{}, 0600); err != nil {
		t.Fatalf("写入测试文件失败: %v", err)
	}

	params, err := p.Load()
	if err != nil {
		t.Fatalf("Load() 空文件不应返回错误, got: %v", err)
	}

	// 验证返回默认空参数集
	if params.Agent != "" {
		t.Errorf("Agent = %q, want empty", params.Agent)
	}
	if params.Ports != nil {
		t.Errorf("Ports = %v, want nil", params.Ports)
	}
	if params.Mounts != nil {
		t.Errorf("Mounts = %v, want nil", params.Mounts)
	}
	if params.Workdir != "" {
		t.Errorf("Workdir = %q, want empty", params.Workdir)
	}
	if params.Envs != nil {
		t.Errorf("Envs = %v, want nil", params.Envs)
	}
	if params.Docker {
		t.Error("Docker = true, want false")
	}
	if params.RunCmd != "" {
		t.Errorf("RunCmd = %q, want empty", params.RunCmd)
	}
}

// TestLoad_EmptyLines 验证包含空行和注释行的文件能正确加载。
//
// 非覆盖案例但重要的边界情况：空行和注释行应被跳过。
func TestLoad_EmptyLines(t *testing.T) {
	p, configDir := setupPersistence(t)

	content := `AGENT=claude

PORTS=3000:3000 8080:8080
# 这是一行注释
WORKDIR=/workspace`
	if err := os.WriteFile(lastArgsPath(configDir), []byte(content), 0600); err != nil {
		t.Fatalf("写入测试文件失败: %v", err)
	}

	params, err := p.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if params.Agent != "claude" {
		t.Errorf("Agent = %q, want %q", params.Agent, "claude")
	}
	if len(params.Ports) != 2 {
		t.Errorf("len(Ports) = %d, want 2", len(params.Ports))
	}
	if params.Workdir != "/workspace" {
		t.Errorf("Workdir = %q, want %q", params.Workdir, "/workspace")
	}
}

// TestSaveLoad_RoundTrip 验证保存后立即加载得到完全一致的结构化参数集。
//
// 验证所有字段的往返完整性，包括多值字段和 Docker 模式。
func TestSaveLoad_RoundTrip(t *testing.T) {
	p, _ := setupPersistence(t)

	original := argsparser.RunParams{
		Agent:   "opencode",
		Ports:   []string{"3000:3000", "8080:80", "443:443"},
		Mounts:  []string{"/data", "/config"},
		Envs:    []string{"KEY1=val1", "KEY2=val2", "KEY3=val3"},
		Workdir: "/app",
		Docker:  true,
		RunCmd:  "",
	}

	if err := p.Save(original); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	loaded, err := p.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// 逐一比对字段
	if loaded.Agent != original.Agent {
		t.Errorf("Agent = %q, want %q", loaded.Agent, original.Agent)
	}
	if len(loaded.Ports) != len(original.Ports) {
		t.Errorf("len(Ports) = %d, want %d", len(loaded.Ports), len(original.Ports))
	}
	for i := range original.Ports {
		if loaded.Ports[i] != original.Ports[i] {
			t.Errorf("Ports[%d] = %q, want %q", i, loaded.Ports[i], original.Ports[i])
		}
	}
	if len(loaded.Mounts) != len(original.Mounts) {
		t.Errorf("len(Mounts) = %d, want %d", len(loaded.Mounts), len(original.Mounts))
	}
	for i := range original.Mounts {
		if loaded.Mounts[i] != original.Mounts[i] {
			t.Errorf("Mounts[%d] = %q, want %q", i, loaded.Mounts[i], original.Mounts[i])
		}
	}
	if loaded.Workdir != original.Workdir {
		t.Errorf("Workdir = %q, want %q", loaded.Workdir, original.Workdir)
	}
	if len(loaded.Envs) != len(original.Envs) {
		t.Errorf("len(Envs) = %d, want %d", len(loaded.Envs), len(original.Envs))
	}
	for i := range original.Envs {
		if loaded.Envs[i] != original.Envs[i] {
			t.Errorf("Envs[%d] = %q, want %q", i, loaded.Envs[i], original.Envs[i])
		}
	}
	if loaded.Docker != original.Docker {
		t.Errorf("Docker = %v, want %v", loaded.Docker, original.Docker)
	}
	if loaded.RunCmd != original.RunCmd {
		t.Errorf("RunCmd = %q, want %q", loaded.RunCmd, original.RunCmd)
	}
}

// TestLoad_WithRunCmd 验证包含 --run 命令的加载。
func TestLoad_WithRunCmd(t *testing.T) {
	p, configDir := setupPersistence(t)

	content := `AGENT=
PORTS=
MOUNTS=
WORKDIR=
ENVS=
MODE=run
RUN_CMD=npm test -- --watch
DIND=false`
	if err := os.WriteFile(lastArgsPath(configDir), []byte(content), 0600); err != nil {
		t.Fatalf("写入测试文件失败: %v", err)
	}

	params, err := p.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if params.RunCmd != "npm test -- --watch" {
		t.Errorf("RunCmd = %q, want %q", params.RunCmd, "npm test -- --watch")
	}
	if params.Docker {
		t.Error("Docker = true, want false")
	}
}
