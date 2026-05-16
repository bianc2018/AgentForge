// Package argsparser 提供 build 和 run 命令的命令行参数统一解析。
//
// Args Parser 位于共享层（Shared Layer），负责将原始命令行参数字符串切片
// 解析为类型安全的结构化参数集。支持短参/长参别名、多值参数收集，
// 并为所有可选参数设定合理的默认值。
package argsparser

// BuildParams 是 build 命令的完整结构化参数集。
type BuildParams struct {
	// Deps 是 -d/--deps 参数指定的依赖列表，逗号分隔。
	// 示例："all"、"claude,golang@1.21,node@20"
	Deps string

	// BaseImage 是 -b/--base-image 参数指定的基础镜像名称。
	// 默认值："docker.1ms.run/centos:7"
	BaseImage string

	// Config 是 -c/--config 参数指定的配置目录路径。
	// 空字符串表示使用默认路径。
	Config string

	// NoCache 是 --no-cache 参数，表示是否跳过 Docker 缓存。
	NoCache bool

	// Rebuild 是 -R/--rebuild 参数，表示是否使用重建模式。
	Rebuild bool

	// MaxRetry 是 --max-retry 参数，表示网络错误时的最大重试次数。
	// 默认值：3
	MaxRetry int

	// GHProxy 是 --gh-proxy 参数，指定 GitHub 代理 URL。
	// 默认值 "https://ghproxy.net"，传空字符串表示不使用代理。
	GHProxy string
}

// DefaultBuildParams 返回 build 命令的默认参数集。
func DefaultBuildParams() BuildParams {
	return BuildParams{
		BaseImage: "docker.1ms.run/centos:7",
		MaxRetry:  3,
		GHProxy:   "https://ghproxy.net",
	}
}

// RunParams 是 run 命令的完整结构化参数集。
type RunParams struct {
	// Agent 是 -a/--agent 参数指定的 AI agent 名称。
	// 可选值：claude、opencode、kimi、deepseek-tui
	// 空字符串表示 bash 模式。
	Agent string

	// Ports 是 -p/--port 参数指定的端口映射列表，可多次出现。
	// 格式："主机端口:容器端口"，如 "3000:3000"
	Ports []string

	// Mounts 是 -m/--mount 参数指定的只读目录挂载列表，可多次出现。
	// 格式：宿主机绝对路径，如 "/host/data"
	Mounts []string

	// Envs 是 -e/--env 参数指定的环境变量列表，可多次出现。
	// 格式："KEY=VALUE"，如 "OPENAI_KEY=sk-xxx"
	Envs []string

	// Workdir 是 -w/--workdir 参数指定的容器内工作目录。
	// 空字符串表示使用默认工作目录。
	Workdir string

	// Recall 是 -r/--recall 参数，表示是否从 .last_args 恢复上次运行参数。
	Recall bool

	// Docker 是 --docker/--dind 参数，表示是否以 Docker-in-Docker 特权模式启动。
	Docker bool

	// RunCmd 是 --run 参数，表示在后台执行指定命令后退出容器。
	// 空字符串表示不启用后台命令模式。
	RunCmd string

	// Config 是 -c/--config 参数指定的配置目录路径。
	// 空字符串表示使用默认路径。
	Config string

	// BaseImage 是 -b/--base-image 参数指定的基础镜像名称。
	// 用于平台推断。
	BaseImage string

	// Platform 是从 BaseImage 推断出的目标平台（""=Linux, "windows"=Windows）。
	// 由 CLI 层在调用 RunEngine 前通过 platform.ResolvePlatform 填充。
	Platform string
}

// DefaultRunParams 返回 run 命令的默认参数集。
func DefaultRunParams() RunParams {
	return RunParams{}
}
