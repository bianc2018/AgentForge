package applysyncer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/agent-forge/cli/internal/endpoint/endpointmanager"
)

// --- UT-10: FormatForAgent ---

func TestFormatForAgent_Claude(t *testing.T) {
	cfg := &endpointmanager.EndpointConfig{
		Provider: "openai",
		URL:      "https://api.openai.com",
		Key:      "sk-test-key-value",
		Model:    "gpt-4",
	}
	content, err := FormatForAgent(cfg, "claude")
	if err != nil {
		t.Fatalf("FormatForAgent(claude) 返回错误: %v", err)
	}
	if !strings.Contains(content, "OPENAI_API_KEY=sk-test-key-value") {
		t.Errorf("claude .env 应包含 OPENAI_API_KEY，got:\n%s", content)
	}
	if !strings.Contains(content, "OPENAI_BASE_URL=https://api.openai.com") {
		t.Errorf("claude .env 应包含 OPENAI_BASE_URL，got:\n%s", content)
	}
	if !strings.Contains(content, "MODEL=gpt-4") {
		t.Errorf("claude .env 应包含 MODEL，got:\n%s", content)
	}
}

func TestFormatForAgent_ClaudeAnthropic(t *testing.T) {
	cfg := &endpointmanager.EndpointConfig{
		Provider: "anthropic",
		URL:      "https://api.anthropic.com",
		Key:      "sk-ant-test",
		Model:    "claude-3-opus",
	}
	content, err := FormatForAgent(cfg, "claude")
	if err != nil {
		t.Fatalf("FormatForAgent(claude, anthropic) 返回错误: %v", err)
	}
	if !strings.Contains(content, "ANTHROPIC_API_KEY=sk-ant-test") {
		t.Errorf("anthropic provider 时应使用 ANTHROPIC_API_KEY，got:\n%s", content)
	}
	if !strings.Contains(content, "ANTHROPIC_BASE_URL=https://api.anthropic.com") {
		t.Errorf("anthropic provider 时应使用 ANTHROPIC_BASE_URL，got:\n%s", content)
	}
	if strings.Contains(content, "OPENAI_API_KEY") {
		t.Errorf("anthropic provider 时不应包含 OPENAI_API_KEY")
	}
}

func TestFormatForAgent_Opencode(t *testing.T) {
	cfg := &endpointmanager.EndpointConfig{
		Provider: "deepseek",
		URL:      "https://api.deepseek.com",
		Key:      "sk-ds-key",
		Model:    "deepseek-chat",
	}
	content, err := FormatForAgent(cfg, "opencode")
	if err != nil {
		t.Fatalf("FormatForAgent(opencode) 返回错误: %v", err)
	}
	if !strings.Contains(content, "OPENAI_API_KEY=sk-ds-key") {
		t.Errorf("opencode .env 应包含 OPENAI_API_KEY，got:\n%s", content)
	}
	if !strings.Contains(content, "OPENAI_BASE_URL=https://api.deepseek.com") {
		t.Errorf("opencode .env 应包含 OPENAI_BASE_URL，got:\n%s", content)
	}
	if !strings.Contains(content, "MODEL=deepseek-chat") {
		t.Errorf("opencode .env 应包含 MODEL，got:\n%s", content)
	}
}

func TestFormatForAgent_Kimi(t *testing.T) {
	cfg := &endpointmanager.EndpointConfig{
		Provider: "openai",
		URL:      "https://api.openai.com",
		Key:      "sk-test-key",
		Model:    "gpt-4",
	}
	content, err := FormatForAgent(cfg, "kimi")
	if err != nil {
		t.Fatalf("FormatForAgent(kimi) 返回错误: %v", err)
	}
	if !strings.Contains(content, "[api]") {
		t.Errorf("kimi config.toml 应包含 [api] section，got:\n%s", content)
	}
	if !strings.Contains(content, `key = "sk-test-key"`) {
		t.Errorf("kimi config.toml 应包含 key，got:\n%s", content)
	}
	if !strings.Contains(content, `base_url = "https://api.openai.com"`) {
		t.Errorf("kimi config.toml 应包含 base_url，got:\n%s", content)
	}
	if !strings.Contains(content, `model = "gpt-4"`) {
		t.Errorf("kimi config.toml 应包含 model，got:\n%s", content)
	}
}

func TestFormatForAgent_DeepseekTui(t *testing.T) {
	cfg := &endpointmanager.EndpointConfig{
		Provider: "deepseek",
		URL:      "https://api.deepseek.com",
		Key:      "sk-ds-key-value",
		Model:    "deepseek-chat",
	}
	content, err := FormatForAgent(cfg, "deepseek-tui")
	if err != nil {
		t.Fatalf("FormatForAgent(deepseek-tui) 返回错误: %v", err)
	}
	if !strings.Contains(content, "DEEPSEEK_API_KEY=sk-ds-key-value") {
		t.Errorf("deepseek-tui .env 应包含 DEEPSEEK_API_KEY，got:\n%s", content)
	}
	if !strings.Contains(content, "DEEPSEEK_BASE_URL=https://api.deepseek.com") {
		t.Errorf("deepseek-tui .env 应包含 DEEPSEEK_BASE_URL，got:\n%s", content)
	}
	if !strings.Contains(content, "MODEL=deepseek-chat") {
		t.Errorf("deepseek-tui .env 应包含 MODEL，got:\n%s", content)
	}
}

func TestFormatForAgent_UnknownAgent(t *testing.T) {
	cfg := &endpointmanager.EndpointConfig{
		Provider: "openai",
		URL:      "https://api.openai.com",
		Key:      "sk-test-key",
	}
	_, err := FormatForAgent(cfg, "unknown-agent")
	if err == nil {
		t.Fatal("未知 agent 应返回错误")
	}
	if !strings.Contains(err.Error(), "不支持的 agent") {
		t.Errorf("错误信息应包含'不支持的 agent'，got: %v", err)
	}
}

func TestFormatForAgent_SpecialCharsInTOML(t *testing.T) {
	cfg := &endpointmanager.EndpointConfig{
		Provider: "openai",
		URL:      "https://api.openai.com/v1",
		Key:      `sk-"quoted"-key`,
		Model:    `gpt-4"turbo"`,
	}
	content, err := FormatForAgent(cfg, "kimi")
	if err != nil {
		t.Fatalf("FormatForAgent(kimi, special chars) 返回错误: %v", err)
	}
	// TOML 中双引号应被转义
	if !strings.Contains(content, `key = "sk-\"quoted\"-key"`) {
		t.Errorf("含特殊字符的 key 在 TOML 中应正确转义，got:\n%s", content)
	}
	if !strings.Contains(content, `model = "gpt-4\"turbo\""`) {
		t.Errorf("含特殊字符的 model 在 TOML 中应正确转义，got:\n%s", content)
	}
}

// --- WriteAgentConfig tests ---

func TestWriteAgentConfig_Claude(t *testing.T) {
	tmpDir := t.TempDir()
	content := "OPENAI_API_KEY=sk-test-key\nOPENAI_BASE_URL=https://api.openai.com\n"

	err := WriteAgentConfig(tmpDir, "claude", content)
	if err != nil {
		t.Fatalf("WriteAgentConfig(claude) 返回错误: %v", err)
	}

	filePath := filepath.Join(tmpDir, ".claude", ".env")
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Fatal(".claude/.env 文件未被创建")
	}

	// 验证文件权限为 0600
	info, err := os.Stat(filePath)
	if err != nil {
		t.Fatalf("无法 stat .claude/.env: %v", err)
	}
	perm := info.Mode().Perm()
	if perm != 0600 {
		t.Errorf("文件权限为 %o, 期望 0600", perm)
	}

	// 验证内容正确
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("读取 .claude/.env 失败: %v", err)
	}
	if string(data) != content {
		t.Errorf("文件内容不匹配:\n期望:\n%s\n实际:\n%s", content, string(data))
	}
}

func TestWriteAgentConfig_DeepseekTui(t *testing.T) {
	tmpDir := t.TempDir()
	content := "DEEPSEEK_API_KEY=sk-ds-key\nDEEPSEEK_BASE_URL=https://api.deepseek.com\n"

	err := WriteAgentConfig(tmpDir, "deepseek-tui", content)
	if err != nil {
		t.Fatalf("WriteAgentConfig(deepseek-tui) 返回错误: %v", err)
	}

	filePath := filepath.Join(tmpDir, ".deepseek", ".env")
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Fatal(".deepseek/.env 文件未被创建")
	}

	info, err := os.Stat(filePath)
	if err != nil {
		t.Fatalf("无法 stat .deepseek/.env: %v", err)
	}
	perm := info.Mode().Perm()
	if perm != 0600 {
		t.Errorf("文件权限为 %o, 期望 0600", perm)
	}
}

func TestWriteAgentConfig_UnknownAgent(t *testing.T) {
	tmpDir := t.TempDir()
	err := WriteAgentConfig(tmpDir, "unknown", "content")
	if err == nil {
		t.Fatal("未知 agent 应返回错误")
	}
}

// --- SyncEndpoint tests ---

func TestSyncEndpoint_DeepseekToAllAgents(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &endpointmanager.EndpointConfig{
		Provider: "deepseek",
		URL:      "https://api.deepseek.com",
		Key:      "sk-ds-key",
		Model:    "deepseek-chat",
	}

	// 不指定 agent（同步到所有适用 agent）
	synced, err := SyncEndpoint(cfg, tmpDir, nil)
	if err != nil {
		t.Fatalf("SyncEndpoint 返回错误: %v", err)
	}

	// deepseek 应同步到 claude, opencode, kimi, deepseek-tui
	expectedAgents := []string{"claude", "opencode", "kimi", "deepseek-tui"}
	for _, agent := range expectedAgents {
		found := false
		for _, s := range synced {
			if s == agent {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("同步结果中未包含 agent %s，synced = %v", agent, synced)
		}
	}

	// 验证各配置文件已创建
	filesToCheck := []string{
		".claude/.env",
		".opencode/.env",
		".kimi/config.toml",
		".deepseek/.env",
	}
	for _, relPath := range filesToCheck {
		fullPath := filepath.Join(tmpDir, relPath)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			t.Errorf("配置文件 %s 未被创建", relPath)
		}
	}
}

func TestSyncEndpoint_WithAgentFilter(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &endpointmanager.EndpointConfig{
		Provider: "openai",
		URL:      "https://api.openai.com",
		Key:      "sk-test-key",
		Model:    "gpt-4",
	}

	// 仅同步到 claude 和 kimi
	agents := []string{"claude", "kimi"}
	synced, err := SyncEndpoint(cfg, tmpDir, agents)
	if err != nil {
		t.Fatalf("SyncEndpoint 返回错误: %v", err)
	}

	if len(synced) != 2 {
		t.Errorf("期望同步 2 个 agent，实际 %d: %v", len(synced), synced)
	}

	// claude 和 kimi 的配置文件应存在
	if _, err := os.Stat(filepath.Join(tmpDir, ".claude", ".env")); os.IsNotExist(err) {
		t.Error(".claude/.env 应存在")
	}
	if _, err := os.Stat(filepath.Join(tmpDir, ".kimi", "config.toml")); os.IsNotExist(err) {
		t.Error(".kimi/config.toml 应存在")
	}

	// opencode 和 deepseek-tui 的配置文件应不存在
	if _, err := os.Stat(filepath.Join(tmpDir, ".opencode", ".env")); !os.IsNotExist(err) {
		t.Error(".opencode/.env 不应存在")
	}
	if _, err := os.Stat(filepath.Join(tmpDir, ".deepseek", ".env")); !os.IsNotExist(err) {
		t.Error(".deepseek/.env 不应存在")
	}
}

func TestSyncEndpoint_AgentConfigPermission(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &endpointmanager.EndpointConfig{
		Provider: "deepseek",
		URL:      "https://api.deepseek.com",
		Key:      "sk-ds-key",
		Model:    "deepseek-chat",
	}

	_, err := SyncEndpoint(cfg, tmpDir, []string{"claude"})
	if err != nil {
		t.Fatalf("SyncEndpoint 返回错误: %v", err)
	}

	filePath := filepath.Join(tmpDir, ".claude", ".env")
	info, err := os.Stat(filePath)
	if err != nil {
		t.Fatalf("无法 stat .claude/.env: %v", err)
	}
	perm := info.Mode().Perm()
	if perm != 0600 {
		t.Errorf("文件权限为 %o, 期望 0600 (NFR-9)", perm)
	}
}

// --- SyncAllEndpoints tests ---

func TestSyncAllEndpoints(t *testing.T) {
	tmpDir := t.TempDir()
	endpointsDir := filepath.Join(tmpDir, "endpoints")

	// 创建两个端点
	ep1Dir := filepath.Join(endpointsDir, "ep-openai")
	ep1Cfg := &endpointmanager.EndpointConfig{
		Provider: "openai",
		URL:      "https://api.openai.com",
		Key:      "sk-openai-key",
		Model:    "gpt-4",
	}
	if err := endpointmanager.WriteEndpointConfig(ep1Dir, ep1Cfg); err != nil {
		t.Fatalf("创建端点 ep-openai 失败: %v", err)
	}

	ep2Dir := filepath.Join(endpointsDir, "ep-deepseek")
	ep2Cfg := &endpointmanager.EndpointConfig{
		Provider: "deepseek",
		URL:      "https://api.deepseek.com",
		Key:      "sk-ds-key",
		Model:    "deepseek-chat",
	}
	if err := endpointmanager.WriteEndpointConfig(ep2Dir, ep2Cfg); err != nil {
		t.Fatalf("创建端点 ep-deepseek 失败: %v", err)
	}

	// 同步所有端点
	result, err := SyncAllEndpoints(endpointsDir, tmpDir, nil)
	if err != nil {
		t.Fatalf("SyncAllEndpoints 返回错误: %v", err)
	}

	// 应同步两个端点
	if len(result) != 2 {
		t.Errorf("期望同步 2 个端点，实际 %d: %v", len(result), result)
	}

	// ep-openai 应同步到 claude 和 opencode
	ep1Synced, ok := result["ep-openai"]
	if !ok {
		t.Fatal("结果中应包含 ep-openai")
	}
	if len(ep1Synced) != 2 {
		t.Errorf("ep-openai 应同步 2 个 agent，实际 %d: %v", len(ep1Synced), ep1Synced)
	}

	// ep-deepseek 应同步到 4 个 agent
	ep2Synced, ok := result["ep-deepseek"]
	if !ok {
		t.Fatal("结果中应包含 ep-deepseek")
	}
	if len(ep2Synced) != 4 {
		t.Errorf("ep-deepseek 应同步 4 个 agent，实际 %d: %v", len(ep2Synced), ep2Synced)
	}
}

func TestSyncAllEndpoints_NonexistentDir(t *testing.T) {
	tmpDir := t.TempDir()
	_, err := SyncAllEndpoints(filepath.Join(tmpDir, "nonexistent"), tmpDir, nil)
	if err == nil {
		t.Fatal("不存在的目录应返回错误")
	}
	if !strings.Contains(err.Error(), "不存在") {
		t.Errorf("错误信息应包含'不存在'，got: %v", err)
	}
}

// ============================================================
// 辅助函数测试
// ============================================================

func TestKnownAgents(t *testing.T) {
	agents := KnownAgents()
	expected := map[string]bool{
		"claude": true, "opencode": true,
		"kimi": true, "deepseek-tui": true,
	}
	if len(agents) != len(expected) {
		t.Fatalf("期望 %d 个 agent，实际 %d: %v", len(expected), len(agents), agents)
	}
	for _, a := range agents {
		if !expected[a] {
			t.Errorf("意外的 agent 名称: %s", a)
		}
	}
}

func TestIsKnownAgent(t *testing.T) {
	tests := []struct {
		name  string
		agent string
		want  bool
	}{
		{name: "claude", agent: "claude", want: true},
		{name: "opencode", agent: "opencode", want: true},
		{name: "kimi", agent: "kimi", want: true},
		{name: "deepseek-tui", agent: "deepseek-tui", want: true},
		{name: "unknown", agent: "unknown-agent", want: false},
		{name: "empty", agent: "", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsKnownAgent(tt.agent)
			if got != tt.want {
				t.Errorf("IsKnownAgent(%q) = %v, 期望 %v", tt.agent, got, tt.want)
			}
		})
	}
}

func TestGetConfigFilePath(t *testing.T) {
	configDir := "/tmp/test-config"
	tests := []struct {
		name  string
		agent string
		want  string
	}{
		{name: "claude", agent: "claude", want: "/tmp/test-config/.claude/.env"},
		{name: "opencode", agent: "opencode", want: "/tmp/test-config/.opencode/.env"},
		{name: "kimi", agent: "kimi", want: "/tmp/test-config/.kimi/config.toml"},
		{name: "deepseek-tui", agent: "deepseek-tui", want: "/tmp/test-config/.deepseek/.env"},
		{name: "unknown", agent: "unknown-agent", want: ""},
		{name: "empty", agent: "", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetConfigFilePath(configDir, tt.agent)
			if got != tt.want {
				t.Errorf("GetConfigFilePath(%q, %q) = %q, 期望 %q", configDir, tt.agent, got, tt.want)
			}
		})
	}
}

// ============================================================
// FormatForAgent 扩展测试 — 覆盖所有可选字段分支
// ============================================================

func TestFormatForAgent_ClaudeAllFields(t *testing.T) {
	cfg := &endpointmanager.EndpointConfig{
		Provider:      "openai",
		URL:           "https://api.openai.com",
		Key:           "sk-test-all",
		Model:         "gpt-4",
		ModelOpus:     "gpt-4-opus",
		ModelSonnet:   "gpt-4-sonnet",
		ModelHaiku:    "gpt-4-haiku",
		ModelSubagent: "gpt-4-subagent",
	}
	content, err := FormatForAgent(cfg, "claude")
	if err != nil {
		t.Fatalf("FormatForAgent 返回错误: %v", err)
	}
	checks := []struct {
		field    string
		expected string
	}{
		{"OPENAI_API_KEY", "OPENAI_API_KEY=sk-test-all"},
		{"OPENAI_BASE_URL", "OPENAI_BASE_URL=https://api.openai.com"},
		{"MODEL", "MODEL=gpt-4"},
		{"MODEL_OPUS", "MODEL_OPUS=gpt-4-opus"},
		{"MODEL_SONNET", "MODEL_SONNET=gpt-4-sonnet"},
		{"MODEL_HAIKU", "MODEL_HAIKU=gpt-4-haiku"},
		{"MODEL_SUBAGENT", "MODEL_SUBAGENT=gpt-4-subagent"},
	}
	for _, c := range checks {
		if !strings.Contains(content, c.expected) {
			t.Errorf("输出应包含 %s，got:\n%s", c.expected, content)
		}
	}
}

func TestFormatForAgent_ClaudeMinimal(t *testing.T) {
	cfg := &endpointmanager.EndpointConfig{
		Provider: "openai",
		URL:      "https://api.openai.com",
		Key:      "sk-test",
	}
	content, err := FormatForAgent(cfg, "claude")
	if err != nil {
		t.Fatalf("FormatForAgent 返回错误: %v", err)
	}
	if strings.Contains(content, "MODEL=") {
		t.Errorf("无模型字段时不应包含 MODEL= 行，got:\n%s", content)
	}
	if strings.Contains(content, "MODEL_OPUS=") {
		t.Errorf("无模型字段时不应包含 MODEL_OPUS= 行")
	}
	if strings.Contains(content, "MODEL_SONNET=") {
		t.Errorf("无模型字段时不应包含 MODEL_SONNET= 行")
	}
	if strings.Contains(content, "MODEL_HAIKU=") {
		t.Errorf("无模型字段时不应包含 MODEL_HAIKU= 行")
	}
	if strings.Contains(content, "MODEL_SUBAGENT=") {
		t.Errorf("无模型字段时不应包含 MODEL_SUBAGENT= 行")
	}
}

func TestFormatForAgent_ClaudeAnthropicAllFields(t *testing.T) {
	cfg := &endpointmanager.EndpointConfig{
		Provider:      "anthropic",
		URL:           "https://api.anthropic.com",
		Key:           "sk-ant-test-all",
		Model:         "claude-3-opus",
		ModelOpus:     "claude-3-opus-20240229",
		ModelSonnet:   "claude-3-sonnet-20240229",
		ModelHaiku:    "claude-3-haiku-20240307",
		ModelSubagent: "claude-3-opus-20240229",
	}
	content, err := FormatForAgent(cfg, "claude")
	if err != nil {
		t.Fatalf("FormatForAgent 返回错误: %v", err)
	}
	if !strings.Contains(content, "ANTHROPIC_API_KEY=sk-ant-test-all") {
		t.Errorf("应包含 ANTHROPIC_API_KEY")
	}
	if !strings.Contains(content, "ANTHROPIC_BASE_URL=https://api.anthropic.com") {
		t.Errorf("应包含 ANTHROPIC_BASE_URL")
	}
	if !strings.Contains(content, "MODEL_OPUS=claude-3-opus-20240229") {
		t.Errorf("应包含 MODEL_OPUS")
	}
	if !strings.Contains(content, "MODEL_SUBAGENT=claude-3-opus-20240229") {
		t.Errorf("应包含 MODEL_SUBAGENT")
	}
	if strings.Contains(content, "OPENAI_API_KEY") {
		t.Errorf("anthropic provider 时不应包含 OPENAI_API_KEY")
	}
}

func TestFormatForAgent_OpencodeAllFields(t *testing.T) {
	cfg := &endpointmanager.EndpointConfig{
		Provider:    "openai",
		URL:         "https://api.openai.com",
		Key:         "sk-test",
		Model:       "gpt-4",
		ModelOpus:   "gpt-4-opus",
		ModelSonnet: "gpt-4-sonnet",
		ModelHaiku:  "gpt-4-haiku",
	}
	content, err := FormatForAgent(cfg, "opencode")
	if err != nil {
		t.Fatalf("FormatForAgent 返回错误: %v", err)
	}
	if !strings.Contains(content, "OPENAI_API_KEY") {
		t.Errorf("opencode 应包含 OPENAI_API_KEY")
	}
	if !strings.Contains(content, "MODEL_OPUS=gpt-4-opus") {
		t.Errorf("应包含 MODEL_OPUS")
	}
	if !strings.Contains(content, "MODEL_HAIKU=gpt-4-haiku") {
		t.Errorf("应包含 MODEL_HAIKU")
	}
}

func TestFormatForAgent_OpencodeMinimal(t *testing.T) {
	cfg := &endpointmanager.EndpointConfig{
		Provider: "openai",
		URL:      "https://api.openai.com",
		Key:      "sk-test",
	}
	content, err := FormatForAgent(cfg, "opencode")
	if err != nil {
		t.Fatalf("FormatForAgent 返回错误: %v", err)
	}
	if !strings.Contains(content, "OPENAI_API_KEY") {
		t.Errorf("opencode 应包含 OPENAI_API_KEY")
	}
	if !strings.Contains(content, "OPENAI_BASE_URL") {
		t.Errorf("opencode 应包含 OPENAI_BASE_URL")
	}
	if strings.Contains(content, "MODEL=") {
		t.Errorf("无模型字段时不应包含 MODEL= 行")
	}
}

func TestFormatForAgent_KimiMinimal(t *testing.T) {
	cfg := &endpointmanager.EndpointConfig{
		Provider: "openai",
		URL:      "https://api.openai.com",
		Key:      "sk-test",
	}
	content, err := FormatForAgent(cfg, "kimi")
	if err != nil {
		t.Fatalf("FormatForAgent 返回错误: %v", err)
	}
	if !strings.Contains(content, "[api]") {
		t.Errorf("kimi 应包含 [api] section")
	}
	if !strings.Contains(content, `key = "sk-test"`) {
		t.Errorf("kimi 应包含 key")
	}
	if !strings.Contains(content, `base_url = "https://api.openai.com"`) {
		t.Errorf("kimi 应包含 base_url")
	}
	if strings.Contains(content, "model =") {
		t.Errorf("无模型字段时不应包含 model = 行")
	}
}

func TestFormatForAgent_DeepseekTuiMinimal(t *testing.T) {
	cfg := &endpointmanager.EndpointConfig{
		Provider: "deepseek",
		URL:      "https://api.deepseek.com",
		Key:      "sk-ds",
	}
	content, err := FormatForAgent(cfg, "deepseek-tui")
	if err != nil {
		t.Fatalf("FormatForAgent 返回错误: %v", err)
	}
	if !strings.Contains(content, "DEEPSEEK_API_KEY=sk-ds") {
		t.Errorf("deepseek-tui 应包含 DEEPSEEK_API_KEY")
	}
	if !strings.Contains(content, "DEEPSEEK_BASE_URL") {
		t.Errorf("deepseek-tui 应包含 DEEPSEEK_BASE_URL")
	}
	if strings.Contains(content, "MODEL=") {
		t.Errorf("无模型字段时不应包含 MODEL= 行")
	}
}

// ============================================================
// quoteTomlString 单元测试
// ============================================================

func TestQuoteTomlString(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "normal string", input: "hello", want: `"hello"`},
		{name: "with double quotes", input: `he"llo`, want: `"he\"llo"`},
		{name: "with backslash", input: `he\llo`, want: `"he\\llo"`},
		{name: "with both", input: `he\"llo`, want: `"he\\\"llo"`},
		{name: "empty string", input: "", want: `""`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := quoteTomlString(tt.input)
			if got != tt.want {
				t.Errorf("quoteTomlString(%q) = %q, 期望 %q", tt.input, got, tt.want)
			}
		})
	}
}

// ============================================================
// WriteAgentConfig 扩展测试
// ============================================================

func TestWriteAgentConfig_Opencode(t *testing.T) {
	tmpDir := t.TempDir()
	content := "OPENAI_API_KEY=sk-test\nOPENAI_BASE_URL=https://api.openai.com\n"

	err := WriteAgentConfig(tmpDir, "opencode", content)
	if err != nil {
		t.Fatalf("WriteAgentConfig(opencode) 返回错误: %v", err)
	}

	filePath := filepath.Join(tmpDir, ".opencode", ".env")
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Fatal(".opencode/.env 文件未被创建")
	}

	info, err := os.Stat(filePath)
	if err != nil {
		t.Fatalf("无法 stat .opencode/.env: %v", err)
	}
	perm := info.Mode().Perm()
	if perm != 0600 {
		t.Errorf("文件权限为 %o, 期望 0600", perm)
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("读取 .opencode/.env 失败: %v", err)
	}
	if string(data) != content {
		t.Errorf("文件内容不匹配:\n期望:\n%s\n实际:\n%s", content, string(data))
	}
}

func TestWriteAgentConfig_Kimi(t *testing.T) {
	tmpDir := t.TempDir()
	content := "[api]\nkey = \"sk-test\"\nbase_url = \"https://api.openai.com\"\n"

	err := WriteAgentConfig(tmpDir, "kimi", content)
	if err != nil {
		t.Fatalf("WriteAgentConfig(kimi) 返回错误: %v", err)
	}

	filePath := filepath.Join(tmpDir, ".kimi", "config.toml")
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Fatal(".kimi/config.toml 文件未被创建")
	}

	info, err := os.Stat(filePath)
	if err != nil {
		t.Fatalf("无法 stat .kimi/config.toml: %v", err)
	}
	perm := info.Mode().Perm()
	if perm != 0600 {
		t.Errorf("文件权限为 %o, 期望 0600", perm)
	}
}

// ============================================================
// SyncEndpoint 扩展测试
// ============================================================

func TestSyncEndpoint_EmptyAgentsSlice(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &endpointmanager.EndpointConfig{
		Provider: "openai",
		URL:      "https://api.openai.com",
		Key:      "sk-test",
	}
	// 传递非 nil 的空切片，测试 len(agents)==0 分支
	synced, err := SyncEndpoint(cfg, tmpDir, []string{})
	if err != nil {
		t.Fatalf("SyncEndpoint 返回错误: %v", err)
	}
	// openai 应同步到 claude 和 opencode
	if len(synced) != 2 {
		t.Errorf("期望同步 2 个 agent，实际 %d: %v", len(synced), synced)
	}
}

func TestSyncEndpoint_UnknownProvider(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &endpointmanager.EndpointConfig{
		Provider: "unknown-provider",
		URL:      "https://example.com",
		Key:      "sk-test",
	}
	_, err := SyncEndpoint(cfg, tmpDir, nil)
	if err == nil {
		t.Fatal("未知 provider 应返回错误")
	}
	if !strings.Contains(err.Error(), "无可服务的 agent") {
		t.Errorf("错误信息应包含'无可服务的 agent'，got: %v", err)
	}
}

func TestSyncEndpoint_UnknownAgentInList(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &endpointmanager.EndpointConfig{
		Provider: "openai",
		URL:      "https://api.openai.com",
		Key:      "sk-test",
	}
	// agent 列表包含已知和未知 agent
	synced, err := SyncEndpoint(cfg, tmpDir, []string{"claude", "unknown-agent", "opencode"})
	if err == nil {
		t.Fatal("存在未知 agent 时应返回错误")
	}
	if !strings.Contains(err.Error(), "不支持的 agent") {
		t.Errorf("错误信息应包含'不支持的 agent'，got: %v", err)
	}
	// 已知 agent 仍应成功同步
	if len(synced) != 2 {
		t.Errorf("期望 2 个已知 agent 成功，实际 %d: %v", len(synced), synced)
	}
}

func TestSyncEndpoint_PartialWriteFailure(t *testing.T) {
	tmpDir := t.TempDir()

	// 将 .kimi 创建为文件，导致写入 .kimi/config.toml 时父目录创建失败
	kimiPath := filepath.Join(tmpDir, ".kimi")
	if err := os.WriteFile(kimiPath, []byte("not a directory"), 0644); err != nil {
		t.Fatalf("创建 .kimi 文件失败: %v", err)
	}

	cfg := &endpointmanager.EndpointConfig{
		Provider: "deepseek",
		URL:      "https://api.deepseek.com",
		Key:      "sk-ds-key",
	}

	synced, err := SyncEndpoint(cfg, tmpDir, []string{"claude", "kimi", "opencode"})
	if err == nil {
		t.Fatal("部分同步失败应返回错误")
	}
	if !strings.Contains(err.Error(), "部分 agent 写入失败") {
		t.Errorf("错误信息应包含'部分 agent 写入失败'，got: %v", err)
	}
	// claude 和 opencode 应成功，kimi 应失败
	if len(synced) != 2 {
		t.Errorf("期望 2 个 agent 成功，实际 %d: %v", len(synced), synced)
	}
	// 验证成功的 agent 配置文件已创建
	if _, err := os.Stat(filepath.Join(tmpDir, ".claude", ".env")); os.IsNotExist(err) {
		t.Error(".claude/.env 应存在")
	}
	if _, err := os.Stat(filepath.Join(tmpDir, ".opencode", ".env")); os.IsNotExist(err) {
		t.Error(".opencode/.env 应存在")
	}
}

// ============================================================
// ReadAndSyncEndpoint 测试
// ============================================================

func TestReadAndSyncEndpoint(t *testing.T) {
	tmpDir := t.TempDir()
	endpointDir := filepath.Join(tmpDir, "endpoints", "ep-openai")

	if err := endpointmanager.WriteEndpointConfig(endpointDir, &endpointmanager.EndpointConfig{
		Provider: "openai",
		URL:      "https://api.openai.com",
		Key:      "sk-test",
		Model:    "gpt-4",
	}); err != nil {
		t.Fatalf("创建端点配置失败: %v", err)
	}

	synced, err := ReadAndSyncEndpoint(endpointDir, tmpDir, nil)
	if err != nil {
		t.Fatalf("ReadAndSyncEndpoint 返回错误: %v", err)
	}
	if len(synced) != 2 {
		t.Errorf("期望同步 2 个 agent，实际 %d: %v", len(synced), synced)
	}
	// 验证配置文件已存在
	for _, relPath := range []string{".claude/.env", ".opencode/.env"} {
		if _, err := os.Stat(filepath.Join(tmpDir, relPath)); os.IsNotExist(err) {
			t.Errorf("配置文件 %s 未被创建", relPath)
		}
	}
}

func TestReadAndSyncEndpoint_EndpointNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	_, err := ReadAndSyncEndpoint(filepath.Join(tmpDir, "nonexistent"), tmpDir, nil)
	if err == nil {
		t.Fatal("不存在的端点配置应返回错误")
	}
	if !strings.Contains(err.Error(), "读取端点配置失败") {
		t.Errorf("错误信息应包含'读取端点配置失败'，got: %v", err)
	}
}

// ============================================================
// SyncAllEndpoints 扩展测试
// ============================================================

func TestSyncAllEndpoints_EmptyDir(t *testing.T) {
	tmpDir := t.TempDir()
	endpointsDir := filepath.Join(tmpDir, "endpoints")
	if err := os.MkdirAll(endpointsDir, 0755); err != nil {
		t.Fatalf("创建 endpoints 目录失败: %v", err)
	}

	result, err := SyncAllEndpoints(endpointsDir, tmpDir, nil)
	if err != nil {
		t.Fatalf("空端点目录应返回 nil error，got: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("空目录应返回空 map，got: %v", result)
	}
}

func TestSyncAllEndpoints_WithFiles(t *testing.T) {
	tmpDir := t.TempDir()
	endpointsDir := filepath.Join(tmpDir, "endpoints")
	if err := os.MkdirAll(endpointsDir, 0755); err != nil {
		t.Fatalf("创建 endpoints 目录失败: %v", err)
	}

	// 在 endpoints 目录下创建普通文件（非目录），应被跳过
	filePath := filepath.Join(endpointsDir, "not-a-dir")
	if err := os.WriteFile(filePath, []byte("not a directory"), 0644); err != nil {
		t.Fatalf("创建文件失败: %v", err)
	}

	// 同时创建一个有效的端点
	ep1Dir := filepath.Join(endpointsDir, "ep-openai")
	if err := endpointmanager.WriteEndpointConfig(ep1Dir, &endpointmanager.EndpointConfig{
		Provider: "openai",
		URL:      "https://api.openai.com",
		Key:      "sk-test",
	}); err != nil {
		t.Fatalf("创建端点配置失败: %v", err)
	}

	result, err := SyncAllEndpoints(endpointsDir, tmpDir, nil)
	if err != nil {
		t.Fatalf("SyncAllEndpoints 返回错误: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("期望 1 个端点，实际 %d: %v", len(result), result)
	}
	if _, ok := result["ep-openai"]; !ok {
		t.Errorf("结果中应包含 ep-openai，got: %v", result)
	}
}

func TestSyncAllEndpoints_PartialFailure(t *testing.T) {
	tmpDir := t.TempDir()
	endpointsDir := filepath.Join(tmpDir, "endpoints")
	if err := os.MkdirAll(endpointsDir, 0755); err != nil {
		t.Fatalf("创建 endpoints 目录失败: %v", err)
	}

	// 创建有效端点
	ep1Dir := filepath.Join(endpointsDir, "ep-openai")
	if err := endpointmanager.WriteEndpointConfig(ep1Dir, &endpointmanager.EndpointConfig{
		Provider: "openai",
		URL:      "https://api.openai.com",
		Key:      "sk-test",
	}); err != nil {
		t.Fatalf("创建端点配置失败: %v", err)
	}

	// 创建空目录（无 endpoint.env），同步时将会失败
	ep2Dir := filepath.Join(endpointsDir, "ep-invalid")
	if err := os.MkdirAll(ep2Dir, 0755); err != nil {
		t.Fatalf("创建空端点目录失败: %v", err)
	}

	result, err := SyncAllEndpoints(endpointsDir, tmpDir, nil)
	if err == nil {
		t.Fatal("部分端点失败应返回错误")
	}
	if !strings.Contains(err.Error(), "部分端点失败") {
		t.Errorf("错误信息应包含'部分端点失败'，got: %v", err)
	}
	// 有效端点应在结果中
	if _, ok := result["ep-openai"]; !ok {
		t.Errorf("ep-openai 应在结果中，got: %v", result)
	}
	// 无效端点不应在结果中
	if _, ok := result["ep-invalid"]; ok {
		t.Errorf("ep-invalid 不应在结果中，got: %v", result)
	}
}

func TestSyncAllEndpoints_ReadDirError(t *testing.T) {
	tmpDir := t.TempDir()
	endpointsDir := filepath.Join(tmpDir, "endpoints")

	// 创建普通文件替代目录，触发 ReadDir 返回非 IsNotExist 错误
	if err := os.WriteFile(endpointsDir, []byte("not a directory"), 0644); err != nil {
		t.Fatalf("创建文件失败: %v", err)
	}

	_, err := SyncAllEndpoints(endpointsDir, tmpDir, nil)
	if err == nil {
		t.Fatal("对文件调用 ReadDir 应返回错误")
	}
	if !strings.Contains(err.Error(), "读取端点配置目录失败") {
		t.Errorf("错误信息应包含'读取端点配置目录失败'，got: %v", err)
	}
}
