package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/agent-forge/cli/internal/endpoint/endpointmanager"
	"github.com/agent-forge/cli/internal/shared/configresolver"
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

// endpointAddCmd represents the endpoint add subcommand
//
// 创建新的 LLM 端点配置。支持全部 8 个可选参数，参数齐全时直接创建。
// 如果同名端点已存在，返回错误并退出码 1。
// 创建成功后输出确认信息。
var endpointAddCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "新增 LLM 端点",
	Long: `创建新的 LLM 端点配置。

支持全部 8 个可选参数（--provider, --url, --key, --model,
--model-opus, --model-sonnet, --model-haiku, --model-subagent），
所有参数齐全时直接创建 endpoint.env 文件。

如果同名端点已存在，返回错误（退出码 1）。
创建成功后文件权限为 0600（NFR-9）。

示例：
  agent-forge endpoint add my-ep \
    --provider openai \
    --url https://api.openai.com \
    --key sk-test-key-value \
    --model gpt-4`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		// --- 读取参数 ---
		provider, _ := cmd.Flags().GetString("provider")
		url, _ := cmd.Flags().GetString("url")
		key, _ := cmd.Flags().GetString("key")
		model, _ := cmd.Flags().GetString("model")
		modelOpus, _ := cmd.Flags().GetString("model-opus")
		modelSonnet, _ := cmd.Flags().GetString("model-sonnet")
		modelHaiku, _ := cmd.Flags().GetString("model-haiku")
		modelSubagent, _ := cmd.Flags().GetString("model-subagent")

		// --- 解析配置目录 ---
		configFlag, _ := cmd.Flags().GetString("config")
		configDir, err := configresolver.Resolve(configFlag)
		if err != nil {
			return newExitCodeError(1,
				fmt.Sprintf("原因: 解析配置目录失败\n"+
					"上下文: 正在为 endpoint add 命令解析配置目录路径\n"+
					"建议: 请确认 -c 参数指定的路径有效，或使用默认路径\n"+
					"错误详情: %s", err.Error()),
			)
		}

		// --- 检查同名端点是否已存在 ---
		endpointDir := filepath.Join(configDir, "endpoints", name)
		if _, err := os.Stat(endpointDir); err == nil {
			return newExitCodeError(1,
				fmt.Sprintf("原因: 端点 %s 已存在\n"+
					"上下文: 正在创建新端点 %s，但配置目录 %s 已存在\n"+
					"建议: 请使用不同的端点名称，或使用 endpoint set 命令修改已有端点",
					name, name, endpointDir),
			)
		}

		// --- 构建配置并写入 ---
		cfg := &endpointmanager.EndpointConfig{
			Provider:      provider,
			URL:           url,
			Key:           key,
			Model:         model,
			ModelOpus:     modelOpus,
			ModelSonnet:   modelSonnet,
			ModelHaiku:    modelHaiku,
			ModelSubagent: modelSubagent,
		}

		if err := endpointmanager.WriteEndpointConfig(endpointDir, cfg); err != nil {
			return newExitCodeError(1,
				fmt.Sprintf("原因: 写入端点配置失败\n"+
					"上下文: 正在创建端点 %s 的配置文件 %s\n"+
					"建议: 请确认对配置目录 %s 有写入权限\n"+
					"错误详情: %s", name, endpointDir, configDir, err.Error()),
			)
		}

		// --- 输出成功信息 ---
		fmt.Printf("端点 %s 创建成功 (%s)\n", name, filepath.Join(endpointDir, endpointmanager.EndpointEnvFilename))
		return nil
	},
}

func init() {
	endpointCmd.AddCommand(endpointAddCmd)

	endpointAddCmd.Flags().String("provider", "", "LLM 服务商（deepseek / openai / anthropic）")
	endpointAddCmd.Flags().String("url", "", "API 基础 URL")
	endpointAddCmd.Flags().String("key", "", "API key")
	endpointAddCmd.Flags().String("model", "", "默认模型")
	endpointAddCmd.Flags().String("model-opus", "", "Opus 模型")
	endpointAddCmd.Flags().String("model-sonnet", "", "Sonnet 模型")
	endpointAddCmd.Flags().String("model-haiku", "", "Haiku 模型")
	endpointAddCmd.Flags().String("model-subagent", "", "Subagent 模型")

	endpointCmd.PersistentFlags().StringP("config", "c", "", "配置目录路径")
}
