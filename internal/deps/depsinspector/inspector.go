// Package depsinspector 提供容器内依赖安装状态检查功能。
//
// 通过生成检测脚本，使用 `docker run --rm` 在临时容器中执行，
// 按 agent/runtime/tool 分类输出各组件的安装状态和版本信息。
// 遵循 REQ-33 规范。
package depsinspector

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// depCheckItem 定义待检测的依赖项。
type depCheckItem struct {
	Name    string // 组件名称（如 "claude", "golang"）
	Type    string // 分类（"agent", "runtime", "tool"）
	Command string // 检测命令（如 "claude --version"）
	Which   string // 检查可执行文件路径（如 "claude"）
}

// depCheckItems 是待检测的依赖项列表。
var depCheckItems = []depCheckItem{
	// --- Agents ---
	{Name: "claude", Type: "agent", Which: "claude", Command: "claude --version 2>/dev/null | head -1"},
	{Name: "opencode", Type: "agent", Which: "opencode", Command: "opencode --version 2>/dev/null | head -1"},
	{Name: "kimi", Type: "agent", Which: "kimi", Command: "kimi --version 2>/dev/null | head -1"},
	{Name: "deepseek-tui", Type: "agent", Which: "deepseek-tui", Command: "deepseek-tui --version 2>/dev/null | head -1"},

	// --- Runtimes ---
	{Name: "golang", Type: "runtime", Which: "go", Command: "go version 2>/dev/null"},
	{Name: "node", Type: "runtime", Which: "node", Command: "node --version 2>/dev/null"},

	// --- Tools ---
	{Name: "docker", Type: "tool", Which: "docker", Command: "docker --version 2>/dev/null"},
	{Name: "rtk", Type: "tool", Which: "rtk", Command: "rtk --version 2>/dev/null | head -1"},
}

// DependencyStatus 表示单个依赖的检测结果。
type DependencyStatus struct {
	Name    string `json:"name"`    // 组件名称
	Type    string `json:"type"`    // 分类（"agent", "runtime", "tool"）
	Status  string `json:"status"`  // 状态（"installed" 或 "missing"）
	Version string `json:"version"` // 版本信息（缺失时为空）
}

// InspectionResult 表示容器内依赖检测的完整结果。
type InspectionResult struct {
	Items []DependencyStatus `json:"items"`
}

// CommandRunner 是 exec.Command 的可 mock 封装。
// 用于在测试中替换实际的 docker 命令执行。
var CommandRunner = func(name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	return cmd.CombinedOutput()
}

// GenerateScript 生成检测脚本。
//
// 脚本遍历所有依赖项，检查对应可执行文件是否存在并获取版本信息。
// 每行输出格式：component|type|status|version
// 返回值是脚本字符串，可独立测试。
func GenerateScript() string {
	var buf bytes.Buffer
	buf.WriteString("#!/bin/bash\n")
	buf.WriteString("# agent-forge deps inspector - auto-generated detection script\n\n")

	for _, item := range depCheckItems {
		buf.WriteString(fmt.Sprintf("# %s\n", item.Name))
		buf.WriteString(fmt.Sprintf(`if command -v %s &>/dev/null; then
  V=$(%s)
  echo "%s|%s|installed|${V:-installed}"
else
  echo "%s|%s|missing|"
fi
`, item.Which, item.Command, item.Name, item.Type, item.Name, item.Type))
	}

	return buf.String()
}

// ParseOutput 解析容器输出为 InspectionResult。
//
// data 是容器的 stdout/stderr 输出字节流。
// 返回解析后的检测结果，可独立测试。
func ParseOutput(data []byte) (*InspectionResult, error) {
	result := &InspectionResult{}
	scanner := bufio.NewScanner(bytes.NewReader(data))

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "|", 4)
		if len(parts) < 3 {
			continue
		}

		status := DependencyStatus{
			Name:   parts[0],
			Type:   parts[1],
			Status: parts[2],
		}
		if len(parts) >= 4 {
			status.Version = parts[3]
		}

		result.Items = append(result.Items, status)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("读取检测结果时出错: %w", err)
	}

	return result, nil
}

// RunDetection 执行完整的依赖检测流程。
//
// 使用 docker run --rm 在指定镜像的临时容器中执行检测脚本，
// 收集并返回各依赖的安装状态。容器执行完毕后自动销毁（--rm）。
//
// 当 imageRef 为空时默认使用 "agent-forge:latest"。
func RunDetection(imageRef string) (*InspectionResult, error) {
	ref := imageRef
	if ref == "" {
		ref = "agent-forge:latest"
	}

	script := GenerateScript()

	// 使用 docker run --rm 执行检测脚本（REQ-33）
	output, err := CommandRunner("docker", "run", "--rm", ref, "bash", "-c", script)
	if err != nil {
		return nil, fmt.Errorf("docker run --rm 执行失败: %w\n输出: %s", err, string(output))
	}

	result, err := ParseOutput(output)
	if err != nil {
		return nil, fmt.Errorf("解析检测结果失败: %w", err)
	}

	return result, nil
}

// FormatResult 将检测结果格式化为表格输出。
func FormatResult(result *InspectionResult) string {
	if result == nil {
		return "未检测到任何组件\n"
	}

	var buf bytes.Buffer

	if len(result.Items) == 0 {
		buf.WriteString("未检测到任何组件\n")
		return buf.String()
	}

	buf.WriteString("依赖检测结果\n")
	buf.WriteString(strings.Repeat("-", 60))
	buf.WriteString("\n")

	// 按类型分组输出
	typeGroups := map[string][]DependencyStatus{}
	for _, item := range result.Items {
		typeGroups[item.Type] = append(typeGroups[item.Type], item)
	}

	for _, typeName := range []string{"agent", "runtime", "tool"} {
		items, ok := typeGroups[typeName]
		if !ok {
			continue
		}
		buf.WriteString(fmt.Sprintf("\n[%s]\n", typeName))
		for _, item := range items {
			statusSymbol := "OK"
			if item.Status != "installed" {
				statusSymbol = "XX"
			}
			versionInfo := item.Version
			if versionInfo == "" {
				versionInfo = "-"
			}
			buf.WriteString(fmt.Sprintf("  %s  %-15s %s\n", statusSymbol, item.Name, versionInfo))
		}
	}

	buf.WriteString("\n")

	// 统计
	total := len(result.Items)
	installed := 0
	for _, item := range result.Items {
		if item.Status == "installed" {
			installed++
		}
	}
	buf.WriteString(fmt.Sprintf("总计: %d/%d 已安装\n", installed, total))

	return buf.String()
}
