package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// doctorCmd represents the doctor command
var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "执行三层环境诊断",
	Long:  `对 Docker 核心依赖、运行时状态和可选工具进行三层环境诊断。`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("doctor 命令已调用（占位实现）")
		return nil
	},
}

func init() {
	doctorCmd.Flags().StringP("config", "c", "", "配置目录路径")
}
