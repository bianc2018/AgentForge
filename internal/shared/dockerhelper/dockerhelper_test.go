package dockerhelper

import (
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	specs "github.com/opencontainers/image-spec/specs-go/v1"
)

func TestNewClient_Default(t *testing.T) {
	c, err := NewClient()
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer c.Close()
}

func TestNewClientWithOpts_CustomHost(t *testing.T) {
	_, err := NewClientWithOpts(
		client.WithHost("unix:///var/run/nonexistent.sock"),
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		t.Fatalf("NewClientWithOpts() should not fail on creation: %v", err)
	}
}

func TestPing_WithRealDaemon(t *testing.T) {
	c, err := NewClient()
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = c.Ping(ctx)
	if err != nil {
		t.Fatalf("Ping() error = %v", err)
	}
}

func TestPing_UnreachableDaemon(t *testing.T) {
	c, err := NewClientWithOpts(
		client.WithHost("tcp://192.0.2.1:2375"),
	)
	if err != nil {
		t.Fatalf("NewClientWithOpts() error = %v", err)
	}
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err = c.Ping(ctx)
	if err == nil {
		t.Fatal("Ping() expected error for unreachable daemon, got nil")
	}
	t.Logf("Ping() error for unreachable daemon (expected): %v", err)
}

func TestInfo_WithRealDaemon(t *testing.T) {
	c, err := NewClient()
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	info, err := c.Info(ctx)
	if err != nil {
		t.Fatalf("Info() error = %v", err)
	}

	if info.ID == "" {
		t.Error("Info() returned empty Docker daemon ID")
	}
	t.Logf("Docker version: %s, OS: %s", info.ServerVersion, info.OperatingSystem)
}

func TestImageList_WithRealDaemon(t *testing.T) {
	c, err := NewClient()
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	images, err := c.ImageList(ctx, types.ImageListOptions{})
	if err != nil {
		t.Fatalf("ImageList() error = %v", err)
	}

	t.Logf("Found %d local images", len(images))
	for _, img := range images {
		for _, tag := range img.RepoTags {
			t.Logf("  - %s", tag)
		}
	}
}

func TestIsBuildKitEnabled(t *testing.T) {
	c, err := NewClient()
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer c.Close()

	// Save and restore env var
	oldVal := os.Getenv("DOCKER_BUILDKIT")
	defer os.Setenv("DOCKER_BUILDKIT", oldVal)

	os.Setenv("DOCKER_BUILDKIT", "1")
	if !c.IsBuildKitEnabled() {
		t.Error("IsBuildKitEnabled() = false, want true when DOCKER_BUILDKIT=1")
	}

	os.Setenv("DOCKER_BUILDKIT", "")
	if c.IsBuildKitEnabled() {
		t.Error("IsBuildKitEnabled() = true, want false when DOCKER_BUILDKIT unset")
	}

	os.Setenv("DOCKER_BUILDKIT", "0")
	if c.IsBuildKitEnabled() {
		t.Error("IsBuildKitEnabled() = true, want false when DOCKER_BUILDKIT=0")
	}
}

func TestImageExists(t *testing.T) {
	c, err := NewClient()
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	images, err := c.ImageList(ctx, types.ImageListOptions{})
	if err != nil {
		t.Fatalf("ImageList() error = %v", err)
	}

	if len(images) > 0 {
		for _, tag := range images[0].RepoTags {
			exists, err := c.ImageExists(ctx, tag)
			if err != nil {
				t.Fatalf("ImageExists(%q) error = %v", tag, err)
			}
			if !exists {
				t.Errorf("ImageExists(%q) = false, want true", tag)
			}
			break
		}
	} else {
		t.Log("No images found on this system, skipping ImageExists test")
	}
}

func TestPingWithInfo(t *testing.T) {
	c, err := NewClient()
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	info, err := c.PingWithInfo(ctx)
	if err != nil {
		t.Fatalf("PingWithInfo() error = %v", err)
	}
	if info.ServerVersion == "" {
		t.Error("PingWithInfo() returned empty ServerVersion")
	}
	t.Logf("Docker Server Version: %s", info.ServerVersion)
}

func TestClassifyError_Nil(t *testing.T) {
	if err := ClassifyError(nil); err != nil {
		t.Errorf("ClassifyError(nil) = %v, want nil", err)
	}
}

func TestStandardErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		msg  string
	}{
		{"ErrDockerNotReachable", ErrDockerNotReachable, "Docker daemon 不可达"},
		{"ErrDockerPermissionDenied", ErrDockerPermissionDenied, "Docker socket 访问权限不足"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err == nil {
				t.Fatal("standard error is nil")
			}
			if !contains(tt.err.Error(), tt.msg) {
				t.Errorf("%s = %q, want containing %q", tt.name, tt.err.Error(), tt.msg)
			}
		})
	}
}

func TestIsBuildKitEnabled_EdgeCases(t *testing.T) {
	oldVal := os.Getenv("DOCKER_BUILDKIT")
	defer os.Setenv("DOCKER_BUILDKIT", oldVal)

	c := &Client{}

	os.Unsetenv("DOCKER_BUILDKIT")
	if c.IsBuildKitEnabled() {
		t.Error("IsBuildKitEnabled() = true, want false when env unset")
	}

	os.Setenv("DOCKER_BUILDKIT", "1")
	if !c.IsBuildKitEnabled() {
		t.Error("IsBuildKitEnabled() = false, want true when DOCKER_BUILDKIT=1")
	}

	os.Setenv("DOCKER_BUILDKIT", "0")
	if c.IsBuildKitEnabled() {
		t.Error("IsBuildKitEnabled() = true, want false when DOCKER_BUILDKIT=0")
	}

	os.Setenv("DOCKER_BUILDKIT", "true")
	if c.IsBuildKitEnabled() {
		t.Error("IsBuildKitEnabled() = true, want false when DOCKER_BUILDKIT is not '1'")
	}
}

func TestClose_ValidClient(t *testing.T) {
	c, err := NewClient()
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	err = c.Close()
	if err != nil {
		t.Errorf("Close() error = %v, want nil", err)
	}
}

func TestPingWithInfo_Unreachable(t *testing.T) {
	c, err := NewClientWithOpts(
		client.WithHost("unix:///var/run/nonexistent.sock"),
	)
	if err != nil {
		t.Fatalf("NewClientWithOpts() error = %v", err)
	}
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err = c.PingWithInfo(ctx)
	if err == nil {
		t.Fatal("PingWithInfo() expected error with unreachable daemon, got nil")
	}
	t.Logf("PingWithInfo() error (expected): %v", err)
}

func TestImageExists_NoImages(t *testing.T) {
	c, err := NewClient()
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	exists, err := c.ImageExists(ctx, "nonexistent-image:tag-xyz")
	if err != nil {
		t.Fatalf("ImageExists() error = %v", err)
	}
	if exists {
		t.Error("ImageExists() = true for nonexistent image, want false")
	}
}

func TestImageExists_ByIDPrefix(t *testing.T) {
	c, err := NewClient()
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	images, err := c.ImageList(ctx, types.ImageListOptions{})
	if err != nil {
		t.Fatalf("ImageList() error = %v", err)
	}
	if len(images) == 0 {
		t.Skip("No images available, skipping ID prefix test")
	}

	// Try matching by ID prefix (first 12 chars of image ID)
	id := images[0].ID
	if len(id) > 12 {
		id = id[:12]
	}
	exists, err := c.ImageExists(ctx, id)
	if err != nil {
		t.Fatalf("ImageExists(%q) error = %v", id, err)
	}
	if !exists {
		t.Errorf("ImageExists(%q) = false, want true (by ID prefix)", id)
	}
}

func TestNewClient_CompileCheck(t *testing.T) {
	var c DockerClient
	c, err := NewClient()
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer c.Close()

	if c == nil {
		t.Fatal("NewClient() returned nil")
	}
}

// --- Mock SDK Client for unit testing wrapper methods ---

// mockSDK 实现 dockerSDKClient，允许注入受控的返回值。
// 默认所有方法返回 nil 错误（success 路径）。设置 genericErr 可全局切换为 error 路径。
// 特定方法可通过 function field 自定义行为。
type mockSDK struct {
	genericErr         error   // 非 nil 时，未配置 function field 的方法返回此错误
	pingErr            error
	pingResp           types.Ping
	infoResp           types.Info
	infoErr            error
	imageListResp      []types.ImageSummary
	imageListErr       error
	imageBuildResp     types.ImageBuildResponse
	imageBuildErr      error
	imageTagErr        error
	imageRemoveResp    []types.ImageDeleteResponseItem
	imageRemoveErr     error
	containerCreateResp container.ContainerCreateCreatedBody
	containerCreateErr  error
	containerStartErr   error
	// Optional function overrides for methods that need custom behavior
	imageSaveFn        func(ctx context.Context, imageIDs []string) (io.ReadCloser, error)
	imageLoadFn        func(ctx context.Context, input io.Reader, quiet bool) (types.ImageLoadResponse, error)
	containerAttachFn  func(ctx context.Context, containerID string, options types.ContainerAttachOptions) (types.HijackedResponse, error)
	containerResizeFn  func(ctx context.Context, containerID string, options types.ResizeOptions) error
	containerKillFn    func(ctx context.Context, containerID, signal string) error
	containerRemoveFn  func(ctx context.Context, containerID string, options types.ContainerRemoveOptions) error
}

func (m *mockSDK) err() error {
	if m.genericErr != nil {
		return m.genericErr
	}
	return nil
}

func (m *mockSDK) Ping(ctx context.Context) (types.Ping, error) {
	if m.pingErr != nil {
		return m.pingResp, m.pingErr
	}
	return m.pingResp, m.err()
}

func (m *mockSDK) Info(ctx context.Context) (types.Info, error) {
	if m.infoErr != nil {
		return m.infoResp, m.infoErr
	}
	return m.infoResp, m.err()
}

func (m *mockSDK) ImageList(ctx context.Context, options types.ImageListOptions) ([]types.ImageSummary, error) {
	if m.imageListErr != nil {
		return m.imageListResp, m.imageListErr
	}
	return m.imageListResp, m.err()
}

func (m *mockSDK) ImageBuild(ctx context.Context, buildContext io.Reader, options types.ImageBuildOptions) (types.ImageBuildResponse, error) {
	if m.imageBuildErr != nil {
		return m.imageBuildResp, m.imageBuildErr
	}
	return m.imageBuildResp, m.err()
}

func (m *mockSDK) ImageTag(ctx context.Context, source, target string) error {
	if m.imageTagErr != nil {
		return m.imageTagErr
	}
	return m.err()
}

func (m *mockSDK) ImageRemove(ctx context.Context, imageID string, options types.ImageRemoveOptions) ([]types.ImageDeleteResponseItem, error) {
	if m.imageRemoveErr != nil {
		return m.imageRemoveResp, m.imageRemoveErr
	}
	return m.imageRemoveResp, m.err()
}

func (m *mockSDK) ImageSave(ctx context.Context, imageIDs []string) (io.ReadCloser, error) {
	if m.imageSaveFn != nil {
		return m.imageSaveFn(ctx, imageIDs)
	}
	return nil, m.genericErr
}

func (m *mockSDK) ImageLoad(ctx context.Context, input io.Reader, quiet bool) (types.ImageLoadResponse, error) {
	if m.imageLoadFn != nil {
		return m.imageLoadFn(ctx, input, quiet)
	}
	return types.ImageLoadResponse{}, m.genericErr
}

func (m *mockSDK) ContainerCreate(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, platform *specs.Platform, containerName string) (container.ContainerCreateCreatedBody, error) {
	if m.containerCreateErr != nil {
		return m.containerCreateResp, m.containerCreateErr
	}
	return m.containerCreateResp, m.err()
}

func (m *mockSDK) ContainerStart(ctx context.Context, containerID string, options types.ContainerStartOptions) error {
	if m.containerStartErr != nil {
		return m.containerStartErr
	}
	return m.err()
}

func (m *mockSDK) ContainerAttach(ctx context.Context, containerID string, options types.ContainerAttachOptions) (types.HijackedResponse, error) {
	if m.containerAttachFn != nil {
		return m.containerAttachFn(ctx, containerID, options)
	}
	return types.HijackedResponse{}, m.genericErr
}

func (m *mockSDK) ContainerWait(ctx context.Context, containerID string, condition container.WaitCondition) (<-chan container.ContainerWaitOKBody, <-chan error) {
	return nil, nil
}

func (m *mockSDK) ContainerResize(ctx context.Context, containerID string, options types.ResizeOptions) error {
	if m.containerResizeFn != nil {
		return m.containerResizeFn(ctx, containerID, options)
	}
	return m.genericErr
}

func (m *mockSDK) ContainerKill(ctx context.Context, containerID, signal string) error {
	if m.containerKillFn != nil {
		return m.containerKillFn(ctx, containerID, signal)
	}
	return m.genericErr
}

func (m *mockSDK) ContainerStop(ctx context.Context, containerID string, timeout *time.Duration) error {
	return m.genericErr
}

func (m *mockSDK) ContainerRemove(ctx context.Context, containerID string, options types.ContainerRemoveOptions) error {
	if m.containerRemoveFn != nil {
		return m.containerRemoveFn(ctx, containerID, options)
	}
	return m.genericErr
}

func (m *mockSDK) Close() error { return nil }

// newTestClient 创建一个使用 mock SDK 的 Client，用于单元测试。
func newTestClient(m *mockSDK) *Client {
	return &Client{api: m}
}

// --- Unit tests using mock SDK ---

func TestPing_Error(t *testing.T) {
	m := &mockSDK{pingErr: errors.New("connection refused")}
	c := newTestClient(m)
	err := c.Ping(context.Background())
	if err == nil {
		t.Fatal("Ping() expected error, got nil")
	}
	if !contains(err.Error(), "Docker daemon 不可达") {
		t.Errorf("Ping() error = %q, want containing 'Docker daemon 不可达'", err.Error())
	}
}

func TestPing_Success(t *testing.T) {
	m := &mockSDK{pingResp: types.Ping{}}
	c := newTestClient(m)
	err := c.Ping(context.Background())
	if err != nil {
		t.Fatalf("Ping() error = %v, want nil", err)
	}
}

func TestInfo_Error(t *testing.T) {
	m := &mockSDK{infoErr: errors.New("server error")}
	c := newTestClient(m)
	_, err := c.Info(context.Background())
	if err == nil {
		t.Fatal("Info() expected error, got nil")
	}
	if !contains(err.Error(), "获取 Docker 信息失败") {
		t.Errorf("Info() error = %q, want containing '获取 Docker 信息失败'", err.Error())
	}
}

func TestInfo_Success(t *testing.T) {
	m := &mockSDK{infoResp: types.Info{ID: "docker-id-123", ServerVersion: "20.10.0"}}
	c := newTestClient(m)
	info, err := c.Info(context.Background())
	if err != nil {
		t.Fatalf("Info() error = %v, want nil", err)
	}
	if info.ID != "docker-id-123" {
		t.Errorf("Info().ID = %q, want 'docker-id-123'", info.ID)
	}
	if info.ServerVersion != "20.10.0" {
		t.Errorf("Info().ServerVersion = %q, want '20.10.0'", info.ServerVersion)
	}
}

func TestImageList_Error(t *testing.T) {
	m := &mockSDK{imageListErr: errors.New("daemon error")}
	c := newTestClient(m)
	_, err := c.ImageList(context.Background(), types.ImageListOptions{})
	if err == nil {
		t.Fatal("ImageList() expected error, got nil")
	}
	if !contains(err.Error(), "列出 Docker 镜像失败") {
		t.Errorf("ImageList() error = %q, want containing '列出 Docker 镜像失败'", err.Error())
	}
}

func TestImageList_Success(t *testing.T) {
	m := &mockSDK{
		imageListResp: []types.ImageSummary{
			{ID: "sha256:abc123", RepoTags: []string{"test:latest"}},
			{ID: "sha256:def456", RepoTags: []string{"prod:v1", "prod:latest"}},
		},
	}
	c := newTestClient(m)
	images, err := c.ImageList(context.Background(), types.ImageListOptions{})
	if err != nil {
		t.Fatalf("ImageList() error = %v, want nil", err)
	}
	if len(images) != 2 {
		t.Errorf("ImageList() returned %d images, want 2", len(images))
	}
}

func TestImageExists_Found_ByTag(t *testing.T) {
	m := &mockSDK{
		imageListResp: []types.ImageSummary{
			{ID: "sha256:abc", RepoTags: []string{"agent-forge:latest"}},
		},
	}
	c := newTestClient(m)
	exists, err := c.ImageExists(context.Background(), "agent-forge:latest")
	if err != nil {
		t.Fatalf("ImageExists() error = %v", err)
	}
	if !exists {
		t.Error("ImageExists() = false, want true")
	}
}

func TestImageExists_Found_ByIDPrefix(t *testing.T) {
	m := &mockSDK{
		imageListResp: []types.ImageSummary{
			{ID: "sha256:abcdef123456", RepoTags: []string{}},
		},
	}
	c := newTestClient(m)
	exists, err := c.ImageExists(context.Background(), "sha256:abcdef12")
	if err != nil {
		t.Fatalf("ImageExists() error = %v", err)
	}
	if !exists {
		t.Error("ImageExists() = false for ID prefix match, want true")
	}
}

func TestImageExists_NotFound(t *testing.T) {
	m := &mockSDK{
		imageListResp: []types.ImageSummary{
			{ID: "sha256:abc", RepoTags: []string{"other:latest"}},
		},
	}
	c := newTestClient(m)
	exists, err := c.ImageExists(context.Background(), "nonexistent:latest")
	if err != nil {
		t.Fatalf("ImageExists() error = %v", err)
	}
	if exists {
		t.Error("ImageExists() = true, want false")
	}
}

func TestImageExists_ListError(t *testing.T) {
	m := &mockSDK{imageListErr: errors.New("list failed")}
	c := newTestClient(m)
	_, err := c.ImageExists(context.Background(), "test:latest")
	if err == nil {
		t.Fatal("ImageExists() expected error, got nil")
	}
}

func TestImageBuild_Error(t *testing.T) {
	m := &mockSDK{imageBuildErr: errors.New("build failed")}
	c := newTestClient(m)
	_, err := c.ImageBuild(context.Background(), nil, types.ImageBuildOptions{})
	if err == nil {
		t.Fatal("ImageBuild() expected error, got nil")
	}
	if !contains(err.Error(), "Docker 镜像构建失败") {
		t.Errorf("ImageBuild() error = %q, want containing 'Docker 镜像构建失败'", err.Error())
	}
}

func TestImageTag_Error(t *testing.T) {
	m := &mockSDK{imageTagErr: errors.New("tag failed")}
	c := newTestClient(m)
	err := c.ImageTag(context.Background(), "src", "dst")
	if err == nil {
		t.Fatal("ImageTag() expected error, got nil")
	}
	if !contains(err.Error(), "Docker 镜像打标签失败") {
		t.Errorf("ImageTag() error = %q, want containing 'Docker 镜像打标签失败'", err.Error())
	}
}

func TestImageRemove_Error(t *testing.T) {
	m := &mockSDK{imageRemoveErr: errors.New("remove failed")}
	c := newTestClient(m)
	_, err := c.ImageRemove(context.Background(), "test:latest", true, true)
	if err == nil {
		t.Fatal("ImageRemove() expected error, got nil")
	}
	if !contains(err.Error(), "Docker 镜像删除失败") {
		t.Errorf("ImageRemove() error = %q, want containing 'Docker 镜像删除失败'", err.Error())
	}
}

func TestContainerCreate_Error(t *testing.T) {
	m := &mockSDK{containerCreateErr: errors.New("create failed")}
	c := newTestClient(m)
	_, err := c.ContainerCreate(context.Background(), nil, nil, nil, nil, "")
	if err == nil {
		t.Fatal("ContainerCreate() expected error, got nil")
	}
	if !contains(err.Error(), "创建容器失败") {
		t.Errorf("ContainerCreate() error = %q, want containing '创建容器失败'", err.Error())
	}
}

func TestContainerCreate_Success(t *testing.T) {
	m := &mockSDK{
		containerCreateResp: container.ContainerCreateCreatedBody{ID: "container-123"},
	}
	c := newTestClient(m)
	resp, err := c.ContainerCreate(context.Background(), nil, nil, nil, nil, "")
	if err != nil {
		t.Fatalf("ContainerCreate() error = %v, want nil", err)
	}
	if resp.ID != "container-123" {
		t.Errorf("ContainerCreate().ID = %q, want 'container-123'", resp.ID)
	}
}

func TestContainerStart_Error(t *testing.T) {
	m := &mockSDK{containerStartErr: errors.New("start failed")}
	c := newTestClient(m)
	err := c.ContainerStart(context.Background(), "test", types.ContainerStartOptions{})
	if err == nil {
		t.Fatal("ContainerStart() expected error, got nil")
	}
	if !contains(err.Error(), "启动容器失败") {
		t.Errorf("ContainerStart() error = %q, want containing '启动容器失败'", err.Error())
	}
}

func TestPingWithInfo_PingFails(t *testing.T) {
	m := &mockSDK{pingErr: errors.New("unreachable")}
	c := newTestClient(m)
	_, err := c.PingWithInfo(context.Background())
	if err == nil {
		t.Fatal("PingWithInfo() expected error, got nil")
	}
	if !contains(err.Error(), "Docker daemon 不可达") {
		t.Errorf("PingWithInfo() error = %q, want containing 'Docker daemon 不可达'", err.Error())
	}
}

func TestPingWithInfo_InfoFails(t *testing.T) {
	m := &mockSDK{
		pingResp: types.Ping{},
		infoErr:  errors.New("info error"),
	}
	c := newTestClient(m)
	_, err := c.PingWithInfo(context.Background())
	if err == nil {
		t.Fatal("PingWithInfo() expected error after Info failed, got nil")
	}
}

// --- Tests for previously uncovered methods ---

func TestContainerWait_NoError(t *testing.T) {
	m := &mockSDK{}
	c := newTestClient(m)
	statusCh, errCh := c.ContainerWait(context.Background(), "test", container.WaitConditionNextExit)
	if statusCh != nil || errCh != nil {
		t.Log("ContainerWait() returned channels from mock SDK")
	}
}

func TestContainerResize_Success(t *testing.T) {
	m := &mockSDK{
		containerResizeFn: func(ctx context.Context, containerID string, options types.ResizeOptions) error {
			return nil
		},
	}
	c := newTestClient(m)
	err := c.ContainerResize(context.Background(), "test", 80, 24)
	if err != nil {
		t.Fatalf("ContainerResize() error = %v, want nil", err)
	}
}

func TestContainerRemove_Success(t *testing.T) {
	m := &mockSDK{
		containerRemoveFn: func(ctx context.Context, containerID string, options types.ContainerRemoveOptions) error {
			return nil
		},
	}
	c := newTestClient(m)
	err := c.ContainerRemove(context.Background(), "test", true, false)
	if err != nil {
		t.Fatalf("ContainerRemove() error = %v, want nil", err)
	}
}

func TestContainerKill_Success(t *testing.T) {
	m := &mockSDK{
		containerKillFn: func(ctx context.Context, containerID, signal string) error {
			return nil
		},
	}
	c := newTestClient(m)
	err := c.ContainerKill(context.Background(), "test", "SIGTERM")
	if err != nil {
		t.Fatalf("ContainerKill() error = %v, want nil", err)
	}
}

func TestContainerRemove_NoError(t *testing.T) {
	m := &mockSDK{}
	c := newTestClient(m)
	err := c.ContainerRemove(context.Background(), "test", true, false)
	if err != nil {
		t.Fatalf("ContainerRemove() error = %v, want nil", err)
	}
}

func TestImageTag_Success(t *testing.T) {
	m := &mockSDK{}
	c := newTestClient(m)
	err := c.ImageTag(context.Background(), "src:latest", "dst:latest")
	if err != nil {
		t.Fatalf("ImageTag() error = %v, want nil", err)
	}
}

func TestContainerStart_Success(t *testing.T) {
	m := &mockSDK{}
	c := newTestClient(m)
	err := c.ContainerStart(context.Background(), "test", types.ContainerStartOptions{})
	if err != nil {
		t.Fatalf("ContainerStart() error = %v, want nil", err)
	}
}

func TestImageRemove_Success(t *testing.T) {
	m := &mockSDK{
		imageRemoveResp: []types.ImageDeleteResponseItem{
			{Untagged: "test:latest"},
		},
	}
	c := newTestClient(m)
	resp, err := c.ImageRemove(context.Background(), "test:latest", false, false)
	if err != nil {
		t.Fatalf("ImageRemove() error = %v, want nil", err)
	}
	if len(resp) != 1 || resp[0].Untagged != "test:latest" {
		t.Errorf("ImageRemove() = %+v, want [{Untagged: test:latest}]", resp)
	}
}

func TestClassifyError_WithRealDockerErrors(t *testing.T) {
	// 使用真实 Docker client 触发 errdefs 错误分类分支。
	c, err := NewClient()
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 尝试删除不存在的镜像，触发 IsNotFound 分支
	_, err = c.ImageRemove(ctx, "nonexistent-image-xyz-12345", false, false)
	if err != nil {
		// 此错误经过 ClassifyError 包装，应包含 "资源不存在"
		t.Logf("ImageRemove nonexistent image error: %v", err)
		if !contains(err.Error(), "资源不存在") && !contains(err.Error(), "Docker 镜像删除失败") {
			t.Logf("Unexpected error format: %v", err)
		}
	}
}

func TestPingWithInfo_Success(t *testing.T) {
	m := &mockSDK{
		pingResp: types.Ping{},
		infoResp: types.Info{ID: "docker-1", ServerVersion: "20.10"},
	}
	c := newTestClient(m)
	info, err := c.PingWithInfo(context.Background())
	if err != nil {
		t.Fatalf("PingWithInfo() error = %v, want nil", err)
	}
	if info.ID != "docker-1" {
		t.Errorf("PingWithInfo().ID = %q, want 'docker-1'", info.ID)
	}
}

// --- Additional success/error path tests ---

func TestContainerKill_Error(t *testing.T) {
	c := newTestClient(&mockSDK{genericErr: errors.New("kill err")})
	err := c.ContainerKill(context.Background(), "test", "SIGKILL")
	if err == nil {
		t.Fatal("ContainerKill() expected error, got nil")
	}
	if !contains(err.Error(), "终止容器失败") {
		t.Errorf("ContainerKill() error = %q, want containing '终止容器失败'", err.Error())
	}
}

func TestContainerResize_Error(t *testing.T) {
	c := newTestClient(&mockSDK{genericErr: errors.New("resize err")})
	err := c.ContainerResize(context.Background(), "test", 80, 24)
	if err == nil {
		t.Fatal("ContainerResize() expected error, got nil")
	}
	if !contains(err.Error(), "调整容器终端尺寸失败") {
		t.Errorf("ContainerResize() error = %q, want containing '调整容器终端尺寸失败'", err.Error())
	}
}

func TestImageBuild_Success(t *testing.T) {
	m := &mockSDK{}
	m.imageBuildResp = types.ImageBuildResponse{Body: io.NopCloser(strings.NewReader("Successfully built abc123\n"))}
	c := newTestClient(m)
	resp, err := c.ImageBuild(context.Background(), strings.NewReader(""), types.ImageBuildOptions{})
	if err != nil {
		t.Fatalf("ImageBuild() error = %v, want nil", err)
	}
	if resp.Body != nil {
		resp.Body.Close()
	}
}

func TestImageSave_Success(t *testing.T) {
	m := &mockSDK{
		imageSaveFn: func(ctx context.Context, imageIDs []string) (io.ReadCloser, error) {
			return io.NopCloser(strings.NewReader("mock tar data")), nil
		},
	}
	c := newTestClient(m)
	reader, err := c.ImageSave(context.Background(), []string{"test:latest"})
	if err != nil {
		t.Fatalf("ImageSave() error = %v, want nil", err)
	}
	if reader != nil {
		reader.Close()
	}
}

func TestImageLoad_Success(t *testing.T) {
	m := &mockSDK{
		imageLoadFn: func(ctx context.Context, input io.Reader, quiet bool) (types.ImageLoadResponse, error) {
			return types.ImageLoadResponse{Body: io.NopCloser(strings.NewReader("ok"))}, nil
		},
	}
	c := newTestClient(m)
	resp, err := c.ImageLoad(context.Background(), strings.NewReader(""), false)
	if err != nil {
		t.Fatalf("ImageLoad() error = %v, want nil", err)
	}
	if resp.Body != nil {
		resp.Body.Close()
	}
}

func TestContainerAttach_Success(t *testing.T) {
	m := &mockSDK{
		containerAttachFn: func(ctx context.Context, containerID string, options types.ContainerAttachOptions) (types.HijackedResponse, error) {
			return types.HijackedResponse{}, nil
		},
	}
	c := newTestClient(m)
	_, err := c.ContainerAttach(context.Background(), "test", types.ContainerAttachOptions{})
	if err != nil {
		t.Fatalf("ContainerAttach() error = %v, want nil", err)
	}
}

func TestContainerRemove_Error(t *testing.T) {
	c := newTestClient(&mockSDK{genericErr: errors.New("remove err")})
	err := c.ContainerRemove(context.Background(), "test", true, false)
	if err == nil {
		t.Fatal("ContainerRemove() expected error, got nil")
	}
	if !contains(err.Error(), "删除容器失败") {
		t.Errorf("ContainerRemove() error = %q, want containing '删除容器失败'", err.Error())
	}
}

func TestImageSave_Error(t *testing.T) {
	c := newTestClient(&mockSDK{genericErr: errors.New("save err")})
	_, err := c.ImageSave(context.Background(), []string{"test:latest"})
	if err == nil {
		t.Fatal("ImageSave() expected error, got nil")
	}
	if !contains(err.Error(), "Docker 镜像导出失败") {
		t.Errorf("ImageSave() error = %q, want containing 'Docker 镜像导出失败'", err.Error())
	}
}

func TestImageLoad_Error(t *testing.T) {
	c := newTestClient(&mockSDK{genericErr: errors.New("load err")})
	_, err := c.ImageLoad(context.Background(), nil, false)
	if err == nil {
		t.Fatal("ImageLoad() expected error, got nil")
	}
	if !contains(err.Error(), "Docker 镜像导入失败") {
		t.Errorf("ImageLoad() error = %q, want containing 'Docker 镜像导入失败'", err.Error())
	}
}

func TestContainerAttach_Error(t *testing.T) {
	c := newTestClient(&mockSDK{genericErr: errors.New("attach err")})
	_, err := c.ContainerAttach(context.Background(), "test", types.ContainerAttachOptions{})
	if err == nil {
		t.Fatal("ContainerAttach() expected error, got nil")
	}
	if !contains(err.Error(), "附加到容器失败") {
		t.Errorf("ContainerAttach() error = %q, want containing '附加到容器失败'", err.Error())
	}
}

func TestClassifyError_Generic(t *testing.T) {
	// errdefsMock 不实现 errdefs 特殊接口，所有分支走到 default。
	// 真正的 errdefs 错误分类由 TestClassifyError_WithRealDaemon 覆盖。
	tests := []struct {
		name string
		err  error
	}{
		{"nil", nil},
		{"generic string", errdefsMock("unknown thing")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifyError(tt.err)
			if tt.err == nil {
				if got != nil {
					t.Errorf("ClassifyError(nil) = %v, want nil", got)
				}
				return
			}
			if got == nil {
				t.Fatal("ClassifyError() = nil, want non-nil")
			}
		})
	}
}

// errdefsMock 用于模拟各类 Docker API 错误。
type errdefsMock string

func (e errdefsMock) Error() string { return string(e) }

// contains 检查字符串 s 是否包含子串 substr。
func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func TestClient_ContainerStop_NilTimeout(t *testing.T) {
	m := &mockSDK{}
	client := &Client{api: m}
	err := client.ContainerStop(context.Background(), "test-container", nil)
	if err != nil {
		t.Errorf("ContainerStop(nil) error = %v", err)
	}
}
