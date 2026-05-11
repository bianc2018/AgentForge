package packagemanager

import (
	"errors"
	"testing"
)

// saveExecCommand 保存当前 execCommand 并在测试后恢复。
func saveExecCommand(t *testing.T) {
	t.Helper()
	orig := execCommand
	t.Cleanup(func() {
		execCommand = orig
	})
}

func mockExecCommand(okCheck string) {
	execCommand = func(name string, args ...string) error {
		if name == "sh" && len(args) > 1 && args[1] == okCheck {
			return nil
		}
		return errors.New("not found")
	}
}

func mockExecCommandAllFail() {
	execCommand = func(name string, args ...string) error {
		return errors.New("not found")
	}
}

// --- UT-16: Detect ---

func TestDetect_AptGetAvailable(t *testing.T) {
	saveExecCommand(t)
	mockExecCommand("apt-get --version")

	pm, err := Detect()
	if err != nil {
		t.Fatalf("Detect() 返回错误: %v", err)
	}
	if pm.Name != "apt-get" {
		t.Errorf("期望 apt-get, 实际 %s", pm.Name)
	}
}

func TestDetect_DnfAvailable(t *testing.T) {
	saveExecCommand(t)
	mockExecCommand("dnf --version")

	pm, err := Detect()
	if err != nil {
		t.Fatalf("Detect() 返回错误: %v", err)
	}
	if pm.Name != "dnf" {
		t.Errorf("期望 dnf, 实际 %s", pm.Name)
	}
}

func TestDetect_YumAvailable(t *testing.T) {
	saveExecCommand(t)
	mockExecCommand("yum --version")

	pm, err := Detect()
	if err != nil {
		t.Fatalf("Detect() 返回错误: %v", err)
	}
	if pm.Name != "yum" {
		t.Errorf("期望 yum, 实际 %s", pm.Name)
	}
}

func TestDetect_BrewAvailable(t *testing.T) {
	saveExecCommand(t)
	mockExecCommand("brew --version")

	pm, err := Detect()
	if err != nil {
		t.Fatalf("Detect() 返回错误: %v", err)
	}
	if pm.Name != "brew" {
		t.Errorf("期望 brew, 实际 %s", pm.Name)
	}
}

func TestDetect_NoPackageManager(t *testing.T) {
	saveExecCommand(t)
	mockExecCommandAllFail()

	_, err := Detect()
	if err == nil {
		t.Fatal("Detect() 应返回错误")
	}
	if !errors.Is(err, ErrNoPackageManager) {
		t.Errorf("错误应为 ErrNoPackageManager, 实际 %v", err)
	}
}

// --- InstallCommand tests ---

func TestInstallCommand_AptGetDocker(t *testing.T) {
	pm := &Manager{Name: "apt-get", InstallCmd: "apt-get install -y %s"}
	cmd := pm.InstallCommand("docker")
	if cmd != "apt-get install -y docker.io" {
		t.Errorf("apt-get 安装 docker 应使用 docker.io, 实际: %s", cmd)
	}
}

func TestInstallCommand_YumDocker(t *testing.T) {
	pm := &Manager{Name: "yum", InstallCmd: "yum install -y %s"}
	cmd := pm.InstallCommand("docker")
	if cmd != "yum install -y docker" {
		t.Errorf("yum 安装 docker 应使用 docker, 实际: %s", cmd)
	}
}

func TestInstallCommand_BrewDocker(t *testing.T) {
	pm := &Manager{Name: "brew", InstallCmd: "brew install --cask %s"}
	cmd := pm.InstallCommand("docker")
	if cmd != "brew install --cask docker" {
		t.Errorf("brew 安装 docker: 实际: %s", cmd)
	}
}
