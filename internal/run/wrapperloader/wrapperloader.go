// Package wrapperloader 生成包含所有已安装 AI agent wrapper 函数的 bash 脚本。
//
// Wrapper Loader 位于运行层（Run Layer），在 bash 模式下（未指定 -a 参数）
// 容器启动时将生成的 bash 脚本注入容器，使开发者可直接通过函数名调用
// 对应的 AI agent 可执行文件。
//
// 生成的脚本包含 claude、opencode、kimi、deepseek-tui 四个 agent 的
// bash 函数定义，每个函数通过 command 内建命令调用对应的可执行文件。
// 脚本语法正确，可通过 bash -n 验证。
package wrapperloader

import (
	"fmt"
	"strings"
)

// agentInfo 描述一个 AI agent 的 wrapper 函数信息。
type agentInfo struct {
	// Name 是函数名，也是对应的可执行文件名称。
	Name string
	// Description 是函数的注释说明。
	Description string
}

// supportedAgents 是支持的 AI agent 定义列表。
// 每个 agent 对应一个同名的 bash 函数包装其可执行文件调用。
var supportedAgents = []agentInfo{
	{Name: "claude", Description: "Claude - Anthropic CLI agent"},
	{Name: "opencode", Description: "Opencode - AI coding agent"},
	{Name: "kimi", Description: "Kimi - AI assistant"},
	{Name: "deepseek-tui", Description: "Deepseek-tui - DeepSeek terminal UI"},
}

// WrapperLoader 负责生成 AI agent wrapper bash 脚本。
//
// 它不依赖任何外部组件，仅生成纯 bash 脚本字符串。
// 生成的脚本可通过 bash -n 验证语法正确性。
type WrapperLoader struct{}

// New 创建一个新的 Wrapper Loader 实例。
func New() *WrapperLoader {
	return &WrapperLoader{}
}

// Generate 生成包含所有已安装 AI agent wrapper 函数的 bash 脚本。
//
// 返回的脚本中包含 claude、opencode、kimi、deepseek-tui 四个 bash 函数。
// 每个函数使用 command 内建命令调用对应的可执行文件，避免递归调用自身。
// 脚本语法正确，可通过 bash -n 验证。
//
// 生成的脚本结构：
//  1. shebang（#!/bin/bash）
//  2. 文件头注释（说明、警告）
//  3. claude() 函数定义
//  4. opencode() 函数定义
//  5. kimi() 函数定义
//  6. deepseek-tui() 函数定义
func (wl *WrapperLoader) Generate() string {
	var builder strings.Builder

	// 文件头
	builder.WriteString("#!/bin/bash\n")
	builder.WriteString("# AgentForge Wrapper Functions - 自动生成\n")
	builder.WriteString("#\n")
	builder.WriteString("# 警告：此文件由 AgentForge Wrapper Loader 自动生成，请勿手动修改。\n")
	builder.WriteString("#\n")
	builder.WriteString("# 在 bash 模式下（未指定 -a 参数），此脚本被加载到容器环境中，\n")
	builder.WriteString("# 使开发者可直接通过函数名调用对应的 AI agent。\n")
	builder.WriteString("# 每个函数使用 'command' 内建命令调用可执行文件，避免递归。\n")
	builder.WriteString("\n")

	// 为每个 agent 生成 wrapper 函数
	for _, agent := range supportedAgents {
		builder.WriteString(generateFunction(agent))
	}

	return builder.String()
}

// generateFunction 生成单个 agent 的 wrapper bash 函数定义。
func generateFunction(agent agentInfo) string {
	var sb strings.Builder

	// 函数注释
	fmt.Fprintf(&sb, "# %s\n", agent.Description)
	fmt.Fprintf(&sb, "#\n")
	fmt.Fprintf(&sb, "# 用法: %s [options] [arguments]\n", agent.Name)
	fmt.Fprintf(&sb, "#\n")
	fmt.Fprintf(&sb, "# 将所有参数透传给 %s 可执行文件。\n", agent.Name)
	fmt.Fprintf(&sb, "# 若 %s 未安装，输出提示信息。\n", agent.Name)

	// 函数定义
	fmt.Fprintf(&sb, "%s() {\n", agent.Name)
	fmt.Fprintf(&sb, "    if ! command -v %s &> /dev/null; then\n", agent.Name)
	fmt.Fprintf(&sb, "        echo \"[AgentForge] %s 未安装，请先通过 build 命令构建包含 %s 的镜像\" >&2\n", agent.Name, agent.Name)
	fmt.Fprintf(&sb, "        return 1\n")
	fmt.Fprintf(&sb, "    fi\n")
	fmt.Fprintf(&sb, "    command %s \"$@\"\n", agent.Name)
	fmt.Fprintf(&sb, "}\n")

	// 空行分隔不同函数
	sb.WriteString("\n")

	return sb.String()
}

// SupportedAgentNames 返回所有支持的 agent 名称列表。
func SupportedAgentNames() []string {
	names := make([]string, len(supportedAgents))
	for i, agent := range supportedAgents {
		names[i] = agent.Name
	}
	return names
}

// SupportedAgentCount 返回支持的 agent 数量。
func SupportedAgentCount() int {
	return len(supportedAgents)
}
