//go:build e2e

package cmd

import (
	"os/exec"
	"strings"
	"testing"
)

// TestE2E_GH22_DepsInspect 覆盖 Scenario "查询容器内依赖安装状态" (E2E)。
//
// 验证场景：
//   - 开发者在宿主机执行 deps 命令
//   - 系统生成检测脚本并通过 docker run --rm 在临时容器中执行
//   - 输出按 agent/runtime/tool 分类显示安装状态和版本号
//   - 检测完成后临时容器自动销毁
//
// 可追溯性: REQ-33 · Scenario: "查询容器内依赖安装状态"
func TestE2E_GH22_DepsInspect(t *testing.T) {
	binaryPath := buildBinary(t)

	// 使用本地已存在的镜像（而非 agent-forge:latest）
	imageRef := "docker.1ms.run/centos:7"

	// 检查镜像是否存在，如果不存在则跳过
	checkCmd := exec.Command("docker", "image", "inspect", imageRef)
	if err := checkCmd.Run(); err != nil {
		t.Skipf("镜像 %s 不存在，跳过 E2E 测试", imageRef)
	}

	// 执行 deps 命令
	cmd := exec.Command(binaryPath, "deps", "-i", imageRef)
	output, err := cmd.CombinedOutput()
	outputStr := string(output)
	if err != nil {
		t.Fatalf("deps 命令执行失败: %v\nOutput: %s", err, outputStr)
	}

	// 验证输出包含标题
	if !strings.Contains(outputStr, "依赖检测结果") {
		t.Errorf("deps 输出应包含 '依赖检测结果' 标题, 实际:\n%s", outputStr)
	}

	// 验证输出包含分类
	categories := []string{"[agent]", "[runtime]", "[tool]"}
	for _, cat := range categories {
		if !strings.Contains(outputStr, cat) {
			t.Errorf("deps 输出应包含分类 %q, 实际:\n%s", cat, outputStr)
		}
	}

	// 验证输出包含组件名
	components := []string{"claude", "opencode", "golang", "node", "docker"}
	for _, comp := range components {
		if !strings.Contains(outputStr, comp) {
			t.Errorf("deps 输出应包含组件 %q, 实际:\n%s", comp, outputStr)
		}
	}

	// 验证输出包含统计信息
	if !strings.Contains(outputStr, "总计:") {
		t.Errorf("deps 输出应包含 '总计:' 统计, 实际:\n%s", outputStr)
	}

	// 验证输出包含安装状态
	if !strings.Contains(outputStr, "OK") && !strings.Contains(outputStr, "XX") {
		t.Errorf("deps 输出应包含 'OK' 或 'XX' 状态标记, 实际:\n%s", outputStr)
	}

	// 验证容器已自动销毁（--rm）
	psCmd := exec.Command("docker", "ps", "-a", "--format", "{{.Image}}")
	psOutput, _ := psCmd.CombinedOutput()
	if strings.Contains(string(psOutput), imageRef) {
		t.Errorf("存在残留容器: docker ps -a 中仍有包含 %s 的条目", imageRef)
	}

	t.Logf("deps 命令输出:\n%s", outputStr)
}

// TestE2E_GH22_DepsDefaultImage 验证 deps 命令对默认镜像的处理。
//
// 测试目的：确保 deps 在默认镜像不存在时给出清晰错误而非崩溃。
func TestE2E_GH22_DepsDefaultImage(t *testing.T) {
	binaryPath := buildBinary(t)

	// 不指定 -i 参数，使用默认 agent-forge:latest
	// 该镜像在大多数环境中不存在，应输出清晰错误
	cmd := exec.Command(binaryPath, "deps")
	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	if err != nil {
		// 预期会失败（镜像不存在），验证错误信息清晰
		if !strings.Contains(outputStr, "依赖检测失败") {
			t.Errorf("默认镜像不存在时应输出清晰的错误信息, 实际:\n%s", outputStr)
		}
		t.Logf("默认镜像不存在时输出（预期）:\n%s", outputStr)
	} else {
		// 如果成功了（镜像存在），验证输出格式
		if !strings.Contains(outputStr, "依赖检测结果") {
			t.Errorf("deps 输出格式不正确, 实际:\n%s", outputStr)
		}
		t.Logf("deps 默认镜像输出:\n%s", outputStr)
	}
}
