package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/agent-forge/cli/internal/doctor/diagnosticengine"
	"github.com/agent-forge/cli/internal/shared/dockerhelper"
)

// doctorHelperAdapter 适配 dockerhelper.Client 到 diagnosticengine.DockerHelper 接口。
type doctorHelperAdapter struct {
	client *dockerhelper.Client
}

func (a *doctorHelperAdapter) Ping(ctx context.Context) error {
	return a.client.Ping(ctx)
}

func (a *doctorHelperAdapter) Info(ctx context.Context) (interface{}, error) {
	info, err := a.client.Info(ctx)
	return info, err
}

// doctorCmd represents the doctor command
var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "执行三层环境诊断",
	Long: `对 Docker 核心依赖、运行时状态和可选工具进行三层环境诊断。

检测过程：
  1. 第一层（核心依赖）：检查 Docker socket 是否可用
  2. 第二层（运行时）：检查 Docker daemon 运行状态和用户权限
  3. 第三层（可选工具）：检查 buildx 等可选插件

检测到缺失核心依赖时将自动使用系统包管理器安装。`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// --- 创建 Docker 客户端 ---
		client, err := dockerhelper.NewClient()
		if err != nil {
			return fmt.Errorf("原因: 创建 Docker 客户端失败\n"+
				"上下文: 无法连接到 Docker daemon\n"+
				"建议: 请确认 Docker 已安装并运行\n"+
				"错误详情: %s", err.Error())
		}
		defer client.Close()

		// --- 创建诊断引擎并执行诊断 ---
		engine := diagnosticengine.New(&doctorHelperAdapter{client: client})
		ctx := context.Background()

		result, err := engine.Diagnose(ctx)
		if err != nil {
			return fmt.Errorf("诊断过程发生错误: %w", err)
		}

		// --- 输出诊断结果 ---
		fmt.Println("========================================")
		fmt.Println("  AgentForge 环境诊断")
		fmt.Println("========================================")
		fmt.Println()

		// 第一层：核心依赖
		printLayerResult("第一层 - 核心依赖", result.CorePassed)
		if !result.CorePassed {
			printIssues(result.Issues, "核心依赖")
		}

		// 第二层：运行时
		printLayerResult("第二层 - 运行时", result.RuntimePassed)
		if !result.RuntimePassed {
			printIssues(result.Issues, "运行时")
		}

		// 第三层：可选工具
		printLayerResult("第三层 - 可选工具", result.OptionalPassed)
		if !result.OptionalPassed {
			printIssues(result.Issues, "可选工具")
		}

		// 第四层：平台兼容性
		printLayerResult("第四层 - 平台兼容性", result.PlatformPassed)
		if !result.PlatformPassed {
			printIssues(result.Issues, "平台兼容性")
		} else {
			printPlatformInfo(result.Issues)
		}

		fmt.Println()

		// 总结
		allPassed := result.CorePassed && result.RuntimePassed && result.OptionalPassed && result.PlatformPassed
		if allPassed {
			fmt.Println("  结果: 全部通过 (4/4)")
		} else {
			failed := 0
			if !result.CorePassed {
				failed++
			}
			if !result.RuntimePassed {
				failed++
			}
			if !result.OptionalPassed {
				failed++
			}
			fmt.Printf("  结果: %d 项未通过\n", failed)
			printIssueSuggestions(result.Issues)
		}

		fmt.Println("========================================")

		// 退出码：全部通过返回 0，任一层失败返回 1
		if !allPassed {
			return newExitCodeError(1, "环境诊断未全部通过")
		}

		return nil
	},
}

// printPlatformInfo 输出平台兼容性层的信息（通过时显示 daemon OS 信息）。
func printPlatformInfo(issues []diagnosticengine.Issue) {
	for _, issue := range issues {
		if issue.Layer == "平台兼容性" && issue.Type == diagnosticengine.IssueAllPassed {
			fmt.Printf("    -> %s\n", issue.Message)
		}
	}
}

// printLayerResult 输出单层诊断结果。
func printLayerResult(name string, passed bool) {
	status := "通过"
	if !passed {
		status = "未通过"
	}
	fmt.Printf("  %s: %s\n", name, status)
}

// printIssues 输出指定层的所有问题。
func printIssues(issues []diagnosticengine.Issue, layer string) {
	for _, issue := range issues {
		if issue.Layer == layer {
			fmt.Printf("    -> %s\n", issue.Message)
		}
	}
}

// printIssueSuggestions 输出所有问题的解决建议。
func printIssueSuggestions(issues []diagnosticengine.Issue) {
	fmt.Println()
	fmt.Println("  建议:")
	for _, issue := range issues {
		if issue.Suggestion != "" {
			fmt.Printf("    - %s\n", issue.Suggestion)
		}
	}
}

func init() {
	doctorCmd.Flags().StringP("config", "c", "", "配置目录路径")
}
