package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	// Version holds the semantic version, injected via -ldflags
	Version = "dev"
	// GitHash holds the short git commit hash, injected via -ldflags
	GitHash = "unknown"
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
func Execute() {
	if err := rootCmd.Execute(); err != nil {
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
	return fmt.Sprintf("agent-forge %s (%s)", Version, GitHash)
}
