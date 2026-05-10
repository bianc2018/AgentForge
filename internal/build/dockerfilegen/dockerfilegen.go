// Package dockerfilegen 根据依赖列表和构建参数动态生成合法的 Dockerfile 内容。
//
// Dockerfile Generator 位于构建层（Build Layer），接受已展开的依赖列表和
// 构建参数，输出完整的 Dockerfile 字符串。支持国内镜像源自动配置和
// GitHub 代理 URL 注入。
package dockerfilegen

import (
	"fmt"
	"strings"

	"github.com/agent-forge/cli/internal/build/depsmodule"
)

// Options 是 Dockerfile 生成所需的参数集。
type Options struct {
	// BaseImage 是基础镜像名称。
	// 空值使用默认值 "docker.1ms.run/centos:7"。
	BaseImage string

	// Deps 是已展开的依赖名称列表（来自 depsmodule.ExpandDeps）。
	// 空列表生成仅包含基础配置的最小 Dockerfile。
	Deps []string

	// GHProxy 是 GitHub 代理 URL。
	// 空字符串表示不使用代理。
	GHProxy string

	// NoCache 表示构建时是否需要跳过缓存（不影响 Dockerfile 生成）。
	NoCache bool
}

const (
	// DefaultBaseImage 是未指定 -b 参数时的默认基础镜像。
	DefaultBaseImage = "docker.1ms.run/centos:7"
)

// Generate 根据提供的选项生成完整的 Dockerfile 内容。
//
// 生成的 Dockerfile 结构：
//  1. FROM 指令（基础镜像）
//  2. 系统基础工具安装（curl、git 等）
//  3. 国内镜像源配置（npm、pip、yum）
//  4. GitHub 代理 URL 注入（可选）
//  5. 依赖安装指令（根据每种依赖的 install method）
//  6. 默认 entrypoint
//
// 返回合法的 Dockerfile 字符串，不包含注释。
func Generate(opts Options) (string, error) {
	baseImage := opts.BaseImage
	if baseImage == "" {
		baseImage = DefaultBaseImage
	}

	var sb strings.Builder

	// 1. FROM
	fmt.Fprintf(&sb, "FROM %s\n", baseImage)

	// 2. 系统基础工具（始终安装）
	sb.WriteString("\n# 安装基础工具\n")
	sb.WriteString("RUN yum install -y epel-release && \\\n")
	sb.WriteString("    yum install -y curl git wget tar gzip unzip && \\\n")
	sb.WriteString("    yum clean all && rm -rf /var/cache/yum/*\n")

	// 3. 语言运行时环境准备
	sb.WriteString("\n# 安装 Node.js (npm) 和 Python3 (pip3)\n")
	sb.WriteString("RUN curl -fsSL https://rpm.nodesource.com/setup_20.x -o /tmp/nodesetup.sh && \\\n")
	sb.WriteString("    bash /tmp/nodesetup.sh && \\\n")
	sb.WriteString("    yum install -y nodejs python3 python3-pip && \\\n")
	sb.WriteString("    rm -f /tmp/nodesetup.sh && \\\n")
	sb.WriteString("    yum clean all && rm -rf /var/cache/yum/*\n")

	// 4. 国内镜像源配置（使用国内基础镜像或显式指定 --mirror）
	sb.WriteString("\n# 配置国内镜像源\n")
	sb.WriteString("RUN npm config set registry https://registry.npmmirror.com && \\\n")
	sb.WriteString("    pip3 config set global.index-url https://mirrors.aliyun.com/pypi/simple/ && \\\n")
	sb.WriteString("    pip3 config set global.trusted-host mirrors.aliyun.com\n")

	// 5. GitHub 代理配置
	if opts.GHProxy != "" {
		sb.WriteString("\n# GitHub 代理配置\n")
		fmt.Fprintf(&sb, "ENV GH_PROXY_URL=%s\n", opts.GHProxy)
	}

	// 6. 安装依赖
	if len(opts.Deps) > 0 {
		sb.WriteString("\n# 安装依赖\n")
		for _, dep := range opts.Deps {
			method, err := depsmodule.ResolveInstallMethod(dep)
			if err != nil {
				return "", fmt.Errorf("解析依赖 %q 安装方式失败: %w", dep, err)
			}

			// 添加注释标记依赖名称
			fmt.Fprintf(&sb, "\n# %s (%s)\n", method.Name, method.Type)

			// 应用 gh-proxy 到 Go 下载
			commands := applyGHProxy(method.Commands, opts.GHProxy)

			for _, cmd := range commands {
				fmt.Fprintf(&sb, "RUN %s\n", cmd)
			}
		}
	}

	// 7. 清理安装缓存
	sb.WriteString("\n# 清理安装缓存\n")
	sb.WriteString("RUN npm cache clean --force 2>/dev/null || true && \\\n")
	sb.WriteString("    pip3 cache purge 2>/dev/null || true && \\\n")
	sb.WriteString("    yum clean all 2>/dev/null || true && \\\n")
	sb.WriteString("    rm -rf /tmp/*\n")

	// 8. 默认 entrypoint
	sb.WriteString("\n# 默认命令\n")
	sb.WriteString("CMD [\"/bin/bash\"]\n")

	return sb.String(), nil
}

// applyGHProxy 在命令中替换 GitHub 相关 URL 为代理 URL。
func applyGHProxy(commands []string, ghProxy string) []string {
	if ghProxy == "" {
		return commands
	}

	result := make([]string, len(commands))
	for i, cmd := range commands {
		// 替换 go.dev 下载 URL
		cmd = strings.ReplaceAll(cmd, "https://go.dev/dl/", ghProxy+"https://go.dev/dl/")
		// 替换 github.com 链接
		cmd = strings.ReplaceAll(cmd, "https://github.com/", ghProxy+"https://github.com/")
		// 替换 nodesource 链接（不受 gh-proxy 影响）
		result[i] = cmd
	}
	return result
}
