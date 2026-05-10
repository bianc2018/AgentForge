// Package wrapperloader 提供 WrapperLoader 的单元测试。
//
// 本文件覆盖 UT-7 (WrapperLoader.Generate)，验证生成的 bash 脚本
// 包含所有 4 个 agent 的 wrapper 函数定义、语法正确性及函数名无冲突。
package wrapperloader

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

// --- UT-7: WrapperLoader.Generate() ---

// TestGenerate_ContainsAllAgents 验证生成的脚本包含所有 4 个 agent 的函数定义。
//
// 覆盖案例：正常路径 — 生成的脚本包含 claude/opencode/kimi/deepseek-tui 的函数定义
func TestGenerate_ContainsAllAgents(t *testing.T) {
	wl := New()
	script := wl.Generate()

	// 验证 shebang
	if !strings.HasPrefix(script, "#!/bin/bash") {
		t.Error("生成的脚本应以 #!/bin/bash 开头")
	}

	// 验证所有 agent 的函数定义存在
	expectedAgents := SupportedAgentNames()
	if len(expectedAgents) == 0 {
		t.Fatal("SupportedAgentNames() 返回空列表")
	}

	for _, name := range expectedAgents {
		// 验证函数定义：name() {
		funcDef := name + "() {"
		if !strings.Contains(script, funcDef) {
			t.Errorf("生成的脚本缺少函数定义: %s", funcDef)
		}

		// 验证函数体内包含 command -v 检测
		cmdCheck := "command -v " + name
		if !strings.Contains(script, cmdCheck) {
			t.Errorf("函数 %s 缺少 command -v 检测", name)
		}

		// 验证函数体内包含 command 调用
		cmdCall := "command " + name + " \"$@\""
		if !strings.Contains(script, cmdCall) {
			t.Errorf("函数 %s 缺少 command 调用: %s", name, cmdCall)
		}
	}
}

// TestGenerate_SyntaxValid 验证生成的 bash 脚本语法正确。
//
// 覆盖案例：脚本语法 — 生成的 bash 内容可通过 bash -n 验证
func TestGenerate_SyntaxValid(t *testing.T) {
	wl := New()
	script := wl.Generate()

	// 将脚本写入临时文件，使用 bash -n 验证语法
	tmpFile, err := os.CreateTemp(t.TempDir(), "wrapper-*.sh")
	if err != nil {
		t.Fatalf("创建临时文件失败: %v", err)
	}
	defer tmpFile.Close()

	if _, err := tmpFile.WriteString(script); err != nil {
		t.Fatalf("写入临时文件失败: %v", err)
	}

	// 使用 bash -n 验证语法
	cmd := exec.Command("bash", "-n", tmpFile.Name())
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Errorf("bash -n 语法检查失败: %v\n输出: %s", err, string(output))
	}
}

// TestGenerate_NoFunctionNameConflict 验证不同 agent 的函数名互不重叠。
//
// 覆盖案例：函数名冲突 — 不同 agent 的函数名互不重叠
func TestGenerate_NoFunctionNameConflict(t *testing.T) {
	wl := New()
	script := wl.Generate()

	// 提取所有 "name() {" 形式的函数定义
	funcNames := extractFunctionNames(t, script)

	// 验证函数数量与 supportedAgents 一致
	expectedCount := SupportedAgentCount()
	if expectedCount == 0 {
		t.Fatal("SupportedAgentCount() 返回 0")
	}

	if len(funcNames) != expectedCount {
		t.Errorf("函数定义数量 = %d, 期望 %d", len(funcNames), expectedCount)
	}

	// 验证函数名互不重叠
	seen := make(map[string]bool)
	for _, name := range funcNames {
		if seen[name] {
			t.Errorf("检测到重复的函数名: %s", name)
		}
		seen[name] = true
	}

	// 验证函数名与 SupportedAgentNames 一致
	expectedNames := SupportedAgentNames()
	nameSet := make(map[string]bool)
	for _, n := range expectedNames {
		nameSet[n] = true
	}
	for _, name := range funcNames {
		if !nameSet[name] {
			t.Errorf("函数名 %s 不在支持的 agent 列表中", name)
		}
	}
}

// extractFunctionNames 从 bash 脚本中提取所有函数定义名称。
func extractFunctionNames(t *testing.T, script string) []string {
	t.Helper()
	var names []string
	lines := strings.Split(script, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// 匹配 "name() {" 格式的函数定义
		if strings.HasSuffix(line, "() {") {
			name := strings.TrimSuffix(line, "() {")
			name = strings.TrimSpace(name)
			// 排除无效的空名称和特殊字符
			if name != "" && !strings.ContainsAny(name, " \t\r\n") {
				names = append(names, name)
			}
		}
	}
	return names
}
