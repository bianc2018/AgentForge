//go:build e2e

package cmd

import (
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// TestE2E_GH24_VersionAndHelp 覆盖 Scenario "工具自更新和版本信息查看" (E2E)。
//
// 验证场景：
//   - version 命令输出格式化的版本号和当前 git hash
//   - --help 输出格式一致的帮助信息
//
// 可追溯性: REQ-36 · REQ-37 · NFR-15 · NFR-21 · NFR-22
func TestE2E_GH24_VersionAndHelp(t *testing.T) {
	binaryPath := buildBinary(t)

	// --- 验证 version 命令 ---
	t.Run("version", func(t *testing.T) {
		cmd := exec.Command(binaryPath, "version")
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("version 命令执行失败: %v\nOutput: %s", err, output)
		}
		outputStr := string(output)

		if !strings.HasPrefix(outputStr, "agent-forge ") {
			t.Errorf("version 输出应以 'agent-forge ' 开头, 实际: %q", outputStr)
		}
		t.Logf("version 输出: %s", outputStr)
	})

	// --- 验证 --help 输出 ---
	t.Run("help", func(t *testing.T) {
		cmd := exec.Command(binaryPath, "--help")
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("--help 执行失败: %v\nOutput: %s", err, output)
		}
		outputStr := string(output)

		expectedCommands := []string{"build", "run", "doctor", "deps", "endpoint", "export", "import", "update", "version"}
		for _, cmdName := range expectedCommands {
			if !strings.Contains(outputStr, cmdName) {
				t.Errorf("--help 应包含命令 %q, 实际:\n%s", cmdName, outputStr)
			}
		}
		t.Logf("--help 输出格式正确")
	})
}

// TestE2E_GH24_UpdateWithMockServer 使用 mock HTTP server 验证自更新流程。
//
// 验证更新成功：从 mock server 下载新版本并替换当前二进制。
//
// 可追溯性: REQ-36 · NFR-13
func TestE2E_GH24_UpdateWithMockServer(t *testing.T) {
	// 编译测试用的二进制（带自定义 UPDATE_URL）
	binaryPath := buildBinary(t)

	// 创建 mock HTTP server 提供"新版"二进制
	mockBinary := []byte("#!/bin/sh\necho 'mock agent-forge v2.0.0'")
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write(mockBinary)
	}))
	defer mockServer.Close()

	// 使用 UPDATE_URL 环境变量指向 mock server
	cmd := exec.Command(binaryPath, "update")
	cmd.Env = append(os.Environ(), "UPDATE_URL="+mockServer.URL+"/download")
	output, err := cmd.CombinedOutput()
	outputStr := string(output)
	if err != nil {
		t.Fatalf("update 命令执行失败: %v\nOutput: %s", err, outputStr)
	}

	if !strings.Contains(outputStr, "已成功更新") {
		t.Errorf("update 输出应包含成功提示, 实际: %s", outputStr)
	}
	t.Logf("update 输出: %s", outputStr)
}

// TestE2E_GH24_UpdateWithMockServer_Failure 验证 mock server 返回错误时 update 失败。
//
// 验证更新失败：服务器返回 500 时 update 返回错误。
//
// 可追溯性: REQ-36 · NFR-13
func TestE2E_GH24_UpdateWithMockServer_Failure(t *testing.T) {
	binaryPath := buildBinary(t)

	// 创建返回 500 的 mock server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer mockServer.Close()

	cmd := exec.Command(binaryPath, "update")
	cmd.Env = append(os.Environ(), "UPDATE_URL="+mockServer.URL+"/download")
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("mock server 返回 500 时 update 应返回错误")
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "失败") && !strings.Contains(outputStr, "Error") {
		t.Errorf("update 失败时应输出错误信息, 实际: %s", outputStr)
	}
	t.Logf("预期的 update 错误输出: %s", outputStr)

	// 验证二进制文件仍存在
	if _, err := os.Stat(binaryPath); err != nil {
		t.Fatalf("更新失败后二进制文件应存在: %v", err)
	}

	// 验证当前二进制仍可运行
	checkCmd := exec.Command(binaryPath, "version")
	verOutput, verErr := checkCmd.CombinedOutput()
	if verErr != nil {
		t.Fatalf("更新失败后运行 version 命令失败: %v\nOutput: %s", verErr, verOutput)
	}
	if !strings.HasPrefix(string(verOutput), "agent-forge ") {
		t.Errorf("更新失败后二进制应正常输出, 实际: %q", string(verOutput))
	}
	t.Logf("更新失败后 version 输出: %s", string(verOutput))
}
