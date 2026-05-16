package progress

import (
	"context"
	"io"
)

// contextWriter 包装 io.Writer，在每次写入前检查 context 是否取消。
//
// 如果 context 已取消，Write 返回 context 错误而不执行实际写入。
// 用于在长时间 I/O 操作（如 io.Copy）中响应 Ctrl+C 中断。
type contextWriter struct {
	ctx context.Context
	w   io.Writer
}

// NewContextWriter 创建 context 感知的 writer。
//
// 每次 Write 调用前检查 ctx.Done()，若已取消则返回 ctx.Err()。
// 这是一个轻量级包装，不会缓冲数据。
func NewContextWriter(ctx context.Context, w io.Writer) io.Writer {
	return &contextWriter{ctx: ctx, w: w}
}

// Write 实现 io.Writer，先检查 context 再写入。
func (cw *contextWriter) Write(p []byte) (int, error) {
	select {
	case <-cw.ctx.Done():
		return 0, cw.ctx.Err()
	default:
	}
	return cw.w.Write(p)
}
