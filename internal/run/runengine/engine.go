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
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/strslice"
	"github.com/docker/go-connections/nat"
	"github.com/moby/term"
	specs "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/agent-forge/cli/internal/run/argspersistence"
	"github.com/agent-forge/cli/internal/run/wrapperloader"
	"github.com/agent-forge/cli/internal/shared/argsparser"
	"github.com/agent-forge/cli/internal/shared/dockerhelper"
	"github.com/agent-forge/cli/internal/shared/logging"
	"github.com/agent-forge/cli/internal/shared/platform"
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

// ExitCodeError 表示容器以指定退出码退出。
//
// 用于 --run 后台命令模式，将容器退出码传递到 CLI 层。
type ExitCodeError struct {
	Code int
}

func (e *ExitCodeError) Error() string {
	return fmt.Sprintf("命令以退出码 %d 结束", e.Code)
}

// ExitCode 实现 ExitCoder 接口，返回容器退出码。
// 这样 cmd/root.go 的 Execute() 可以通过 ExitCoder 接口检测到 --run 模式
// 的容器退出码，并使用正确的进程退出码退出。
func (e *ExitCodeError) ExitCode() int {
	return e.Code
}

// Engine 是运行引擎，负责编排完整的容器运行流程。
type Engine struct {
	helper    *dockerhelper.Client
	configDir string
}

// New 创建新的运行引擎。
//
// 需要已经初始化的 Docker Helper 客户端和配置目录路径。
// configDir 用于 --recall 模式读取 .last_args 文件，以及每次运行后持久化参数。
func New(helper *dockerhelper.Client, configDir string) *Engine {
	return &Engine{helper: helper, configDir: configDir}
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

	// 注入 TERM 环境变量（自动检测主机 TERM，fallback xterm-256color）
	if !hasEnvKey(params.Envs, "TERM") {
		hostTerm := os.Getenv("TERM")
		if hostTerm == "" {
			hostTerm = "xterm-256color"
		}
		env = append(env, "TERM="+hostTerm)
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

	// 后台命令模式：关闭 Tty 交互，不挂载 Stdin
	isRunCmdMode := params.RunCmd != ""

	// 确定工作目录：-w 显式指定 > os.Getwd() > /workspace
	// 注意：Windows 上 os.Getwd() 返回 "D:\code\..." 格式，Docker 容器只认 Unix
	// 路径，因此分为两个变量：
	//   nativeWorkdir  — 宿主机原生路径，用于 os.Stat 和 mount Source
	//   containerWorkdir — Linux 兼容路径，用于 WorkingDir 和 mount Target
	nativeWorkdir := params.Workdir
	needsMount := params.Workdir != "" // -w 显式指定的路径总是需要挂载

	if nativeWorkdir == "" {
		if wd, err := os.Getwd(); err == nil {
			nativeWorkdir = wd
			needsMount = true
		} else {
			nativeWorkdir = "/workspace"
			// needsMount 保持 false，/workspace 兜底无主机路径可挂载
		}
	}
	containerWorkdir := toContainerPath(nativeWorkdir, params.Platform)

	config := &container.Config{
		Image:        ImageName,
		Tty:          !isRunCmdMode,
		OpenStdin:    !isRunCmdMode,
		AttachStdin:  !isRunCmdMode,
		AttachStdout: true,
		AttachStderr: true,
		Cmd:          cmd,
		Env:          env,
		WorkingDir:   containerWorkdir,
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

	// 工作目录自动挂载（读写，1:1 映射）
	// 仅当主机路径存在时才绑定挂载，否则由 Docker 在容器内自动创建目录
	// 注意：os.Stat 使用原生路径，mount Source/Target 使用 WSL 转换路径
	if needsMount {
		if _, statErr := os.Stat(nativeWorkdir); statErr == nil {
			alreadyMounted := false
			for _, m := range mounts {
				if m.Target == containerWorkdir {
					alreadyMounted = true
					break
				}
			}
			if !alreadyMounted {
				mounts = append(mounts, mount.Mount{
					Type:     mount.TypeBind,
					Source:   nativeWorkdir,
					Target:   containerWorkdir,
					ReadOnly: false,
				})
			}
		}
	}

	// 后台命令模式启用 AutoRemove，交互模式不自动删除
	autoRemove := isRunCmdMode

	hostConfig := &container.HostConfig{
		PortBindings: portBindings,
		Mounts:       mounts,
		AutoRemove:   autoRemove,
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
//  1. 后台命令模式（--run）：直接执行指定命令，bash -c <command>
//  2. Docker-in-Docker + agent：启动 dockerd 后执行 agent 命令
//  3. Docker-in-Docker + bash：启动 dockerd 后进入 bash（加载 wrapper）
//  4. Agent 模式：直接执行 agent 命令
//  5. Bash 模式（默认）：启动 bash（加载 wrapper）
func buildCmd(params argsparser.RunParams, wrapperScript string) strslice.StrSlice {
	isWindows := params.Platform == platform.PlatformWindows

	// 后台命令模式：直接执行指定命令，Tty=false
	if params.RunCmd != "" {
		if isWindows {
			return strslice.StrSlice{"powershell", "-Command", params.RunCmd}
		}
		return strslice.StrSlice{"bash", "-c", params.RunCmd}
	}

	if params.Docker {
		// Docker-in-Docker 模式（仅 Linux 支持）
		var finalCmd string
		if params.Agent != "" {
			finalCmd = params.Agent
		} else {
			finalCmd = `eval "$AGENTFORGE_WRAPPER"; exec bash`
		}
		return strslice.StrSlice{"bash", "-c", fmt.Sprintf(dockerdStartScript, finalCmd)}
	}

	if params.Agent != "" {
		// Agent 模式：Cmd 为 agent 可执行文件
		return strslice.StrSlice{params.Agent}
	}

	// Bash/PowerShell 模式：加载 wrapper 后启动 shell
	if wrapperScript != "" {
		if isWindows {
			return strslice.StrSlice{"powershell", "-Command", fmt.Sprintf(`$env:AGENTFORGE_WRAPPER; Invoke-Expression $env:AGENTFORGE_WRAPPER; powershell`)}
		}
		return strslice.StrSlice{"bash", "-c", `eval "$AGENTFORGE_WRAPPER"; exec bash`}
	}

	if isWindows {
		return strslice.StrSlice{"powershell"}
	}
	return strslice.StrSlice{"bash"}
}

// Run 执行容器运行流程。
//
// 流程：
//  1. 如果是 recall 模式（-r/--recall），从 Args Persistence 读取 .last_args 恢复参数
//  2. 如果为 bash 模式，从 Wrapper Loader 生成 wrapper 函数脚本
//  3. 调用 AssembleContainerConfig 组装配置
//  4. 调用 Docker Helper 的 ContainerCreate 创建容器
//  5. 调用 Docker Helper 的 ContainerStart 启动容器
//
// 在 agent 模式和 bash 模式下，附加到容器流（Tty 交互式）。
// 在后台命令模式（--run）下，通过 ContainerWait 等待容器退出，
// 传递退出码为 ExitCodeError。
//
// 每次成功运行后将参数持久化到 .last_args 文件（NFR-12），
// 但 recall 模式不会重复持久化已保存的参数。
func (e *Engine) Run(ctx context.Context, params argsparser.RunParams) error {
	// --- 步骤 1: 处理 recall 模式 ---
	if params.Recall {
		p := argspersistence.New(e.configDir)
		loaded, err := p.Load()
		if err != nil {
			if errors.Is(err, argspersistence.ErrFileNotFound) {
				return err // REQ-17: 文件不存在，不启动容器
			}
			return fmt.Errorf("读取 .last_args 失败: %w", err)
		}
		// 使用恢复的参数集，保留当前配置目录
		params = *loaded
		params.Recall = false
	}

	// --- 步骤 2: bash 模式下生成 wrapper 脚本 ---
	var wrapperScript string
	if params.Agent == "" && params.RunCmd == "" {
		wl := wrapperloader.New()
		wrapperScript = wl.Generate()
	}

	// --- 步骤 3: 组装容器配置 ---
	config, hostConfig, networkingConfig := AssembleContainerConfig(params, wrapperScript)

	// Windows 平台不支持 DIND 模式
	if params.Platform == platform.PlatformWindows && params.Docker {
		logging.Warn("Windows 容器不支持 Docker-in-Docker 模式，已忽略 --docker 参数")
		params.Docker = false
	}

	// --- 步骤 4: 创建容器 ---
	var createPlatform *specs.Platform
	if params.Platform == platform.PlatformWindows {
		createPlatform = &specs.Platform{OS: "windows", Architecture: "amd64"}
	}
	resp, err := e.helper.ContainerCreate(ctx, config, hostConfig, networkingConfig, createPlatform, "")
	if err != nil {
		return fmt.Errorf("创建容器失败: %w\n建议: 请先执行 agent-forge build 命令构建镜像，或确认镜像名称和 Docker daemon 运行状态", err)
	}

	containerID := resp.ID
	logging.Info("容器已创建", "container_id", containerID[:12], "agent", params.Agent)

	// --- 步骤 5: 启动容器 ---
	if err := e.helper.ContainerStart(ctx, containerID, types.ContainerStartOptions{}); err != nil {
		return fmt.Errorf("启动容器失败: %w\n建议: 请检查端口是否已被占用，以及 Docker daemon 是否有足够的资源创建新容器", err)
	}

	logging.Info("容器已启动", "container_id", containerID[:12])

	// --- 步骤 6: 后台命令模式（--run）---
	if params.RunCmd != "" {
		// 持久化参数（仅非 recall 模式）
		p := argspersistence.New(e.configDir)
		if saveErr := p.Save(params); saveErr != nil {
			return fmt.Errorf("持久化运行参数失败: %w\n建议: 请检查配置目录的写入权限", saveErr)
		}

		// 等待容器退出
		statusCh, errCh := e.helper.ContainerWait(ctx, containerID, container.WaitConditionNotRunning)
		select {
		case status := <-statusCh:
			if status.StatusCode != 0 {
				return &ExitCodeError{Code: int(status.StatusCode)}
			}
			return nil
		case err := <-errCh:
			return fmt.Errorf("等待容器退出失败: %w\n建议: 请检查 Docker daemon 运行状态", err)
		case <-ctx.Done():
			return fmt.Errorf("等待容器退出被中断: %w", ctx.Err())
		}
	}

	// --- 步骤 7: 交互模式 — 附加到容器并建立双向流拷贝 ---
	logging.Debug("进入交互模式，附加到容器", "container_id", containerID[:12])
	attachResp, err := e.helper.ContainerAttach(ctx, containerID, types.ContainerAttachOptions{
		Stream: true,
		Stdin:  true,
		Stdout: true,
		Stderr: true,
	})
	if err != nil {
		return fmt.Errorf("附加到容器失败: %w\n建议: 请检查终端 TTY 配置和 Docker daemon 状态", err)
	}
	defer attachResp.Close()

	// 同步终端尺寸到容器 PTY，解决启动后 bash 提示符不立即可见的问题
	if isTerm := term.IsTerminal(os.Stdin.Fd()); isTerm {
		if ws, err := term.GetWinsize(os.Stdin.Fd()); err == nil {
			if resizeErr := e.helper.ContainerResize(ctx, containerID, uint(ws.Height), uint(ws.Width)); resizeErr != nil {
				logging.Error("同步终端尺寸到容器失败，继续运行", "error", resizeErr)
			}
		}
	}

	// --- 步骤 8: 终端原始模式（支持 Ctrl+C 等控制字符传递）---
	stdinFd := os.Stdin.Fd()
	isTerminal := term.IsTerminal(stdinFd)
	var oldState *term.State
	if isTerminal {
		state, err := term.SetRawTerminal(stdinFd)
		if err != nil {
			return fmt.Errorf("设置终端原始模式失败: %w", err)
		}
		defer term.RestoreTerminal(stdinFd, state)
		oldState = state
	}

	// --- 步骤 9: 双向流拷贝 ---
	streamErr := make(chan error, 2)

	// TTY 模式下 Docker 不进行多路复用，直接拷贝
	// stdout/stderr: 容器输出 → 用户终端
	go func() {
		_, err := io.Copy(os.Stdout, attachResp.Reader)
		streamErr <- err
	}()

	// stdin: 用户终端（或管道输入） → 容器
	go func() {
		_, err := io.Copy(attachResp.Conn, os.Stdin)
		if err != nil && !errors.Is(err, io.EOF) {
			streamErr <- err
		}
	}()

	// --- 步骤 10: 信号处理（Ctrl+C → 容器）---
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	// --- 步骤 11: 等待流结束或信号 ---
	select {
	case err := <-streamErr:
		if err != nil && !errors.Is(err, io.EOF) {
			// 在终端恢复后才打印错误，避免输出混乱
			if oldState != nil {
				term.RestoreTerminal(stdinFd, oldState)
			}
			return fmt.Errorf("容器流异常断开: %w", err)
		}
	case sig := <-sigCh:
		// 用户按下 Ctrl+C，转发信号到容器
		logging.Info("收到信号", "signal", sig.String(), "container_id", containerID[:12])
		if oldState != nil {
			term.RestoreTerminal(stdinFd, oldState)
		}

		if params.Platform == platform.PlatformWindows {
			// Windows 容器不支持 POSIX 信号，使用 ContainerStop
			stopCtx := context.Background()
			if stopErr := e.helper.ContainerStop(stopCtx, containerID, nil); stopErr != nil {
				logging.Error("停止 Windows 容器失败", "error", stopErr)
				return fmt.Errorf("停止 Windows 容器失败: %w", stopErr)
			}
			return fmt.Errorf("已停止 Windows 容器（Ctrl+C）")
		}

		// 将 Unix 信号转换为 Docker 可识别的信号名称
		sigName := signalToDocker(sig)
		killCtx := context.Background()
		if killErr := e.helper.ContainerKill(killCtx, containerID, sigName); killErr != nil {
			logging.Error("发送信号到容器失败", "error", killErr, "signal", sigName)
			return fmt.Errorf("发送 %s 信号到容器失败: %w", sigName, killErr)
		}
		return fmt.Errorf("已发送 %s 信号到容器", sigName)
	}

	// --- 步骤 12: 持久化运行参数（NFR-12）---
	p := argspersistence.New(e.configDir)
	if saveErr := p.Save(params); saveErr != nil {
		return fmt.Errorf("持久化运行参数失败: %w\n建议: 请检查配置目录的写入权限", saveErr)
	}

	return nil
}

// hasEnvKey 检查环境变量列表中是否已包含指定 key（忽略大小写）。
//
// env 列表中的每项格式为 "KEY=value" 或 "KEY="。
func hasEnvKey(env []string, key string) bool {
	prefix := strings.ToUpper(key) + "="
	for _, e := range env {
		if strings.HasPrefix(strings.ToUpper(e), prefix) {
			return true
		}
	}
	return false
}

// toContainerPath 将宿主机原生路径转换为容器内兼容路径。
//
// Linux 平台：转换为 WSL 风格的 /mnt/<drive>/ 路径（与 Docker Desktop WSL2 后端一致）。
// Windows 平台：转换为 Windows 容器内的 C:\... 路径。
func toContainerPath(nativePath string, plt string) string {
	if plt == platform.PlatformWindows {
		return toWindowsContainerPath(nativePath)
	}
	return toLinuxContainerPath(nativePath)
}

// toLinuxContainerPath 将宿主机原生路径转换为 WSL 风格的 Linux 兼容路径。
//
//   - C:\Users\me → /mnt/c/Users/me
//   - D:\code\AgentForge → /mnt/d/code/AgentForge
//   - \\server\share\foo → /mnt/server/share/foo  (UNC)
//   - /workspace → /workspace  (已是 Linux 路径，原样返回)
func toLinuxContainerPath(nativePath string) string {
	vol := filepath.VolumeName(nativePath)
	path := filepath.ToSlash(nativePath[len(vol):])

	if vol == "" {
		return path
	}

	drive := strings.TrimRight(vol, ":")
	drive = strings.ToLower(drive)
	drive = strings.ReplaceAll(drive, "\\", "/")
	drive = strings.TrimPrefix(drive, "//")

	return "/mnt/" + drive + path
}

// toWindowsContainerPath 将 WSL/Linux 路径转换为 Windows 容器内原生路径。
//
//   - /mnt/d/code/AgentForge → D:\code\AgentForge
//   - /home/user/project → C:\home\user\project
//   - /workspace → C:\workspace
func toWindowsContainerPath(nativePath string) string {
	path := filepath.ToSlash(nativePath)

	// WSL 路径 /mnt/<drive>/... → <drive>:\...
	if strings.HasPrefix(path, "/mnt/") {
		rest := path[5:] // skip "/mnt/"
		if len(rest) >= 1 {
			drive := string(rest[0])
			restPath := rest[1:]
			return strings.ToUpper(drive) + ":" + strings.ReplaceAll(restPath, "/", "\\")
		}
	}

	// 普通 Linux 路径 → C:\... (作为默认)
	return "C:" + strings.ReplaceAll(path, "/", "\\")
}

// signalToDocker 将 os.Signal 转换为 Docker API 可识别的信号名称。
//
// Docker Engine API 的 ContainerKill 接收 Unix 信号名称（如 "SIGINT"、"SIGTERM"、"SIGKILL"）。
func signalToDocker(sig os.Signal) string {
	switch sig {
	case syscall.SIGINT:
		return "SIGINT"
	case syscall.SIGTERM:
		return "SIGTERM"
	case syscall.SIGKILL:
		return "SIGKILL"
	case syscall.SIGHUP:
		return "SIGHUP"
	default:
		return "SIGKILL"
	}
}

