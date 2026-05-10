// Package depsmodule 管理依赖元信息和安装方式解析。
//
// Deps Module 位于构建层（Build Layer），维护 agent、runtime、tool 三类依赖的
// 安装方式元信息，提供依赖列表展开和安装指令解析功能。
package depsmodule

import (
	"fmt"
	"regexp"
	"strings"
)

// DepType 表示依赖的类型。
type DepType int

const (
	// DepAgent 表示 AI agent 类依赖。
	DepAgent DepType = iota
	// DepRuntime 表示编程语言运行时依赖。
	DepRuntime
	// DepTool 表示工具类依赖。
	DepTool
	// DepSystemPkg 表示通过系统包管理器安装的依赖（未知名称默认归为此类）。
	DepSystemPkg
)

// String 返回依赖类型的可读名称。
func (dt DepType) String() string {
	switch dt {
	case DepAgent:
		return "agent"
	case DepRuntime:
		return "runtime"
	case DepTool:
		return "tool"
	case DepSystemPkg:
		return "system"
	default:
		return "unknown"
	}
}

// InstallMethod 描述一个依赖的安装方式。
type InstallMethod struct {
	// Name 是依赖的名称。
	Name string
	// Type 是依赖的类型。
	Type DepType
	// Version 是依赖的版本（如 "1.21"），空表示未指定版本。
	Version string
	// Commands 是在 Dockerfile 中执行安装所需的 shell 命令列表。
	Commands []string
}

// depInfo 是依赖的元信息。
type depInfo struct {
	name     string
	depType  DepType
	defaultVersion string
	commands func(version string) []string
}

// 已知依赖的注册表
var knownDeps = map[string]*depInfo{
	// --- Agents ---
	"claude": {
		name:    "claude",
		depType: DepAgent,
		commands: func(version string) []string {
			return []string{"npm install -g @anthropic-ai/claude-code"}
		},
	},
	"opencode": {
		name:    "opencode",
		depType: DepAgent,
		commands: func(version string) []string {
			return []string{"npm install -g @opencode-ai/opencode"}
		},
	},
	"kimi": {
		name:    "kimi",
		depType: DepAgent,
		commands: func(version string) []string {
			return []string{"npm install -g kimi-cli"}
		},
	},
	"deepseek-tui": {
		name:    "deepseek-tui",
		depType: DepAgent,
		commands: func(version string) []string {
			return []string{"npm install -g deepseek-tui"}
		},
	},

	// --- Runtimes ---
	"golang": {
		name:    "golang",
		depType: DepRuntime,
		commands: func(version string) []string {
			v := version
			if v == "" {
				v = "1.22.3"
			}
			// Ensure version uses the full semver format (1.22 -> 1.22.3)
			parts := strings.Split(v, ".")
			if len(parts) == 2 {
				v = v + ".0"
			}
			return []string{
				fmt.Sprintf("curl -fsSL https://go.dev/dl/go%s.linux-amd64.tar.gz -o /tmp/go.tar.gz", v),
				"rm -rf /usr/local/go && tar -C /usr/local -xzf /tmp/go.tar.gz",
				"ln -sf /usr/local/go/bin/go /usr/local/bin/go",
				"rm -f /tmp/go.tar.gz",
			}
		},
	},
	"node": {
		name:    "node",
		depType: DepRuntime,
		commands: func(version string) []string {
			v := version
			if v == "" {
				v = "16.x"
			}
			// CentOS 7 glibc 2.17 仅支持 Node < 18
			return []string{
				fmt.Sprintf("curl -fsSL https://rpm.nodesource.com/setup_%s -o /tmp/nodesetup.sh", v),
				"bash /tmp/nodesetup.sh",
				"yum install -y nodejs",
				"rm -f /tmp/nodesetup.sh",
			}
		},
	},

	// --- Tools ---
	"speckit": {
		name:    "speckit",
		depType: DepTool,
		commands: func(version string) []string {
			return []string{"npm install -g @anthropic-ai/speckit"}
		},
	},
	"openspec": {
		name:    "openspec",
		depType: DepTool,
		commands: func(version string) []string {
			return []string{"pip3 install open-spec"}
		},
	},
	"gitnexus": {
		name:    "gitnexus",
		depType: DepTool,
		commands: func(version string) []string {
			return []string{"npm install -g gitnexus"}
		},
	},
	"docker": {
		name:    "docker",
		depType: DepTool,
		commands: func(version string) []string {
			return []string{"yum install -y docker"}
		},
	},
	"rtk": {
		name:    "rtk",
		depType: DepTool,
		commands: func(version string) []string {
			return []string{"pip3 install rtk"}
		},
	},

}

// allDeps 是所有依赖的完整列表（用于 "all" 元标签展开）。
var allDeps = []string{
	"claude", "opencode", "kimi", "deepseek-tui",
	"golang@1.22.3", "node@16",
	"speckit", "openspec", "gitnexus", "docker", "rtk",
}

// miniDeps 是常用依赖子集（用于 "mini" 元标签展开）。
var miniDeps = []string{
	"claude", "opencode",
	"golang@1.22.3", "node@16",
	"speckit", "gitnexus",
}

// ExpandDeps 将逗号分隔的依赖字符串展开为去重的依赖名称列表。
//
// 支持的特殊值：
//   - "all"：展开为已知的所有依赖的完整列表
//   - "mini"：展开为常用子集
//
// 输入格式：逗号分隔的依赖名称或元标签列表。
// 支持混合元标签 + 单体依赖，自动去重。
// 未知名称将原样保留在返回列表中。
//
// 示例：
//
//	ExpandDeps("all")              -> [claude opencode kimi deepseek-tui golang@1.22.3 node@16 speckit ...]
//	ExpandDeps("mini")             -> [claude opencode golang@1.22.3 node@16 speckit gitnexus]
//	ExpandDeps("claude,golang@1.21") -> [claude golang@1.21]
//	ExpandDeps("")                 -> []
func ExpandDeps(input string) []string {
	if input == "" {
		return []string{}
	}

	parts := strings.Split(input, ",")
	seen := make(map[string]bool)
	var result []string

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		var expanded []string
		switch part {
		case "all":
			expanded = allDeps
		case "mini":
			expanded = miniDeps
		default:
			expanded = []string{part}
		}

		for _, dep := range expanded {
			if !seen[dep] {
				seen[dep] = true
				result = append(result, dep)
			}
		}
	}

	return result
}

// validDepNameRegex 匹配有效的依赖名称格式。
var validDepNameRegex = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]*(@[a-zA-Z0-9._x*]+)?$`)

// ResolveInstallMethod 根据依赖名称返回对应的安装方式。
//
// name 的格式为 "依赖名" 或 "依赖名@版本"。
//
// 已知的依赖名返回其预定义的安装方式和命令列表。
// 未知的依赖名自动作为系统包名生成 yum install 指令。
// 格式无效的名称（如 "@1.21" 缺少基础名）返回错误。
func ResolveInstallMethod(name string) (*InstallMethod, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("依赖名称为空")
	}

	if !validDepNameRegex.MatchString(name) {
		return nil, fmt.Errorf("无效的依赖名称格式: %q", name)
	}

	baseName, version := splitNameVersion(name)

	if info, ok := knownDeps[baseName]; ok {
		cmds := info.commands(version)
		return &InstallMethod{
			Name:     name,
			Type:     info.depType,
			Version:  version,
			Commands: cmds,
		}, nil
	}

	// 未知名称：作为系统包通过 yum 安装
	return &InstallMethod{
		Name:    name,
		Type:    DepSystemPkg,
		Version: version,
		Commands: []string{
			fmt.Sprintf("yum install -y %s", baseName),
		},
	}, nil
}

// splitNameVersion 将 "name@version" 拆分为基础名称和版本。
// 如果没有 @ 符号，版本为空字符串。
func splitNameVersion(name string) (baseName, version string) {
	if idx := strings.LastIndex(name, "@"); idx > 0 {
		return name[:idx], name[idx+1:]
	}
	return name, ""
}

// IsKnownDep 检查给定的基础名称是否是已知的依赖。
func IsKnownDep(name string) bool {
	name = strings.TrimSpace(name)
	baseName, _ := splitNameVersion(name)
	_, ok := knownDeps[baseName]
	return ok
}

// ListAllKnownDeps 返回所有已知依赖的基础名称。
func ListAllKnownDeps() []string {
	var names []string
	for name := range knownDeps {
		names = append(names, name)
	}
	return names
}
