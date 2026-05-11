package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/agent-forge/cli/internal/distribution/engine"
	"github.com/agent-forge/cli/internal/shared/dockerhelper"
)

// exportCmd represents the export command
var exportCmd = &cobra.Command{
	Use:   "export [filename]",
	Short: "导出 Docker 镜像为 tar 文件",
	Long: `将 AgentForge Docker 镜像导出为 tar 文件。

默认导出 agent-forge:latest 镜像到 agent-forge.tar 文件。
可通过参数指定输出文件名，通过 --image 指定其他镜像。`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// --- 获取参数 ---
		filename := "agent-forge.tar"
		if len(args) > 0 {
			filename = args[0]
		}
		imageRef, _ := cmd.Flags().GetString("image")
		if imageRef == "" {
			imageRef = "agent-forge:latest"
		}

		// --- 创建 Docker 客户端 ---
		client, err := dockerhelper.NewClient()
		if err != nil {
			return fmt.Errorf("原因: 创建 Docker 客户端失败\n"+
				"上下文: 无法连接到 Docker daemon\n"+
				"建议: 请确认 Docker 已安装并运行\n"+
				"错误详情: %s", err.Error())
		}
		defer client.Close()

		// --- 导出镜像 ---
		distEngine := engine.New(client)
		ctx := context.Background()

		if err := distEngine.Export(ctx, imageRef, filename); err != nil {
			return fmt.Errorf("导出镜像失败: %w", err)
		}

		fmt.Printf("镜像 %s 已成功导出到 %s\n", imageRef, filename)
		return nil
	},
}

func init() {
	exportCmd.Flags().StringP("image", "i", "", "要导出的镜像引用（默认 agent-forge:latest）")
}
