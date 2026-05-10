// Package argspersistence 提供 ArgsPersistence 的集成测试（IT-3）。
//
// 本文件覆盖 IT-3 的所有案例，在真实临时文件系统上验证 .last_args 文件的
// 持久化和恢复功能。测试覆盖：save 写入、load 读取、文件不存在、
// 数据往返完整性、多值参数序列化/反序列化。
package argspersistence

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/agent-forge/cli/internal/shared/argsparser"
)

// configDir 返回集成测试用的配置目录。
// 使用 t.TempDir() 确保测试间隔离和自动清理。
func configDir(t *testing.T) string {
	t.Helper()
	return t.TempDir()
}

// itAssertFileContains 验证文件内容包含指定的 key=value 对。
// 前缀 it 避免与 persistence_test.go 中的 assertFileContains 冲突。
func itAssertFileContains(t *testing.T, content, key, value string) {
	t.Helper()
	expected := key + "=" + value
	if !strings.Contains(content, expected) {
		t.Errorf("文件内容应包含 %q, 实际内容: %s", expected, content)
	}
}

// TestIT3_Save_WritesCorrectContent 验证 save 写入后文件内容与传入参数完全一致。
//
// 覆盖案例：save 写入 — 保存后文件内容与传入参数完全一致
func TestIT3_Save_WritesCorrectContent(t *testing.T) {
	p := New(configDir(t))

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

	// 读取文件内容验证
	data, err := os.ReadFile(p.filePath())
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	content := string(data)

	itAssertFileContains(t, content, "AGENT", "claude")
	itAssertFileContains(t, content, "PORTS", "3000:3000")
	itAssertFileContains(t, content, "MOUNTS", "/host/data")
	itAssertFileContains(t, content, "WORKDIR", "/workspace")
	itAssertFileContains(t, content, "ENVS", "OPENAI_KEY=sk-xxx")
	itAssertFileContains(t, content, "MODE", "normal")
	itAssertFileContains(t, content, "RUN_CMD", "")
	itAssertFileContains(t, content, "DIND", "false")
}

// TestIT3_Load_ReadsCorrectParams 验证 load 读取 .last_args 还原为结构化参数。
//
// 覆盖案例：load 读取 — 读取 .last_args 还原为结构化参数
func TestIT3_Load_ReadsCorrectParams(t *testing.T) {
	dir := configDir(t)
	p := New(dir)

	// 先保存一组参数
	original := argsparser.RunParams{
		Agent:   "opencode",
		Ports:   []string{"3000:3000", "8080:80"},
		Mounts:  []string{"/data", "/config"},
		Workdir: "/app",
		Envs:    []string{"KEY1=val1", "KEY2=val2"},
		Docker:  false,
		RunCmd:  "",
	}

	if err := p.Save(original); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// 从文件加载
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

// TestIT3_Load_FileNotFound 验证 .last_args 不存在时返回 ErrFileNotFound。
//
// 覆盖案例：load 文件不存在 — 返回 ErrFileNotFound
func TestIT3_Load_FileNotFound(t *testing.T) {
	p := New(configDir(t))

	_, err := p.Load()
	if err == nil {
		t.Fatal("Load() 应返回错误，但返回 nil")
	}
	if err != ErrFileNotFound {
		t.Errorf("Load() error = %v, want %v", err, ErrFileNotFound)
	}
}

// TestIT3_RoundTrip_DataIntegrity 验证数据往返完整性。
//
// 覆盖案例：保存后立即读取 — 数据往返无误（同一配置 save 再 load 得到相同参数集）
//
// 验证全部 8 个字段的往返完整性，包括多值字段和 Docker 模式。
func TestIT3_RoundTrip_DataIntegrity(t *testing.T) {
	p := New(configDir(t))

	original := argsparser.RunParams{
		Agent:   "deepseek-tui",
		Ports:   []string{"3000:3000", "8080:80", "443:443"},
		Mounts:  []string{"/data", "/config", "/logs"},
		Envs:    []string{"API_KEY=sk-test", "DEBUG=true", "LOG_LEVEL=info"},
		Workdir: "/home",
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

	// full field comparison
	if loaded.Agent != original.Agent {
		t.Errorf("Agent = %q, want %q", loaded.Agent, original.Agent)
	}
	if len(loaded.Ports) != len(original.Ports) {
		t.Errorf("len(Ports) = %d, want %d", len(loaded.Ports), len(original.Ports))
	} else {
		for i := range original.Ports {
			if loaded.Ports[i] != original.Ports[i] {
				t.Errorf("Ports[%d] = %q, want %q", i, loaded.Ports[i], original.Ports[i])
			}
		}
	}
	if len(loaded.Mounts) != len(original.Mounts) {
		t.Errorf("len(Mounts) = %d, want %d", len(loaded.Mounts), len(original.Mounts))
	} else {
		for i := range original.Mounts {
			if loaded.Mounts[i] != original.Mounts[i] {
				t.Errorf("Mounts[%d] = %q, want %q", i, loaded.Mounts[i], original.Mounts[i])
			}
		}
	}
	if loaded.Workdir != original.Workdir {
		t.Errorf("Workdir = %q, want %q", loaded.Workdir, original.Workdir)
	}
	if len(loaded.Envs) != len(original.Envs) {
		t.Errorf("len(Envs) = %d, want %d", len(loaded.Envs), len(original.Envs))
	} else {
		for i := range original.Envs {
			if loaded.Envs[i] != original.Envs[i] {
				t.Errorf("Envs[%d] = %q, want %q", i, loaded.Envs[i], original.Envs[i])
			}
		}
	}
	if loaded.Docker != original.Docker {
		t.Errorf("Docker = %v, want %v", loaded.Docker, original.Docker)
	}
	if loaded.RunCmd != original.RunCmd {
		t.Errorf("RunCmd = %q, want %q", loaded.RunCmd, original.RunCmd)
	}
}

// TestIT3_MultiValue_RoundTrip 验证包含多值参数的数据往返完整性。
//
// 覆盖案例：包含多值参数（多端口、多挂载）— 正确序列化和反序列化
//
// 验证多端口、多环境变量、多挂载路径在 save→load 后完全一致。
func TestIT3_MultiValue_RoundTrip(t *testing.T) {
	p := New(configDir(t))

	// 5 个端口映射、4 个挂载路径、3 个环境变量
	original := argsparser.RunParams{
		Agent:   "kimi",
		Ports:   []string{"8080:80", "3000:3000", "9090:9090", "5432:5432", "6379:6379"},
		Mounts:  []string{"/mnt/data", "/mnt/config", "/var/log", "/opt/cache"},
		Envs:    []string{"KIMI_KEY=sk-test", "LOG_DIR=/var/log", "CACHE_DIR=/opt/cache"},
		Workdir: "/app",
	}

	if err := p.Save(original); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	loaded, err := p.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// 验证多值参数完整性
	if len(loaded.Ports) != len(original.Ports) {
		t.Errorf("len(Ports) = %d, want %d", len(loaded.Ports), len(original.Ports))
	} else {
		for i := range original.Ports {
			if loaded.Ports[i] != original.Ports[i] {
				t.Errorf("Ports[%d] = %q, want %q", i, loaded.Ports[i], original.Ports[i])
			}
		}
	}
	if len(loaded.Mounts) != len(original.Mounts) {
		t.Errorf("len(Mounts) = %d, want %d", len(loaded.Mounts), len(original.Mounts))
	} else {
		for i := range original.Mounts {
			if loaded.Mounts[i] != original.Mounts[i] {
				t.Errorf("Mounts[%d] = %q, want %q", i, loaded.Mounts[i], original.Mounts[i])
			}
		}
	}
	if len(loaded.Envs) != len(original.Envs) {
		t.Errorf("len(Envs) = %d, want %d", len(loaded.Envs), len(original.Envs))
	} else {
		for i := range original.Envs {
			if loaded.Envs[i] != original.Envs[i] {
				t.Errorf("Envs[%d] = %q, want %q", i, loaded.Envs[i], original.Envs[i])
			}
		}
	}
}

// TestIT3_RunCommand_RoundTrip 验证 --run 参数在 save/load 后的完整性。
//
// 覆盖边界情况：包含 --run 带空格和特殊字符的命令。
func TestIT3_RunCommand_RoundTrip(t *testing.T) {
	p := New(configDir(t))

	original := argsparser.RunParams{
		RunCmd: "npm test -- --coverage --watch",
	}

	if err := p.Save(original); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	loaded, err := p.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if loaded.RunCmd != original.RunCmd {
		t.Errorf("RunCmd = %q, want %q", loaded.RunCmd, original.RunCmd)
	}
	if loaded.Docker {
		t.Error("Docker = true, want false")
	}
}

// TestIT3_FilePermission 验证 save 后文件权限为 0600。
//
// NFR-9 compliance check: only owner can read/write.
func TestIT3_FilePermission(t *testing.T) {
	p := New(configDir(t))

	params := argsparser.RunParams{
		Agent: "claude",
	}

	if err := p.Save(params); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	info, err := os.Stat(p.filePath())
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}

	got := info.Mode().Perm()
	want := os.FileMode(0600)
	if got != want {
		t.Errorf("file permission = %o, want %o", got, want)
	}
}

// TestIT3_DockerMode_SaveLoad 验证 Docker 模式的完整 save/load 周期。
//
// 验证 DIND=true 时 MODE=docker 被正确写入和读取。
func TestIT3_DockerMode_SaveLoad(t *testing.T) {
	p := New(configDir(t))

	original := argsparser.RunParams{
		Agent:  "claude",
		Docker: true,
	}

	if err := p.Save(original); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// 验证文件包含正确的模式标记
	data, err := os.ReadFile(p.filePath())
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	content := string(data)
	itAssertFileContains(t, content, "MODE", "docker")
	itAssertFileContains(t, content, "DIND", "true")

	// load 验证
	loaded, err := p.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !loaded.Docker {
		t.Error("Docker = false, want true")
	}
}

// TestIT3_ConfigDirCreated 验证 save 时自动创建不存在的配置目录。
//
// 验证 Persistence.Save 在 configDir 不存在时自动创建。
func TestIT3_ConfigDirCreated(t *testing.T) {
	// 使用深层不存在的路径
	base := configDir(t)
	deepDir := filepath.Join(base, "deep", "nested", "config")
	p := New(deepDir)

	params := argsparser.RunParams{
		Agent: "claude",
	}

	if err := p.Save(params); err != nil {
		t.Fatalf("Save() 即使在深层路径也应自动创建目录, error = %v", err)
	}

	// 验证目录和文件存在
	if _, err := os.Stat(deepDir); os.IsNotExist(err) {
		t.Fatal("配置目录未被自动创建")
	}
	if _, err := os.Stat(filepath.Join(deepDir, LastArgsFileName)); os.IsNotExist(err) {
		t.Fatal(".last_args 文件未被创建")
	}
}

// TestIT3_EmptyValues_RoundTrip 验证空值字段的 save/load 完整性。
//
// 验证所有字段为空时 save 后 load 不崩溃且返回空参数集。
func TestIT3_EmptyValues_RoundTrip(t *testing.T) {
	p := New(configDir(t))

	original := argsparser.RunParams{}

	if err := p.Save(original); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	loaded, err := p.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if loaded.Agent != "" {
		t.Errorf("Agent = %q, want empty", loaded.Agent)
	}
	if loaded.Ports != nil {
		t.Errorf("Ports = %v, want nil", loaded.Ports)
	}
	if loaded.Mounts != nil {
		t.Errorf("Mounts = %v, want nil", loaded.Mounts)
	}
	if loaded.Workdir != "" {
		t.Errorf("Workdir = %q, want empty", loaded.Workdir)
	}
	if loaded.Envs != nil {
		t.Errorf("Envs = %v, want nil", loaded.Envs)
	}
	if loaded.Docker {
		t.Error("Docker = true, want false")
	}
	if loaded.RunCmd != "" {
		t.Errorf("RunCmd = %q, want empty", loaded.RunCmd)
	}
}
