// Package runengine 提供纯单元测试，不依赖真实 Docker daemon。
//
// 本文件覆盖以下测试维度：
//   - buildCmd：所有 5 种启动模式的命令组装
//   - hasEnvKey：环境变量键查找（大小写不敏感、边界情况）
//   - signalToDocker：Unix 信号到 Docker 信号名称的映射
//   - AssembleContainerConfig：边缘场景（TERM 注入、端口异常、工作目录去重/不存在等）
//   - Engine.Run：通过 mock HTTP server 模拟 Docker API，测试完整容器生命周期流程
package runengine

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/strslice"
	"github.com/docker/docker/client"

	"github.com/agent-forge/cli/internal/run/argspersistence"
	"github.com/agent-forge/cli/internal/shared/argsparser"
	"github.com/agent-forge/cli/internal/shared/dockerhelper"
)

// ============================================================================
// buildCmd 单元测试
// ============================================================================

func TestBuildCmd_RunCommandMode(t *testing.T) {
	cmd := buildCmd(argsparser.RunParams{RunCmd: "npm test"}, "")
	expected := strslice.StrSlice{"bash", "-c", "npm test"}
	if !strSliceEqual(cmd, expected) {
		t.Errorf("buildCmd(RunCmd) = %v, want %v", cmd, expected)
	}
}

func TestBuildCmd_DockerWithAgent(t *testing.T) {
	cmd := buildCmd(argsparser.RunParams{Docker: true, Agent: "claude"}, "")
	if len(cmd) != 3 || cmd[0] != "bash" || cmd[1] != "-c" {
		t.Fatalf("buildCmd(Docker+Agent) = %v, want [bash -c <script>]", cmd)
	}
	script := cmd[2]
	if !strings.Contains(script, "dockerd") {
		t.Error("Docker+Agent: 脚本应启动 dockerd")
	}
	if !strings.Contains(script, "claude") {
		t.Error("Docker+Agent: 脚本应在 dockerd 就绪后执行 agent")
	}
	if !strings.Contains(script, "exec") {
		t.Error("Docker+Agent: 脚本应使用 exec 替换 shell")
	}
}

func TestBuildCmd_DockerWithBash(t *testing.T) {
	cmd := buildCmd(argsparser.RunParams{Docker: true, Agent: ""}, "wrapper_content")
	if len(cmd) != 3 || cmd[0] != "bash" || cmd[1] != "-c" {
		t.Fatalf("buildCmd(Docker+Bash) = %v, want [bash -c <script>]", cmd)
	}
	script := cmd[2]
	if !strings.Contains(script, "dockerd") {
		t.Error("Docker+Bash: 脚本应启动 dockerd")
	}
	if !strings.Contains(script, "AGENTFORGE_WRAPPER") {
		t.Error("Docker+Bash: 脚本应引用 AGENTFORGE_WRAPPER")
	}
	if !strings.Contains(script, "exec bash") {
		t.Error("Docker+Bash: 脚本应在 dockerd 就绪后 exec bash")
	}
}

func TestBuildCmd_AgentMode(t *testing.T) {
	cmd := buildCmd(argsparser.RunParams{Agent: "opencode"}, "")
	expected := strslice.StrSlice{"opencode"}
	if !strSliceEqual(cmd, expected) {
		t.Errorf("buildCmd(Agent) = %v, want %v", cmd, expected)
	}
}

func TestBuildCmd_BashWithWrapper(t *testing.T) {
	cmd := buildCmd(argsparser.RunParams{}, "wrapper_content")
	if len(cmd) != 3 || cmd[0] != "bash" || cmd[1] != "-c" {
		t.Fatalf("buildCmd(Bash+Wrapper) = %v, want [bash -c <script>]", cmd)
	}
	if !strings.Contains(cmd[2], "AGENTFORGE_WRAPPER") {
		t.Error("Bash+Wrapper: 脚本应引用 AGENTFORGE_WRAPPER")
	}
	if !strings.Contains(cmd[2], "exec bash") {
		t.Error("Bash+Wrapper: 脚本应在加载 wrapper 后 exec bash")
	}
}

func TestBuildCmd_BashDefault(t *testing.T) {
	cmd := buildCmd(argsparser.RunParams{}, "")
	expected := strslice.StrSlice{"bash"}
	if !strSliceEqual(cmd, expected) {
		t.Errorf("buildCmd(default) = %v, want %v", cmd, expected)
	}
}

// ============================================================================
// hasEnvKey 单元测试
// ============================================================================

func TestHasEnvKey(t *testing.T) {
	tests := []struct {
		name string
		env  []string
		key  string
		want bool
	}{
		{"exact match", []string{"TERM=xterm", "PATH=/usr/bin"}, "TERM", true},
		{"case insensitive key lookup", []string{"term=xterm"}, "TERM", true},
		{"case insensitive env key", []string{"TERM=xterm"}, "term", true},
		{"key not found", []string{"HOME=/root", "PATH=/usr/bin"}, "TERM", false},
		{"empty env list", []string{}, "TERM", false},
		{"prefix partial match", []string{"TERMINAL=foo"}, "TERM", false},
		{"empty value", []string{"TERM="}, "TERM", true},
		{"multiple envs with match", []string{"A=1", "TERM=xterm", "B=2"}, "TERM", true},
		{"case insensitive in value part", []string{"TERM=VT100"}, "TERM", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hasEnvKey(tt.env, tt.key)
			if got != tt.want {
				t.Errorf("hasEnvKey(%v, %q) = %v, want %v", tt.env, tt.key, got, tt.want)
			}
		})
	}
}

// ============================================================================
// signalToDocker 单元测试
// ============================================================================

func TestSignalToDocker(t *testing.T) {
	tests := []struct {
		name string
		sig  os.Signal
		want string
	}{
		{"SIGINT -> SIGINT", syscall.SIGINT, "SIGINT"},
		{"SIGTERM -> SIGTERM", syscall.SIGTERM, "SIGTERM"},
		{"SIGKILL -> SIGKILL", syscall.SIGKILL, "SIGKILL"},
		{"SIGHUP -> SIGHUP", syscall.SIGHUP, "SIGHUP"},
		{"unknown signal defaults to SIGKILL", syscall.SIGQUIT, "SIGKILL"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := signalToDocker(tt.sig)
			if got != tt.want {
				t.Errorf("signalToDocker(%v) = %q, want %q", tt.sig, got, tt.want)
			}
		})
	}
}

// ============================================================================
// AssembleContainerConfig 边缘场景
// ============================================================================

func TestAssembleContainerConfig_TermFromHostEnv(t *testing.T) {
	// 保存并宿主机 TERM，测试中使用自定义值
	savedTerm := os.Getenv("TERM")
	os.Setenv("TERM", "xterm-direct")
	defer os.Setenv("TERM", savedTerm)

	params := argsparser.RunParams{}
	config, _, _ := AssembleContainerConfig(params, "")

	found := false
	for _, e := range config.Env {
		if e == "TERM=xterm-direct" {
			found = true
			break
		}
	}
	if !found {
		t.Error("应使用宿主机的 TERM 环境变量值")
	}
}

func TestAssembleContainerConfig_UserSpecifiedTerm(t *testing.T) {
	// 用户显式指定 TERM，不被宿主 TERM 覆盖
	params := argsparser.RunParams{
		Envs: []string{"TERM=vt100"},
	}
	config, _, _ := AssembleContainerConfig(params, "")

	var termVals []string
	for _, e := range config.Env {
		if strings.HasPrefix(e, "TERM=") {
			termVals = append(termVals, e)
		}
	}
	if len(termVals) != 1 {
		t.Errorf("期望恰好 1 个 TERM 条目，得到 %d: %v", len(termVals), termVals)
	}
	if len(termVals) > 0 && termVals[0] != "TERM=vt100" {
		t.Errorf("期望 TERM=vt100，得到 %s", termVals[0])
	}
}

func TestAssembleContainerConfig_UserTermOverrideCaseInsensitive(t *testing.T) {
	// 用户用小写 key 指定 term，应被 hasEnvKey 大小写不敏感地识别
	params := argsparser.RunParams{
		Envs: []string{"term=ansi"},
	}
	config, _, _ := AssembleContainerConfig(params, "")

	count := 0
	for _, e := range config.Env {
		if strings.HasPrefix(strings.ToUpper(e), "TERM=") {
			count++
		}
	}
	if count != 1 {
		t.Errorf("期望恰好 1 个 TERM 条目（大小写不敏感），得到 %d", count)
	}
}

func TestAssembleContainerConfig_MalformedPorts(t *testing.T) {
	// 缺少冒号的端口格式应在 ExposedPorts 和 PortBindings 中被跳过
	// 注意：":3000" 和 "3000:" 会被 SplitN 解析为 2 段，因此会生成配置
	// 但 "invalid"（无冒号）只产生 1 段，应被跳过
	params := argsparser.RunParams{
		Ports: []string{"invalid", ":3000", "3000:", ""},
	}
	config, hostConfig, _ := AssembleContainerConfig(params, "")

	// "invalid" 无冒号 → 不生成绑定。但 ":3000" 和 "3000:" 各生成一个。
	// 所以总 PortBindings 应为 2（不是 0）
	if len(hostConfig.PortBindings) != 2 {
		t.Errorf("期望 2 个 PortBindings（:3000 和 3000:），得到 %d: %v",
			len(hostConfig.PortBindings), hostConfig.PortBindings)
	}
	if len(config.ExposedPorts) != 2 {
		t.Errorf("期望 2 个 ExposedPorts，得到 %d: %v",
			len(config.ExposedPorts), config.ExposedPorts)
	}

	// 验证空字符串（""）和 "invalid"（无冒号）的处理
	// SplitN("", ":") → [""] → len=1 → 跳过 ✓
	// SplitN("invalid", ":") → ["invalid"] → len=1 → 跳过 ✓
}

func TestAssembleContainerConfig_EmptyPorts(t *testing.T) {
	params := argsparser.RunParams{
		Ports: []string{},
	}
	_, hostConfig, _ := AssembleContainerConfig(params, "")
	if len(hostConfig.PortBindings) != 0 {
		t.Errorf("空端口列表应生成 0 个 PortBindings，得到 %d", len(hostConfig.PortBindings))
	}
}

func TestAssembleContainerConfig_WorkdirAlreadyMounted(t *testing.T) {
	// 当 -w 指定的工作目录已经通过 -m 挂载时，不应重复挂载（去重）
	existingDir, _ := os.Getwd()

	params := argsparser.RunParams{
		Workdir: existingDir,
		Mounts:  []string{existingDir},
	}
	_, hostConfig, _ := AssembleContainerConfig(params, "")

	count := 0
	for _, m := range hostConfig.Mounts {
		if m.Target == existingDir {
			count++
		}
	}
	if count != 1 {
		t.Errorf("工作目录已在 -m 中时不应重复挂载，得到 %d 个条目", count)
	}
}

func TestAssembleContainerConfig_WorkdirNotExist(t *testing.T) {
	// 当工作目录在宿主机上不存在时，不应自动挂载（os.Stat 失败）
	nonexistentPath := "/definitely-does-not-exist-agentforge-test-xyz-123"

	params := argsparser.RunParams{
		Workdir: nonexistentPath,
	}
	config, hostConfig, _ := AssembleContainerConfig(params, "")

	if config.WorkingDir != nonexistentPath {
		t.Errorf("WorkingDir = %q, want %q", config.WorkingDir, nonexistentPath)
	}
	for _, m := range hostConfig.Mounts {
		if m.Target == nonexistentPath {
			t.Error("不存在的宿主路径不应自动挂载")
		}
	}
}

func TestAssembleContainerConfig_NoWorkdirAutoDetect(t *testing.T) {
	// 未指定 -w 时，应自动检测工作目录
	params := argsparser.RunParams{}
	config, _, _ := AssembleContainerConfig(params, "")
	if config.WorkingDir == "" {
		t.Error("未指定 -w 时，WorkingDir 不应为空（应自动检测）")
	}
}

func TestAssembleContainerConfig_NoMountsOnlyWorkdir(t *testing.T) {
	// 未指定 -m 时，应至少包含工作目录的自动挂载
	params := argsparser.RunParams{}
	_, hostConfig, _ := AssembleContainerConfig(params, "")
	if len(hostConfig.Mounts) < 1 {
		t.Error("至少应包含工作目录自动挂载")
	}
}

func TestAssembleContainerConfig_OnlyUserMountsNoWorkdir(t *testing.T) {
	// 指定 -m 但未指定 -w 时：-m 挂载 + 自动检测的工作目录挂载
	wd, _ := os.Getwd()
	params := argsparser.RunParams{
		Mounts: []string{"/data1", "/data2"},
	}
	_, hostConfig, _ := AssembleContainerConfig(params, "")

	// 应包含 2 个用户挂载 + 工作目录自动挂载
	mountCount := len(hostConfig.Mounts)
	if mountCount < 3 {
		t.Errorf("期望至少 3 个挂载（2 个用户 + 工作目录），得到 %d", mountCount)
	}

	// 验证用户挂载为只读
	foundData1, foundData2 := false, false
	for _, m := range hostConfig.Mounts {
		if m.Source == "/data1" {
			foundData1 = true
			if !m.ReadOnly {
				t.Error("/data1 应为只读挂载")
			}
		}
		if m.Source == "/data2" {
			foundData2 = true
			if !m.ReadOnly {
				t.Error("/data2 应为只读挂载")
			}
		}
		// 验证工作目录挂载为可读写
		if m.Target == wd {
			if m.ReadOnly {
				t.Error("工作目录自动挂载应为可读写")
			}
		}
	}
	if !foundData1 {
		t.Error("未找到 /data1 挂载")
	}
	if !foundData2 {
		t.Error("未找到 /data2 挂载")
	}
}

func TestAssembleContainerConfig_RunCmdDisablesTty(t *testing.T) {
	// --run 模式下 Tty/OpenStdin/AttachStdin 均应为 false
	params := argsparser.RunParams{RunCmd: "echo test"}
	config, hostConfig, _ := AssembleContainerConfig(params, "")

	if config.Tty {
		t.Error("Tty 应为 false（RunCmd 模式）")
	}
	if config.OpenStdin {
		t.Error("OpenStdin 应为 false（RunCmd 模式）")
	}
	if config.AttachStdin {
		t.Error("AttachStdin 应为 false（RunCmd 模式）")
	}
	if !hostConfig.AutoRemove {
		t.Error("AutoRemove 应为 true（RunCmd 模式）")
	}
}

func TestAssembleContainerConfig_WorkdirOverridesAutoDetect(t *testing.T) {
	// -w 显式指定工作目录时，应使用指定值而非自动检测
	params := argsparser.RunParams{Workdir: "/my/custom/path"}
	config, _, _ := AssembleContainerConfig(params, "")

	if config.WorkingDir != "/my/custom/path" {
		t.Errorf("WorkingDir = %q, want %q", config.WorkingDir, "/my/custom/path")
	}
}

func TestAssembleContainerConfig_MultipleEnvs(t *testing.T) {
	// 多环境变量 + TERM 自动注入无冲突
	params := argsparser.RunParams{
		Envs: []string{"KEY1=val1", "KEY2=val2", "KEY3=val3"},
	}
	config, _, _ := AssembleContainerConfig(params, "")

	// 3 用户 + 1 TERM = 4
	if len(config.Env) != 4 {
		t.Errorf("Env 长度 = %d, 期望 4（3 用户 + TERM）", len(config.Env))
	}
}

// ============================================================================
// Engine.Run 单元测试 — recall 失败路径（无需 Docker）
// ============================================================================

func TestEngineRun_RecallFileNotFound(t *testing.T) {
	// recall 模式且 .last_args 不存在时，应在调用 Docker API 之前返回错误
	configDir := t.TempDir()
	engine := New(nil, configDir)

	params := argsparser.RunParams{Recall: true}
	err := engine.Run(context.Background(), params)

	if err == nil {
		t.Fatal("recall 模式无 .last_args 应返回错误")
	}
	if !errors.Is(err, argspersistence.ErrFileNotFound) {
		t.Errorf("期望 ErrFileNotFound，得到 %v", err)
	}

	// 确认 .last_args 文件不存在（即没有调用 ContainerCreate）
	lastArgsPath := filepath.Join(configDir, ".last_args")
	if _, statErr := os.Stat(lastArgsPath); !os.IsNotExist(statErr) {
		t.Error("不应创建 .last_args 文件（不应调用 Docker API）")
	}
}

// ============================================================================
// Engine.Run 单元测试 — mock Docker API 服务器（完整生命周期）
// ============================================================================

// mockDockerAPI 创建一个模拟 Docker Engine API 的 HTTP 测试服务器。
//
// 支持以下端点：
//   - GET /_ping → 200 + API-Version 头（SDK 版本协商）
//   - POST /containers/create → 201 + 容器 ID
//   - POST /containers/{id}/start → 204
//   - POST /containers/{id}/wait → 200 + {"StatusCode": exitCode}
//
// attachEndpoint 为 true 时注册 /attach 端点（需要 hijack）。
type mockDockerAPIConfig struct {
	exitCode int64 // ContainerWait 返回的退出码
}

func newMockDockerAPI(t *testing.T, cfg mockDockerAPIConfig) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		method := r.Method

		// 根据请求路径分发
		switch {
		case strings.HasSuffix(path, "/_ping"):
			// Docker 版本协商端点
			w.Header().Set("API-Version", "1.41")
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))

		case strings.HasSuffix(path, "/containers/create") && method == http.MethodPost:
			// 容器创建
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(container.ContainerCreateCreatedBody{
				ID: "mock-container-id",
			})

		case strings.Contains(path, "/containers/") && strings.HasSuffix(path, "/start") && method == http.MethodPost:
			// 容器启动
			w.WriteHeader(http.StatusNoContent)

		case strings.Contains(path, "/containers/") && strings.HasSuffix(path, "/wait") && method == http.MethodPost:
			// 容器等待 — 返回指定退出码
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(container.ContainerWaitOKBody{
				StatusCode: cfg.exitCode,
			})

		default:
			t.Logf("[mock-docker] 未处理请求: %s %s", method, path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

// newMockDockerClient 创建指向 mock Docker API 服务器的 dockerhelper.Client。
//
// 注意：使用 tcp:// 而非 http:// 作为主机 URL 的 scheme。
// Docker SDK 的 setupHijackConn 使用 cli.proto 作为 net.Dial 的网络类型参数，
// "http" 不是有效的 Go 网络类型，会导致「dial http: unknown network http」错误。
// 使用 tcp:// 后 cli.proto = "tcp"，同时 cli.scheme 仍保持默认值 "http"，
// 因此正常 HTTP 请求和 hijack 连接均能正确工作。
func newMockDockerClient(t *testing.T, serverURL string) *dockerhelper.Client {
	t.Helper()

	dockerHost := strings.Replace(serverURL, "http://", "tcp://", 1)
	helper, err := dockerhelper.NewClientWithOpts(
		client.WithHost(dockerHost),
	)
	if err != nil {
		t.Fatalf("NewClientWithOpts(mock) 失败: %v", err)
	}
	return helper
}

func TestEngineRun_RunCmdMode_ExitZero(t *testing.T) {
	// 后台命令模式，命令以退出码 0 正常退出
	ts := newMockDockerAPI(t, mockDockerAPIConfig{exitCode: 0})
	defer ts.Close()

	helper := newMockDockerClient(t, ts.URL)
	defer helper.Close()

	configDir := t.TempDir()
	engine := New(helper, configDir)

	params := argsparser.RunParams{
		RunCmd: "echo hello",
	}
	err := engine.Run(context.Background(), params)

	if err != nil {
		t.Errorf("RunCmd 模式（退出码 0）应返回 nil，得到: %v", err)
	}

	// 验证 .last_args 已持久化
	lastArgsPath := filepath.Join(configDir, ".last_args")
	if _, statErr := os.Stat(lastArgsPath); os.IsNotExist(statErr) {
		t.Error(".last_args 文件未被持久化")
	}
}

func TestEngineRun_RunCmdMode_NonZeroExit(t *testing.T) {
	// 后台命令模式，命令以非零退出码退出
	ts := newMockDockerAPI(t, mockDockerAPIConfig{exitCode: 42})
	defer ts.Close()

	helper := newMockDockerClient(t, ts.URL)
	defer helper.Close()

	configDir := t.TempDir()
	engine := New(helper, configDir)

	params := argsparser.RunParams{
		RunCmd: "exit 42",
	}
	err := engine.Run(context.Background(), params)

	if err == nil {
		t.Fatal("非零退出码应返回 ExitCodeError")
	}

	var exitErr *ExitCodeError
	if !errors.As(err, &exitErr) {
		t.Fatalf("期望 ExitCodeError，得到 %T: %v", err, err)
	}
	if exitErr.Code != 42 {
		t.Errorf("退出码 = %d, 期望 42", exitErr.Code)
	}

	// 验证 ExitCoder 接口
	if ec := exitErr.ExitCode(); ec != 42 {
		t.Errorf("ExitCode() = %d, 期望 42", ec)
	}

	// 验证错误消息包含退出码
	if !strings.Contains(exitErr.Error(), "42") {
		t.Errorf("错误消息应包含退出码 42，得到: %s", exitErr.Error())
	}

	// 验证 .last_args 已持久化（即使在非零退出码的情况下）
	lastArgsPath := filepath.Join(configDir, ".last_args")
	if _, statErr := os.Stat(lastArgsPath); os.IsNotExist(statErr) {
		t.Error(".last_args 文件未被持久化")
	}
}

func TestEngineRun_AgentMode_ContainerCreateFailure(t *testing.T) {
	// 使用无效 Docker 地址模拟 ContainerCreate 失败
	helper, err := dockerhelper.NewClientWithOpts(
		client.WithHost("tcp://127.0.0.1:1"),
	)
	if err != nil {
		t.Fatalf("NewClientWithOpts(invalid) 失败: %v", err)
	}
	defer helper.Close()

	engine := New(helper, t.TempDir())
	params := argsparser.RunParams{Agent: "claude"}

	err = engine.Run(context.Background(), params)
	if err == nil {
		t.Fatal("ContainerCreate 失败应返回错误")
	}
	if !strings.Contains(err.Error(), "创建容器失败") {
		t.Errorf("错误应包含「创建容器失败」，得到: %v", err)
	}
}

func TestEngineRun_BashMode_ContainerCreateFailure(t *testing.T) {
	// Bash 模式 + ContainerCreate 失败。
	// 与 Agent 模式不同，Bash 模式会先生成 wrapper 脚本，再创建容器。
	helper, err := dockerhelper.NewClientWithOpts(
		client.WithHost("tcp://127.0.0.1:1"),
	)
	if err != nil {
		t.Fatalf("NewClientWithOpts(invalid) 失败: %v", err)
	}
	defer helper.Close()

	engine := New(helper, t.TempDir())
	params := argsparser.RunParams{} // 空 = Bash 模式

	err = engine.Run(context.Background(), params)
	if err == nil {
		t.Fatal("ContainerCreate 失败应返回错误")
	}
	if !strings.Contains(err.Error(), "创建容器失败") {
		t.Errorf("错误应包含「创建容器失败」，得到: %v", err)
	}
}

func TestEngineRun_ContainerStartFailure(t *testing.T) {
	// ContainerCreate 成功但 ContainerStart 失败
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		switch {
		case strings.HasSuffix(path, "/_ping"):
			w.Header().Set("API-Version", "1.41")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))

		case strings.HasSuffix(path, "/containers/create") && r.Method == http.MethodPost:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(container.ContainerCreateCreatedBody{
				ID: "mock-container-id",
			})

		case strings.Contains(path, "/containers/") && strings.HasSuffix(path, "/start") && r.Method == http.MethodPost:
			// 模拟 ContainerStart 失败
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"message":"port is already allocated"}`))

		default:
			t.Logf("[mock-docker] 未处理: %s %s", r.Method, path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ts.Close()

	helper := newMockDockerClient(t, ts.URL)
	defer helper.Close()

	engine := New(helper, t.TempDir())
	params := argsparser.RunParams{RunCmd: "echo test"}

	err := engine.Run(context.Background(), params)
	if err == nil {
		t.Fatal("ContainerStart 失败应返回错误")
	}
	if !strings.Contains(err.Error(), "启动容器失败") {
		t.Errorf("错误应包含「启动容器失败」，得到: %v", err)
	}
}

func TestEngineRun_RunCmdMode_ContextCancelled(t *testing.T) {
	// 后台命令模式，context 被取消（模拟 Ctrl+C）
	// 注：取消的 context 会导致 ContainerCreate 阶段失败，而非 ContainerWait
	ts := newMockDockerAPI(t, mockDockerAPIConfig{exitCode: 0})
	defer ts.Close()

	helper := newMockDockerClient(t, ts.URL)
	defer helper.Close()

	engine := New(helper, t.TempDir())
	params := argsparser.RunParams{RunCmd: "sleep 60"}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // 立即取消 context

	err := engine.Run(ctx, params)
	if err == nil {
		t.Fatal("取消 context 应返回错误")
	}
	// 取消的 context 会使 Docker SDK 的 HTTP 请求立即失败
	if !strings.Contains(err.Error(), "context canceled") {
		t.Errorf("错误应包含「context canceled」，得到: %v", err)
	}
}

// ============================================================================
// Engine.Run 单元测试 — 交互模式（attach + 双向流拷贝）
// ============================================================================

// newInteractiveMockServer 创建支持 attach hijacking 的 mock Docker API 服务器。
func newInteractiveMockServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		method := r.Method

		switch {
		case strings.HasSuffix(path, "/_ping"):
			w.Header().Set("API-Version", "1.41")
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))

		case strings.HasSuffix(path, "/containers/create") && method == http.MethodPost:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(container.ContainerCreateCreatedBody{
				ID: "mock-container-id",
			})

		case strings.Contains(path, "/containers/") && strings.HasSuffix(path, "/start") && method == http.MethodPost:
			w.WriteHeader(http.StatusNoContent)

		case strings.Contains(path, "/containers/") && strings.HasSuffix(path, "/attach") && method == http.MethodPost:
			hj, ok := w.(http.Hijacker)
			if !ok {
				http.Error(w, "hijacking not supported", http.StatusInternalServerError)
				return
			}
			conn, _, err := hj.Hijack()
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			conn.Write([]byte("HTTP/1.1 101 UPGRADED\r\nContent-Type: application/vnd.docker.raw-stream\r\n\r\n"))
			conn.Write([]byte("$\r\n"))
			conn.Close()

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

// replaceStdinWithClosedPipe 替换 os.Stdin 为已关闭 pipe reader，
// 使 io.Copy(attachResp.Conn, os.Stdin) 立即返回。
// 返回的函数用于恢复 os.Stdin。
func replaceStdinWithClosedPipe(t *testing.T) func() {
	t.Helper()
	stdinReader, stdinWriter, err := os.Pipe()
	if err != nil {
		t.Fatalf("创建 pipe 失败: %v", err)
	}
	stdinWriter.Close()
	origStdin := os.Stdin
	os.Stdin = stdinReader
	return func() {
		os.Stdin = origStdin
		stdinReader.Close()
	}
}

func TestEngineRun_InteractiveMode_ContainerAttachFailure(t *testing.T) {
	// 创建和启动成功，但 ContainerAttach 失败
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		method := r.Method

		switch {
		case strings.HasSuffix(path, "/_ping"):
			w.Header().Set("API-Version", "1.41")
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))

		case strings.HasSuffix(path, "/containers/create") && method == http.MethodPost:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(container.ContainerCreateCreatedBody{
				ID: "mock-container-id",
			})

		case strings.Contains(path, "/containers/") && strings.HasSuffix(path, "/start") && method == http.MethodPost:
			w.WriteHeader(http.StatusNoContent)

		case strings.Contains(path, "/containers/") && strings.HasSuffix(path, "/attach") && method == http.MethodPost:
			// 模拟 attach 失败
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"message":"container is not running"}`))

		default:
			t.Logf("[mock-docker] 未处理: %s %s", method, path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ts.Close()

	helper := newMockDockerClient(t, ts.URL)
	defer helper.Close()

	configDir := t.TempDir()
	engine := New(helper, configDir)

	params := argsparser.RunParams{Agent: "claude"}
	err := engine.Run(context.Background(), params)

	if err == nil {
		t.Fatal("ContainerAttach 失败应返回错误")
	}
	if !strings.Contains(err.Error(), "附加到容器失败") {
		t.Errorf("错误应包含「附加到容器失败」，得到: %v", err)
	}
}

func TestEngineRun_InteractiveMode_Agent(t *testing.T) {
	// 完整交互模式：容器创建 → 启动 → attach hijack → 双向流拷贝 → 保存参数
	//
	// 此测试通过 HTTP 连接 hijacking 模拟 Docker 的 attach raw-stream 协议。
	// Docker SDK 的 postHijacked 使用 httputil.ClientConn 发送请求并读取 101 响应，
	// 然后 Hijack 获取原始连接用于双向流拷贝。
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		method := r.Method

		switch {
		case strings.HasSuffix(path, "/_ping"):
			w.Header().Set("API-Version", "1.41")
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))

		case strings.HasSuffix(path, "/containers/create") && method == http.MethodPost:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(container.ContainerCreateCreatedBody{
				ID: "mock-container-id",
			})

		case strings.Contains(path, "/containers/") && strings.HasSuffix(path, "/start") && method == http.MethodPost:
			w.WriteHeader(http.StatusNoContent)

		case strings.Contains(path, "/containers/") && strings.HasSuffix(path, "/attach") && method == http.MethodPost:
			// Hijack 连接以模拟 Docker attach raw-stream 协议
			hj, ok := w.(http.Hijacker)
			if !ok {
				http.Error(w, "hijacking not supported", http.StatusInternalServerError)
				return
			}
			conn, _, err := hj.Hijack()
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			// 发送 101 Switching Protocols 响应（Docker raw-stream 协议标准），
			// 然后写入一些数据（模拟容器提示符），最后关闭连接
			conn.Write([]byte("HTTP/1.1 101 UPGRADED\r\nContent-Type: application/vnd.docker.raw-stream\r\n\r\n"))
			conn.Write([]byte("$\r\n"))
			conn.Close()

		default:
			t.Logf("[mock-docker] 未处理: %s %s", method, path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ts.Close()

	helper := newMockDockerClient(t, ts.URL)
	defer helper.Close()

	// 替换 os.Stdin 为已关闭的 pipe reader，使 io.Copy(attachResp.Conn, os.Stdin)
	// 立即返回（pipe 写端已关闭，Read 返回 EOF）
	stdinReader, stdinWriter, err := os.Pipe()
	if err != nil {
		t.Fatalf("创建 pipe 失败: %v", err)
	}
	defer stdinReader.Close()
	stdinWriter.Close()

	origStdin := os.Stdin
	os.Stdin = stdinReader
	defer func() { os.Stdin = origStdin }()

	configDir := t.TempDir()
	engine := New(helper, configDir)

	params := argsparser.RunParams{Agent: "claude"}
	err = engine.Run(context.Background(), params)

	if err != nil {
		t.Errorf("交互模式应返回 nil，得到: %v", err)
	}

	// 验证 .last_args 已持久化
	lastArgsPath := filepath.Join(configDir, ".last_args")
	if _, statErr := os.Stat(lastArgsPath); os.IsNotExist(statErr) {
		t.Error(".last_args 文件未被持久化")
	}
}

func TestEngineRun_InteractiveMode_StreamError(t *testing.T) {
	// 交互模式：os.Stdout 替换为已关闭 pipe，io.Copy 返回错误触发流异常分支
	ts := newInteractiveMockServer(t)
	defer ts.Close()

	helper := newMockDockerClient(t, ts.URL)
	defer helper.Close()

	restoreStdin := replaceStdinWithClosedPipe(t)
	defer restoreStdin()

	// 替换 os.Stdout 为已关闭 pipe 使 io.Copy 返回写错误
	stdoutReader, stdoutWriter, _ := os.Pipe()
	origStdout := os.Stdout
	os.Stdout = stdoutWriter
	stdoutWriter.Close()
	defer func() {
		os.Stdout = origStdout
		stdoutReader.Close()
	}()

	configDir := t.TempDir()
	engine := New(helper, configDir)
	params := argsparser.RunParams{Agent: "claude"}

	err := engine.Run(context.Background(), params)
	if err == nil {
		t.Fatal("流错误应触发错误返回")
	}
	if !strings.Contains(err.Error(), "容器流异常断开") {
		t.Errorf("错误应包含「容器流异常断开」，得到: %v", err)
	}
}

func TestEngineRun_InteractiveMode_SaveParamsError(t *testing.T) {
	// 交互模式：configDir 是文件，Save 失败触发持久化错误分支
	ts := newInteractiveMockServer(t)
	defer ts.Close()

	helper := newMockDockerClient(t, ts.URL)
	defer helper.Close()

	restoreStdin := replaceStdinWithClosedPipe(t)
	defer restoreStdin()

	// configDir 是文件而非目录，Save 时 os.MkdirAll 会失败
	parentDir := t.TempDir()
	configDir := filepath.Join(parentDir, "config_file")
	if err := os.WriteFile(configDir, []byte("not a dir"), 0644); err != nil {
		t.Fatalf("创建 config 文件失败: %v", err)
	}

	engine := New(helper, configDir)
	params := argsparser.RunParams{Agent: "claude"}

	err := engine.Run(context.Background(), params)
	if err == nil {
		t.Fatal("save 失败应返回错误")
	}
	if !strings.Contains(err.Error(), "持久化运行参数失败") {
		t.Errorf("错误应包含「持久化运行参数失败」，得到: %v", err)
	}
}

// ============================================================================
// Engine.Run 单元测试 — 其他剩余分支
// ============================================================================

func TestEngineRun_RecallMode_LoadError(t *testing.T) {
	// 创建不可读的 .last_args 文件，触发非 ErrFileNotFound 错误路径
	configDir := t.TempDir()
	lastArgsPath := filepath.Join(configDir, ".last_args")
	if err := os.WriteFile(lastArgsPath, []byte("data"), 0000); err != nil {
		t.Fatalf("创建 .last_args 失败: %v", err)
	}

	engine := New(nil, configDir)
	params := argsparser.RunParams{Recall: true}

	err := engine.Run(context.Background(), params)
	if err == nil {
		t.Fatal("recall 加载失败应返回错误")
	}
	if !strings.Contains(err.Error(), "读取 .last_args 失败") {
		t.Errorf("错误应包含「读取 .last_args 失败」，得到: %v", err)
	}
}

func TestEngineRun_RecallMode_Success(t *testing.T) {
	// Recall 模式成功：创建有效 .last_args 文件，RunCmd 模式执行
	ts := newMockDockerAPI(t, mockDockerAPIConfig{exitCode: 0})
	defer ts.Close()

	helper := newMockDockerClient(t, ts.URL)
	defer helper.Close()

	configDir := t.TempDir()

	// 创建有效的 .last_args（RunCmd 模式）
	p := argspersistence.New(configDir)
	if err := p.Save(argsparser.RunParams{RunCmd: "echo test"}); err != nil {
		t.Fatalf("保存 .last_args 失败: %v", err)
	}

	engine := New(helper, configDir)
	params := argsparser.RunParams{Recall: true}

	err := engine.Run(context.Background(), params)
	if err != nil {
		t.Errorf("recall 模式应成功，得到: %v", err)
	}
}

func TestEngineRun_RunCmdMode_ContainerWaitError(t *testing.T) {
	// ContainerWait 返回错误
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		method := r.Method

		switch {
		case strings.HasSuffix(path, "/_ping"):
			w.Header().Set("API-Version", "1.41")
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))

		case strings.HasSuffix(path, "/containers/create") && method == http.MethodPost:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(container.ContainerCreateCreatedBody{
				ID: "mock-container-id",
			})

		case strings.Contains(path, "/containers/") && strings.HasSuffix(path, "/start") && method == http.MethodPost:
			w.WriteHeader(http.StatusNoContent)

		case strings.Contains(path, "/containers/") && strings.HasSuffix(path, "/wait") && method == http.MethodPost:
			// 返回 500 模拟 ContainerWait 失败
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"message":"container not found"}`))

		default:
			t.Logf("[mock-docker] 未处理: %s %s", method, path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ts.Close()

	helper := newMockDockerClient(t, ts.URL)
	defer helper.Close()

	configDir := t.TempDir()
	engine := New(helper, configDir)

	err := engine.Run(context.Background(), argsparser.RunParams{RunCmd: "echo test"})
	if err == nil {
		t.Fatal("ContainerWait 失败应返回错误")
	}
	if !strings.Contains(err.Error(), "等待容器退出失败") {
		t.Errorf("错误应包含「等待容器退出失败」，得到: %v", err)
	}
}

func TestEngineRun_RunCmdMode_SaveParamsError(t *testing.T) {
	// 持久化参数失败：configDir 是文件而非目录
	ts := newMockDockerAPI(t, mockDockerAPIConfig{exitCode: 0})
	defer ts.Close()

	helper := newMockDockerClient(t, ts.URL)
	defer helper.Close()

	parentDir := t.TempDir()
	configDir := filepath.Join(parentDir, "config_file")
	if err := os.WriteFile(configDir, []byte("not a dir"), 0644); err != nil {
		t.Fatalf("创建 config 文件失败: %v", err)
	}

	engine := New(helper, configDir)

	err := engine.Run(context.Background(), argsparser.RunParams{RunCmd: "echo hello"})
	if err == nil {
		t.Fatal("save 失败应返回错误")
	}
	if !strings.Contains(err.Error(), "持久化运行参数失败") {
		t.Errorf("错误应包含「持久化运行参数失败」，得到: %v", err)
	}
}

// ============================================================================
// AssembleContainerConfig 额外分支
// ============================================================================

func TestAssembleContainerConfig_TermFromHostEmpty(t *testing.T) {
	// 宿主 TERM 环境变量为空时，应 fallback 到 xterm-256color
	savedTerm := os.Getenv("TERM")
	os.Unsetenv("TERM")
	defer os.Setenv("TERM", savedTerm)

	params := argsparser.RunParams{}
	config, _, _ := AssembleContainerConfig(params, "")

	found := false
	for _, e := range config.Env {
		if e == "TERM=xterm-256color" {
			found = true
			break
		}
	}
	if !found {
		t.Error("TERM 为空时应使用 xterm-256color fallback")
	}
}

func TestAssembleContainerConfig_GetwdFails(t *testing.T) {
	// os.Getwd() 失败时（当前目录已被删除），应 fallback 到 /workspace
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("获取当前目录失败: %v", err)
	}
	defer os.Chdir(originalDir)

	// chdir 到临时目录后删除它，使 Getwd 返回 ENOENT
	dir, err := os.MkdirTemp("", "getwd-fail-*")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir 失败: %v", err)
	}
	if err := os.RemoveAll(dir); err != nil {
		t.Fatalf("删除目录失败: %v", err)
	}

	params := argsparser.RunParams{}
	config, _, _ := AssembleContainerConfig(params, "")
	if config.WorkingDir != "/workspace" {
		t.Errorf("os.Getwd() fail 时应 fallback 到 /workspace, 得到 %s", config.WorkingDir)
	}
}

func TestAssembleContainerConfig_InvalidContainerPort(t *testing.T) {
	// 端口格式 host:container 中 container 为不合法值（非数字），
	// nat.NewPort 返回 err，PortBindings 中跳过（continue），ExposedPorts 中跳过。
	params := argsparser.RunParams{
		Ports: []string{"3000:abc"},
	}
	_, hostConfig, _ := AssembleContainerConfig(params, "")

	if len(hostConfig.PortBindings) != 0 {
		t.Errorf("无效容器端口不应生成 PortBindings，得到 %d", len(hostConfig.PortBindings))
	}
}

func TestEngineRun_RunCmdMode_ContextCancelledDuringWait(t *testing.T) {
	// RunCmd 模式：在 ContainerWait 阻塞期间取消 context，
	// 覆盖 select 中的 case <-ctx.Done() 分支。
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// waitCh 让 /wait 处理程序阻塞，确保 ContainerWait 一直挂起
	waitCh := make(chan struct{})

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		method := r.Method

		switch {
		case strings.HasSuffix(path, "/_ping"):
			w.Header().Set("API-Version", "1.41")
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))

		case strings.HasSuffix(path, "/containers/create") && method == http.MethodPost:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(container.ContainerCreateCreatedBody{
				ID: "mock-container-id",
			})

		case strings.Contains(path, "/containers/") && strings.HasSuffix(path, "/start") && method == http.MethodPost:
			w.WriteHeader(http.StatusNoContent)

		case strings.Contains(path, "/containers/") && strings.HasSuffix(path, "/wait") && method == http.MethodPost:
			// 用 select 同时监听 waitCh 和客户端断开，防止泄漏
			select {
			case <-waitCh:
			case <-r.Context().Done():
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(container.ContainerWaitOKBody{
				StatusCode: 0,
			})

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	// 重要：ts.Close 必须先注册（后执行），close(waitCh) 后注册（先执行），
	// 确保 mock handler 先解除阻塞再关闭 server，避免 server 等待活跃连接超时。
	defer ts.Close()
	defer close(waitCh)

	helper := newMockDockerClient(t, ts.URL)
	defer helper.Close()

	engine := New(helper, t.TempDir())
	params := argsparser.RunParams{RunCmd: "sleep 10"}

	// 50ms 后取消 context，此时 select 正阻塞在三个 channel 上
	time.AfterFunc(50*time.Millisecond, cancel)

	err := engine.Run(ctx, params)
	if err == nil {
		t.Fatal("等待期间 context 取消应返回错误")
	}

	// select 调度不确定：可能是 ctx.Done() 被选中，也可能是 errCh 先收到取消错误
	if !strings.Contains(err.Error(), "等待容器退出被中断") &&
		!strings.Contains(err.Error(), "等待容器退出失败") {
		t.Errorf("错误应包含中断或失败信息，得到: %v", err)
	}
}

func TestEngineRun_InteractiveMode_SignalHandling(t *testing.T) {
	// 交互模式：向进程发送 SIGINT，测试信号转发到容器的完整路径
	// hijack 连接保持打开，使流拷贝 goroutine 阻塞，select 等待信号
	done := make(chan struct{})

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		method := r.Method

		switch {
		case strings.HasSuffix(path, "/_ping"):
			w.Header().Set("API-Version", "1.41")
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))

		case strings.HasSuffix(path, "/containers/create") && method == http.MethodPost:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(container.ContainerCreateCreatedBody{
				ID: "mock-container-id",
			})

		case strings.Contains(path, "/containers/") && strings.HasSuffix(path, "/start") && method == http.MethodPost:
			w.WriteHeader(http.StatusNoContent)

		case strings.Contains(path, "/containers/") && strings.HasSuffix(path, "/attach") && method == http.MethodPost:
			hj, ok := w.(http.Hijacker)
			if !ok {
				http.Error(w, "hijacking not supported", http.StatusInternalServerError)
				return
			}
			conn, _, err := hj.Hijack()
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			conn.Write([]byte("HTTP/1.1 101 UPGRADED\r\nContent-Type: application/vnd.docker.raw-stream\r\n\r\n"))
			// 保持连接打开，使 stdout io.Copy 阻塞
			<-done
			conn.Close()

		case strings.Contains(path, "/containers/") && strings.HasSuffix(path, "/kill") && method == http.MethodPost:
			// 返回错误使信号处理走 killErr 分支
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"message":"container not found"}`))

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	// 先注册 ts.Close（后执行），再注册 close(done)（先执行），
	// 确保 hijack handler 先解除阻塞再关闭 server 的活跃连接跟踪。
	defer ts.Close()
	defer close(done)

	helper := newMockDockerClient(t, ts.URL)
	defer helper.Close()

	// stdin: pipe（不关闭写端），使 stdin io.Copy 阻塞
	stdinReader, stdinWriter, err := os.Pipe()
	if err != nil {
		t.Fatalf("创建 stdin pipe 失败: %v", err)
	}

	origStdin := os.Stdin
	os.Stdin = stdinReader
	defer func() {
		os.Stdin = origStdin
		stdinReader.Close()
		stdinWriter.Close()
	}()

	// stdout: pipe，接收容器输出
	stdoutReader, stdoutWriter, err := os.Pipe()
	if err != nil {
		t.Fatalf("创建 stdout pipe 失败: %v", err)
	}

	origStdout := os.Stdout
	os.Stdout = stdoutWriter
	defer func() {
		os.Stdout = origStdout
		stdoutReader.Close()
		stdoutWriter.Close()
	}()

	configDir := t.TempDir()
	engine := New(helper, configDir)
	params := argsparser.RunParams{Agent: "claude"}

	// 50ms 后发送 SIGINT
	time.AfterFunc(50*time.Millisecond, func() {
		syscall.Kill(syscall.Getpid(), syscall.SIGINT)
	})

	err = engine.Run(context.Background(), params)
	if err == nil {
		t.Fatal("信号处理应返回错误")
	}
	if !strings.Contains(err.Error(), "发送 SIGINT 信号到容器失败") {
		t.Errorf("错误应包含「发送 SIGINT 信号到容器失败」，得到: %v", err)
	}

	// 验证 .last_args 未被持久化（信号导致提前返回）
	lastArgsPath := filepath.Join(configDir, ".last_args")
	if _, statErr := os.Stat(lastArgsPath); !os.IsNotExist(statErr) {
		t.Error("信号发生时不应持久化参数")
	}
}

// ============================================================================
// toContainerPath 单元测试
// ============================================================================

func TestToContainerPath_LinuxPathUnchanged(t *testing.T) {
	// Linux / WSL 绝对路径应原样返回
	tests := []string{
		"/workspace",
		"/home/user/project",
		"/",
		"/mnt/data",
	}
	for _, path := range tests {
		got := toContainerPath(path, "")
		if got != path {
			t.Errorf("toContainerPath(%q) = %q, want %q (already-Linux path should be unchanged)", path, got, path)
		}
	}
}

func TestToContainerPath_WindowsPathOnWindows(t *testing.T) {
	// Windows 路径转换仅在 Windows 平台生效。
	// 在非 Windows 平台上，filepath.VolumeName 不识别 Windows 卷标，
	// 路径被原样返回（运行时不会收到 Windows 路径，此行为安全）。
	vol := filepath.VolumeName(`D:\code\AgentForge`)
	if vol == "" {
		// 非 Windows 平台：Windows 路径不会被转换，但运行时也不会产生这种路径
		t.Skip("非 Windows 平台，filepath.VolumeName 不识别 Windows 卷标，跳过 Windows 特有测试")
		return
	}

	// Windows 平台：验证盘符 → /mnt/<drive>/ 转换
	tests := []struct {
		input string
		want  string
	}{
		{`D:\code\AgentForge`, "/mnt/d/code/AgentForge"},
		{`C:\Users\me\project`, "/mnt/c/Users/me/project"},
		{`E:\`, "/mnt/e/"},
		{`C:\`, "/mnt/c/"},
		{`\\server\share\foo\bar`, "/mnt/server/share/foo/bar"},
	}
	for _, tt := range tests {
		got := toContainerPath(tt.input, "")
		if got != tt.want {
			t.Errorf("toContainerPath(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestToContainerPath_RelativePath(t *testing.T) {
	// 相对路径无卷标，应原样返回（仅做斜杠转换）
	got := toContainerPath("foo/bar/baz", "")
	if got != "foo/bar/baz" {
		t.Errorf("toContainerPath(relative) = %q, want %q", got, "foo/bar/baz")
	}
}

// ============================================================================
// 辅助函数
// ============================================================================

func strSliceEqual(a, b strslice.StrSlice) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// --- toWindowsContainerPath tests ---

func TestToWindowsContainerPath_WSL(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"/mnt/d/code/AgentForge", `D:\code\AgentForge`},
		{"/mnt/c/Users/me", `C:\Users\me`},
		{"/home/user/project", `C:\home\user\project`},
		{"/workspace", `C:\workspace`},
		{"/mnt/x", `X:`},
	}
	for _, tt := range tests {
		got := toWindowsContainerPath(tt.input)
		if got != tt.want {
			t.Errorf("toWindowsContainerPath(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestToContainerPath_Windows(t *testing.T) {
	got := toContainerPath("/mnt/d/code/AgentForge", "windows")
	want := `D:\code\AgentForge`
	if got != want {
		t.Errorf("toContainerPath(/mnt/d/code/AgentForge, windows) = %q, want %q", got, want)
	}
}
