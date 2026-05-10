// Package runengine 编排容器运行流程。
//
// RunEngine 位于运行层（Run Layer），负责解析 run 命令参数、组装 ContainerCreate
// 配置结构、调用 Docker Helper 的容器生命周期 API，完成从参数到运行容器的完整编排。
//
// 支持四种启动模式：agent 交互模式、bash+ wrapper 模式、Docker-in-Docker 特权模式、
// 后台命令模式。当前文件实现 agent 交互模式的基础能力。
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

	"github.com/agent-forge/cli/internal/shared/argsparser"
	"github.com/agent-forge/cli/internal/shared/dockerhelper"
)

// ImageName 是 RunEngine 默认使用的 Docker 镜像名称。
const ImageName = "agent-forge:latest"

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
// 配置规则：
//   - agent 模式（-a 指定 agent）：Cmd 设置为 agent 命令，Tty=true，OpenStdin=true
//   - bash 模式（未指定 -a）：Cmd 设置为 ["bash"]，Tty=true，OpenStdin=true
//   - -p 端口映射：转换为 PortBindings 和 ExposedPorts
//   - -m 目录挂载：转换为只读 Bind Mount
//   - -e 环境变量：转换为 Env 字符串数组
//   - -w 工作目录：设置 WorkingDir
func AssembleContainerConfig(params argsparser.RunParams) (*container.Config, *container.HostConfig, *network.NetworkingConfig) {
	// --- 容器配置 (Config) ---

	// 构建 Cmd
	var cmd strslice.StrSlice
	if params.Agent != "" {
		// Agent 模式：Cmd 为 agent 可执行文件
		cmd = strslice.StrSlice{params.Agent}
	} else {
		// Bash 模式：直接启动 bash 交互式终端
		cmd = strslice.StrSlice{"bash"}
	}

	// 构建环境变量
	var env []string
	for _, e := range params.Envs {
		env = append(env, e)
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

	// --- 网络配置 ---
	networkingConfig := &network.NetworkingConfig{}

	return config, hostConfig, networkingConfig
}

// Run 执行容器运行流程。
//
// 流程：
//  1. 调用 AssembleContainerConfig 组装配置
//  2. 调用 Docker Helper 的 ContainerCreate 创建容器
//  3. 调用 Docker Helper 的 ContainerStart 启动容器
//  4. 调用 Docker Helper 的 ContainerAttach 附加到容器流
//
// 在 agent 模式下，attach 使标准输入/输出/错误流连接到容器终端。
func (e *Engine) Run(ctx context.Context, params argsparser.RunParams) error {
	config, hostConfig, networkingConfig := AssembleContainerConfig(params)

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

// Close 释放运行引擎使用的资源。
func (e *Engine) Close() error {
	return e.helper.Close()
}
