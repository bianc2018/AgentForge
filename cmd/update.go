package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/agent-forge/cli/internal/update/engine"
)

// updateCmd represents the update command
var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "更新 AgentForge 到最新版本",
	Long: `从 Git 远程仓库或 UPDATE_URL 下载最新版本 CLI 二进制并更新。

更新过程：
  1. 备份当前版本
  2. 从 UPDATE_URL 下载最新版本
  3. 验证下载完整性
  4. 替换当前二进制
  5. 更新失败时自动回滚

可通过 UPDATE_URL 环境变量指定自定义更新源。`,
	RunE: func(cmd *cobra.Command, args []string) error {
		updater := engine.New()
		if err := updater.Update(); err != nil {
			return fmt.Errorf("自更新失败: %w", err)
		}

		fmt.Println("AgentForge 已成功更新到最新版本")
		return nil
	},
}

func init() {
}
