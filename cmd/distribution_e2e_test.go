//go:build e2e

package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestE2E_GH23_ExportImport 覆盖 Scenario "导出和导入镜像实现离线分发" (E2E)。
//
// 验证场景：
//   - 导出镜像为 tar 文件
//   - 导入 tar 文件后镜像在 docker images 中可见
//
// 可追溯性: REQ-34 · REQ-35 · Scenario: "导出和导入镜像实现离线分发"
func TestE2E_GH23_ExportImport(t *testing.T) {
	binaryPath := buildBinary(t)

	// 使用本地镜像进行测试
	imageRef := "docker.1ms.run/centos:7"

	// 检查镜像是否存在
	checkCmd := exec.Command("docker", "image", "inspect", imageRef)
	if err := checkCmd.Run(); err != nil {
		t.Skipf("镜像 %s 不存在，跳过 E2E 测试", imageRef)
	}

	exportPath := filepath.Join(t.TempDir(), "agent-forge.tar")

	// --- 测试 export ---
	t.Run("export", func(t *testing.T) {
		cmd := exec.Command(binaryPath, "export", exportPath, "-i", imageRef)
		output, err := cmd.CombinedOutput()
		outputStr := string(output)
		if err != nil {
			t.Fatalf("export 命令执行失败: %v\nOutput: %s", err, outputStr)
		}

		// 验证输出消息包含成功提示
		if !strings.Contains(outputStr, "已成功导出") {
			t.Errorf("export 输出应包含成功提示, 实际: %s", outputStr)
		}

		// 验证文件存在且非空
		info, err := os.Stat(exportPath)
		if err != nil {
			t.Fatalf("导出文件不存在: %v", err)
		}
		if info.Size() == 0 {
			t.Error("导出文件不应为空")
		}

		t.Logf("导出成功，文件大小: %d bytes", info.Size())
	})

	// --- 测试 import ---
	t.Run("import", func(t *testing.T) {
		cmd := exec.Command(binaryPath, "import", exportPath)
		output, err := cmd.CombinedOutput()
		outputStr := string(output)
		if err != nil {
			t.Fatalf("import 命令执行失败: %v\nOutput: %s", err, outputStr)
		}

		// 验证输出消息包含成功提示
		if !strings.Contains(outputStr, "已成功") {
			t.Errorf("import 输出应包含成功提示, 实际: %s", outputStr)
		}

		// 验证镜像在 docker images 中可见
		inspectCmd := exec.Command("docker", "image", "inspect", imageRef)
		if err := inspectCmd.Run(); err != nil {
			t.Errorf("导入后镜像 %s 应在 docker images 中可见: %v", imageRef, err)
		}

		t.Logf("导入成功，镜像 %s 可见", imageRef)
	})
}

// TestE2E_GH23_ExportNonexistentImage 验证导出不存在的镜像返回错误。
func TestE2E_GH23_ExportNonexistentImage(t *testing.T) {
	binaryPath := buildBinary(t)
	exportPath := filepath.Join(t.TempDir(), "nonexistent.tar")

	cmd := exec.Command(binaryPath, "export", exportPath, "-i", "nonexistent-image-12345")
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("导出不存在的镜像应返回非零退出码")
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "不存在") {
		t.Errorf("输出应包含 '不存在' 错误, 实际: %s", outputStr)
	}
	t.Logf("预期的错误输出: %s", outputStr)
}

// TestE2E_GH23_ImportNonexistentFile 验证导入不存在的文件返回错误。
func TestE2E_GH23_ImportNonexistentFile(t *testing.T) {
	binaryPath := buildBinary(t)

	cmd := exec.Command(binaryPath, "import", "/tmp/nonexistent-file-12345.tar")
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("导入不存在的文件应返回非零退出码")
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "不存在") {
		t.Errorf("输出应包含 '不存在' 错误, 实际: %s", outputStr)
	}
	t.Logf("预期的错误输出: %s", outputStr)
}
