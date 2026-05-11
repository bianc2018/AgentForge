// Package diagnosticengine 提供三层环境诊断功能。
//
// 执行三层诊断流程：
//   - 第一层（核心依赖）：检测 Docker 核心依赖是否安装
//   - 第二层（运行时）：检查 Docker daemon 运行状态和用户权限
//   - 第三层（可选工具）：检查 buildx 等可选工具
//
// 检测到缺失核心依赖时调用 Package Manager Adapter 自动安装。
package diagnosticengine

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/agent-forge/cli/internal/doctor/packagemanager"
)

// DockerHelper 是 DiagnosticEngine 需要的 Docker 操作接口。
// 通过接口而非具体类型，便于测试时 mock。
type DockerHelper interface {
	Ping(ctx context.Context) error
	Info(ctx context.Context) (interface{}, error)
}

// IssueType 表示诊断问题的类型。
type IssueType int

const (
	// IssueCoreMissing 表示核心依赖缺失。
	IssueCoreMissing IssueType = iota
	// IssueRuntimeError 表示运行时异常。
	IssueRuntimeError
	// IssuePermissionError 表示权限问题。
	IssuePermissionError
	// IssueOptionalToolMissing 表示可选工具缺失。
	IssueOptionalToolMissing
	// IssueAllPassed 表示所有检查通过。
	IssueAllPassed
)

// Issue 表示诊断过程中发现的一个问题。
type Issue struct {
	// Type 是问题的类型。
	Type IssueType
	// Layer 是发现问题的诊断层（"核心依赖", "运行时", "可选工具"）。
	Layer string
	// Message 是问题的描述信息。
	Message string
	// Suggestion 是解决问题的建议。
	Suggestion string
}

// Diagnosis 表示三层诊断的完整结果。
type Diagnosis struct {
	// CorePassed 表示第一层（核心依赖）是否通过。
	CorePassed bool
	// RuntimePassed 表示第二层（运行时）是否通过。
	RuntimePassed bool
	// OptionalPassed 表示第三层（可选工具）是否通过。
	OptionalPassed bool
	// Issues 是诊断过程中发现的所有问题。
	Issues []Issue
}

// Engine 是诊断引擎，执行三层环境诊断。
type Engine struct {
	helper     DockerHelper
	socketPath string                                   // Docker socket 路径，默认 /var/run/docker.sock
	execFunc   func(name string, args ...string) error  // mockable exec
	detectPM   func() (*packagemanager.Manager, error)  // mockable pm detect
	installPM  func(pm *packagemanager.Manager, pkg string) (string, error) // mockable pm install
	socketStat func(path string) error                  // mockable os.Stat
}

// Option 是 Engine 的配置选项。
type Option func(*Engine)

// defaultSocketPath 是默认的 Docker socket 路径。
const defaultSocketPath = "/var/run/docker.sock"

// WithExecFunc 设置可 mock 的 exec 函数。
func WithExecFunc(fn func(name string, args ...string) error) Option {
	return func(e *Engine) {
		e.execFunc = fn
	}
}

// WithDetectPM 设置可 mock 的包管理器检测函数。
func WithDetectPM(fn func() (*packagemanager.Manager, error)) Option {
	return func(e *Engine) {
		e.detectPM = fn
	}
}

// WithInstallPM 设置可 mock 的包管理器安装函数。
func WithInstallPM(fn func(pm *packagemanager.Manager, pkg string) (string, error)) Option {
	return func(e *Engine) {
		e.installPM = fn
	}
}

// WithSocketPath 设置 Docker socket 路径（用于测试）。
func WithSocketPath(path string) Option {
	return func(e *Engine) {
		e.socketPath = path
	}
}

// New 创建一个新的诊断引擎。
func New(helper DockerHelper, opts ...Option) *Engine {
	e := &Engine{
		helper:     helper,
		socketPath: defaultSocketPath,
		execFunc: func(name string, args ...string) error {
			cmd := exec.Command(name, args...)
			return cmd.Run()
		},
		detectPM: func() (*packagemanager.Manager, error) {
			return packagemanager.Detect()
		},
		installPM: func(pm *packagemanager.Manager, pkg string) (string, error) {
			return pm.Install(pkg)
		},
		socketStat: func(path string) error {
			_, err := os.Stat(path)
			return err
		},
	}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

// Diagnose 执行完整的三层环境诊断。
//
// 依次执行核心依赖检查、运行时检查和可选工具检查。
// 如果核心依赖缺失，尝试自动安装并重新检查。
//
// 返回诊断结果，包含每层的通过状态和所有发现的问题。
func (e *Engine) Diagnose(ctx context.Context) (*Diagnosis, error) {
	diag := &Diagnosis{}

	// ---- 第一层：核心依赖 ----
	e.checkCoreDependency(diag)

	// 如果核心依赖缺失，尝试自动安装
	if !diag.CorePassed {
		e.autoInstallCore(diag)
		// 安装后重新检查核心依赖（REQ-32）
		e.checkCoreDependency(diag)
	}

	// 核心依赖未通过时，跳过后续检测
	if !diag.CorePassed {
		return diag, nil
	}

	// ---- 第二层：运行时 ----
	e.checkRuntime(ctx, diag)

	// ---- 第三层：可选工具 ----
	e.checkOptionalTools(diag)

	return diag, nil
}

// checkCoreDependency 检查第一层核心依赖。
func (e *Engine) checkCoreDependency(diag *Diagnosis) {
	if err := e.socketStat(e.socketPath); err != nil {
		diag.CorePassed = false
		diag.Issues = append(diag.Issues, Issue{
			Type:       IssueCoreMissing,
			Layer:      "核心依赖",
			Message:    fmt.Sprintf("Docker socket 不存在: %s", e.socketPath),
			Suggestion: "请安装 Docker Engine，或检查 Docker 是否已启动",
		})
		return
	}

	diag.CorePassed = true
}

// autoInstallCore 尝试自动安装核心依赖。
func (e *Engine) autoInstallCore(diag *Diagnosis) {
	diag.Issues = append(diag.Issues, Issue{
		Type:       IssueCoreMissing,
		Layer:      "核心依赖",
		Message:    "Docker socket 缺失，尝试自动安装 Docker",
		Suggestion: "自动安装中...",
	})

	pm, err := e.detectPM()
	if err != nil {
		diag.Issues = append(diag.Issues, Issue{
			Type:       IssueCoreMissing,
			Layer:      "核心依赖",
			Message:    fmt.Sprintf("未找到可用的包管理器: %s", err.Error()),
			Suggestion: "请手动安装 Docker Engine (https://docs.docker.com/engine/install/)",
		})
		return
	}

	output, err := e.installPM(pm, "docker")
	if err != nil {
		diag.Issues = append(diag.Issues, Issue{
			Type:       IssueCoreMissing,
			Layer:      "核心依赖",
			Message:    fmt.Sprintf("自动安装 Docker 失败: %s", err.Error()),
			Suggestion: "安装失败，请手动安装 Docker Engine",
		})
		_ = output
		return
	}

	diag.Issues = append(diag.Issues, Issue{
		Type:       IssueCoreMissing,
		Layer:      "核心依赖",
		Message:    fmt.Sprintf("Docker 已通过 %s 安装成功", pm.Name),
		Suggestion: "已安装 Docker，请重新执行 doctor 确认",
	})
}

// checkRuntime 检查第二层运行时。
func (e *Engine) checkRuntime(ctx context.Context, diag *Diagnosis) {
	if err := e.helper.Ping(ctx); err != nil {
		diag.RuntimePassed = false

		if isPermissionError(err) {
			diag.Issues = append(diag.Issues, Issue{
				Type:       IssuePermissionError,
				Layer:      "运行时",
				Message:    fmt.Sprintf("Docker daemon 权限不足: %s", err.Error()),
				Suggestion: "请将当前用户加入 docker 用户组：sudo usermod -aG docker $USER",
			})
		} else {
			diag.Issues = append(diag.Issues, Issue{
				Type:       IssueRuntimeError,
				Layer:      "运行时",
				Message:    fmt.Sprintf("Docker daemon 未运行或不可达: %s", err.Error()),
				Suggestion: "请启动 Docker daemon：systemctl start docker",
			})
		}
		return
	}

	diag.RuntimePassed = true

	// 额外检查 Info 获取版本信息
	if _, err := e.helper.Info(ctx); err != nil {
		diag.Issues = append(diag.Issues, Issue{
			Type:       IssueRuntimeError,
			Layer:      "运行时",
			Message:    fmt.Sprintf("获取 Docker 信息失败: %s", err.Error()),
			Suggestion: "Docker 基本功能正常但无法获取详细信息",
		})
	}
}

// checkOptionalTools 检查第三层可选工具。
func (e *Engine) checkOptionalTools(diag *Diagnosis) {
	if err := e.execFunc("docker", "buildx", "version"); err != nil {
		diag.OptionalPassed = false
		diag.Issues = append(diag.Issues, Issue{
			Type:       IssueOptionalToolMissing,
			Layer:      "可选工具",
			Message:    "buildx 插件未安装",
			Suggestion: "建议安装 buildx 以支持高级构建功能：docker buildx install",
		})
		return
	}

	diag.OptionalPassed = true
}

// isPermissionError 判断错误是否为权限错误。
func isPermissionError(err error) bool {
	return strings.Contains(err.Error(), "permission denied") ||
		strings.Contains(err.Error(), "access denied")
}
