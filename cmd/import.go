package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/agent-forge/cli/internal/distribution/engine"
	"github.com/agent-forge/cli/internal/shared/dockerhelper"
)

// importCmd represents the import command
var importCmd = &cobra.Command{
	Use:   "import <file>",
	Short: "从 tar 文件导入 Docker 镜像",
	Long:  `从指定的 tar 文件加载 Docker 镜像，使其在 docker images 中可见并可用于 run 命令。`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		filename := args[0]

		// --- 创建 Docker 客户端 ---
		client, err := dockerhelper.NewClient()
		if err != nil {
			return fmt.Errorf("原因: 创建 Docker 客户端失败\n"+
				"上下文: 无法连接到 Docker daemon\n"+
				"建议: 请确认 Docker 已安装并运行\n"+
				"错误详情: %s", err.Error())
		}
		defer client.Close()

		// --- 导入镜像 ---
		distEngine := engine.New(client)
		ctx := context.Background()

		if err := distEngine.Import(ctx, filename); err != nil {
			return fmt.Errorf("导入镜像失败: %w", err)
		}

		fmt.Printf("镜像已成功从 %s 导入\n", filename)
		return nil
	},
}

func init() {
}
