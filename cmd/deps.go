package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/agent-forge/cli/internal/deps/depsinspector"
	"github.com/agent-forge/cli/internal/shared/platform"
)

// depsCmd represents the deps command
var depsCmd = &cobra.Command{
	Use:   "deps",
	Short: "查询容器内依赖安装状态",
	Long: `在临时容器中执行检测脚本，按 agent/runtime/tool 分类输出各组件安装状态和版本号。

使用 agent-forge:latest 镜像创建临时容器，检测完成后自动销毁。
可通过 -i/--image 参数指定其他镜像。`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// --- 获取参数 ---
		imageRef, _ := cmd.Flags().GetString("image")

		// --- 推断平台 ---
		plt := ""
		if imageRef != "" {
			plt = platform.InferPlatform(imageRef)
		}

		// --- 执行依赖检测 ---
		result, err := depsinspector.RunDetection(imageRef, plt)
		if err != nil {
			return fmt.Errorf("依赖检测失败: %w", err)
		}

		// --- 输出结果 ---
		fmt.Print(depsinspector.FormatResult(result))

		return nil
	},
}

func init() {
	depsCmd.Flags().StringP("image", "i", "", "目标镜像引用（默认 agent-forge:latest）")
	depsCmd.Flags().StringP("config", "c", "", "配置目录路径")
}
