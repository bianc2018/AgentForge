package runengine

import (
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

	config, hostConfig, _ := AssembleContainerConfig(params)

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

// TestAssembleContainerConfig_BashMode 验证 bash 模式的配置组装。
//
// 对应 UT-17 覆盖案例：bash 模式 — Cmd 设置为 bash，Tty=true
func TestAssembleContainerConfig_BashMode(t *testing.T) {
	params := argsparser.RunParams{
		Agent: "", // 空 Agent 表示 bash 模式
	}

	config, _, _ := AssembleContainerConfig(params)

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

// TestAssembleContainerConfig_PortMapping 验证端口映射配置。
//
// 对应 UT-17 覆盖案例：端口映射 — PortBindings 正确转换 `-p 3000:3000`
func TestAssembleContainerConfig_PortMapping(t *testing.T) {
	params := argsparser.RunParams{
		Ports: []string{"3000:3000", "8080:80"},
	}

	_, hostConfig, _ := AssembleContainerConfig(params)

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

	_, hostConfig, _ := AssembleContainerConfig(params)

	if len(hostConfig.Mounts) != 1 {
		t.Fatalf("len(Mounts) = %d, want 1", len(hostConfig.Mounts))
	}

	m := hostConfig.Mounts[0]
	if m.Type != mount.TypeBind {
		t.Errorf("Mount.Type = %q, want %q", m.Type, mount.TypeBind)
	}
	if m.Source != "/host/data" {
		t.Errorf("Mount.Source = %q, want %q", m.Source, "/host/data")
	}
	if m.Target != "/host/data" {
		t.Errorf("Mount.Target = %q, want %q", m.Target, "/host/data")
	}
	if !m.ReadOnly {
		t.Error("Mount.ReadOnly = false, want true")
	}
}

// TestAssembleContainerConfig_EnvironmentVariables 验证环境变量配置。
//
// 对应 UT-17 覆盖案例：环境变量 — Env 数组包含所有 -e 指定的键值对
func TestAssembleContainerConfig_EnvironmentVariables(t *testing.T) {
	params := argsparser.RunParams{
		Envs: []string{"OPENAI_KEY=sk-xxx", "DEBUG=true"},
	}

	config, _, _ := AssembleContainerConfig(params)

	if len(config.Env) != 2 {
		t.Fatalf("len(Env) = %d, want 2", len(config.Env))
	}
	if config.Env[0] != "OPENAI_KEY=sk-xxx" {
		t.Errorf("Env[0] = %q, want %q", config.Env[0], "OPENAI_KEY=sk-xxx")
	}
	if config.Env[1] != "DEBUG=true" {
		t.Errorf("Env[1] = %q, want %q", config.Env[1], "DEBUG=true")
	}
}

// TestAssembleContainerConfig_WorkingDirectory 验证工作目录配置。
//
// 对应 UT-17 覆盖案例：工作目录 — WorkingDir 指定为 -w 参数值
func TestAssembleContainerConfig_WorkingDirectory(t *testing.T) {
	params := argsparser.RunParams{
		Workdir: "/workspace",
	}

	config, _, _ := AssembleContainerConfig(params)

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

	config, hostConfig, _ := AssembleContainerConfig(params)

	if len(hostConfig.PortBindings) != 3 {
		t.Errorf("len(PortBindings) = %d, want 3", len(hostConfig.PortBindings))
	}
	if len(hostConfig.Mounts) != 3 {
		t.Errorf("len(Mounts) = %d, want 3", len(hostConfig.Mounts))
	}
	if len(config.ExposedPorts) != 3 {
		t.Errorf("len(ExposedPorts) = %d, want 3", len(config.ExposedPorts))
	}

	// 验证所有挂载均为只读
	for i, m := range hostConfig.Mounts {
		if !m.ReadOnly {
			t.Errorf("Mount[%d].ReadOnly = false, want true", i)
		}
		if m.Type != mount.TypeBind {
			t.Errorf("Mount[%d].Type = %q, want %q", i, m.Type, mount.TypeBind)
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

	config, _, _ := AssembleContainerConfig(params)

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

	config, _, _ := AssembleContainerConfig(params)

	if config.Image != ImageName {
		t.Errorf("Image = %q, want %q", config.Image, ImageName)
	}
}

// TestAssembleContainerConfig_EmptyParams 验证空参数时的默认值。
//
// 确保空参数下 Cmd 为 ["bash"]，且不会崩溃。
func TestAssembleContainerConfig_EmptyParams(t *testing.T) {
	params := argsparser.RunParams{}

	config, hostConfig, _ := AssembleContainerConfig(params)

	// 空参数应为 bash 模式
	if len(config.Cmd) != 1 || config.Cmd[0] != "bash" {
		t.Errorf("Cmd = %v, want [bash]", config.Cmd)
	}
	if len(config.Env) != 0 {
		t.Errorf("Env = %v, want empty", config.Env)
	}
	if config.WorkingDir != "" {
		t.Errorf("WorkingDir = %q, want empty", config.WorkingDir)
	}
	if len(hostConfig.Mounts) != 0 {
		t.Errorf("Mounts = %v, want empty", hostConfig.Mounts)
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

	config, _, _ := AssembleContainerConfig(params)

	// 检查 Cmd 是否为 strslice.StrSlice
	if _, ok := any(config.Cmd).(strslice.StrSlice); !ok {
		t.Errorf("Cmd type = %T, want strslice.StrSlice", config.Cmd)
	}
}

// TestNew 验证 Engine 构造函数。
func TestNew(t *testing.T) {
	// New 需要 Docker Helper 客户端，跳过实际创建
	// 此处仅验证 New 函数签名正确
	_ = New(nil)
}
