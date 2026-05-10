package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// importCmd represents the import command
var importCmd = &cobra.Command{
	Use:   "import <file>",
	Short: "从 tar 文件导入 Docker 镜像",
	Long:  `从指定的 tar 文件加载 Docker 镜像，使其在 docker images 中可见并可用于 run 命令。`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("import 命令已调用（占位实现），文件：%s\n", args[0])
		return nil
	},
}

func init() {
}
