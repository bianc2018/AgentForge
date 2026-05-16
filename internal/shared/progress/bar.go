package progress

import (
	"fmt"
	"io"
	"strings"
	"sync"
)

// Bar 是基于步骤数的百分比进度条。
//
// TTY 环境下使用 ANSI 控制序列原地刷新渲染进度条；
// 非 TTY 环境每步输出一行文本（如 "[3/5] 下载依赖"）。
// Bar 是线程安全的。
type Bar struct {
	w       io.Writer
	total   int
	current int
	tty     bool
	desc    string
	width   int // 进度条宽度（字符数），默认 30
	mu      sync.Mutex
}

// NewBar 创建新的进度条。
//
// total 为总步骤数，必须 > 0。
func NewBar(w io.Writer, total int) *Bar {
	return &Bar{
		w:     w,
		total: total,
		tty:   IsTTY(w),
		width: 30,
	}
}

// SetDescription 设置进度条描述文本（如 "下载依赖"）。
func (b *Bar) SetDescription(desc string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.desc = desc
}

// Tick 将进度增加 1 并刷新显示。
func (b *Bar) Tick() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.current++
	b.renderLocked()
}

// Set 将进度设置为指定值并刷新显示。
func (b *Bar) Set(n int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.current = n
	b.renderLocked()
}

// Done 标记进度条完成（current = total），输出完成状态并换行。
func (b *Bar) Done() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.current = b.total
	b.renderLocked()
	if b.tty {
		fmt.Fprint(b.w, "\n")
	}
}

// Write 实现 io.Writer，在进度条上方写入日志行。
//
// 在 TTY 模式下，先清除当前进度条行，写入日志，再重绘进度条。
// 非 TTY 模式下直接写入。
func (b *Bar) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.tty && b.total > 0 {
		// 清除当前进度行 -> 写日志 -> 重绘进度条
		fmt.Fprint(b.w, ansiClearLine)
		n, err := b.w.Write(p)
		b.renderLocked()
		return n, err
	}
	return b.w.Write(p)
}

// renderLocked 在持有锁的情况下渲染进度条。
func (b *Bar) renderLocked() {
	if b.total == 0 {
		return
	}

	pct := b.current * 100 / b.total
	if pct > 100 {
		pct = 100
	}

	if b.tty {
		b.renderANSILocked(pct)
	} else {
		b.renderPlainLocked(pct)
	}
}

// renderANSILocked TTY 模式下使用 ANSI 原地刷新。
func (b *Bar) renderANSILocked(pct int) {
	filled := b.width * b.current / b.total
	bar := strings.Repeat("=", filled)
	if filled < b.width && b.current < b.total {
		bar += ">"
		bar += strings.Repeat(" ", b.width-filled-1)
	}

	line := fmt.Sprintf("%s[%s] %d/%d %3d%%", ansiClearLine, bar, b.current, b.total, pct)
	if b.desc != "" {
		line = fmt.Sprintf("%s%s %s", ansiClearLine, b.desc, bar)
		line = fmt.Sprintf("%s %d/%d", line, b.current, b.total)
	}
	fmt.Fprint(b.w, line)
}

// renderPlainLocked 非 TTY 模式下输出一行文本。
func (b *Bar) renderPlainLocked(pct int) {
	if b.desc != "" {
		fmt.Fprintf(b.w, "[%d/%d] %s (%d%%)\n", b.current, b.total, b.desc, pct)
	} else {
		fmt.Fprintf(b.w, "[%d/%d] %d%%\n", b.current, b.total, pct)
	}
}
