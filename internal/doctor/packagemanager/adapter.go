// Package packagemanager 提供包管理器的自动识别和安装指令生成。
//
// 自动识别当前操作系统的包管理器（优先顺序：apt-get、dnf、yum、brew），
// 并为缺失的核心依赖（如 docker）生成对应的安装命令。
// 遵循 NFR-19（自动安装缺失的核心依赖）。
package packagemanager

import (
	"errors"
	"fmt"
	"os/exec"
)

// ErrNoPackageManager 表示系统上没有可用的包管理器。
var ErrNoPackageManager = errors.New("未找到可用的包管理器（已检查 apt-get、dnf、yum、brew）")

// Manager 表示一个包管理器及其相关信息。
type Manager struct {
	// Name 是包管理器的名称（如 "apt-get"）。
	Name string
	// InstallCmd 是安装指定包的命令模板。
	// %s 将被替换为包名称（如 "docker.io"、"docker"）。
	InstallCmd string
	// CheckCmd 是检查包管理器是否可用的命令。
	CheckCmd string
}

// Detect 返回当前系统上可用的第一个包管理器。
//
// 检测顺序：
//  1. apt-get（Debian/Ubuntu）
//  2. dnf（Fedora/RHEL 8+）
//  3. yum（CentOS 7/RHEL 7）
//  4. brew（macOS）
//
// 返回的 Manager 可用于后续的安装操作。
// 如果无可用的包管理器，返回 ErrNoPackageManager。
func Detect() (*Manager, error) {
	managers := []Manager{
		{
			Name:       "apt-get",
			InstallCmd: "apt-get install -y %s",
			CheckCmd:   "apt-get --version",
		},
		{
			Name:       "dnf",
			InstallCmd: "dnf install -y %s",
			CheckCmd:   "dnf --version",
		},
		{
			Name:       "yum",
			InstallCmd: "yum install -y %s",
			CheckCmd:   "yum --version",
		},
		{
			Name:       "brew",
			InstallCmd: "brew install --cask %s",
			CheckCmd:   "brew --version",
		},
	}

	for _, m := range managers {
		if isCommandAvailable(m.CheckCmd) {
			return &m, nil
		}
	}

	return nil, ErrNoPackageManager
}

// InstallCommand 返回安装指定包名的完整命令字符串。
//
// 根据包管理器类型选择合适的包名：
//   - apt-get 安装 docker.io（而非 docker）
//   - dnf/yum 安装 docker
//   - brew 安装 docker（--cask）
//
// 示例：
//
//	pm.InstallCommand("docker") → "apt-get install -y docker.io"
func (pm *Manager) InstallCommand(pkg string) string {
	switch pm.Name {
	case "apt-get":
		if pkg == "docker" {
			return fmt.Sprintf(pm.InstallCmd, "docker.io")
		}
	case "brew":
		return fmt.Sprintf(pm.InstallCmd, "docker")
	default:
		return fmt.Sprintf(pm.InstallCmd, pkg)
	}
	return fmt.Sprintf(pm.InstallCmd, pkg)
}

// Install 执行包安装操作。
//
// 通过 exec.Command 调用系统包管理器安装指定的包。
// 返回安装过程中的输出和可能的错误。
func (pm *Manager) Install(pkg string) (string, error) {
	cmdStr := pm.InstallCommand(pkg)

	// 将命令字符串拆分为命令和参数
	// 格式如："apt-get install -y docker.io"
	cmd := exec.Command("sh", "-c", cmdStr)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("包管理器安装失败: %s\n命令: %s\n输出: %s", err.Error(), cmdStr, string(output))
	}

	return string(output), nil
}

// execCommand 是可 mock 的 exec.Command 封装。
// 默认实现使用 os/exec.Command。
var execCommand = func(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	return cmd.Run()
}

// isCommandAvailable 检查系统中是否存在指定命令。
func isCommandAvailable(checkCmd string) bool {
	// checkCmd 格式如："apt-get --version"
	return execCommand("sh", "-c", checkCmd) == nil
}
