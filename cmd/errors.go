// Package cmd 包含 AgentForge CLI 的所有命令定义和路由。
package cmd

// ExitCoder 是带有退出码的错误接口。
//
// cmd/root.go 的 Execute() 会检查命令返回的 error 是否实现了此接口，
// 并根据 ExitCode() 的返回值决定进程的退出码。
// runengine.ExitCodeError 也实现了此接口，用于 --run 模式下传递容器退出码。
type ExitCoder interface {
	error
	ExitCode() int
}

// exitCodeError 是带有退出码的命令执行错误。
// 仅在此包内部使用，通过 ExitCoder 接口检测。
type exitCodeError struct {
	code int
	msg  string
}

func (e *exitCodeError) Error() string {
	return e.msg
}

func (e *exitCodeError) ExitCode() int {
	return e.code
}

// newExitCodeError 创建一个带有指定退出码的命令执行错误。
//
// 参数：
//   - code: 退出码（0=成功，1=执行错误，2=参数错误）
//   - msg: 错误信息，建议遵循 NFR-16 格式（原因 + 上下文 + 建议）
func newExitCodeError(code int, msg string) error {
	return &exitCodeError{code: code, msg: msg}
}
