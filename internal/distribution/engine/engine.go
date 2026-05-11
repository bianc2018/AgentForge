// Package engine 提供 Docker 镜像导出和导入功能。
//
// 通过 Docker SDK 的 ImageSave/ImageLoad API 实现镜像分发，
// 支持自定义导出文件名（默认 agent-forge.tar）。
// 遵循 REQ-34（导出）和 REQ-35（导入）规范。
package engine

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/agent-forge/cli/internal/shared/dockerhelper"
)

// DistributionEngine 是镜像分发引擎。
type DistributionEngine struct {
	helper *dockerhelper.Client
}

// New 创建镜像分发引擎。
//
// helper 是 Docker Helper 客户端。
func New(helper *dockerhelper.Client) *DistributionEngine {
	return &DistributionEngine{
		helper: helper,
	}
}

// Export 将指定镜像导出为 tar 文件。
//
// imageRef 是镜像引用（如 "agent-forge:latest"），
// outputPath 是输出 tar 文件路径（如 "agent-forge.tar"）。
//
// 如果镜像不存在，返回 os.ErrNotExist 包装的错误。
// 如果输出目录不可写，返回写入错误。
func (e *DistributionEngine) Export(ctx context.Context, imageRef, outputPath string) error {
	// 检查镜像是否存在
	exists, err := e.helper.ImageExists(ctx, imageRef)
	if err != nil {
		return fmt.Errorf("检查镜像失败: %w", err)
	}
	if !exists {
		return fmt.Errorf("镜像 %s 不存在", imageRef)
	}

	// 通过 ImageSave API 导出镜像
	reader, err := e.helper.ImageSave(ctx, []string{imageRef})
	if err != nil {
		return fmt.Errorf("导出镜像失败: %w", err)
	}
	defer reader.Close()

	// 写入 tar 文件
	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("创建输出文件失败: %w", err)
	}
	defer outFile.Close()

	written, err := io.Copy(outFile, reader)
	if err != nil {
		return fmt.Errorf("写入导出文件失败: %w", err)
	}

	if written == 0 {
		return fmt.Errorf("导出文件为空")
	}

	return nil
}

// Import 从 tar 文件加载 Docker 镜像。
//
// inputPath 是 tar 文件路径。
// 加载后可通过 docker images 确认镜像可见。
//
// 如果文件不存在，返回 os.ErrNotExist 包装的错误。
// 如果 tar 文件格式无效，返回导入错误。
func (e *DistributionEngine) Import(ctx context.Context, inputPath string) error {
	// 检查文件是否存在
	if _, err := os.Stat(inputPath); os.IsNotExist(err) {
		return fmt.Errorf("导入文件 %s 不存在", inputPath)
	}

	// 打开 tar 文件
	inFile, err := os.Open(inputPath)
	if err != nil {
		return fmt.Errorf("打开导入文件失败: %w", err)
	}
	defer inFile.Close()

	// 通过 ImageLoad API 导入镜像
	resp, err := e.helper.ImageLoad(ctx, inFile, false)
	if err != nil {
		return fmt.Errorf("导入镜像失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应（确认导入成功）
	_, _ = io.Copy(io.Discard, resp.Body)

	return nil
}
