package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/agent-forge/cli/internal/endpoint/endpointmanager"
	"github.com/agent-forge/cli/internal/endpoint/provideragentmatrix"
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

当必要参数（--provider、--url、--key）缺失时，自动进入交互式
配置模式（NFR-14/REQ-23），逐一提示用户输入缺失的配置项。

如果同名端点已存在，返回错误（退出码 1）。
创建成功后文件权限为 0600（NFR-9）。

示例：
  # 带全部参数直接创建
  agent-forge endpoint add my-ep \
    --provider openai \
    --url https://api.openai.com \
    --key sk-test-key-value \
    --model gpt-4

  # 缺参交互式创建
  agent-forge endpoint add my-ep`,
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

		// --- 交互式缺参提示模式 (REQ-23/NFR-14) ---
		// 当必要参数（provider、url、key）缺失时，进入交互模式逐一提示
		if provider == "" || url == "" || key == "" {
			fmt.Println("进入交互式配置模式（缺少必要参数，请逐项输入）...")

			if provider == "" {
				fmt.Print("请输入 provider (可选值: deepseek/openai/anthropic): ")
				input, err := promptForInput()
				if err != nil {
					return newExitCodeError(1,
						fmt.Sprintf("原因: 读取用户输入失败\n"+
							"上下文: 正在交互式创建端点 %s\n"+
							"建议: 请确保标准输入可用，或通过命令行参数直接提供配置值",
							name),
					)
				}
				provider = input
			}
			if url == "" {
				fmt.Print("请输入 API URL (如 https://api.openai.com): ")
				input, err := promptForInput()
				if err != nil {
					return newExitCodeError(1,
						fmt.Sprintf("原因: 读取用户输入失败\n"+
							"上下文: 正在交互式创建端点 %s\n"+
							"建议: 请确保标准输入可用，或通过命令行参数直接提供配置值",
							name),
					)
				}
				url = input
			}
			if key == "" {
				fmt.Print("请输入 API key: ")
				input, err := promptForInput()
				if err != nil {
					return newExitCodeError(1,
						fmt.Sprintf("原因: 读取用户输入失败\n"+
							"上下文: 正在交互式创建端点 %s\n"+
							"建议: 请确保标准输入可用，或通过命令行参数直接提供配置值",
							name),
					)
				}
				key = input
			}
			if model == "" {
				fmt.Print("请输入默认模型 (可选，直接回车跳过): ")
				input, err := promptForInput()
				if err != nil {
					return newExitCodeError(1,
						fmt.Sprintf("原因: 读取用户输入失败\n"+
							"上下文: 正在交互式创建端点 %s\n"+
							"建议: 请确保标准输入可用，或通过命令行参数直接提供配置值",
							name),
					)
				}
				model = input
			}
		}

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

// endpointProvidersCmd represents the endpoint providers subcommand
//
// 列出所有受支持的 LLM 服务商及其可服务的 AI agent。
// 数据来源为 Provider-Agent Matrix 静态映射表（REQ-19）。
// 符合 NFR-5（1秒内完成）要求，纯内存操作无需 I/O。
var endpointProvidersCmd = &cobra.Command{
	Use:   "providers",
	Short: "列出所有支持的 LLM 服务商及其可服务的 AI agent",
	Long: `列出所有支持的 LLM 服务商及其可服务的 AI agent。

从 Provider-Agent Matrix 静态映射表读取数据，
输出服务商与可服务 agent 的对照表（REQ-19）。

每个 provider 及其可服务的 agent 列表在 1 秒内输出（NFR-5）。`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		providers := provideragentmatrix.GetProviders()

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "PROVIDER\t可服务的 Agent")
		fmt.Fprintln(w, "--------\t---------------")

		for _, p := range providers {
			agents := provideragentmatrix.GetAgentsForProvider(p)
			fmt.Fprintf(w, "%s\t%s\n", p, strings.Join(agents, ", "))
		}

		return w.Flush()
	},
}

// endpointListCmd represents the endpoint list subcommand
//
// 以表格形式列出所有已配置的 LLM 端点，包含 NAME、PROVIDER 和 MODEL 三列（REQ-20）。
// 遍历 <config-dir>/endpoints/ 目录下的每个子目录，读取 endpoint.env 获取 PROVIDER 和 MODEL。
// 如果端点目录不存在（首次使用时），输出空表头并正常退出。
var endpointListCmd = &cobra.Command{
	Use:   "list",
	Short: "列出所有端点的名称、服务商和模型",
	Long: `以表格形式列出所有已配置的 LLM 端点。

遍历端点配置目录下的所有端点，读取每个端点的 PROVIDER 和 MODEL 字段，
以 NAME / PROVIDER / MODEL 三列表格输出（REQ-20）。

示例：
  NAME          PROVIDER    MODEL
  my-ep         openai      gpt-4
  deepseek-ep   deepseek    deepseek-chat`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		configFlag, _ := cmd.Flags().GetString("config")
		configDir, err := configresolver.Resolve(configFlag)
		if err != nil {
			return newExitCodeError(1,
				fmt.Sprintf("原因: 解析配置目录失败\n"+
					"上下文: 正在为 endpoint list 命令解析配置目录路径\n"+
					"建议: 请确认 -c 参数指定的路径有效，或使用默认路径\n"+
					"错误详情: %s", err.Error()),
			)
		}

		endpointsDir := filepath.Join(configDir, "endpoints")

		// 如果端点目录不存在，输出空表头正常退出
		entries, err := os.ReadDir(endpointsDir)
		if err != nil {
			if os.IsNotExist(err) {
				w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
				fmt.Fprintln(w, "NAME\tPROVIDER\tMODEL")
				fmt.Fprintln(w, "----\t--------\t-----")
				return w.Flush()
			}
			return newExitCodeError(1,
				fmt.Sprintf("原因: 读取端点目录失败\n"+
					"上下文: 正在读取端点配置目录 %s\n"+
					"建议: 请确认配置目录存在且有读取权限\n"+
					"错误详情: %s", endpointsDir, err.Error()),
			)
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "NAME\tPROVIDER\tMODEL")
		fmt.Fprintln(w, "----\t--------\t-----")

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			name := entry.Name()
			epDir := filepath.Join(endpointsDir, name)
			cfg, err := endpointmanager.ReadEndpointConfig(epDir)
			if err != nil {
				// 无法读取时输出占位符确保表格对齐
				fmt.Fprintf(w, "%s\t(error)\t(error)\n", name)
				continue
			}
			fmt.Fprintf(w, "%s\t%s\t%s\n", name, cfg.Provider, cfg.Model)
		}

		return w.Flush()
	},
}

// endpointShowCmd represents the endpoint show subcommand
//
// 查看指定端点的详细配置，所有字段完整显示（REQ-21）。
// API key 前 8 位字符 + "***" + 后 4 位字符的掩码格式显示（NFR-6）。
// 端点不存在时返回退出码 1。
var endpointShowCmd = &cobra.Command{
	Use:   "show <name>",
	Short: "查看指定端点的详细配置",
	Long: `查看指定 LLM 端点的全部配置字段。

API key 以前 8 位字符 + *** + 后 4 位字符的掩码格式显示（NFR-6/REQ-21）。
端点不存在时返回退出码 1。

示例：
  agent-forge endpoint show my-ep

输出：
  名称:           my-ep
  Provider:       openai
  URL:            https://api.openai.com
  Key:            sk-test-***alue
  Model:          gpt-4`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		configFlag, _ := cmd.Flags().GetString("config")
		configDir, err := configresolver.Resolve(configFlag)
		if err != nil {
			return newExitCodeError(1,
				fmt.Sprintf("原因: 解析配置目录失败\n"+
					"上下文: 正在为 endpoint show 命令解析配置目录路径\n"+
					"建议: 请确认 -c 参数指定的路径有效，或使用默认路径\n"+
					"错误详情: %s", err.Error()),
			)
		}

		endpointDir := filepath.Join(configDir, "endpoints", name)

		cfg, err := endpointmanager.ReadEndpointConfig(endpointDir)
		if err != nil {
			return newExitCodeError(1,
				fmt.Sprintf("原因: 读取端点配置失败\n"+
					"上下文: 正在查看端点 %s 的配置\n"+
					"建议: 请确认端点 %s 已存在，或使用 endpoint list 查看所有可用端点\n"+
					"错误详情: %s", name, name, err.Error()),
			)
		}

		// 使用 tabwriter 对齐显示全部字段，KEY 做掩码处理 (NFR-6)
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintf(w, "名称:\t%s\n", name)
		fmt.Fprintf(w, "Provider:\t%s\n", cfg.Provider)
		fmt.Fprintf(w, "URL:\t%s\n", cfg.URL)
		fmt.Fprintf(w, "Key:\t%s\n", endpointmanager.MaskKey(cfg.Key))
		fmt.Fprintf(w, "Model:\t%s\n", cfg.Model)

		if cfg.ModelOpus != "" {
			fmt.Fprintf(w, "Model (Opus):\t%s\n", cfg.ModelOpus)
		}
		if cfg.ModelSonnet != "" {
			fmt.Fprintf(w, "Model (Sonnet):\t%s\n", cfg.ModelSonnet)
		}
		if cfg.ModelHaiku != "" {
			fmt.Fprintf(w, "Model (Haiku):\t%s\n", cfg.ModelHaiku)
		}
		if cfg.ModelSubagent != "" {
			fmt.Fprintf(w, "Model (Subagent):\t%s\n", cfg.ModelSubagent)
		}

		return w.Flush()
	},
}

// endpointSetCmd represents the endpoint set subcommand
//
// 修改已有 LLM 端点的配置参数（REQ-24）。
// 支持与 add 相同的全部 8 个可选参数，只需提供要修改的字段。
// 仅更新指定的字段，未提供的字段保持原值不变。
// 端点不存在时返回退出码 1。
// 修改成功后文件权限保持 0600（NFR-9）。
var endpointSetCmd = &cobra.Command{
	Use:   "set <name>",
	Short: "修改已有端点的配置参数",
	Long: `修改已有 LLM 端点的配置参数（REQ-24）。

支持与 add 相同的全部 8 个可选参数（--provider, --url, --key, --model,
--model-opus, --model-sonnet, --model-haiku, --model-subagent），
只需提供要修改的字段。未提供的字段保持原值不变。

端点不存在时返回退出码 1。
修改成功后文件权限保持 0600（NFR-9）。

示例：
  # 修改 API key 和模型
  agent-forge endpoint set my-ep --key sk-new-key --model gpt-5

  # 修改 endpoint 的 URL
  agent-forge endpoint set my-ep --url https://api.new-provider.com`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		// --- 读取提供的参数（只取非空的作为更新） ---
		provider, _ := cmd.Flags().GetString("provider")
		url, _ := cmd.Flags().GetString("url")
		key, _ := cmd.Flags().GetString("key")
		model, _ := cmd.Flags().GetString("model")
		modelOpus, _ := cmd.Flags().GetString("model-opus")
		modelSonnet, _ := cmd.Flags().GetString("model-sonnet")
		modelHaiku, _ := cmd.Flags().GetString("model-haiku")
		modelSubagent, _ := cmd.Flags().GetString("model-subagent")

		// --- 确认至少有一个字段要更新 ---
		if provider == "" && url == "" && key == "" && model == "" &&
			modelOpus == "" && modelSonnet == "" && modelHaiku == "" && modelSubagent == "" {
			return newExitCodeError(1,
				fmt.Sprintf("原因: 未指定要更新的字段\n"+
					"上下文: 正在修改端点 %s，但未提供任何更新参数\n"+
					"建议: 请至少提供一个要修改的参数，如 --key, --model, --url 等。\n"+
					"使用 endpoint set --help 查看所有可用的更新参数", name),
			)
		}

		// --- 解析配置目录 ---
		configFlag, _ := cmd.Flags().GetString("config")
		configDir, err := configresolver.Resolve(configFlag)
		if err != nil {
			return newExitCodeError(1,
				fmt.Sprintf("原因: 解析配置目录失败\n"+
					"上下文: 正在为 endpoint set 命令解析配置目录路径\n"+
					"建议: 请确认 -c 参数指定的路径有效，或使用默认路径\n"+
					"错误详情: %s", err.Error()),
			)
		}

		endpointDir := filepath.Join(configDir, "endpoints", name)

		// --- 构建更新配置（仅填充提供的字段，其余保持零值） ---
		updates := &endpointmanager.EndpointConfig{
			Provider:      provider,
			URL:           url,
			Key:           key,
			Model:         model,
			ModelOpus:     modelOpus,
			ModelSonnet:   modelSonnet,
			ModelHaiku:    modelHaiku,
			ModelSubagent: modelSubagent,
		}

		if err := endpointmanager.UpdateEndpointConfig(endpointDir, updates); err != nil {
			return newExitCodeError(1,
				fmt.Sprintf("原因: 更新端点配置失败\n"+
					"上下文: 正在修改端点 %s 的配置\n"+
					"建议: 请确认端点 %s 已存在，或使用 endpoint list 查看所有可用端点\n"+
					"错误详情: %s", name, name, err.Error()),
			)
		}

		fmt.Printf("端点 %s 已更新\n", name)
		return nil
	},
}

// endpointRmCmd represents the endpoint rm subcommand
//
// 删除指定 LLM 端点及其对应目录（REQ-25）。
// 递归删除 <config-dir>/endpoints/<name>/ 整个目录。
// 端点不存在时返回退出码 1。
// 删除后 endpoint list 输出中不再包含被删除的端点。
var endpointRmCmd = &cobra.Command{
	Use:   "rm <name>",
	Short: "删除指定端点及其对应目录",
	Long: `删除指定 LLM 端点及其对应配置目录（REQ-25）。

递归删除 <config-dir>/endpoints/<name>/ 整个目录。
删除后 endpoint list 输出中不再包含被删除的端点。
端点不存在时返回退出码 1。

示例：
  agent-forge endpoint rm my-ep`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		// --- 解析配置目录 ---
		configFlag, _ := cmd.Flags().GetString("config")
		configDir, err := configresolver.Resolve(configFlag)
		if err != nil {
			return newExitCodeError(1,
				fmt.Sprintf("原因: 解析配置目录失败\n"+
					"上下文: 正在为 endpoint rm 命令解析配置目录路径\n"+
					"建议: 请确认 -c 参数指定的路径有效，或使用默认路径\n"+
					"错误详情: %s", err.Error()),
			)
		}

		endpointDir := filepath.Join(configDir, "endpoints", name)

		if err := endpointmanager.RemoveEndpointConfig(endpointDir); err != nil {
			return newExitCodeError(1,
				fmt.Sprintf("原因: 删除端点失败\n"+
					"上下文: 正在删除端点 %s 的配置目录 %s\n"+
					"建议: 请确认端点 %s 已存在，或使用 endpoint list 查看所有可用端点\n"+
					"错误详情: %s", name, endpointDir, name, err.Error()),
			)
		}

		fmt.Printf("端点 %s 已删除\n", name)
		return nil
	},
}

// endpointTestCmd represents the endpoint test subcommand
//
// 测试指定 LLM 端点的连通性（REQ-26/REQ-27）。
// 通过 Go net/http 向端点发送 POST chat/completions 请求，测量延迟并输出回复摘要。
// 请求超时时间设为 30 秒（NFR-4）。
// 端点不可达/超时/认证失败时输出符合 NFR-16 格式的错误信息，退出码非零。
var endpointTestCmd = &cobra.Command{
	Use:   "test <name>",
	Short: "测试端点连通性",
	Long: `测试指定 LLM 端点的连通性（REQ-26/REQ-27）。

通过 Go net/http 向端点 URL 发送 POST chat/completions 请求，
测量请求延迟并输出回复摘要。请求超时时间设为 30 秒（NFR-4）。

端点可达时输出延迟和回复摘要，退出码 0。
端点不可达、超时或返回认证错误时输出清晰的错误信息，退出码非零。

示例：
  agent-forge endpoint test my-ep

输出（成功）:
  >> 端点 my-ep 连通性测试
  请求延迟: 320ms
  响应模型: gpt-4
  回复摘要: Hello! How can I help you today?

输出（失败）:
  >> 端点 my-ep 连通性测试
  原因: 请求超时（30 秒）
  上下文: 在 30 秒内无法连接到 https://api.example.com/chat/completions
  建议: 请检查网络连通性，确认 URL 可达，或增加超时时间`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		// --- 解析配置目录 ---
		configFlag, _ := cmd.Flags().GetString("config")
		configDir, err := configresolver.Resolve(configFlag)
		if err != nil {
			return newExitCodeError(1,
				fmt.Sprintf("原因: 解析配置目录失败\n"+
					"上下文: 正在为 endpoint test 命令解析配置目录路径\n"+
					"建议: 请确认 -c 参数指定的路径有效，或使用默认路径\n"+
					"错误详情: %s", err.Error()),
			)
		}

		endpointDir := filepath.Join(configDir, "endpoints", name)

		fmt.Printf(">> 端点 %s 连通性测试\n", name)

		result, err := endpointmanager.TestEndpoint(endpointDir)
		if err != nil {
			return newExitCodeError(1,
				fmt.Sprintf(">> 端点 %s 连通性测试\n%s", name, err.Error()),
			)
		}

		// 输出成功结果
		latencyStr := fmt.Sprintf("%dms", result.Latency.Milliseconds())
		if result.Latency.Milliseconds() < 1000 {
			latencyStr = fmt.Sprintf("%dms", result.Latency.Milliseconds())
		} else {
			latencyStr = fmt.Sprintf("%.2fs", result.Latency.Seconds())
		}

		fmt.Printf("  请求延迟: %s\n", latencyStr)
		fmt.Printf("  响应模型: %s\n", result.Model)
		fmt.Printf("  回复摘要: %s\n", result.ResponsePreview)
		return nil
	},
}

func init() {
	endpointCmd.AddCommand(endpointProvidersCmd)
	endpointCmd.AddCommand(endpointListCmd)
	endpointCmd.AddCommand(endpointShowCmd)
	endpointCmd.AddCommand(endpointAddCmd)
	endpointCmd.AddCommand(endpointSetCmd)
	endpointCmd.AddCommand(endpointRmCmd)
	endpointCmd.AddCommand(endpointTestCmd)

	endpointAddCmd.Flags().String("provider", "", "LLM 服务商（deepseek / openai / anthropic）")
	endpointAddCmd.Flags().String("url", "", "API 基础 URL")
	endpointAddCmd.Flags().String("key", "", "API key")
	endpointAddCmd.Flags().String("model", "", "默认模型")
	endpointAddCmd.Flags().String("model-opus", "", "Opus 模型")
	endpointAddCmd.Flags().String("model-sonnet", "", "Sonnet 模型")
	endpointAddCmd.Flags().String("model-haiku", "", "Haiku 模型")
	endpointAddCmd.Flags().String("model-subagent", "", "Subagent 模型")

	endpointSetCmd.Flags().String("provider", "", "LLM 服务商（deepseek / openai / anthropic）")
	endpointSetCmd.Flags().String("url", "", "API 基础 URL")
	endpointSetCmd.Flags().String("key", "", "API key")
	endpointSetCmd.Flags().String("model", "", "默认模型")
	endpointSetCmd.Flags().String("model-opus", "", "Opus 模型")
	endpointSetCmd.Flags().String("model-sonnet", "", "Sonnet 模型")
	endpointSetCmd.Flags().String("model-haiku", "", "Haiku 模型")
	endpointSetCmd.Flags().String("model-subagent", "", "Subagent 模型")

	endpointCmd.PersistentFlags().StringP("config", "c", "", "配置目录路径")
}

// promptForInput 从标准输入读取一行用户输入，去除首尾空白。
func promptForInput() (string, error) {
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(input), nil
}
