// Package buildengine 编排 Docker 镜像构建流程。
//
// BuildEngine 位于构建层（Build Layer），整合 Deps Module、Dockerfile Generator
// 和 Docker Helper 完成完整的镜像构建生命周期。负责参数展开、Dockerfile 生成、
// 构建执行和结果验证。
package buildengine

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/docker/docker/api/types"

	"github.com/agent-forge/cli/internal/build/depsmodule"
	"github.com/agent-forge/cli/internal/build/dockerfilegen"
	"github.com/agent-forge/cli/internal/shared/dockerhelper"
)

// ImageTag 是构建完成后赋予镜像的标准标签。
const ImageTag = "agent-forge:latest"

// BuildParams 是 BuildEngine 构建操作所需的完整参数集。
type BuildParams struct {
	// Deps 是 -d 参数的原始值（逗号分隔的依赖列表或元标签）。
	Deps string
	// BaseImage 是 -b 参数指定的基础镜像。
	BaseImage string
	// Config 是 -c 参数指定的配置目录。
	Config string
	// NoCache 是 --no-cache 参数。
	NoCache bool
	// Rebuild 是 -R/--rebuild 参数。
	Rebuild bool
	// MaxRetry 是 --max-retry 参数（网络错误时的最大重试次数）。
	MaxRetry int
	// GHProxy 是 --gh-proxy 参数指定的 GitHub 代理 URL。
	GHProxy string
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

// Build 执行完整的镜像构建流程。
//
// 流程：
//  1. 验证参数
//  2. 展开依赖列表（ExpandDeps）
//  3. 生成 Dockerfile 内容（Generate）
//  4. 创建 tar 构建上下文
//  5. 调用 ImageBuild API 执行构建
//  6. 读取构建输出
//  7. 通过 ImageList 确认镜像可见
//
// 返回构建输出日志和可能的错误。
// 错误类型 *InvalidParamsError 表示参数错误（退出码 2），其他错误为构建失败（退出码 1）。
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

	// 4. 创建 tar 构建上下文
	buildContext, err := createBuildContext(dockerfile)
	if err != nil {
		return "", fmt.Errorf("创建构建上下文失败: %w", err)
	}

	// 5. 执行构建
	buildOpts := types.ImageBuildOptions{
		Tags:       []string{ImageTag},
		NoCache:    params.NoCache,
		Remove:     true,
		ForceRemove: true,
		PullParent: true,
		Dockerfile: "Dockerfile",
	}

	resp, err := e.helper.ImageBuild(ctx, buildContext, buildOpts)
	if err != nil {
		return "", fmt.Errorf("Docker 构建失败: %w", err)
	}
	defer resp.Body.Close()

	// 6. 读取构建输出
	output, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取构建输出失败: %w", err)
	}
	outputStr := string(output)

	// 7. 检查构建是否成功
	if !isBuildSuccessful(outputStr) {
		return outputStr, &BuildError{
			Message:  "镜像构建失败",
			Output:   outputStr,
			ExitCode: 1,
		}
	}

	// 8. 确认镜像可见
	exists, err := e.helper.ImageExists(ctx, ImageTag)
	if err != nil {
		return outputStr, fmt.Errorf("构建后验证镜像失败: %w", err)
	}
	if !exists {
		return outputStr, &BuildError{
			Message:  fmt.Sprintf("构建完成后镜像 %s 未在本地镜像列表中可见", ImageTag),
			Output:   outputStr,
			ExitCode: 1,
		}
	}

	return outputStr, nil
}

// validateParams 检查构建参数是否有效。
func validateParams(params BuildParams) error {
	if params.MaxRetry < 0 {
		return &InvalidParamsError{Reason: "max-retry 不能为负数"}
	}
	return nil
}

// createBuildContext 从 Dockerfile 内容创建 tar 格式的构建上下文。
func createBuildContext(dockerfile string) (io.Reader, error) {
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

// InvalidParamsError 表示构建参数无效。
type InvalidParamsError struct {
	Reason string
}

func (e *InvalidParamsError) Error() string {
	return fmt.Sprintf("构建参数无效: %s", e.Reason)
}

// BuildError 表示构建过程失败的详细错误。
type BuildError struct {
	Message  string
	Output   string
	ExitCode int
}

func (e *BuildError) Error() string {
	return e.Message
}

// Close 释放构建引擎使用的资源。
func (e *Engine) Close() error {
	return e.helper.Close()
}
