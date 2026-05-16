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

// ImageFamily 表示基础镜像的发行版家族。
type ImageFamily int

const (
	// FamilyUnknown 表示无法识别的镜像家族，回退到 RHEL/CentOS 行为。
	FamilyUnknown ImageFamily = iota
	// FamilyRHEL 表示 Red Hat 系（CentOS、RHEL、Fedora），使用 yum/dnf。
	FamilyRHEL
	// FamilyDebian 表示 Debian 系（Ubuntu、Debian），使用 apt。
	FamilyDebian
	// FamilyWindows 表示 Windows 系（Nanoserver、ServerCore），使用 PowerShell。
	FamilyWindows
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

// detectImageFamily 根据基础镜像名称推断发行版家族。
func detectImageFamily(baseImage string) ImageFamily {
	lower := strings.ToLower(baseImage)
	// Windows 关键字优先匹配
	for _, kw := range []string{"windows", "nanoserver", "servercore"} {
		if strings.Contains(lower, kw) {
			return FamilyWindows
		}
	}
	for _, kw := range []string{"ubuntu", "debian"} {
		if strings.Contains(lower, kw) {
			return FamilyDebian
		}
	}
	for _, kw := range []string{"centos", "rhel", "fedora", "rocky", "almalinux", "oraclelinux"} {
		if strings.Contains(lower, kw) {
			return FamilyRHEL
		}
	}
	return FamilyUnknown
}

// runtimeNeeds 描述一组依赖需要的运行时环境。
type runtimeNeeds struct {
	needsNpm bool // 至少一个依赖使用了 npm 安装命令
	needsPip bool // 至少一个依赖使用了 pip/pip3 安装命令
	needsGCC bool // 需要 gcc 编译工具（npm 原生模块需要）
}

// analyzeRuntimeNeeds 预扫描依赖的安装命令，判断需要哪些运行时环境。
func analyzeRuntimeNeeds(deps []string) (*runtimeNeeds, error) {
	needs := &runtimeNeeds{}
	for _, dep := range deps {
		method, err := depsmodule.ResolveInstallMethod(dep)
		if err != nil {
			return nil, fmt.Errorf("分析依赖 %q 安装方式失败: %w", dep, err)
		}
		for _, cmd := range method.Commands {
			cmdLower := strings.ToLower(cmd)
			if strings.Contains(cmdLower, "npm") {
				needs.needsNpm = true
				// npm 安装原生模块通常需要 gcc/g++
				needs.needsGCC = true
			}
			if strings.Contains(cmdLower, "pip3") || strings.Contains(cmdLower, "pip install") {
				needs.needsPip = true
			}
		}
	}
	return needs, nil
}

// Generate 根据提供的选项生成完整的 Dockerfile 内容。
//
// 生成的 Dockerfile 结构：
//  1. FROM 指令（基础镜像）
//  2. 镜像源配置（根据基础镜像家族选择 yum 或 apt）
//  3. 系统基础工具安装（curl、git 等）
//  4. 条件编译工具安装（仅 npm 依赖需要时）
//  5. 条件语言运行时安装（Node.js/npm、Python3/pip，按需安装）
//  6. 条件 npm/pip 镜像源配置
//  7. GitHub 代理 URL 注入（可选）
//  8. 依赖安装指令（根据每种依赖的 install method）
//  9. 清理安装缓存
//  10. 默认 entrypoint
//
// 返回合法的 Dockerfile 字符串，不包含注释。
func Generate(opts Options) (string, error) {
	baseImage := opts.BaseImage
	if baseImage == "" {
		baseImage = DefaultBaseImage
	}
	family := detectImageFamily(baseImage)

	// 预分析依赖的运行时需求
	var needs *runtimeNeeds
	if len(opts.Deps) > 0 {
		var err error
		needs, err = analyzeRuntimeNeeds(opts.Deps)
		if err != nil {
			return "", err
		}
	} else {
		needs = &runtimeNeeds{}
	}

	var sb strings.Builder

	// 1. FROM
	fmt.Fprintf(&sb, "FROM %s\n", baseImage)

	// 2. 镜像源配置 + 基础工具安装
	switch family {
	case FamilyDebian:
		writeDebianSetup(&sb)
	case FamilyWindows:
		writeWindowsSetup(&sb)
	default:
		writeRHELSetup(&sb)
	}

	// Windows 跳过 curl 配置（使用 Invoke-WebRequest 替代）
	if family != FamilyWindows {
		sb.WriteString("\n# 配置 curl 超时与重试（应对网络缓慢）\n")
		sb.WriteString("RUN echo '--connect-timeout 30' >> /root/.curlrc && \\\n")
		sb.WriteString("    echo '--max-time 600' >> /root/.curlrc && \\\n")
		sb.WriteString("    echo '--retry 3' >> /root/.curlrc && \\\n")
		sb.WriteString("    echo '--retry-delay 10' >> /root/.curlrc\n")
	}

	// 3. 条件编译工具安装（Windows 跳过——使用 MSVC 或预编译二进制）
	if needs.needsGCC && family != FamilyWindows {
		switch family {
		case FamilyDebian:
			sb.WriteString("\n# 安装编译工具（用于 npm 原生模块编译）\n")
			sb.WriteString("RUN apt-get update && \\\n")
			sb.WriteString("    apt-get install -y build-essential && \\\n")
			sb.WriteString("    rm -rf /var/lib/apt/lists/*\n")
		default:
			sb.WriteString("\n# 安装编译工具（用于 npm 原生模块编译）\n")
			sb.WriteString("RUN yum install -y make gcc gcc-c++ && \\\n")
			sb.WriteString("    yum clean all && rm -rf /var/cache/yum/*\n")
		}
	}

	// 4. 条件语言运行时安装
	if needs.needsNpm {
		switch family {
		case FamilyWindows:
			// Windows: 下载 Node.js MSI 静默安装
			sb.WriteString("\n# 安装 Node.js (npm) — Windows MSI 静默安装\n")
			sb.WriteString("RUN Invoke-WebRequest -Uri 'https://nodejs.org/dist/v22.11.0/node-v22.11.0-x64.msi' -OutFile \"$env:TEMP\\node.msi\"; \\\n")
			sb.WriteString("    Start-Process msiexec.exe -ArgumentList '/i',\"$env:TEMP\\node.msi\",'/qn','/norestart' -Wait -NoNewWindow; \\\n")
			sb.WriteString("    Remove-Item \"$env:TEMP\\node.msi\" -Force\n")

			sb.WriteString("\n# 配置 npm 国内镜像源\n")
			sb.WriteString("ENV npm_config_registry=https://registry.npmmirror.com\n")
		case FamilyDebian:
			// Ubuntu/Debian 使用 NodeSource deb 仓库，支持最新版
			sb.WriteString("\n# 安装 Node.js (npm)\n")
			sb.WriteString("RUN curl -fsSL https://deb.nodesource.com/setup_22.x -o /tmp/nodesetup.sh && \\\n")
			sb.WriteString("    bash /tmp/nodesetup.sh && \\\n")
			sb.WriteString("    apt-get install -y nodejs && \\\n")
			sb.WriteString("    rm -f /tmp/nodesetup.sh && \\\n")
			sb.WriteString("    rm -rf /var/lib/apt/lists/*\n")

			sb.WriteString("\n# 配置 npm 国内镜像源\n")
			sb.WriteString("ENV npm_config_registry=https://registry.npmmirror.com\n")
		default:
			// CentOS 7 glibc 2.17 无法支持 Node.js >= 18，使用 nodesource 16.x
			sb.WriteString("\n# 安装 Node.js (npm)\n")
			sb.WriteString("RUN curl -fsSL https://rpm.nodesource.com/setup_16.x -o /tmp/nodesetup.sh && \\\n")
			sb.WriteString("    bash /tmp/nodesetup.sh && \\\n")
			sb.WriteString("    yum install -y nodejs && \\\n")
			sb.WriteString("    rm -f /tmp/nodesetup.sh && \\\n")
			sb.WriteString("    yum clean all && rm -rf /var/cache/yum/*\n")

			sb.WriteString("\n# 配置 npm 国内镜像源\n")
			sb.WriteString("ENV npm_config_registry=https://registry.npmmirror.com\n")
		}
	}
	if needs.needsPip {
		switch family {
		case FamilyWindows:
			// Windows: 下载 Python 安装器
			sb.WriteString("\n# 安装 Python3 (pip3) — Windows 安装器\n")
			sb.WriteString("RUN Invoke-WebRequest -Uri 'https://www.python.org/ftp/python/3.12.7/python-3.12.7-amd64.exe' -OutFile \"$env:TEMP\\python.exe\"; \\\n")
			sb.WriteString("    Start-Process \"$env:TEMP\\python.exe\" -ArgumentList '/quiet','InstallAllUsers=1','PrependPath=1','Include_pip=1' -Wait -NoNewWindow; \\\n")
			sb.WriteString("    Remove-Item \"$env:TEMP\\python.exe\" -Force\n")

			sb.WriteString("\n# 配置 pip 国内镜像源\n")
			sb.WriteString("ENV PIP_INDEX_URL=https://mirrors.aliyun.com/pypi/simple/\n")
		case FamilyDebian:
			sb.WriteString("\n# 安装 Python3 (pip3)\n")
			sb.WriteString("RUN apt-get update && \\\n")
			sb.WriteString("    apt-get install -y python3 python3-pip && \\\n")
			sb.WriteString("    rm -rf /var/lib/apt/lists/*\n")

			sb.WriteString("\n# 配置 pip 国内镜像源\n")
			sb.WriteString("RUN pip3 config set global.index-url https://mirrors.aliyun.com/pypi/simple/\n")
		default:
			sb.WriteString("\n# 安装 Python3 (pip3)\n")
			sb.WriteString("RUN yum install -y python3 python3-pip && \\\n")
			sb.WriteString("    yum clean all && rm -rf /var/cache/yum/*\n")

			// CentOS 7 的 pip 9.0.3 不支持 `pip config set`，使用环境变量
			sb.WriteString("\n# 配置 pip 国内镜像源\n")
			sb.WriteString("ENV PIP_INDEX_URL=https://mirrors.aliyun.com/pypi/simple/\n")
			sb.WriteString("ENV PIP_TRUSTED_HOST=mirrors.aliyun.com\n")
		}
	}

	// 5. GitHub 代理配置
	if opts.GHProxy != "" {
		sb.WriteString("\n# GitHub 代理配置\n")
		fmt.Fprintf(&sb, "ENV GH_PROXY_URL=%s\n", opts.GHProxy)
	}

	// 6. 安装依赖
	if len(opts.Deps) > 0 {
		sb.WriteString("\n# 安装依赖\n")
		for _, dep := range opts.Deps {
			// 跳过已由语言运行时处理的依赖（避免重复安装和版本冲突）
			if needs.needsNpm && strings.HasPrefix(dep, "node") {
				fmt.Fprintf(&sb, "\n# %s: 跳过（Node.js 已由运行时层安装）\n", dep)
				continue
			}

			method, err := depsmodule.ResolveInstallMethod(dep)
			if err != nil {
				return "", fmt.Errorf("解析依赖 %q 安装方式失败: %w", dep, err)
			}

			fmt.Fprintf(&sb, "\n# %s (%s)\n", method.Name, method.Type)

			commands := applyGHProxy(method.Commands, opts.GHProxy)

			for _, cmd := range commands {
				cmd = adaptCommandForFamily(cmd, family)
				fmt.Fprintf(&sb, "RUN %s\n", cmd)
			}
		}
	}

	// 7. 清理安装缓存
	sb.WriteString("\n# 清理安装缓存\n")
	if family == FamilyWindows {
		if needs.needsNpm || needs.needsPip {
			sb.WriteString("RUN ")
			if needs.needsNpm {
				sb.WriteString("npm cache clean --force 2>$null || $true; ")
			}
			if needs.needsPip {
				sb.WriteString("pip3 cache purge 2>$null || $true; ")
			}
			sb.WriteString(`Remove-Item -Force "$env:TEMP\*" -ErrorAction SilentlyContinue`)
			sb.WriteString("\n")
		}
	} else {
		sb.WriteString("RUN ")
		if needs.needsNpm {
			sb.WriteString("npm cache clean --force 2>/dev/null || true && \\\n    ")
		}
		if needs.needsPip {
			sb.WriteString("pip3 cache purge 2>/dev/null || true && \\\n    ")
		}
		switch family {
		case FamilyDebian:
			sb.WriteString("apt-get clean 2>/dev/null || true && \\\n")
			sb.WriteString("    rm -rf /var/lib/apt/lists/* /tmp/*\n")
		default:
			sb.WriteString("yum clean all 2>/dev/null || true && \\\n")
			sb.WriteString("    rm -rf /tmp/*\n")
		}
	}

	// 8. 默认 entrypoint
	sb.WriteString("\n# 默认命令\n")
	if family == FamilyWindows {
		sb.WriteString("CMD [\"powershell\"]\n")
	} else {
		sb.WriteString("CMD [\"/bin/bash\"]\n")
	}

	return sb.String(), nil
}

// writeDebianSetup 生成 Debian/Ubuntu 系的镜像源配置和基础工具安装。
func writeDebianSetup(sb *strings.Builder) {
	// 配置 apt 阿里云镜像源（加速 apt-get update）
	sb.WriteString("\n# 配置 APT 镜像源（阿里云）\n")
	sb.WriteString("RUN sed -i 's|http://archive.ubuntu.com/ubuntu/|http://mirrors.aliyun.com/ubuntu/|g' /etc/apt/sources.list && \\\n")
	sb.WriteString("    sed -i 's|http://security.ubuntu.com/ubuntu/|http://mirrors.aliyun.com/ubuntu/|g' /etc/apt/sources.list && \\\n")
	sb.WriteString("    sed -i 's|http://deb.debian.org/debian|http://mirrors.aliyun.com/debian|g' /etc/apt/sources.list || true\n")

	// 基础工具安装
	sb.WriteString("\n# 安装基础工具\n")
	sb.WriteString("RUN apt-get update && \\\n")
	sb.WriteString("    apt-get install -y curl git wget tar gzip unzip ca-certificates && \\\n")
	sb.WriteString("    rm -rf /var/lib/apt/lists/*\n")
}

// writeRHELSetup 生成 RHEL/CentOS 系的镜像源配置和基础工具安装。
func writeRHELSetup(sb *strings.Builder) {
	// CentOS 7 已于 2024 年 6 月 EOL，mirrorlist 已失效
	sb.WriteString("\n# 配置 YUM 镜像源（CentOS 7 EOL，切换阿里云 Vault）\n")
	sb.WriteString("RUN sed -i 's|^mirrorlist=|#mirrorlist=|' /etc/yum.repos.d/CentOS-*.repo && \\\n")
	sb.WriteString("    sed -i 's|^#baseurl=http://mirror.centos.org/centos/$releasever|baseurl=https://mirrors.aliyun.com/centos-vault/7.9.2009|' /etc/yum.repos.d/CentOS-*.repo || true\n")

	// 基础工具
	sb.WriteString("\n# 安装基础工具\n")
	sb.WriteString("RUN yum install -y epel-release && \\\n")
	sb.WriteString("    yum install -y curl git wget tar gzip unzip && \\\n")
	sb.WriteString("    yum clean all && rm -rf /var/cache/yum/*\n")
}

// writeWindowsSetup 生成 Windows 系的 SHELL 指令和基础工具安装。
//
// Windows 容器默认 shell 为 cmd.exe，显式切换到 PowerShell。
// Nanoserver 缺少 git、curl 等工具，通过 Invoke-WebRequest 按需下载。
func writeWindowsSetup(sb *strings.Builder) {
	sb.WriteString("\n# 使用 PowerShell 作为默认 shell\n")
	sb.WriteString("SHELL [\"powershell\", \"-Command\"]\n")

	sb.WriteString("\n# 安装 Git for Windows\n")
	sb.WriteString("RUN Invoke-WebRequest -Uri 'https://github.com/git-for-windows/git/releases/download/v2.47.1.windows.2/Git-2.47.1.2-64-bit.exe' -OutFile \"$env:TEMP\\Git.exe\"; \\\n")
	sb.WriteString("    Start-Process -FilePath \"$env:TEMP\\Git.exe\" -ArgumentList '/VERYSILENT','/NORESTART','/NOCANCEL','/SP-','/SUPPRESSMSGBOXES','/CLOSEAPPLICATIONS','/RESTARTAPPLICATIONS' -Wait -NoNewWindow; \\\n")
	sb.WriteString("    Remove-Item \"$env:TEMP\\Git.exe\" -Force\n")

	sb.WriteString("\n# 将 Git 加入 PATH\n")
	sb.WriteString("RUN $env:PATH = [Environment]::GetEnvironmentVariable('PATH', 'Machine') + ';' + [Environment]::GetEnvironmentVariable('PATH', 'User'); \\\n")
	sb.WriteString("    [Environment]::SetEnvironmentVariable('PATH', $env:PATH, 'Machine')\n")
}

// pkgNameMap 将 RHEL 系常见包名映射为 Debian 系等效包名。
// 用于 DepSystemPkg 类型依赖在 Debian 系基础镜像上使用正确的包名。
var pkgNameMap = map[string]string{
	"docker": "docker.io",
}

// adaptCommandForFamily 将 RHEL 系包管理命令翻译为 Debian 系等效命令，
// 将 Linux 命令翻译为 PowerShell 等效命令。
// 对于非系统包管理的命令（curl、npm、pip 等）原样返回。
func adaptCommandForFamily(cmd string, family ImageFamily) string {
	switch family {
	case FamilyDebian:
		cmd = strings.ReplaceAll(cmd, "yum install -y", "apt-get install -y")
		cmd = strings.ReplaceAll(cmd, "yum clean all && rm -rf /var/cache/yum/*", "apt-get clean && rm -rf /var/lib/apt/lists/*")
		for rhelPkg, debPkg := range pkgNameMap {
			cmd = strings.ReplaceAll(cmd, "apt-get install -y "+rhelPkg, "apt-get install -y "+debPkg)
		}
	case FamilyWindows:
		cmd = adaptToWindowsCommand(cmd)
	}
	return cmd
}

// adaptToWindowsCommand 将 Linux shell 命令翻译为 PowerShell 等效命令。
//
// 翻译规则：
//   - curl -fsSL <url> -o <path> → Invoke-WebRequest -Uri <url> -OutFile <path>
//   - tar -xzf <file> → Expand-Archive -Path <file> -DestinationPath <dest>
//   - bash <script> → powershell -File <script>
//   - chmod +x <file> → 跳过（Windows 不需要可执行位）
func adaptToWindowsCommand(cmd string) string {
	// curl -fsSL <url> -o <path> → Invoke-WebRequest
	cmd = strings.ReplaceAll(cmd, "curl -fsSL", "Invoke-WebRequest -Uri")
	// Replace -o flag with -OutFile
	cmd = strings.ReplaceAll(cmd, " -o ", " -OutFile ")
	// tar -xzf <file> → Expand-Archive (assume same directory)
	cmd = strings.ReplaceAll(cmd, "tar -xzf ", "Expand-Archive -Path ")
	// bash <script> → powershell -File <script>
	cmd = strings.ReplaceAll(cmd, "bash ", "powershell -File ")
	// chmod +x → skip (write-host as no-op)
	cmd = strings.ReplaceAll(cmd, "chmod +x ", "Write-Host 'skip chmod: ")
	// Append closing quote for chmod skip
	if strings.Contains(cmd, "Write-Host 'skip chmod: ") && !strings.HasSuffix(cmd, "'") {
		cmd += "'"
	}
	// /tmp/ → $env:TEMP\
	cmd = strings.ReplaceAll(cmd, "/tmp/", `$env:TEMP\`)
	// Replace Linux path separators with Windows in file paths (not URLs)
	// Only for paths starting with / that aren't http/https
	if !strings.Contains(cmd, "http") {
		cmd = strings.ReplaceAll(cmd, "/usr/local/", `C:\ProgramData\`)
		cmd = strings.ReplaceAll(cmd, "/usr/bin/", `C:\ProgramData\`)
	}
	return cmd
}

// applyGHProxy 在命令中替换 GitHub 相关 URL 为代理 URL。
// Go 下载（golang.google.cn）可直接访问，不需要代理。
// 仅对 github.com 域名应用代理替换。
func applyGHProxy(commands []string, ghProxy string) []string {
	if ghProxy == "" {
		return commands
	}

	normalized := strings.TrimRight(ghProxy, "/") + "/"
	result := make([]string, len(commands))
	for i, cmd := range commands {
		cmd = strings.ReplaceAll(cmd, "https://github.com/", normalized+"https://github.com/")
		result[i] = cmd
	}
	return result
}
