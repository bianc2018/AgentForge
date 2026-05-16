package logging

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseLevel_Valid(t *testing.T) {
	tests := []struct {
		input string
		want  Level
	}{
		{"debug", LevelDebug},
		{"DEBUG", LevelDebug},
		{"Debug", LevelDebug},
		{"  debug  ", LevelDebug},
		{"info", LevelInfo},
		{"INFO", LevelInfo},
		{"warn", LevelWarn},
		{"WARNING", LevelWarn},
		{"warning", LevelWarn},
		{"error", LevelError},
		{"ERROR", LevelError},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := parseLevel(tt.input)
			if err != nil {
				t.Errorf("parseLevel(%q) unexpected error: %v", tt.input, err)
			}
			if got != tt.want {
				t.Errorf("parseLevel(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseLevel_Invalid(t *testing.T) {
	tests := []string{"", "verbose", "trace", "123", "log"}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			_, err := parseLevel(input)
			if err == nil {
				t.Errorf("parseLevel(%q) should return error", input)
			}
		})
	}
}

func TestLevelFromEnv(t *testing.T) {
	tests := []struct {
		envVal string
		want   Level
	}{
		{"", LevelInfo},
		{"debug", LevelDebug},
		{"INFO", LevelInfo},
		{"warn", LevelWarn},
		{"invalid", LevelInfo},
	}

	for _, tt := range tests {
		t.Run("env="+tt.envVal, func(t *testing.T) {
			if tt.envVal == "" {
				os.Unsetenv("AGENTFORGE_LOG_LEVEL")
			} else {
				os.Setenv("AGENTFORGE_LOG_LEVEL", tt.envVal)
			}
			got := levelFromEnv()
			if got != tt.want {
				t.Errorf("levelFromEnv() = %v, want %v", got, tt.want)
			}
		})
	}
	os.Unsetenv("AGENTFORGE_LOG_LEVEL")
}

func TestDailyWriter_Rotation(t *testing.T) {
	dir := t.TempDir()
	w, err := newDailyWriter(dir)
	if err != nil {
		t.Fatalf("newDailyWriter() error = %v", err)
	}
	defer w.Close()

	// 写入一条日志
	msg := "test message\n"
	n, err := w.Write([]byte(msg))
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if n != len(msg) {
		t.Errorf("Write() wrote %d bytes, want %d", n, len(msg))
	}

	// 验证文件存在
	files, err := filepath.Glob(filepath.Join(dir, "*.log"))
	if err != nil {
		t.Fatalf("Glob error = %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 log file, got %d: %v", len(files), files)
	}

	// 验证文件内容
	data, err := os.ReadFile(files[0])
	if err != nil {
		t.Fatalf("ReadFile error = %v", err)
	}
	if string(data) != msg {
		t.Errorf("file content = %q, want %q", string(data), msg)
	}
}

func TestDailyWriter_Append(t *testing.T) {
	dir := t.TempDir()
	w, err := newDailyWriter(dir)
	if err != nil {
		t.Fatalf("newDailyWriter() error = %v", err)
	}
	defer w.Close()

	w.Write([]byte("line1\n"))
	w.Write([]byte("line2\n"))

	files, _ := filepath.Glob(filepath.Join(dir, "*.log"))
	if len(files) != 1 {
		t.Fatalf("expected 1 log file, got %d", len(files))
	}

	data, _ := os.ReadFile(files[0])
	if !strings.Contains(string(data), "line1") || !strings.Contains(string(data), "line2") {
		t.Errorf("file should contain both lines, got: %q", string(data))
	}
}

func TestInit_CreatesLogDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "subdir")
	Init(dir)
	Info("hello", "key", "val")

	files, err := filepath.Glob(filepath.Join(dir, "log", "*.log"))
	if err != nil {
		t.Fatalf("Glob error = %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 log file, got %d", len(files))
	}
}

func TestInit_NoErrorOnUnwritableDir(t *testing.T) {
	// /proc 不可写，应静默降级
	Init("/proc/agent-forge-nonexistent")
	// 不应 panic
	Info("should not panic")
}

func TestDebug_NotLoggedAtDefaultLevel(t *testing.T) {
	dir := t.TempDir()
	os.Setenv("AGENTFORGE_LOG_LEVEL", "info")
	defer os.Unsetenv("AGENTFORGE_LOG_LEVEL")

	Init(dir)
	Debug("debug message")
	Info("info message")

	files, _ := filepath.Glob(filepath.Join(dir, "log", "*.log"))
	if len(files) == 0 {
		t.Skip("no log files created (may have failed silently)")
	}
	data, _ := os.ReadFile(files[0])
	if strings.Contains(string(data), "debug message") {
		t.Error("DEBUG message should not appear at INFO level")
	}
	if !strings.Contains(string(data), "info message") {
		t.Error("INFO message should appear at INFO level")
	}
}

func TestDebug_LoggedAtDebugLevel(t *testing.T) {
	dir := t.TempDir()
	os.Setenv("AGENTFORGE_LOG_LEVEL", "debug")
	defer os.Unsetenv("AGENTFORGE_LOG_LEVEL")

	Init(dir)
	Debug("debug message")

	files, _ := filepath.Glob(filepath.Join(dir, "log", "*.log"))
	if len(files) == 0 {
		t.Skip("no log files created")
	}
	data, _ := os.ReadFile(files[0])
	if !strings.Contains(string(data), "debug message") {
		t.Error("DEBUG message should appear at DEBUG level")
	}
}

// saveLogger saves the current globalLogger and returns a restore function.
func saveLogger() func() {
	old := globalLogger
	return func() { globalLogger = old }
}

func TestLevel_String(t *testing.T) {
	tests := []struct {
		level Level
		want  string
	}{
		{LevelDebug, "DEBUG"},
		{LevelInfo, "INFO"},
		{LevelWarn, "WARN"},
		{LevelError, "ERROR"},
		{Level(99), "UNKNOWN"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.level.String(); got != tt.want {
				t.Errorf("Level(%d).String() = %q, want %q", tt.level, got, tt.want)
			}
		})
	}
}

func TestNewDailyWriter_MkdirError(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "not-a-dir")
	if err := os.WriteFile(filePath, []byte(""), 0644); err != nil {
		t.Fatalf("WriteFile error = %v", err)
	}
	_, err := newDailyWriter(filePath)
	if err == nil {
		t.Error("expected error when directory path is a file")
	}
}

func TestDailyWriter_Close_NilFile(t *testing.T) {
	w := &dailyWriter{dir: t.TempDir()}
	if err := w.Close(); err != nil {
		t.Errorf("Close() with nil file should not error, got: %v", err)
	}
}

func TestDailyWriter_WriteError(t *testing.T) {
	dir := t.TempDir()
	w, err := newDailyWriter(dir)
	if err != nil {
		t.Fatalf("newDailyWriter() error = %v", err)
	}
	defer w.Close()

	// First write succeeds
	_, err = w.Write([]byte("first\n"))
	if err != nil {
		t.Fatalf("first Write() error = %v", err)
	}

	// Remove the directory and force date rotation
	os.RemoveAll(dir)
	w.currDate = "2000-01-01"

	// Write should fail: closes old file, then OpenFile fails because dir is gone
	_, err = w.Write([]byte("second\n"))
	if err == nil {
		t.Error("expected error when directory is removed")
	}
}

func TestDailyWriter_DateRotation(t *testing.T) {
	dir := t.TempDir()
	w, err := newDailyWriter(dir)
	if err != nil {
		t.Fatalf("newDailyWriter() error = %v", err)
	}
	defer w.Close()

	// First write - creates today's log file
	_, err = w.Write([]byte("first\n"))
	if err != nil {
		t.Fatalf("first Write() error = %v", err)
	}

	files, _ := filepath.Glob(filepath.Join(dir, "*.log"))
	if len(files) != 1 {
		t.Fatalf("expected 1 log file, got %d", len(files))
	}

	// Force date rotation by setting currDate to the past
	w.currDate = "2000-01-01"

	// Second write triggers rotation: closes old file, opens new file, writes
	_, err = w.Write([]byte("second\n"))
	if err != nil {
		t.Fatalf("second Write() error = %v", err)
	}

	// Note: file names use time.Now() not currDate, so only 1 file exists.
	// This verifies the rotation code path: close old handle + re-open new file.
	files, _ = filepath.Glob(filepath.Join(dir, "*.log"))
	if len(files) != 1 {
		t.Fatalf("expected 1 log file (same date), got %d", len(files))
	}

	// Same-day write should append to the re-opened file (no new file)
	_, err = w.Write([]byte("third\n"))
	if err != nil {
		t.Fatalf("third Write() error = %v", err)
	}

	// All three messages should appear in the single log file
	data, _ := os.ReadFile(files[0])
	if !strings.Contains(string(data), "first") {
		t.Error("file should contain 'first'")
	}
	if !strings.Contains(string(data), "second") {
		t.Error("file should contain 'second'")
	}
	if !strings.Contains(string(data), "third") {
		t.Error("file should contain 'third'")
	}
}

func TestCustomHandler_Enabled(t *testing.T) {
	tests := []struct {
		handlerLevel Level
		slogLevel    slog.Level
		want         bool
	}{
		// Handler at LevelDebug
		{LevelDebug, slog.LevelDebug, true},
		{LevelDebug, slog.LevelInfo, true},
		{LevelDebug, slog.LevelWarn, true},
		{LevelDebug, slog.LevelError, true},
		{LevelDebug, slog.Level(99), true}, // unrecognized level -> default true
		// Handler at LevelInfo
		{LevelInfo, slog.LevelDebug, false},
		{LevelInfo, slog.LevelInfo, true},
		{LevelInfo, slog.LevelWarn, true},
		{LevelInfo, slog.LevelError, true},
		// Handler at LevelWarn
		{LevelWarn, slog.LevelDebug, false},
		{LevelWarn, slog.LevelInfo, false},
		{LevelWarn, slog.LevelWarn, true},
		{LevelWarn, slog.LevelError, true},
		// Handler at LevelError
		{LevelError, slog.LevelDebug, false},
		{LevelError, slog.LevelInfo, false},
		{LevelError, slog.LevelWarn, false},
		{LevelError, slog.LevelError, true},
	}
	for _, tt := range tests {
		name := "handler=" + tt.handlerLevel.String() + "/slog=" + tt.slogLevel.String()
		t.Run(name, func(t *testing.T) {
			h := &customHandler{level: tt.handlerLevel}
			got := h.Enabled(context.Background(), tt.slogLevel)
			if got != tt.want {
				t.Errorf("Enabled(%v) = %v, want %v", tt.slogLevel, got, tt.want)
			}
		})
	}
}

func TestCustomHandler_WithAttrs(t *testing.T) {
	var buf bytes.Buffer
	baseHandler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	h := &customHandler{handler: baseHandler, level: LevelDebug}

	h2 := h.WithAttrs([]slog.Attr{slog.String("test_key", "test_val")})
	logger := slog.New(h2)
	logger.Info("msg")

	if !strings.Contains(buf.String(), "test_val") {
		t.Error("WithAttrs did not propagate attrs to handler output")
	}
}

func TestCustomHandler_WithGroup(t *testing.T) {
	var buf bytes.Buffer
	baseHandler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	h := &customHandler{handler: baseHandler, level: LevelDebug}

	h2 := h.WithGroup("test_group")
	logger := slog.New(h2)
	logger.Info("msg", slog.String("key", "val"))

	if !strings.Contains(buf.String(), "test_group") {
		t.Error("WithGroup did not propagate group name to handler output")
	}
}

func TestWarn_Error_Logged(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("AGENTFORGE_LOG_LEVEL", "debug")
	defer saveLogger()()

	Init(dir)
	Warn("warning message", "count", 42)
	Error("error message", "err", "something failed")

	files, _ := filepath.Glob(filepath.Join(dir, "log", "*.log"))
	if len(files) == 0 {
		t.Skip("no log files created")
	}
	data, _ := os.ReadFile(files[0])
	if !strings.Contains(string(data), "warning message") {
		t.Error("Warn message should appear in log output")
	}
	if !strings.Contains(string(data), "error message") {
		t.Error("Error message should appear in log output")
	}
}

func TestLogger_NilSafety(t *testing.T) {
	defer saveLogger()()
	globalLogger = nil

	// These should not panic when globalLogger is nil
	Debug("debug message")
	Info("info message")
	Warn("warn message")
	Error("error message")
}
