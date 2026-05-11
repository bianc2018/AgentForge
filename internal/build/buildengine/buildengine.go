// Package buildengine 编排 Docker 镜像构建流程。
//
// BuildEngine 位于构建层（Build Layer），整合 Deps Module、Dockerfile Generator
// 和 Docker Helper 完成完整的镜像构建生命周期。负责参数展开、Dockerfile 生成、
// 构建执行、结果验证以及网络错误重试和重建模式。
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
	Deps      string // -d 参数（逗号分隔的依赖列表或元标签）
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

// Build 执行完整的镜像构建流程，支持指数退避重试和重建模式。
//
// 流程：
//  1. 验证参数
//  2. 展开依赖列表（ExpandDeps）
//  3. 生成 Dockerfile 内容（Generate）
//  4. 创建 tar 构建上下文
//  5. 确定构建标签（普通模式用 ImageTag，rebuild 模式用临时标签）
//  6. 循环执行：调用 ImageBuild API -> 读取输出 -> 判断成功
//  7. 网络错误时按指数退避重试（最多 MaxRetry 次）
//  8. 构建成功后：
//     - 普通模式：通过 ImageList 确认镜像可见
//     - Rebuild 模式：执行原子替换（临时标签 -> ImageTag，删除旧镜像）
//  9. 构建失败时（rebuild 模式）：清理临时标签，保留原镜像
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
	buildContextBytes := buildContext.Bytes()

	// 确定构建标签
	buildTag := ImageTag
	tmpTag := ""
	if params.Rebuild {
		params.NoCache = true // 重建模式自动叠加 --no-cache
		tmpTag = fmt.Sprintf("agent-forge:tmp-%d", time.Now().UnixNano())
		buildTag = tmpTag
	}

	// 构建选项
	// PullParent 被禁用：上游 CentOS 7 官方镜像已被 registry 删除，
	// PullParent: true 会导致已缓存的基础镜像也无法使用。
	// 基础镜像需通过前置步骤确保已缓存。
	buildOpts := types.ImageBuildOptions{
		Tags:        []string{buildTag},
		NoCache:     params.NoCache,
		Remove:      true,
		ForceRemove: true,
		Dockerfile:  "Dockerfile",
	}

	// 5. 执行构建（带重试）
	var outputBuf bytes.Buffer
	var lastErr error
	buildSucceeded := false

	for attempt := 0; attempt <= params.MaxRetry; attempt++ {
		select {
		case <-ctx.Done():
			return outputBuf.String(), fmt.Errorf("构建被中断: %w", ctx.Err())
		default:
		}

		if attempt > 0 {
			backoff := CalculateBackoff(attempt)
			waitMsg := fmt.Sprintf("\n[重试 %d/%d] 等待 %v 后重新构建...\n", attempt, params.MaxRetry, backoff)
			outputBuf.WriteString(waitMsg)
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return outputBuf.String(), fmt.Errorf("重试等待中被中断: %w", ctx.Err())
			}
		}

		tarReader := bytes.NewReader(buildContextBytes)

		resp, err := e.helper.ImageBuild(ctx, tarReader, buildOpts)
		if err != nil {
			lastErr = err
			if isRetryableError(err) {
				continue
			}
			return outputBuf.String(), fmt.Errorf("Docker 构建失败: %w", err)
		}

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

		if isBuildSuccessful(outputStr) {
			buildSucceeded = true
			break
		}

		// 检查构建输出中是否包含可重试的网络错误（curl SSL、连接重置等）
		if attempt < params.MaxRetry && isRetryableError(fmt.Errorf(outputStr)) {
			outputBuf.WriteString(fmt.Sprintf("\n[检测到网络错误，将进行重试 %d/%d]\n", attempt+1, params.MaxRetry))
			continue
		}

		lastErr = &BuildError{
			Message:  "镜像构建失败",
			Output:   outputBuf.String(),
			ExitCode: 1,
		}
		break
	}

	// 6. 处理构建结果
	if buildSucceeded {
		if params.Rebuild {
			if err := e.handleRebuildSuccess(ctx, tmpTag); err != nil {
				return outputBuf.String(), err
			}
		} else {
			exists, checkErr := e.helper.ImageExists(ctx, buildTag)
			if checkErr != nil {
				return outputBuf.String(), fmt.Errorf("构建后验证镜像失败: %w", checkErr)
			}
			if !exists {
				return outputBuf.String(), &BuildError{
					Message:  fmt.Sprintf("构建完成后镜像 %s 未在本地镜像列表中可见", buildTag),
					Output:   outputBuf.String(),
					ExitCode: 1,
				}
			}
		}
		return outputBuf.String(), nil
	}

	// 构建失败：rebuild 模式下清理临时标签
	if params.Rebuild && tmpTag != "" {
		if _, cleanupErr := e.helper.ImageRemove(ctx, tmpTag, true, true); cleanupErr != nil {
			outputBuf.WriteString(fmt.Sprintf("[清理] 清理临时标签 %s 失败: %v\n", tmpTag, cleanupErr))
		} else {
			outputBuf.WriteString(fmt.Sprintf("[清理] 已清理临时标签 %s，原镜像保持不变\n", tmpTag))
		}
	}

	// 所有重试耗尽或构建失败
	if lastErr == nil {
		lastErr = &RetryExhaustedError{
			MaxRetry: params.MaxRetry,
		}
	}
	return outputBuf.String(), lastErr
}

// handleRebuildSuccess 处理重建成功后的原子标签替换。
//
// 流程：
//  1. 记录旧镜像 ID（用于后续删除）
//  2. 将临时标签指向新镜像的 ImageTag
//  3. 删除旧镜像
//  4. 删除临时标签
func (e *Engine) handleRebuildSuccess(ctx context.Context, tmpTag string) error {
	// 1. 查找旧镜像 ID（如果存在）
	oldImageID := ""
	oldExists, err := e.helper.ImageExists(ctx, ImageTag)
	if err != nil {
		return fmt.Errorf("重建：检查旧镜像失败: %w", err)
	}
	if oldExists {
		// 通过 ImageList 获取旧镜像 ID
		images, listErr := e.helper.ImageList(ctx, types.ImageListOptions{})
		if listErr != nil {
			return fmt.Errorf("重建：列出镜像失败: %w", listErr)
		}
		for _, img := range images {
			for _, tag := range img.RepoTags {
				if tag == ImageTag {
					oldImageID = img.ID
					break
				}
			}
			if oldImageID != "" {
				break
			}
		}
	}

	// 2. 将临时标签指向正式标签
	if err := e.helper.ImageTag(ctx, tmpTag, ImageTag); err != nil {
		return fmt.Errorf("重建：标签替换失败（临时标签 %s -> %s）: %w", tmpTag, ImageTag, err)
	}

	// 3. 删除旧镜像（如果有）
	if oldImageID != "" {
		if _, err := e.helper.ImageRemove(ctx, oldImageID, true, true); err != nil {
			// 旧镜像删除失败不阻止流程，记录即可
			return fmt.Errorf("重建：删除旧镜像失败: %w", err)
		}
	}

	// 4. 删除临时标签
	if _, err := e.helper.ImageRemove(ctx, tmpTag, true, true); err != nil {
		// 临时标签可能已被 ImageTag 引用，忽略残留标签删除错误
	}

	return nil
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
