//go:build e2e

package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// buildBinary 编译 CLI 二进制并返回路径。
func buildBinary(t *testing.T) string {
	t.Helper()
	binaryPath := filepath.Join(t.TempDir(), "agent-forge")
	cmd := exec.Command("/tmp/go/bin/go", "build", "-o", binaryPath, ".")
	cmd.Dir = ".." // root of the project (since we're in cmd/)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("构建二进制失败: %v\nOutput: %s", err, output)
	}
	return binaryPath
}

// runEndpoint 执行 endpoint 命令并返回输出。
func runEndpoint(t *testing.T, binaryPath, configDir string, args ...string) (string, error) {
	t.Helper()
	cmdArgs := append([]string{"endpoint"}, args...)
	cmdArgs = append(cmdArgs, "-c", configDir)
	cmd := exec.Command(binaryPath, cmdArgs...)
	output, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(output)), err
}

// TestE2E_GH12_EndpointAddWithAllParams 覆盖 GH-12 Scenario "带全部参数新增 LLM 端点"。
//
// Given 无（端点目录不存在）
// When 开发者执行 endpoint add my-ep --provider openai --url https://api.openai.com
//
//	--key sk-test-key-value --model gpt-4 --model-opus gpt-4-32k
//	--model-sonnet gpt-4-turbo --model-haiku gpt-3.5-turbo --model-subagent gpt-4-mini
//
// Then 端点 my-ep 创建成功
// And endpoint list 输出表中包含 my-ep
// And endpoint show my-ep 显示 API key 为 sk-test-***alue 掩码格式
func TestE2E_GH12_EndpointAddWithAllParams(t *testing.T) {
	binary := buildBinary(t)
	configDir := t.TempDir()

	// When 带全部 8 个参数新增端点
	output, err := runEndpoint(t, binary, configDir,
		"add", "my-ep",
		"--provider", "openai",
		"--url", "https://api.openai.com",
		"--key", "sk-test-key-value",
		"--model", "gpt-4",
		"--model-opus", "gpt-4-32k",
		"--model-sonnet", "gpt-4-turbo",
		"--model-haiku", "gpt-3.5-turbo",
		"--model-subagent", "gpt-4-mini",
	)
	if err != nil {
		t.Fatalf("endpoint add 失败: %v\nOutput: %s", err, output)
	}

	// Then 端点创建成功
	if !strings.Contains(output, "创建成功") {
		t.Errorf("应包含'创建成功'，got: %s", output)
	}

	// 验证 endpoint.env 文件存在
	envPath := filepath.Join(configDir, "endpoints", "my-ep", "endpoint.env")
	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		t.Fatal("endpoint.env 文件未被创建")
	}

	// And endpoint list 包含 my-ep
	listOutput, err := runEndpoint(t, binary, configDir, "list")
	if err != nil {
		t.Fatalf("endpoint list 失败: %v", err)
	}
	if !strings.Contains(listOutput, "my-ep") {
		t.Errorf("endpoint list 应包含 my-ep，got:\n%s", listOutput)
	}
	if !strings.Contains(listOutput, "openai") {
		t.Errorf("endpoint list 应包含 openai，got:\n%s", listOutput)
	}
	if !strings.Contains(listOutput, "gpt-4") {
		t.Errorf("endpoint list 应包含 gpt-4，got:\n%s", listOutput)
	}

	// And endpoint show 显示掩码 key
	showOutput, err := runEndpoint(t, binary, configDir, "show", "my-ep")
	if err != nil {
		t.Fatalf("endpoint show 失败: %v", err)
	}
	if !strings.Contains(showOutput, "sk-test-***alue") {
		t.Errorf("API key 应显示为 sk-test-***alue 掩码格式，got:\n%s", showOutput)
	}
	if strings.Contains(showOutput, "sk-test-key-value") {
		t.Error("API key 不应以明文完整显示")
	}
}
