package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/agent-forge/cli/internal/shared/configresolver"
	"github.com/agent-forge/cli/internal/shared/logging"
)

var (
	// Version holds the semantic version, injected via -ldflags
	Version = "dev"
	// GitHash holds the short git commit hash, injected via -ldflags
	GitHash = "unknown"
	// BuildTime holds the ISO 8601 build timestamp, injected via -ldflags
	BuildTime = "unknown"
)

// rootCmd represents the base command
var rootCmd = &cobra.Command{
	Use:   "agent-forge",
	Short: "AgentForge -- AI Coding Agent 容器生命周期管理工具",
	Long: `AgentForge 是一个基于 Docker 的 AI Coding Agent 容器管理工具。

它提供构建、运行、端点管理、环境诊断等完整的容器生命周期管理能力，
帮助开发者快速搭建和运行 AI agent 开发环境。`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Default command: run 'run' if no subcommand is provided
		return runCmd.RunE(cmd, args)
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once.
//
// 退出码规范：
//   - 0: 命令执行成功
//   - 1: 通用执行错误（Docker 错误、容器启动失败等）
//   - 2: 参数错误（无效参数、缺少必填参数、参数格式错误）
//
// 特殊命令（如 run --run）可能使用其他退出码传递子进程退出状态。
func Execute() {
	// 初始化日志系统（使用默认配置目录，各子命令可通过 -c 覆盖）
	configDir, _ := configresolver.Resolve("")
	logging.Init(configDir)

	if err := rootCmd.Execute(); err != nil {
		// 检查是否实现了 ExitCoder 接口（自定义退出码）
		var ec ExitCoder
		if errors.As(err, &ec) {
			os.Exit(ec.ExitCode())
		}
		// 默认退出码 1
		os.Exit(1)
	}
}

func init() {
	// Register all subcommands
	rootCmd.AddCommand(buildCmd)
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(endpointCmd)
	rootCmd.AddCommand(doctorCmd)
	rootCmd.AddCommand(depsCmd)
	rootCmd.AddCommand(exportCmd)
	rootCmd.AddCommand(importCmd)
	rootCmd.AddCommand(updateCmd)
	rootCmd.AddCommand(versionCmd)
}

// VersionInfo returns a formatted version string
func VersionInfo() string {
	return fmt.Sprintf("agent-forge %s (%s) built at %s", Version, GitHash, BuildTime)
}
