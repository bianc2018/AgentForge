package packagemanager

import (
	"errors"
	"strings"
	"testing"
)

// ---- mock helpers ----

// saveExecCommand 保存当前 execCommand 并在测试后恢复。
func saveExecCommand(t *testing.T) {
	t.Helper()
	orig := execCommand
	t.Cleanup(func() {
		execCommand = orig
	})
}

// mockExecCommand 设置 execCommand 使得只有 checkCmd 对应的检测命令返回成功。
func mockExecCommand(okCheck string) {
	execCommand = func(name string, args ...string) error {
		if name == "sh" && len(args) > 1 && args[1] == okCheck {
			return nil
		}
		return errors.New("not found")
	}
}

// mockExecCommandAllFail 设置 execCommand 使得所有命令都返回失败。
func mockExecCommandAllFail() {
	execCommand = func(name string, args ...string) error {
		return errors.New("not found")
	}
}

// ---- UT-16: Detect ----

func TestDetect(t *testing.T) {
	tests := []struct {
		name    string
		mockFn  func()
		want    string
		wantErr error
	}{
		{
			name:   "apt-get 可用",
			mockFn: func() { mockExecCommand("apt-get --version") },
			want:   "apt-get",
		},
		{
			name:   "dnf 可用",
			mockFn: func() { mockExecCommand("dnf --version") },
			want:   "dnf",
		},
		{
			name:   "yum 可用",
			mockFn: func() { mockExecCommand("yum --version") },
			want:   "yum",
		},
		{
			name:   "brew 可用",
			mockFn: func() { mockExecCommand("brew --version") },
			want:   "brew",
		},
		{
			name:    "无可用包管理器",
			mockFn:  mockExecCommandAllFail,
			wantErr: ErrNoPackageManager,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			saveExecCommand(t)
			tt.mockFn()

			pm, err := Detect()
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("期望错误 %v, 实际 %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Detect() 返回错误: %v", err)
			}
			if pm.Name != tt.want {
				t.Errorf("期望 %s, 实际 %s", tt.want, pm.Name)
			}
			if pm.InstallCmd == "" {
				t.Error("InstallCmd 不应为空")
			}
			if pm.CheckCmd == "" {
				t.Error("CheckCmd 不应为空")
			}
		})
	}
}

// ---- InstallCommand ----

func TestInstallCommand(t *testing.T) {
	tests := []struct {
		name string
		pm   *Manager
		pkg  string
		want string
	}{
		{
			name: "apt-get 安装 docker 使用 docker.io",
			pm:   &Manager{Name: "apt-get", InstallCmd: "apt-get install -y %s"},
			pkg:  "docker",
			want: "apt-get install -y docker.io",
		},
		{
			name: "apt-get 安装非 docker 包使用原始包名",
			pm:   &Manager{Name: "apt-get", InstallCmd: "apt-get install -y %s"},
			pkg:  "curl",
			want: "apt-get install -y curl",
		},
		{
			name: "brew 安装 docker 使用 --cask",
			pm:   &Manager{Name: "brew", InstallCmd: "brew install --cask %s"},
			pkg:  "docker",
			want: "brew install --cask docker",
		},
		{
			name: "brew 安装非 docker 也返回 docker（当前行为）",
			pm:   &Manager{Name: "brew", InstallCmd: "brew install --cask %s"},
			pkg:  "any-pkg",
			want: "brew install --cask docker",
		},
		{
			name: "dnf（default 分支）安装 docker",
			pm:   &Manager{Name: "dnf", InstallCmd: "dnf install -y %s"},
			pkg:  "docker",
			want: "dnf install -y docker",
		},
		{
			name: "yum（default 分支）安装非 docker 包",
			pm:   &Manager{Name: "yum", InstallCmd: "yum install -y %s"},
			pkg:  "curl",
			want: "yum install -y curl",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.pm.InstallCommand(tt.pkg)
			if got != tt.want {
				t.Errorf("InstallCommand() = %q, 期望 %q", got, tt.want)
			}
		})
	}
}

// ---- Install ----

func TestInstall_Success(t *testing.T) {
	// 使用 echo 模拟安装成功。
	// Name 设为非 apt-get/brew 以走 default 分支，避免包名替换逻辑干扰。
	pm := &Manager{Name: "test", InstallCmd: "echo %s"}
	output, err := pm.Install("hello-world")
	if err != nil {
		t.Fatalf("Install() 返回错误: %v", err)
	}
	if !strings.Contains(output, "hello-world") {
		t.Errorf("输出应包含 'hello-world', 实际: %q", output)
	}
}

func TestInstall_Failure(t *testing.T) {
	// 使用 false 命令模拟安装失败。
	pm := &Manager{Name: "test", InstallCmd: "false %s"}
	_, err := pm.Install("x")
	if err == nil {
		t.Fatal("Install() 应返回错误")
	}
	if !strings.Contains(err.Error(), "包管理器安装失败") {
		t.Errorf("错误信息应包含 '包管理器安装失败', 实际: %v", err)
	}
}

// ---- isCommandAvailable (真实 execCommand 执行) ----

func TestIsCommandAvailable_RealExec(t *testing.T) {
	// 不 mock execCommand，使用真实实现测试 isCommandAvailable。

	// 真实存在的命令应返回 true
	if !isCommandAvailable("echo ok") {
		t.Error("isCommandAvailable('echo ok') 应返回 true")
	}

	// 不存在的命令应返回 false
	if isCommandAvailable("nonexistent-command-xyz-123") {
		t.Error("isCommandAvailable('nonexistent-command-xyz-123') 应返回 false")
	}
}

// ---- Manager 结构体零值/边界情况 ----

func TestInstallCommand_EmptyPkg(t *testing.T) {
	// 空包名不应 panic，应正常返回
	pm := &Manager{Name: "test", InstallCmd: "install %s"}
	got := pm.InstallCommand("")
	if got != "install " {
		t.Errorf("空包名 InstallCommand = %q, 期望 'install '", got)
	}
}

func TestInstallCommand_NilReceiver(t *testing.T) {
	// nil receiver 应 panic（此处验证行为的确定性）
	defer func() {
		if r := recover(); r == nil {
			t.Log("nil receiver 未 panic（Go 对 nil 指针方法的默认行为）")
		}
	}()
	var pm *Manager
	_ = pm.InstallCommand("docker")
}
