//go:build perf

package buildengine

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/agent-forge/cli/internal/build/dockerfilegen"
	"github.com/agent-forge/cli/internal/shared/dockerhelper"
	"github.com/docker/docker/api/types"
)

// TestPerf_PT1_BuildTimeAllDeps 测量 build -d all --max-retry 3 的端到端构建时间。
//
// 测量内容: build -d all --max-retry 3 命令从执行到退出码返回 0 的端到端耗时
// 阈值: ≤ 15 分钟（基础镜像已缓存 + 国内镜像源可用）
// 测量方法: 使用 time 包裹 Build 调用，记录 wall clock 时间
// 执行次数: 每次测试执行 1 次构建，期望 CI 通过 -count=3 运行 3 次取最大值
// 可追溯性: NFR-1
func TestPerf_PT1_BuildTimeAllDeps(t *testing.T) {
	// ---- Given: Docker Engine 已安装并运行 ----
	helper, err := dockerhelper.NewClient()
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer helper.Close()

	pingCtx, pingCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer pingCancel()
	if err := helper.Ping(pingCtx); err != nil {
		t.Skipf("Docker Engine 未运行，跳过 PT-1 性能测试: %v", err)
	}

	// 基础镜像必须已缓存（由前置构建步骤保证）
	cached, err := helper.ImageExists(pingCtx, dockerfilegen.DefaultBaseImage)
	if err != nil || !cached {
		t.Skipf("基础镜像 %s 未缓存，跳过 PT-1 性能测试（需先构建缓存基础镜像）", dockerfilegen.DefaultBaseImage)
	}

	engine := New(helper)
	defer engine.Close()

	// ---- When: 开发者执行 build -d all --max-retry 3 ----
	// 超时设为 20 分钟（阈值 15 分钟 + 缓冲）
	buildCtx, buildCancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer buildCancel()

	start := time.Now()
	output, err := engine.Build(buildCtx, BuildParams{
		Deps:     "all",
		MaxRetry: 3,
		GHProxy:  "https://gh-proxy.com/",
	})
	elapsed := time.Since(start)

	t.Logf("PT-1 构建耗时: %v", elapsed)

	// Then: 构建过程退出码为 0
	if err != nil {
		t.Logf("=== 完整构建输出 (len=%d, 截断至 2000) ===\n%s\n=== END ===", len(output), truncateString(output, 2000))
		t.Fatalf("Build() error = %v", err)
	}

	// Then: 构建耗时 ≤ 15 分钟（NFR-1）
	threshold := 15 * time.Minute
	if elapsed > threshold {
		t.Errorf("PT-1: 构建耗时 %v 超过阈值 %v", elapsed, threshold)
	}

	// ---- Cleanup: 清理构建产物 ----
	_, err = helper.ImageRemove(buildCtx, ImageTag, true, true)
	if err != nil {
		t.Logf("Warning: 清理镜像 %s 失败: %v", ImageTag, err)
	}

	// ---- 性能结果日志 ----
	t.Logf("=== PT-1 性能结果 ===\n构建耗时: %v\n阈值: %v\n结果: %s", elapsed, threshold, boolStr(elapsed <= threshold))
}

// boolStr 返回通过/失败文本。
func boolStr(ok bool) string {
	if ok {
		return "通过"
	}
	return "失败"
}

// TestPerf_PT2_ImageSizeRatio 比较 build -d all 和 build -d mini 的镜像体积比。
//
// 测量内容: build -d mini 构建的镜像体积与 build -d all 构建的镜像体积之比
// 阈值: mini 体积 < all 体积的 60%
// 测量方法: 通过 SDK ImageList 获取镜像大小（Size 字段），计算比例
// 执行次数: 分别构建 all 和 mini 各 1 次，验证比例
// 可追溯性: NFR-2
func TestPerf_PT2_ImageSizeRatio(t *testing.T) {
	// ---- Given: Docker Engine 已安装并运行 ----
	helper, err := dockerhelper.NewClient()
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer helper.Close()

	pingCtx, pingCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer pingCancel()
	if err := helper.Ping(pingCtx); err != nil {
		t.Skipf("Docker Engine 未运行，跳过 PT-2 性能测试: %v", err)
	}

	// 基础镜像必须已缓存
	cached, err := helper.ImageExists(pingCtx, dockerfilegen.DefaultBaseImage)
	if err != nil || !cached {
		t.Skipf("基础镜像 %s 未缓存，跳过 PT-2 性能测试", dockerfilegen.DefaultBaseImage)
	}

	engine := New(helper)
	defer engine.Close()

	// ---- Step 1: 构建 -d mini 镜像 ----
	t.Log("PT-2 Step 1/2: 构建 -d mini 镜像...")

	buildCtx, buildCancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer buildCancel()

	output, err := engine.Build(buildCtx, BuildParams{
		Deps:     "mini",
		MaxRetry: 3,
		GHProxy:  "https://gh-proxy.com/",
	})
	if err != nil {
		t.Fatalf("mini 构建失败: %v\nOutput (前 1000 字符): %s", err, truncateString(output, 1000))
	}

	// 获取 mini 镜像体积
	miniSize, err := getImageSize(helper, buildCtx, ImageTag)
	if err != nil {
		t.Fatalf("获取 mini 镜像大小失败: %v", err)
	}
	t.Logf("mini 镜像大小: %d bytes (%.2f MB)", miniSize, float64(miniSize)/(1024*1024))

	// 清理 mini 镜像，为 all 构建腾出标签
	_, err = helper.ImageRemove(buildCtx, ImageTag, true, true)
	if err != nil {
		t.Fatalf("清理 mini 镜像失败: %v", err)
	}

	// ---- Step 2: 构建 -d all 镜像 ----
	t.Log("PT-2 Step 2/2: 构建 -d all 镜像...")

	output, err = engine.Build(buildCtx, BuildParams{
		Deps:     "all",
		MaxRetry: 3,
		GHProxy:  "https://gh-proxy.com/",
	})
	if err != nil {
		t.Fatalf("all 构建失败: %v\nOutput (前 1000 字符): %s", err, truncateString(output, 1000))
	}

	// 获取 all 镜像体积
	allSize, err := getImageSize(helper, buildCtx, ImageTag)
	if err != nil {
		t.Fatalf("获取 all 镜像大小失败: %v", err)
	}
	t.Logf("all 镜像大小: %d bytes (%.2f MB)", allSize, float64(allSize)/(1024*1024))

	// ---- Step 3: 验证比例阈值 ----
	ratio := float64(miniSize) / float64(allSize)
	t.Logf("PT-2: mini/all 体积比 = %.2f%%", ratio*100)

	if ratio >= 0.6 {
		t.Errorf("PT-2: mini/all 体积比 %.2f%% 超过阈值 60%%", ratio*100)
	}

	// ---- Cleanup: 清理 all 镜像 ----
	_, err = helper.ImageRemove(buildCtx, ImageTag, true, true)
	if err != nil {
		t.Logf("Warning: 清理 all 镜像失败: %v", err)
	}
}

// getImageSize 通过 ImageList 查询指定 tag 镜像的 Size 字段。
func getImageSize(helper *dockerhelper.Client, ctx context.Context, tag string) (int64, error) {
	images, err := helper.ImageList(ctx, types.ImageListOptions{})
	if err != nil {
		return 0, fmt.Errorf("ImageList() 失败: %w", err)
	}
	for _, img := range images {
		for _, t := range img.RepoTags {
			if t == tag {
				return img.Size, nil
			}
		}
	}
	return 0, fmt.Errorf("镜像 %s 不在本地镜像列表中", tag)
}

// truncateString 截断字符串到指定长度，用于日志输出。
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "\n... (已截断)"
}
