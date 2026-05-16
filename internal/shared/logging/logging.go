// Package logging 提供按日期轮转的文件日志功能。
//
// 日志写入 <config-dir>/log/<YYYY-MM-DD>.log，每天自动切换文件。
// 日志级别通过环境变量 AGENTFORGE_LOG_LEVEL 配置（debug/info/warn/error），默认 info。
// 初始化失败时静默降级，不阻塞命令执行。
package logging

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Level 表示日志级别。
type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

// String 返回日志级别的字符串表示。
func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// parseLevel 从字符串解析日志级别，不区分大小写。
func parseLevel(s string) (Level, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return LevelDebug, nil
	case "info":
		return LevelInfo, nil
	case "warn", "warning":
		return LevelWarn, nil
	case "error":
		return LevelError, nil
	default:
		return LevelInfo, fmt.Errorf("无效的日志级别: %q，使用默认值 info", s)
	}
}

// levelFromEnv 从环境变量 AGENTFORGE_LOG_LEVEL 读取日志级别，默认 info。
func levelFromEnv() Level {
	val := os.Getenv("AGENTFORGE_LOG_LEVEL")
	if val == "" {
		return LevelInfo
	}
	level, err := parseLevel(val)
	if err != nil {
		return LevelInfo
	}
	return level
}

// dailyWriter 实现按天轮转的文件写入器。
//
// 每次 Write 前检查当前日期，若日期变化则关闭旧文件、打开新文件。
// 所有公开方法均为线程安全。
type dailyWriter struct {
	mu       sync.Mutex
	dir      string
	currDate string
	file     *os.File
}

// newDailyWriter 创建按天轮转的文件写入器。
func newDailyWriter(dir string) (*dailyWriter, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("创建日志目录失败: %w", err)
	}
	return &dailyWriter{dir: dir}, nil
}

// Write 实现 io.Writer 接口。
func (w *dailyWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	today := time.Now().Format("2006-01-02")
	if w.currDate != today {
		if w.file != nil {
			w.file.Close()
		}
		path := filepath.Join(w.dir, today+".log")
		f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return 0, fmt.Errorf("打开日志文件失败: %w", err)
		}
		w.file = f
		w.currDate = today
	}

	return w.file.Write(p)
}

// Close 关闭当前日志文件。
func (w *dailyWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file != nil {
		return w.file.Close()
	}
	return nil
}

// customHandler 实现 slog.Handler，支持自定义级别过滤。
type customHandler struct {
	handler slog.Handler
	level   Level
}

func (h *customHandler) Enabled(_ context.Context, level slog.Level) bool {
	switch level {
	case slog.LevelDebug:
		return h.level <= LevelDebug
	case slog.LevelInfo:
		return h.level <= LevelInfo
	case slog.LevelWarn:
		return h.level <= LevelWarn
	case slog.LevelError:
		return h.level <= LevelError
	}
	return true
}

func (h *customHandler) Handle(ctx context.Context, r slog.Record) error {
	return h.handler.Handle(ctx, r)
}

func (h *customHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &customHandler{handler: h.handler.WithAttrs(attrs), level: h.level}
}

func (h *customHandler) WithGroup(name string) slog.Handler {
	return &customHandler{handler: h.handler.WithGroup(name), level: h.level}
}

var globalLogger *slog.Logger

// Init 初始化全局日志系统。
//
// configDir 为配置目录路径。日志文件写入 <configDir>/log/<YYYY-MM-DD>.log。
// 日志级别由环境变量 AGENTFORGE_LOG_LEVEL 控制。
//
// 初始化失败时静默失败，后续日志调用不会 panic。
func Init(configDir string) {
	level := levelFromEnv()

	logDir := filepath.Join(configDir, "log")
	writer, err := newDailyWriter(logDir)
	if err != nil {
		// 静默失败：日志初始化失败不阻塞命令
		globalLogger = slog.New(slog.NewJSONHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelDebug}))
		return
	}

	baseHandler := slog.NewJSONHandler(writer, &slog.HandlerOptions{
		Level: slog.LevelDebug, // 由 customHandler 控制实际级别
	})
	handler := &customHandler{handler: baseHandler, level: level}
	globalLogger = slog.New(handler)

	Info("日志系统已初始化", "dir", logDir, "log_level", level.String())
}

// Debug 记录 DEBUG 级别日志。
func Debug(msg string, args ...any) {
	if globalLogger != nil {
		globalLogger.Debug(msg, args...)
	}
}

// Info 记录 INFO 级别日志。
func Info(msg string, args ...any) {
	if globalLogger != nil {
		globalLogger.Info(msg, args...)
	}
}

// Warn 记录 WARN 级别日志。
func Warn(msg string, args ...any) {
	if globalLogger != nil {
		globalLogger.Warn(msg, args...)
	}
}

// Error 记录 ERROR 级别日志。
func Error(msg string, args ...any) {
	if globalLogger != nil {
		globalLogger.Error(msg, args...)
	}
}
