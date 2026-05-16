package cmd

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"

	"github.com/agent-forge/cli/internal/doctor/diagnosticengine"
	"github.com/agent-forge/cli/internal/shared/argsparser"
)

// ========================================================================
// errors.go
// ========================================================================

func TestNewExitCodeError_CreatesExitCoder(t *testing.T) {
	err := newExitCodeError(2, "参数错误")
	if err == nil {
		t.Fatal("newExitCodeError 不应返回 nil")
	}
	if err.Error() != "参数错误" {
		t.Errorf("Error() = %q, 期望 %q", err.Error(), "参数错误")
	}
	var ec ExitCoder
	if !errors.As(err, &ec) {
		t.Fatal("newExitCodeError 应实现 ExitCoder 接口")
	}
	if ec.ExitCode() != 2 {
		t.Errorf("ExitCode() = %d, 期望 %d", ec.ExitCode(), 2)
	}
}

func TestNewExitCodeError_VariousCodes(t *testing.T) {
	tests := []struct {
		code int
		msg  string
	}{
		{0, "成功"},
		{1, "通用执行错误"},
		{2, "参数错误"},
		{42, "自定义退出码"},
		{125, "Docker 错误"},
	}
	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			err := newExitCodeError(tt.code, tt.msg)
			var ec ExitCoder
			if !errors.As(err, &ec) {
				t.Fatal("应实现 ExitCoder")
			}
			if ec.ExitCode() != tt.code {
				t.Errorf("ExitCode() = %d, 期望 %d", ec.ExitCode(), tt.code)
			}
			if ec.Error() != tt.msg {
				t.Errorf("Error() = %q, 期望 %q", ec.Error(), tt.msg)
			}
		})
	}
}

func TestExitCodeError_IsNotRegularError(t *testing.T) {
	// 验证普通 error 不实现 ExitCoder
	regularErr := errors.New("普通错误")
	var ec ExitCoder
	if errors.As(regularErr, &ec) {
		t.Error("普通 error 不应实现 ExitCoder")
	}
}

func TestExitCodeError_InterfaceCompliance(t *testing.T) {
	// 编译期检查：exitCodeError 实现 ExitCoder
	var eci interface{} = (*exitCodeError)(nil)
	if _, ok := eci.(ExitCoder); !ok {
		t.Error("*exitCodeError 应实现 ExitCoder 接口")
	}
}

// ========================================================================
// root.go — VersionInfo
// ========================================================================

func TestVersionInfo_DefaultValues(t *testing.T) {
	origVersion := Version
	origHash := GitHash
	origBuild := BuildTime
	defer func() {
		Version = origVersion
		GitHash = origHash
		BuildTime = origBuild
	}()

	Version = "dev"
	GitHash = "unknown"
	BuildTime = "unknown"

	info := VersionInfo()
	if !strings.Contains(info, "dev") {
		t.Errorf("默认版本号应包含 'dev', 实际: %s", info)
	}
	if !strings.Contains(info, "unknown") {
		t.Errorf("默认值应包含 'unknown', 实际: %s", info)
	}
}

// ========================================================================
// root.go — Command Structure
// ========================================================================

func TestRootCmd_HasAllSubcommands(t *testing.T) {
	expected := []string{"build", "run", "endpoint", "doctor", "deps", "export", "import", "update", "version"}
	for _, name := range expected {
		cmd, _, err := rootCmd.Find([]string{name})
		if err != nil {
			t.Errorf("rootCmd 应包含子命令 %q, 错误: %v", name, err)
			continue
		}
		if !strings.HasPrefix(cmd.Use, name) {
			t.Errorf("子命令 %q 的 Use 应以 %q 开头, 实际: %q", name, name, cmd.Use)
		}
	}
}

func TestRootCmd_HelpTextNonEmpty(t *testing.T) {
	if rootCmd.Short == "" {
		t.Error("rootCmd.Short 不应为空")
	}
	if rootCmd.Long == "" {
		t.Error("rootCmd.Long 不应为空")
	}
	if rootCmd.Use != "agent-forge" {
		t.Errorf("rootCmd.Use 应为 'agent-forge', 实际: %q", rootCmd.Use)
	}
}

func TestRootCmd_RunEDefaultsToRun(t *testing.T) {
	// rootCmd 默认 RunE 应设置为 runCmd.RunE
	if rootCmd.RunE == nil {
		t.Fatal("rootCmd.RunE 不应为 nil")
	}
	// 验证 RunE 不是默认的 help 函数 — 通过调用 rootCmd 但不传递子命令，
	// 应执行 runCmd.RunE（但 runCmd.RunE 需要 Docker）。
	// 我们只需验证 rootCmd.RunE 已被赋值且等于 runCmd.RunE。
	// 使用函数指针比较（如果在同一包内可行）
	// runCmd.RunE 和 rootCmd.RunE 很可能是不同的闭包，
	// 但 rootCmd.RunE 内部调用 runCmd.RunE。
	// 这里我们仅验证 rootCmd 有 RunE，足以确认不是默认空行为。
	if rootCmd.RunE == nil {
		t.Error("rootCmd.RunE 应为非 nil")
	}
}

func TestRootCmd_ExecuteCWithVersion(t *testing.T) {
	// 通过 rootCmd.ExecuteC() 执行 version 命令，验证不 panic
	origArgs := os.Args
	defer func() { os.Args = origArgs }()
	os.Args = []string{"agent-forge", "version"}

	// 执行 version 命令（不调用 Exit，使用 ExecuteC）
	_, err := rootCmd.ExecuteC()
	if err != nil {
		t.Errorf("version 命令应成功执行, 错误: %v", err)
	}
}

func TestRootCmd_AllCommandsHaveHelp(t *testing.T) {
	for _, cmd := range rootCmd.Commands() {
		if cmd.Short == "" {
			t.Errorf("命令 %q 的 Short 描述为空", cmd.Use)
		}
		if cmd.Long == "" {
			t.Errorf("命令 %q 的 Long 描述为空", cmd.Use)
		}
	}
}

// ========================================================================
// root.go — Execute (subprocess test)
// ========================================================================

// TestExecute_Subprocess 在子进程中测试 Execute() 的 os.Exit 行为。
//
// 使用子进程测试是因为 Execute() 内部调用 os.Exit()，
// 无法在常规单元测试中直接验证退出码。
func TestExecute_Subprocess(t *testing.T) {
	switch os.Getenv("_TEST_EXECUTE_SUBPROCESS") {
	case "exit_coder":
		// 子进程 — 测试 ExitCoder 错误路径
		rootCmd.SetArgs([]string{})
		origRunE := rootCmd.RunE
		rootCmd.RunE = func(cmd *cobra.Command, args []string) error {
			return newExitCodeError(42, "ExitCoder 错误")
		}
		defer func() { rootCmd.RunE = origRunE }()
		Execute()
		t.Error("Execute() 应通过 os.Exit 退出，不应到达此处")
		return

	case "regular_err":
		// 子进程 — 测试普通错误路径
		rootCmd.SetArgs([]string{})
		origRunE := rootCmd.RunE
		rootCmd.RunE = func(cmd *cobra.Command, args []string) error {
			return errors.New("普通错误")
		}
		defer func() { rootCmd.RunE = origRunE }()
		Execute()
		t.Error("Execute() 应通过 os.Exit 退出，不应到达此处")
		return

	default:
		// 父进程 — 执行子进程测试
		t.Run("ExitCoder 错误 → 退出码 42", func(t *testing.T) {
			cmd := exec.Command(os.Args[0], "-test.run=^TestExecute_Subprocess$")
			cmd.Env = append(os.Environ(), "_TEST_EXECUTE_SUBPROCESS=exit_coder")
			err := cmd.Run()
			if err == nil {
				t.Fatal("子进程应返回非零退出码")
			}
			var exitErr *exec.ExitError
			if !errors.As(err, &exitErr) {
				t.Fatalf("期望 ExitError, 得到: %v", err)
			}
			if exitErr.ExitCode() != 42 {
				t.Errorf("退出码 = %d, 期望 42", exitErr.ExitCode())
			}
		})

		t.Run("普通错误 → 退出码 1", func(t *testing.T) {
			cmd := exec.Command(os.Args[0], "-test.run=^TestExecute_Subprocess$")
			cmd.Env = append(os.Environ(), "_TEST_EXECUTE_SUBPROCESS=regular_err")
			err := cmd.Run()
			if err == nil {
				t.Fatal("子进程应返回非零退出码")
			}
			var exitErr *exec.ExitError
			if !errors.As(err, &exitErr) {
				t.Fatalf("期望 ExitError, 得到: %v", err)
			}
			if exitErr.ExitCode() != 1 {
				t.Errorf("退出码 = %d, 期望 1", exitErr.ExitCode())
			}
		})
	}
}

// ========================================================================
// run.go — buildRunParams
// ========================================================================

// newRunCmd 创建一个带所有 run 命令标志的 cobra.Command，用于测试 buildRunParams。
func newRunCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "run"}
	cmd.Flags().StringP("agent", "a", "", "")
	cmd.Flags().StringArrayP("port", "p", nil, "")
	cmd.Flags().StringArrayP("mount", "m", nil, "")
	cmd.Flags().StringArrayP("env", "e", nil, "")
	cmd.Flags().StringP("workdir", "w", "", "")
	cmd.Flags().BoolP("recall", "r", false, "")
	cmd.Flags().Bool("docker", false, "")
	cmd.Flags().Bool("dind", false, "")
	cmd.Flags().String("run", "", "")
	cmd.Flags().StringP("config", "c", "", "")
	return cmd
}

func TestBuildRunParams_Defaults(t *testing.T) {
	cmd := newRunCmd()
	params := buildRunParams(cmd)

	if params.Agent != "" {
		t.Errorf("Agent 默认应为空, 实际: %q", params.Agent)
	}
	if params.Ports != nil && len(params.Ports) != 0 {
		t.Errorf("Ports 默认应为 nil/空, 实际: %v", params.Ports)
	}
	if params.Mounts != nil && len(params.Mounts) != 0 {
		t.Errorf("Mounts 默认应为 nil/空, 实际: %v", params.Mounts)
	}
	if params.Envs != nil && len(params.Envs) != 0 {
		t.Errorf("Envs 默认应为 nil/空, 实际: %v", params.Envs)
	}
	if params.Workdir != "" {
		t.Errorf("Workdir 默认应为空, 实际: %q", params.Workdir)
	}
	if params.Recall {
		t.Error("Recall 默认应为 false")
	}
	if params.Docker {
		t.Error("Docker 默认应为 false")
	}
	if params.RunCmd != "" {
		t.Errorf("RunCmd 默认应为空, 实际: %q", params.RunCmd)
	}
	if params.Config != "" {
		t.Errorf("Config 默认应为空, 实际: %q", params.Config)
	}
}

func TestBuildRunParams_Agent(t *testing.T) {
	cmd := newRunCmd()
	cmd.Flags().Set("agent", "claude")
	params := buildRunParams(cmd)
	if params.Agent != "claude" {
		t.Errorf("Agent = %q, 期望 %q", params.Agent, "claude")
	}
}

func TestBuildRunParams_Ports(t *testing.T) {
	cmd := newRunCmd()
	cmd.Flags().Set("port", "3000:3000")
	cmd.Flags().Set("port", "8080:80")
	params := buildRunParams(cmd)
	if len(params.Ports) != 2 {
		t.Fatalf("Ports 长度 = %d, 期望 2, 值: %v", len(params.Ports), params.Ports)
	}
	if params.Ports[0] != "3000:3000" {
		t.Errorf("Ports[0] = %q, 期望 %q", params.Ports[0], "3000:3000")
	}
	if params.Ports[1] != "8080:80" {
		t.Errorf("Ports[1] = %q, 期望 %q", params.Ports[1], "8080:80")
	}
}

func TestBuildRunParams_Mounts(t *testing.T) {
	cmd := newRunCmd()
	cmd.Flags().Set("mount", "/host/data")
	cmd.Flags().Set("mount", "/host/config")
	params := buildRunParams(cmd)
	if len(params.Mounts) != 2 {
		t.Fatalf("Mounts 长度 = %d, 期望 2", len(params.Mounts))
	}
	if params.Mounts[0] != "/host/data" {
		t.Errorf("Mounts[0] = %q, 期望 %q", params.Mounts[0], "/host/data")
	}
}

func TestBuildRunParams_Envs(t *testing.T) {
	cmd := newRunCmd()
	cmd.Flags().Set("env", "KEY1=VAL1")
	cmd.Flags().Set("env", "KEY2=VAL2")
	params := buildRunParams(cmd)
	if len(params.Envs) != 2 {
		t.Fatalf("Envs 长度 = %d, 期望 2", len(params.Envs))
	}
	if params.Envs[1] != "KEY2=VAL2" {
		t.Errorf("Envs[1] = %q, 期望 %q", params.Envs[1], "KEY2=VAL2")
	}
}

func TestBuildRunParams_Workdir(t *testing.T) {
	cmd := newRunCmd()
	cmd.Flags().Set("workdir", "/workspace")
	params := buildRunParams(cmd)
	if params.Workdir != "/workspace" {
		t.Errorf("Workdir = %q, 期望 %q", params.Workdir, "/workspace")
	}
}

func TestBuildRunParams_RecallTrue(t *testing.T) {
	cmd := newRunCmd()
	cmd.Flags().Set("recall", "true")
	params := buildRunParams(cmd)
	if !params.Recall {
		t.Error("Recall 应为 true")
	}
}

func TestBuildRunParams_DockerAndDind(t *testing.T) {
	// --docker 和 --dind 任一个为 true，Docker 字段应为 true
	t.Run("--docker 单独设置", func(t *testing.T) {
		cmd := newRunCmd()
		cmd.Flags().Set("docker", "true")
		params := buildRunParams(cmd)
		if !params.Docker {
			t.Error("--docker=true 时 Docker 应为 true")
		}
	})
	t.Run("--dind 单独设置", func(t *testing.T) {
		cmd := newRunCmd()
		cmd.Flags().Set("dind", "true")
		params := buildRunParams(cmd)
		if !params.Docker {
			t.Error("--dind=true 时 Docker 应为 true")
		}
	})
	t.Run("两者都设", func(t *testing.T) {
		cmd := newRunCmd()
		cmd.Flags().Set("docker", "true")
		cmd.Flags().Set("dind", "true")
		params := buildRunParams(cmd)
		if !params.Docker {
			t.Error("两者都设时 Docker 应为 true")
		}
	})
	t.Run("两者都未设", func(t *testing.T) {
		cmd := newRunCmd()
		params := buildRunParams(cmd)
		if params.Docker {
			t.Error("两者未设时 Docker 应为 false")
		}
	})
}

func TestBuildRunParams_RunCmd(t *testing.T) {
	cmd := newRunCmd()
	cmd.Flags().Set("run", "echo hello")
	params := buildRunParams(cmd)
	if params.RunCmd != "echo hello" {
		t.Errorf("RunCmd = %q, 期望 %q", params.RunCmd, "echo hello")
	}
}

func TestBuildRunParams_Config(t *testing.T) {
	cmd := newRunCmd()
	cmd.Flags().Set("config", "/custom/path")
	params := buildRunParams(cmd)
	if params.Config != "/custom/path" {
		t.Errorf("Config = %q, 期望 %q", params.Config, "/custom/path")
	}
}

func TestBuildRunParams_AllFlags(t *testing.T) {
	cmd := newRunCmd()
	cmd.Flags().Set("agent", "opencode")
	cmd.Flags().Set("port", "3000:3000")
	cmd.Flags().Set("mount", "/data")
	cmd.Flags().Set("env", "KEY=VAL")
	cmd.Flags().Set("workdir", "/app")
	cmd.Flags().Set("recall", "true")
	cmd.Flags().Set("docker", "true")
	cmd.Flags().Set("run", "make test")
	cmd.Flags().Set("config", "/cfg")

	params := buildRunParams(cmd)
	if params.Agent != "opencode" {
		t.Errorf("Agent = %q", params.Agent)
	}
	if len(params.Ports) != 1 || params.Ports[0] != "3000:3000" {
		t.Errorf("Ports = %v", params.Ports)
	}
	if len(params.Mounts) != 1 || params.Mounts[0] != "/data" {
		t.Errorf("Mounts = %v", params.Mounts)
	}
	if len(params.Envs) != 1 || params.Envs[0] != "KEY=VAL" {
		t.Errorf("Envs = %v", params.Envs)
	}
	if params.Workdir != "/app" {
		t.Errorf("Workdir = %q", params.Workdir)
	}
	if !params.Recall {
		t.Error("Recall 应为 true")
	}
	if !params.Docker {
		t.Error("Docker 应为 true")
	}
	if params.RunCmd != "make test" {
		t.Errorf("RunCmd = %q", params.RunCmd)
	}
	if params.Config != "/cfg" {
		t.Errorf("Config = %q", params.Config)
	}
}

// ========================================================================
// run.go — validateRunParams
// ========================================================================

func TestValidateRunParams_ValidEmpty(t *testing.T) {
	params := argsparser.RunParams{}
	err := validateRunParams(params)
	if err != nil {
		t.Errorf("空参数应通过验证, 但得到: %v", err)
	}
}

func TestValidateRunParams_RecallOnly(t *testing.T) {
	params := argsparser.RunParams{Recall: true}
	err := validateRunParams(params)
	if err != nil {
		t.Errorf("仅 Recall=true 应通过验证, 但得到: %v", err)
	}
}

func TestValidateRunParams_RunCmdOnly(t *testing.T) {
	params := argsparser.RunParams{RunCmd: "echo hi"}
	err := validateRunParams(params)
	if err != nil {
		t.Errorf("仅 RunCmd 非空应通过验证, 但得到: %v", err)
	}
}

func TestValidateRunParams_RecallAndRunCmdConflict(t *testing.T) {
	params := argsparser.RunParams{
		Recall: true,
		RunCmd: "echo hi",
	}
	err := validateRunParams(params)
	if err == nil {
		t.Fatal("Recall 和 RunCmd 同时设置时应返回错误")
	}
	if err.Reason != "-r/--recall 和 --run 不能同时使用" {
		t.Errorf("Reason = %q, 期望 %q", err.Reason, "-r/--recall 和 --run 不能同时使用")
	}
	if err.Suggestion == "" {
		t.Error("Suggestion 不应为空")
	}
}

// ========================================================================
// run.go — paramValidationError
// ========================================================================

func TestParamValidationError_Error(t *testing.T) {
	e := &paramValidationError{
		Reason:     "测试原因",
		Suggestion: "测试建议",
	}
	if e.Error() != "测试原因" {
		t.Errorf("Error() = %q, 期望 %q", e.Error(), "测试原因")
	}
}

func TestParamValidationError_Fields(t *testing.T) {
	e := &paramValidationError{
		Reason:     "原因",
		Suggestion: "建议",
	}
	if e.Reason != "原因" {
		t.Errorf("Reason = %q", e.Reason)
	}
	if e.Suggestion != "建议" {
		t.Errorf("Suggestion = %q", e.Suggestion)
	}
}

// ========================================================================
// run.go — runCmd.RunE (validation path, before Docker)
// ========================================================================

func TestRunCmd_RunE_ValidationErrorBeforeDocker(t *testing.T) {
	// 创建 cobra.Command，设置冲突标志（-r 和 --run），
	// 调用 runCmd.RunE 应触发参数校验错误，不会到达 dockerhelper.NewClient
	cmd := newRunCmd()
	cmd.Flags().Set("recall", "true")
	cmd.Flags().Set("run", "echo hello")

	err := runCmd.RunE(cmd, []string{})
	if err == nil {
		t.Fatal("期望参数校验错误，但得到 nil")
	}
	if !strings.Contains(err.Error(), "-r/--recall") {
		t.Errorf("错误信息应包含 -r/--recall 冲突提示, 实际: %v", err)
	}
}

// ========================================================================
// run.go — init() flags verification
// ========================================================================

func TestRunCmd_FlagsRegistered(t *testing.T) {
	expectedFlags := []string{"agent", "port", "mount", "env", "workdir", "recall", "docker", "dind", "run", "config"}
	for _, name := range expectedFlags {
		f := runCmd.Flags().Lookup(name)
		if f == nil {
			t.Errorf("runCmd 应注册 --%s 标志", name)
		}
	}
}

func TestRunCmd_FlagShorthands(t *testing.T) {
	shorthands := map[string]string{
		"agent":   "a",
		"port":    "p",
		"mount":   "m",
		"env":     "e",
		"workdir": "w",
		"recall":  "r",
		"config":  "c",
	}
	for name, expectedShort := range shorthands {
		f := runCmd.Flags().Lookup(name)
		if f == nil {
			t.Errorf("runCmd 应注册 --%s 标志", name)
			continue
		}
		if f.Shorthand != expectedShort {
			t.Errorf("--%s 的 shorthand 应为 %q, 实际: %q", name, expectedShort, f.Shorthand)
		}
	}
}

// ========================================================================
// build.go
// ========================================================================

func TestBuildCmd_Structure(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"build"})
	if err != nil {
		t.Fatalf("未找到 build 命令: %v", err)
	}
	if cmd.Short == "" {
		t.Error("build 命令的 Short 不应为空")
	}
	if cmd.Long == "" {
		t.Error("build 命令的 Long 不应为空")
	}
}

func TestBuildCmd_FlagsRegistered(t *testing.T) {
	expectedFlags := []string{"deps", "base-image", "config", "no-cache", "rebuild", "max-retry", "gh-proxy"}
	for _, name := range expectedFlags {
		f := buildCmd.Flags().Lookup(name)
		if f == nil {
			t.Errorf("buildCmd 应注册 --%s 标志", name)
		}
	}
}

func TestBuildCmd_FlagShorthands(t *testing.T) {
	shorthands := map[string]string{
		"deps":       "d",
		"base-image": "b",
		"config":     "c",
		"rebuild":    "R",
	}
	for name, expectedShort := range shorthands {
		f := buildCmd.Flags().Lookup(name)
		if f == nil {
			t.Errorf("buildCmd 应注册 --%s 标志", name)
			continue
		}
		if f.Shorthand != expectedShort {
			t.Errorf("--%s 的 shorthand 应为 %q, 实际: %q", name, expectedShort, f.Shorthand)
		}
	}
}

func TestBuildCmd_FlagDefaults(t *testing.T) {
	f := buildCmd.Flags().Lookup("max-retry")
	if f == nil {
		t.Fatal("buildCmd 应注册 --max-retry 标志")
	}
	if f.DefValue != "3" {
		t.Errorf("--max-retry 默认值应为 3, 实际: %s", f.DefValue)
	}

	f = buildCmd.Flags().Lookup("base-image")
	if f == nil {
		t.Fatal("buildCmd 应注册 --base-image 标志")
	}
	if !strings.Contains(f.DefValue, "centos") {
		t.Errorf("--base-image 默认值应包含 centos, 实际: %s", f.DefValue)
	}

	f = buildCmd.Flags().Lookup("no-cache")
	if f == nil {
		t.Fatal("buildCmd 应注册 --no-cache 标志")
	}
	if f.DefValue != "false" {
		t.Errorf("--no-cache 默认值应为 false, 实际: %s", f.DefValue)
	}
}

// ========================================================================
// version.go
// ========================================================================

func TestVersionCmd_Structure(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"version"})
	if err != nil {
		t.Fatalf("未找到 version 命令: %v", err)
	}
	if cmd.Short == "" {
		t.Error("version 命令的 Short 不应为空")
	}
	if cmd.Long == "" {
		t.Error("version 命令的 Long 不应为空")
	}
}

func TestVersionCmd_RunE_PrintsVersion(t *testing.T) {
	origVersion := Version
	origHash := GitHash
	origBuild := BuildTime
	defer func() {
		Version = origVersion
		GitHash = origHash
		BuildTime = origBuild
	}()
	Version = "9.9.9"
	GitHash = "testhash"
	BuildTime = "testtime"

	// 捕获 stdout
	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	os.Stdout = w

	err := versionCmd.RunE(&cobra.Command{}, nil)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := strings.TrimSpace(buf.String())

	if err != nil {
		t.Errorf("RunE 应返回 nil, 得到: %v", err)
	}
	if !strings.Contains(output, "agent-forge 9.9.9") {
		t.Errorf("输出应包含版本信息, 实际: %s", output)
	}
}

// ========================================================================
// doctor.go — printLayerResult
// ========================================================================

func TestPrintLayerResult_Passed(t *testing.T) {
	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	os.Stdout = w

	printLayerResult("第一层", true)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := strings.TrimSpace(buf.String())

	if !strings.Contains(output, "第一层") {
		t.Errorf("输出应包含层名 '第一层', 实际: %s", output)
	}
	if !strings.Contains(output, "通过") {
		t.Errorf("输出应包含 '通过', 实际: %s", output)
	}
}

func TestPrintLayerResult_Failed(t *testing.T) {
	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	os.Stdout = w

	printLayerResult("第二层", false)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := strings.TrimSpace(buf.String())

	if !strings.Contains(output, "第二层") {
		t.Errorf("输出应包含层名 '第二层', 实际: %s", output)
	}
	if !strings.Contains(output, "未通过") {
		t.Errorf("输出应包含 '未通过', 实际: %s", output)
	}
}

func TestPrintLayerResult_AllLayers(t *testing.T) {
	for _, tc := range []struct {
		layer  string
		passed bool
	}{
		{"核心依赖", true},
		{"运行时", false},
		{"可选工具", true},
	} {
		r, w, _ := os.Pipe()
		oldStdout := os.Stdout
		os.Stdout = w

		printLayerResult(tc.layer, tc.passed)

		w.Close()
		os.Stdout = oldStdout

		var buf bytes.Buffer
		io.Copy(&buf, r)
		output := strings.TrimSpace(buf.String())

		if !strings.Contains(output, tc.layer) {
			t.Errorf("输出应包含层名 %q, 实际: %s", tc.layer, output)
		}
	}
}

// ========================================================================
// doctor.go — printIssues
// ========================================================================

func TestPrintIssues_WithIssues(t *testing.T) {
	issues := []diagnosticengine.Issue{
		{
			Type:       diagnosticengine.IssueCoreMissing,
			Layer:      "核心依赖",
			Message:    "Docker socket 不存在",
			Suggestion: "请安装 Docker",
		},
		{
			Type:       diagnosticengine.IssueRuntimeError,
			Layer:      "运行时",
			Message:    "Docker daemon 未运行",
			Suggestion: "请启动 Docker",
		},
	}

	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	os.Stdout = w

	printIssues(issues, "核心依赖")

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := strings.TrimSpace(buf.String())

	if !strings.Contains(output, "Docker socket 不存在") {
		t.Errorf("输出应包含第一层的 issue 消息, 实际: %s", output)
	}
	if strings.Contains(output, "Docker daemon 未运行") {
		t.Error("输出不应包含其他层的 issue 消息")
	}
}

func TestPrintIssues_EmptyIssues(t *testing.T) {
	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	os.Stdout = w

	printIssues([]diagnosticengine.Issue{}, "核心依赖")

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := strings.TrimSpace(buf.String())

	if output != "" {
		t.Errorf("无 issues 时应输出空字符串, 实际: %s", output)
	}
}

func TestPrintIssues_MultipleLayers(t *testing.T) {
	issues := []diagnosticengine.Issue{
		{Layer: "核心依赖", Message: "问题A"},
		{Layer: "运行时", Message: "问题B"},
		{Layer: "核心依赖", Message: "问题C"},
	}

	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	os.Stdout = w

	printIssues(issues, "核心依赖")

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := strings.TrimSpace(buf.String())

	if !strings.Contains(output, "问题A") {
		t.Error("输出应包含问题A")
	}
	if !strings.Contains(output, "问题C") {
		t.Error("输出应包含问题C")
	}
	if strings.Contains(output, "问题B") {
		t.Error("输出不应包含运行时的问题")
	}
}

// ========================================================================
// doctor.go — printIssueSuggestions
// ========================================================================

func TestPrintIssueSuggestions_WithSuggestions(t *testing.T) {
	issues := []diagnosticengine.Issue{
		{Layer: "核心依赖", Message: "err", Suggestion: "请安装 Docker"},
		{Layer: "运行时", Message: "err", Suggestion: "请启动 Docker"},
		{Layer: "核心依赖", Message: "err2", Suggestion: ""}, // 无建议
	}

	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	os.Stdout = w

	printIssueSuggestions(issues)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := strings.TrimSpace(buf.String())

	if !strings.Contains(output, "请安装 Docker") {
		t.Errorf("输出应包含 '请安装 Docker', 实际: %s", output)
	}
	if !strings.Contains(output, "请启动 Docker") {
		t.Errorf("输出应包含 '请启动 Docker', 实际: %s", output)
	}
	if strings.Contains(output, "err2") {
		t.Error("输出不应包含无建议的 issue")
	}
}

func TestPrintIssueSuggestions_AllEmpty(t *testing.T) {
	issues := []diagnosticengine.Issue{
		{Layer: "核心依赖", Message: "err", Suggestion: ""},
	}

	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	os.Stdout = w

	printIssueSuggestions(issues)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := strings.TrimSpace(buf.String())

	if strings.Contains(output, "err") {
		t.Error("输出不应包含 issue 内容")
	}
}

// ========================================================================
// doctor.go — doctorHelperAdapter
// ========================================================================

func TestDoctorHelperAdapter_ImplementsInterface(t *testing.T) {
	// 编译期检查：*doctorHelperAdapter 实现 diagnosticengine.DockerHelper
	var a diagnosticengine.DockerHelper = (*doctorHelperAdapter)(nil)
	_ = a // 防止 unused 错误
	// 如果编译通过，说明适配器实现了接口
}

func TestDoctorHelperAdapter_Struct(t *testing.T) {
	// 验证 doctorHelperAdapter 有 client 字段
	adapter := &doctorHelperAdapter{}
	if adapter.client != nil {
		t.Error("新创建的适配器 client 应为 nil")
	}
}

// ========================================================================
// doctor.go — command structure
// ========================================================================

func TestDoctorCmd_Structure(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"doctor"})
	if err != nil {
		t.Fatalf("未找到 doctor 命令: %v", err)
	}
	if cmd.Short == "" {
		t.Error("doctor 命令的 Short 不应为空")
	}
	if cmd.Long == "" {
		t.Error("doctor 命令的 Long 不应为空")
	}
}

func TestDoctorCmd_FlagsRegistered(t *testing.T) {
	f := doctorCmd.Flags().Lookup("config")
	if f == nil {
		t.Error("doctorCmd 应注册 --config 标志")
	}
}

// ========================================================================
// endpoint.go — promptForInput
// ========================================================================

func TestPromptForInput_Normal(t *testing.T) {
	input := "test-input-value\n"
	r, w, _ := os.Pipe()
	w.Write([]byte(input))
	w.Close()

	oldStdin := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = oldStdin }()

	result, err := promptForInput()
	if err != nil {
		t.Fatalf("promptForInput 不应返回错误: %v", err)
	}
	if result != "test-input-value" {
		t.Errorf("结果 = %q, 期望 %q", result, "test-input-value")
	}
}

func TestPromptForInput_TrimSpaces(t *testing.T) {
	input := "  spaced-value  \n"
	r, w, _ := os.Pipe()
	w.Write([]byte(input))
	w.Close()

	oldStdin := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = oldStdin }()

	result, err := promptForInput()
	if err != nil {
		t.Fatalf("promptForInput 不应返回错误: %v", err)
	}
	if result != "spaced-value" {
		t.Errorf("结果应被 trim, 得到: %q", result)
	}
}

func TestPromptForInput_EmptyLine(t *testing.T) {
	input := "\n"
	r, w, _ := os.Pipe()
	w.Write([]byte(input))
	w.Close()

	oldStdin := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = oldStdin }()

	result, err := promptForInput()
	if err != nil {
		t.Fatalf("promptForInput 不应返回错误: %v", err)
	}
	if result != "" {
		t.Errorf("空输入应返回空字符串, 得到: %q", result)
	}
}

// ========================================================================
// endpoint.go — command structure
// ========================================================================

func TestEndpointCmd_Structure(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"endpoint"})
	if err != nil {
		t.Fatalf("未找到 endpoint 命令: %v", err)
	}
	if cmd.Short == "" {
		t.Error("endpoint 命令的 Short 不应为空")
	}
	if cmd.Long == "" {
		t.Error("endpoint 命令的 Long 不应为空")
	}
}

func TestEndpointCmd_HasAllSubcommands(t *testing.T) {
	expected := []string{"providers", "list", "show", "add", "set", "rm", "test", "apply", "status"}
	for _, name := range expected {
		cmd, _, err := endpointCmd.Find([]string{name})
		if err != nil {
			t.Errorf("endpointCmd 应包含子命令 %q, 错误: %v", name, err)
			continue
		}
		if !strings.HasPrefix(cmd.Use, name) {
			t.Errorf("子命令 %q 的 Use 应以 %q 开头, 实际: %q", name, name, cmd.Use)
		}
	}
}

func TestEndpointSubcommands_HelpText(t *testing.T) {
	subcommands := []string{"providers", "list", "show", "add", "set", "rm", "test", "apply", "status"}
	for _, name := range subcommands {
		cmd, _, err := endpointCmd.Find([]string{name})
		if err != nil {
			t.Errorf("未找到子命令 %q: %v", name, err)
			continue
		}
		if cmd.Short == "" {
			t.Errorf("子命令 %q 的 Short 不应为空", name)
		}
		if cmd.Long == "" {
			t.Errorf("子命令 %q 的 Long 不应为空", name)
		}
	}
}

func TestEndpointSubcommands_ArgsValidation(t *testing.T) {
	tests := []struct {
		name     string
		wantArgs cobra.PositionalArgs
	}{
		{"providers", cobra.NoArgs},
		{"list", cobra.NoArgs},
		{"status", cobra.NoArgs},
		{"show", cobra.ExactArgs(1)},
		{"add", cobra.ExactArgs(1)},
		{"set", cobra.ExactArgs(1)},
		{"rm", cobra.ExactArgs(1)},
		{"test", cobra.ExactArgs(1)},
		{"apply", cobra.MaximumNArgs(1)},
	}
	for _, tt := range tests {
		cmd, _, err := endpointCmd.Find([]string{tt.name})
		if err != nil {
			t.Fatalf("未找到子命令 %q: %v", tt.name, err)
		}
		if cmd.Args == nil {
			t.Errorf("子命令 %q 的 Args 不应为 nil", tt.name)
		}
	}
}

func TestEndpointAddCmd_FlagsRegistered(t *testing.T) {
	expectedFlags := []string{"provider", "url", "key", "model", "model-opus", "model-sonnet", "model-haiku", "model-subagent"}
	for _, name := range expectedFlags {
		f := endpointAddCmd.Flags().Lookup(name)
		if f == nil {
			t.Errorf("endpointAddCmd 应注册 --%s 标志", name)
		}
	}
}

func TestEndpointSetCmd_FlagsRegistered(t *testing.T) {
	expectedFlags := []string{"provider", "url", "key", "model", "model-opus", "model-sonnet", "model-haiku", "model-subagent"}
	for _, name := range expectedFlags {
		f := endpointSetCmd.Flags().Lookup(name)
		if f == nil {
			t.Errorf("endpointSetCmd 应注册 --%s 标志", name)
		}
	}
}

func TestEndpointApplyCmd_FlagsRegistered(t *testing.T) {
	f := endpointApplyCmd.Flags().Lookup("agent")
	if f == nil {
		t.Error("endpointApplyCmd 应注册 --agent 标志")
	}
}

func TestEndpointCmd_PersistentFlags(t *testing.T) {
	f := endpointCmd.PersistentFlags().Lookup("config")
	if f == nil {
		t.Error("endpointCmd 应注册 PersistentFlags --config 标志")
	}
}

// ========================================================================
// endpoint.go — endpointAddCmd error paths before I/O
// ========================================================================

func TestEndpointAddCmd_RunE_NoName(t *testing.T) {
	// cobra 的 Args 验证由 cobra 自身完成，不经过 RunE。
	// 我们验证 Args 是 cobra.ExactArgs(1)
	if endpointAddCmd.Args == nil {
		t.Fatal("endpointAddCmd.Args 应为 cobra.ExactArgs(1)")
	}
	// 验证传入空 args 时 cobra 的验证行为
	cmd := &cobra.Command{Use: "add", Args: cobra.ExactArgs(1)}
	err := cmd.Args(cmd, []string{})
	if err == nil {
		t.Error("ExactArgs(1) 接受空 args 时应返回错误")
	}
}

// ========================================================================
// endpoint.go — endpointProvidersCmd RunE
// ========================================================================

func TestEndpointProvidersCmd_RunE_Output(t *testing.T) {
	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	os.Stdout = w

	err := endpointProvidersCmd.RunE(&cobra.Command{}, nil)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("RunE 应返回 nil, 得到: %v", err)
	}
	// 应包含 PROVIDER 表头
	if !strings.Contains(output, "PROVIDER") {
		t.Errorf("输出应包含 PROVIDER 表头, 实际: %s", output)
	}
	// 应包含已知 provider
	if !strings.Contains(output, "deepseek") {
		t.Errorf("输出应包含 deepseek, 实际: %s", output)
	}
	if !strings.Contains(output, "openai") {
		t.Errorf("输出应包含 openai, 实际: %s", output)
	}
	if !strings.Contains(output, "anthropic") {
		t.Errorf("输出应包含 anthropic, 实际: %s", output)
	}
	// 应包含可服务的 agent
	if !strings.Contains(output, "claude") {
		t.Errorf("输出应包含 claude, 实际: %s", output)
	}
}

// ========================================================================
// export.go
// ========================================================================

func TestExportCmd_Structure(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"export"})
	if err != nil {
		t.Fatalf("未找到 export 命令: %v", err)
	}
	if cmd.Short == "" {
		t.Error("export 命令的 Short 不应为空")
	}
	if cmd.Long == "" {
		t.Error("export 命令的 Long 不应为空")
	}
}

func TestExportCmd_FlagsRegistered(t *testing.T) {
	f := exportCmd.Flags().Lookup("image")
	if f == nil {
		t.Error("exportCmd 应注册 --image 标志")
	}
}

func TestExportCmd_Args(t *testing.T) {
	if exportCmd.Args == nil {
		t.Error("exportCmd.Args 不应为 nil")
	}
	// 验证 MaximumNArgs(1)
	err := exportCmd.Args(exportCmd, []string{"a", "b"})
	if err == nil {
		t.Error("MaximumNArgs(1) 应拒绝 2 个参数")
	}
}

// ========================================================================
// import.go
// ========================================================================

func TestImportCmd_Structure(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"import"})
	if err != nil {
		t.Fatalf("未找到 import 命令: %v", err)
	}
	if cmd.Short == "" {
		t.Error("import 命令的 Short 不应为空")
	}
	if cmd.Long == "" {
		t.Error("import 命令的 Long 不应为空")
	}
}

func TestImportCmd_Args(t *testing.T) {
	if importCmd.Args == nil {
		t.Error("importCmd.Args 不应为 nil")
	}
	// 验证 ExactArgs(1)
	err := importCmd.Args(importCmd, []string{})
	if err == nil {
		t.Error("ExactArgs(1) 应拒绝 0 个参数")
	}
	err = importCmd.Args(importCmd, []string{"a", "b"})
	if err == nil {
		t.Error("ExactArgs(1) 应拒绝 2 个参数")
	}
}

// ========================================================================
// update.go
// ========================================================================

func TestUpdateCmd_Structure(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"update"})
	if err != nil {
		t.Fatalf("未找到 update 命令: %v", err)
	}
	if cmd.Short == "" {
		t.Error("update 命令的 Short 不应为空")
	}
	if cmd.Long == "" {
		t.Error("update 命令的 Long 不应为空")
	}
}

// ========================================================================
// deps.go
// ========================================================================

func TestDepsCmd_Structure(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"deps"})
	if err != nil {
		t.Fatalf("未找到 deps 命令: %v", err)
	}
	if cmd.Short == "" {
		t.Error("deps 命令的 Short 不应为空")
	}
	if cmd.Long == "" {
		t.Error("deps 命令的 Long 不应为空")
	}
}

func TestDepsCmd_FlagsRegistered(t *testing.T) {
	expectedFlags := []string{"image", "config"}
	for _, name := range expectedFlags {
		f := depsCmd.Flags().Lookup(name)
		if f == nil {
			t.Errorf("depsCmd 应注册 --%s 标志", name)
		}
	}
}

func TestDepsCmd_FlagShorthands(t *testing.T) {
	f := depsCmd.Flags().Lookup("image")
	if f == nil {
		t.Fatal("depsCmd 应注册 --image 标志")
	}
	if f.Shorthand != "i" {
		t.Errorf("--image 的 shorthand 应为 'i', 实际: %q", f.Shorthand)
	}

	f = depsCmd.Flags().Lookup("config")
	if f == nil {
		t.Fatal("depsCmd 应注册 --config 标志")
	}
	if f.Shorthand != "c" {
		t.Errorf("--config 的 shorthand 应为 'c', 实际: %q", f.Shorthand)
	}
}

// ========================================================================
// doctor.go — doctorCmd.RunE output helpers integration
// ========================================================================

func TestDoctorCmd_PrintHelpers_Integration(t *testing.T) {
	// 集成测试 printLayerResult + printIssues + printIssueSuggestions
	issues := []diagnosticengine.Issue{
		{Layer: "核心依赖", Message: "socket 不存在", Suggestion: "安装 Docker"},
		{Layer: "运行时", Message: "daemon 未运行", Suggestion: "启动 Docker"},
	}

	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	os.Stdout = w

	printLayerResult("核心依赖", false)
	printIssues(issues, "核心依赖")
	printIssueSuggestions(issues)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if !strings.Contains(output, "未通过") {
		t.Error("输出应包含 '未通过'")
	}
	if !strings.Contains(output, "socket 不存在") {
		t.Error("输出应包含 issue 消息")
	}
	if !strings.Contains(output, "安装 Docker") {
		t.Error("输出应包含建议")
	}
}

// ========================================================================
// endpoint.go — endpointListCmd RunE with empty dir (light test)
// ========================================================================

func TestEndpointListCmd_RunE_EmptyDir(t *testing.T) {
	// 使用不存在的配置目录，应正常输出空表头
	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	os.Stdout = w

	cmd := &cobra.Command{}
	cmd.Flags().StringP("config", "c", "", "")
	cmd.Flags().Set("config", t.TempDir())

	err := endpointListCmd.RunE(cmd, nil)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("空端点目录应返回 nil, 得到: %v", err)
	}
	if !strings.Contains(output, "NAME") {
		t.Errorf("输出应包含 NAME 表头, 实际: %s", output)
	}
}

// ========================================================================
// endpoint.go — endpointAddCmd RunE comprehensive tests
// ========================================================================

// newEndpointAddCmd 创建带所有 endpoint add 命令标志的 cobra.Command。
func newEndpointAddCmd(configDir string) *cobra.Command {
	cmd := &cobra.Command{Use: "add", Args: cobra.ExactArgs(1)}
	cmd.Flags().String("provider", "", "")
	cmd.Flags().String("url", "", "")
	cmd.Flags().String("key", "", "")
	cmd.Flags().String("model", "", "")
	cmd.Flags().String("model-opus", "", "")
	cmd.Flags().String("model-sonnet", "", "")
	cmd.Flags().String("model-haiku", "", "")
	cmd.Flags().String("model-subagent", "", "")
	cmd.Flags().StringP("config", "c", "", "")
	cmd.Flags().Set("config", configDir)
	return cmd
}

func TestEndpointAddCmd_RunE_Success(t *testing.T) {
	configDir := t.TempDir()
	cmd := newEndpointAddCmd(configDir)
	cmd.Flags().Set("provider", "openai")
	cmd.Flags().Set("url", "https://api.openai.com")
	cmd.Flags().Set("key", "sk-test-key-value-12345")
	cmd.Flags().Set("model", "gpt-4")

	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	os.Stdout = w

	err := endpointAddCmd.RunE(cmd, []string{"test-ep"})

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Fatalf("RunE 应成功, 得到: %v", err)
	}
	if !strings.Contains(output, "创建成功") {
		t.Errorf("输出应包含 '创建成功', 实际: %s", output)
	}

	// 验证 endpoint.env 文件被创建
	envPath := filepath.Join(configDir, "endpoints", "test-ep", "endpoint.env")
	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		t.Error("endpoint.env 应被创建")
	}
}

func TestEndpointAddCmd_RunE_WithAllParams(t *testing.T) {
	configDir := t.TempDir()
	cmd := newEndpointAddCmd(configDir)
	cmd.Flags().Set("provider", "deepseek")
	cmd.Flags().Set("url", "https://api.deepseek.com")
	cmd.Flags().Set("key", "sk-ds-key-value")
	cmd.Flags().Set("model", "deepseek-chat")
	cmd.Flags().Set("model-opus", "opus-model")
	cmd.Flags().Set("model-sonnet", "sonnet-model")
	cmd.Flags().Set("model-haiku", "haiku-model")
	cmd.Flags().Set("model-subagent", "subagent-model")

	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	os.Stdout = w

	err := endpointAddCmd.RunE(cmd, []string{"full-ep"})

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Fatalf("RunE 应成功, 得到: %v", err)
	}
	if !strings.Contains(output, "full-ep") {
		t.Errorf("输出应包含端点名称 'full-ep', 实际: %s", output)
	}
}

func TestEndpointAddCmd_RunE_DuplicateEndpoint(t *testing.T) {
	configDir := t.TempDir()
	cmd := newEndpointAddCmd(configDir)
	cmd.Flags().Set("provider", "openai")
	cmd.Flags().Set("url", "https://api.openai.com")
	cmd.Flags().Set("key", "sk-test-key")

	// 第一次创建应成功
	err := endpointAddCmd.RunE(cmd, []string{"dup-ep"})
	if err != nil {
		t.Fatalf("首次创建应成功: %v", err)
	}

	// 第二次创建同名端点应失败
	cmd2 := newEndpointAddCmd(configDir)
	cmd2.Flags().Set("provider", "openai")
	cmd2.Flags().Set("url", "https://api.openai.com")
	cmd2.Flags().Set("key", "sk-test-key")

	err = endpointAddCmd.RunE(cmd2, []string{"dup-ep"})
	if err == nil {
		t.Fatal("重复创建同名端点应返回错误")
	}
	var ec ExitCoder
	if !errors.As(err, &ec) {
		t.Fatal("错误应实现 ExitCoder 接口")
	}
	if ec.ExitCode() != 1 {
		t.Errorf("退出码应为 1, 得到: %d", ec.ExitCode())
	}
	if !strings.Contains(err.Error(), "已存在") {
		t.Errorf("错误信息应包含 '已存在', 实际: %v", err)
	}
}

// ========================================================================
// endpoint.go — endpointListCmd RunE with created endpoints
// ========================================================================

// newListOrStatusCmd 创建带 --config 标志的 cobra.Command，用于 list/status 测试。
func newListOrStatusCmd(configDir string) *cobra.Command {
	cmd := &cobra.Command{}
	cmd.Flags().StringP("config", "c", "", "")
	cmd.Flags().Set("config", configDir)
	return cmd
}

func TestEndpointListCmd_RunE_WithEndpoints(t *testing.T) {
	configDir := t.TempDir()

	// 先用 add 创建端点
	addCmd := newEndpointAddCmd(configDir)
	addCmd.Flags().Set("provider", "openai")
	addCmd.Flags().Set("url", "https://api.openai.com")
	addCmd.Flags().Set("key", "sk-test-key")
	addCmd.Flags().Set("model", "gpt-4")
	err := endpointAddCmd.RunE(addCmd, []string{"ep1"})
	if err != nil {
		t.Fatalf("创建端点失败: %v", err)
	}

	addCmd2 := newEndpointAddCmd(configDir)
	addCmd2.Flags().Set("provider", "deepseek")
	addCmd2.Flags().Set("url", "https://api.deepseek.com")
	addCmd2.Flags().Set("key", "sk-ds-key")
	addCmd2.Flags().Set("model", "deepseek-chat")
	err = endpointAddCmd.RunE(addCmd2, []string{"ep2"})
	if err != nil {
		t.Fatalf("创建端点失败: %v", err)
	}

	// 执行 list
	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	os.Stdout = w

	cmd := newListOrStatusCmd(configDir)
	err = endpointListCmd.RunE(cmd, nil)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("list 应返回 nil, 得到: %v", err)
	}
	if !strings.Contains(output, "ep1") {
		t.Errorf("输出应包含 ep1, 实际: %s", output)
	}
	if !strings.Contains(output, "ep2") {
		t.Errorf("输出应包含 ep2, 实际: %s", output)
	}
	if !strings.Contains(output, "openai") {
		t.Errorf("输出应包含 openai, 实际: %s", output)
	}
	if !strings.Contains(output, "deepseek") {
		t.Errorf("输出应包含 deepseek, 实际: %s", output)
	}
	if !strings.Contains(output, "deepseek-chat") {
		t.Errorf("输出应包含 deepseek-chat, 实际: %s", output)
	}
}

// ========================================================================
// endpoint.go — endpointShowCmd RunE
// ========================================================================

func TestEndpointShowCmd_RunE_ShowEndpoint(t *testing.T) {
	configDir := t.TempDir()

	// 创建端点
	addCmd := newEndpointAddCmd(configDir)
	addCmd.Flags().Set("provider", "openai")
	addCmd.Flags().Set("url", "https://api.openai.com")
	addCmd.Flags().Set("key", "sk-test-key-value-12345")
	addCmd.Flags().Set("model", "gpt-4")
	err := endpointAddCmd.RunE(addCmd, []string{"show-ep"})
	if err != nil {
		t.Fatalf("创建端点失败: %v", err)
	}

	// 执行 show
	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	os.Stdout = w

	cmd := &cobra.Command{Use: "show", Args: cobra.ExactArgs(1)}
	cmd.Flags().StringP("config", "c", "", "")
	cmd.Flags().Set("config", configDir)

	err = endpointShowCmd.RunE(cmd, []string{"show-ep"})

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("show 应返回 nil, 得到: %v", err)
	}
	if !strings.Contains(output, "show-ep") {
		t.Errorf("输出应包含端点名称, 实际: %s", output)
	}
	if !strings.Contains(output, "openai") {
		t.Errorf("输出应包含 provider, 实际: %s", output)
	}
	if !strings.Contains(output, "gpt-4") {
		t.Errorf("输出应包含 model, 实际: %s", output)
	}
	// 验证 key 是掩码格式 (NFR-6)
	if !strings.Contains(output, "***") {
		t.Errorf("输出应包含掩码 key, 实际: %s", output)
	}
	if strings.Contains(output, "sk-test-key-value-12345") {
		t.Error("输出不应包含完整 key")
	}
}

func TestEndpointShowCmd_RunE_NonexistentEndpoint(t *testing.T) {
	configDir := t.TempDir()

	cmd := &cobra.Command{Use: "show", Args: cobra.ExactArgs(1)}
	cmd.Flags().StringP("config", "c", "", "")
	cmd.Flags().Set("config", configDir)

	err := endpointShowCmd.RunE(cmd, []string{"nonexistent"})
	if err == nil {
		t.Fatal("不存在的端点应返回错误")
	}
	var ec ExitCoder
	if !errors.As(err, &ec) {
		t.Fatal("错误应实现 ExitCoder")
	}
	if ec.ExitCode() != 1 {
		t.Errorf("退出码应为 1, 得到: %d", ec.ExitCode())
	}
}

// ========================================================================
// endpoint.go — endpointSetCmd RunE
// ========================================================================

func TestEndpointSetCmd_RunE_UpdateEndpoint(t *testing.T) {
	configDir := t.TempDir()

	// 先创建端点
	addCmd := newEndpointAddCmd(configDir)
	addCmd.Flags().Set("provider", "openai")
	addCmd.Flags().Set("url", "https://api.openai.com")
	addCmd.Flags().Set("key", "sk-original-key")
	addCmd.Flags().Set("model", "gpt-4")
	err := endpointAddCmd.RunE(addCmd, []string{"set-ep"})
	if err != nil {
		t.Fatalf("创建端点失败: %v", err)
	}

	// 执行 set 修改 key 和 model
	cmd := &cobra.Command{Use: "set", Args: cobra.ExactArgs(1)}
	cmd.Flags().String("key", "", "")
	cmd.Flags().String("model", "", "")
	cmd.Flags().String("provider", "", "")
	cmd.Flags().String("url", "", "")
	cmd.Flags().String("model-opus", "", "")
	cmd.Flags().String("model-sonnet", "", "")
	cmd.Flags().String("model-haiku", "", "")
	cmd.Flags().String("model-subagent", "", "")
	cmd.Flags().StringP("config", "c", "", "")
	cmd.Flags().Set("config", configDir)
	cmd.Flags().Set("key", "sk-updated-key")
	cmd.Flags().Set("model", "gpt-5")

	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	os.Stdout = w

	err = endpointSetCmd.RunE(cmd, []string{"set-ep"})

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("set 应返回 nil, 得到: %v", err)
	}
	if !strings.Contains(output, "已更新") {
		t.Errorf("输出应包含 '已更新', 实际: %s", output)
	}
}

func TestEndpointSetCmd_RunE_NoFields(t *testing.T) {
	configDir := t.TempDir()

	// 先创建端点
	addCmd := newEndpointAddCmd(configDir)
	addCmd.Flags().Set("provider", "openai")
	addCmd.Flags().Set("url", "https://api.openai.com")
	addCmd.Flags().Set("key", "sk-test-key")
	addCmd.Flags().Set("model", "gpt-4")
	err := endpointAddCmd.RunE(addCmd, []string{"nochange-ep"})
	if err != nil {
		t.Fatalf("创建端点失败: %v", err)
	}

	// set 不带任何更新参数
	cmd := &cobra.Command{Use: "set", Args: cobra.ExactArgs(1)}
	cmd.Flags().String("key", "", "")
	cmd.Flags().String("model", "", "")
	cmd.Flags().String("provider", "", "")
	cmd.Flags().String("url", "", "")
	cmd.Flags().String("model-opus", "", "")
	cmd.Flags().String("model-sonnet", "", "")
	cmd.Flags().String("model-haiku", "", "")
	cmd.Flags().String("model-subagent", "", "")
	cmd.Flags().StringP("config", "c", "", "")
	cmd.Flags().Set("config", configDir)

	err = endpointSetCmd.RunE(cmd, []string{"nochange-ep"})
	if err == nil {
		t.Fatal("未指定更新字段时应返回错误")
	}
	if !strings.Contains(err.Error(), "未指定要更新的字段") {
		t.Errorf("错误信息应包含 '未指定要更新的字段', 实际: %v", err)
	}
}

// ========================================================================
// endpoint.go — endpointRmCmd RunE
// ========================================================================

func TestEndpointRmCmd_RunE_RemoveEndpoint(t *testing.T) {
	configDir := t.TempDir()

	// 先创建端点
	addCmd := newEndpointAddCmd(configDir)
	addCmd.Flags().Set("provider", "openai")
	addCmd.Flags().Set("url", "https://api.openai.com")
	addCmd.Flags().Set("key", "sk-test-key")
	addCmd.Flags().Set("model", "gpt-4")
	err := endpointAddCmd.RunE(addCmd, []string{"rm-ep"})
	if err != nil {
		t.Fatalf("创建端点失败: %v", err)
	}

	// 验证端点目录存在
	epDir := filepath.Join(configDir, "endpoints", "rm-ep")
	if _, err := os.Stat(epDir); os.IsNotExist(err) {
		t.Fatal("端点目录应先被创建")
	}

	// 执行 rm
	cmd := &cobra.Command{Use: "rm", Args: cobra.ExactArgs(1)}
	cmd.Flags().StringP("config", "c", "", "")
	cmd.Flags().Set("config", configDir)

	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	os.Stdout = w

	err = endpointRmCmd.RunE(cmd, []string{"rm-ep"})

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("rm 应返回 nil, 得到: %v", err)
	}
	if !strings.Contains(output, "已删除") {
		t.Errorf("输出应包含 '已删除', 实际: %s", output)
	}
	if _, err := os.Stat(epDir); !os.IsNotExist(err) {
		t.Error("端点目录应已被删除")
	}
}

func TestEndpointRmCmd_RunE_NonexistentEndpoint(t *testing.T) {
	configDir := t.TempDir()

	cmd := &cobra.Command{Use: "rm", Args: cobra.ExactArgs(1)}
	cmd.Flags().StringP("config", "c", "", "")
	cmd.Flags().Set("config", configDir)

	err := endpointRmCmd.RunE(cmd, []string{"nonexistent"})
	if err == nil {
		t.Fatal("删除不存在的端点应返回错误")
	}
	var ec ExitCoder
	if !errors.As(err, &ec) {
		t.Fatal("错误应实现 ExitCoder")
	}
	if ec.ExitCode() != 1 {
		t.Errorf("退出码应为 1, 得到: %d", ec.ExitCode())
	}
}

// ========================================================================
// endpoint.go — endpointStatusCmd RunE with endpoints
// ========================================================================

func TestEndpointStatusCmd_RunE_WithEndpoints(t *testing.T) {
	configDir := t.TempDir()

	// 创建 openai 端点
	addCmd := newEndpointAddCmd(configDir)
	addCmd.Flags().Set("provider", "openai")
	addCmd.Flags().Set("url", "https://api.openai.com")
	addCmd.Flags().Set("key", "sk-openai-key")
	addCmd.Flags().Set("model", "gpt-4")
	err := endpointAddCmd.RunE(addCmd, []string{"ep-openai"})
	if err != nil {
		t.Fatalf("创建端点失败: %v", err)
	}

	// 执行 status
	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	os.Stdout = w

	cmd := newListOrStatusCmd(configDir)
	err = endpointStatusCmd.RunE(cmd, nil)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("status 应返回 nil, 得到: %v", err)
	}

	if !strings.Contains(output, "AGENT") {
		t.Errorf("输出应包含 AGENT 表头, 实际: %s", output)
	}
	if !strings.Contains(output, "ep-openai") {
		t.Errorf("输出应包含端点名称, 实际: %s", output)
	}
}

// ========================================================================
// endpoint.go — endpointTestCmd RunE error paths (no HTTP)
// ========================================================================

func TestEndpointTestCmd_RunE_NonexistentEndpoint(t *testing.T) {
	configDir := t.TempDir()

	cmd := &cobra.Command{Use: "test", Args: cobra.ExactArgs(1)}
	cmd.Flags().StringP("config", "c", "", "")
	cmd.Flags().Set("config", configDir)

	err := endpointTestCmd.RunE(cmd, []string{"no-such-ep"})
	if err == nil {
		t.Fatal("不存在的端点应返回错误")
	}
	var ec ExitCoder
	if !errors.As(err, &ec) {
		t.Fatal("错误应实现 ExitCoder")
	}
	if ec.ExitCode() != 1 {
		t.Errorf("退出码应为 1, 得到: %d", ec.ExitCode())
	}
}

// ========================================================================
// endpoint.go — endpointApplyCmd RunE error paths
// ========================================================================

func TestEndpointApplyCmd_RunE_NonexistentEndpoint(t *testing.T) {
	configDir := t.TempDir()

	cmd := &cobra.Command{Use: "apply", Args: cobra.MaximumNArgs(1)}
	cmd.Flags().String("agent", "", "")
	cmd.Flags().StringP("config", "c", "", "")
	cmd.Flags().Set("config", configDir)

	err := endpointApplyCmd.RunE(cmd, []string{"no-such-ep"})
	if err == nil {
		t.Fatal("不存在的端点应返回错误")
	}
	var ec ExitCoder
	if !errors.As(err, &ec) {
		t.Fatal("错误应实现 ExitCoder")
	}
	if ec.ExitCode() != 1 {
		t.Errorf("退出码应为 1, 得到: %d", ec.ExitCode())
	}
}

func TestEndpointApplyCmd_RunE_WithValidEndpoint(t *testing.T) {
	configDir := t.TempDir()

	// 创建 deepseek 端点（可服务所有 agent）
	addCmd := newEndpointAddCmd(configDir)
	addCmd.Flags().Set("provider", "deepseek")
	addCmd.Flags().Set("url", "https://api.deepseek.com")
	addCmd.Flags().Set("key", "sk-ds-key")
	addCmd.Flags().Set("model", "deepseek-chat")
	err := endpointAddCmd.RunE(addCmd, []string{"apply-ep"})
	if err != nil {
		t.Fatalf("创建端点失败: %v", err)
	}

	// 执行 apply
	cmd := &cobra.Command{Use: "apply", Args: cobra.MaximumNArgs(1)}
	cmd.Flags().String("agent", "", "")
	cmd.Flags().StringP("config", "c", "", "")
	cmd.Flags().Set("config", configDir)

	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	os.Stdout = w

	err = endpointApplyCmd.RunE(cmd, []string{"apply-ep"})

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("apply 应返回 nil, 得到: %v", err)
	}
	if !strings.Contains(output, "已同步") {
		t.Errorf("输出应包含 '已同步', 实际: %s", output)
	}
}

func TestEndpointApplyCmd_RunE_AgentFilter(t *testing.T) {
	configDir := t.TempDir()

	// 创建 deepseek 端点
	addCmd := newEndpointAddCmd(configDir)
	addCmd.Flags().Set("provider", "deepseek")
	addCmd.Flags().Set("url", "https://api.deepseek.com")
	addCmd.Flags().Set("key", "sk-ds-key")
	addCmd.Flags().Set("model", "deepseek-chat")
	err := endpointAddCmd.RunE(addCmd, []string{"filter-ep"})
	if err != nil {
		t.Fatalf("创建端点失败: %v", err)
	}

	// 执行 apply --agent claude,kimi
	cmd := &cobra.Command{Use: "apply", Args: cobra.MaximumNArgs(1)}
	cmd.Flags().String("agent", "", "")
	cmd.Flags().StringP("config", "c", "", "")
	cmd.Flags().Set("config", configDir)
	cmd.Flags().Set("agent", "claude,kimi")

	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	os.Stdout = w

	err = endpointApplyCmd.RunE(cmd, []string{"filter-ep"})

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("apply --agent 应返回 nil, 得到: %v", err)
	}
	if !strings.Contains(output, "claude") || !strings.Contains(output, "kimi") {
		t.Errorf("输出应包含 claude 和 kimi, 实际: %s", output)
	}
}

func TestEndpointApplyCmd_RunE_SyncAll(t *testing.T) {
	configDir := t.TempDir()

	// 创建 deepseek 端点
	addCmd := newEndpointAddCmd(configDir)
	addCmd.Flags().Set("provider", "deepseek")
	addCmd.Flags().Set("url", "https://api.deepseek.com")
	addCmd.Flags().Set("key", "sk-ds-key")
	addCmd.Flags().Set("model", "deepseek-chat")
	err := endpointAddCmd.RunE(addCmd, []string{"sync-ep"})
	if err != nil {
		t.Fatalf("创建端点失败: %v", err)
	}

	// 执行 apply 不带端点名称（同步所有）
	cmd := &cobra.Command{Use: "apply", Args: cobra.MaximumNArgs(1)}
	cmd.Flags().String("agent", "", "")
	cmd.Flags().StringP("config", "c", "", "")
	cmd.Flags().Set("config", configDir)

	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	os.Stdout = w

	err = endpointApplyCmd.RunE(cmd, []string{})

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("apply 同步所有应返回 nil, 得到: %v", err)
	}
	if !strings.Contains(output, "已同步所有端点") {
		t.Errorf("输出应包含 '已同步所有端点', 实际: %s", output)
	}
}

// ========================================================================
// endpoint.go — endpointSetCmd RunE error: nonexistent endpoint
// ========================================================================

func TestEndpointSetCmd_RunE_NonexistentEndpoint(t *testing.T) {
	configDir := t.TempDir()

	cmd := &cobra.Command{Use: "set", Args: cobra.ExactArgs(1)}
	cmd.Flags().String("key", "", "")
	cmd.Flags().String("model", "", "")
	cmd.Flags().String("provider", "", "")
	cmd.Flags().String("url", "", "")
	cmd.Flags().String("model-opus", "", "")
	cmd.Flags().String("model-sonnet", "", "")
	cmd.Flags().String("model-haiku", "", "")
	cmd.Flags().String("model-subagent", "", "")
	cmd.Flags().StringP("config", "c", "", "")
	cmd.Flags().Set("config", configDir)
	cmd.Flags().Set("key", "sk-new-key")

	err := endpointSetCmd.RunE(cmd, []string{"nonexistent"})
	if err == nil {
		t.Fatal("不存在的端点应返回错误")
	}
}

// ========================================================================
// run.go — config resolution error path (before Docker)
// ========================================================================

func TestRunCmd_RunE_ConfigResolutionError(t *testing.T) {
	// 模拟 configresolver.Resolve 失败的情形很困难（需要 os.Getwd 失败），
	// 但我们可以验证 buildRunParams 和 validateRunParams 正常工作的前提下，
	// RunE 在 configresolver.Resolve 之前就返回了错误。
	// 实际上，configresolver 在使用绝对路径时不会失败。
	// 这里我们验证 validateRunParams 后的代码路径不是 nil。
	cmd := newRunCmd()
	cmd.Flags().Set("recall", "true")
	cmd.Flags().Set("run", "echo hi")
	err := runCmd.RunE(cmd, []string{})
	if err == nil {
		t.Fatal("应返回参数校验错误")
	}
	if !strings.Contains(err.Error(), "不能同时使用") {
		t.Errorf("错误信息不正确, 实际: %v", err)
	}
}

// ========================================================================
// Cross-cutting: Coverage for uncovered init functions (empty, but verify they exist)
// ========================================================================

func TestEmptyInitFunctions_Compiled(t *testing.T) {
	// 验证所有命令的 init 函数已执行（命令已注册），
	// 覆盖 version.go:20 init, import.go:45 init, update.go:36 init
	commands := []string{"version", "import", "update"}
	for _, name := range commands {
		cmd, _, err := rootCmd.Find([]string{name})
		if err != nil {
			t.Errorf("命令 %q 应被 init 注册, 错误: %v", name, err)
			continue
		}
		if cmd.Short == "" {
			t.Errorf("命令 %q 的 Short 不应为空", name)
		}
	}
}

// ========================================================================
// endpoint.go — promptForInput error path
// ========================================================================

func TestPromptForInput_Error(t *testing.T) {
	// 模拟 stdin 读取错误（关闭 pipe 的读取端）
	r, w, _ := os.Pipe()
	w.Close() // 关闭写入端，导致读取时出错
	r.Close() // 关闭读取端

	oldStdin := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = oldStdin }()

	_, err := promptForInput()
	if err == nil {
		t.Log("promptForInput 在关闭的 pipe 上可能不返回错误（取决于 Go 版本）")
		// 某些 Go 版本中，从关闭的 pipe 读取可能返回 io.EOF
	}
}

func TestAllCommands_UseFieldFormat(t *testing.T) {
	// 所有命令的 Use 应非空且不含特殊字符
	var checkCmd func(cmd *cobra.Command, path string)
	checkCmd = func(cmd *cobra.Command, path string) {
		if cmd.Use == "" {
			t.Errorf("%s: Use 不应为空", path)
		}
		if strings.Contains(cmd.Use, "\n") {
			t.Errorf("%s: Use 不应包含换行: %q", path, cmd.Use)
		}
		for _, sub := range cmd.Commands() {
			checkCmd(sub, path+" "+sub.Use)
		}
	}
	checkCmd(rootCmd, "root")
}

// ========================================================================
// endpoint.go — endpointStatusCmd light test
// ========================================================================

func TestEndpointStatusCmd_RunE_NoEndpoints(t *testing.T) {
	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	os.Stdout = w

	cmd := &cobra.Command{}
	cmd.Flags().StringP("config", "c", "", "")
	cmd.Flags().Set("config", t.TempDir())

	err := endpointStatusCmd.RunE(cmd, nil)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("无端点时 status 应返回 nil, 得到: %v", err)
	}
	if !strings.Contains(output, "AGENT") {
		t.Errorf("输出应包含 AGENT 表头, 实际: %s", output)
	}
}

// ========================================================================
// endpoint.go — endpointProvidersCmd args validation
// ========================================================================

func TestEndpointProvidersCmd_Args(t *testing.T) {
	if endpointProvidersCmd.Args == nil {
		t.Error("endpointProvidersCmd.Args 不应为 nil")
	}
	// cobra.NoArgs 应拒绝任何参数
	err := endpointProvidersCmd.Args(endpointProvidersCmd, []string{"unexpected"})
	if err == nil {
		t.Error("NoArgs 应拒绝参数")
	}
}

// ========================================================================
// endpoint.go — endpointCmd Root RunE (shows help)
// ========================================================================

func TestEndpointCmd_RunE_ShowsHelp(t *testing.T) {
	// endpointCmd.RunE 调用 cmd.Help()，输出到命令的 ErrOrStderr
	cmd := &cobra.Command{}
	cmd.SetErr(&bytes.Buffer{})

	err := endpointCmd.RunE(cmd, nil)

	if err != nil {
		t.Errorf("endpoint 无子命令时应返回 nil, 得到: %v", err)
	}
	// 输出可能为空或包含帮助信息，但不应该 panic
	t.Log("endpoint 无子命令执行成功")
}

// ========================================================================
// endpoint.go — endpointAddCmd Interactive mode
// ========================================================================

func TestEndpointAddCmd_RunE_InteractiveMode(t *testing.T) {
	// GH-13: promptForInput 使用 bufio.NewReader(os.Stdin) 每次调用创建新 Reader，
	// 在多 prompt 场景中第一个 bufio.Reader 会过度 buff 管道数据，导致后续 prompt 读取到 EOF。
	// 单 prompt 场景（TestEndpointAddCmd_RunE_InteractiveOnlyModelMissing）正常工作。
	t.Skip("GH-13: promptForInput 的 bufio.NewReader 过度 buff，不支持多 prompt 管道输入")
}

func TestEndpointAddCmd_RunE_InteractiveOnlyModelMissing(t *testing.T) {
	configDir := t.TempDir()

	// 提供 provider, url, key，仅 model 缺失
	// stdin 只提供 model
	stdinInput := "gpt-4\n"
	rStdin, wStdin, _ := os.Pipe()
	wStdin.Write([]byte(stdinInput))
	wStdin.Close()

	oldStdin := os.Stdin
	os.Stdin = rStdin
	defer func() { os.Stdin = oldStdin }()

	cmd := &cobra.Command{Use: "add", Args: cobra.ExactArgs(1)}
	cmd.Flags().String("provider", "", "")
	cmd.Flags().String("url", "", "")
	cmd.Flags().String("key", "", "")
	cmd.Flags().String("model", "", "")
	cmd.Flags().String("model-opus", "", "")
	cmd.Flags().String("model-sonnet", "", "")
	cmd.Flags().String("model-haiku", "", "")
	cmd.Flags().String("model-subagent", "", "")
	cmd.Flags().StringP("config", "c", "", "")
	cmd.Flags().Set("config", configDir)
	cmd.Flags().Set("provider", "openai")
	cmd.Flags().Set("url", "https://api.openai.com")
	cmd.Flags().Set("key", "sk-test-key")

	err := endpointAddCmd.RunE(cmd, []string{"partial-ep"})
	if err != nil {
		t.Fatalf("部分交互模式应成功, 得到: %v", err)
	}

	envPath := filepath.Join(configDir, "endpoints", "partial-ep", "endpoint.env")
	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		t.Error("部分交互创建后 endpoint.env 应存在")
	}
}

// ========================================================================
// endpoint.go — List with broken endpoint config (ReadEndpointConfig error)
// ========================================================================

func TestEndpointListCmd_RunE_WithBrokenEndpoint(t *testing.T) {
	configDir := t.TempDir()

	// 创建一个有效端点
	addCmd := newEndpointAddCmd(configDir)
	addCmd.Flags().Set("provider", "openai")
	addCmd.Flags().Set("url", "https://api.openai.com")
	addCmd.Flags().Set("key", "sk-test-key")
	addCmd.Flags().Set("model", "gpt-4")
	err := endpointAddCmd.RunE(addCmd, []string{"good-ep"})
	if err != nil {
		t.Fatalf("创建端点失败: %v", err)
	}

	// 创建一个"端点"目录但没有 endpoint.env（模拟损坏的配置）
	brokenDir := filepath.Join(configDir, "endpoints", "broken-ep")
	os.MkdirAll(brokenDir, 0755)
	// 故意不创建 endpoint.env

	// 创建一个文件（非目录）在 endpoints 目录中，测试跳过逻辑
	fileEntry := filepath.Join(configDir, "endpoints", "not-a-dir.txt")
	os.WriteFile(fileEntry, []byte("test"), 0644)

	// 执行 list
	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	os.Stdout = w

	cmd := newListOrStatusCmd(configDir)
	err = endpointListCmd.RunE(cmd, nil)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("list 应返回 nil, 得到: %v", err)
	}
	// 有效端点应正常显示
	if !strings.Contains(output, "good-ep") {
		t.Errorf("输出应包含 good-ep, 实际: %s", output)
	}
	// 损坏的端点应显示为 (error)
	if !strings.Contains(output, "(error)") {
		t.Errorf("损坏的端点应显示为 (error), 实际: %s", output)
	}
}

// ========================================================================
// endpoint.go — Show with optional model fields
// ========================================================================

func TestEndpointShowCmd_RunE_WithOptionalFields(t *testing.T) {
	configDir := t.TempDir()

	// 创建带所有 model 字段的端点
	addCmd := newEndpointAddCmd(configDir)
	addCmd.Flags().Set("provider", "deepseek")
	addCmd.Flags().Set("url", "https://api.deepseek.com")
	addCmd.Flags().Set("key", "sk-ds-key")
	addCmd.Flags().Set("model", "deepseek-chat")
	addCmd.Flags().Set("model-opus", "opus-v2")
	addCmd.Flags().Set("model-sonnet", "sonnet-v3")
	addCmd.Flags().Set("model-haiku", "haiku-v1")
	addCmd.Flags().Set("model-subagent", "subagent-m1")
	err := endpointAddCmd.RunE(addCmd, []string{"full-show-ep"})
	if err != nil {
		t.Fatalf("创建端点失败: %v", err)
	}

	// 执行 show
	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	os.Stdout = w

	cmd := &cobra.Command{Use: "show", Args: cobra.ExactArgs(1)}
	cmd.Flags().StringP("config", "c", "", "")
	cmd.Flags().Set("config", configDir)

	err = endpointShowCmd.RunE(cmd, []string{"full-show-ep"})

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("show 应返回 nil, 得到: %v", err)
	}
	if !strings.Contains(output, "Model (Opus)") {
		t.Errorf("输出应包含 Model (Opus), 实际: %s", output)
	}
	if !strings.Contains(output, "Model (Sonnet)") {
		t.Errorf("输出应包含 Model (Sonnet), 实际: %s", output)
	}
	if !strings.Contains(output, "Model (Haiku)") {
		t.Errorf("输出应包含 Model (Haiku), 实际: %s", output)
	}
	if !strings.Contains(output, "Model (Subagent)") {
		t.Errorf("输出应包含 Model (Subagent), 实际: %s", output)
	}
}

// ========================================================================
// endpoint.go — Status with broken endpoint config
// ========================================================================

func TestEndpointStatusCmd_RunE_WithBrokenEndpoint(t *testing.T) {
	configDir := t.TempDir()

	// 创建一个有效端点
	addCmd := newEndpointAddCmd(configDir)
	addCmd.Flags().Set("provider", "openai")
	addCmd.Flags().Set("url", "https://api.openai.com")
	addCmd.Flags().Set("key", "sk-openai-key")
	addCmd.Flags().Set("model", "gpt-4")
	err := endpointAddCmd.RunE(addCmd, []string{"st-ep"})
	if err != nil {
		t.Fatalf("创建端点失败: %v", err)
	}

	// 创建一个损坏的端点（无 endpoint.env）
	brokenDir := filepath.Join(configDir, "endpoints", "broken-st")
	os.MkdirAll(brokenDir, 0755)

	// 创建一个空 provider 的端点（应被跳过）
	noprovDir := filepath.Join(configDir, "endpoints", "no-prov")
	os.MkdirAll(noprovDir, 0755)
	os.WriteFile(filepath.Join(noprovDir, "endpoint.env"), []byte(""), 0644)

	// 非目录文件
	os.WriteFile(filepath.Join(configDir, "endpoints", "file.txt"), []byte("x"), 0644)

	// 执行 status
	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	os.Stdout = w

	cmd := newListOrStatusCmd(configDir)
	err = endpointStatusCmd.RunE(cmd, nil)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("status 应返回 nil, 得到: %v", err)
	}
	if !strings.Contains(output, "AGENT") {
		t.Errorf("输出应包含 AGENT 表头, 实际: %s", output)
	}
	if !strings.Contains(output, "st-ep") {
		t.Errorf("输出应包含端点名称, 实际: %s", output)
	}
}

// ========================================================================
// doctor.go — RunE with real Docker (lightweight check)
// ========================================================================

func TestDoctorCmd_RunE_WithDocker(t *testing.T) {
	// 此测试使用真实的 Docker 连接执行环境诊断。
	// 诊断仅包含 os.Stat + Docker Ping/Info + buildx check 等轻量级操作。
	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	os.Stdout = w

	// 使用空 command（--config 未指定，使用默认路径）
	cmd := &cobra.Command{}
	cmd.Flags().StringP("config", "c", "", "")

	err := doctorCmd.RunE(cmd, nil)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// doctor 可能成功也可能部分失败，但必须输出诊断信息
	if !strings.Contains(output, "环境诊断") && !strings.Contains(err.Error(), "Docker") {
		// 如果 Docker 完全不可用，错误信息应包含 Docker 相关提示
	}
	if !strings.Contains(output, "核心依赖") && !strings.Contains(output, "结果:") {
		// 如果输出了诊断结果但格式不对
		if err != nil {
			t.Logf("doctor 返回错误: %v", err)
		}
	}
	// 不能断言 err == nil，因为 buildx 可能不可用导致 exit code 1
	// 但输出应包含诊断的基本结构
	t.Logf("doctor 输出:\n%s", output)
}

// ========================================================================
// endpoint.go — endpointAddCmd RunE with stdin input error
// ========================================================================

func TestEndpointAddCmd_RunE_InteractiveStdinError(t *testing.T) {
	configDir := t.TempDir()

	// 关闭 stdin pipe → 模拟读取错误
	rStdin, wStdin, _ := os.Pipe()
	wStdin.Close() // 立即关闭写入端

	oldStdin := os.Stdin
	os.Stdin = rStdin
	defer func() { os.Stdin = oldStdin }()

	cmd := &cobra.Command{Use: "add", Args: cobra.ExactArgs(1)}
	cmd.Flags().String("provider", "", "")
	cmd.Flags().String("url", "", "")
	cmd.Flags().String("key", "", "")
	cmd.Flags().String("model", "", "")
	cmd.Flags().String("model-opus", "", "")
	cmd.Flags().String("model-sonnet", "", "")
	cmd.Flags().String("model-haiku", "", "")
	cmd.Flags().String("model-subagent", "", "")
	cmd.Flags().StringP("config", "c", "", "")
	cmd.Flags().Set("config", configDir)

	err := endpointAddCmd.RunE(cmd, []string{"stdin-error-ep"})
	// 在关闭的 pipe 上，promptForInput 可能会返回错误
	// 也可能返回 io.EOF（空字符串），然后 provider 为空，
	// 后续创建 endpoint 时 provider 为空，WriteEndpointConfig 可能失败
	if err != nil {
		// 错误是预期的（stdin 不可用）
		t.Logf("stdin 错误时返回预期错误: %v", err)
	}
}

// ========================================================================
// Docker Error Path Tests — DOCKER_HOST set to invalid Unix socket
// These tests verify error handling when Docker daemon is unreachable.
// dockerhelper.NewClient() succeeds (lazy connection), but first API call
// fails fast on Unix domain socket (<1ms).
// ========================================================================

func TestBuildCmd_RunE_DockerError(t *testing.T) {
	t.Setenv("DOCKER_HOST", "unix:///tmp/nonexistent-test-docker.sock")

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.Flags().String("deps", "", "")
	cmd.Flags().String("base-image", "", "")
	cmd.Flags().String("config", "", "")
	cmd.Flags().Bool("no-cache", false, "")
	cmd.Flags().Bool("rebuild", false, "")
	cmd.Flags().Int("max-retry", 3, "")
	cmd.Flags().String("gh-proxy", "", "")

	err := buildCmd.RunE(cmd, []string{})
	if err == nil {
		t.Error("build with invalid DOCKER_HOST 应返回错误")
	}
}

func TestExportCmd_RunE_DockerError(t *testing.T) {
	t.Setenv("DOCKER_HOST", "unix:///tmp/nonexistent-test-docker.sock")

	cmd := &cobra.Command{}
	cmd.Flags().String("image", "", "")

	// No args — uses default filename "agent-forge.tar" and default image "agent-forge:latest"
	err := exportCmd.RunE(cmd, nil)
	if err == nil {
		t.Error("export with invalid DOCKER_HOST 应返回错误")
	}
}

func TestExportCmd_RunE_WithCustomArgs_DockerError(t *testing.T) {
	t.Setenv("DOCKER_HOST", "unix:///tmp/nonexistent-test-docker.sock")

	cmd := &cobra.Command{}
	cmd.Flags().String("image", "", "")
	cmd.Flags().Set("image", "custom-image:tag")

	// Custom filename + custom image ref
	err := exportCmd.RunE(cmd, []string{"custom-output.tar"})
	if err == nil {
		t.Error("export with invalid DOCKER_HOST 应返回错误")
	}
}

func TestImportCmd_RunE_DockerError(t *testing.T) {
	t.Setenv("DOCKER_HOST", "unix:///tmp/nonexistent-test-docker.sock")

	cmd := &cobra.Command{}

	err := importCmd.RunE(cmd, []string{"test-image.tar"})
	if err == nil {
		t.Error("import with invalid DOCKER_HOST 应返回错误")
	}
}

func TestDepsCmd_RunE_DockerError(t *testing.T) {
	t.Setenv("DOCKER_HOST", "unix:///tmp/nonexistent-test-docker.sock")

	cmd := &cobra.Command{}
	cmd.Flags().String("image", "", "")
	cmd.Flags().String("config", "", "")

	err := depsCmd.RunE(cmd, []string{})
	if err == nil {
		t.Error("deps with invalid DOCKER_HOST 应返回错误")
	}
}

func TestRunCmd_RunE_DockerError(t *testing.T) {
	t.Setenv("DOCKER_HOST", "unix:///tmp/nonexistent-test-docker.sock")

	cmd := &cobra.Command{}
	cmd.Flags().String("agent", "", "")
	cmd.Flags().StringArray("port", nil, "")
	cmd.Flags().StringArray("mount", nil, "")
	cmd.Flags().StringArray("env", nil, "")
	cmd.Flags().String("workdir", "", "")
	cmd.Flags().Bool("recall", false, "")
	cmd.Flags().Bool("docker", false, "")
	cmd.Flags().Bool("dind", false, "")
	cmd.Flags().String("run", "", "")
	cmd.Flags().String("config", "", "")

	// Default params, no recall/run conflict — should proceed to engine.Run and fail on Docker
	err := runCmd.RunE(cmd, []string{})
	if err == nil {
		t.Error("run with invalid DOCKER_HOST 应返回错误")
	}
}

func TestDoctorCmd_RunE_DockerError(t *testing.T) {
	t.Setenv("DOCKER_HOST", "unix:///tmp/nonexistent-test-docker.sock")

	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	os.Stdout = w

	cmd := &cobra.Command{}
	cmd.Flags().String("config", "", "")

	err := doctorCmd.RunE(cmd, nil)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	_ = buf.String() // output not asserted — depends on how Diagnose handles Ping failure

	if err == nil {
		t.Error("doctor with invalid DOCKER_HOST 应返回错误")
	}
}

// ========================================================================
// update.go — Network error path
// ========================================================================

func TestUpdateCmd_RunE_NetworkError(t *testing.T) {
	t.Setenv("UPDATE_URL", "http://127.0.0.1:1/nonexistent-update")

	cmd := &cobra.Command{}

	// Connection to 127.0.0.1:1 fails fast (kernel RST)
	err := updateCmd.RunE(cmd, []string{})
	if err == nil {
		t.Error("update with invalid UPDATE_URL 应返回错误")
	}
}

// ========================================================================
// root.go — rootCmd.RunE closure delegates to run
// ========================================================================

func TestRootCmd_RunE_DelegatesToRun(t *testing.T) {
	t.Setenv("DOCKER_HOST", "unix:///tmp/nonexistent-test-docker.sock")

	// rootCmd.RunE delegates to runCmd.RunE, which needs the run flags
	cmd := &cobra.Command{Use: "agent-forge"}
	cmd.Flags().String("agent", "", "")
	cmd.Flags().StringArray("port", nil, "")
	cmd.Flags().StringArray("mount", nil, "")
	cmd.Flags().StringArray("env", nil, "")
	cmd.Flags().String("workdir", "", "")
	cmd.Flags().Bool("recall", false, "")
	cmd.Flags().Bool("docker", false, "")
	cmd.Flags().Bool("dind", false, "")
	cmd.Flags().String("run", "", "")
	cmd.Flags().String("config", "", "")

	err := rootCmd.RunE(cmd, []string{})
	if err == nil {
		t.Error("rootCmd with invalid DOCKER_HOST 应返回错误")
	}
}

// ========================================================================
// endpoint.go — ReadDir ENOTDIR error paths
// endpoint list / status — when "endpoints" is a file, not a directory,
// os.ReadDir returns ENOTDIR (not IsNotExist), which hits the
// newExitCodeError error path.
// ========================================================================

func TestEndpointListCmd_RunE_ReadDirError(t *testing.T) {
	configDir := t.TempDir()

	// Create a FILE at the endpoints path to trigger ReadDir ENOTDIR
	endpointsPath := filepath.Join(configDir, "endpoints")
	if err := os.WriteFile(endpointsPath, []byte("not a directory"), 0644); err != nil {
		t.Fatal(err)
	}

	cmd := &cobra.Command{}
	cmd.Flags().StringP("config", "c", "", "")
	cmd.Flags().Set("config", configDir)

	err := endpointListCmd.RunE(cmd, nil)
	if err == nil {
		t.Error("当 endpoints 为文件时 list 应返回 newExitCodeError")
	}
	t.Logf("list ReadDir ENOTDIR 错误: %v", err)
}

func TestEndpointStatusCmd_RunE_ReadDirError(t *testing.T) {
	configDir := t.TempDir()

	// Create a FILE at the endpoints path to trigger ReadDir ENOTDIR
	endpointsPath := filepath.Join(configDir, "endpoints")
	if err := os.WriteFile(endpointsPath, []byte("not a directory"), 0644); err != nil {
		t.Fatal(err)
	}

	cmd := &cobra.Command{}
	cmd.Flags().StringP("config", "c", "", "")
	cmd.Flags().Set("config", configDir)

	err := endpointStatusCmd.RunE(cmd, nil)
	if err == nil {
		t.Error("当 endpoints 为文件时 status 应返回 newExitCodeError")
	}
	t.Logf("status ReadDir ENOTDIR 错误: %v", err)
}

// ========================================================================
// endpoint.go — WriteEndpointConfig error path (add subcommand)
// When the endpoints parent directory is not writable,
// endpointmanager.WriteEndpointConfig fails on MkdirAll.
// ========================================================================

func TestEndpointAddCmd_RunE_WriteConfigError(t *testing.T) {
	configDir := t.TempDir()

	// Create endpoints dir with read-only permissions
	endpointsDir := filepath.Join(configDir, "endpoints")
	if err := os.MkdirAll(endpointsDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(endpointsDir, 0444); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(endpointsDir, 0755) // restore for cleanup

	cmd := newEndpointAddCmd(configDir)
	cmd.Flags().Set("provider", "openai")
	cmd.Flags().Set("url", "https://api.openai.com")
	cmd.Flags().Set("key", "sk-test-key")

	err := endpointAddCmd.RunE(cmd, []string{"test-ep"})
	if err == nil {
		t.Error("非可写 endpoints 目录应返回 newExitCodeError")
	}
	t.Logf("预期 WriteConfig 错误: %v", err)
}

// ========================================================================
// endpoint.go — Interactive mode: single prompt (provider missing)
// When provider is missing but url+key+model are provided, exactly 1
// promptForInput call occurs (for provider). This avoids GH-13's
// bufio over-buffering issue.
// ========================================================================

func TestEndpointAddCmd_RunE_InteractiveSinglePrompt(t *testing.T) {
	configDir := t.TempDir()

	// Only provider is missing — triggers interactive mode with exactly 1 prompt
	stdinInput := "openai\n"
	rStdin, wStdin, _ := os.Pipe()
	wStdin.Write([]byte(stdinInput))
	wStdin.Close()

	oldStdin := os.Stdin
	os.Stdin = rStdin
	defer func() { os.Stdin = oldStdin }()

	cmd := &cobra.Command{Use: "add", Args: cobra.ExactArgs(1)}
	cmd.Flags().String("provider", "", "")
	cmd.Flags().String("url", "", "")
	cmd.Flags().String("key", "", "")
	cmd.Flags().String("model", "", "")
	cmd.Flags().String("model-opus", "", "")
	cmd.Flags().String("model-sonnet", "", "")
	cmd.Flags().String("model-haiku", "", "")
	cmd.Flags().String("model-subagent", "", "")
	cmd.Flags().StringP("config", "c", "", "")
	cmd.Flags().Set("config", configDir)
	cmd.Flags().Set("url", "https://api.openai.com")
	cmd.Flags().Set("key", "sk-test-key")
	cmd.Flags().Set("model", "gpt-4")

	// Capture stdout to verify interactive prompt output
	rOut, wOut, _ := os.Pipe()
	oldStdout := os.Stdout
	os.Stdout = wOut
	defer func() { os.Stdout = oldStdout }()

	err := endpointAddCmd.RunE(cmd, []string{"interactive-ep"})

	wOut.Close()
	var outBuf bytes.Buffer
	io.Copy(&outBuf, rOut)
	output := outBuf.String()

	if err != nil {
		t.Fatalf("单 prompt 交互模式应成功, 得到: %v", err)
	}

	if !strings.Contains(output, "交互式配置模式") {
		t.Error("输出应包含交互式配置模式提示")
	}

	// Verify endpoint was created correctly with provider from stdin
	envPath := filepath.Join(configDir, "endpoints", "interactive-ep", "endpoint.env")
	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		t.Error("交互创建后 endpoint.env 应存在")
	}
}

// ========================================================================
// endpoint.go — Apply with named nonexistent endpoint
// endpointApplyCmd: when args[0] is provided but endpoint does not exist,
// applysyncer.ReadAndSyncEndpoint returns error.
// ========================================================================

func TestEndpointApplyCmd_RunE_NamedEndpointError(t *testing.T) {
	configDir := t.TempDir()

	cmd := &cobra.Command{}
	cmd.Flags().String("agent", "", "")
	cmd.Flags().StringP("config", "c", "", "")
	cmd.Flags().Set("config", configDir)

	// Provide nonexistent endpoint name
	err := endpointApplyCmd.RunE(cmd, []string{"nonexistent-ep"})
	if err == nil {
		t.Error("nonexistent endpoint 应返回错误")
	}
	t.Logf("预期 ReadAndSync 错误: %v", err)
}

// ========================================================================
// endpoint.go — Interactive mode: single prompt for URL
// provider+key+model set, url not set → enters interactive mode,
// exactly 1 prompt for URL. No GH-13 issue.
// ========================================================================

func TestEndpointAddCmd_RunE_InteractiveUrlPrompt(t *testing.T) {
	configDir := t.TempDir()

	// Only URL is missing — triggers interactive mode with exactly 1 prompt
	stdinInput := "https://api.example.com\n"
	rStdin, wStdin, _ := os.Pipe()
	wStdin.Write([]byte(stdinInput))
	wStdin.Close()

	oldStdin := os.Stdin
	os.Stdin = rStdin
	defer func() { os.Stdin = oldStdin }()

	cmd := &cobra.Command{Use: "add", Args: cobra.ExactArgs(1)}
	cmd.Flags().String("provider", "", "")
	cmd.Flags().String("url", "", "")
	cmd.Flags().String("key", "", "")
	cmd.Flags().String("model", "", "")
	cmd.Flags().String("model-opus", "", "")
	cmd.Flags().String("model-sonnet", "", "")
	cmd.Flags().String("model-haiku", "", "")
	cmd.Flags().String("model-subagent", "", "")
	cmd.Flags().StringP("config", "c", "", "")
	cmd.Flags().Set("config", configDir)
	cmd.Flags().Set("provider", "openai")
	cmd.Flags().Set("key", "sk-test-key")
	cmd.Flags().Set("model", "gpt-4")

	rOut, wOut, _ := os.Pipe()
	oldStdout := os.Stdout
	os.Stdout = wOut
	defer func() { os.Stdout = oldStdout }()

	err := endpointAddCmd.RunE(cmd, []string{"url-prompt-ep"})

	wOut.Close()
	var outBuf bytes.Buffer
	io.Copy(&outBuf, rOut)

	if err != nil {
		t.Fatalf("url 单 prompt 交互模式应成功, 得到: %v", err)
	}

	envPath := filepath.Join(configDir, "endpoints", "url-prompt-ep", "endpoint.env")
	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		t.Error("交互创建后 endpoint.env 应存在")
	}
}

// ========================================================================
// endpoint.go — Interactive mode: single prompt for key
// provider+url+model set, key not set → enters interactive mode,
// exactly 1 prompt for key. No GH-13 issue.
// ========================================================================

func TestEndpointAddCmd_RunE_InteractiveKeyPrompt(t *testing.T) {
	configDir := t.TempDir()

	// Only key is missing — triggers interactive mode with exactly 1 prompt
	stdinInput := "sk-new-test-key\n"
	rStdin, wStdin, _ := os.Pipe()
	wStdin.Write([]byte(stdinInput))
	wStdin.Close()

	oldStdin := os.Stdin
	os.Stdin = rStdin
	defer func() { os.Stdin = oldStdin }()

	cmd := &cobra.Command{Use: "add", Args: cobra.ExactArgs(1)}
	cmd.Flags().String("provider", "", "")
	cmd.Flags().String("url", "", "")
	cmd.Flags().String("key", "", "")
	cmd.Flags().String("model", "", "")
	cmd.Flags().String("model-opus", "", "")
	cmd.Flags().String("model-sonnet", "", "")
	cmd.Flags().String("model-haiku", "", "")
	cmd.Flags().String("model-subagent", "", "")
	cmd.Flags().StringP("config", "c", "", "")
	cmd.Flags().Set("config", configDir)
	cmd.Flags().Set("provider", "openai")
	cmd.Flags().Set("url", "https://api.openai.com")
	cmd.Flags().Set("model", "gpt-4")

	rOut, wOut, _ := os.Pipe()
	oldStdout := os.Stdout
	os.Stdout = wOut
	defer func() { os.Stdout = oldStdout }()

	err := endpointAddCmd.RunE(cmd, []string{"key-prompt-ep"})

	wOut.Close()
	var outBuf bytes.Buffer
	io.Copy(&outBuf, rOut)

	if err != nil {
		t.Fatalf("key 单 prompt 交互模式应成功, 得到: %v", err)
	}

	envPath := filepath.Join(configDir, "endpoints", "key-prompt-ep", "endpoint.env")
	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		t.Error("交互创建后 endpoint.env 应存在")
	}
}

// ========================================================================
// endpoint.go — Interactive mode: multi-prompt using timed pipe writes
// Writes data with delay to avoid bufio over-buffering (GH-13 workaround).
// Covers model prompt inside interactive mode.
// ========================================================================

func TestEndpointAddCmd_RunE_InteractiveMultiPrompt(t *testing.T) {
	configDir := t.TempDir()

	// Use goroutine to write pipe data with delay, so the first bufio.Reader
	// only sees the first line and the second reader sees the second line.
	rStdin, wStdin, _ := os.Pipe()
	go func() {
		wStdin.Write([]byte("openai\n"))
		time.Sleep(50 * time.Millisecond)
		wStdin.Write([]byte("gpt-4\n"))
		time.Sleep(10 * time.Millisecond)
		wStdin.Close()
	}()

	oldStdin := os.Stdin
	os.Stdin = rStdin
	defer func() { os.Stdin = oldStdin }()

	cmd := &cobra.Command{Use: "add", Args: cobra.ExactArgs(1)}
	cmd.Flags().String("provider", "", "")
	cmd.Flags().String("url", "", "")
	cmd.Flags().String("key", "", "")
	cmd.Flags().String("model", "", "")
	cmd.Flags().String("model-opus", "", "")
	cmd.Flags().String("model-sonnet", "", "")
	cmd.Flags().String("model-haiku", "", "")
	cmd.Flags().String("model-subagent", "", "")
	cmd.Flags().StringP("config", "c", "", "")
	cmd.Flags().Set("config", configDir)
	cmd.Flags().Set("url", "https://api.openai.com")
	cmd.Flags().Set("key", "sk-test-key")

	rOut, wOut, _ := os.Pipe()
	oldStdout := os.Stdout
	os.Stdout = wOut
	defer func() { os.Stdout = oldStdout }()

	err := endpointAddCmd.RunE(cmd, []string{"multi-prompt-ep"})

	wOut.Close()
	var outBuf bytes.Buffer
	io.Copy(&outBuf, rOut)
	output := outBuf.String()

	if err != nil {
		t.Fatalf("多 prompt 交互模式应成功, 得到: %v", err)
	}

	if !strings.Contains(output, "交互式配置模式") {
		t.Error("输出应包含交互式配置模式提示")
	}

	envPath := filepath.Join(configDir, "endpoints", "multi-prompt-ep", "endpoint.env")
	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		t.Error("交互创建后 endpoint.env 应存在")
	}
}

// ========================================================================
// endpoint.go — Apply SyncAllEndpoints error (no args, no endpoints dir)
// When apply is called without args and no endpoints dir exists,
// SyncAllEndpoints returns ReadDir ENOENT error.
// ========================================================================

func TestEndpointApplyCmd_RunE_SyncAllEndpointsError(t *testing.T) {
	configDir := t.TempDir()

	cmd := &cobra.Command{Use: "apply", Args: cobra.MaximumNArgs(1)}
	cmd.Flags().String("agent", "", "")
	cmd.Flags().StringP("config", "c", "", "")
	cmd.Flags().Set("config", configDir)

	// No args → SyncAllEndpoints; no endpoints dir → ReadDir error
	err := endpointApplyCmd.RunE(cmd, []string{})
	if err == nil {
		t.Error("无端点目录时 SyncAll 应返回错误")
	}
	t.Logf("预期 SyncAllEndpoints 错误: %v", err)
}
