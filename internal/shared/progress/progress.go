// Package progress 提供终端进度显示组件，支持进度日志、进度条和 spinner 动画三种模式。
//
// 自动检测输出是否为 TTY：TTY 环境下使用 ANSI 控制序列渲染动画/进度条，
// 非 TTY 环境（CI、管道重定向）自动降级为纯文本输出。
package progress

import (
	"io"
	"os"
	"sync"

	"golang.org/x/term"
)

// IsTTY 检测给定的 io.Writer 是否为终端。
//
// 仅当 writer 是 *os.File 且其文件描述符指向终端时返回 true。
// 管道、文件重定向等场景返回 false。
func IsTTY(w io.Writer) bool {
	if f, ok := w.(*os.File); ok {
		return term.IsTerminal(int(f.Fd()))
	}
	return false
}

// Log 是进度日志模式，将文本行直接透传写入底层 io.Writer。
//
// 适用于 Docker 构建日志等无固定步骤数的流式输出场景。
// Log 是线程安全的。
type Log struct {
	w  io.Writer
	mu sync.Mutex
}

// NewLog 创建新的进度日志写入器。
func NewLog(w io.Writer) *Log {
	return &Log{w: w}
}

// Write 实现 io.Writer 接口，将数据直接写入底层 writer。
func (l *Log) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.w.Write(p)
}

// ansiClearLine 清除当前行（ANSI 控制序列）。
const ansiClearLine = "\r\033[K"

// spinnerFrames 定义 spinner 动画的旋转字符帧。
var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
