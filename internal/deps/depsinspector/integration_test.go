// Package depsinspector 提供 DepsInspector 的集成测试（IT-8）。
//
// 本文件覆盖 IT-8 的所有案例，验证检测脚本生成、临时容器执行、
// 输出解析和容器自动销毁。
package depsinspector

import (
	"os/exec"
	"strings"
	"testing"
)

// --- IT-8: DepsInspector 容器内依赖检测集成测试 ---

// TestIT8_ScriptGenerate_ValidBash 验证生成的检测脚本是合法的 bash 脚本。
//
// 覆盖案例：检测脚本生成 — 输出合法的 bash 脚本，按 agent/runtime/tool 分类。
func TestIT8_ScriptGenerate_ValidBash(t *testing.T) {
	script := GenerateScript()

	// 脚本应为非空字符串
	if script == "" {
		t.Fatal("GenerateScript() 不应返回空字符串")
	}

	// 脚本应以 shebang 开头
	if !strings.HasPrefix(script, "#!/bin/bash") {
		t.Error("检测脚本应以 #!/bin/bash 开头")
	}

	// 脚本应包含 agent/runtime/tool 分类
	categories := []struct {
		name  string
		items []string
	}{
		{"agent", []string{"claude", "opencode", "kimi", "deepseek-tui"}},
		{"runtime", []string{"golang", "node"}},
		{"tool", []string{"docker", "rtk"}},
	}
	for _, cat := range categories {
		for _, item := range cat.items {
			if !strings.Contains(script, item) {
				t.Errorf("检测脚本应包含 %s 类别中的 %s", cat.name, item)
			}
		}
	}

	// 脚本应使用 pipe 分隔格式（component|type|status|version）
	if !strings.Contains(script, "|") {
		t.Error("检测脚本应包含 pipe 分隔的输出格式")
	}

	// 验证 bash 语法
	cmd := exec.Command("bash", "-n")
	cmd.Stdin = strings.NewReader(script)
	if err := cmd.Run(); err != nil {
		t.Fatalf("检测脚本 bash 语法错误: %v", err)
	}
}

// TestIT8_ScriptGenerate_AllCategories 验证检测脚本覆盖所有分类。
//
// 覆盖案例：检测脚本生成 — 按 agent/skill/tool/runtime 分类。
func TestIT8_ScriptGenerate_AllCategories(t *testing.T) {
	script := GenerateScript()

	// 验证包含所有依赖项的输出标记
	expectedItems := []string{
		"claude", "opencode", "kimi", "deepseek-tui",
		"golang", "node",
		"docker", "rtk",
	}
	for _, item := range expectedItems {
		if !strings.Contains(script, item) {
			t.Errorf("检测脚本应包含对 %s 的检测逻辑", item)
		}
	}

	// 验证每个检测项都有 command -v 检查
	if !strings.Contains(script, "command -v") {
		t.Error("检测脚本应使用 command -v 检查可执行文件是否存在")
	}
}

// TestIT8_ParseOutput_AllInstalled 解析所有已安装的输出。
//
// 覆盖案例：输出解析 — 检测结果分类输出。
func TestIT8_ParseOutput_AllInstalled(t *testing.T) {
	output := `claude|agent|installed|Claude Code 1.0.0
opencode|agent|installed|opencode 0.55.0
kimi|agent|installed|kimi-cli 2.0.0
deepseek-tui|agent|installed|0.8.27
golang|runtime|installed|go version go1.22.3 linux/amd64
node|runtime|installed|v16.20.0
docker|tool|installed|Docker version 24.0.0
rtk|tool|installed|rtk 1.0.0
`

	result, err := ParseOutput([]byte(output))
	if err != nil {
		t.Fatalf("ParseOutput() 返回错误: %v", err)
	}

	if result == nil {
		t.Fatal("ParseOutput() 返回 nil")
	}

	if len(result.Items) != 8 {
		t.Errorf("应解析出 8 个依赖项，实际 %d", len(result.Items))
	}

	// 验证所有依赖项状态为 installed
	for _, item := range result.Items {
		if item.Status != "installed" {
			t.Errorf("%s 状态应为 installed, 实际 %s", item.Name, item.Status)
		}
		if item.Version == "" {
			t.Errorf("%s 应有版本信息", item.Name)
		}
	}
}

// TestIT8_ParseOutput_AllMissing 解析所有缺失的输出。
//
// 覆盖案例：输出解析 — 缺失组件处理。
func TestIT8_ParseOutput_AllMissing(t *testing.T) {
	output := `claude|agent|missing|
opencode|agent|missing|
`
	result, err := ParseOutput([]byte(output))
	if err != nil {
		t.Fatalf("ParseOutput() 返回错误: %v", err)
	}

	if len(result.Items) != 2 {
		t.Errorf("应解析出 2 个依赖项，实际 %d", len(result.Items))
	}

	for _, item := range result.Items {
		if item.Status != "missing" {
			t.Errorf("%s 状态应为 missing, 实际 %s", item.Name, item.Status)
		}
	}
}

// TestIT8_ParseOutput_MixedResults 解析混合结果。
//
// 覆盖案例：输出解析 — 混合 installed/missing。
func TestIT8_ParseOutput_MixedResults(t *testing.T) {
	output := `claude|agent|installed|Claude Code 1.0.0
node|runtime|missing|
`

	result, err := ParseOutput([]byte(output))
	if err != nil {
		t.Fatalf("ParseOutput() 返回错误: %v", err)
	}

	if len(result.Items) != 2 {
		t.Errorf("应解析出 2 个依赖项，实际 %d", len(result.Items))
	}

	// claude installed
	if result.Items[0].Name != "claude" || result.Items[0].Status != "installed" {
		t.Errorf("claude 应为 installed, 实际 %+v", result.Items[0])
	}
	// node missing
	if result.Items[1].Name != "node" || result.Items[1].Status != "missing" {
		t.Errorf("node 应为 missing, 实际 %+v", result.Items[1])
	}
}

// TestIT8_ParseOutput_EmptyInput 解析空输入。
//
// 覆盖案例：输出解析 — 空输入处理。
func TestIT8_ParseOutput_EmptyInput(t *testing.T) {
	result, err := ParseOutput([]byte(""))
	if err != nil {
		t.Fatalf("ParseOutput() 返回错误: %v", err)
	}

	if result == nil {
		t.Fatal("ParseOutput() 返回 nil")
	}

	if len(result.Items) != 0 {
		t.Errorf("空输入应解析出 0 个依赖项，实际 %d", len(result.Items))
	}
}

// TestIT8_ParseOutput_WithComments 解析包含注释的输出。
//
// 覆盖案例：输出解析 — 跳过注释。
func TestIT8_ParseOutput_WithComments(t *testing.T) {
	output := `# agent-forge deps inspector
claude|agent|installed|1.0.0
# this is a comment
golang|runtime|missing|
`

	result, err := ParseOutput([]byte(output))
	if err != nil {
		t.Fatalf("ParseOutput() 返回错误: %v", err)
	}

	if len(result.Items) != 2 {
		t.Errorf("应解析出 2 个依赖项（跳过注释），实际 %d", len(result.Items))
	}
}

// TestIT8_FormatResult_Formatting 验证 FormatResult 输出格式。
//
// 覆盖案例：输出格式化 — 表格格式正确。
func TestIT8_FormatResult_Formatting(t *testing.T) {
	result := &InspectionResult{
		Items: []DependencyStatus{
			{Name: "claude", Type: "agent", Status: "installed", Version: "1.0.0"},
			{Name: "golang", Type: "runtime", Status: "missing", Version: ""},
		},
	}

	formatted := FormatResult(result)

	expectedParts := []string{
		"依赖检测结果",
		"[agent]",
		"[runtime]",
		"claude",
		"golang",
		"总计:",
	}
	for _, part := range expectedParts {
		if !strings.Contains(formatted, part) {
			t.Errorf("格式化输出应包含 %q, 实际:\n%s", part, formatted)
		}
	}
}

// TestIT8_FormatResult_NilResult 验证 nil 结果格式化。
func TestIT8_FormatResult_NilResult(t *testing.T) {
	formatted := FormatResult(nil)
	if formatted != "未检测到任何组件\n" {
		t.Errorf("nil 结果应输出 '未检测到任何组件', 实际: %q", formatted)
	}
}

// TestIT8_DockerRun_RunWithExistingImage 在真实 Docker 容器中执行检测。
//
// 覆盖案例：
//   - 临时容器执行: 通过 docker run --rm 成功执行检测脚本
//   - 容器自动销毁: 检测完成后容器不在 docker ps -a 中
//
// 使用已存在的测试镜像执行检测，验证完整流程。
func TestIT8_DockerRun_RunWithExistingImage(t *testing.T) {
	// 使用本地已存在的镜像（必须是 docker pull 过的）
	imageRef := "docker.1ms.run/centos:7"

	// 跳过：如果镜像不存在
	checkCmd := exec.Command("docker", "image", "inspect", imageRef)
	if err := checkCmd.Run(); err != nil {
		t.Skipf("镜像 %s 不存在，跳过集成测试", imageRef)
	}

	result, err := RunDetection(imageRef, "")
	if err != nil {
		t.Fatalf("RunDetection() 返回错误: %v", err)
	}

	if result == nil {
		t.Fatal("RunDetection() 返回 nil")
	}

	// 验证结果包含所有依赖项
	expectedNames := map[string]bool{
		"claude": false, "opencode": false, "kimi": false, "deepseek-tui": false,
		"golang": false, "node": false,
		"docker": false, "rtk": false,
	}
	for _, item := range result.Items {
		if _, ok := expectedNames[item.Name]; ok {
			expectedNames[item.Name] = true
		}
		// 验证每个项都有有效的状态
		if item.Status != "installed" && item.Status != "missing" {
			t.Errorf("%s 状态应为 installed 或 missing, 实际 %s", item.Name, item.Status)
		}
		// 验证分类正确
		if item.Type != "agent" && item.Type != "runtime" && item.Type != "tool" {
			t.Errorf("%s 分类无效: %s", item.Name, item.Type)
		}
	}
	for name, found := range expectedNames {
		if !found {
			t.Errorf("结果中缺少 %s", name)
		}
	}

	// 验证容器已自动销毁（--rm）
	psCmd := exec.Command("docker", "ps", "-a", "--format", "{{.Image}}")
	psOutput, _ := psCmd.CombinedOutput()
	if strings.Contains(string(psOutput), imageRef) {
		t.Errorf("存在残留容器: docker ps -a 中仍有包含 %s 的条目", imageRef)
	}
}

func TestGenerateScriptWindows_ContainsGetCommand(t *testing.T) {
	script := GenerateScriptWindows()
	if script == "" {
		t.Fatal("GenerateScriptWindows() returned empty string")
	}
	if !strings.Contains(script, "Get-Command") {
		t.Error("Windows script should use Get-Command")
	}
	if !strings.Contains(script, "Write-Output") {
		t.Error("Windows script should use Write-Output")
	}
	// Should not contain bash syntax
	if strings.Contains(script, "#!/bin/bash") {
		t.Error("Windows script should not contain bash shebang")
	}
	if strings.Contains(script, "command -v") {
		t.Error("Windows script should not use command -v")
	}
}
