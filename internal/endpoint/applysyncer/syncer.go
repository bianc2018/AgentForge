// Package applysyncer 提供 LLM 端点配置到 AI agent 配置文件的同步功能。
//
// 读取端点配置后，根据 Provider-Agent Matrix 确定该 provider 可服务的 agent 列表，
// 然后按各 agent 期望的格式将端点配置写入对应的配置文件。
//
// 支持的 agent 格式：
//   - claude:     .claude/.env（key=value 格式）
//   - opencode:   .opencode/.env（key=value 格式）
//   - kimi:       .kimi/config.toml（TOML 格式）
//   - deepseek-tui: .deepseek/.env（key=value 格式）
//
// 所有写入的文件权限设为 0600（NFR-9）。
package applysyncer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/agent-forge/cli/internal/endpoint/endpointmanager"
	"github.com/agent-forge/cli/internal/endpoint/provideragentmatrix"
)

// agentConfigFile 定义各 agent 的配置文件名和路径（相对于配置父目录）。
type agentConfigFile struct {
	relPath string // 相对于 config-dir 的路径，如 ".claude/.env"
}

// agentConfigFiles 是 agent 名称到配置文件的静态映射。
// 路径相对于配置父目录（由 Config Resolver 解析，默认 $(pwd)/coding-config）。
var agentConfigFiles = map[string]agentConfigFile{
	"claude":       {relPath: ".claude/.env"},
	"opencode":     {relPath: ".opencode/.env"},
	"kimi":         {relPath: ".kimi/config.toml"},
	"deepseek-tui": {relPath: ".deepseek/.env"},
}

// KnownAgents 返回所有受支持的 agent 名称列表。
func KnownAgents() []string {
	agents := make([]string, 0, len(agentConfigFiles))
	for agent := range agentConfigFiles {
		agents = append(agents, agent)
	}
	return agents
}

// IsKnownAgent 检查指定的 agent 名称是否受支持。
func IsKnownAgent(agent string) bool {
	_, ok := agentConfigFiles[agent]
	return ok
}

// GetConfigFilePath 返回指定 agent 在 configDir 下的完整配置文件路径。
func GetConfigFilePath(configDir string, agent string) string {
	info, ok := agentConfigFiles[agent]
	if !ok {
		return ""
	}
	return filepath.Join(configDir, info.relPath)
}

// FormatForAgent 将端点配置格式化为指定 agent 的配置文件内容。
//
// 各 agent 的格式：
//   - claude: key=value .env 格式，使用 OPENAI_API_KEY/BASE_URL 或 ANTHROPIC_API_KEY/BASE_URL
//   - opencode: key=value .env 格式，使用 OPENAI_API_KEY/BASE_URL
//   - kimi: TOML 格式的 [api] section
//   - deepseek-tui: key=value .env 格式，使用 DEEPSEEK_API_KEY/BASE_URL
//
// 未知 agent 名称返回 ErrUnknownAgent。
func FormatForAgent(cfg *endpointmanager.EndpointConfig, agent string) (string, error) {
	switch agent {
	case "claude":
		return formatClaude(cfg)
	case "opencode":
		return formatOpencode(cfg)
	case "kimi":
		return formatKimi(cfg)
	case "deepseek-tui":
		return formatDeepseekTui(cfg)
	default:
		return "", fmt.Errorf("不支持的 agent 名称: %s", agent)
	}
}

// formatClaude 生成 claude 的 .env 配置文件内容。
//
// provider 为 anthropic 时使用 ANTHROPIC_API_KEY/ANTHROPIC_BASE_URL，
// 其他 provider（openai、deepseek）时使用 OPENAI_API_KEY/OPENAI_BASE_URL。
func formatClaude(cfg *endpointmanager.EndpointConfig) (string, error) {
	var b strings.Builder

	switch cfg.Provider {
	case "anthropic":
		b.WriteString(fmt.Sprintf("ANTHROPIC_API_KEY=%s\n", cfg.Key))
		b.WriteString(fmt.Sprintf("ANTHROPIC_BASE_URL=%s\n", cfg.URL))
	default:
		// deepseek 和 openai 均使用 OpenAI 兼容格式
		b.WriteString(fmt.Sprintf("OPENAI_API_KEY=%s\n", cfg.Key))
		b.WriteString(fmt.Sprintf("OPENAI_BASE_URL=%s\n", cfg.URL))
	}

	if cfg.Model != "" {
		b.WriteString(fmt.Sprintf("MODEL=%s\n", cfg.Model))
	}
	if cfg.ModelOpus != "" {
		b.WriteString(fmt.Sprintf("MODEL_OPUS=%s\n", cfg.ModelOpus))
	}
	if cfg.ModelSonnet != "" {
		b.WriteString(fmt.Sprintf("MODEL_SONNET=%s\n", cfg.ModelSonnet))
	}
	if cfg.ModelHaiku != "" {
		b.WriteString(fmt.Sprintf("MODEL_HAIKU=%s\n", cfg.ModelHaiku))
	}
	if cfg.ModelSubagent != "" {
		b.WriteString(fmt.Sprintf("MODEL_SUBAGENT=%s\n", cfg.ModelSubagent))
	}

	return b.String(), nil
}

// formatOpencode 生成 opencode 的 .env 配置文件内容。
//
// opencode 始终使用 OPENAI_API_KEY/OPENAI_BASE_URL（OpenAI 兼容格式）。
func formatOpencode(cfg *endpointmanager.EndpointConfig) (string, error) {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("OPENAI_API_KEY=%s\n", cfg.Key))
	b.WriteString(fmt.Sprintf("OPENAI_BASE_URL=%s\n", cfg.URL))

	if cfg.Model != "" {
		b.WriteString(fmt.Sprintf("MODEL=%s\n", cfg.Model))
	}
	if cfg.ModelOpus != "" {
		b.WriteString(fmt.Sprintf("MODEL_OPUS=%s\n", cfg.ModelOpus))
	}
	if cfg.ModelSonnet != "" {
		b.WriteString(fmt.Sprintf("MODEL_SONNET=%s\n", cfg.ModelSonnet))
	}
	if cfg.ModelHaiku != "" {
		b.WriteString(fmt.Sprintf("MODEL_HAIKU=%s\n", cfg.ModelHaiku))
	}

	return b.String(), nil
}

// formatKimi 生成 kimi 的 TOML 格式配置文件内容。
//
// kimi 使用 TOML 格式，配置放在 [api] section 下。
// 特殊字符在 TOML 字符串中会被正确转义（使用 %q 格式化）。
func formatKimi(cfg *endpointmanager.EndpointConfig) (string, error) {
	var b strings.Builder

	b.WriteString("[api]\n")
	b.WriteString(fmt.Sprintf("key = %s\n", quoteTomlString(cfg.Key)))
	b.WriteString(fmt.Sprintf("base_url = %s\n", quoteTomlString(cfg.URL)))

	if cfg.Model != "" {
		b.WriteString(fmt.Sprintf("model = %s\n", quoteTomlString(cfg.Model)))
	}

	return b.String(), nil
}

// formatDeepseekTui 生成 deepseek-tui 的 .env 配置文件内容。
//
// deepseek-tui 使用 DEEPSEEK_API_KEY/DEEPSEEK_BASE_URL 环境变量。
func formatDeepseekTui(cfg *endpointmanager.EndpointConfig) (string, error) {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("DEEPSEEK_API_KEY=%s\n", cfg.Key))
	b.WriteString(fmt.Sprintf("DEEPSEEK_BASE_URL=%s\n", cfg.URL))

	if cfg.Model != "" {
		b.WriteString(fmt.Sprintf("MODEL=%s\n", cfg.Model))
	}

	return b.String(), nil
}

// quoteTomlString 将字符串包装为 TOML 双引号字符串，并转义其中的特殊字符。
func quoteTomlString(s string) string {
	// 对 TOML 字符串中的双引号和反斜杠进行转义
	escaped := strings.ReplaceAll(s, `\`, `\\`)
	escaped = strings.ReplaceAll(escaped, `"`, `\"`)
	return `"` + escaped + `"`
}

// WriteAgentConfig 将格式化后的配置内容写入指定 agent 的配置文件。
//
// 自动创建配置文件的父目录（权限 0755）。
// 配置文件权限设为 0600（NFR-9）。
// 返回的 error 包含操作上下文，供上游遵循 NFR-16 格式。
func WriteAgentConfig(configDir string, agent string, content string) error {
	info, ok := agentConfigFiles[agent]
	if !ok {
		return fmt.Errorf("不支持的 agent 名称: %s", agent)
	}

	filePath := filepath.Join(configDir, info.relPath)
	parentDir := filepath.Dir(filePath)

	// 创建父目录
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return fmt.Errorf("创建 agent 配置目录失败 %s: %w", parentDir, err)
	}

	// 写入文件，权限 0600
	if err := os.WriteFile(filePath, []byte(content), 0600); err != nil {
		return fmt.Errorf("写入 agent 配置文件失败 %s: %w", filePath, err)
	}

	return nil
}

// SyncEndpoint 将指定端点的配置同步到适用的 agent 配置文件中。
//
// 参数：
//   - endpointCfg: 端点配置
//   - configDir: 配置父目录（由 Config Resolver 解析，默认 $(pwd)/coding-config）
//   - agents: 要同步的 agent 列表，为空时同步到该 provider 对应的所有 agent
//
// 返回成功同步的 agent 名称列表和同步过程中发生的错误。
// 即使部分 agent 同步失败，也会继续同步其余 agent（尽力而为）。
// 所有同步失败的 agent 会被收集在返回的 error 中。
func SyncEndpoint(endpointCfg *endpointmanager.EndpointConfig, configDir string, agents []string) ([]string, error) {
	if agents == nil || len(agents) == 0 {
		// 未指定 agent 时，查询该 provider 可服务的所有 agent
		agents = provideragentmatrix.GetAgentsForProvider(endpointCfg.Provider)
	}

	if len(agents) == 0 {
		return nil, fmt.Errorf("provider %s 无可服务的 agent", endpointCfg.Provider)
	}

	var synced []string
	var errs []string

	for _, agent := range agents {
		if !IsKnownAgent(agent) {
			errs = append(errs, fmt.Sprintf("不支持的 agent 名称: %s", agent))
			continue
		}

		content, err := FormatForAgent(endpointCfg, agent)
		if err != nil {
			errs = append(errs, fmt.Sprintf("格式化 %s 配置失败: %s", agent, err.Error()))
			continue
		}

		if err := WriteAgentConfig(configDir, agent, content); err != nil {
			errs = append(errs, fmt.Sprintf("写入 %s 配置失败: %s", agent, err.Error()))
			continue
		}

		synced = append(synced, agent)
	}

	if len(errs) > 0 {
		return synced, fmt.Errorf("同步端点配置时部分 agent 写入失败:\n%s", strings.Join(errs, "\n"))
	}

	return synced, nil
}

// ReadAndSyncEndpoint 读取指定端点目录的配置并将其同步到适用的 agent 配置文件。
//
// 这是 ReadEndpointConfig + SyncEndpoint 的便捷组合。
func ReadAndSyncEndpoint(endpointDir string, configDir string, agents []string) ([]string, error) {
	cfg, err := endpointmanager.ReadEndpointConfig(endpointDir)
	if err != nil {
		return nil, fmt.Errorf("读取端点配置失败: %w", err)
	}

	return SyncEndpoint(cfg, configDir, agents)
}

// SyncAllEndpoints 遍历所有端点，将每个端点的配置同步到适用的 agent 配置文件。
//
// 遍历 <configDir>/endpoints/ 目录下的每个子目录，
// 对每个端点调用 ReadAndSyncEndpoint。
// 即使部分端点同步失败，也会继续同步其余端点（尽力而为）。
//
// 返回成功同步的 agent-端点 映射和同步过程中发生的错误。
func SyncAllEndpoints(endpointsDir string, configDir string, agents []string) (map[string][]string, error) {
	entries, err := os.ReadDir(endpointsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("端点配置目录不存在: %s", endpointsDir)
		}
		return nil, fmt.Errorf("读取端点配置目录失败 %s: %w", endpointsDir, err)
	}

	result := make(map[string][]string)
	var errs []string

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		epName := entry.Name()
		epDir := filepath.Join(endpointsDir, epName)

		synced, err := ReadAndSyncEndpoint(epDir, configDir, agents)
		if err != nil {
			errs = append(errs, fmt.Sprintf("端点 %s 同步失败: %s", epName, err.Error()))
		} else {
			result[epName] = synced
		}
	}

	if len(errs) > 0 {
		return result, fmt.Errorf("同步端点配置时部分端点失败:\n%s", strings.Join(errs, "\n"))
	}

	return result, nil
}
