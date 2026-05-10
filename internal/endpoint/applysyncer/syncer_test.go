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
