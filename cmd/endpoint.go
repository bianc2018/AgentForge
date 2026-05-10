package cmd

import (
	"github.com/spf13/cobra"
)

// endpointCmd represents the endpoint command (stub — subcommands will be added in T-47)
var endpointCmd = &cobra.Command{
	Use:   "endpoint",
	Short: "管理 LLM 端点配置",
	Long: `管理 LLM 服务商端点配置，包括增删改查、连通性测试和配置同步。

子命令：providers, list, show, add, set, rm, test, apply, status`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// If no subcommand, show help
		return cmd.Help()
	},
}

func init() {
	endpointCmd.Flags().StringP("config", "c", "", "配置目录路径")
}
