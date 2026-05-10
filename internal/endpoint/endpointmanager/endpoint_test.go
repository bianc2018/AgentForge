package endpointmanager

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// 确保 EndpointConfig 结构体可被正确创建和访问
func TestEndpointConfig_Defaults(t *testing.T) {
	cfg := &EndpointConfig{}
	if cfg.Provider != "" || cfg.URL != "" || cfg.Key != "" || cfg.Model != "" {
		t.Error("新建 EndpointConfig 应所有字段为空")
	}
	if cfg.ModelOpus != "" || cfg.ModelSonnet != "" || cfg.ModelHaiku != "" || cfg.ModelSubagent != "" {
		t.Error("新建 EndpointConfig 的可选字段应默认为空")
	}
}

// --- UT-8 基础: MaskKey ---

func TestMaskKey_NormalPath(t *testing.T) {
	key := "sk-test-key-value"
	want := "sk-test-***alue"
	got := MaskKey(key)
	if got != want {
		t.Errorf("MaskKey(%q) = %q, want %q", key, got, want)
	}
}

func TestMaskKey_ShortKey(t *testing.T) {
	key := "abcdefgh" // 8 chars, < 12
	got := MaskKey(key)
	if got == key {
		t.Error("短 key 掩码后不应与原 key 相同")
	}
	if !strings.Contains(got, "***") {
		t.Error("短 key 掩码结果应包含 ***")
	}
	if len(got) > len(key)+3 {
		t.Error("短 key 掩码结果不应超过 (原长度 + 3)")
	}
}

func TestMaskKey_EmptyKey(t *testing.T) {
	if got := MaskKey(""); got != "" {
		t.Errorf("MaskKey(\"\") = %q, want \"\"", got)
	}
}

func TestMaskKey_Exact12Chars(t *testing.T) {
	key := "123456789abc" // exactly 12 chars
	want := "12345678***9abc"
	got := MaskKey(key)
	if got != want {
		t.Errorf("MaskKey(%q) = %q, want %q", key, got, want)
	}
}

func TestMaskKey_SpecialChars(t *testing.T) {
	key := "sk-test_key@123"
	got := MaskKey(key)
	if got == key {
		t.Error("含特殊字符的 key 掩码后不应与原 key 相同")
	}
	if !strings.Contains(got, "***") {
		t.Error("含特殊字符的 key 掩码结果应包含 ***")
	}
}

// --- UT-9 基础: ParseEndpointEnv ---

func TestParseEndpointEnv_NormalPath(t *testing.T) {
	content := `PROVIDER=openai
URL=https://api.openai.com
KEY=sk-test-key-value
MODEL=gpt-4
MODEL_OPUS=gpt-4-32k
MODEL_SONNET=gpt-4-turbo
MODEL_HAIKU=gpt-3.5-turbo
MODEL_SUBAGENT=gpt-3.5-turbo
`
	cfg, err := ParseEndpointEnv(content)
	if err != nil {
		t.Fatalf("ParseEndpointEnv 返回错误: %v", err)
	}
	if cfg.Provider != "openai" {
		t.Errorf("Provider = %q, want %q", cfg.Provider, "openai")
	}
	if cfg.URL != "https://api.openai.com" {
		t.Errorf("URL = %q, want %q", cfg.URL, "https://api.openai.com")
	}
	if cfg.Key != "sk-test-key-value" {
		t.Errorf("Key = %q, want %q", cfg.Key, "sk-test-key-value")
	}
	if cfg.Model != "gpt-4" {
		t.Errorf("Model = %q, want %q", cfg.Model, "gpt-4")
	}
	if cfg.ModelOpus != "gpt-4-32k" {
		t.Errorf("ModelOpus = %q, want %q", cfg.ModelOpus, "gpt-4-32k")
	}
	if cfg.ModelSonnet != "gpt-4-turbo" {
		t.Errorf("ModelSonnet = %q, want %q", cfg.ModelSonnet, "gpt-4-turbo")
	}
	if cfg.ModelHaiku != "gpt-3.5-turbo" {
		t.Errorf("ModelHaiku = %q, want %q", cfg.ModelHaiku, "gpt-3.5-turbo")
	}
	if cfg.ModelSubagent != "gpt-3.5-turbo" {
		t.Errorf("ModelSubagent = %q, want %q", cfg.ModelSubagent, "gpt-3.5-turbo")
	}
}

func TestParseEndpointEnv_MissingOptionalFields(t *testing.T) {
	content := `PROVIDER=deepseek
URL=https://api.deepseek.com
KEY=sk-ds-key
MODEL=deepseek-chat
`
	cfg, err := ParseEndpointEnv(content)
	if err != nil {
		t.Fatalf("ParseEndpointEnv 返回错误: %v", err)
	}
	if cfg.Provider != "deepseek" {
		t.Errorf("Provider = %q, want %q", cfg.Provider, "deepseek")
	}
	if cfg.URL != "https://api.deepseek.com" {
		t.Errorf("URL = %q, want %q", cfg.URL, "https://api.deepseek.com")
	}
	// 缺少的可选字段应为空字符串
	if cfg.ModelOpus != "" {
		t.Errorf("ModelOpus 应为空，got %q", cfg.ModelOpus)
	}
	if cfg.ModelSonnet != "" {
		t.Errorf("ModelSonnet 应为空，got %q", cfg.ModelSonnet)
	}
	if cfg.ModelHaiku != "" {
		t.Errorf("ModelHaiku 应为空，got %q", cfg.ModelHaiku)
	}
	if cfg.ModelSubagent != "" {
		t.Errorf("ModelSubagent 应为空，got %q", cfg.ModelSubagent)
	}
}

func TestParseEndpointEnv_ExtraBlankLines(t *testing.T) {
	content := `PROVIDER=openai
URL=https://api.openai.com

KEY=sk-test-key


MODEL=gpt-4
`
	cfg, err := ParseEndpointEnv(content)
	if err != nil {
		t.Fatalf("ParseEndpointEnv 返回错误: %v", err)
	}
	if cfg.Provider != "openai" || cfg.URL != "https://api.openai.com" {
		t.Error("多余空白行后字段仍应被正确解析")
	}
	if cfg.Key != "sk-test-key" {
		t.Errorf("Key = %q, want %q", cfg.Key, "sk-test-key")
	}
	if cfg.Model != "gpt-4" {
		t.Errorf("Model = %q, want %q", cfg.Model, "gpt-4")
	}
}

func TestParseEndpointEnv_MalformedLines(t *testing.T) {
	content := `PROVIDER=anthropic
URL=https://api.anthropic.com
KEY=sk-anthropic-key
NOT_A_KEY_VALUE
=orphan-value
key-only
MODEL=claude-3-opus
`
	cfg, err := ParseEndpointEnv(content)
	if err != nil {
		t.Fatalf("ParseEndpointEnv 返回错误: %v", err)
	}
	if cfg.Provider != "anthropic" {
		t.Errorf("Provider = %q, want %q", cfg.Provider, "anthropic")
	}
	if cfg.URL != "https://api.anthropic.com" {
		t.Errorf("URL = %q, want %q", cfg.URL, "https://api.anthropic.com")
	}
	if cfg.Model != "claude-3-opus" {
		t.Errorf("Model = %q, want %q", cfg.Model, "claude-3-opus")
	}
}

func TestParseEndpointEnv_EmptyFile(t *testing.T) {
	cfg, err := ParseEndpointEnv("")
	if err != nil {
		t.Fatalf("ParseEndpointEnv 返回错误: %v", err)
	}
	if cfg == nil {
		t.Fatal("空文件不应返回 nil cfg")
	}
	if cfg.Provider != "" || cfg.URL != "" || cfg.Key != "" {
		t.Error("空文件应返回空配置")
	}
}

// --- 文件读写测试 (WriteEndpointConfig / ReadEndpointConfig) ---

func TestWriteAndReadEndpointConfig(t *testing.T) {
	// 在临时目录中测试文件读写
	tmpDir := t.TempDir()
	endpointDir := filepath.Join(tmpDir, "endpoints", "test-ep")

	cfg := &EndpointConfig{
		Provider:      "openai",
		URL:           "https://api.openai.com",
		Key:           "sk-test-key-value",
		Model:         "gpt-4",
		ModelOpus:     "gpt-4-32k",
		ModelSonnet:   "gpt-4-turbo",
		ModelHaiku:    "gpt-3.5-turbo",
		ModelSubagent: "gpt-3.5-turbo",
	}

	err := WriteEndpointConfig(endpointDir, cfg)
	if err != nil {
		t.Fatalf("WriteEndpointConfig 返回错误: %v", err)
	}

	// 验证文件是否存在
	filePath := filepath.Join(endpointDir, "endpoint.env")
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Fatal("endpoint.env 文件未被创建")
	}

	// 验证文件权限为 0600
	info, err := os.Stat(filePath)
	if err != nil {
		t.Fatalf("无法 stat endpoint.env: %v", err)
	}
	perm := info.Mode().Perm()
	if perm != 0600 {
		t.Errorf("文件权限为 %o, 期望 0600", perm)
	}

	// 验证目录权限为 0755
	dirInfo, err := os.Stat(endpointDir)
	if err != nil {
		t.Fatalf("无法 stat 端点目录: %v", err)
	}
	dirPerm := dirInfo.Mode().Perm()
	if dirPerm != 0755 {
		t.Errorf("目录权限为 %o, 期望 0755", dirPerm)
	}

	// 验证读取回来一致
	readCfg, err := ReadEndpointConfig(endpointDir)
	if err != nil {
		t.Fatalf("ReadEndpointConfig 返回错误: %v", err)
	}
	if readCfg.Provider != cfg.Provider {
		t.Errorf("Read Provider = %q, want %q", readCfg.Provider, cfg.Provider)
	}
	if readCfg.URL != cfg.URL {
		t.Errorf("Read URL = %q, want %q", readCfg.URL, cfg.URL)
	}
	if readCfg.Key != cfg.Key {
		t.Errorf("Read Key = %q, want %q", readCfg.Key, cfg.Key)
	}
	if readCfg.Model != cfg.Model {
		t.Errorf("Read Model = %q, want %q", readCfg.Model, cfg.Model)
	}
}

func TestWriteEndpointConfig_MinimalFields(t *testing.T) {
	tmpDir := t.TempDir()
	endpointDir := filepath.Join(tmpDir, "minimal-ep")

	cfg := &EndpointConfig{
		Provider: "deepseek",
		URL:      "https://api.deepseek.com",
		Key:      "sk-ds-key",
		Model:    "deepseek-chat",
	}

	err := WriteEndpointConfig(endpointDir, cfg)
	if err != nil {
		t.Fatalf("WriteEndpointConfig 返回错误: %v", err)
	}

	// 验证可选字段在写入时也被包含（为空值）
	filePath := filepath.Join(endpointDir, "endpoint.env")
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("读取 endpoint.env 失败: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "MODEL_OPUS=") {
		t.Error("写入内容应包含 MODEL_OPUS 字段")
	}
	if !strings.Contains(content, "MODEL_SUBAGENT=") {
		t.Error("写入内容应包含 MODEL_SUBAGENT 字段")
	}

	// 验证回读
	readCfg, err := ReadEndpointConfig(endpointDir)
	if err != nil {
		t.Fatalf("ReadEndpointConfig 返回错误: %v", err)
	}
	if readCfg.Provider != "deepseek" {
		t.Errorf("Provider = %q, want %q", readCfg.Provider, "deepseek")
	}
	if readCfg.ModelOpus != "" {
		t.Errorf("ModelOpus 应为空，got %q", readCfg.ModelOpus)
	}
}

func TestReadEndpointConfig_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	nonexistentDir := filepath.Join(tmpDir, "nonexistent-ep")

	_, err := ReadEndpointConfig(nonexistentDir)
	if err == nil {
		t.Fatal("读取不存在的端点应返回错误")
	}
	if !strings.Contains(err.Error(), "不存在") {
		t.Errorf("错误信息应包含'不存在'，got: %v", err)
	}
}
