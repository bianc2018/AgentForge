//go:build e2e

package buildengine

import (
	"bytes"
	"context"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/agent-forge/cli/internal/shared/dockerhelper"
	"github.com/docker/docker/api/types"
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
