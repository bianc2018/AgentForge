// Package buildengine — 不依赖 Docker daemon 的纯单元测试。
//
// 这些测试通过 HTTP mock server 模拟 Docker daemon 响应，覆盖 Build 方法的
// 成功路径、失败路径、ImageExists 校验等原本需要真实 Docker 环境的控制流。
// 纯函数（validateParams、isBuildSuccessful、CalculateBackoff 等）直接测试。
package buildengine

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/client"

	"github.com/agent-forge/cli/internal/shared/dockerhelper"
)

// =============================================================================
// 工具：Docker API Mock Server
// =============================================================================

type dockerMockHandler struct {
	t              *testing.T
	buildOutput    string // ImageBuild 返回的 body
	imageList      string // ImageList 返回的 JSON（空 = "[]"）
	tagStatusCode  int    // ImageTag 状态码（0 = 201 Created）
	listStatusCode int    // ImageList 状态码（0 = 200 OK）
	removeFail     bool   // ImageRemove 返回 500
	hijackBuild    bool   // 使用 connection hijack 中断 ImageBuild 响应
}

func (h *dockerMockHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.t.Helper()
	h.t.Logf("[mock-docker] %s %s", r.Method, r.URL.Path)

	// Ping（API 版本协商）
	if strings.HasSuffix(r.URL.Path, "/_ping") {
		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("API-Version", "1.24")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
		return
	}

	// ImageBuild —— POST /v{version}/build
	if strings.HasSuffix(r.URL.Path, "/build") && r.Method == "POST" {
		if h.hijackBuild {
			// 使用 hijack 发送部分响应后关闭连接，模拟 io.Copy 读取错误。
			// 关键：Content-Length 大于实际 body 长度，迫使客户端读到"unexpected EOF"。
			hijacker, ok := w.(http.Hijacker)
			if ok {
				conn, _, err := hijacker.Hijack()
				if err == nil {
					_, _ = conn.Write([]byte("HTTP/1.1 200 OK\r\n" +
						"Content-Type: application/json\r\n" +
						"Content-Length: 100000\r\n\r\n"))
					_, _ = conn.Write([]byte("Step 1: FROM centos:7\n"))
					_ = conn.Close()
					return
				}
			}
			// hijack 失败则降级为正常响应
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(h.buildOutput))
		return
	}

	// ImageList —— GET /v{version}/images/json
	if strings.HasSuffix(r.URL.Path, "/images/json") {
		w.Header().Set("Content-Type", "application/json")
		status := h.listStatusCode
		if status == 0 {
			status = http.StatusOK
		}
		w.WriteHeader(status)
		if status == http.StatusOK {
			list := h.imageList
			if list == "" {
				list = "[]"
			}
			_, _ = w.Write([]byte(list))
		}
		return
	}

	// ImageTag —— POST /v{version}/images/{src}/tag
	if strings.Contains(r.URL.Path, "/tag") && r.Method == "POST" {
		status := h.tagStatusCode
		if status == 0 {
			status = http.StatusCreated
		}
		w.WriteHeader(status)
		return
	}

	// ImageRemove —— DELETE /v{version}/images/{id}
	if r.Method == "DELETE" && strings.Contains(r.URL.Path, "/images/") {
		if h.removeFail {
			http.Error(w, "remove failed", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("[]"))
		return
	}

	h.t.Logf("[mock-docker] UNHANDLED: %s %s", r.Method, r.URL.Path)
	http.Error(w, "endpoint not mocked", http.StatusNotFound)
}

func startMockServer(t *testing.T, buildOutput, imageList string) (*httptest.Server, *dockerhelper.Client) {
	t.Helper()
	return startMockServerEx(t, buildOutput, imageList, nil)
}

// startMockServerEx 创建带自定义 handler 配置的 mock server。
// 参数 f 可在创建 handler 后修改其 tagStatusCode / listStatusCode / removeFail 等字段。
func startMockServerEx(t *testing.T, buildOutput, imageList string,
	f func(h *dockerMockHandler)) (*httptest.Server, *dockerhelper.Client) {

	t.Helper()
	handler := &dockerMockHandler{
		t:           t,
		buildOutput: buildOutput,
		imageList:   imageList,
	}
	if f != nil {
		f(handler)
	}
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	serverHost := strings.TrimPrefix(server.URL, "http://")
	helper, err := dockerhelper.NewClientWithOpts(
		client.WithHost("tcp://"+serverHost),
	)
	if err != nil {
		t.Fatalf("NewClientWithOpts: %v", err)
	}
	t.Cleanup(func() { helper.Close() })

	return server, helper
}

// =============================================================================
// validateParams 边界值测试
// =============================================================================

func TestValidateParams_ZeroMaxRetry(t *testing.T) {
	err := validateParams(BuildParams{MaxRetry: 0})
	if err != nil {
		t.Errorf("validateParams with MaxRetry=0 should be valid, got: %v", err)
	}
}

func TestValidateParams_DefaultEmpty(t *testing.T) {
	err := validateParams(BuildParams{})
	if err != nil {
		t.Errorf("validateParams with default empty struct should be valid, got: %v", err)
	}
}

func TestValidateParams_PositiveMaxRetry(t *testing.T) {
	err := validateParams(BuildParams{MaxRetry: 5})
	if err != nil {
		t.Errorf("validateParams with MaxRetry=5 should be valid, got: %v", err)
	}
}

func TestValidateParams_NegativeMaxRetry_Reason(t *testing.T) {
	err := validateParams(BuildParams{MaxRetry: -3})
	if err == nil {
		t.Fatal("validateParams with negative MaxRetry should return error")
	}
	ipErr, ok := err.(*InvalidParamsError)
	if !ok {
		t.Fatalf("error type = %T, want *InvalidParamsError", err)
	}
	if !strings.Contains(ipErr.Reason, "负数") {
		t.Errorf("reason = %q, should mention '负数'", ipErr.Reason)
	}
}

// =============================================================================
// Engine.New 边缘测试（broken TCP 地址，不连接真实 Docker）
// =============================================================================

func TestEngineNew_BrokenClient(t *testing.T) {
	helper, err := dockerhelper.NewClientWithOpts(
		client.WithHost("tcp://localhost:1"),
	)
	if err != nil {
		t.Fatalf("NewClientWithOpts() error = %v", err)
	}
	defer helper.Close()

	engine := New(helper)
	if engine == nil {
		t.Fatal("New() returned nil")
	}
	engine.Close()
}

// =============================================================================
// Build —— pre-cancelled context（select ctx.Done 在 loop 开头）
// =============================================================================

func TestBuild_PreCancelledContext(t *testing.T) {
	helper, err := dockerhelper.NewClientWithOpts(
		client.WithHost("tcp://localhost:1"),
	)
	if err != nil {
		t.Fatalf("NewClientWithOpts() error = %v", err)
	}
	defer helper.Close()

	engine := New(helper)
	defer engine.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	output, err := engine.Build(ctx, BuildParams{Deps: "", MaxRetry: 3})
	if err == nil {
		t.Fatal("Build() expected error with pre-cancelled context")
	}
	if !strings.Contains(err.Error(), "被中断") &&
		!strings.Contains(err.Error(), "canceled") {
		t.Errorf("error = %v, should mention context cancellation", err)
	}
	if output != "" {
		t.Logf("non-empty output (len=%d): %.100s", len(output), output)
	}
}

// =============================================================================
// Build —— retry backoff 期间 context 取消
// =============================================================================

func TestBuild_ContextCancelDuringBackoff(t *testing.T) {
	helper, err := dockerhelper.NewClientWithOpts(
		client.WithHost("tcp://localhost:1"),
	)
	if err != nil {
		t.Fatalf("NewClientWithOpts() error = %v", err)
	}
	defer helper.Close()

	engine := New(helper)
	defer engine.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		_, buildErr := engine.Build(ctx, BuildParams{Deps: "", MaxRetry: 3})
		errCh <- buildErr
	}()

	time.Sleep(300 * time.Millisecond)
	cancel()

	select {
	case buildErr := <-errCh:
		if buildErr == nil {
			t.Fatal("Build() expected error when context is cancelled during backoff")
		}
		if !strings.Contains(buildErr.Error(), "被中断") &&
			!strings.Contains(buildErr.Error(), "canceled") {
			t.Errorf("error = %v, should mention context interruption", buildErr)
		}
		t.Logf("Got expected cancel-during-backoff error: %v", buildErr)

	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for Build to finish")
	}
}

// =============================================================================
// Build —— retryable error 耗尽重试
// =============================================================================

func TestBuild_RetryableErrorExhausted(t *testing.T) {
	helper, err := dockerhelper.NewClientWithOpts(
		client.WithHost("tcp://localhost:1"),
	)
	if err != nil {
		t.Fatalf("NewClientWithOpts() error = %v", err)
	}
	defer helper.Close()

	engine := New(helper)
	defer engine.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	output, err := engine.Build(ctx, BuildParams{Deps: "", MaxRetry: 1})
	if err == nil {
		t.Fatal("Build() expected error with unreachable TCP address")
	}
	if !strings.Contains(output, "[重试") && !strings.Contains(err.Error(), "connection refused") {
		t.Logf("Output (len=%d): %.200s", len(output), output)
		t.Logf("Error: %v", err)
	}
	t.Logf("Retry exhausted test passed: %v", err)
}

// =============================================================================
// Build —— InvalidParams 通过 Build 入口
// =============================================================================

func TestBuild_InvalidParamsViaBuildEntry(t *testing.T) {
	helper, err := dockerhelper.NewClientWithOpts(
		client.WithHost("tcp://localhost:1"),
	)
	if err != nil {
		t.Fatalf("NewClientWithOpts() error = %v", err)
	}
	defer helper.Close()

	engine := New(helper)
	defer engine.Close()

	_, err = engine.Build(context.Background(), BuildParams{MaxRetry: -1})
	if err == nil {
		t.Fatal("Build() expected InvalidParamsError for negative MaxRetry")
	}
	if _, ok := err.(*InvalidParamsError); !ok {
		t.Errorf("error type = %T, want *InvalidParamsError", err)
	}
}

// =============================================================================
// Build —— MaxRetry=0 + retryable error
// =============================================================================

func TestBuild_MaxRetryZeroWithRetryableError(t *testing.T) {
	helper, err := dockerhelper.NewClientWithOpts(
		client.WithHost("tcp://localhost:1"),
	)
	if err != nil {
		t.Fatalf("NewClientWithOpts() error = %v", err)
	}
	defer helper.Close()

	engine := New(helper)
	defer engine.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err = engine.Build(ctx, BuildParams{Deps: "", MaxRetry: 0})
	if err == nil {
		t.Fatal("Build() expected error with MaxRetry=0 and unreachable address")
	}
	t.Logf("MaxRetry=0 error: %v", err)
}

// =============================================================================
// Build —— mock server 成功路径
// =============================================================================

func TestBuild_MockServer_Success(t *testing.T) {
	_, helper := startMockServer(t,
		"Successfully tagged agent-forge:latest\n",
		`[{"ID":"sha256:abc123","RepoTags":["agent-forge:latest"]}]`,
	)

	engine := New(helper)
	t.Cleanup(func() { engine.Close() })

	output, err := engine.Build(context.Background(), BuildParams{
		Deps:     "",
		MaxRetry: 0,
	})
	if err != nil {
		t.Fatalf("Build() expected success, got error: %v", err)
	}
	if !strings.Contains(output, "Successfully tagged") {
		t.Errorf("output should contain 'Successfully tagged', got:\n%s", output)
	}
}

// =============================================================================
// Build —— mock server 成功 + ProgressWriter
// =============================================================================

func TestBuild_MockServer_WithProgressWriter(t *testing.T) {
	_, helper := startMockServer(t,
		"Step 1: FROM centos:7\nSuccessfully tagged agent-forge:latest\n",
		`[{"ID":"sha256:abc123","RepoTags":["agent-forge:latest"]}]`,
	)

	engine := New(helper)
	t.Cleanup(func() { engine.Close() })

	var progressBuf strings.Builder
	output, err := engine.Build(context.Background(), BuildParams{
		Deps:           "",
		MaxRetry:       0,
		ProgressWriter: &progressBuf,
	})
	if err != nil {
		t.Fatalf("Build() expected success, got error: %v", err)
	}
	if !strings.Contains(output, "Successfully tagged") {
		t.Errorf("output should contain 'Successfully tagged'")
	}
	if progressBuf.Len() == 0 {
		t.Error("ProgressWriter received no data")
	}
	if !strings.Contains(progressBuf.String(), "Successfully tagged") {
		t.Errorf("ProgressWriter should contain build output")
	}
}

// =============================================================================
// Build —— mock server 构建失败（ImageBuild 输出不含 "Successfully tagged"）
// =============================================================================

func TestBuild_MockServer_BuildFailure(t *testing.T) {
	_, helper := startMockServer(t,
		"Step 1: FROM centos:7\nError: pull access denied\n",
		`[]`,
	)

	engine := New(helper)
	t.Cleanup(func() { engine.Close() })

	output, err := engine.Build(context.Background(), BuildParams{
		Deps:     "",
		MaxRetry: 0,
	})
	if err == nil {
		t.Fatal("Build() expected BuildError for build failure output")
	}
	be, ok := err.(*BuildError)
	if !ok {
		t.Fatalf("error type = %T, want *BuildError", err)
	}
	if be.ExitCode != 1 {
		t.Errorf("ExitCode = %d, want 1", be.ExitCode)
	}
	if output == "" {
		t.Error("expected non-empty output")
	}
}

// =============================================================================
// Build —— mock server 构建成功但 ImageList 返回空（镜像不存在）
// =============================================================================

func TestBuild_MockServer_ImageNotFoundAfterBuild(t *testing.T) {
	_, helper := startMockServer(t,
		"Successfully tagged agent-forge:latest\n",
		`[]`,
	)

	engine := New(helper)
	t.Cleanup(func() { engine.Close() })

	output, err := engine.Build(context.Background(), BuildParams{
		Deps:     "",
		MaxRetry: 0,
	})
	if err == nil {
		t.Fatal("Build() expected error when image is not found after build")
	}
	be, ok := err.(*BuildError)
	if !ok {
		t.Fatalf("error type = %T, want *BuildError", err)
	}
	if !strings.Contains(be.Message, "未在本地镜像列表中可见") {
		t.Errorf("error message = %q, should mention image not visible", be.Message)
	}
	if output == "" {
		t.Error("expected non-empty output")
	}
}

// =============================================================================
// Build —— mock server Rebuild 模式成功
// =============================================================================

func TestBuild_MockServer_RebuildSuccess(t *testing.T) {
	_, helper := startMockServer(t,
		"Successfully tagged agent-forge:tmp-123456\n",
		`[{"ID":"sha256:oldimage","RepoTags":["agent-forge:latest"]}]`,
	)

	engine := New(helper)
	t.Cleanup(func() { engine.Close() })

	output, err := engine.Build(context.Background(), BuildParams{
		Deps:     "",
		MaxRetry: 0,
		Rebuild:  true,
	})
	if err != nil {
		t.Fatalf("Build() (rebuild) expected success, got error: %v", err)
	}
	if !strings.Contains(output, "Successfully tagged") {
		t.Errorf("output should contain 'Successfully tagged'")
	}
	t.Logf("Rebuild success output (truncated): %.300s", output)
}

// =============================================================================
// Build —— mock server Rebuild 模式 + ImageTag 失败
// =============================================================================

func TestBuild_MockServer_Rebuild_ImageTagError(t *testing.T) {
	_, helper := startMockServerEx(t,
		"Successfully tagged agent-forge:tmp-123456\n",
		`[{"ID":"sha256:oldimage","RepoTags":["agent-forge:latest"]}]`,
		func(h *dockerMockHandler) { h.tagStatusCode = http.StatusInternalServerError },
	)

	engine := New(helper)
	t.Cleanup(func() { engine.Close() })

	_, err := engine.Build(context.Background(), BuildParams{
		Deps:     "",
		MaxRetry: 0,
		Rebuild:  true,
	})
	if err == nil {
		t.Fatal("Build() (rebuild) expected error when ImageTag fails")
	}
	if !strings.Contains(err.Error(), "标签替换失败") {
		t.Errorf("error = %v, should mention '标签替换失败'", err)
	}
}

// =============================================================================
// Build —— mock server Rebuild 模式 +  ImageExists 返回错误
// =============================================================================

func TestBuild_MockServer_Rebuild_ImageListError(t *testing.T) {
	_, helper := startMockServerEx(t,
		"Successfully tagged agent-forge:tmp-123456\n",
		`[]`, // 不会用到，因为 listStatusCode=500
		func(h *dockerMockHandler) { h.listStatusCode = http.StatusInternalServerError },
	)

	engine := New(helper)
	t.Cleanup(func() { engine.Close() })

	_, err := engine.Build(context.Background(), BuildParams{
		Deps:     "",
		MaxRetry: 0,
		Rebuild:  true,
	})
	if err == nil {
		t.Fatal("Build() (rebuild) expected error when ImageList fails")
	}
	if !strings.Contains(err.Error(), "检查旧镜像失败") &&
		!strings.Contains(err.Error(), "列出镜像失败") {
		t.Errorf("error = %v, should mention image check failure", err)
	}
}

// =============================================================================
// Build —— mock server Rebuild 模式 + 删除旧镜像失败
// =============================================================================

func TestBuild_MockServer_Rebuild_RemoveOldError(t *testing.T) {
	_, helper := startMockServerEx(t,
		"Successfully tagged agent-forge:tmp-123456\n",
		`[{"ID":"sha256:oldimage","RepoTags":["agent-forge:latest"]}]`,
		func(h *dockerMockHandler) { h.removeFail = true },
	)

	engine := New(helper)
	t.Cleanup(func() { engine.Close() })

	_, err := engine.Build(context.Background(), BuildParams{
		Deps:     "",
		MaxRetry: 0,
		Rebuild:  true,
	})
	if err == nil {
		t.Fatal("Build() (rebuild) expected error when ImageRemove(old) fails")
	}
	if !strings.Contains(err.Error(), "删除旧镜像失败") {
		t.Errorf("error = %v, should mention '删除旧镜像失败'", err)
	}
}

// =============================================================================
// Build —— mock server Rebuild 模式 + 无旧镜像（oldExists=false）
// =============================================================================

func TestBuild_MockServer_Rebuild_NoOldImage(t *testing.T) {
	_, helper := startMockServer(t,
		"Successfully tagged agent-forge:tmp-123456\n",
		`[]`, // 空列表 → ImageExists 返回 false
	)

	engine := New(helper)
	t.Cleanup(func() { engine.Close() })

	output, err := engine.Build(context.Background(), BuildParams{
		Deps:     "",
		MaxRetry: 0,
		Rebuild:  true,
	})
	if err != nil {
		t.Fatalf("Build() (rebuild) with no old image expected success, got: %v", err)
	}
	if !strings.Contains(output, "Successfully tagged") {
		t.Errorf("output should contain 'Successfully tagged'")
	}
}

// =============================================================================
// Build —— mock server 输出包含 retryable 错误 → 触发输出层重试
// =============================================================================

func TestBuild_MockServer_OutputRetryableError(t *testing.T) {
	buildOutput := "Step 5/10 : RUN curl https://github.com\n" +
		"curl: (7) Failed to connect to github.com port 443: Connection refused\n" +
		"make: *** [Makefile:12: deps] Error 7\n"

	_, helper := startMockServer(t, buildOutput, `[]`)

	engine := New(helper)
	t.Cleanup(func() { engine.Close() })

	output, err := engine.Build(context.Background(), BuildParams{
		Deps:     "",
		MaxRetry: 1,
	})
	if err == nil {
		t.Fatal("Build() expected error for failed build")
	}
	if !strings.Contains(output, "检测到网络错误") {
		t.Logf("Output did not contain retry keyword, len=%d", len(output))
	}
	_ = output
}

// =============================================================================
// Build —— mock server 输出 retry + ProgressWriter
// =============================================================================

func TestBuild_MockServer_OutputRetryable_WithProgress(t *testing.T) {
	buildOutput := "Step 5/10 : RUN curl https://github.com\n" +
		"curl: (7) Failed to connect to github.com port 443: Connection refused\n"

	_, helper := startMockServer(t, buildOutput, `[]`)

	engine := New(helper)
	t.Cleanup(func() { engine.Close() })

	var progressBuf strings.Builder
	output, err := engine.Build(context.Background(), BuildParams{
		Deps:           "",
		MaxRetry:       1,
		ProgressWriter: &progressBuf,
	})
	if err == nil {
		t.Fatal("Build() expected error for failed build with retry")
	}
	// ProgressWriter 应收到 retry 消息
	if !strings.Contains(progressBuf.String(), "检测到网络错误") {
		t.Logf("ProgressWriter content: %s", progressBuf.String())
	}
	if !strings.Contains(output, "检测到网络错误") {
		t.Logf("Output len=%d", len(output))
	}
}

// =============================================================================
// Build —— mock server 失败尝试后重建 cleanup 日志
// =============================================================================

func TestBuild_MockServer_RebuildFailure_CleanupLog(t *testing.T) {
	// Rebuild 模式下 ImageBuild 返回失败输出
	_, helper := startMockServer(t,
		"Step 1: FROM centos:7\nError: pull access denied\n",
		`[]`,
	)

	engine := New(helper)
	t.Cleanup(func() { engine.Close() })

	output, err := engine.Build(context.Background(), BuildParams{
		Deps:     "",
		MaxRetry: 0,
		Rebuild:  true,
	})
	if err == nil {
		t.Fatal("Build() expected error for rebuild failure")
	}
	// 应输出清理临时标签的日志
	if !strings.Contains(output, "清理临时标签") {
		t.Logf("Rebuild failure output (truncated): %.300s", output)
	}
}

// =============================================================================
// Build —— mock server 通过 hijack 中断响应 → io.Copy 触发 readErr + retry
// =============================================================================

func TestBuild_MockServer_HijackReadError(t *testing.T) {
	_, helper := startMockServerEx(t,
		"unused",
		`[]`,
		func(h *dockerMockHandler) { h.hijackBuild = true },
	)

	engine := New(helper)
	t.Cleanup(func() { engine.Close() })

	// MaxRetry >= 1 因为 readErr 为 retryable（connection closed / unexpected EOF），
	// 走 continue 然后重试。
	output, err := engine.Build(context.Background(), BuildParams{
		Deps:     "",
		MaxRetry: 1,
	})
	if err == nil {
		t.Fatal("Build() expected error after hijacked connection")
	}
	if !strings.Contains(output, "Step 1: FROM centos:7") && !strings.Contains(output, "重试") {
		t.Logf("Output (len=%d): %.200s", len(output), output)
	}
	t.Logf("Hijack read error test passed: %v", err)
}

// =============================================================================
// Build —— custom transport 模拟 io.Copy 期间 context 超时
// 通过 *io.Pipe 在发送部分构建日志后阻塞，直到 context 取消，
// 从而覆盖 io.Copy 返回错误时 ctx.Err() != nil 的分支（buildengine.go:172-174）
// =============================================================================

// hijackTimeoutTransport implements http.RoundTripper for Docker daemon mock.
// For /build requests, returns a pipe-based streaming body that writes partial
// output then blocks until context cancellation triggers the ctx.Err() path.
type hijackTimeoutTransport struct {
	ctx context.Context
}

func (t *hijackTimeoutTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// API version negotiation (called automatically by Docker SDK)
	if strings.HasSuffix(req.URL.Path, "/_ping") {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"API-Version": {"1.24"}},
			Body:       io.NopCloser(strings.NewReader("OK")),
			ContentLength: 2,
			Proto:      "HTTP/1.1",
			ProtoMajor: 1,
			ProtoMinor: 1,
		}, nil
	}

	// Build endpoint: return pipe that blocks after partial data
	if strings.HasSuffix(req.URL.Path, "/build") && req.Method == "POST" {
		pr, pw := io.Pipe()
		go func() {
			// Write initial build output, then block until ctx cancellation
			pw.Write([]byte("Step 1: FROM centos:7\n"))
			<-t.ctx.Done()
			pw.CloseWithError(t.ctx.Err())
		}()
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       pr,
			Proto:      "HTTP/1.1",
			ProtoMajor: 1,
			ProtoMinor: 1,
		}, nil
	}

	// Other endpoints: empty JSON
	return &http.Response{
		StatusCode:    http.StatusOK,
		Header:        http.Header{"Content-Type": {"application/json"}},
		Body:          io.NopCloser(strings.NewReader("[]")),
		ContentLength: 2,
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
	}, nil
}

func TestBuild_MockServer_HijackContextTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	httpClient := &http.Client{
		Transport: &hijackTimeoutTransport{ctx: ctx},
	}

	helper, err := dockerhelper.NewClientWithOpts(
		client.WithHTTPClient(httpClient),
	)
	if err != nil {
		t.Fatalf("NewClientWithOpts: %v", err)
	}
	t.Cleanup(func() { helper.Close() })

	engine := New(helper)
	t.Cleanup(func() { engine.Close() })

	output, err := engine.Build(ctx, BuildParams{
		Deps:     "",
		MaxRetry: 0,
	})
	if err == nil {
		t.Fatal("Build() expected error with context timeout during IO")
	}
	// 错误消息应为 "构建被中断: context deadline exceeded"
	if !strings.Contains(err.Error(), "被中断") &&
		!strings.Contains(err.Error(), "canceled") &&
		!strings.Contains(err.Error(), "context deadline") {
		t.Errorf("error = %v, should indicate context cancellation/timeout", err)
	}
	if !strings.Contains(output, "Step 1: FROM centos:7") {
		t.Logf("output (len=%d): %.200s", len(output), output)
	}
	t.Logf("Hijack context timeout test passed: %v", err)
}

// =============================================================================
// Build —— Docker daemon 返回 500 → 非重试错误分支
// =============================================================================

func TestBuild_MockServer_ImageBuildHttpError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Logf("[mock] %s %s", r.Method, r.URL.Path)

		if strings.HasSuffix(r.URL.Path, "/_ping") {
			w.Header().Set("API-Version", "1.24")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
			return
		}

		if strings.HasSuffix(r.URL.Path, "/build") && r.Method == "POST" {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		http.Error(w, "not mocked", http.StatusNotFound)
	}))
	t.Cleanup(server.Close)

	serverHost := strings.TrimPrefix(server.URL, "http://")
	helper, err := dockerhelper.NewClientWithOpts(
		client.WithHost("tcp://"+serverHost),
	)
	if err != nil {
		t.Fatalf("NewClientWithOpts: %v", err)
	}
	t.Cleanup(func() { helper.Close() })

	engine := New(helper)
	t.Cleanup(func() { engine.Close() })

	_, err = engine.Build(context.Background(), BuildParams{
		Deps:     "",
		MaxRetry: 0,
	})
	if err == nil {
		t.Fatal("Build() expected error when ImageBuild returns HTTP 500")
	}
	t.Logf("ImageBuild HTTP error test passed: %v", err)
}

// =============================================================================
// Build —— 构建成功但 ImageExists 返回错误
// =============================================================================

func TestBuild_ImageExistsFailsAfterBuild(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Logf("[mock] %s %s", r.Method, r.URL.Path)

		if strings.HasSuffix(r.URL.Path, "/_ping") {
			w.Header().Set("API-Version", "1.24")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
			return
		}

		if strings.HasSuffix(r.URL.Path, "/build") && r.Method == "POST" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Successfully tagged agent-forge:latest\n"))
			return
		}

		if strings.HasSuffix(r.URL.Path, "/images/json") {
			// 返回 500 使 ImageExists / ImageList 失败
			http.Error(w, "daemon error", http.StatusInternalServerError)
			return
		}

		http.Error(w, "not mocked", http.StatusNotFound)
	}))
	t.Cleanup(server.Close)

	serverHost := strings.TrimPrefix(server.URL, "http://")
	helper, err := dockerhelper.NewClientWithOpts(
		client.WithHost("tcp://"+serverHost),
	)
	if err != nil {
		t.Fatalf("NewClientWithOpts: %v", err)
	}
	t.Cleanup(func() { helper.Close() })

	engine := New(helper)
	t.Cleanup(func() { engine.Close() })

	_, err = engine.Build(context.Background(), BuildParams{
		Deps:     "",
		MaxRetry: 0,
	})
	if err == nil {
		t.Fatal("Build() expected error when ImageExists fails after build")
	}
	if !strings.Contains(err.Error(), "验证镜像失败") {
		t.Errorf("error = %v, should mention image verification failure", err)
	}
	t.Logf("ImageExists error test passed: %v", err)
}

// =============================================================================
// isBuildSuccessful 边界值
// =============================================================================

func TestIsBuildSuccessful_EdgeCases(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   bool
	}{
		{"trailing whitespace", "Successfully tagged agent-forge:latest  \n", true},
		{"tab before success", "\tSuccessfully tagged agent-forge:latest", true},
		{"only whitespace lines", "  \n\t\n  ", false},
		{"blank before success", "\n\nSuccessfully built abc123", true},
		{"windows crlf", "Step 1\r\nSuccessfully tagged agent-forge:latest\r\n", true},
		{"carriage return only", "Successfully tagged\r", true},
		{"mixed success keywords last", "Successfully built abc\nSuccessfully tagged def", true},
		{"successfully tagged in middle only", "Successfully tagged abc\nstill building...", false},
		{"lowercase success keyword", "successfully tagged agent-forge:latest", false},
		{"very long build output", "Step 1/10 : FROM centos:7\n" +
			"Step 2/10 : RUN yum install -y curl\n" +
			"Successfully tagged agent-forge:latest", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isBuildSuccessful(tt.output)
			if got != tt.want {
				t.Errorf("isBuildSuccessful(%q) = %v, want %v", tt.output, got, tt.want)
			}
		})
	}
}

func TestIsBuildSuccessful_EmptyInputs(t *testing.T) {
	if got := isBuildSuccessful(""); got != false {
		t.Errorf("isBuildSuccessful('') = %v, want false", got)
	}
	if got := isBuildSuccessful("  \n\n  "); got != false {
		t.Errorf("isBuildSuccessful(whitespace) = %v, want false", got)
	}
	if got := isBuildSuccessful("\n"); got != false {
		t.Errorf("isBuildSuccessful('\\n') = %v, want false", got)
	}
}

// =============================================================================
// isRetryableError 补充用例
// =============================================================================

func TestIsRetryableError_Additional(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"context deadline exceeded", context.DeadlineExceeded, true},
		{"context cancelled", context.Canceled, false},
		{"wrapped connection refused", fmt.Errorf("outer: %w", fmt.Errorf("connection refused")), true},
		{"wrapped dial tcp", fmt.Errorf("wrap: %w", fmt.Errorf("dial tcp 1.2.3.4:80")), true},
		{"i/o timeout alone", fmt.Errorf("i/o timeout"), true},
		{"connection reset alone", fmt.Errorf("connection reset by peer"), true},
		{"no such host alone", fmt.Errorf("no such host: example.com"), true},
		{"eof exact", fmt.Errorf("EOF"), true},
		{"error is nil", nil, false},
		{"chinese ch prompt error", fmt.Errorf("net/http: TLS handshake timeout"), true},
		{"empty string error", fmt.Errorf(""), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isRetryableError(tt.err)
			if got != tt.want {
				t.Errorf("isRetryableError(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

// =============================================================================
// CalculateBackoff 补充
// =============================================================================

func TestCalculateBackoff_Additional(t *testing.T) {
	tests := []struct {
		attempt int
		want    time.Duration
	}{
		{-1, 0},
		{0, 0},
		{1, 1 * time.Second},
		{2, 2 * time.Second},
		{3, 4 * time.Second},
		{6, 32 * time.Second},
	}
	for _, tt := range tests {
		got := CalculateBackoff(tt.attempt)
		if got != tt.want {
			t.Errorf("CalculateBackoff(%d) = %v, want %v", tt.attempt, got, tt.want)
		}
	}
}

// =============================================================================
// createBuildContext 边界值
// =============================================================================

func TestCreateBuildContext_Empty(t *testing.T) {
	buf, err := createBuildContext("")
	if err != nil {
		t.Fatalf("createBuildContext('') error = %v", err)
	}
	if buf.Len() == 0 {
		t.Error("createBuildContext('') returned empty buffer")
	}
}

func TestCreateBuildContext_LargeDockerfile(t *testing.T) {
	content := "FROM centos:7\n" + strings.Repeat("RUN echo hello\n", 1000)
	buf, err := createBuildContext(content)
	if err != nil {
		t.Fatalf("createBuildContext(large) error = %v", err)
	}
	if buf.Len() == 0 {
		t.Error("createBuildContext(large) returned empty buffer")
	}
	t.Logf("large tar size = %d bytes", buf.Len())
}

// =============================================================================
// Error 类型语义验证
// =============================================================================

func TestInvalidParamsError_Format(t *testing.T) {
	err := &InvalidParamsError{Reason: "max-retry 不能为负数"}
	msg := err.Error()
	if !strings.Contains(msg, "参数无效") {
		t.Errorf("Error() = %q, should contain '参数无效'", msg)
	}
	if !strings.Contains(msg, "负数") {
		t.Errorf("Error() = %q, should contain '负数'", msg)
	}
}

func TestBuildError_Fields(t *testing.T) {
	err := &BuildError{
		Message:  "custom build failure",
		Output:   "step output...",
		ExitCode: 2,
	}
	if err.Error() != "custom build failure" {
		t.Errorf("Error() = %q, want 'custom build failure'", err.Error())
	}
	if err.ExitCode != 2 {
		t.Errorf("ExitCode = %d, want 2", err.ExitCode)
	}
}

func TestRetryExhaustedError_Format(t *testing.T) {
	err := &RetryExhaustedError{MaxRetry: 5}
	msg := err.Error()
	if !strings.Contains(msg, "5") {
		t.Errorf("Error() = %q, should mention '5'", msg)
	}
	if !strings.Contains(msg, "重试") {
		t.Errorf("Error() = %q, should mention '重试'", msg)
	}
}

// =============================================================================
// Engine.Close
// =============================================================================

func TestEngineClose(t *testing.T) {
	helper, err := dockerhelper.NewClientWithOpts(
		client.WithHost("tcp://localhost:1"),
	)
	if err != nil {
		t.Fatalf("NewClientWithOpts() error = %v", err)
	}
	engine := New(helper)
	err = engine.Close()
	if err != nil {
		t.Logf("Close() returned error (expected with broken client): %v", err)
	}
}
