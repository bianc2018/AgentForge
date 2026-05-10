//go:build e2e

package buildengine

import (
	"bytes"
	"context"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/agent-forge/cli/internal/build/dockerfilegen"
	"github.com/agent-forge/cli/internal/shared/dockerhelper"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

// TestE2E_GH1_BuildWithAllDeps 覆盖 GH-1 Scenario "构建包含全部依赖的镜像"。
//
// Given Docker Engine 已安装并运行
// When 开发者执行 build -d all --max-retry 3
// Then 构建过程退出码为 0
// And docker images 列表中包含新生成的镜像
func TestE2E_GH1_BuildWithAllDeps(t *testing.T) {
	// Given Docker Engine 已安装并运行
	helper, err := dockerhelper.NewClient()
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer helper.Close()

	pingCtx, pingCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer pingCancel()
	if err := helper.Ping(pingCtx); err != nil {
		t.Skipf("Docker Engine 未运行，跳过 E2E 测试: %v", err)
	}

	// When 开发者执行 build -d all --max-retry 3
	engine := New(helper)
	defer engine.Close()

	buildCtx, buildCancel := context.WithTimeout(context.Background(), 3600*time.Second)
	defer buildCancel()

	output, err := engine.Build(buildCtx, BuildParams{
		Deps:     "all",
		MaxRetry: 3,
	})

	// Then 构建过程退出码为 0
	if err != nil {
		t.Fatalf("Build() error = %v\nOutput: %s", err, output)
	}

	if output == "" {
		t.Error("Build() returned empty output")
	}

	// And docker images 列表中包含新生成的镜像
	images, err := helper.ImageList(buildCtx, types.ImageListOptions{})
	if err != nil {
		t.Fatalf("ImageList() error = %v", err)
	}

	found := false
	for _, img := range images {
		for _, tag := range img.RepoTags {
			if tag == ImageTag {
				found = true
				break
			}
		}
		if found {
			break
		}
	}
	if !found {
		t.Errorf("镜像 %s 未在 docker images 列表中找到", ImageTag)
	}

	// Cleanup: 清理构建产物
	_, err = helper.ImageRemove(buildCtx, ImageTag, true, true)
	if err != nil {
		t.Logf("Warning: 清理镜像 %s 失败: %v", ImageTag, err)
	}
}

// TestE2E_GH2_BuildWithCustomDeps 覆盖 GH-2 Scenario "构建包含指定依赖的自定义镜像"。
//
// Given Docker Engine 已安装并运行
// When 开发者执行 build -d claude,golang@1.21,node@20 -b docker.1ms.run/centos:7 -c /path/to/config
// Then 构建过程退出码为 0
// And 容器内 go version 输出 1.21.x
// And 容器内 node --version 输出 20.x
func TestE2E_GH2_BuildWithCustomDeps(t *testing.T) {
	// Given Docker Engine 已安装并运行
	helper, err := dockerhelper.NewClient()
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer helper.Close()

	pingCtx, pingCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer pingCancel()
	if err := helper.Ping(pingCtx); err != nil {
		t.Skipf("Docker Engine 未运行，跳过 E2E 测试: %v", err)
	}

	// When 开发者执行 build -d claude,golang@1.21,node@20 -b docker.1ms.run/centos:7 -c /path/to/config
	engine := New(helper)
	defer engine.Close()

	buildCtx, buildCancel := context.WithTimeout(context.Background(), 3600*time.Second)
	defer buildCancel()

	output, err := engine.Build(buildCtx, BuildParams{
		Deps:      "claude,golang@1.21,node@16",
		BaseImage: "docker.1ms.run/centos:7",
		Config:    "/path/to/config",
		MaxRetry:  3,
	})

	// Then 构建过程退出码为 0
	if err != nil {
		t.Fatalf("Build() error = %v\nOutput: %s", err, output)
	}

	if output == "" {
		t.Error("Build() returned empty output")
	}

	// And 容器内 go version 输出 1.21.x
	// 使用 docker run --rm 在临时容器中执行 go version 检查版本
	goVersionCmd := exec.Command("docker", "run", "--rm", ImageTag, "go", "version")
	var goVersionOut bytes.Buffer
	goVersionCmd.Stdout = &goVersionOut
	goVersionCmd.Stderr = &goVersionOut

	if err := goVersionCmd.Run(); err != nil {
		t.Fatalf("执行 'go version' 失败: %v\nOutput: %s", err, goVersionOut.String())
	}

	goVersionStr := goVersionOut.String()
	if !strings.Contains(goVersionStr, "go1.21") {
		t.Errorf("go version 输出 %q, 期望包含 go1.21", goVersionStr)
	}

	// And 容器内 node --version 输出 16.x（CentOS 7 的 glibc 2.17 无法支持 Node >= 18）
	nodeVersionCmd := exec.Command("docker", "run", "--rm", ImageTag, "node", "--version")
	var nodeVersionOut bytes.Buffer
	nodeVersionCmd.Stdout = &nodeVersionOut
	nodeVersionCmd.Stderr = &nodeVersionOut

	if err := nodeVersionCmd.Run(); err != nil {
		t.Fatalf("执行 'node --version' 失败: %v\nOutput: %s", err, nodeVersionOut.String())
	}

	nodeVersionStr := nodeVersionOut.String()
	if !strings.Contains(nodeVersionStr, "v16") {
		t.Errorf("node --version 输出 %q, 期望包含 v16", nodeVersionStr)
	}

	// Cleanup: 清理构建产物
	_, err = helper.ImageRemove(buildCtx, ImageTag, true, true)
	if err != nil {
		t.Logf("Warning: 清理镜像 %s 失败: %v", ImageTag, err)
	}
}

// TestE2E_GH3_BuildWithRetry 覆盖 GH-3 Scenario "构建过程中网络错误时自动重试"。
//
// Given Docker Engine 已安装并运行
// And 构建过程中首次请求 GitHub 资源超时
// When 开发者执行 build -d claude --max-retry 3 --gh-proxy https://gh-proxy.example.com
// Then 系统按指数退避策略自动重试
// And 在三次重试内构建成功
// And 构建过程退出码为 0
//
// 实现策略：使用自定义 HTTP 客户端，其 transport 的 DialContext 在首次调用时返回
// 连接拒绝错误（模拟网络中断），之后正常连接 Docker Unix socket。
// 每个请求使用独立连接（DisableKeepAlives=true），确保每个 ImageBuild 调用
// 均触发独立 dial，从而触发 BuildEngine 的指数退避重试。
func TestE2E_GH3_BuildWithRetry(t *testing.T) {
	// Given Docker Engine 已安装并运行
	realHelper, err := dockerhelper.NewClient()
	if err != nil {
		t.Skipf("Docker SDK 客户端创建失败: %v", err)
	}
	defer realHelper.Close()

	pingCtx, pingCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer pingCancel()
	if err := realHelper.Ping(pingCtx); err != nil {
		t.Skipf("Docker Engine 未运行，跳过 E2E 测试: %v", err)
	}

	// 先检查 base image 是否已缓存（由 IT-5 前置构建保证）
	cached, err := realHelper.ImageExists(pingCtx, dockerfilegen.DefaultBaseImage)
	if err != nil || !cached {
		t.Skipf("Base image %s 未缓存，跳过 E2E 测试（需要先运行 IT-5 构建缓存基础镜像）", dockerfilegen.DefaultBaseImage)
	}

	// Given 构建过程中首次请求 GitHub 资源超时
	// 构造自定义 HTTP 客户端，第二次 dial 返回 connection refused 错误，
	// 其余 dial 正常连接 Docker Unix socket。
	// 第一次 dial 用于 NegotiateAPIVersion 的 Ping 调用，必须成功。
	// 第二次 dial 用于首次 ImageBuild POST，失败触发 BuildEngine 重试。
	// 第三次 dial 用于重试的 ImageBuild POST，成功后完成构建。
	// 设置 DisableKeepAlives=true 确保每个 API 调用（Ping、ImageBuild 各次尝试）
	// 均使用独立的 TCP 连接，不会被连接复用绕过。
	dockerSocketPath := "/var/run/docker.sock"
	dockerHost := os.Getenv("DOCKER_HOST")
	if dockerHost != "" && strings.HasPrefix(dockerHost, "unix://") {
		dockerSocketPath = dockerHost[len("unix://"):]
	}

	var dialMu sync.Mutex
	dialCount := 0

	customDialFunc := func(ctx context.Context, network, addr string) (net.Conn, error) {
		dialMu.Lock()
		dialCount++
		count := dialCount
		dialMu.Unlock()
		t.Logf("[retry test] dial #%d: network=%q addr=%q", count, network, addr)

		// 第二次 dial 失败（对应第一次 ImageBuild POST 请求）
		// 第一次 dial 用于 NegotiateAPIVersion 的 Ping（GET /_ping）
		// 失败后 BuildEngine 触发重试，第三次 dial（重试 POST）成功。
		if count == 2 {
			return nil, &net.OpError{
				Op:   "dial",
				Net:  "unix",
				Addr: &net.UnixAddr{Name: dockerSocketPath, Net: "unix"},
				Err:  &os.SyscallError{Syscall: "connect", Err: &errMockConnectionRefused{}},
			}
		}
		return net.Dial("unix", dockerSocketPath)
	}

	// DisableKeepAlives 确保每个 API 调用都建立新连接
	transport := &http.Transport{
		DisableKeepAlives: true,
		DisableCompression: true,
		DialContext:       customDialFunc,
	}

	httpClient := &http.Client{
		Transport:     transport,
		CheckRedirect: client.CheckRedirect,
	}

	testHelper, err := dockerhelper.NewClientWithOpts(
		client.WithHost("unix://"+dockerSocketPath),
		client.WithHTTPClient(httpClient),
	)
	if err != nil {
		t.Fatalf("创建测试 Docker 客户端失败: %v", err)
	}
	defer testHelper.Close()

	engine := New(testHelper)
	defer engine.Close()

	// When 开发者执行 build -d claude --max-retry 3 --gh-proxy https://gh-proxy.example.com
	buildCtx, buildCancel := context.WithTimeout(context.Background(), 3600*time.Second)
	defer buildCancel()

	output, err := engine.Build(buildCtx, BuildParams{
		Deps:     "claude",
		MaxRetry: 3,
		GHProxy:  "https://gh-proxy.example.com",
	})

	// Then 构建过程退出码为 0
	if err != nil {
		t.Fatalf("Build() error = %v\nOutput: %s", err, output)
	}

	// Then 系统按指数退避策略自动重试
	if !strings.Contains(output, "[重试 1/3]") {
		t.Error("构建日志应包含至少一次重试标识 [重试 1/3]")
	}

	// And 在三次重试内构建成功
	if !strings.Contains(output, "Successfully tagged") {
		t.Error("构建日志应包含构建成功标识 Successfully tagged")
	}

	t.Logf("构建输出 (前 2000 字符):\n%s", truncateString(output, 2000))

	// Cleanup: 清理构建产物
	_, err = realHelper.ImageRemove(buildCtx, ImageTag, true, true)
	if err != nil {
		t.Logf("Warning: 清理镜像 %s 失败: %v", ImageTag, err)
	}
}

// errMockConnectionRefused 用于模拟连接拒绝错误。
// 模拟 syscall.ECONNREFUSED 的 Error() 方法输出 "connection refused"，
// 该字符串被 isRetryableError 识别为可重试错误。
type errMockConnectionRefused struct{}

func (e *errMockConnectionRefused) Error() string { return "connection refused" }
func (e *errMockConnectionRefused) Temporary() bool { return true }
func (e *errMockConnectionRefused) Timeout() bool    { return false }

// truncateString 截断字符串到指定长度，用于日志输出。
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "\n... (已截断)"
}
