// Package dockerhelper 封装 Docker Engine API 调用，提供类型安全的
// 容器和镜像操作，统一处理 Docker API 错误类型。
//
// dockerhelper 位于共享层（Shared Layer），依赖外部 Docker Engine >= 20.10。
// 所有对外部 Docker daemon 的通信均通过此包完成。
package dockerhelper

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/docker/errdefs"
	specs "github.com/opencontainers/image-spec/specs-go/v1"
)

// Client 封装 Docker SDK client，提供便捷的 Docker daemon 操作方法。
type Client struct {
	api *client.Client
}

// NewClient 创建 Docker Helper 客户端。
//
// 使用 Docker SDK 的默认环境配置创建客户端（参考 DOCKER_HOST、DOCKER_API_VERSION 等环境变量），
// 自动协商 API 版本以确保兼容性。
func NewClient() (*Client, error) {
	cli, err := client.NewClientWithOpts(
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return nil, fmt.Errorf("创建 Docker 客户端失败: %w", err)
	}
	return &Client{api: cli}, nil
}

// NewClientWithOpts 使用自定义选项创建 Docker Helper 客户端。
// 适用于需要自定义 Docker daemon 地址的测试场景。
func NewClientWithOpts(opts ...client.Opt) (*Client, error) {
	cli, err := client.NewClientWithOpts(opts...)
	if err != nil {
		return nil, fmt.Errorf("创建 Docker 客户端失败: %w", err)
	}
	// 始终启用 API 版本协商
	client.WithAPIVersionNegotiation()(cli)
	return &Client{api: cli}, nil
}

// Close 关闭 Docker 客户端连接，释放底层资源。
func (c *Client) Close() error {
	return c.api.Close()
}

// Ping 检测 Docker daemon 是否可达。
//
// 返回 nil 表示 daemon 可正常通信；非 nil 表示 daemon 不可达或连接异常。
func (c *Client) Ping(ctx context.Context) error {
	_, err := c.api.Ping(ctx)
	if err != nil {
		return fmt.Errorf("Docker daemon 不可达: %w", err)
	}
	return nil
}

// Info 获取 Docker daemon 系统信息，包括版本、OS、容器数量等。
func (c *Client) Info(ctx context.Context) (types.Info, error) {
	info, err := c.api.Info(ctx)
	if err != nil {
		return types.Info{}, fmt.Errorf("获取 Docker 信息失败: %w", err)
	}
	return info, nil
}

// ImageList 返回本地 Docker 镜像列表。
//
// opts 可用于过滤镜像列表（如按引用名称过滤）。
func (c *Client) ImageList(ctx context.Context, opts types.ImageListOptions) ([]types.ImageSummary, error) {
	images, err := c.api.ImageList(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("列出 Docker 镜像失败: %w", err)
	}
	return images, nil
}

// IsBuildKitEnabled 检查是否启用了 Docker BuildKit。
//
// 通过检查 DOCKER_BUILDKIT 环境变量是否为 "1" 来判断。
// 注意：BuildKit 启用状态也可以在 Docker daemon 配置中设定，
// 此方法仅检测环境变量级别的开关。
func (c *Client) IsBuildKitEnabled() bool {
	return os.Getenv("DOCKER_BUILDKIT") == "1"
}

// PingWithInfo 组合操作：先 Ping 确认连通性，再获取 Info。
// 如果 Ping 失败，直接返回错误，不执行 Info 调用。
func (c *Client) PingWithInfo(ctx context.Context) (*types.Info, error) {
	if err := c.Ping(ctx); err != nil {
		return nil, err
	}
	info, err := c.Info(ctx)
	if err != nil {
		return nil, err
	}
	return &info, nil
}

// ImageExists 检查指定名称或 ID 的镜像是否存在于本地。
func (c *Client) ImageExists(ctx context.Context, ref string) (bool, error) {
	images, err := c.ImageList(ctx, types.ImageListOptions{})
	if err != nil {
		return false, err
	}
	for _, img := range images {
		for _, tag := range img.RepoTags {
			if tag == ref {
				return true, nil
			}
		}
		// Also check by ID prefix
		if len(img.ID) >= len(ref) && img.ID[:len(ref)] == ref {
			return true, nil
		}
	}
	return false, nil
}

// ImageBuild 执行 Docker 镜像构建。
//
// buildContext 是构建上下文的 tar 流（包含 Dockerfile 和需要 COPY 的文件）。
// options 包含构建参数（标签、缓存策略、构建参数等）。
func (c *Client) ImageBuild(ctx context.Context, buildContext io.Reader, options types.ImageBuildOptions) (types.ImageBuildResponse, error) {
	resp, err := c.api.ImageBuild(ctx, buildContext, options)
	if err != nil {
		err = ClassifyError(fmt.Errorf("Docker 镜像构建失败: %w", err))
		return types.ImageBuildResponse{}, err
	}
	return resp, nil
}

// ImageTag 为指定镜像添加标签。
//
// source 是源镜像的 ID 或名称，target 是目标标签（如 "agent-forge:latest"）。
func (c *Client) ImageTag(ctx context.Context, source, target string) error {
	err := c.api.ImageTag(ctx, source, target)
	if err != nil {
		return ClassifyError(fmt.Errorf("Docker 镜像打标签失败: %w", err))
	}
	return nil
}

// ImageRemove 删除指定镜像。
//
// imageID 是镜像的 ID 或名称。
// force 为 true 时强制删除（即使容器正在使用该镜像）。
// prune 为 true 时删除未打标签的父镜像。
func (c *Client) ImageRemove(ctx context.Context, imageID string, force, prune bool) ([]types.ImageDeleteResponseItem, error) {
	opts := types.ImageRemoveOptions{
		Force:         force,
		PruneChildren: prune,
	}
	resp, err := c.api.ImageRemove(ctx, imageID, opts)
	if err != nil {
		return nil, ClassifyError(fmt.Errorf("Docker 镜像删除失败: %w", err))
	}
	return resp, nil
}

// ImageSave 将指定镜像保存为 tar 流。
//
// imageIDs 是要导出的镜像 ID 或名称列表。
// 返回的 io.ReadCloser 包含 tar 格式的镜像数据。
func (c *Client) ImageSave(ctx context.Context, imageIDs []string) (io.ReadCloser, error) {
	reader, err := c.api.ImageSave(ctx, imageIDs)
	if err != nil {
		return nil, ClassifyError(fmt.Errorf("Docker 镜像导出失败: %w", err))
	}
	return reader, nil
}

// ImageLoad 从 tar 流加载镜像。
//
// input 是包含 tar 格式镜像数据的读取器。
// quiet 为 true 时减少输出。
// 返回 ImageLoadResponse 包含加载结果。
func (c *Client) ImageLoad(ctx context.Context, input io.Reader, quiet bool) (types.ImageLoadResponse, error) {
	resp, err := c.api.ImageLoad(ctx, input, quiet)
	if err != nil {
		return types.ImageLoadResponse{}, ClassifyError(fmt.Errorf("Docker 镜像导入失败: %w", err))
	}
	return resp, nil
}

// ContainerCreate 创建 Docker 容器。
//
// config 是容器配置（Image、Cmd、Env、WorkingDir 等），
// hostConfig 是主机配置（端口映射、挂载、特权模式等），
// networkingConfig 是网络配置，
// platform 是平台配置（可选，传 nil 使用默认值），
// containerName 是容器名称（空字符串由 Docker 自动生成）。
func (c *Client) ContainerCreate(
	ctx context.Context,
	config *container.Config,
	hostConfig *container.HostConfig,
	networkingConfig *network.NetworkingConfig,
	platform *specs.Platform,
	containerName string,
) (container.ContainerCreateCreatedBody, error) {
	resp, err := c.api.ContainerCreate(ctx, config, hostConfig, networkingConfig, platform, containerName)
	if err != nil {
		return resp, ClassifyError(fmt.Errorf("创建容器失败: %w", err))
	}
	return resp, nil
}

// ContainerStart 启动指定容器。
func (c *Client) ContainerStart(ctx context.Context, containerID string, opts types.ContainerStartOptions) error {
	err := c.api.ContainerStart(ctx, containerID, opts)
	if err != nil {
		return ClassifyError(fmt.Errorf("启动容器失败: %w", err))
	}
	return nil
}

// ContainerAttach 附加到指定容器的标准输入/输出/错误流。
//
// 返回 HijackedResponse，包含底层的连接和读取器。
// 调用者必须在完成后关闭响应。
func (c *Client) ContainerAttach(ctx context.Context, containerID string, opts types.ContainerAttachOptions) (types.HijackedResponse, error) {
	resp, err := c.api.ContainerAttach(ctx, containerID, opts)
	if err != nil {
		return resp, ClassifyError(fmt.Errorf("附加到容器失败: %w", err))
	}
	return resp, nil
}

// ContainerWait 等待容器达到指定状态，返回退出状态码。
//
// condition 指定等待的条件（如 "next-exit" 等待容器退出）。
func (c *Client) ContainerWait(ctx context.Context, containerID string, condition container.WaitCondition) (<-chan container.ContainerWaitOKBody, <-chan error) {
	return c.api.ContainerWait(ctx, containerID, condition)
}

// ContainerRemove 删除指定容器。
//
// force 为 true 时强制删除正在运行的容器。
// removeVolumes 为 true 时删除容器的匿名卷。
func (c *Client) ContainerRemove(ctx context.Context, containerID string, force, removeVolumes bool) error {
	opts := types.ContainerRemoveOptions{
		Force:         force,
		RemoveVolumes: removeVolumes,
	}
	err := c.api.ContainerRemove(ctx, containerID, opts)
	if err != nil {
		return ClassifyError(fmt.Errorf("删除容器失败: %w", err))
	}
	return nil
}

// Standard errors for Docker connection issues.
var (
	// ErrDockerNotReachable 表示 Docker daemon 无法连接。
	ErrDockerNotReachable = errors.New("Docker daemon 不可达，请确认 Docker 已安装并运行")
	// ErrDockerPermissionDenied 表示当前用户无权限访问 Docker socket。
	ErrDockerPermissionDenied = errors.New("Docker socket 访问权限不足，请确认当前用户属于 docker 用户组")
)

// ClassifyError 将 Docker API 错误分类为业务语义明确的错误。
//
// 可用于诊断场景，根据不同的错误类型给出对应的处理建议。
func ClassifyError(err error) error {
	if err == nil {
		return nil
	}

	switch {
	case errdefs.IsNotFound(err):
		return fmt.Errorf("Docker 资源不存在: %w", err)
	case errdefs.IsConflict(err):
		return fmt.Errorf("Docker 资源冲突: %w", err)
	case errdefs.IsForbidden(err):
		return fmt.Errorf("Docker 操作被禁止: %w", err)
	case errdefs.IsNotImplemented(err):
		return fmt.Errorf("Docker 操作未实现: %w", err)
	case errdefs.IsSystem(err):
		return fmt.Errorf("Docker 系统错误: %w", err)
	case errdefs.IsDeadline(err), errdefs.IsUnavailable(err):
		return fmt.Errorf("Docker 操作超时或服务不可用: %w", err)
	case errdefs.IsCancelled(err):
		return fmt.Errorf("Docker 操作被取消: %w", err)
	default:
		return fmt.Errorf("Docker 操作错误: %w", err)
	}
}
