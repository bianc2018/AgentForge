// Package runengine 编排容器运行流程。
//
// RunEngine 位于运行层（Run Layer），负责解析 run 命令参数、组装 ContainerCreate
// 配置结构、调用 Docker Helper 的容器生命周期 API，完成从参数到运行容器的完整编排。
//
// 支持四种启动模式：agent 交互模式、bash+ wrapper 模式、Docker-in-Docker 特权模式、
// 后台命令模式。
package runengine

import (
	"context"
	"fmt"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/strslice"
	"github.com/docker/go-connections/nat"

	"github.com/agent-forge/cli/internal/run/wrapperloader"
	"github.com/agent-forge/cli/internal/shared/argsparser"
	"github.com/agent-forge/cli/internal/shared/dockerhelper"
)

// ImageName 是 RunEngine 默认使用的 Docker 镜像名称。
const ImageName = "agent-forge:latest"

// dockerdStartScript 是 Docker-in-Docker 模式下用于启动 dockerd 守护进程
// 并等待其就绪的 bash 脚本模板。%s 为 dockerd 就绪后要执行的命令（agent 或 bash）。
const dockerdStartScript = `nohup dockerd > /var/log/dockerd.log 2>&1 &
sleep 2
while ! docker info >/dev/null 2>&1; do
  sleep 0.5
done
exec %s`

// Engine 是运行引擎，负责编排完整的容器运行流程。
type Engine struct {
	helper *dockerhelper.Client
}

// New 创建新的运行引擎。
//
// 需要已经初始化的 Docker Helper 客户端。
func New(helper *dockerhelper.Client) *Engine {
	return &Engine{helper: helper}
}

// AssembleContainerConfig 根据运行参数组装 ContainerCreate 所需的配置结构体。
//
// 该函数是纯数据组装，不涉及任何 I/O 或外部调用，因此无需 mock。
// 返回容器配置（container.Config）、主机配置（container.HostConfig）和网络配置。
//
// wrapperScript 是 bash 模式下要注入的 agent wrapper 脚本内容。为空时 bash 模式
// 直接启动 bash，不加载 wrapper 函数。
//
// 配置规则：
//   - agent 模式（-a 指定 agent）：Cmd 设置为 agent 命令，Tty=true，OpenStdin=true
//   - bash 模式（未指定 -a）：Cmd 设置为 bash 加载 wrapper，Tty=true，OpenStdin=true
//   - Docker-in-Docker 模式（--docker/--dind）：Privileged=true，User="root"，
//     自动启动 dockerd 并等待就绪
//   - -p 端口映射：转换为 PortBindings 和 ExposedPorts
//   - -m 目录挂载：转换为只读 Bind Mount
//   - -e 环境变量：转换为 Env 字符串数组
//   - -w 工作目录：设置 WorkingDir
func AssembleContainerConfig(params argsparser.RunParams, wrapperScript string) (*container.Config, *container.HostConfig, *network.NetworkingConfig) {
	// --- 容器配置 (Config) ---

	// 构建 Cmd
	cmd := buildCmd(params, wrapperScript)

	// 构建环境变量
	var env []string
	for _, e := range params.Envs {
		env = append(env, e)
	}

	// bash 模式下注入 wrapper 脚本作为环境变量
	if params.Agent == "" && wrapperScript != "" {
		env = append(env, "AGENTFORGE_WRAPPER="+wrapperScript)
	}

	// 构建端口映射（ExposedPorts: 声明容器内可用的端口）
	exposedPorts := make(nat.PortSet)
	for _, p := range params.Ports {
		parts := strings.SplitN(p, ":", 2)
		if len(parts) == 2 {
			containerPortStr := parts[1]
			portKey, err := nat.NewPort("tcp", containerPortStr)
			if err == nil {
				exposedPorts[portKey] = struct{}{}
			}
		}
	}

	config := &container.Config{
		Image:        ImageName,
		Tty:          true,
		OpenStdin:    true,
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		Cmd:          cmd,
		Env:          env,
		WorkingDir:   params.Workdir,
		ExposedPorts: exposedPorts,
	}

	// Docker-in-Docker 模式：设置 root 用户
	if params.Docker {
		config.User = "root"
	}

	// --- 主机配置 (HostConfig) ---

	// 构建端口绑定
	portBindings := make(nat.PortMap)
	for _, p := range params.Ports {
		parts := strings.SplitN(p, ":", 2)
		if len(parts) == 2 {
			hostPort := parts[0]
			containerPortStr := parts[1]
			portKey, err := nat.NewPort("tcp", containerPortStr)
			if err != nil {
				continue
			}
			portBindings[portKey] = []nat.PortBinding{
				{
					HostPort: hostPort,
				},
			}
		}
	}

	// 构建目录挂载（只读）
	var mounts []mount.Mount
	for _, m := range params.Mounts {
		mounts = append(mounts, mount.Mount{
			Type:     mount.TypeBind,
			Source:   m,
			Target:   m,
			ReadOnly: true,
		})
	}

	hostConfig := &container.HostConfig{
		PortBindings: portBindings,
		Mounts:       mounts,
		AutoRemove:   false, // 交互模式不自动删除
	}

	// Docker-in-Docker 模式：设置特权模式
	if params.Docker {
		hostConfig.Privileged = true
	}

	// --- 网络配置 ---
	networkingConfig := &network.NetworkingConfig{}

	return config, hostConfig, networkingConfig
}

// buildCmd 根据运行模式构建容器入口命令。
//
// 支持四种运行模式：
//  1. Docker-in-Docker + agent：启动 dockerd 后执行 agent 命令
//  2. Docker-in-Docker + bash：启动 dockerd 后进入 bash（加载 wrapper）
//  3. Agent 模式：直接执行 agent 命令
//  4. Bash 模式（默认）：启动 bash（加载 wrapper）
func buildCmd(params argsparser.RunParams, wrapperScript string) strslice.StrSlice {
	if params.Docker {
		// Docker-in-Docker 模式
		var finalCmd string
		if params.Agent != "" {
			finalCmd = params.Agent
		} else {
			// DIND + bash 模式：加载 wrapper 后启动 bash
			finalCmd = `eval "$AGENTFORGE_WRAPPER"; exec bash`
		}
		return strslice.StrSlice{"bash", "-c", fmt.Sprintf(dockerdStartScript, finalCmd)}
	}

	if params.Agent != "" {
		// Agent 模式：Cmd 为 agent 可执行文件
		return strslice.StrSlice{params.Agent}
	}

	// Bash 模式：加载 wrapper 后启动 bash
	if wrapperScript != "" {
		return strslice.StrSlice{"bash", "-c", `eval "$AGENTFORGE_WRAPPER"; exec bash`}
	}

	return strslice.StrSlice{"bash"}
}

// Run 执行容器运行流程。
//
// 流程：
//  1. 如果为 bash 模式，从 Wrapper Loader 生成 wrapper 函数脚本
//  2. 调用 AssembleContainerConfig 组装配置
//  3. 调用 Docker Helper 的 ContainerCreate 创建容器
//  4. 调用 Docker Helper 的 ContainerStart 启动容器
//  5. 调用 Docker Helper 的 ContainerAttach 附加到容器流
//
// 在 agent 模式和 bash 模式下，attach 使标准输入/输出/错误流连接到容器终端。
// 在 Docker-in-Docker 模式下，容器启动后 dockerd 守护进程自动启动。
func (e *Engine) Run(ctx context.Context, params argsparser.RunParams) error {
	// bash 模式下生成 wrapper 脚本
	var wrapperScript string
	if params.Agent == "" {
		wl := wrapperloader.New()
		wrapperScript = wl.Generate()
	}

	config, hostConfig, networkingConfig := AssembleContainerConfig(params, wrapperScript)

	// 创建容器
	resp, err := e.helper.ContainerCreate(ctx, config, hostConfig, networkingConfig, nil, "")
	if err != nil {
		return fmt.Errorf("创建容器失败: %w", err)
	}

	containerID := resp.ID

	// 启动容器
	if err := e.helper.ContainerStart(ctx, containerID, types.ContainerStartOptions{}); err != nil {
		return fmt.Errorf("启动容器失败: %w", err)
	}

	// 附加到容器（交互模式）
	attachResp, err := e.helper.ContainerAttach(ctx, containerID, types.ContainerAttachOptions{
		Stream: true,
		Stdin:  true,
		Stdout: true,
		Stderr: true,
	})
	if err != nil {
		return fmt.Errorf("附加到容器失败: %w", err)
	}
	defer attachResp.Close()

	return nil
}

