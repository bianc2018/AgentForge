package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "显示版本号和 git hash",
	Long:  `输出格式化的版本号和当前 git hash（通过编译期 -ldflags 注入）。`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println(VersionInfo())
		return nil
	},
}

func init() {
}
