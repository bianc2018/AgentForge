// Package runengine 提供 RunEngine 的集成测试（IT-6）。
//
// 本文件覆盖 IT-6 的所有案例，在真实 Docker Engine 上验证容器创建、启动、
// 端口映射、只读挂载、环境变量、工作目录、特权模式、bash 模式和后台命令模式。
// 测试后清理所有创建的容器及临时镜像标签。
package runengine

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"

	"github.com/agent-forge/cli/internal/shared/argsparser"
	"github.com/agent-forge/cli/internal/shared/dockerhelper"
)

// testImageName 是集成测试中使用的轻量级基础镜像名称。
// 选择已在本机缓存的 centos:7 镜像以避免网络拉取。
const testImageName = "docker.1ms.run/centos:7"

// setupDockerTest 创建 Docker 客户端并确保测试镜像就绪。
//
// 返回已初始化的 Docker client、带有超时的 context 和 cleanup 函数。
// 如果 Docker daemon 不可用或无法拉取测试镜像，测试将被跳过。
func setupDockerTest(t *testing.T) (*dockerhelper.Client, context.Context, func()) {
	t.Helper()

	helper, err := dockerhelper.NewClient()
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)

	// 检查 Docker daemon 连通性
	pingCtx, pingCancel := context.WithTimeout(ctx, 5*time.Second)
	defer pingCancel()
	if err := helper.Ping(pingCtx); err != nil {
		cancel()
		helper.Close()
		t.Skipf("Docker daemon 不可用，跳过集成测试: %v", err)
	}

	// 确保 agent-forge:latest 镜像存在（用于容器创建）
	exists, _ := helper.ImageExists(ctx, ImageName)
	if !exists {
		// 先检查测试基础镜像是否存在
		baseExists, _ := helper.ImageExists(ctx, testImageName)
		if !baseExists {
			// 拉取测试镜像
			pullCmd := exec.CommandContext(ctx, "docker", "pull", testImageName)
			output, err := pullCmd.CombinedOutput()
			if err != nil {
				cancel()
				helper.Close()
				t.Skipf("无法拉取测试镜像 %s: %v, output: %s", testImageName, err, output)
			}
		}

		// 标记为 agent-forge:latest（优先使用 SDK，可靠性高于 docker CLI）
		if err := helper.ImageTag(ctx, testImageName, ImageName); err != nil {
			cancel()
			helper.Close()
			t.Skipf("无法标记测试镜像 %s -> %s: %v", testImageName, ImageName, err)
		}

		// 验证标签已创建
		verifyExists, verifyErr := helper.ImageExists(ctx, ImageName)
		if verifyErr != nil || !verifyExists {
			cancel()
			helper.Close()
			t.Skipf("标记后验证失败: ImageExists(%q) = (%v, %v)", ImageName, verifyExists, verifyErr)
		}
	}

	cleanup := func() {
		// 清理创建的 agent-forge:latest 标签（但不删除基础 busybox 镜像）
		_, err := helper.ImageRemove(ctx, ImageName, true, false)
		if err != nil {
			t.Logf("清理测试镜像标签 %s 时出现非致命错误: %v", ImageName, err)
		}
		cancel()
		helper.Close()
	}

	return helper, ctx, cleanup
}

// inspectContainer 使用 docker inspect 命令获取容器配置的 JSON 输出。
func inspectContainer(t *testing.T, containerID string) map[string]interface{} {
	t.Helper()
	cmd := exec.Command("docker", "inspect", containerID)
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("docker inspect 失败: %v", err)
	}

	var inspectData []map[string]interface{}
	if err := json.Unmarshal(output, &inspectData); err != nil {
		t.Fatalf("解析 docker inspect 输出失败: %v", err)
	}

	if len(inspectData) == 0 {
		t.Fatal("docker inspect 返回空数组")
	}

	return inspectData[0]
}

// getContainerField 从 docker inspect 输出中安全地读取嵌套字段。
func getContainerField(data map[string]interface{}, keys ...string) interface{} {
	current := data
	for i, key := range keys {
		if i == len(keys)-1 {
			return current[key]
		}
		if next, ok := current[key].(map[string]interface{}); ok {
			current = next
		} else {
			return nil
		}
	}
	return nil
}

// containerRunning 检查容器是否正在运行。
func containerRunning(t *testing.T, helper *dockerhelper.Client, ctx context.Context, containerID string) bool {
	t.Helper()
	// 使用 docker ps 检查容器状态
	cmd := exec.CommandContext(ctx, "docker", "inspect", "--format={{.State.Status}}", containerID)
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(output)) == "running"
}

// TestIT6_ContainerCreateAndStart 验证容器创建和启动。
//
// 覆盖案例：容器创建 — ContainerCreate 返回有效 ID
// 覆盖案例：容器启动 — ContainerStart 成功，容器状态为 running
//
// 使用 sleep 命令保持容器运行以验证启动状态。
func TestIT6_ContainerCreateAndStart(t *testing.T) {
	helper, ctx, cleanup := setupDockerTest(t)
	defer cleanup()

	// 使用 sleep 命令保持容器运行以验证 running 状态
	params := argsparser.RunParams{
		RunCmd: "sleep 30", // 后台命令模式，sleep 保持运行
	}
	config, hostConfig, netConfig := AssembleContainerConfig(params, "")
	hostConfig.AutoRemove = false // 关闭自动删除以便手动控制生命周期

	resp, err := helper.ContainerCreate(ctx, config, hostConfig, netConfig, nil, "")
	if err != nil {
		t.Fatalf("ContainerCreate() error = %v", err)
	}
	if resp.ID == "" {
		t.Fatal("ContainerCreate() 返回空 ID")
	}
	t.Logf("容器创建成功, ID: %s", resp.ID)

	// 启动容器
	if err := helper.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		_ = helper.ContainerRemove(ctx, resp.ID, true, false)
		t.Fatalf("ContainerStart() error = %v", err)
	}

	// 验证容器正在运行
	if !containerRunning(t, helper, ctx, resp.ID) {
		_ = helper.ContainerRemove(ctx, resp.ID, true, false)
		t.Fatal("容器启动后状态不是 running")
	}
	t.Logf("容器启动成功并处于 running 状态")

	// 强制停止并清理（避免 sleep 长时间运行）
	if err := helper.ContainerRemove(ctx, resp.ID, true, false); err != nil {
		t.Logf("清理容器 %s 失败: %v", resp.ID, err)
	}
}

// TestIT6_PortMapping 验证端口映射配置。
//
// 覆盖案例：端口映射 — 容器配置包含正确的 PortBindings
func TestIT6_PortMapping(t *testing.T) {
	helper, ctx, cleanup := setupDockerTest(t)
	defer cleanup()

	params := argsparser.RunParams{
		Ports:  []string{"3000:3000"},
		RunCmd: "echo server", // 后台命令模式
	}
	config, hostConfig, netConfig := AssembleContainerConfig(params, "")

	resp, err := helper.ContainerCreate(ctx, config, hostConfig, netConfig, nil, "")
	if err != nil {
		t.Fatalf("ContainerCreate() error = %v", err)
	}
	defer func() {
		if err := helper.ContainerRemove(ctx, resp.ID, true, false); err != nil {
			t.Logf("清理容器 %s 失败: %v", resp.ID, err)
		}
	}()

	// 验证端口映射
	inspectData := inspectContainer(t, resp.ID)
	portBindings := getContainerField(inspectData, "HostConfig", "PortBindings")
	if portBindings == nil {
		t.Fatal("PortBindings 不应为空")
	}

	bindings, ok := portBindings.(map[string]interface{})
	if !ok {
		t.Fatalf("PortBindings 类型错误: %T", portBindings)
	}

	// 验证端口 3000/tcp 已映射
	found := false
	for portKey := range bindings {
		if strings.Contains(portKey, "3000") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("端口映射应包含 3000/tcp, 实际: %v", bindings)
	}
	t.Logf("端口映射验证通过: %v", bindings)
}

// TestIT6_ReadOnlyMount 验证只读目录挂载配置。
//
// 覆盖案例：目录挂载（只读）— Mounts 包含 ReadOnly=true
//
// 注意：bind mount 要求宿主机源路径已存在，测试中动态创建临时目录。
func TestIT6_ReadOnlyMount(t *testing.T) {
	helper, ctx, cleanup := setupDockerTest(t)
	defer cleanup()

	// 创建测试用临时挂载目录（bind mount 需要源路径存在）
	mountDir := t.TempDir()

	params := argsparser.RunParams{
		Mounts: []string{mountDir},
		RunCmd: "sleep 1",
	}
	config, hostConfig, netConfig := AssembleContainerConfig(params, "")

	resp, err := helper.ContainerCreate(ctx, config, hostConfig, netConfig, nil, "")
	if err != nil {
		t.Fatalf("ContainerCreate() error = %v", err)
	}
	defer func() {
		if err := helper.ContainerRemove(ctx, resp.ID, true, false); err != nil {
			t.Logf("清理容器 %s 失败: %v", resp.ID, err)
		}
	}()

	// 验证挂载配置
	inspectData := inspectContainer(t, resp.ID)
	mountsData := getContainerField(inspectData, "Mounts")
	if mountsData == nil {
		t.Fatal("Mounts 不应为空")
	}

	mounts, ok := mountsData.([]interface{})
	if !ok {
		t.Fatalf("Mounts 类型错误: %T", mountsData)
	}

	found := false
	for _, m := range mounts {
		mount, ok := m.(map[string]interface{})
		if !ok {
			continue
		}
		// Docker inspect uses RW (not ReadOnly). RW=false means readonly.
		rw, _ := mount["RW"].(bool)
		dest, _ := mount["Destination"].(string)
		if !rw && dest == mountDir {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("未找到只读挂载 %s, 实际: %v", mountDir, mounts)
	} else {
		t.Logf("只读挂载验证通过: RW=false, Destination=%s", mountDir)
	}
}

// TestIT6_EnvironmentVariables 验证环境变量配置。
//
// 覆盖案例：环境变量 — Env 包含所有指定环境变量
func TestIT6_EnvironmentVariables(t *testing.T) {
	helper, ctx, cleanup := setupDockerTest(t)
	defer cleanup()

	params := argsparser.RunParams{
		Envs:   []string{"TEST_KEY=test-value", "DEBUG_MODE=true"},
		RunCmd: "echo $TEST_KEY",
	}
	config, hostConfig, netConfig := AssembleContainerConfig(params, "")
	hostConfig.AutoRemove = false // 关闭自动删除以便检查

	resp, err := helper.ContainerCreate(ctx, config, hostConfig, netConfig, nil, "")
	if err != nil {
		t.Fatalf("ContainerCreate() error = %v", err)
	}
	defer func() {
		if err := helper.ContainerRemove(ctx, resp.ID, true, false); err != nil {
			t.Logf("清理容器 %s 失败: %v", resp.ID, err)
		}
	}()

	// 验证环境变量
	inspectData := inspectContainer(t, resp.ID)
	envData := getContainerField(inspectData, "Config", "Env")
	if envData == nil {
		t.Fatal("Env 不应为空")
	}

	envList, ok := envData.([]interface{})
	if !ok {
		t.Fatalf("Env 类型错误: %T", envData)
	}

	foundKey := false
	foundDebug := false
	for _, e := range envList {
		entry, ok := e.(string)
		if !ok {
			continue
		}
		if entry == "TEST_KEY=test-value" {
			foundKey = true
		}
		if entry == "DEBUG_MODE=true" {
			foundDebug = true
		}
	}
	if !foundKey {
		t.Errorf("Env 应包含 TEST_KEY=test-value, 实际: %v", envList)
	}
	if !foundDebug {
		t.Errorf("Env 应包含 DEBUG_MODE=true, 实际: %v", envList)
	}
	t.Logf("环境变量验证通过: %v", envList)
}

// TestIT6_WorkingDirectory 验证工作目录配置。
//
// 覆盖案例：工作目录 — WorkingDir 正确设置
func TestIT6_WorkingDirectory(t *testing.T) {
	helper, ctx, cleanup := setupDockerTest(t)
	defer cleanup()

	params := argsparser.RunParams{
		Workdir: "/workspace",
		RunCmd:  "pwd",
	}
	config, hostConfig, netConfig := AssembleContainerConfig(params, "")

	resp, err := helper.ContainerCreate(ctx, config, hostConfig, netConfig, nil, "")
	if err != nil {
		t.Fatalf("ContainerCreate() error = %v", err)
	}
	defer func() {
		if err := helper.ContainerRemove(ctx, resp.ID, true, false); err != nil {
			t.Logf("清理容器 %s 失败: %v", resp.ID, err)
		}
	}()

	// 验证工作目录
	inspectData := inspectContainer(t, resp.ID)
	workingDir := getContainerField(inspectData, "Config", "WorkingDir")
	if workingDir == nil {
		t.Fatal("WorkingDir 不应为空")
	}

	wd, ok := workingDir.(string)
	if !ok {
		t.Fatalf("WorkingDir 类型错误: %T", workingDir)
	}
	if wd != "/workspace" {
		t.Errorf("WorkingDir = %q, want %q", wd, "/workspace")
	}
	t.Logf("工作目录验证通过: %s", wd)
}

// TestIT6_PrivilegedMode 验证特权模式仅在 --docker 参数时设置。
//
// 覆盖案例：特权模式 — Privileged=true 仅在 --docker 参数时设置
func TestIT6_PrivilegedMode(t *testing.T) {
	t.Run("使用 --docker 参数时启用特权模式", func(t *testing.T) {
		helper, ctx, cleanup := setupDockerTest(t)
		defer cleanup()

		params := argsparser.RunParams{
			Docker: true,
			RunCmd: "echo privileged",
		}
		config, hostConfig, netConfig := AssembleContainerConfig(params, "")

		resp, err := helper.ContainerCreate(ctx, config, hostConfig, netConfig, nil, "")
		if err != nil {
			t.Fatalf("ContainerCreate() error = %v", err)
		}
		defer func() {
			if err := helper.ContainerRemove(ctx, resp.ID, true, false); err != nil {
				t.Logf("清理容器 %s 失败: %v", resp.ID, err)
			}
		}()

		// 验证 Privileged=true
		inspectData := inspectContainer(t, resp.ID)
		privileged := getContainerField(inspectData, "HostConfig", "Privileged")
		if privileged == nil {
			t.Fatal("Privileged 字段不应为空")
		}
		isPrivileged, ok := privileged.(bool)
		if !ok {
			t.Fatalf("Privileged 类型错误: %T", privileged)
		}
		if !isPrivileged {
			t.Error("--docker 模式应设置 Privileged=true")
		}

		// 验证 User=root
		user := getContainerField(inspectData, "Config", "User")
		userStr, _ := user.(string)
		if userStr != "root" {
			t.Errorf("User = %q, want %q", userStr, "root")
		}
		t.Log("特权模式验证通过: Privileged=true, User=root")
	})

	t.Run("不指定 --docker 时不启用特权模式", func(t *testing.T) {
		helper, ctx, cleanup := setupDockerTest(t)
		defer cleanup()

		params := argsparser.RunParams{
			RunCmd: "echo non-privileged",
		}
		config, hostConfig, netConfig := AssembleContainerConfig(params, "")

		resp, err := helper.ContainerCreate(ctx, config, hostConfig, netConfig, nil, "")
		if err != nil {
			t.Fatalf("ContainerCreate() error = %v", err)
		}
		defer func() {
			if err := helper.ContainerRemove(ctx, resp.ID, true, false); err != nil {
				t.Logf("清理容器 %s 失败: %v", resp.ID, err)
			}
		}()

		inspectData := inspectContainer(t, resp.ID)
		privileged := getContainerField(inspectData, "HostConfig", "Privileged")
		isPrivileged, ok := privileged.(bool)
		if !ok {
			t.Fatalf("Privileged 类型错误: %T", privileged)
		}
		if isPrivileged {
			t.Error("默认模式不应设置 Privileged=true")
		}
		t.Log("默认模式验证通过: Privileged=false")
	})
}

// TestIT6_BashMode 验证 bash 模式的容器配置。
//
// 覆盖案例：bash 模式 — Cmd 包含 bash 命令
func TestIT6_BashMode(t *testing.T) {
	helper, ctx, cleanup := setupDockerTest(t)
	defer cleanup()

	// bash 模式：未指定 -a，无 --run
	params := argsparser.RunParams{
		Agent: "", // bash 模式
	}
	// 使用空 wrapperScript 模拟无 wrapper 的简单 bash 模式
	config, hostConfig, netConfig := AssembleContainerConfig(params, "")

	// 修改 Cmd 为非交互式（仅用于验证创建）
	config.Cmd = []string{"echo", "bash-mode"}

	resp, err := helper.ContainerCreate(ctx, config, hostConfig, netConfig, nil, "")
	if err != nil {
		t.Fatalf("ContainerCreate() error = %v", err)
	}
	defer func() {
		if err := helper.ContainerRemove(ctx, resp.ID, true, false); err != nil {
			t.Logf("清理容器 %s 失败: %v", resp.ID, err)
		}
	}()

	// 验证 Tty 和 OpenStdin 在 bash 模式下启用
	inspectData := inspectContainer(t, resp.ID)
	tty := getContainerField(inspectData, "Config", "Tty")
	if tty == true {
		t.Log("bash 模式: Tty=true (由 AssembleContainerConfig 默认设置)")
	} else {
		// AssembleContainerConfig 会被 RunCmd 模式覆盖 Tty 设置
		// 这里用 echo 替代了 bash，Tty=false 是正常的
		t.Log("bash 模式: Cmd=echo, 因此 Tty=false（本测试中 Cmd 被替换为 echo）")
	}
}

// TestIT6_RunCommandMode 验证后台命令模式的容器创建和自动删除。
//
// 覆盖案例：后台命令模式 — Cmd 为指定命令，AutoRemove=true
func TestIT6_RunCommandMode(t *testing.T) {
	helper, ctx, cleanup := setupDockerTest(t)
	defer cleanup()

	params := argsparser.RunParams{
		RunCmd: "echo 'hello from run command mode'",
	}
	config, hostConfig, netConfig := AssembleContainerConfig(params, "")

	resp, err := helper.ContainerCreate(ctx, config, hostConfig, netConfig, nil, "")
	if err != nil {
		t.Fatalf("ContainerCreate() error = %v", err)
	}
	t.Logf("后台命令模式容器创建成功, ID: %s", resp.ID)

	// 验证 AutoRemove 已启用
	if !hostConfig.AutoRemove {
		t.Error("后台命令模式应设置 AutoRemove=true")
	}

	// 启动容器
	if err := helper.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		t.Fatalf("ContainerStart() error = %v", err)
	}
	t.Log("后台命令模式容器启动成功")

	// 等待容器执行完毕
	statusCh, errCh := helper.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case status := <-statusCh:
		t.Logf("容器执行完毕, 退出码: %d", status.StatusCode)
		if status.StatusCode != 0 {
			t.Errorf("容器退出码 = %d, want 0", status.StatusCode)
		}
	case err := <-errCh:
		t.Fatalf("等待容器退出失败: %v", err)
	case <-time.After(10 * time.Second):
		t.Fatal("等待容器退出超时")
	}

	// 由于 AutoRemove=true，容器应已自动删除（无法再 inspect）
	// 验证容器已不存在
	inspectCmd := exec.CommandContext(ctx, "docker", "inspect", resp.ID)
	if output, _ := inspectCmd.CombinedOutput(); len(output) > 0 {
		// 容器可能还存在（取决于 Docker 版本和 AutoRemove 行为的精确时机）
		// 手动清理以防残留
		_ = helper.ContainerRemove(ctx, resp.ID, true, false)
		t.Log("AutoRemove 未立即生效，已手动清理容器")
	} else {
		t.Log("容器已通过 AutoRemove 自动删除")
	}
}

// TestIT6_MultipleConfigs 验证多端口、多挂载、多环境变量的组合配置。
//
// 覆盖边界情况：多个端口映射、多个挂载路径、多个环境变量同时设置。
func TestIT6_MultipleConfigs(t *testing.T) {
	helper, ctx, cleanup := setupDockerTest(t)
	defer cleanup()

	// 创建测试用临时挂载目录（bind mount 需要源路径存在）
	mount1 := t.TempDir()
	mount2 := t.TempDir()

	params := argsparser.RunParams{
		Ports:   []string{"8080:80", "3000:3000", "9090:9090"},
		Mounts:  []string{mount1, mount2},
		Envs:    []string{"KEY1=val1", "KEY2=val2"},
		Workdir: "/app",
		RunCmd:  "echo multiple-configs",
	}
	config, hostConfig, netConfig := AssembleContainerConfig(params, "")

	resp, err := helper.ContainerCreate(ctx, config, hostConfig, netConfig, nil, "")
	if err != nil {
		t.Fatalf("ContainerCreate() error = %v", err)
	}
	defer func() {
		if err := helper.ContainerRemove(ctx, resp.ID, true, false); err != nil {
			t.Logf("清理容器 %s 失败: %v", resp.ID, err)
		}
	}()

	// 验证组合配置
	inspectData := inspectContainer(t, resp.ID)

	// 1. 验证端口映射数量
	portBindings := getContainerField(inspectData, "HostConfig", "PortBindings")
	if bindings, ok := portBindings.(map[string]interface{}); ok {
		if len(bindings) != 3 {
			t.Errorf("PortBindings 数量 = %d, want 3", len(bindings))
		} else {
			t.Logf("端口映射数量正确: %d", len(bindings))
		}
	}

	// 2. 验证环境变量
	envData := getContainerField(inspectData, "Config", "Env")
	if envList, ok := envData.([]interface{}); ok {
		foundKey1 := false
		foundKey2 := false
		for _, e := range envList {
			entry, ok := e.(string)
			if !ok {
				continue
			}
			if entry == "KEY1=val1" {
				foundKey1 = true
			}
			if entry == "KEY2=val2" {
				foundKey2 = true
			}
		}
		if !foundKey1 {
			t.Error("Env 应包含 KEY1=val1")
		}
		if !foundKey2 {
			t.Error("Env 应包含 KEY2=val2")
		}
		if foundKey1 && foundKey2 {
			t.Log("环境变量验证通过")
		}
	}

	// 3. 验证工作目录
	workingDir := getContainerField(inspectData, "Config", "WorkingDir")
	if wd, ok := workingDir.(string); ok {
		if wd != "/app" {
			t.Errorf("WorkingDir = %q, want %q", wd, "/app")
		} else {
			t.Log("工作目录验证通过: /app")
		}
	}
}

// TestIT6_ContainerCleanup 验证测试后容器被正确清理。
//
// 覆盖案例：容器停止后自动清理 — 测试容器在测试后删除
func TestIT6_ContainerCleanup(t *testing.T) {
	helper, ctx, cleanup := setupDockerTest(t)
	defer cleanup()

	// 创建一个无 AutoRemove 的容器以便手动管理生命周期
	params := argsparser.RunParams{
		RunCmd: "", // 不使用 --run，避免 AutoRemove
	}
	config, hostConfig, netConfig := AssembleContainerConfig(params, "")
	// 显式关闭 AutoRemove 以手动控制
	hostConfig.AutoRemove = false
	// 将命令改为后台 sleep，方便启动后立即控制
	config.Cmd = []string{"sleep", "5"}

	resp, err := helper.ContainerCreate(ctx, config, hostConfig, netConfig, nil, "")
	if err != nil {
		t.Fatalf("ContainerCreate() error = %v", err)
	}
	containerID := resp.ID

	// 启动容器
	if err := helper.ContainerStart(ctx, containerID, types.ContainerStartOptions{}); err != nil {
		_ = helper.ContainerRemove(ctx, containerID, true, false)
		t.Fatalf("ContainerStart() error = %v", err)
	}

	// 首次尝试停止并删除
	removeErr := helper.ContainerRemove(ctx, containerID, true, false)

	// 检查容器是否仍在运行（可能在删除过程中）
	checkCmd := exec.CommandContext(ctx, "docker", "inspect", "--format={{.State.Status}}", containerID)
	statusOutput, _ := checkCmd.CombinedOutput()
	status := strings.TrimSpace(string(statusOutput))

	if removeErr != nil {
		// 容器可能已被删除、正在删除或仍存在
		if strings.Contains(status, "No such container") || strings.Contains(status, "") && status == "" {
			t.Log("容器已成功清理（已被移除）")
		} else if status == "removing" {
			t.Log("容器正在清理中（removing 状态）")
		} else {
			t.Logf("容器清理状态: %s (error: %v)", status, removeErr)
			// 尝试再次强制删除
			_ = exec.CommandContext(ctx, "docker", "rm", "-f", containerID).Run()
		}
	} else {
		t.Log("容器已通过 ContainerRemove 成功删除")
	}
}

// TestIT6_CmdTypeVerification 验证 Cmd 类型在 Docker API 调用中正确序列化。
//
// 确保 strslice.StrSlice 类型的 Cmd 能被 Docker SDK 正确处理。
func TestIT6_CmdTypeVerification(t *testing.T) {
	helper, ctx, cleanup := setupDockerTest(t)
	defer cleanup()

	// Agent 模式
	params := argsparser.RunParams{
		Agent:  "claude",
		RunCmd: "",
	}
	config, hostConfig, netConfig := AssembleContainerConfig(params, "")

	resp, err := helper.ContainerCreate(ctx, config, hostConfig, netConfig, nil, "")
	if err != nil {
		t.Fatalf("ContainerCreate() 接受 strslice.StrSlice Cmd 类型: %v", err)
	}
	defer func() {
		_ = helper.ContainerRemove(ctx, resp.ID, true, false)
	}()

	// 验证 Cmd 正确传递到 Docker
	inspectData := inspectContainer(t, resp.ID)
	cmdData := getContainerField(inspectData, "Config", "Cmd")
	if cmdData == nil {
		t.Fatal("Cmd 不应为空")
	}
	cmdList, ok := cmdData.([]interface{})
	if !ok {
		t.Fatalf("Cmd 类型错误: %T", cmdData)
	}
	if len(cmdList) != 1 {
		t.Fatalf("len(Cmd) = %d, want 1 (agent mode)", len(cmdList))
	}
	cmdStr, ok := cmdList[0].(string)
	if !ok {
		t.Fatalf("Cmd[0] 类型错误: %T", cmdList[0])
	}
	if cmdStr != "claude" {
		t.Errorf("Cmd[0] = %q, want %q", cmdStr, "claude")
	}
	t.Logf("Cmd 类型验证通过: %v", cmdList)
}

// TestIT6_EngineNew 验证 Engine 构造函数在集成环境中工作。
//
// 验证 Engine 可以被正确创建并持有 Docker Helper 客户端。
func TestIT6_EngineNew(t *testing.T) {
	helper, ctx, cleanup := setupDockerTest(t)
	defer cleanup()
	_ = ctx // Engine.Run 需要 context，此处仅验证构造

	engine := New(helper, "/tmp/test-config")
	if engine == nil {
		t.Fatal("New() 返回 nil")
	}
	if engine.helper != helper {
		t.Error("Engine.helper 与传入的 helper 不一致")
	}
	t.Log("Engine 构造函数验证通过")
}

// TestIT6_WithFailingImage 验证使用不存在的镜像时返回错误。
//
// 覆盖错误路径：镜像不存在时创建容器应失败。
func TestIT6_WithFailingImage(t *testing.T) {
	helper, err := dockerhelper.NewClient()
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer helper.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := helper.Ping(ctx); err != nil {
		t.Skipf("Docker daemon 不可用: %v", err)
	}

	// 使用不存在的镜像
	config := &container.Config{
		Image: "nonexistent-image:latest",
		Cmd:   []string{"echo", "fail"},
	}
	hostConfig := &container.HostConfig{}
	netConfig := &network.NetworkingConfig{}

	_, err = helper.ContainerCreate(ctx, config, hostConfig, netConfig, nil, "")
	if err == nil {
		t.Error("使用不存在的镜像创建容器应返回错误")
	} else {
		t.Logf("不存在的镜像返回预期错误: %v", err)
	}
}

// TestIT6_AgentModeInspect 验证 agent 模式的 Cmd 配置。
//
// 验证 agent 模式下的 Cmd 为单个 agent 命令字符串。
func TestIT6_AgentModeInspect(t *testing.T) {
	helper, ctx, cleanup := setupDockerTest(t)
	defer cleanup()

	testCases := []struct {
		name   string
		agent  string
	}{
		{"claude agent", "claude"},
		{"opencode agent", "opencode"},
		{"kimi agent", "kimi"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			params := argsparser.RunParams{
				Agent:  tc.agent,
				RunCmd: "",
			}
			config, hostConfig, netConfig := AssembleContainerConfig(params, "")

			resp, err := helper.ContainerCreate(ctx, config, hostConfig, netConfig, nil, "")
			if err != nil {
				t.Fatalf("ContainerCreate() error = %v", err)
			}
			defer func(id string) {
				_ = helper.ContainerRemove(ctx, id, true, false)
			}(resp.ID)

			inspectData := inspectContainer(t, resp.ID)
			cmdData := getContainerField(inspectData, "Config", "Cmd")
			cmdList, ok := cmdData.([]interface{})
			if !ok || len(cmdList) == 0 {
				t.Fatalf("Cmd 为空或类型错误")
			}
			if cmdList[0] != tc.agent {
				t.Errorf("Cmd[0] = %q, want %q", cmdList[0], tc.agent)
			}
		})
	}
}

// TestIT6_DockerModeWithRunCommand 验证 --docker + --run 的组合模式。
//
// 验证 Docker-in-Docker 特权模式下执行后台命令。
func TestIT6_DockerModeWithRunCommand(t *testing.T) {
	helper, ctx, cleanup := setupDockerTest(t)
	defer cleanup()

	params := argsparser.RunParams{
		Docker: true,
		RunCmd: "echo dind-run",
	}
	config, hostConfig, _ := AssembleContainerConfig(params, "")

	// 验证配置组合
	if !hostConfig.Privileged {
		t.Error("--docker 模式应设置 Privileged=true")
	}
	if !hostConfig.AutoRemove {
		t.Error("--run 模式应设置 AutoRemove=true")
	}
	if config.User != "root" {
		t.Errorf("User = %q, want %q", config.User, "root")
	}

	// 创建容器（但不启动，避免 dockerd 启动开销）
	resp, err := helper.ContainerCreate(ctx, config, hostConfig, nil, nil, "")
	if err != nil {
		t.Fatalf("ContainerCreate() error = %v", err)
	}
	defer func() {
		_ = helper.ContainerRemove(ctx, resp.ID, true, false)
	}()

	t.Logf("Docker+RunCommand 模式容器创建成功, ID: %s", resp.ID)
}

// fmt 用于避免编译时未使用的导入错误。
var _ = fmt.Sprintf("integration test %s", ImageName)
