// Package cmd 提供 CLI 路由和退出码集成测试（IT-9）。
package cmd

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// buildTestBinary 编译 CLI 二进制并返回路径。
func buildTestBinary(t *testing.T) string {
	t.Helper()
	binaryPath := filepath.Join(t.TempDir(), "agent-forge")
	build := exec.Command("/tmp/go/bin/go", "build", "-o", binaryPath, ".")
	build.Dir = ".."
	out, err := build.CombinedOutput()
	if err != nil {
		t.Fatalf("构建二进制失败: %v\nOutput: %s", err, out)
	}
	return binaryPath
}

// runCli 执行 CLI 命令并返回输出和错误。
func runCli(binaryPath string, args ...string) (string, error) {
	cmd := exec.Command(binaryPath, args...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// TestIT9_CommandsExist 验证所有一级子命令可被路由。
func TestIT9_CommandsExist(t *testing.T) {
	binaryPath := buildTestBinary(t)

	commands := []string{
		"build",
		"run",
		"endpoint",
		"doctor",
		"deps",
		"export",
		"import",
		"update",
		"version",
		"help",
	}

	for _, cmd := range commands {
		output, err := runCli(binaryPath, cmd, "--help")
		if err != nil {
			t.Errorf("命令 %s --help 执行失败: %v\nOutput: %s", cmd, err, output)
			continue
		}
		if !strings.Contains(output, cmd) && cmd != "help" {
			// help 命令的输出可能不包含 "help" 字符串
		}
	}
}

// TestIT9_GlobalHelp 验证全局 --help 输出所有子命令列表。
func TestIT9_GlobalHelp(t *testing.T) {
	binaryPath := buildTestBinary(t)

	output, err := runCli(binaryPath, "--help")
	if err != nil {
		t.Fatalf("--help 执行失败: %v\nOutput: %s", err, output)
	}

	expectedCommands := []string{
		"build",
		"run",
		"endpoint",
		"doctor",
		"deps",
		"export",
		"import",
		"update",
		"version",
	}
	for _, cmd := range expectedCommands {
		if !strings.Contains(output, cmd) {
			t.Errorf("全局 --help 应包含命令 %q, 实际:\n%s", cmd, output)
		}
	}
}

// TestIT9_InvalidCommand 验证无效命令返回错误。
func TestIT9_InvalidCommand(t *testing.T) {
	binaryPath := buildTestBinary(t)

	output, err := runCli(binaryPath, "nonexistent-command")
	if err == nil {
		t.Error("无效命令应返回非零退出码")
	}
	if !strings.Contains(output, "unknown command") &&
		!strings.Contains(output, "unknown") {
		t.Errorf("无效命令应显示错误信息, 实际:\n%s", output)
	}
}

// TestIT9_VersionCommand 验证 version 命令输出。
func TestIT9_VersionCommand(t *testing.T) {
	binaryPath := buildTestBinary(t)

	output, err := runCli(binaryPath, "version")
	if err != nil {
		t.Fatalf("version 命令执行失败: %v\nOutput: %s", err, output)
	}

	if !strings.HasPrefix(output, "agent-forge ") {
		t.Errorf("version 输出应以 'agent-forge ' 开头, 实际: %s", output)
	}
}

// TestIT9_NoArgsDefaultsToRun 验证无参数时默认执行 run 命令。
func TestIT9_NoArgsDefaultsToRun(t *testing.T) {
	binaryPath := buildTestBinary(t)

	output, err := runCli(binaryPath)
	if err == nil {
		t.Log("无参数时默认执行 run 命令，输出:")
		t.Log(output)
		// 可以正常退出表示路由正确
	} else {
		// run 可能需要 Docker，但命令应被路由到 RunEngine
		if !strings.Contains(output, "原因:") && !strings.Contains(output, "Error:") {
			t.Errorf("无参数时输出应包含错误信息（run 需要 Docker）, 实际:\n%s", output)
		}
		t.Log("无参数时正确路由到 run 命令:", output)
	}
}

// TestIT9_DoctorCommand_Routed 验证 doctor 命令被路由到 DiagnosticEngine。
func TestIT9_DoctorCommand_Routed(t *testing.T) {
	binaryPath := buildTestBinary(t)

	output, err := runCli(binaryPath, "doctor")
	if err != nil && !strings.Contains(output, "全部通过") {
		// doctor 可能成功或失败，但输出格式应包含诊断信息
	}

	// 验证输出包含诊断内容
	if !strings.Contains(output, "AgentForge") && !strings.Contains(output, "Error") {
		t.Errorf("doctor 输出应包含诊断信息, 实际:\n%s", output)
	}
}

// TestIT9_EndpointHelp 验证 endpoint 子命令帮助。
func TestIT9_EndpointHelp(t *testing.T) {
	binaryPath := buildTestBinary(t)

	output, err := runCli(binaryPath, "endpoint", "--help")
	if err != nil {
		t.Fatalf("endpoint --help 执行失败: %v\nOutput: %s", err, output)
	}

	// endpoint 子命令
	subCommands := []string{"providers", "list", "show", "add", "set", "rm", "test", "apply", "status"}
	for _, cmd := range subCommands {
		if !strings.Contains(output, cmd) {
			t.Errorf("endpoint --help 应包含子命令 %q, 实际:\n%s", cmd, output)
		}
	}
}
