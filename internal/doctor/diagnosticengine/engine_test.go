package diagnosticengine

import (
	"context"
	"errors"
	"testing"

	"github.com/agent-forge/cli/internal/doctor/packagemanager"
)

// mockHelper 实现一个可 mock 的 Docker helper
type mockHelper struct {
	pingErr error
	infoErr error
}

func (m *mockHelper) Ping(ctx context.Context) error {
	return m.pingErr
}

func (m *mockHelper) Info(ctx context.Context) (interface{}, error) {
	return nil, m.infoErr
}

// --- UT-18: Diagnose (ClassifyIssue) ---

func TestDiagnose_SocketNotExist(t *testing.T) {
	// Case 1: socket 不可达标记为核心依赖缺失
	engine := New(nil,
		WithSocketPath("/nonexistent/socket.sock"),
	)
	// socketStat 会返回 os.IsNotExist 错误
	engine.socketStat = func(path string) error {
		return errors.New("file does not exist")
	}

	diag, err := engine.Diagnose(context.Background())
	if err != nil {
		t.Fatalf("Diagnose() 返回错误: %v", err)
	}

	if diag.CorePassed {
		t.Error("socket 不存在时 CorePassed 应为 false")
	}

	found := false
	for _, issue := range diag.Issues {
		if issue.Type == IssueCoreMissing && issue.Layer == "核心依赖" {
			found = true
			break
		}
	}
	if !found {
		t.Error("应包含 IssueCoreMissing 类型的核心依赖问题")
	}
}

func TestDiagnose_PingFailed(t *testing.T) {
	// Case 2: Docker Ping 失败标记为运行时异常
	engine := New(&mockHelper{pingErr: errors.New("connection refused")},
		WithSocketPath("/tmp/fake-socket"),
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

func TestDiagnose_PermissionDenied(t *testing.T) {
	// Case 3: 权限不足标记为运行时权限问题
	engine := New(&mockHelper{pingErr: errors.New("permission denied")},
		WithSocketPath("/tmp/fake-socket"),
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
		}
		if issue.Type == IssuePermissionError &&
			issue.Suggestion != "" {
			foundSuggestion = true
		}
	}
	if !foundIssue {
		t.Error("应包含 IssuePermissionError 类型的权限问题")
	}
	if !foundSuggestion {
		t.Error("权限问题应包含建议信息")
	}
}

func TestDiagnose_BuildxNotAvailable(t *testing.T) {
	// Case 4: BuildKit 不可用标记为可选工具提示
	engine := New(&mockHelper{},
		WithSocketPath("/tmp/fake-socket"),
	)
	engine.socketStat = func(path string) error {
		return nil
	}
	// 注意：这里 mock helper 的 Ping 成功（pingErr = nil）
	// 但实际的 mockHelper 结构体需要正确处理 Ping 返回 nil

	// 使用 execFunc mock 让 buildx 检查失败
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
		t.Error("RuntimePassed 应为 true（Ping 已 mock 为成功）")
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

func TestDiagnose_AllPassed(t *testing.T) {
	// Case 5: 所有检查通过返回全部通过
	engine := New(&mockHelper{},
		WithSocketPath("/tmp/fake-socket"),
	)
	engine.socketStat = func(path string) error {
		return nil
	}
	engine.execFunc = func(name string, args ...string) error {
		return nil // 所有命令可用
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
	if !diag.OptionalPassed {
		t.Error("OptionalPassed 应为 true")
	}

	// 不应有 problems
	for _, issue := range diag.Issues {
		if issue.Type != IssueAllPassed {
			t.Errorf("不应有非通过的 issue: %+v", issue)
		}
	}
}

// --- Option 函数与默认 socketStat 路径 (UT-18 扩展) ---

func TestOptionFunctions(t *testing.T) {
	execCalled := false
	detectCalled := false
	installCalled := false

	// 使用真实的 Option 函数，而非直接字段赋值，以覆盖 WithExecFunc / WithDetectPM / WithInstallPM
	engine := New(nil,
		WithExecFunc(func(name string, args ...string) error {
			execCalled = true
			return nil
		}),
		WithDetectPM(func() (*packagemanager.Manager, error) {
			detectCalled = true
			return nil, errors.New("no package manager")
		}),
		WithInstallPM(func(pm *packagemanager.Manager, pkg string) (string, error) {
			installCalled = true
			return "installed", nil
		}),
		WithSocketPath("/nonexistent/option-test-socket"),
	)

	// 不覆盖 engine.socketStat，验证默认 os.Stat 实现
	if err := engine.socketStat("/nonexistent/option-test-socket"); err == nil {
		t.Error("默认 socketStat 在不存在的路径上应返回错误")
	}

	// 验证 WithExecFunc 设置的函数可用
	if err := engine.execFunc("echo", "hello"); err != nil {
		t.Errorf("execFunc 应返回 nil: %v", err)
	}
	if !execCalled {
		t.Error("WithExecFunc 设置的 execFunc 未被调用")
	}

	// 验证 WithDetectPM 设置的函数可用
	_, err := engine.detectPM()
	if err == nil {
		t.Error("WithDetectPM 应返回错误")
	}
	if !detectCalled {
		t.Error("WithDetectPM 未被调用")
	}

	// 验证 WithInstallPM 设置的函数可用
	out, err := engine.installPM(nil, "docker")
	if err != nil {
		t.Errorf("WithInstallPM 应返回 nil: %v", err)
	}
	if out != "installed" {
		t.Errorf("WithInstallPM 应返回 'installed', got %q", out)
	}
	if !installCalled {
		t.Error("WithInstallPM 未被调用")
	}
}

// --- Ping 成功但 Info 失败路径 (UT-18 扩展) ---

func TestDiagnose_PingOKButInfoFails(t *testing.T) {
	tests := []struct {
		name    string
		infoErr error
	}{
		{"Info返回具体错误", errors.New("unable to connect to daemon")},
		{"Info返回空错误", errors.New("")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := New(&mockHelper{infoErr: tt.infoErr},
				WithSocketPath("/tmp/diag-info-fake"),
			)
			engine.socketStat = func(path string) error { return nil }
			engine.execFunc = func(name string, args ...string) error { return nil }

			diag, err := engine.Diagnose(context.Background())
			if err != nil {
				t.Fatalf("Diagnose() 返回错误: %v", err)
			}

			if !diag.CorePassed {
				t.Error("CorePassed 应为 true")
			}
			if !diag.RuntimePassed {
				t.Error("Ping 成功时 RuntimePassed 应为 true")
			}
			if !diag.OptionalPassed {
				t.Error("OptionalPassed 应为 true")
			}

			// 虽然 RuntimePassed=true，但 Info 失败应记录一条额外 issue
			found := false
			for _, issue := range diag.Issues {
				if issue.Type == IssueRuntimeError &&
					contains(issue.Message, "获取 Docker 信息失败") {
					found = true
					break
				}
			}
			if !found {
				t.Error("应包含获取 Docker 信息失败的 issue")
			}
		})
	}
}
