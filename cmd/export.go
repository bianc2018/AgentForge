package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// exportCmd represents the export command
var exportCmd = &cobra.Command{
	Use:   "export [filename]",
	Short: "导出 Docker 镜像为 tar 文件",
	Long: `将 AgentForge Docker 镜像导出为 tar 文件，默认文件名为 agent-forge.tar。`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		filename := "agent-forge.tar"
		if len(args) > 0 {
			filename = args[0]
		}
		fmt.Printf("export 命令已调用（占位实现），文件名：%s\n", filename)
		return nil
	},
}

func init() {
}
