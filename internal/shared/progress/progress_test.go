package progress

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

func TestLog_Write(t *testing.T) {
	var buf bytes.Buffer
	log := NewLog(&buf)

	n, err := log.Write([]byte("hello\n"))
	if err != nil {
		t.Fatalf("Log.Write() error = %v", err)
	}
	if n != 6 {
		t.Errorf("Log.Write() n = %d, want 6", n)
	}
	if buf.String() != "hello\n" {
		t.Errorf("Log.Write() output = %q, want %q", buf.String(), "hello\n")
	}
}

func TestLog_ThreadSafety(t *testing.T) {
	var buf bytes.Buffer
	log := NewLog(&buf)

	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			log.Write([]byte("x"))
			done <- true
		}()
	}
	for i := 0; i < 10; i++ {
		<-done
	}
	if buf.Len() != 10 {
		t.Errorf("Log concurrent writes: len = %d, want 10", buf.Len())
	}
}

func TestIsTTY(t *testing.T) {
	// os.Stdout might or might not be a TTY — we just check it doesn't panic
	_ = IsTTY(os.Stdout)

	// bytes.Buffer is definitely not a TTY
	var buf bytes.Buffer
	if IsTTY(&buf) {
		t.Error("bytes.Buffer should not be detected as TTY")
	}
}

func TestNewContextWriter_NormalWrite(t *testing.T) {
	ctx := context.Background()
	var buf bytes.Buffer
	w := NewContextWriter(ctx, &buf)

	n, err := w.Write([]byte("test"))
	if err != nil {
		t.Fatalf("NewContextWriter.Write() error = %v", err)
	}
	if n != 4 {
		t.Errorf("NewContextWriter.Write() n = %d, want 4", n)
	}
	if buf.String() != "test" {
		t.Errorf("NewContextWriter.Write() output = %q, want %q", buf.String(), "test")
	}
}

func TestNewContextWriter_Cancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	var buf bytes.Buffer
	w := NewContextWriter(ctx, &buf)

	_, err := w.Write([]byte("test"))
	if err == nil {
		t.Fatal("NewContextWriter.Write() expected error for cancelled context")
	}
	if err != context.Canceled {
		t.Errorf("NewContextWriter.Write() error = %v, want context.Canceled", err)
	}
	if buf.Len() != 0 {
		t.Error("cancelled writer should not write to underlying writer")
	}
}

func TestBar_New(t *testing.T) {
	var buf bytes.Buffer
	bar := NewBar(&buf, 10)
	if bar.total != 10 {
		t.Errorf("bar.total = %d, want 10", bar.total)
	}
	if bar.current != 0 {
		t.Errorf("bar.current = %d, want 0", bar.current)
	}
}

func TestBar_Tick(t *testing.T) {
	var buf bytes.Buffer
	bar := NewBar(&buf, 5)
	for i := 0; i < 5; i++ {
		bar.Tick()
	}
	if bar.current != 5 {
		t.Errorf("bar.current = %d, want 5 after 5 ticks", bar.current)
	}
}

func TestBar_Set(t *testing.T) {
	var buf bytes.Buffer
	bar := NewBar(&buf, 10)
	bar.Set(7)
	if bar.current != 7 {
		t.Errorf("bar.current = %d, want 7", bar.current)
	}
}

func TestBar_NonTTY_Output(t *testing.T) {
	var buf bytes.Buffer
	bar := NewBar(&buf, 3)
	bar.SetDescription("下载依赖")

	bar.Tick()
	output := buf.String()
	if !strings.Contains(output, "[1/3]") {
		t.Errorf("Non-TTY bar output should contain [1/3]: %q", output)
	}
	if !strings.Contains(output, "下载依赖") {
		t.Errorf("Non-TTY bar output should contain description: %q", output)
	}

	bar.Done()
	// Done should print final output
	if !strings.Contains(buf.String(), "[3/3]") {
		t.Errorf("Done output should contain [3/3]: %q", buf.String())
	}
}

func TestBar_Write_NonTTY(t *testing.T) {
	var buf bytes.Buffer
	bar := NewBar(&buf, 5)

	bar.Write([]byte("log line\n"))
	if !strings.Contains(buf.String(), "log line") {
		t.Errorf("Bar.Write() should pass through log lines in non-TTY mode: %q", buf.String())
	}
}

func TestSpinner_NonTTY_StartStop(t *testing.T) {
	var buf bytes.Buffer
	sp := NewSpinner(&buf)

	sp.Start("正在构建...")

	// Give it a moment to write the initial message
	time.Sleep(50 * time.Millisecond)

	sp.Stop()

	output := buf.String()
	if !strings.Contains(output, "正在构建") {
		t.Errorf("Spinner output should contain message: %q", output)
	}
}

func TestSpinner_NonTTY_Success(t *testing.T) {
	var buf bytes.Buffer
	sp := NewSpinner(&buf)

	sp.Start("正在部署...")
	time.Sleep(50 * time.Millisecond)
	sp.Success("部署完成")

	output := buf.String()
	if !strings.Contains(output, "[OK]") {
		t.Errorf("Spinner Success should output [OK]: %q", output)
	}
	if !strings.Contains(output, "部署完成") {
		t.Errorf("Spinner Success should contain message: %q", output)
	}
}

func TestSpinner_NonTTY_Fail(t *testing.T) {
	var buf bytes.Buffer
	sp := NewSpinner(&buf)

	sp.Start("正在连接...")
	time.Sleep(50 * time.Millisecond)
	sp.Fail("连接失败")

	output := buf.String()
	if !strings.Contains(output, "[FAIL]") {
		t.Errorf("Spinner Fail should output [FAIL]: %q", output)
	}
	if !strings.Contains(output, "连接失败") {
		t.Errorf("Spinner Fail should contain message: %q", output)
	}
}

func TestSpinner_DoubleStart(t *testing.T) {
	var buf bytes.Buffer
	sp := NewSpinner(&buf)

	sp.Start("第一次")
	sp.Start("第二次") // should be no-op
	sp.Stop()

	// Should not panic
}

func TestSpinner_Write_DuringSpinner(t *testing.T) {
	var buf bytes.Buffer
	sp := NewSpinner(&buf)

	sp.Start("正在构建...")
	time.Sleep(50 * time.Millisecond)

	sp.Write([]byte("Docker 输出行\n"))
	sp.Stop()

	output := buf.String()
	if !strings.Contains(output, "Docker 输出行") {
		t.Errorf("Spinner.Write() should output log lines: %q", output)
	}
}

func TestSpinner_StopWhenNotRunning(t *testing.T) {
	var buf bytes.Buffer
	sp := NewSpinner(&buf)

	// Should not panic
	sp.Stop()
	sp.Success("done")
	sp.Fail("fail")
}

func TestBar_Write_RetainsOutput(t *testing.T) {
	var buf bytes.Buffer
	bar := NewBar(&buf, 3)

	// Write some log lines during progress bar display
	bar.Write([]byte("Step 1: 下载基础镜像\n"))
	bar.Tick()
	bar.Write([]byte("Step 2: 安装依赖\n"))
	bar.Tick()
	bar.Write([]byte("Step 3: 清理缓存\n"))
	bar.Tick()
	bar.Done()

	output := buf.String()
	if !strings.Contains(output, "Step 1") {
		t.Error("Output should contain log line 'Step 1'")
	}
	if !strings.Contains(output, "Step 2") {
		t.Error("Output should contain log line 'Step 2'")
	}
}

// ---------------------------------------------------------------------------
// IsTTY — pipe file (*os.File but not a terminal)
// ---------------------------------------------------------------------------

func TestIsTTY_PipeFile(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	defer w.Close()

	if IsTTY(w) {
		t.Error("pipe writer should not be detected as TTY")
	}
}

// ---------------------------------------------------------------------------
// Bar — SetDescription
// ---------------------------------------------------------------------------

func TestBar_SetDescription(t *testing.T) {
	var buf bytes.Buffer
	bar := NewBar(&buf, 5)
	bar.SetDescription("测试描述")
	if bar.desc != "测试描述" {
		t.Errorf("bar.desc = %q, want %q", bar.desc, "测试描述")
	}
}

// ---------------------------------------------------------------------------
// Bar — renderLocked edge cases: total==0  and  percent clamp
// ---------------------------------------------------------------------------

func TestBar_TotalZero(t *testing.T) {
	var buf bytes.Buffer
	b := &Bar{w: &buf, total: 0, current: 5}
	// renderLocked should return immediately when total == 0,
	// preventing division-by-zero on the percent calculation.
	b.renderLocked()
	if buf.Len() != 0 {
		t.Errorf("expected no output when total=0, got %q", buf.String())
	}
}

func TestBar_PercentClamp(t *testing.T) {
	var buf bytes.Buffer
	b := &Bar{w: &buf, total: 5, current: 10, desc: "test"}
	b.renderLocked()
	output := buf.String()
	if !strings.Contains(output, "100%") {
		t.Errorf("expected percent clamped to 100%%, got %q", output)
	}
	if strings.Contains(output, "200%") {
		t.Errorf("should not show 200%%, got %q", output)
	}
}

// ---------------------------------------------------------------------------
// Bar — TTY ANSI rendering (table-driven)
// ---------------------------------------------------------------------------

func TestBar_TTY_Render(t *testing.T) {
	tests := []struct {
		name    string
		total   int
		current int
		desc    string
		wantSub string   // substring expected
		wantNot []string // substrings that must NOT appear
	}{
		{
			name:    "incomplete no desc",
			total:   5, current: 2, desc: "",
			wantSub: "2/5",
		},
		{
			name:    "complete no desc",
			total:   5, current: 5, desc: "",
			wantSub: "100%",
		},
		{
			name:    "incomplete with desc",
			total:   5, current: 2, desc: "编译",
			wantSub: "编译",
		},
		{
			name:    "complete with desc",
			total:   5, current: 5, desc: "编译",
			wantSub: "编译",
		},
		{
			name:    "zero current with desc",
			total:   5, current: 0, desc: "初始化",
			wantSub: "初始化",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			b := &Bar{w: &buf, total: tt.total, current: tt.current, desc: tt.desc, tty: true, width: 30}
			b.renderLocked()
			output := buf.String()

			if !strings.Contains(output, tt.wantSub) {
				t.Errorf("output should contain %q, got %q", tt.wantSub, output)
			}
			for _, n := range tt.wantNot {
				if strings.Contains(output, n) {
					t.Errorf("output should NOT contain %q, got %q", n, output)
				}
			}
			if !strings.Contains(output, ansiClearLine) {
				t.Errorf("TTY output should contain ANSI clear line, got %q", output)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Bar — non-TTY rendering without description (else branch in renderPlainLocked)
// ---------------------------------------------------------------------------

func TestBar_NonTTY_NoDesc(t *testing.T) {
	var buf bytes.Buffer
	bar := NewBar(&buf, 3)
	// No description set

	bar.Tick()
	output := buf.String()

	if !strings.Contains(output, "[1/3]") {
		t.Errorf("expected [1/3] in output, got %q", output)
	}
	// Without desc the format is "[N/M] P%", NOT "(P%)"
	if strings.Contains(output, "(33%)") {
		t.Errorf("without desc should NOT use parenthesized percent format, got %q", output)
	}
	if !strings.Contains(output, "33%") {
		t.Errorf("without desc should contain percent, got %q", output)
	}
}

// ---------------------------------------------------------------------------
// Bar — TTY Done (newline appended)
// ---------------------------------------------------------------------------

func TestBar_TTY_Done(t *testing.T) {
	var buf bytes.Buffer
	bar := NewBar(&buf, 5)
	bar.tty = true

	bar.Tick()
	bar.Done()

	output := buf.String()
	t.Logf("TTY Done output: %q", output)

	if !strings.Contains(output, "\n") {
		t.Error("TTY Done should append a newline")
	}
	if !strings.Contains(output, "5/5") {
		t.Error("TTY Done should show 5/5")
	}
}

// ---------------------------------------------------------------------------
// Bar — TTY Write (clear + write + re-render)
// ---------------------------------------------------------------------------

func TestBar_TTY_Write(t *testing.T) {
	var buf bytes.Buffer
	bar := NewBar(&buf, 5)
	bar.tty = true

	n, err := bar.Write([]byte("log message\n"))
	if err != nil {
		t.Fatal(err)
	}
	if n != 12 {
		t.Fatalf("Write n = %d, want 12", n)
	}

	output := buf.String()
	t.Logf("TTY Bar Write output: %q", output)

	if !strings.Contains(output, ansiClearLine) {
		t.Error("TTY Write should contain ANSI clear line")
	}
	if !strings.Contains(output, "log message") {
		t.Error("TTY Write should contain the written data")
	}
	if !strings.Contains(output, "0/5") {
		t.Error("TTY Write should re-render bar with progress after writing")
	}
}

// ---------------------------------------------------------------------------
// Spinner — renderFrame
// ---------------------------------------------------------------------------

func TestSpinner_renderFrame(t *testing.T) {
	sp := NewSpinner(&bytes.Buffer{})
	sp.message = "test"

	result0 := sp.renderFrame(0)
	want0 := "⠋ test"
	if result0 != want0 {
		t.Errorf("renderFrame(0) = %q, want %q", result0, want0)
	}

	result9 := sp.renderFrame(9)
	want9 := "⠏ test"
	if result9 != want9 {
		t.Errorf("renderFrame(9) = %q, want %q", result9, want9)
	}
}

// ---------------------------------------------------------------------------
// Spinner — TTY Stop
// ---------------------------------------------------------------------------

func TestSpinner_TTY_Stop(t *testing.T) {
	var buf bytes.Buffer
	sp := &Spinner{
		w:      &buf,
		tty:    true,
		active: true,
		stopCh: make(chan struct{}),
		doneCh: make(chan struct{}),
	}
	// Simulate a running background goroutine
	go func() {
		defer close(sp.doneCh)
		<-sp.stopCh
	}()

	sp.Stop()

	if !strings.Contains(buf.String(), ansiClearLine) {
		t.Errorf("TTY Stop should output clear line, got: %q", buf.String())
	}
}

// ---------------------------------------------------------------------------
// Spinner — TTY Success
// ---------------------------------------------------------------------------

func TestSpinner_TTY_Success(t *testing.T) {
	var buf bytes.Buffer
	sp := &Spinner{
		w:      &buf,
		tty:    true,
		active: true,
		stopCh: make(chan struct{}),
		doneCh: make(chan struct{}),
	}
	go func() {
		defer close(sp.doneCh)
		<-sp.stopCh
	}()

	sp.Success("部署完成")

	output := buf.String()
	if !strings.Contains(output, "✓") {
		t.Errorf("TTY Success should contain checkmark, got: %q", output)
	}
	if !strings.Contains(output, "部署完成") {
		t.Errorf("TTY Success should contain message, got: %q", output)
	}
}

// ---------------------------------------------------------------------------
// Spinner — TTY Fail
// ---------------------------------------------------------------------------

func TestSpinner_TTY_Fail(t *testing.T) {
	var buf bytes.Buffer
	sp := &Spinner{
		w:      &buf,
		tty:    true,
		active: true,
		stopCh: make(chan struct{}),
		doneCh: make(chan struct{}),
	}
	go func() {
		defer close(sp.doneCh)
		<-sp.stopCh
	}()

	sp.Fail("连接失败")

	output := buf.String()
	if !strings.Contains(output, "✗") {
		t.Errorf("TTY Fail should contain cross mark, got: %q", output)
	}
	if !strings.Contains(output, "连接失败") {
		t.Errorf("TTY Fail should contain message, got: %q", output)
	}
}

// ---------------------------------------------------------------------------
// Spinner — TTY Write (TTY branch of Write)
// ---------------------------------------------------------------------------

func TestSpinner_TTY_Write(t *testing.T) {
	var buf bytes.Buffer
	sp := &Spinner{
		w:       &buf,
		tty:     true,
		active:  true,
		message: "loading",
	}

	n, err := sp.Write([]byte("log line\n"))
	if err != nil {
		t.Fatal(err)
	}
	if n != 9 {
		t.Fatalf("Write n = %d, want 9", n)
	}

	output := buf.String()
	if !strings.Contains(output, ansiClearLine) {
		t.Errorf("TTY Write should contain ANSI clear line, got: %q", output)
	}
	if !strings.Contains(output, "log line") {
		t.Errorf("TTY Write should contain written data, got: %q", output)
	}
	// Must re-render spinner frame after writing
	if !strings.Contains(output, "⠋") {
		t.Errorf("TTY Write should re-render spinner frame, got: %q", output)
	}
}

// ---------------------------------------------------------------------------
// Spinner — TTY animation loop (run + runTTY)
// ---------------------------------------------------------------------------

func TestSpinner_TTY_Animation(t *testing.T) {
	var buf bytes.Buffer
	sp := &Spinner{
		w:       &buf,
		tty:     true,
		message: "工作中",
		stopCh:  make(chan struct{}),
		doneCh:  make(chan struct{}),
	}

	// run() dispatches to runTTY() when tty==true
	go sp.run()

	// Allow at least one ticker tick (100 ms interval)
	time.Sleep(250 * time.Millisecond)

	close(sp.stopCh)
	<-sp.doneCh

	output := buf.String()
	t.Logf("TTY animation output (%d bytes): %q", len(output), output[:min(len(output), 200)])

	if !strings.Contains(output, "工作中") {
		t.Errorf("animation output should contain message text, got: %q", output)
	}

	// Should have emitted at least one spinner frame character
	found := false
	for _, f := range spinnerFrames {
		if strings.Contains(output, f) {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("animation output should contain a spinner frame character, got: %q", output)
	}

	if !strings.Contains(output, ansiClearLine) {
		t.Errorf("animation output should contain ANSI clear line, got: %q", output)
	}
}

// ---------------------------------------------------------------------------
// Spinner — Write when not active (else branch)
// ---------------------------------------------------------------------------

func TestSpinner_Write_NotActive(t *testing.T) {
	var buf bytes.Buffer
	sp := NewSpinner(&buf)
	// sp is not active and not TTY

	n, err := sp.Write([]byte("hello"))
	if err != nil {
		t.Fatal(err)
	}
	if n != 5 {
		t.Errorf("Write n = %d, want 5", n)
	}
	if buf.String() != "hello" {
		t.Errorf("Write output = %q, want %q", buf.String(), "hello")
	}
}

// min helper for the animation test
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
