//go:build e2e

package cmd

import (
	"os/exec"
	"strings"
	"testing"
)

// TestE2E_GH21_EnvironmentDiagnosis 覆盖 Scenario "环境诊断" (E2E)。
//
// 验证场景：
//   - 正常环境：Docker Engine 已安装并运行，doctor 输出三层诊断结果
//   - 异常环境：核心依赖缺失，doctor 输出诊断问题和建议
//
// 测试验证输出格式而非诊断结果，因为实际环境可能不同。
// 可追溯性: REQ-31 · REQ-32 · NFR-17 · NFR-18 · NFR-19
func TestE2E_GH21_EnvironmentDiagnosis(t *testing.T) {
	binaryPath := buildBinary(t)

	// 执行 doctor 命令
	cmd := exec.Command(binaryPath, "doctor")
	output, err := cmd.CombinedOutput()
	outputStr := string(output)
	if err != nil {
		t.Logf("doctor 命令输出（含错误）:\n%s", outputStr)
	}

	// 验证输出包含层名
	expectedIndicators := []string{
		"AgentForge",
		"环境诊断",
		"核心依赖",
		"运行时",
		"可选工具",
	}
	for _, indicator := range expectedIndicators {
		if !strings.Contains(outputStr, indicator) {
			t.Errorf("doctor 输出应包含 %q, 实际输出:\n%s", indicator, outputStr)
		}
	}

	// 验证输出包含通过/未通过状态
	if !strings.Contains(outputStr, "通过") && !strings.Contains(outputStr, "未通过") {
		t.Errorf("doctor 输出应包含 '通过' 或 '未通过' 状态, 实际输出:\n%s", outputStr)
	}

	// 验证输出包含分隔线
	if !strings.Contains(outputStr, "====") {
		t.Errorf("doctor 输出应包含分隔线, 实际输出:\n%s", outputStr)
	}

	// 验证输出包含结果总结
	if !strings.Contains(outputStr, "结果:") {
		t.Errorf("doctor 输出应包含 '结果:' 总结, 实际输出:\n%s", outputStr)
	}

	t.Logf("doctor 命令输出:\n%s", outputStr)
}

// TestE2E_GH21_DiagnosisWithConfig 验证带有 -c 配置目录参数的 doctor 命令。
//
// 测试目的：确保 doctor 命令能正常解析 -c 参数。
func TestE2E_GH21_DiagnosisWithConfig(t *testing.T) {
	binaryPath := buildBinary(t)
	configDir := t.TempDir()

	cmd := exec.Command(binaryPath, "doctor", "-c", configDir)
	output, err := cmd.CombinedOutput()
	outputStr := string(output)
	if err != nil {
		t.Logf("doctor -c 命令输出（含错误）:\n%s", outputStr)
	}

	if !strings.Contains(outputStr, "AgentForge") {
		t.Errorf("doctor -c 输出应包含诊断信息, 实际输出:\n%s", outputStr)
	}

	t.Logf("doctor -c 命令输出:\n%s", outputStr)
}
