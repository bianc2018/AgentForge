package main

import (
	"os"
	"testing"
)

// TestMain_Version 直接调用 main()。
// 通过临时设置 os.Args 为 version 命令（不需要 Docker），
// Execute() 会正常返回（version 命令不调用 os.Exit），覆盖 main() 的代码路径。
func TestMain_Version(t *testing.T) {
	origArgs := os.Args
	defer func() { os.Args = origArgs }()
	os.Args = []string{"agent-forge", "version"}

	// main() 调用 cmd.Execute()，version 命令正常运行并返回，
	// 不会触发 os.Exit，因此测试不会中断。
	main()
}
