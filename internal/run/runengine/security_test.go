// Package runengine 提供 RunEngine 的安全测试（ST-2）。
//
// 本文件覆盖 ST-2 的所有案例，验证非特权容器默认安全模式（NFR-7）。
//
// 安全目标：
//   仅当显式传入 --docker 或 --dind 时容器以特权模式启动，
//   否则以非特权模式运行，防止容器逃逸和宿主内核权限滥用。
//
// 模拟的攻击向量：容器逃逸、宿主内核权限滥用
package runengine

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/agent-forge/cli/internal/shared/argsparser"
	"github.com/agent-forge/cli/internal/shared/dockerhelper"
)

// TestST2_DefaultRunNoPrivilege 验证默认 run 无 --docker/--dind 时容器为非特权模式。
//
// 覆盖案例：默认 run 无 --docker/--dind — Privileged=false, User 非 root
//
// 模拟的攻击向量：未指定 --docker 参数时意外启用特权模式导致容器逃逸风险
func TestST2_DefaultRunNoPrivilege(t *testing.T) {
	helper, ctx, cleanup := setupSecurityTest(t)
	defer cleanup()

	// 默认 run，无任何参数（bash 模式）
	params := argsparser.RunParams{
		// 不指定任何参数 — 模拟默认 run
	}
	config, hostConfig, netConfig := AssembleContainerConfig(params, "")
	hostConfig.AutoRemove = false

	// 使用 sleep 命令保持容器运行以便 inspect
	config.Cmd = []string{"sleep", "30"}

	resp, err := helper.ContainerCreate(ctx, config, hostConfig, netConfig, nil, "")
	if err != nil {
		t.Fatalf("ContainerCreate() error = %v", err)
	}
	defer func() {
		_ = helper.ContainerRemove(ctx, resp.ID, true, false)
	}()

	// 验证 Privileged=false（核心安全断言）
	if hostConfig.Privileged {
		t.Error("默认 run 模式: Privileged=true, want false — 违反 NFR-7")
	} else {
		t.Log("默认 run 模式: Privileged=false 验证通过")
	}

	// 验证 User 不是 root（非特权模式）
	if config.User == "root" {
		t.Error("默认 run 模式: User=root, want '' (non-root) — 违反 NFR-7")
	} else {
		t.Logf("默认 run 模式: User=%q 验证通过 (非 root)", config.User)
	}

	// 通过 docker inspect 二次确认
	inspectData := inspectContainer(t, resp.ID)
	privField := getContainerField(inspectData, "HostConfig", "Privileged")
	if isPrivileged, ok := privField.(bool); ok && isPrivileged {
		t.Error("docker inspect: HostConfig.Privileged=true, want false — 违反 NFR-7")
	} else if ok {
		t.Log("docker inspect: HostConfig.Privileged=false 确认通过")
	}

	userField := getContainerField(inspectData, "Config", "User")
	userStr, _ := userField.(string)
	if userStr == "root" {
		t.Error("docker inspect: Config.User=root, want 非 root — 违反 NFR-7")
	} else {
		t.Logf("docker inspect: Config.User=%q 验证通过", userStr)
	}
}

// TestST2_AgentModeNonPrivileged 验证 run -a claude 为非特权模式。
//
// 覆盖案例：run -a claude — 非特权模式，无额外 docker.sock 挂载
//
// 模拟的攻击向量：通过指定 agent 参数意外获得宿主 Docker 访问权限
func TestST2_AgentModeNonPrivileged(t *testing.T) {
	helper, ctx, cleanup := setupSecurityTest(t)
	defer cleanup()

	// run -a claude，不带 --docker 参数
	params := argsparser.RunParams{
		Agent: "claude",
	}
	config, hostConfig, netConfig := AssembleContainerConfig(params, "")
	hostConfig.AutoRemove = false

	// 使用 sleep 命令替代 claude 以保持容器运行
	config.Cmd = []string{"sleep", "30"}

	resp, err := helper.ContainerCreate(ctx, config, hostConfig, netConfig, nil, "")
	if err != nil {
		t.Fatalf("ContainerCreate() error = %v", err)
	}
	defer func() {
		_ = helper.ContainerRemove(ctx, resp.ID, true, false)
	}()

	// 核心断言 1: Privileged=false
	if hostConfig.Privileged {
		t.Error("agent 模式无 --docker: Privileged=true, want false — 违反 NFR-7")
	} else {
		t.Log("agent 模式无 --docker: Privileged=false 验证通过")
	}

	// 核心断言 2: User 不是 root
	if config.User == "root" {
		t.Error("agent 模式无 --docker: User=root, want '' (non-root) — 违反 NFR-7")
	} else {
		t.Logf("agent 模式无 --docker: User=%q 验证通过", config.User)
	}

	// 核心断言 3: 无 docker.sock 挂载（默认容器不应挂载 Docker socket）
	if len(hostConfig.Mounts) > 0 {
		for _, m := range hostConfig.Mounts {
			if strings.Contains(m.Source, "docker.sock") || strings.Contains(m.Target, "docker.sock") {
				t.Errorf("agent 模式无 --docker: 不应挂载 docker.sock, 发现挂载: Source=%s, Target=%s",
					m.Source, m.Target)
			}
		}
	}

	// docker inspect 确认
	inspectData := inspectContainer(t, resp.ID)
	privField := getContainerField(inspectData, "HostConfig", "Privileged")
	if isPrivileged, ok := privField.(bool); ok && isPrivileged {
		t.Error("docker inspect: HostConfig.Privileged=true, want false — 违反 NFR-7")
	}

	userField := getContainerField(inspectData, "Config", "User")
	userStr, _ := userField.(string)
	if userStr == "root" {
		t.Error("docker inspect: Config.User=root, want 非 root — 违反 NFR-7")
	}

	t.Log("agent 模式非特权验证全部通过")
}

// TestST2_DockerModePrivileged 验证 run --docker 启用特权模式。
//
// 覆盖案例：run --docker — Privileged=true, User="root", dockerd 已配置
//
// 模拟的攻击向量：验证显式指定的特权模式正确生效
func TestST2_DockerModePrivileged(t *testing.T) {
	helper, ctx, cleanup := setupSecurityTest(t)
	defer cleanup()

	params := argsparser.RunParams{
		Docker: true,
		Agent:  "", // bash 模式 + DIND
	}
	config, hostConfig, netConfig := AssembleContainerConfig(params, "")
	hostConfig.AutoRemove = false

	// 验证 Cmd 包含 dockerd 启动脚本（在替换为 sleep 之前）
	if len(config.Cmd) != 3 || config.Cmd[0] != "bash" || config.Cmd[1] != "-c" {
		t.Fatalf("DIND 模式 Cmd 配置错误, 期望 [bash -c <dockerd_start_script>], 实际: %v", config.Cmd)
	}
	cmdStr := config.Cmd[2]
	if !strings.Contains(cmdStr, "dockerd") {
		t.Error("DIND 模式 Cmd 应包含 dockerd 启动命令")
	}
	if !strings.Contains(cmdStr, "docker info") {
		t.Error("DIND 模式 Cmd 应包含 dockerd 就绪检查 (docker info)")
	}
	if !strings.Contains(cmdStr, "exec bash") {
		t.Error("DIND + bash 模式 Cmd 应在 dockerd 就绪后 exec bash")
	}
	t.Log("DIND Cmd 配置验证通过: 包含 dockerd 启动 + 就绪等待 + exec bash")

	// 核心断言 1: Privileged=true（--docker 标志启用特权）
	if !hostConfig.Privileged {
		t.Error("--docker 模式: Privileged=false, want true — 违反 NFR-7")
	} else {
		t.Log("--docker 模式: Privileged=true 验证通过")
	}

	// 核心断言 2: User=root（特权模式需 root 用户启动 dockerd）
	if config.User != "root" {
		t.Errorf("--docker 模式: User=%q, want %q — 违反 DIND 安全要求", config.User, "root")
	} else {
		t.Log("--docker 模式: User=root 验证通过")
	}

	// 将 Cmd 改为 sleep 以便创建容器并 inspect
	config.Cmd = []string{"sleep", "30"}

	resp, err := helper.ContainerCreate(ctx, config, hostConfig, netConfig, nil, "")
	if err != nil {
		t.Fatalf("ContainerCreate() error = %v", err)
	}
	defer func() {
		_ = helper.ContainerRemove(ctx, resp.ID, true, false)
	}()

	// docker inspect 确认
	inspectData := inspectContainer(t, resp.ID)
	privField := getContainerField(inspectData, "HostConfig", "Privileged")
	if isPrivileged, ok := privField.(bool); ok {
		if !isPrivileged {
			t.Error("docker inspect: HostConfig.Privileged=false, want true — 违反 NFR-7")
		} else {
			t.Log("docker inspect: HostConfig.Privileged=true 确认通过")
		}
	}

	userField := getContainerField(inspectData, "Config", "User")
	userStr, _ := userField.(string)
	if userStr != "root" {
		t.Errorf("docker inspect: Config.User=%q, want %q", userStr, "root")
	} else {
		t.Log("docker inspect: Config.User=root 确认通过")
	}

	t.Log("--docker 特权模式验证全部通过")
}

// TestST2_DindModePrivileged 验证 run --dind 启用特权模式（--dind 等价于 --docker）。
//
// 覆盖案例：run --dind — Privileged=true, User="root", dockerd 已启动
//
// 当前的实现中 --dind 与 --docker 使用相同的 Dind 布尔标志，
// 因此两者的行为必须完全一致。
func TestST2_DindModePrivileged(t *testing.T) {
	helper, ctx, cleanup := setupSecurityTest(t)
	defer cleanup()

	params := argsparser.RunParams{
		Docker: true, // --docker/--dind 使用相同标志
		Agent:  "claude",
	}
	config, hostConfig, netConfig := AssembleContainerConfig(params, "")
	hostConfig.AutoRemove = false

	// 验证 Cmd 包含 dockerd + agent 执行脚本
	if len(config.Cmd) != 3 || config.Cmd[0] != "bash" || config.Cmd[1] != "-c" {
		t.Fatalf("DIND+agent 模式 Cmd 错误: %v", config.Cmd)
	}
	cmdStr := config.Cmd[2]
	if !strings.Contains(cmdStr, "dockerd") {
		t.Error("DIND+agent 模式 Cmd 应包含 dockerd 启动")
	}
	if !strings.Contains(cmdStr, "claude") {
		t.Error("DIND+agent 模式 Cmd 应在 dockerd 就绪后执行 agent 命令")
	}
	if !strings.Contains(cmdStr, "exec") {
		t.Error("DIND+agent 模式 Cmd 应使用 exec 替换 shell")
	}
	t.Log("DIND+agent Cmd 配置验证通过")

	// 核心断言 1: Privileged=true
	if !hostConfig.Privileged {
		t.Error("--dind 模式: Privileged=false, want true — 违反 NFR-7")
	} else {
		t.Log("--dind 模式: Privileged=true 验证通过")
	}

	// 核心断言 2: User=root
	if config.User != "root" {
		t.Errorf("--dind 模式: User=%q, want %q — 违反 DIND 安全要求", config.User, "root")
	} else {
		t.Log("--dind 模式: User=root 验证通过")
	}

	// 创建容器验证上述配置在 Docker 中生效
	config.Cmd = []string{"sleep", "30"}

	resp, err := helper.ContainerCreate(ctx, config, hostConfig, netConfig, nil, "")
	if err != nil {
		t.Fatalf("ContainerCreate() error = %v", err)
	}
	defer func() {
		_ = helper.ContainerRemove(ctx, resp.ID, true, false)
	}()

	// docker inspect 二次确认
	inspectData := inspectContainer(t, resp.ID)
	privField := getContainerField(inspectData, "HostConfig", "Privileged")
	if isPrivileged, ok := privField.(bool); ok && !isPrivileged {
		t.Error("docker inspect: HostConfig.Privileged=false, want true")
	} else if ok {
		t.Log("docker inspect: HostConfig.Privileged=true 确认通过")
	}

	userField := getContainerField(inspectData, "Config", "User")
	userStr, _ := userField.(string)
	if userStr != "root" {
		t.Errorf("docker inspect: Config.User=%q, want %q", userStr, "root")
	} else {
		t.Log("docker inspect: Config.User=root 确认通过")
	}

	t.Log("--dind 特权模式验证全部通过")
}

// TestST2_ComplexParamsNoPrivilege 验证带参数但不指定 --docker 时不启用特权。
//
// 覆盖案例：未指定 --docker/--dind 时即使指定 -a + -p + -m + -e 等参数也不启用特权
//
// 模拟的攻击向量：通过传递业务参数意外激活特权模式
func TestST2_ComplexParamsNoPrivilege(t *testing.T) {
	helper, ctx, cleanup := setupSecurityTest(t)
	defer cleanup()

	// 创建临时目录模拟宿主机挂载路径
	hostMountDir := t.TempDir()

	// run -a claude -p 3000:3000 -m <tmpdir> -e KEY=VAL -w /workspace，不指定 --docker
	params := argsparser.RunParams{
		Agent:   "claude",
		Ports:   []string{"3000:3000"},
		Mounts:  []string{hostMountDir},
		Envs:    []string{"KEY=VAL"},
		Workdir: "/workspace",
		// 不指定 Docker: 关键 — 验证默认不启用特权
	}
	config, hostConfig, netConfig := AssembleContainerConfig(params, "")
	hostConfig.AutoRemove = false

	config.Cmd = []string{"sleep", "30"}

	resp, err := helper.ContainerCreate(ctx, config, hostConfig, netConfig, nil, "")
	if err != nil {
		t.Fatalf("ContainerCreate() error = %v", err)
	}
	defer func() {
		_ = helper.ContainerRemove(ctx, resp.ID, true, false)
	}()

	// 核心断言: Privileged=false（即使指定了 -a -p -m -e 等完整参数）
	if hostConfig.Privileged {
		t.Error("复杂参数无 --docker: Privileged=true, want false — 违反 NFR-7")
	} else {
		t.Log("复杂参数无 --docker: Privileged=false 验证通过")
	}

	// 核心断言: User 非 root
	if config.User == "root" {
		t.Error("复杂参数无 --docker: User=root, want '' (non-root) — 违反 NFR-7")
	} else {
		t.Logf("复杂参数无 --docker: User=%q 验证通过", config.User)
	}

	// docker inspect 确认
	inspectData := inspectContainer(t, resp.ID)
	privField := getContainerField(inspectData, "HostConfig", "Privileged")
	if isPrivileged, ok := privField.(bool); ok && isPrivileged {
		t.Error("docker inspect: HostConfig.Privileged=true, want false")
	}

	userField := getContainerField(inspectData, "Config", "User")
	userStr, _ := userField.(string)
	if userStr == "root" {
		t.Error("docker inspect: Config.User=root, want 非 root")
	}

	// 额外验证：确认业务参数（端口、挂载、环境变量、工作目录）已正确设置
	// 但不影响特权模式状态
	inspectData2 := inspectContainer(t, resp.ID)

	// 验证端口映射存在
	portBindings := getContainerField(inspectData2, "HostConfig", "PortBindings")
	if portBindings == nil {
		t.Error("PortBindings 不应为空，端口映射被忽略")
	} else {
		t.Log("端口映射存在且不影响特权模式")
	}

	// 验证环境变量存在
	envData := getContainerField(inspectData2, "Config", "Env")
	if envData == nil {
		t.Error("Env 不应为空，环境变量被忽略")
	} else {
		t.Log("环境变量存在且不影响特权模式")
	}

	// 验证工作目录
	wd := getContainerField(inspectData2, "Config", "WorkingDir")
	if wd == nil {
		t.Error("WorkingDir 不应为空，工作目录被忽略")
	} else {
		t.Log("工作目录存在且不影响特权模式")
	}

	t.Log("复杂参数无 --docker 安全验证全部通过")
}

// setupSecurityTest 创建 Docker 客户端并确保测试镜像就绪。
//
// 返回已初始化的 Docker client、带有超时的 context 和 cleanup 函数。
// 如果 Docker daemon 不可用，测试将被跳过。
func setupSecurityTest(t *testing.T) (*dockerhelper.Client, context.Context, func()) {
	t.Helper()

	helper, err := dockerhelper.NewClient()
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)

	// 检查 Docker daemon 连通性
	pingCtx, pingCancel := context.WithTimeout(ctx, 5*time.Second)
	defer pingCancel()
	if err := helper.Ping(pingCtx); err != nil {
		cancel()
		helper.Close()
		t.Skipf("Docker daemon 不可用，跳过安全测试: %v", err)
	}

	// 确保 agent-forge:latest 镜像存在（用于容器创建）
	exists, _ := helper.ImageExists(ctx, ImageName)
	if !exists {
		// 尝试拉取测试基础镜像并标记
		baseExists, _ := helper.ImageExists(ctx, testImageName)
		if !baseExists {
			// 跳过而不是失败，安全测试偏好在已就绪环境中运行
			cancel()
			helper.Close()
			t.Skipf("测试镜像 %s 不存在且无法自动准备，跳过安全测试", ImageName)
		}

		// 标记为基础镜像
		if err := helper.ImageTag(ctx, testImageName, ImageName); err != nil {
			cancel()
			helper.Close()
			t.Skipf("无法标记测试镜像 %s -> %s: %v", testImageName, ImageName, err)
		}
	}

	cleanup := func() {
		// 清理 agent-forge:latest 标签（不删除基础镜像）
		_, _ = helper.ImageRemove(ctx, ImageName, true, false)
		cancel()
		helper.Close()
	}

	return helper, ctx, cleanup
}
