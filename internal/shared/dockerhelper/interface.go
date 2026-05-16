// Package dockerhelper 封装 Docker Engine API 调用。
package dockerhelper

import (
	"context"
	"io"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	specs "github.com/opencontainers/image-spec/specs-go/v1"
)

// DockerClient 定义 Docker 操作的完整接口。
//
// mockgen 从此接口生成 mock，用于各模块的单元测试。
// *Client 实现了此接口的全部方法。
//
// regenerate mock in separate package: mockgen -destination=mock/mock_client.go -package=mock github.com/agent-forge/cli/internal/shared/dockerhelper DockerClient
type DockerClient interface {
	Ping(ctx context.Context) error
	Info(ctx context.Context) (types.Info, error)
	IsBuildKitEnabled() bool
	PingWithInfo(ctx context.Context) (*types.Info, error)

	ImageList(ctx context.Context, opts types.ImageListOptions) ([]types.ImageSummary, error)
	ImageExists(ctx context.Context, ref string) (bool, error)
	ImageBuild(ctx context.Context, buildContext io.Reader, options types.ImageBuildOptions) (types.ImageBuildResponse, error)
	ImageTag(ctx context.Context, source, target string) error
	ImageRemove(ctx context.Context, imageID string, force, prune bool) ([]types.ImageDeleteResponseItem, error)
	ImageSave(ctx context.Context, imageIDs []string) (io.ReadCloser, error)
	ImageLoad(ctx context.Context, input io.Reader, quiet bool) (types.ImageLoadResponse, error)

	ContainerCreate(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, platform *specs.Platform, containerName string) (container.ContainerCreateCreatedBody, error)
	ContainerStart(ctx context.Context, containerID string, opts types.ContainerStartOptions) error
	ContainerAttach(ctx context.Context, containerID string, opts types.ContainerAttachOptions) (types.HijackedResponse, error)
	ContainerWait(ctx context.Context, containerID string, condition container.WaitCondition) (<-chan container.ContainerWaitOKBody, <-chan error)
	ContainerResize(ctx context.Context, containerID string, height, width uint) error
	ContainerKill(ctx context.Context, containerID, signal string) error
		ContainerStop(ctx context.Context, containerID string, timeout *time.Duration) error
	ContainerRemove(ctx context.Context, containerID string, force, removeVolumes bool) error

	Close() error
}

// Compile-time check: *Client implements DockerClient.
var _ DockerClient = (*Client)(nil)
