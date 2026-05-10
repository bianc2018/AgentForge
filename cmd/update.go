package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// updateCmd represents the update command
var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "更新 AgentForge 到最新版本",
	Long:  `从 Git 远程仓库或 UPDATE_URL 下载最新版本 CLI 二进制并更新。`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("update 命令已调用（占位实现）")
		return nil
	},
}

func init() {
}
