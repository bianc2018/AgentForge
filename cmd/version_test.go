package cmd

import (
	"strings"
	"testing"
)

// TestVersionInfo_Format 验证 VersionInfo 输出格式。
//
// 覆盖案例：正常路径输出 "agent-forge X.Y.Z (hash)"。
func TestVersionInfo_Format(t *testing.T) {
	// 保存并恢复原始值
	origVersion := Version
	origHash := GitHash
	defer func() {
		Version = origVersion
		GitHash = origHash
	}()

	Version = "1.2.3"
	GitHash = "abcdef12"

	info := VersionInfo()

	if !strings.HasPrefix(info, "agent-forge ") {
		t.Errorf("输出应以 'agent-forge ' 开头, 实际: %s", info)
	}
	if !strings.Contains(info, "1.2.3") {
		t.Errorf("输出应包含版本号 1.2.3, 实际: %s", info)
	}
	if !strings.Contains(info, "abcdef12") {
		t.Errorf("输出应包含 git hash abcdef12, 实际: %s", info)
	}
}

// TestVersionInfo_UnknownHash 验证 git hash 为 unknown 时的输出。
//
// 覆盖案例：空 hash 输出 "(unknown)"。
func TestVersionInfo_UnknownHash(t *testing.T) {
	origVersion := Version
	origHash := GitHash
	defer func() {
		Version = origVersion
		GitHash = origHash
	}()

	Version = "1.0.0"
	GitHash = "unknown"

	info := VersionInfo()

	if !strings.Contains(info, "(unknown)") {
		t.Errorf("git hash 为 unknown 时应输出 '(unknown)', 实际: %s", info)
	}
}
