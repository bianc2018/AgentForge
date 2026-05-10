// Package buildengine 编排 Docker 镜像构建流程。
//
// BuildEngine 位于构建层（Build Layer），整合 Deps Module、Dockerfile Generator
// 和 Docker Helper 完成完整的镜像构建生命周期。负责参数展开、Dockerfile 生成、
// 构建执行、结果验证以及网络错误重试。
package buildengine

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/docker/docker/api/types"

	"github.com/agent-forge/cli/internal/build/depsmodule"
	"github.com/agent-forge/cli/internal/build/dockerfilegen"
	"github.com/agent-forge/cli/internal/shared/dockerhelper"
)

// ImageTag 是构建完成后赋予镜像的标准标签。
const ImageTag = "agent-forge:latest"

// BuildParams 是 BuildEngine 构建操作所需的完整参数集。
type BuildParams struct {
	Deps     string // -d 参数（逗号分隔的依赖列表或元标签）
	BaseImage string // -b 参数（基础镜像）
	Config    string // -c 参数（配置目录）
	NoCache   bool   // --no-cache 参数
	Rebuild   bool   // -R/--rebuild 参数
	MaxRetry  int    // --max-retry 参数（默认 3）
	GHProxy   string // --gh-proxy 参数（GitHub 代理 URL）
}

// Engine 是构建引擎，负责编排完整的镜像构建流程。
type Engine struct {
	helper *dockerhelper.Client
}

// New 创建新的构建引擎。
//
// 需要已经初始化的 Docker Helper 客户端。
func New(helper *dockerhelper.Client) *Engine {
	return &Engine{helper: helper}
}

// Build 执行完整的镜像构建流程，支持指数退避重试。
//
// 流程：
//  1. 验证参数
//  2. 展开依赖列表（ExpandDeps）
//  3. 生成 Dockerfile 内容（Generate）
//  4. 创建 tar 构建上下文
//  5. 循环执行：调用 ImageBuild API -> 读取输出 -> 判断成功
//  6. 网络错误时按指数退避重试（最多 MaxRetry 次）
//  7. 构建成功后通过 ImageList 确认镜像可见
//
// 返回构建输出日志和可能的错误。
//   - nil：构建成功
//   - *InvalidParamsError：参数错误（退出码 2）
//   - *BuildError：构建失败（退出码 1）
//   - *RetryExhaustedError：重试耗尽（退出码 1）
func (e *Engine) Build(ctx context.Context, params BuildParams) (string, error) {
	// 1. 参数验证
	if err := validateParams(params); err != nil {
		return "", err
	}

	// 2. 展开依赖
	deps := depsmodule.ExpandDeps(params.Deps)

	// 3. 生成 Dockerfile
	dockerfile, err := dockerfilegen.Generate(dockerfilegen.Options{
		BaseImage: params.BaseImage,
		Deps:      deps,
		GHProxy:   params.GHProxy,
		NoCache:   params.NoCache,
	})
	if err != nil {
		return "", fmt.Errorf("生成 Dockerfile 失败: %w", err)
	}

	// 4. 创建可重读的 tar 构建上下文
	buildContext, err := createBuildContext(dockerfile)
	if err != nil {
		return "", fmt.Errorf("创建构建上下文失败: %w", err)
	}
	// 保存到 bytes.Reader 以便重试时重新读取
	buildContextBytes := buildContext.Bytes()

	// 构建选项（不随重试变化）
	buildOpts := types.ImageBuildOptions{
		Tags:        []string{ImageTag},
		NoCache:     params.NoCache,
		Remove:      true,
		ForceRemove: true,
		PullParent:  true,
		Dockerfile:  "Dockerfile",
	}

	// 5. 执行构建（带重试）
	var outputBuf bytes.Buffer
	var lastErr error

	for attempt := 0; attempt <= params.MaxRetry; attempt++ {
		// 检查 context 是否已取消
		select {
		case <-ctx.Done():
			return outputBuf.String(), fmt.Errorf("构建被中断: %w", ctx.Err())
		default:
		}

		// 重试等待
		if attempt > 0 {
			backoff := CalculateBackoff(attempt)
			waitMsg := fmt.Sprintf("\n[重试 %d/%d] 等待 %v 后重新构建...\n", attempt, params.MaxRetry, backoff)
			outputBuf.WriteString(waitMsg)

			// 等待期间也检查 context
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return outputBuf.String(), fmt.Errorf("重试等待中被中断: %w", ctx.Err())
			}
		}

		// 重新创建 tar reader（从保存的字节切片）
		tarReader := bytes.NewReader(buildContextBytes)

		// 调用 ImageBuild API
		resp, err := e.helper.ImageBuild(ctx, tarReader, buildOpts)
		if err != nil {
			lastErr = err
			if isRetryableError(err) {
				continue
			}
			// 非重试性错误
			return outputBuf.String(), fmt.Errorf("Docker 构建失败: %w", err)
		}

		// 读取构建输出
		output, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			lastErr = readErr
			if isRetryableError(readErr) {
				continue
			}
			return outputBuf.String(), fmt.Errorf("读取构建输出失败: %w", readErr)
		}

		outputStr := string(output)
		outputBuf.WriteString(outputStr)

		// 检查构建是否成功
		if isBuildSuccessful(outputStr) {
			// 构建成功，确认镜像可见
			exists, checkErr := e.helper.ImageExists(ctx, ImageTag)
			if checkErr != nil {
				return outputBuf.String(), fmt.Errorf("构建后验证镜像失败: %w", checkErr)
			}
			if !exists {
				return outputBuf.String(), &BuildError{
					Message:  fmt.Sprintf("构建完成后镜像 %s 未在本地镜像列表中可见", ImageTag),
					Output:   outputBuf.String(),
					ExitCode: 1,
				}
			}
			// 构建成功
			return outputBuf.String(), nil
		}

		// 构建完成但失败了（Dockerfile 或依赖问题）——不可重试
		lastErr = &BuildError{
			Message:  "镜像构建失败",
			Output:   outputBuf.String(),
			ExitCode: 1,
		}
		break
	}

	// 所有重试耗尽
	if lastErr == nil {
		lastErr = &RetryExhaustedError{
			MaxRetry: params.MaxRetry,
		}
	}
	return outputBuf.String(), lastErr
}

// CalculateBackoff 计算第 N 次重试的等待时间。
//
// 第 1 次重试等待 1 秒，此后每次翻倍：
//
//	attempt=1 -> 1s
//	attempt=2 -> 2s
//	attempt=3 -> 4s
//	attempt=N -> 2^(N-1)s
func CalculateBackoff(attempt int) time.Duration {
	if attempt <= 0 {
		return 0
	}
	return time.Duration(1<<uint(attempt-1)) * time.Second
}

// isRetryableError 判断错误是否可重试。
//
// 网络相关的瞬时错误可重试，非网络错误（如参数错误、镜像不存在等）不重试。
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "no such host") ||
		strings.Contains(msg, "i/o timeout") ||
		strings.Contains(msg, "connection reset") ||
		strings.Contains(msg, "dial tcp") ||
		strings.Contains(msg, "tls handshake timeout") ||
		strings.Contains(msg, "context deadline exceeded") ||
		strings.Contains(msg, "cannot connect") ||
		strings.Contains(msg, "eof")
}

// validateParams 检查构建参数是否有效。
func validateParams(params BuildParams) error {
	if params.MaxRetry < 0 {
		return &InvalidParamsError{Reason: "max-retry 不能为负数"}
	}
	return nil
}

// createBuildContext 从 Dockerfile 内容创建 tar 格式的构建上下文。
//
// 返回的 bytes.Buffer 可多次读取（通过 Bytes() 获取底层数据）。
func createBuildContext(dockerfile string) (*bytes.Buffer, error) {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	hdr := &tar.Header{
		Name: "Dockerfile",
		Size: int64(len(dockerfile)),
		Mode: 0644,
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return nil, fmt.Errorf("写入 tar 头失败: %w", err)
	}
	if _, err := tw.Write([]byte(dockerfile)); err != nil {
		return nil, fmt.Errorf("写入 tar 内容失败: %w", err)
	}
	if err := tw.Close(); err != nil {
		return nil, fmt.Errorf("关闭 tar writer 失败: %w", err)
	}

	return &buf, nil
}

// isBuildSuccessful 从构建输出中判断构建是否成功。
//
// Docker 构建成功的输出最后一行通常包含 "Successfully tagged" 或 "Successfully built"。
func isBuildSuccessful(output string) bool {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) == 0 {
		return false
	}
	lastLine := lines[len(lines)-1]
	return strings.Contains(lastLine, "Successfully tagged") ||
		strings.Contains(lastLine, "Successfully built")
}

// InvalidParamsError 表示构建参数无效，对应退出码 2。
type InvalidParamsError struct {
	Reason string
}

func (e *InvalidParamsError) Error() string {
	return fmt.Sprintf("构建参数无效: %s", e.Reason)
}

// BuildError 表示构建过程失败，对应退出码 1。
type BuildError struct {
	Message  string
	Output   string
	ExitCode int
}

func (e *BuildError) Error() string {
	return e.Message
}

// RetryExhaustedError 表示所有重试均已耗尽，对应退出码 1。
// 符合 NFR-16 格式：包含原因、上下文和建议。
type RetryExhaustedError struct {
	MaxRetry int
}

func (e *RetryExhaustedError) Error() string {
	return fmt.Sprintf(
		"镜像构建失败：网络错误，已重试 %d 次\n"+
			"上下文：Docker 构建过程中的网络请求持续失败\n"+
			"建议：请检查网络连接是否正常，或使用 --gh-proxy 参数指定 GitHub 代理",
		e.MaxRetry,
	)
}

// Close 释放构建引擎使用的资源。
func (e *Engine) Close() error {
	return e.helper.Close()
}
