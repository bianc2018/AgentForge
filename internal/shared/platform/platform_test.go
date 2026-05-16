package platform

import (
	"testing"
)

func TestInferPlatform_WindowsKeywords(t *testing.T) {
	tests := []struct {
		baseImage string
		want      string
	}{
		// Windows 关键字检测
		{"mcr.microsoft.com/powershell:lts-nanoserver-1809", PlatformWindows},
		{"mcr.microsoft.com/windows/servercore:ltsc2022", PlatformWindows},
		{"mcr.microsoft.com/powershell:lts-windowsservercore-1809", PlatformWindows},
		// 大小写不敏感
		{"Mcr.Microsoft.Com/Windows/ServerCore:ltsc2022", PlatformWindows},
		{"some-registry/NanoServer:latest", PlatformWindows},
		// Linux 镜像返回空
		{"docker.1ms.run/centos:7", PlatformLinux},
		{"ubuntu:22.04", PlatformLinux},
		{"debian:12", PlatformLinux},
		{"alpine:3.19", PlatformLinux},
		{"fedora:39", PlatformLinux},
		// 空字符串
		{"", PlatformLinux},
	}

	for _, tt := range tests {
		t.Run(tt.baseImage, func(t *testing.T) {
			got := InferPlatform(tt.baseImage)
			if got != tt.want {
				t.Errorf("InferPlatform(%q) = %q, want %q", tt.baseImage, got, tt.want)
			}
		})
	}
}

func TestResolvePlatform_ImageInference(t *testing.T) {
	tests := []struct {
		name          string
		baseImage     string
		daemonOSType  string
		wantPlatform  string
		wantImage     string
		wantErr       bool
	}{
		// 从镜像名推断 Windows
		{
			name:         "Windows 镜像推断",
			baseImage:    "mcr.microsoft.com/powershell:lts-nanoserver-1809",
			daemonOSType: "windows",
			wantPlatform: PlatformWindows,
			wantImage:    "mcr.microsoft.com/powershell:lts-nanoserver-1809",
		},
		// 从镜像名推断 Linux
		{
			name:         "Linux 镜像推断",
			baseImage:    "ubuntu:22.04",
			daemonOSType: "linux",
			wantPlatform: PlatformLinux,
			wantImage:    "ubuntu:22.04",
		},
		// 未指定镜像——Linux daemon 回退
		{
			name:         "Linux daemon 回退",
			baseImage:    "",
			daemonOSType: "linux",
			wantPlatform: PlatformLinux,
			wantImage:    DefaultLinuxBaseImage,
		},
		// 未指定镜像——Windows daemon 回退
		{
			name:         "Windows daemon 回退",
			baseImage:    "",
			daemonOSType: "windows",
			wantPlatform: PlatformWindows,
			wantImage:    DefaultWindowsBaseImage,
		},
		// Windows 镜像 + Linux daemon → 报错
		{
			name:         "不兼容组合",
			baseImage:    "mcr.microsoft.com/powershell:lts-nanoserver-1809",
			daemonOSType: "linux",
			wantErr:      true,
		},
		// Linux 镜像 + Windows daemon → 通过（Docker Desktop 支持双平台）
		{
			name:         "Linux 镜像在 Windows daemon",
			baseImage:    "ubuntu:22.04",
			daemonOSType: "windows",
			wantPlatform: PlatformLinux,
			wantImage:    "ubuntu:22.04",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			platform, image, err := ResolvePlatform(tt.baseImage, tt.daemonOSType)
			if tt.wantErr {
				if err == nil {
					t.Error("ResolvePlatform() 期望错误但返回 nil")
				}
				return
			}
			if err != nil {
				t.Errorf("ResolvePlatform() 未期望错误: %v", err)
				return
			}
			if platform != tt.wantPlatform {
				t.Errorf("platform = %q, want %q", platform, tt.wantPlatform)
			}
			if image != tt.wantImage {
				t.Errorf("image = %q, want %q", image, tt.wantImage)
			}
		})
	}
}

func TestInferPlatform_EdgeCases(t *testing.T) {
	// 镜像名含 "windows" 但不在 mcr.microsoft.com 域名下
	if got := InferPlatform("my-registry/windows-tool:latest"); got != PlatformWindows {
		t.Errorf("含 windows 关键字的自定义镜像应推断为 Windows: got %q", got)
	}
}
