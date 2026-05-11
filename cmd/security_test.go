//go:build security

package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestST1_EndpointShowMaskedKey 验证 endpoint show 中 KEY 为掩码格式。
//
// 覆盖案例：endpoint show <name> — KEY 字段输出为掩码格式
// 模拟的攻击向量：肩窥攻击、共享终端屏幕截取
// 可追溯性: NFR-6
func TestST1_EndpointShowMaskedKey(t *testing.T) {
	binaryPath := filepath.Join(t.TempDir(), "agent-forge")
	buildCmd := exec.Command("/tmp/go/bin/go", "build", "-o", binaryPath, ".")
	buildCmd.Dir = ".."
	output, err := buildCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("构建二进制失败: %v\nOutput: %s", err, output)
	}

	configDir := t.TempDir()

	// 创建端点（使用长 key 验证完整掩码）
	addCmd := exec.Command(binaryPath, "endpoint", "add", "my-ep",
		"--provider", "openai",
		"--url", "https://api.openai.com",
		"--key", "sk-test-key-value-12345",
		"--model", "gpt-4",
		"-c", configDir,
	)
	if addOut, addErr := addCmd.CombinedOutput(); addErr != nil {
		t.Fatalf("创建端点失败: %v\n%s", addErr, addOut)
	}

	// Case 1: endpoint show 显示掩码格式
	showCmd := exec.Command(binaryPath, "endpoint", "show", "my-ep", "-c", configDir)
	showOut, _ := showCmd.CombinedOutput()
	showStr := string(showOut)

	// 应显示掩码格式
	if !strings.Contains(showStr, "***") {
		t.Errorf("endpoint show 应显示掩码 (***)，got:\n%s", showStr)
	}
	// 不应显示完整 key
	if strings.Contains(showStr, "sk-test-key-value-12345") {
		t.Error("endpoint show 不应显示完整 API key")
	}
}

// TestST1_EndpointListNoKey 验证 endpoint list 不输出 KEY 字段。
//
// 覆盖案例：endpoint list — 不输出 KEY 字段（仅 NAME/PROVIDER/MODEL）
// 可追溯性: NFR-6
func TestST1_EndpointListNoKey(t *testing.T) {
	binaryPath := filepath.Join(t.TempDir(), "agent-forge")
	buildCmd := exec.Command("/tmp/go/bin/go", "build", "-o", binaryPath, ".")
	buildCmd.Dir = ".."
	output, err := buildCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("构建二进制失败: %v\nOutput: %s", err, output)
	}

	configDir := t.TempDir()

	// 创建端点
	addCmd := exec.Command(binaryPath, "endpoint", "add", "my-ep",
		"--provider", "openai",
		"--url", "https://api.openai.com",
		"--key", "sk-secret-key",
		"--model", "gpt-4",
		"-c", configDir,
	)
	if addOut, addErr := addCmd.CombinedOutput(); addErr != nil {
		t.Fatalf("创建端点失败: %v\n%s", addErr, addOut)
	}

	// Case 2: endpoint list 不输出 KEY
	listCmd := exec.Command(binaryPath, "endpoint", "list", "-c", configDir)
	listOut, _ := listCmd.CombinedOutput()
	listStr := string(listOut)

	// list 应包含 NAME, PROVIDER, MODEL 表头
	if !strings.Contains(listStr, "NAME") {
		t.Error("endpoint list 应包含 NAME 列")
	}
	if !strings.Contains(listStr, "PROVIDER") {
		t.Error("endpoint list 应包含 PROVIDER 列")
	}
	// list 不应包含 KEY 列
	if strings.Contains(listStr, "KEY") {
		t.Error("endpoint list 不应包含 KEY 列")
	}
}

// TestST1_VersionOutputNoLeak 验证 version/info 输出不泄露任何配置信息。
//
// 覆盖案例：version/info 输出 — 不泄露任何配置信息
// 可追溯性: NFR-6
func TestST1_VersionOutputNoLeak(t *testing.T) {
	binaryPath := filepath.Join(t.TempDir(), "agent-forge")
	buildCmd := exec.Command("/tmp/go/bin/go", "build", "-o", binaryPath, ".")
	buildCmd.Dir = ".."
	output, err := buildCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("构建二进制失败: %v\nOutput: %s", err, output)
	}

	// Case 5: version 输出不应泄露配置信息
	verCmd := exec.Command(binaryPath, "version")
	verOut, _ := verCmd.CombinedOutput()
	verStr := string(verOut)

	// version 只能包含版本号和 hash，不应包含 API key、URL 等
	if strings.Contains(verStr, "sk-") {
		t.Error("version 输出不应包含 API key 字符")
	}
	if strings.Contains(verStr, "api.") {
		t.Error("version 输出不应包含 URL 字符")
	}
}

// TestST1_EndpointAddOutputNoFullKey 验证 endpoint add 回显不显示完整 key。
//
// 覆盖案例：endpoint add — 回显确认不显示完整 key
// 可追溯性: NFR-6
func TestST1_EndpointAddOutputNoFullKey(t *testing.T) {
	binaryPath := filepath.Join(t.TempDir(), "agent-forge")
	buildCmd := exec.Command("/tmp/go/bin/go", "build", "-o", binaryPath, ".")
	buildCmd.Dir = ".."
	output, err := buildCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("构建二进制失败: %v\nOutput: %s", err, output)
	}

	configDir := t.TempDir()

	// 创建端点
	addCmd := exec.Command(binaryPath, "endpoint", "add", "my-ep",
		"--provider", "openai",
		"--url", "https://api.openai.com",
		"--key", "sk-secret-key-value",
		"--model", "gpt-4",
		"-c", configDir,
	)
	addOut, _ := addCmd.CombinedOutput()
	addStr := string(addOut)

	// add 的输出不应包含原始 key
	if strings.Contains(addStr, "sk-secret-key-value") {
		t.Error("endpoint add 输出不应显示完整 key")
	}
}

// --- ST-4: 配置文件权限安全（NFR-9） ---

// TestST4_EndpointAddFilePermission 验证 endpoint add 后 endpoint.env 权限为 0600。
//
// 覆盖案例：endpoint add — endpoint.env 文件权限为 0600
// 模拟的攻击向量：多用户系统上其他用户窃取 API key
// 可追溯性: NFR-9
func TestST4_EndpointAddFilePermission(t *testing.T) {
	binaryPath := filepath.Join(t.TempDir(), "agent-forge")
	buildCmd := exec.Command("/tmp/go/bin/go", "build", "-o", binaryPath, ".")
	buildCmd.Dir = ".."
	output, err := buildCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("构建二进制失败: %v\nOutput: %s", err, output)
	}

	configDir := t.TempDir()

	// 创建端点
	addCmd := exec.Command(binaryPath, "endpoint", "add", "my-ep",
		"--provider", "openai",
		"--url", "https://api.openai.com",
		"--key", "sk-test-key",
		"--model", "gpt-4",
		"-c", configDir,
	)
	if addOut, addErr := addCmd.CombinedOutput(); addErr != nil {
		t.Fatalf("创建端点失败: %v\n%s", addErr, addOut)
	}

	// 验证文件权限为 0600
	envPath := filepath.Join(configDir, "endpoints", "my-ep", "endpoint.env")
	info, err := os.Stat(envPath)
	if err != nil {
		t.Fatalf("无法 stat endpoint.env: %v", err)
	}
	perm := info.Mode().Perm()
	if perm != 0600 {
		t.Errorf("endpoint.env 权限为 %o, 期望 0600 (NFR-9)", perm)
	}
}

// TestST4_EndpointSetFilePermission 验证 endpoint set 后修改文件权限仍为 0600。
//
// 覆盖案例：endpoint set — 修改后的文件权限仍为 0600
// 可追溯性: NFR-9
func TestST4_EndpointSetFilePermission(t *testing.T) {
	binaryPath := filepath.Join(t.TempDir(), "agent-forge")
	buildCmd := exec.Command("/tmp/go/bin/go", "build", "-o", binaryPath, ".")
	buildCmd.Dir = ".."
	output, err := buildCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("构建二进制失败: %v\nOutput: %s", err, output)
	}

	configDir := t.TempDir()

	// 创建端点
	addCmd := exec.Command(binaryPath, "endpoint", "add", "my-ep",
		"--provider", "openai",
		"--url", "https://api.openai.com",
		"--key", "sk-test-key",
		"--model", "gpt-4",
		"-c", configDir,
	)
	if addOut, addErr := addCmd.CombinedOutput(); addErr != nil {
		t.Fatalf("创建端点失败: %v\n%s", addErr, addOut)
	}

	// set 修改
	setCmd := exec.Command(binaryPath, "endpoint", "set", "my-ep",
		"--key", "sk-new-key",
		"--model", "gpt-5",
		"-c", configDir,
	)
	if setOut, setErr := setCmd.CombinedOutput(); setErr != nil {
		t.Fatalf("修改端点失败: %v\n%s", setErr, setOut)
	}

	// 验证 set 后文件权限仍为 0600
	envPath := filepath.Join(configDir, "endpoints", "my-ep", "endpoint.env")
	info, err := os.Stat(envPath)
	if err != nil {
		t.Fatalf("无法 stat endpoint.env: %v", err)
	}
	perm := info.Mode().Perm()
	if perm != 0600 {
		t.Errorf("set 后 endpoint.env 权限为 %o, 期望 0600 (NFR-9)", perm)
	}
}

// TestST4_EndpointApplyAgentConfigPermission 验证 endpoint apply 后 agent 配置文件权限为 0600。
//
// 覆盖案例：endpoint apply — agent 配置文件权限为 0600（claude/opencode/kimi/dstui）
// 可追溯性: NFR-9
func TestST4_EndpointApplyAgentConfigPermission(t *testing.T) {
	binaryPath := filepath.Join(t.TempDir(), "agent-forge")
	buildCmd := exec.Command("/tmp/go/bin/go", "build", "-o", binaryPath, ".")
	buildCmd.Dir = ".."
	output, err := buildCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("构建二进制失败: %v\nOutput: %s", err, output)
	}

	configDir := t.TempDir()

	// 创建 deepseek 端点（可服务所有 4 个 agent）
	addCmd := exec.Command(binaryPath, "endpoint", "add", "my-ep",
		"--provider", "deepseek",
		"--url", "https://api.deepseek.com",
		"--key", "sk-ds-key",
		"--model", "deepseek-chat",
		"-c", configDir,
	)
	if addOut, addErr := addCmd.CombinedOutput(); addErr != nil {
		t.Fatalf("创建端点失败: %v\n%s", addErr, addOut)
	}

	// apply 同步
	applyCmd := exec.Command(binaryPath, "endpoint", "apply", "my-ep", "-c", configDir)
	if applyOut, applyErr := applyCmd.CombinedOutput(); applyErr != nil {
		t.Fatalf("apply 失败: %v\n%s", applyErr, applyOut)
	}

	// 验证各 agent 配置文件权限为 0600
	agentFiles := []string{
		".claude/.env",
		".opencode/.env",
		".kimi/config.toml",
		".deepseek/.env",
	}
	for _, relPath := range agentFiles {
		fullPath := filepath.Join(configDir, relPath)
		info, err := os.Stat(fullPath)
		if err != nil {
			t.Errorf("无法 stat %s: %v", relPath, err)
			continue
		}
		perm := info.Mode().Perm()
		if perm != 0600 {
			t.Errorf("%s 权限为 %o, 期望 0600 (NFR-9)", relPath, perm)
		}
	}
}

// TestST4_EndpointsDirPermission 验证 endpoints/ 目录权限不为 0777。
//
// 覆盖案例：目录权限 — endpoints/ 目录权限不为 0777（合理限制）
// 可追溯性: NFR-9
func TestST4_EndpointsDirPermission(t *testing.T) {
	binaryPath := filepath.Join(t.TempDir(), "agent-forge")
	buildCmd := exec.Command("/tmp/go/bin/go", "build", "-o", binaryPath, ".")
	buildCmd.Dir = ".."
	output, err := buildCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("构建二进制失败: %v\nOutput: %s", err, output)
	}

	configDir := t.TempDir()

	// 创建端点（这会创建 endpoints/ 目录）
	addCmd := exec.Command(binaryPath, "endpoint", "add", "my-ep",
		"--provider", "openai",
		"--url", "https://api.openai.com",
		"--key", "sk-test-key",
		"--model", "gpt-4",
		"-c", configDir,
	)
	if addOut, addErr := addCmd.CombinedOutput(); addErr != nil {
		t.Fatalf("创建端点失败: %v\n%s", addErr, addOut)
	}

	// 验证 endpoints/ 目录权限
	endpointsDir := filepath.Join(configDir, "endpoints")
	info, err := os.Stat(endpointsDir)
	if err != nil {
		t.Fatalf("无法 stat endpoints/: %v", err)
	}
	perm := info.Mode().Perm()
	if perm == 0777 {
		t.Error("endpoints/ 目录权限不应为 0777")
	}
	// 目录应为 0755（或更严格）
	if perm > 0755 {
		t.Errorf("endpoints/ 目录权限为 %o, 期望 <= 0755", perm)
	}
}
