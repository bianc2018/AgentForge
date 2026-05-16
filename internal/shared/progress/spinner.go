package progress

import (
	"fmt"
	"io"
	"sync"
	"time"
)

// Spinner 是旋转动画指示器，用于耗时不确定的等待任务。
//
// TTY 环境下渲染旋转字符帧动画（帧间隔 100ms）；
// 非 TTY 环境每 5 秒输出一行文本状态更新。
// Spinner 是线程安全的。
type Spinner struct {
	w       io.Writer
	tty     bool
	message string
	active  bool
	mu      sync.Mutex
	stopCh  chan struct{}
	doneCh  chan struct{}
}

// NewSpinner 创建新的 spinner 动画指示器。
func NewSpinner(w io.Writer) *Spinner {
	return &Spinner{
		w: w,
	}
}

// Start 启动 spinner 动画，显示指定消息。
//
// 如果 spinner 已在运行，调用方负责先停止之前的动画。
func (s *Spinner) Start(message string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.active {
		return
	}

	s.message = message
	s.active = true
	s.tty = IsTTY(s.w)
	s.stopCh = make(chan struct{})
	s.doneCh = make(chan struct{})

	go s.run()
}

// Stop 停止 spinner 动画并清除动画行。
func (s *Spinner) Stop() {
	s.mu.Lock()
	if !s.active {
		s.mu.Unlock()
		return
	}
	s.active = false
	s.mu.Unlock()

	close(s.stopCh)
	<-s.doneCh

	if s.tty {
		fmt.Fprint(s.w, ansiClearLine)
	}
}

// Success 停止 spinner 并以成功状态显示最终消息。
func (s *Spinner) Success(message string) {
	s.mu.Lock()
	if !s.active {
		s.mu.Unlock()
		return
	}
	s.active = false
	s.mu.Unlock()

	close(s.stopCh)
	<-s.doneCh

	if s.tty {
		fmt.Fprintf(s.w, "%s✓ %s\n", ansiClearLine, message)
	} else {
		fmt.Fprintf(s.w, "[OK] %s\n", message)
	}
}

// Fail 停止 spinner 并以失败状态显示最终消息。
func (s *Spinner) Fail(message string) {
	s.mu.Lock()
	if !s.active {
		s.mu.Unlock()
		return
	}
	s.active = false
	s.mu.Unlock()

	close(s.stopCh)
	<-s.doneCh

	// 写入最终消息
	s.mu.Lock()
	if s.tty {
		fmt.Fprintf(s.w, "%s✗ %s\n", ansiClearLine, message)
	} else {
		fmt.Fprintf(s.w, "[FAIL] %s\n", message)
	}
	s.mu.Unlock()
}

// Write 实现 io.Writer，在 spinner 上方写入日志行。
//
// TTY 模式下清除当前 spinner 行，写入日志，然后重绘 spinner。
// 非 TTY 模式下直接写入。
func (s *Spinner) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.tty && s.active {
		fmt.Fprint(s.w, ansiClearLine)
		n, err := s.w.Write(p)
		if s.active {
			fmt.Fprint(s.w, s.renderFrame(0))
		}
		return n, err
	}
	return s.w.Write(p)
}

// run 在后台 goroutine 中运行动画循环。
func (s *Spinner) run() {
	defer close(s.doneCh)

	if s.tty {
		s.runTTY()
	} else {
		s.runPlain()
	}
}

// runTTY TTY 模式：100ms 帧间隔的字符动画。
func (s *Spinner) runTTY() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	frameIdx := 0
	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.mu.Lock()
			msg := s.message
			s.mu.Unlock()
			fmt.Fprintf(s.w, "%s%s %s", ansiClearLine, spinnerFrames[frameIdx], msg)
			frameIdx = (frameIdx + 1) % len(spinnerFrames)
		}
	}
}

// runPlain 非 TTY 模式：每 5 秒输出一行文本状态更新。
func (s *Spinner) runPlain() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	// 首次启动时输出
	s.mu.Lock()
	fmt.Fprintf(s.w, "[...] %s\n", s.message)
	s.mu.Unlock()

	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.mu.Lock()
			msg := s.message
			s.mu.Unlock()
			fmt.Fprintf(s.w, "[...] %s (仍在进行中)\n", msg)
		}
	}
}

// renderFrame 返回当前帧的 spinner 字符串（不含回车清除）。
func (s *Spinner) renderFrame(idx int) string {
	return fmt.Sprintf("%s %s", spinnerFrames[idx], s.message)
}
