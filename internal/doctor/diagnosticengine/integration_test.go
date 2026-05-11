// Package diagnosticengine 提供 DiagnosticEngine 的集成测试（IT-10）。
//
// 本文件覆盖 IT-10 的所有案例，验证三层诊断流程和自动修复机制。
// 使用 Engine 的功能选项（Option）和函数注入进行 mock，模拟各种真实场景。
package diagnosticengine

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/agent-forge/cli/internal/doctor/packagemanager"
)

// --- IT-10: DiagnosticEngine 三层环境诊断集成测试 ---

// itHelper 实现一个可控制行为的 mock DockerHelper。
type itHelper struct {
	pingErr error
	infoErr error
}

func (h *itHelper) Ping(ctx context.Context) error {
	return h.pingErr
}

func (h *itHelper) Info(ctx context.Context) (interface{}, error) {
	return nil, h.infoErr
}

// callTracker 跟踪函数调用次数，用于验证流程顺序。
type callTracker struct {
	mu      sync.Mutex
	calls   []string
}

func (t *callTracker) add(name string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.calls = append(t.calls, name)
}

func (t *callTracker) count(name string) int {
	t.mu.Lock()
	defer t.mu.Unlock()
	n := 0
	for _, c := range t.calls {
		if c == name {
			n++
		}
	}
	return n
}

// TestIT10_AllThreeLayersPassed 验证三层全部通过的场景。
//
// 覆盖案例：三层全部通过 — Docker daemon 运行，Docker 已安装，buildx 可选。
// 测试目标：CorePassed、RuntimePassed、OptionalPassed 均为 true，无 issue。
func TestIT10_AllThreeLayersPassed(t *testing.T) {
	engine := New(&itHelper{},
		WithSocketPath("/tmp/it10-fake-socket"),
	)
	engine.socketStat = func(path string) error {
		return nil // socket 正常
	}
	engine.execFunc = func(name string, args ...string) error {
		return nil // buildx 可用
	}

	diag, err := engine.Diagnose(context.Background())
	if err != nil {
		t.Fatalf("Diagnose() 返回错误: %v", err)
	}

	if !diag.CorePassed {
		t.Error("三层全部通过时 CorePassed 应为 true")
	}
	if !diag.RuntimePassed {
		t.Error("三层全部通过时 RuntimePassed 应为 true")
	}
	if !diag.OptionalPassed {
		t.Error("三层全部通过时 OptionalPassed 应为 true")
	}

	// 应无任何问题
	for _, issue := range diag.Issues {
		t.Errorf("全部通过时不应有 issue, 发现: Type=%d Layer=%s Message=%s",
			issue.Type, issue.Layer, issue.Message)
	}
}

// TestIT10_CoreDependencyMissing 验证核心依赖缺失且无自动修复能力的场景。
//
// 覆盖案例：核心依赖缺失 — 检测到 Docker 未安装（mock 场景），且包管理器不可用。
// 测试目标：CorePassed=false，包含 IssueCoreMissing，跳过后续层。
func TestIT10_CoreDependencyMissing(t *testing.T) {
	engine := New(&itHelper{},
		WithSocketPath("/nonexistent/it10-socket"),
	)
	// socket 不可达
	engine.socketStat = func(path string) error {
		return errors.New("socket not found")
	}
	// 包管理器也不可用
	engine.detectPM = func() (*packagemanager.Manager, error) {
		return nil, packagemanager.ErrNoPackageManager
	}

	diag, err := engine.Diagnose(context.Background())
	if err != nil {
		t.Fatalf("Diagnose() 返回错误: %v", err)
	}

	if diag.CorePassed {
		t.Error("核心依赖缺失时 CorePassed 应为 false")
	}

	// 确保后续层未被检查
	if diag.RuntimePassed {
		t.Error("核心依赖缺失时应跳过运行时检查，RuntimePassed 应为 false")
	}
	if diag.OptionalPassed {
		t.Error("核心依赖缺失时应跳过可选工具检查，OptionalPassed 应为 false")
	}

	// 应包含核心依赖缺失类型的 issue
	foundCoreIssue := false
	foundAutoFailed := false
	for _, issue := range diag.Issues {
		if issue.Type == IssueCoreMissing && issue.Layer == "核心依赖" {
			if issue.Message == "Docker socket 不存在: /nonexistent/it10-socket" {
				foundCoreIssue = true
			}
			if issue.Message == "未找到可用的包管理器" ||
				contains(issue.Message, "未找到可用的包管理器") {
				foundAutoFailed = true
			}
		}
	}
	if !foundCoreIssue {
		t.Error("应包含 socket 不存在的核心依赖缺失 issue")
	}
	if !foundAutoFailed {
		t.Error("应包含包管理器不可用的提示")
	}
}

// TestIT10_RuntimeError 验证核心依赖正常但运行时异常的场景。
//
// 覆盖案例：运行时异常 — Docker daemon 未运行（mock 场景）。
// 测试目标：CorePassed=true，RuntimePassed=false，
// 包含 IssueRuntimeError，OptionalPassed 因运行时失败也被标记为 false。
func TestIT10_RuntimeError(t *testing.T) {
	engine := New(&itHelper{pingErr: errors.New("connection refused")},
		WithSocketPath("/tmp/it10-fake-socket"),
	)
	engine.socketStat = func(path string) error {
		return nil // socket 存在
	}

	diag, err := engine.Diagnose(context.Background())
	if err != nil {
		t.Fatalf("Diagnose() 返回错误: %v", err)
	}

	if !diag.CorePassed {
		t.Error("socket 存在时 CorePassed 应为 true")
	}
	if diag.RuntimePassed {
		t.Error("Ping 失败时 RuntimePassed 应为 false")
	}

	found := false
	for _, issue := range diag.Issues {
		if issue.Type == IssueRuntimeError && issue.Layer == "运行时" {
			found = true
			break
		}
	}
	if !found {
		t.Error("应包含 IssueRuntimeError 类型的运行时问题")
	}
}

// TestIT10_PermissionDenied 验证运行时权限不足的场景。
//
// 覆盖案例：运行时异常 — 权限不足。
// 测试目标：CorePassed=true，RuntimePassed=false，
// 包含 IssuePermissionError，建议信息不为空。
func TestIT10_PermissionDenied(t *testing.T) {
	engine := New(&itHelper{pingErr: errors.New("permission denied")},
		WithSocketPath("/tmp/it10-fake-socket"),
	)
	engine.socketStat = func(path string) error {
		return nil
	}

	diag, err := engine.Diagnose(context.Background())
	if err != nil {
		t.Fatalf("Diagnose() 返回错误: %v", err)
	}

	if !diag.CorePassed {
		t.Error("socket 存在时 CorePassed 应为 true")
	}
	if diag.RuntimePassed {
		t.Error("权限拒绝时 RuntimePassed 应为 false")
	}

	foundIssue := false
	foundSuggestion := false
	for _, issue := range diag.Issues {
		if issue.Type == IssuePermissionError && issue.Layer == "运行时" {
			foundIssue = true
			if issue.Suggestion != "" {
				foundSuggestion = true
			}
		}
	}
	if !foundIssue {
		t.Error("应包含 IssuePermissionError 类型的权限问题")
	}
	if !foundSuggestion {
		t.Error("权限问题应包含建议信息")
	}
}

// TestIT10_OptionalToolMissing 验证核心和运行时通过但可选工具缺失的场景。
//
// 覆盖案例：可选工具缺失 — buildx 不可用。
// 测试目标：CorePassed=true，RuntimePassed=true，OptionalPassed=false，
// 包含 IssueOptionalToolMissing。
func TestIT10_OptionalToolMissing(t *testing.T) {
	engine := New(&itHelper{},
		WithSocketPath("/tmp/it10-fake-socket"),
	)
	engine.socketStat = func(path string) error {
		return nil
	}
	engine.execFunc = func(name string, args ...string) error {
		if name == "docker" && len(args) > 0 && args[0] == "buildx" {
			return errors.New("not found")
		}
		return nil
	}

	diag, err := engine.Diagnose(context.Background())
	if err != nil {
		t.Fatalf("Diagnose() 返回错误: %v", err)
	}

	if !diag.CorePassed {
		t.Error("CorePassed 应为 true")
	}
	if !diag.RuntimePassed {
		t.Error("RuntimePassed 应为 true")
	}
	if diag.OptionalPassed {
		t.Error("buildx 缺失时 OptionalPassed 应为 false")
	}

	found := false
	for _, issue := range diag.Issues {
		if issue.Type == IssueOptionalToolMissing {
			found = true
			break
		}
	}
	if !found {
		t.Error("应包含 IssueOptionalToolMissing 类型的可选工具提示")
	}
}

// TestIT10_AutoRepairFlow 验证核心依赖缺失后自动修复的完整流程。
//
// 覆盖案例：自动修复 — 包管理器安装后重新检测。
// 核心步骤：
//  1. socketStat 首次返回错误（核心依赖缺失）
//  2. detectPM 返回包管理器
//  3. installPM 成功安装 docker
//  4. socketStat 第二次调用返回 nil（安装后重新检查通过）
//  5. 继续检查运行时和可选工具
//
// 测试目标：修复后 CorePassed=true，完整诊断继续执行到所有层。
func TestIT10_AutoRepairFlow(t *testing.T) {
	callCount := 0
	engine := New(&itHelper{},
		WithSocketPath("/tmp/it10-fake-socket"),
	)
	engine.socketStat = func(path string) error {
		callCount++
		if callCount <= 1 {
			// 第一次 checkCoreDependency：socket 不存在
			return errors.New("socket not found")
		}
		// 第二次（auto-install 后重新检查）：socket 存在
		return nil
	}
	engine.detectPM = func() (*packagemanager.Manager, error) {
		return &packagemanager.Manager{
			Name:       "apt-get",
			InstallCmd: "apt-get install -y %s",
			CheckCmd:   "apt-get --version",
		}, nil
	}
	engine.installPM = func(pm *packagemanager.Manager, pkg string) (string, error) {
		return fmt.Sprintf("%s installed successfully", pkg), nil
	}
	engine.execFunc = func(name string, args ...string) error {
		return nil // buildx 可用
	}

	diag, err := engine.Diagnose(context.Background())
	if err != nil {
		t.Fatalf("Diagnose() 返回错误: %v", err)
	}

	if !diag.CorePassed {
		t.Error("自动修复后 CorePassed 应为 true")
	}
	if !diag.RuntimePassed {
		t.Error("自动修复后运行时检查应为 true")
	}
	if !diag.OptionalPassed {
		t.Error("自动修复后可选工具检查应为 true")
	}

	// socketStat 应被调用 2 次：初始检查 + 修复后重新检查
	if callCount != 2 {
		t.Errorf("socketStat 应被调用 2 次（初始+修复后），实际 %d 次", callCount)
	}

	// 验证安装相关的 issue 存在
	foundInstallIssue := false
	for _, issue := range diag.Issues {
		if issue.Type == IssueCoreMissing &&
			contains(issue.Message, "安装成功") {
			foundInstallIssue = true
			break
		}
	}
	if !foundInstallIssue {
		t.Error("应包含自动安装成功的提示")
	}
}

// TestIT10_AutoRepairFailed_DetectPM 验证包管理器检测失败时自动修复失败的场景。
//
// 覆盖案例：自动修复失败 — 无法检测到包管理器。
// 测试目标：auto-install 输出"未找到可用包管理器"错误，CorePassed 保持 false。
func TestIT10_AutoRepairFailed_DetectPM(t *testing.T) {
	engine := New(&itHelper{},
		WithSocketPath("/nonexistent/it10-socket"),
	)
	engine.socketStat = func(path string) error {
		return errors.New("socket not found")
	}
	engine.detectPM = func() (*packagemanager.Manager, error) {
		return nil, errors.New("no package manager found")
	}

	diag, err := engine.Diagnose(context.Background())
	if err != nil {
		t.Fatalf("Diagnose() 返回错误: %v", err)
	}

	if diag.CorePassed {
		t.Error("包管理器不可用时 CorePassed 应为 false")
	}

	found := false
	for _, issue := range diag.Issues {
		if issue.Type == IssueCoreMissing &&
			contains(issue.Message, "未找到可用的包管理器") {
			found = true
			break
		}
	}
	if !found {
		t.Error("应包含未找到包管理器的错误提示")
	}
}

// TestIT10_AutoRepairFailed_Install 验证安装失败时自动修复失败的场景。
//
// 覆盖案例：自动修复失败 — 包管理器安装失败。
// 测试目标：installPM 返回错误，CorePassed 保持 false，输出安装失败提示。
func TestIT10_AutoRepairFailed_Install(t *testing.T) {
	engine := New(&itHelper{},
		WithSocketPath("/nonexistent/it10-socket"),
	)
	engine.socketStat = func(path string) error {
		return errors.New("socket not found")
	}
	engine.detectPM = func() (*packagemanager.Manager, error) {
		return &packagemanager.Manager{
			Name: "apt-get",
		}, nil
	}
	engine.installPM = func(pm *packagemanager.Manager, pkg string) (string, error) {
		return "", errors.New("permission denied")
	}

	diag, err := engine.Diagnose(context.Background())
	if err != nil {
		t.Fatalf("Diagnose() 返回错误: %v", err)
	}

	if diag.CorePassed {
		t.Error("安装失败时 CorePassed 应为 false")
	}

	found := false
	for _, issue := range diag.Issues {
		if issue.Type == IssueCoreMissing &&
			contains(issue.Message, "自动安装 Docker 失败") {
			found = true
			break
		}
	}
	if !found {
		t.Error("应包含自动安装失败的错误提示")
	}
}

// contains 是 strings.Contains 的辅助函数，避免导入 strings 包。
func contains(s, substr string) bool {
	return len(s) >= len(substr) && containsSubstring(s, substr)
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
