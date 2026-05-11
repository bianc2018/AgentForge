// Package engine 提供 DistributionEngine 的集成测试（IT-7）。
//
// 本文件覆盖 IT-7 的所有案例，在真实 Docker Engine 上验证镜像导出和导入功能。
package engine

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/agent-forge/cli/internal/shared/dockerhelper"
)

// testImageRef 是测试用的镜像引用，必须是本地已存在的镜像。
const testImageRef = "docker.1ms.run/centos:7"

// TestIT7_Export_FileExistsAndNonEmpty 验证 export 输出 tar 文件存在且非空。
//
// 覆盖案例：export — ImageSave API 调用成功，输出 tar 文件存在且非空。
func TestIT7_Export_FileExistsAndNonEmpty(t *testing.T) {
	client, err := dockerhelper.NewClient()
	if err != nil {
		t.Fatalf("创建 Docker 客户端失败: %v", err)
	}
	defer client.Close()

	skipIfImageNotExist(t, client)

	engine := New(client)
	ctx := context.Background()
	outputPath := t.TempDir() + "/export-test.tar"

	if err := engine.Export(ctx, testImageRef, outputPath); err != nil {
		t.Fatalf("Export() 返回错误: %v", err)
	}

	// 验证文件存在
	info, err := os.Stat(outputPath)
	if err != nil {
		t.Fatalf("导出文件不存在: %v", err)
	}

	// 验证文件非空
	if info.Size() == 0 {
		t.Error("导出文件不应为空")
	}

	t.Logf("导出的 tar 文件大小: %d bytes", info.Size())
}

// TestIT7_Import_ImageVisible 验证 import 后镜像在 ImageList 中可见。
//
// 覆盖案例：import — ImageLoad API 调用成功，镜像在 ImageList 中可见。
func TestIT7_Import_ImageVisible(t *testing.T) {
	client, err := dockerhelper.NewClient()
	if err != nil {
		t.Fatalf("创建 Docker 客户端失败: %v", err)
	}
	defer client.Close()

	skipIfImageNotExist(t, client)

	engine := New(client)
	ctx := context.Background()
	outputPath := t.TempDir() + "/import-test.tar"

	// 先导出
	if err := engine.Export(ctx, testImageRef, outputPath); err != nil {
		t.Fatalf("Export() 返回错误: %v", err)
	}

	// 导入
	if err := engine.Import(ctx, outputPath); err != nil {
		t.Fatalf("Import() 返回错误: %v", err)
	}

	// 验证镜像可见
	exists, err := client.ImageExists(ctx, testImageRef)
	if err != nil {
		t.Fatalf("ImageExists() 返回错误: %v", err)
	}
	if !exists {
		t.Errorf("导入后镜像 %s 应在 ImageList 中可见", testImageRef)
	}
}

// TestIT7_Export_NonexistentImage 验证导出不存在的镜像返回错误。
//
// 覆盖案例：export 不存在的镜像 — 返回错误。
func TestIT7_Export_NonexistentImage(t *testing.T) {
	client, err := dockerhelper.NewClient()
	if err != nil {
		t.Fatalf("创建 Docker 客户端失败: %v", err)
	}
	defer client.Close()

	engine := New(client)
	ctx := context.Background()
	outputPath := t.TempDir() + "/nonexistent.tar"

	err = engine.Export(ctx, "this-image-does-not-exist-12345", outputPath)
	if err == nil {
		t.Fatal("导出不存在的镜像应返回错误")
	}

	// 验证错误信息包含"镜像不存在"
	if !strings.Contains(err.Error(), "镜像") || !strings.Contains(err.Error(), "不存在") {
		t.Errorf("错误信息应包含 '镜像不存在', 实际: %v", err)
	}
}

// TestIT7_Import_NonexistentFile 验证导入不存在的文件返回错误。
//
// 覆盖案例：import 不存在的文件 — 返回错误。
func TestIT7_Import_NonexistentFile(t *testing.T) {
	client, err := dockerhelper.NewClient()
	if err != nil {
		t.Fatalf("创建 Docker 客户端失败: %v", err)
	}
	defer client.Close()

	engine := New(client)
	ctx := context.Background()

	err = engine.Import(ctx, "/tmp/nonexistent-file-12345.tar")
	if err == nil {
		t.Fatal("导入不存在的文件应返回错误")
	}

	if !strings.Contains(err.Error(), "不存在") {
		t.Errorf("错误信息应包含'不存在', 实际: %v", err)
	}
}

// TestIT7_ExportImport_RoundTrip 验证完整的导出再导入流程。
//
// 覆盖案例：
//   - export 后 import: 导入后的镜像可用
//   - 测试后清理导出文件
func TestIT7_ExportImport_RoundTrip(t *testing.T) {
	client, err := dockerhelper.NewClient()
	if err != nil {
		t.Fatalf("创建 Docker 客户端失败: %v", err)
	}
	defer client.Close()

	skipIfImageNotExist(t, client)

	engine := New(client)
	ctx := context.Background()
	outputPath := t.TempDir() + "/roundtrip-test.tar"

	// 导出
	if err := engine.Export(ctx, testImageRef, outputPath); err != nil {
		t.Fatalf("Export() 返回错误: %v", err)
	}

	// 导入（验证 tar 格式有效）
	if err := engine.Import(ctx, outputPath); err != nil {
		t.Fatalf("Import() 返回错误: %v", err)
	}

	// 验证镜像仍可见
	exists, err := client.ImageExists(ctx, testImageRef)
	if err != nil {
		t.Fatalf("ImageExists() 返回错误: %v", err)
	}
	if !exists {
		t.Errorf("导入后镜像 %s 应在 ImageList 中可见", testImageRef)
	}

	t.Log("导出-导入往返测试通过")
}

// skipIfImageNotExist 如果测试镜像不存在则跳过测试。
func skipIfImageNotExist(t *testing.T, client *dockerhelper.Client) {
	t.Helper()
	ctx := context.Background()
	exists, err := client.ImageExists(ctx, testImageRef)
	if err != nil {
		t.Fatalf("检查镜像存在性失败: %v", err)
	}
	if !exists {
		t.Skipf("镜像 %s 不存在，跳过集成测试", testImageRef)
	}
}
