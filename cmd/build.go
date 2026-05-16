package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/agent-forge/cli/internal/build/buildengine"
	"github.com/agent-forge/cli/internal/shared/dockerhelper"
)

// buildCmd represents the build command
var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "构建 AI Coding Agent 的 Docker 镜像",
	Long: `根据指定的依赖列表构建包含 AI Coding Agent 运行环境的 Docker 镜像。

支持通过 -d 参数指定依赖列表，通过 -b 参数指定基础镜像，
通过 --no-cache 跳过缓存，通过 -R/--rebuild 安全重建等。`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// --- 获取参数 ---
		deps, _ := cmd.Flags().GetString("deps")
		baseImage, _ := cmd.Flags().GetString("base-image")
		config, _ := cmd.Flags().GetString("config")
		noCache, _ := cmd.Flags().GetBool("no-cache")
		rebuild, _ := cmd.Flags().GetBool("rebuild")
		maxRetry, _ := cmd.Flags().GetInt("max-retry")
		ghProxy, _ := cmd.Flags().GetString("gh-proxy")

		params := buildengine.BuildParams{
			Deps:           deps,
			BaseImage:      baseImage,
			Config:         config,
			NoCache:        noCache,
			Rebuild:        rebuild,
			MaxRetry:       maxRetry,
			GHProxy:        ghProxy,
			ProgressWriter: os.Stdout,
		}

		// --- 创建 Docker 客户端 ---
		helper, err := dockerhelper.NewClient()
		if err != nil {
			return fmt.Errorf("Docker 客户端初始化失败: %w", err)
		}
		defer helper.Close()

		// --- 执行构建 ---
		_, err = buildengine.New(helper).Build(cmd.Context(), params)
		if err != nil {
			return err
		}
		return nil
	},
}

func init() {
	buildCmd.Flags().StringP("deps", "d", "", "逗号分隔的依赖列表")
	buildCmd.Flags().StringP("base-image", "b", "docker.1ms.run/centos:7", "基础镜像（默认使用国内镜像源镜像）")
	buildCmd.Flags().StringP("config", "c", "", "配置目录路径")
	buildCmd.Flags().Bool("no-cache", false, "强制跳过 Docker 缓存")
	buildCmd.Flags().BoolP("rebuild", "R", false, "重建模式：使用临时标签构建，成功后替换原标签")
	buildCmd.Flags().Int("max-retry", 3, "网络错误时的最大重试次数")
	buildCmd.Flags().String("gh-proxy", "https://ghproxy.net", "GitHub 代理 URL（传空字符串禁用）")
}
