package runengine

import (
	"strings"
	"testing"

	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/strslice"
	"github.com/docker/go-connections/nat"

	"github.com/agent-forge/cli/internal/shared/argsparser"
)

// TestAssembleContainerConfig_AgentMode 验证 agent 模式的配置组装。
//
// 对应 UT-17 覆盖案例：agent 模式 — Cmd 设置为 agent 命令，Tty=true, OpenStdin=true
func TestAssembleContainerConfig_AgentMode(t *testing.T) {
	params := argsparser.RunParams{
		Agent: "claude",
	}

	config, hostConfig, _ := AssembleContainerConfig(params, "")

	if config.Image != ImageName {
		t.Errorf("Image = %q, want %q", config.Image, ImageName)
	}
	if !config.Tty {
		t.Error("Tty = false, want true")
	}
	if !config.OpenStdin {
		t.Error("OpenStdin = false, want true")
	}
	if !config.AttachStdin {
		t.Error("AttachStdin = false, want true")
	}
	if !config.AttachStdout {
		t.Error("AttachStdout = false, want true")
	}
	if !config.AttachStderr {
		t.Error("AttachStderr = false, want true")
	}
	if len(config.Cmd) != 1 || config.Cmd[0] != "claude" {
		t.Errorf("Cmd = %v, want [claude]", config.Cmd)
	}
	if hostConfig.AutoRemove {
		t.Error("AutoRemove = true, want false for interactive mode")
	}
}

// TestAssembleContainerConfig_BashMode 验证 bash 模式的配置组装（无 wrapper）。
//
// 对应 UT-17 覆盖案例：bash 模式 — Cmd 设置为 bash，Tty=true
func TestAssembleContainerConfig_BashMode(t *testing.T) {
	params := argsparser.RunParams{
		Agent: "", // 空 Agent 表示 bash 模式
	}

	config, _, _ := AssembleContainerConfig(params, "")

	if !config.Tty {
		t.Error("Tty = false, want true")
	}
	if !config.OpenStdin {
		t.Error("OpenStdin = false, want true")
	}
	if len(config.Cmd) != 1 || config.Cmd[0] != "bash" {
		t.Errorf("Cmd = %v, want [bash]", config.Cmd)
	}
}

// TestAssembleContainerConfig_BashModeWithWrapper 验证 bash 模式下注入 wrapper 脚本。
//
// 对应 UT-17 覆盖案例：bash 模式 — Cmd 包含加载 wrapper 的命令，env 包含 wrapper 脚本
func TestAssembleContainerConfig_BashModeWithWrapper(t *testing.T) {
	wrapperScript := "claude() { command claude \"$@\"; }\nopencode() { command opencode \"$@\"; }"
	params := argsparser.RunParams{
		Agent: "", // bash 模式
	}

	config, _, _ := AssembleContainerConfig(params, wrapperScript)

	// 验证 Tty 已启用
	if !config.Tty {
		t.Error("Tty = false, want true")
	}
	if !config.OpenStdin {
		t.Error("OpenStdin = false, want true")
	}

	// 验证 Cmd 为 ["bash", "-c", "<script>"]
	if len(config.Cmd) != 3 || config.Cmd[0] != "bash" || config.Cmd[1] != "-c" {
		t.Fatalf("Cmd = %v, want [bash -c <script>]", config.Cmd)
	}
	cmdStr := config.Cmd[2]
	if !strings.Contains(cmdStr, "AGENTFORGE_WRAPPER") {
		t.Errorf("Cmd should reference AGENTFORGE_WRAPPER env var, got: %s", cmdStr)
	}
	if !strings.Contains(cmdStr, "exec bash") {
		t.Errorf("Cmd should exec bash after sourcing wrapper, got: %s", cmdStr)
	}

	// 验证 AGENTFORGE_WRAPPER 环境变量包含 wrapper 脚本
	foundWrapper := false
	for _, e := range config.Env {
		if strings.HasPrefix(e, "AGENTFORGE_WRAPPER=") {
			foundWrapper = true
			if !strings.Contains(e, "claude") || !strings.Contains(e, "opencode") {
				t.Errorf("AGENTFORGE_WRAPPER should contain wrapper functions, got: %s", e)
			}
			break
		}
	}
	if !foundWrapper {
		t.Error("Environment should contain AGENTFORGE_WRAPPER with wrapper script")
	}
}

// TestAssembleContainerConfig_DockerInDockerMode 验证 Docker-in-Docker 特权模式配置。
//
// 对应 UT-17 覆盖案例：Docker-in-Docker 模式 — Privileged=true, User="root"，dockerd 自动启动
func TestAssembleContainerConfig_DockerInDockerMode(t *testing.T) {
	params := argsparser.RunParams{
		Docker: true,
		Agent:  "", // bash 模式
	}

	config, hostConfig, _ := AssembleContainerConfig(params, "claude() { :; }")

	// 验证 User 为 root
	if config.User != "root" {
		t.Errorf("User = %q, want %q", config.User, "root")
	}

	// 验证 Privileged
	if !hostConfig.Privileged {
		t.Error("Privileged = false, want true for DIND mode")
	}

	// 验证 AutoRemove 仍然是 false（交互模式）
	if hostConfig.AutoRemove {
		t.Error("AutoRemove = true, want false for interactive DIND mode")
	}

	// 验证 Cmd 包含 dockerd 启动脚本
	if len(config.Cmd) != 3 || config.Cmd[0] != "bash" || config.Cmd[1] != "-c" {
		t.Fatalf("Cmd = %v, want [bash -c <dockerd_start_script>]", config.Cmd)
	}
	cmdStr := config.Cmd[2]
	if !strings.Contains(cmdStr, "dockerd") {
		t.Errorf("Cmd should start dockerd, got: %s", cmdStr)
	}
	if !strings.Contains(cmdStr, "docker info") {
		t.Errorf("Cmd should wait for dockerd readiness, got: %s", cmdStr)
	}

	// 验证 AGENTFORGE_WRAPPER 环境变量在 DIND + bash 模式中仍然存在
	foundWrapper := false
	for _, e := range config.Env {
		if strings.HasPrefix(e, "AGENTFORGE_WRAPPER=") {
			foundWrapper = true
			break
		}
	}
	if !foundWrapper {
		t.Error("DIND + bash mode should have AGENTFORGE_WRAPPER env var")
	}
}

// TestAssembleContainerConfig_DockerInDockerWithAgent 验证 Docker-in-Docker + agent 模式。
//
// 验证 agent 模式下 DIND 的正确行为：dockerd 启动后执行 agent 命令。
func TestAssembleContainerConfig_DockerInDockerWithAgent(t *testing.T) {
	params := argsparser.RunParams{
		Docker: true,
		Agent:  "claude",
	}

	config, hostConfig, _ := AssembleContainerConfig(params, "")

	// 验证 User 为 root
	if config.User != "root" {
		t.Errorf("User = %q, want %q", config.User, "root")
	}

	// 验证 Privileged
	if !hostConfig.Privileged {
		t.Error("Privileged = false, want true for DIND mode")
	}

	// 验证 dockerd 启动后执行 agent 命令
	if len(config.Cmd) != 3 || config.Cmd[0] != "bash" || config.Cmd[1] != "-c" {
		t.Fatalf("Cmd = %v, want [bash -c <script>]", config.Cmd)
	}
	cmdStr := config.Cmd[2]
	if !strings.Contains(cmdStr, "dockerd") {
		t.Errorf("Cmd should start dockerd, got: %s", cmdStr)
	}
	if !strings.Contains(cmdStr, "claude") {
		t.Errorf("Cmd should execute agent after dockerd ready, got: %s", cmdStr)
	}
	if !strings.Contains(cmdStr, "exec") {
		t.Errorf("Cmd should use exec to replace shell with agent, got: %s", cmdStr)
	}
}

// TestAssembleContainerConfig_NoUnnecessaryDockerPrivilege 验证默认模式无特权。
//
// 对应 NFR-7: 仅显式传入 --docker/--dind 时启用特权模式
func TestAssembleContainerConfig_NoUnnecessaryDockerPrivilege(t *testing.T) {
	t.Run("run without any flags", func(t *testing.T) {
		params := argsparser.RunParams{}
		_, hostConfig, _ := AssembleContainerConfig(params, "")
		if hostConfig.Privileged {
			t.Error("Privileged = true, want false for default run")
		}
	})

	t.Run("run with agent only", func(t *testing.T) {
		params := argsparser.RunParams{Agent: "claude"}
		config, hostConfig, _ := AssembleContainerConfig(params, "")
		if hostConfig.Privileged {
			t.Error("Privileged = true, want false for agent mode without --docker")
		}
		if config.User == "root" {
			t.Error("User = root, want non-root for agent mode without --docker")
		}
	})

	t.Run("run with ports and mounts", func(t *testing.T) {
		params := argsparser.RunParams{
			Ports:  []string{"3000:3000"},
			Mounts: []string{"/data"},
		}
		_, hostConfig, _ := AssembleContainerConfig(params, "")
		if hostConfig.Privileged {
			t.Error("Privileged = true, want false even with ports/mounts without --docker")
		}
	})
}

// TestAssembleContainerConfig_DockerUserNotRoot 验证非 DIND 模式下 User 不是 root。
func TestAssembleContainerConfig_DockerUserNotRoot(t *testing.T) {
	params := argsparser.RunParams{
		Agent: "opencode",
	}
	config, _, _ := AssembleContainerConfig(params, "")

	if config.User == "root" {
		t.Error("User = root, want empty (non-root) for non-DIND mode")
	}
}

// TestAssembleContainerConfig_PortMapping 验证端口映射配置。
//
// 对应 UT-17 覆盖案例：端口映射 — PortBindings 正确转换 `-p 3000:3000`
func TestAssembleContainerConfig_PortMapping(t *testing.T) {
	params := argsparser.RunParams{
		Ports: []string{"3000:3000", "8080:80"},
	}

	_, hostConfig, _ := AssembleContainerConfig(params, "")

	// 验证 PortBindings
	if len(hostConfig.PortBindings) != 2 {
		t.Fatalf("len(PortBindings) = %d, want 2", len(hostConfig.PortBindings))
	}

	// 检查端口 3000/tcp
	port3000, err := nat.NewPort("tcp", "3000")
	if err != nil {
		t.Fatalf("NewPort failed: %v", err)
	}
	bindings, ok := hostConfig.PortBindings[port3000]
	if !ok {
		t.Errorf("PortBindings missing %q", port3000)
	} else if len(bindings) != 1 || bindings[0].HostPort != "3000" {
		t.Errorf("PortBindings[%q] = %v, want [{3000}]", port3000, bindings)
	}

	// 检查端口 80/tcp
	port80, err := nat.NewPort("tcp", "80")
	if err != nil {
		t.Fatalf("NewPort failed: %v", err)
	}
	bindings, ok = hostConfig.PortBindings[port80]
	if !ok {
		t.Errorf("PortBindings missing %q", port80)
	} else if len(bindings) != 1 || bindings[0].HostPort != "8080" {
		t.Errorf("PortBindings[%q] = %v, want [{8080}]", port80, bindings)
	}
}

// TestAssembleContainerConfig_ReadOnlyMount 验证只读目录挂载配置。
//
// 对应 UT-17 覆盖案例：只读挂载 — Mounts 正确设置为只读（NFR-8）
func TestAssembleContainerConfig_ReadOnlyMount(t *testing.T) {
	params := argsparser.RunParams{
		Mounts: []string{"/host/data"},
	}

	_, hostConfig, _ := AssembleContainerConfig(params, "")

	// 应包含 -m 只读挂载 + 工作目录自动挂载（来自 os.Getwd()）
	if len(hostConfig.Mounts) < 1 {
		t.Fatalf("len(Mounts) = %d, want >= 1", len(hostConfig.Mounts))
	}

	// 查找 -m 指定的只读挂载
	var roMount *mount.Mount
	for i := range hostConfig.Mounts {
		if hostConfig.Mounts[i].Source == "/host/data" && hostConfig.Mounts[i].Target == "/host/data" {
			roMount = &hostConfig.Mounts[i]
			break
		}
	}
	if roMount == nil {
		t.Fatal("-m /host/data mount not found in Mounts")
	}
	if roMount.Type != mount.TypeBind {
		t.Errorf("Mount.Type = %q, want %q", roMount.Type, mount.TypeBind)
	}
	if !roMount.ReadOnly {
		t.Error("Mount.ReadOnly = false, want true for -m mount")
	}
}

// TestAssembleContainerConfig_EnvironmentVariables 验证环境变量配置。
//
// 对应 UT-17 覆盖案例：环境变量 — Env 数组包含所有 -e 指定的键值对
func TestAssembleContainerConfig_EnvironmentVariables(t *testing.T) {
	params := argsparser.RunParams{
		Envs: []string{"OPENAI_KEY=sk-xxx", "DEBUG=true"},
	}

	config, _, _ := AssembleContainerConfig(params, "")

	// 应包含 2 个用户指定的 env + 自动注入的 TERM
	if len(config.Env) != 3 {
		t.Fatalf("len(Env) = %d, want 3 (2 user + TERM)", len(config.Env))
	}
	if config.Env[0] != "OPENAI_KEY=sk-xxx" {
		t.Errorf("Env[0] = %q, want %q", config.Env[0], "OPENAI_KEY=sk-xxx")
	}
	if config.Env[1] != "DEBUG=true" {
		t.Errorf("Env[1] = %q, want %q", config.Env[1], "DEBUG=true")
	}
	// TERM 自动注入（用户未显式设置）
	if config.Env[2] != "TERM=xterm-256color" {
		t.Errorf("Env[2] = %q, want %q", config.Env[2], "TERM=xterm-256color")
	}
}

// TestAssembleContainerConfig_WorkingDirectory 验证工作目录配置。
//
// 对应 UT-17 覆盖案例：工作目录 — WorkingDir 指定为 -w 参数值
func TestAssembleContainerConfig_WorkingDirectory(t *testing.T) {
	params := argsparser.RunParams{
		Workdir: "/workspace",
	}

	config, _, _ := AssembleContainerConfig(params, "")

	if config.WorkingDir != "/workspace" {
		t.Errorf("WorkingDir = %q, want %q", config.WorkingDir, "/workspace")
	}
}

// TestAssembleContainerConfig_MultiPortAndMount 验证多端口/多挂载配置。
//
// 对应 UT-17 覆盖案例：多端口/多挂载 — 多个 -p 和 -m 参数全部包含
func TestAssembleContainerConfig_MultiPortAndMount(t *testing.T) {
	params := argsparser.RunParams{
		Ports:  []string{"3000:3000", "8080:80", "443:443"},
		Mounts: []string{"/data", "/config", "/logs"},
	}

	config, hostConfig, _ := AssembleContainerConfig(params, "")

	if len(hostConfig.PortBindings) != 3 {
		t.Errorf("len(PortBindings) = %d, want 3", len(hostConfig.PortBindings))
	}
	// 应包含 3 个 -m 挂载 + 工作目录自动挂载
	if len(hostConfig.Mounts) < 3 {
		t.Errorf("len(Mounts) = %d, want >= 3", len(hostConfig.Mounts))
	}
	if len(config.ExposedPorts) != 3 {
		t.Errorf("len(ExposedPorts) = %d, want 3", len(config.ExposedPorts))
	}

	// 验证 -m 指定的挂载均为只读
	roPaths := map[string]bool{"/data": true, "/config": true, "/logs": true}
	for _, m := range hostConfig.Mounts {
		if roPaths[m.Source] {
			if !m.ReadOnly {
				t.Errorf("Mount %q.ReadOnly = false, want true for -m mount", m.Source)
			}
		}
		if m.Type != mount.TypeBind {
			t.Errorf("Mount %q.Type = %q, want %q", m.Source, m.Type, mount.TypeBind)
		}
	}
}

// TestAssembleContainerConfig_ExposedPorts 验证 ExposedPorts 存在。
//
// 确保声明端口映射时 ExposedPorts 也被正确设置。
func TestAssembleContainerConfig_ExposedPorts(t *testing.T) {
	params := argsparser.RunParams{
		Ports: []string{"3000:3000"},
	}

	config, _, _ := AssembleContainerConfig(params, "")

	portKey, err := nat.NewPort("tcp", "3000")
	if err != nil {
		t.Fatalf("NewPort failed: %v", err)
	}
	if _, ok := config.ExposedPorts[portKey]; !ok {
		t.Errorf("ExposedPorts missing %q", portKey)
	}
}

// TestAssembleContainerConfig_DefaultImage 验证默认镜像名称。
func TestAssembleContainerConfig_DefaultImage(t *testing.T) {
	params := argsparser.RunParams{}

	config, _, _ := AssembleContainerConfig(params, "")

	if config.Image != ImageName {
		t.Errorf("Image = %q, want %q", config.Image, ImageName)
	}
}

// TestAssembleContainerConfig_EmptyParams 验证空参数时的默认值。
//
// 确保空参数下 Cmd 为 ["bash"]，且不会崩溃。
func TestAssembleContainerConfig_EmptyParams(t *testing.T) {
	params := argsparser.RunParams{}

	config, hostConfig, _ := AssembleContainerConfig(params, "")

	// 空参数应为 bash 模式
	if len(config.Cmd) != 1 || config.Cmd[0] != "bash" {
		t.Errorf("Cmd = %v, want [bash]", config.Cmd)
	}
	// 空参数时自动注入 TERM，自动检测工作目录和挂载
	if len(config.Env) != 1 || config.Env[0] != "TERM=xterm-256color" {
		t.Errorf("Env = %v, want [TERM=xterm-256color]", config.Env)
	}
	if config.WorkingDir == "" {
		t.Error("WorkingDir should not be empty (auto-detected from os.Getwd())")
	}
	// 工作目录来自 os.Getwd()，应至少有一个 rw 挂载
	if len(hostConfig.Mounts) < 1 {
		t.Error("Mounts should have at least the workdir auto-mount")
	}
	if len(hostConfig.PortBindings) != 0 {
		t.Errorf("PortBindings = %v, want empty", hostConfig.PortBindings)
	}
}

// TestAssembleContainerConfig_CmdType 验证 Cmd 类型为 strslice.StrSlice。
//
// Docker SDK v20.10 要求 Cmd 为 strslice.StrSlice 类型，确保类型正确。
func TestAssembleContainerConfig_CmdType(t *testing.T) {
	params := argsparser.RunParams{
		Agent: "opencode",
	}

	config, _, _ := AssembleContainerConfig(params, "")

	// 检查 Cmd 是否为 strslice.StrSlice
	if _, ok := any(config.Cmd).(strslice.StrSlice); !ok {
		t.Errorf("Cmd type = %T, want strslice.StrSlice", config.Cmd)
	}
}

// TestNew 验证 Engine 构造函数。
func TestNew(t *testing.T) {
	// New 需要 Docker Helper 客户端和配置目录，跳过实际创建
	// 此处仅验证 New 函数签名正确
	_ = New(nil, "")
}

// TestAssembleContainerConfig_RunCommandMode 验证后台命令模式的配置组装。
//
// 对应 UT-17 覆盖案例：后台命令模式 — AutoRemove=true, Tty=false,
// Cmd 为 ["bash", "-c", "命令"], 无 wrapper 环境变量
func TestAssembleContainerConfig_RunCommandMode(t *testing.T) {
	params := argsparser.RunParams{
		RunCmd: "npm test",
	}

	config, hostConfig, _ := AssembleContainerConfig(params, "")

	// 验证 AutoRemove 已启用
	if !hostConfig.AutoRemove {
		t.Error("AutoRemove = false, want true for run command mode")
	}

	// 验证 Tty 已关闭
	if config.Tty {
		t.Error("Tty = true, want false for run command mode")
	}

	// 验证 OpenStdin 已关闭
	if config.OpenStdin {
		t.Error("OpenStdin = true, want false for run command mode")
	}

	// 验证 AttachStdin 已关闭
	if config.AttachStdin {
		t.Error("AttachStdin = true, want false for run command mode")
	}

	// 验证 AttachStdout 和 AttachStderr 仍打开
	if !config.AttachStdout {
		t.Error("AttachStdout = false, want true")
	}
	if !config.AttachStderr {
		t.Error("AttachStderr = false, want true")
	}

	// 验证 Cmd 为 ["bash", "-c", "npm test"]
	if len(config.Cmd) != 3 {
		t.Fatalf("len(Cmd) = %d, want 3", len(config.Cmd))
	}
	if config.Cmd[0] != "bash" {
		t.Errorf("Cmd[0] = %q, want %q", config.Cmd[0], "bash")
	}
	if config.Cmd[1] != "-c" {
		t.Errorf("Cmd[1] = %q, want %q", config.Cmd[1], "-c")
	}
	if config.Cmd[2] != "npm test" {
		t.Errorf("Cmd[2] = %q, want %q", config.Cmd[2], "npm test")
	}

	// 验证无 AGENTFORGE_WRAPPER 环境变量
	for _, e := range config.Env {
		if strings.HasPrefix(e, "AGENTFORGE_WRAPPER=") {
			t.Errorf("Run command mode should not have AGENTFORGE_WRAPPER, got: %s", e)
		}
	}
}

// TestAssembleContainerConfig_RunCommandModeWithDocker 验证后台命令模式 + Docker 模式。
//
// 验证 --run 与 --docker 同时指定时：Cmd 为后台命令，Privileged 仍启用。
func TestAssembleContainerConfig_RunCommandModeWithDocker(t *testing.T) {
	params := argsparser.RunParams{
		RunCmd: "docker ps",
		Docker: true,
	}

	config, hostConfig, _ := AssembleContainerConfig(params, "")

	// 验证 AutoRemove 已启用
	if !hostConfig.AutoRemove {
		t.Error("AutoRemove = false, want true for run command mode")
	}

	// 验证 Tty 已关闭
	if config.Tty {
		t.Error("Tty = true, want false for run command mode")
	}

	// 验证 Privileged 仍然可用（--docker 标志）
	if !hostConfig.Privileged {
		t.Error("Privileged = false, want true when --docker is specified")
	}

	// 验证 User 为 root
	if config.User != "root" {
		t.Errorf("User = %q, want %q", config.User, "root")
	}

	// 验证 Cmd 为 ["bash", "-c", "docker ps"]
	if len(config.Cmd) != 3 || config.Cmd[0] != "bash" || config.Cmd[1] != "-c" || config.Cmd[2] != "docker ps" {
		t.Errorf("Cmd = %v, want [bash -c docker ps]", config.Cmd)
	}
}

// TestExitCodeError 验证 ExitCodeError 类型。
//
// 验证错误消息格式和 ExitCode 字段，以及 ExitCoder 接口的实现。
func TestExitCodeError(t *testing.T) {
	err := &ExitCodeError{Code: 42}
	if err.Code != 42 {
		t.Errorf("Code = %d, want 42", err.Code)
	}
	// 验证 ExitCoder 接口
	if ec := err.ExitCode(); ec != 42 {
		t.Errorf("ExitCode() = %d, want 42", ec)
	}
	errMsg := err.Error()
	if errMsg == "" {
		t.Error("Error() returned empty string")
	}
	if !strings.Contains(errMsg, "42") {
		t.Errorf("Error() = %q, should contain '42'", errMsg)
	}
}
