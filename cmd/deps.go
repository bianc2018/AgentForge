package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// depsCmd represents the deps command
var depsCmd = &cobra.Command{
	Use:   "deps",
	Short: "查询容器内依赖安装状态",
	Long:  `在临时容器中执行检测脚本，按 agent/skill/tool/runtime 分类输出各组件安装状态和版本号。`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("deps 命令已调用（占位实现）")
		return nil
	},
}

func init() {
	depsCmd.Flags().StringP("config", "c", "", "配置目录路径")
}
