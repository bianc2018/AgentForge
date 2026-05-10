package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "启动 AI Coding Agent 容器",
	Long: `启动 AI Coding Agent 容器，支持多种启动模式。

通过 -a 指定 AI agent 将启动对应的交互式终端；
不指定 -a 时以 bash 模式启动，加载所有已安装的 AI agent wrapper 函数。`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("run 命令已调用（占位实现）")
		return nil
	},
}

func init() {
	runCmd.Flags().StringP("agent", "a", "", "AI agent 名称（claude/opencode/kimi/deepseek-tui）")
	runCmd.Flags().StringArrayP("port", "p", nil, "端口映射（如 -p 3000:3000，可多次使用）")
	runCmd.Flags().StringArrayP("mount", "m", nil, "只读目录挂载（如 -m /host/data，可多次使用）")
	runCmd.Flags().StringArrayP("env", "e", nil, "环境变量（如 -e KEY=VAL，可多次使用）")
	runCmd.Flags().StringP("workdir", "w", "", "容器内工作目录（默认为当前目录）")
	runCmd.Flags().BoolP("recall", "r", false, "从 .last_args 恢复上次运行参数")
	runCmd.Flags().Bool("docker", false, "以 Docker-in-Docker 特权模式启动")
	runCmd.Flags().Bool("dind", false, "以 Docker-in-Docker 特权模式启动（同 --docker）")
	runCmd.Flags().String("run", "", "在后台执行指定命令后退出容器")
	runCmd.Flags().StringP("config", "c", "", "配置目录路径")
}
