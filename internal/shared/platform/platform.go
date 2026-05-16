// Package platform 提供跨平台能力——从基础镜像名称推断目标平台、
// Docker daemon 类型检测和平台兼容性校验。
//
// 平台推断规则：
//   - 镜像名含 windows/nanoserver/servercore → "windows"
//   - 其他所有情况 → ""（Linux 默认）
//
// 未指定基础镜像时，通过 Docker daemon OSType 回退选择默认镜像和平台。
package platform

import (
	"fmt"
	"strings"
)

const (
	// PlatformWindows 表示 Windows 容器平台。
	PlatformWindows = "windows"
	// PlatformLinux 表示 Linux 容器平台（空字符串保持向后兼容）。
	PlatformLinux = ""

	// DefaultLinuxBaseImage 是 Linux 平台的默认基础镜像。
	DefaultLinuxBaseImage = "docker.1ms.run/centos:7"
	// DefaultWindowsBaseImage 是 Windows 平台的默认基础镜像。
	DefaultWindowsBaseImage = "mcr.microsoft.com/powershell:lts-nanoserver-1809"
)

// windowsKeywords 是用于推断 Windows 平台的镜像名关键词。
var windowsKeywords = []string{"windows", "nanoserver", "servercore"}

// InferPlatform 根据基础镜像名称推断目标平台。
//
// 规则：
//   - 镜像名含 windows/nanoserver/servercore（不区分大小写）→ "windows"
//   - 其他所有情况 → ""（Linux）
func InferPlatform(baseImage string) string {
	lower := strings.ToLower(baseImage)
	for _, kw := range windowsKeywords {
		if strings.Contains(lower, kw) {
			return PlatformWindows
		}
	}
	return PlatformLinux
}

// ResolvePlatform 合并镜像推断和 daemon 回退，返回最终平台和默认镜像。
//
// 流程：
//  1. 如果 baseImage 非空，从镜像名推断平台
//  2. 如果 baseImage 为空，根据 daemonOSType 选择默认镜像和平台
//  3. 校验兼容性：Windows 镜像 + Linux daemon → 错误
//
// 返回 (platform, defaultImage, error)。
func ResolvePlatform(baseImage, daemonOSType string) (string, string, error) {
	var platform string

	if baseImage != "" {
		platform = InferPlatform(baseImage)
	} else {
		// 未指定基础镜像，根据 daemon OS 选择默认值
		if strings.EqualFold(daemonOSType, PlatformWindows) {
			platform = PlatformWindows
			baseImage = DefaultWindowsBaseImage
		} else {
			platform = PlatformLinux
			baseImage = DefaultLinuxBaseImage
		}
	}

	// 校验兼容性：Windows 镜像只能在 Windows daemon 上运行
	if platform == PlatformWindows && !strings.EqualFold(daemonOSType, PlatformWindows) {
		return "", "", fmt.Errorf("当前 Docker daemon 不支持 Windows 容器，请在 Windows Docker 主机上运行")
	}

	return platform, baseImage, nil
}
