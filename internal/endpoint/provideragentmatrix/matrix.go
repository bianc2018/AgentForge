// Package provideragentmatrix 提供 LLM provider 与 AI agent 之间的静态映射关系。
//
// 该映射表硬编码在源码中，定义每个 LLM provider 可服务于哪些 agent，
// 用于 endpoint providers 查询、endpoint apply 配置同步和 endpoint status 映射展示。
package provideragentmatrix

// providerAgents 是 provider 到可服务 agent 列表的静态映射表。
// key 为 provider 名称，value 为该 provider 可服务的 agent 名称列表。
// 当前支持的 provider：deepseek、openai、anthropic
var providerAgents = map[string][]string{
	"deepseek":  {"claude", "opencode", "kimi", "deepseek-tui"},
	"openai":    {"claude", "opencode"},
	"anthropic": {"claude"},
}

// GetProviders 返回所有受支持的 LLM provider 名称列表。
//
// 每次调用返回新的切片副本，避免调用方修改内部映射数据。
// 返回值按固定顺序排列：[deepseek, openai, anthropic]。
func GetProviders() []string {
	return []string{"deepseek", "openai", "anthropic"}
}

// GetAgentsForProvider 返回指定 provider 可服务的 agent 名称列表。
//
// 如果 provider 受支持，返回对应的 agent 列表（切片副本）。
// 如果 provider 不受支持，返回 nil 空切片。
func GetAgentsForProvider(provider string) []string {
	agents, ok := providerAgents[provider]
	if !ok {
		return nil
	}
	// 返回副本，防止调用方修改内部映射
	result := make([]string, len(agents))
	copy(result, agents)
	return result
}
