//go:build perf

package runengine

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/api/types"

	"github.com/agent-forge/cli/internal/shared/argsparser"
	"github.com/agent-forge/cli/internal/shared/dockerhelper"
)

// TestPerf_PT3_ContainerStartupTime 测量 run -a <agent> 容器启动到交互终端就绪时间。
//
// 测量内容: run -a <agent> 命令从执行到容器内交互终端可接受输入的端到端时间
// 阈值: ≤ 10 秒
// 测量方法: 通过 SDK 监听 ContainerStart 到容器进入 running 状态的时间差
// 执行次数: 每次测试执行 1 次启动，期望 CI 通过 -count=5 运行 5 次取 p95
// 可追溯性: NFR-3
func TestPerf_PT3_ContainerStartupTime(t *testing.T) {
	// ---- Given: Docker Engine 已安装并运行 ----
	helper, err := dockerhelper.NewClient()
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer helper.Close()

	pingCtx, pingCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer pingCancel()
	if err := helper.Ping(pingCtx); err != nil {
		t.Skipf("Docker Engine 未运行，跳过 PT-3 性能测试: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// 确保 agent-forge:latest 镜像存在
	perfEnsureImageExists(t, ctx, helper)

	// ---- When: 开发者执行 run -a <agent> ----
	// 使用 agent 模式运行参数，模拟 run -a claude 的交互终端配置
	params := argsparser.RunParams{
		Agent: "claude",
	}

	// 组装交互终端配置（Tty=true, OpenStdin=true）
	config, hostConfig, netConfig := AssembleContainerConfig(params, "")
	// 用 sleep 命令替代 claude（测试镜像中无 agent 二进制文件），
	// 保留完整的交互式终端配置：Tty=true, OpenStdin=true
	config.Cmd = []string{"sleep", "300"}
	hostConfig.AutoRemove = false

	// 记录开始时间
	start := time.Now()

	// 创建容器
	resp, err := helper.ContainerCreate(ctx, config, hostConfig, netConfig, nil, "")
	if err != nil {
		t.Fatalf("ContainerCreate() error = %v", err)
	}
	containerID := resp.ID
	t.Logf("容器创建成功, ID: %s", containerID)

	// 清理：测试结束后强制删除容器
	defer func() {
		if removeErr := helper.ContainerRemove(ctx, containerID, true, false); removeErr != nil {
			t.Logf("清理容器 %s 时出现非致命错误: %v", containerID, removeErr)
		}
	}()

	// 启动容器（这是关键计时点：从启动请求到容器 running）
	if err := helper.ContainerStart(ctx, containerID, types.ContainerStartOptions{}); err != nil {
		t.Fatalf("ContainerStart() error = %v", err)
	}

	// 等待容器进入 running 状态
	if err := perfWaitContainerRunning(containerID, 10*time.Second); err != nil {
		t.Fatalf("容器未能进入 running 状态: %v", err)
	}

	// 停止计时
	elapsed := time.Since(start)
	t.Logf("PT-3 容器启动耗时: %v", elapsed)

	// ---- Then: 交互终端就绪时间 ≤ 10 秒（NFR-3） ----
	threshold := 10 * time.Second
	if elapsed > threshold {
		t.Errorf("PT-3: 容器启动耗时 %v 超过阈值 %v", elapsed, threshold)
	}

	// ---- 性能结果日志 ----
	t.Logf("=== PT-3 性能结果 ===\n启动耗时: %v\n阈值: %v\n结果: %s", elapsed, threshold, perfBoolStr(elapsed <= threshold))
}

// perfEnsureImageExists 确保 agent-forge:latest 镜像存在。
//
// 如果镜像不存在，尝试拉取 centos:7 测试镜像并标记为目标镜像。
// 如果拉取失败则跳过测试。
func perfEnsureImageExists(t *testing.T, ctx context.Context, helper *dockerhelper.Client) {
	t.Helper()

	exists, err := helper.ImageExists(ctx, ImageName)
	if err != nil {
		t.Fatalf("检查镜像 %s 失败: %v", ImageName, err)
	}
	if exists {
		return
	}

	baseImage := "docker.1ms.run/centos:7"
	t.Logf("镜像 %s 不存在，尝试拉取 %s 并标记", ImageName, baseImage)

	baseExists, err := helper.ImageExists(ctx, baseImage)
	if err != nil {
		t.Fatalf("检查基础镜像 %s 失败: %v", baseImage, err)
	}

	if !baseExists {
		pullCmd := exec.CommandContext(ctx, "docker", "pull", baseImage)
		output, err := pullCmd.CombinedOutput()
		if err != nil {
			t.Skipf("无法拉取测试镜像 %s: %v\n%s", baseImage, err, output)
		}
		t.Logf("成功拉取测试镜像 %s", baseImage)
	}

	// 标记为基础镜像
	if err := helper.ImageTag(ctx, baseImage, ImageName); err != nil {
		t.Skipf("无法标记镜像 %s -> %s: %v", baseImage, ImageName, err)
	}

	// 验证标签已创建
	if verifyExists, verifyErr := helper.ImageExists(ctx, ImageName); verifyErr != nil || !verifyExists {
		t.Skipf("标记后验证失败: ImageExists(%q) = (%v, %v)", ImageName, verifyExists, verifyErr)
	}
	t.Logf("成功标记镜像 %s -> %s", baseImage, ImageName)
}

// perfWaitContainerRunning 通过 docker inspect 命令等待容器进入 running 状态。
func perfWaitContainerRunning(containerID string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		running, err := perfIsContainerRunning(containerID)
		if err != nil {
			return err
		}
		if running {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("容器 %s 在 %v 内未进入 running 状态", containerID, timeout)
}

// perfIsContainerRunning 通过 docker inspect 检查容器是否处于 running 状态。
func perfIsContainerRunning(containerID string) (bool, error) {
	cmd := exec.Command("docker", "inspect", "--format", "{{.State.Status}}", containerID)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return false, fmt.Errorf("docker inspect 失败: %w, stderr: %s", err, strings.TrimSpace(stderr.String()))
	}
	status := strings.TrimSpace(stdout.String())
	return status == "running", nil
}

// perfBoolStr 返回通过/失败文本。
func perfBoolStr(ok bool) string {
	if ok {
		return "通过"
	}
	return "失败"
}
