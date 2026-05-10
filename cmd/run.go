package cmd

import (
	"context"
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/agent-forge/cli/internal/run/argspersistence"
	"github.com/agent-forge/cli/internal/run/runengine"
	"github.com/agent-forge/cli/internal/shared/argsparser"
	"github.com/agent-forge/cli/internal/shared/configresolver"
	"github.com/agent-forge/cli/internal/shared/dockerhelper"
)

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "启动 AI Coding Agent 容器",
	Long: `启动 AI Coding Agent 容器，支持多种启动模式。

通过 -a 指定 AI agent 将启动对应的交互式终端；
不指定 -a 时以 bash 模式启动，加载所有已安装的 AI agent wrapper 函数。`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// --- 读取参数 ---
		params := buildRunParams(cmd)

		// --- 校验参数 ---
		if err := validateRunParams(params); err != nil {
			return fmt.Errorf("参数错误: %s\n建议: %s", err.Reason, err.Suggestion)
		}

		// --- 解析配置目录 ---
		// 原因: 配置目录路径解析失败
		// 建议: 请确认 -c 参数指定的路径有效
		configPath, err := configresolver.Resolve(params.Config)
		if err != nil {
			return fmt.Errorf("解析配置目录失败: %w\n建议: 请确认 -c 参数指定的路径有效", err)
		}

		// --- 创建 Docker 客户端 ---
		// 原因: Docker daemon 连接失败
		// 上下文: 正在创建 Docker API 客户端
		// 建议: 请确认 Docker 已安装并运行，当前用户属于 docker 用户组
		client, err := dockerhelper.NewClient()
		if err != nil {
			return newExitCodeError(1,
				fmt.Sprintf("原因: 创建 Docker 客户端失败\n"+
					"上下文: 无法连接到 Docker daemon\n"+
					"建议: 请确认 Docker 已安装并运行，当前用户属于 docker 用户组\n"+
					"错误详情: %s", err.Error()),
			)
		}
		defer client.Close()

		// --- 创建 RunEngine 并执行 ---
		engine := runengine.New(client, configPath)
		ctx := context.Background()

		if err := engine.Run(ctx, params); err != nil {
			// ErrFileNotFound → exit code 2（参数错误），REQ-17
			if errors.Is(err, argspersistence.ErrFileNotFound) {
				return newExitCodeError(2,
					fmt.Sprintf("原因: %s\n"+
						"上下文: 正在使用 -r/--recall 参数恢复上次运行参数\n"+
						"建议: 请先不带 -r 参数执行一次 run 命令以保存运行参数，或直接指定运行参数",
						err.Error()),
				)
			}
			// 其他错误由 RunEngine 携带具体错误信息，统一由 Execute() 按 ExitCoder 接口处理
			return err
		}

		return nil
	},
}

// buildRunParams 从 cobra 命令标志中读取 run 命令的参数。
func buildRunParams(cmd *cobra.Command) argsparser.RunParams {
	params := argsparser.DefaultRunParams()

	params.Agent, _ = cmd.Flags().GetString("agent")
	params.Ports, _ = cmd.Flags().GetStringArray("port")
	params.Mounts, _ = cmd.Flags().GetStringArray("mount")
	params.Envs, _ = cmd.Flags().GetStringArray("env")
	params.Workdir, _ = cmd.Flags().GetString("workdir")
	params.Recall, _ = cmd.Flags().GetBool("recall")

	dockerMode, _ := cmd.Flags().GetBool("docker")
	dind, _ := cmd.Flags().GetBool("dind")
	params.Docker = dockerMode || dind

	params.RunCmd, _ = cmd.Flags().GetString("run")
	params.Config, _ = cmd.Flags().GetString("config")

	return params
}

// paramValidationError 表示参数校验错误。
type paramValidationError struct {
	Reason     string
	Suggestion string
}

func (e *paramValidationError) Error() string {
	return e.Reason
}

// validateRunParams 验证 run 命令的参数组合是否合法。
//
// 退出码: 2（参数错误）
// 校验的规则:
//   - -r/--recall 和 --run 不能同时使用
//   - --docker 和 --dind 功能重复但可共存（兼容处理）
func validateRunParams(params argsparser.RunParams) *paramValidationError {
	// -r 和 --run 互斥
	if params.Recall && params.RunCmd != "" {
		return &paramValidationError{
			Reason:     "-r/--recall 和 --run 不能同时使用",
			Suggestion: "请仅使用其中一种模式。-r 恢复上次运行参数，--run 在后台执行指定命令",
		}
	}

	return nil
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
