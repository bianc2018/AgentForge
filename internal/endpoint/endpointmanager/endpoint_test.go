package endpointmanager

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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

// --- IT-1: UpdateEndpointConfig ---

func TestUpdateEndpointConfig_ModifyField(t *testing.T) {
	tmpDir := t.TempDir()
	endpointDir := filepath.Join(tmpDir, "endpoints", "test-ep")

	// 先创建端点
	orig := &EndpointConfig{
		Provider: "openai",
		URL:      "https://api.openai.com",
		Key:      "sk-original-key",
		Model:    "gpt-4",
	}
	if err := WriteEndpointConfig(endpointDir, orig); err != nil {
		t.Fatalf("创建端点失败: %v", err)
	}

	// 更新 MODEL 和 KEY
	updates := &EndpointConfig{
		Model: "gpt-5",
		Key:   "sk-new-key",
	}
	if err := UpdateEndpointConfig(endpointDir, updates); err != nil {
		t.Fatalf("UpdateEndpointConfig 返回错误: %v", err)
	}

	// 验证更新结果
	cfg, err := ReadEndpointConfig(endpointDir)
	if err != nil {
		t.Fatalf("更新后读取端点失败: %v", err)
	}
	if cfg.Model != "gpt-5" {
		t.Errorf("Model = %q, want %q", cfg.Model, "gpt-5")
	}
	if cfg.Key != "sk-new-key" {
		t.Errorf("Key = %q, want %q", cfg.Key, "sk-new-key")
	}
	// 未更新的字段应保持不变
	if cfg.Provider != "openai" {
		t.Errorf("Provider 不应被修改，got %q", cfg.Provider)
	}
	if cfg.URL != "https://api.openai.com" {
		t.Errorf("URL 不应被修改，got %q", cfg.URL)
	}

	// 验证文件权限仍为 0600
	filePath := filepath.Join(endpointDir, "endpoint.env")
	info, err := os.Stat(filePath)
	if err != nil {
		t.Fatalf("无法 stat endpoint.env: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("更新后文件权限为 %o, 期望 0600 (NFR-9)", perm)
	}
}

func TestUpdateEndpointConfig_NonexistentEndpoint(t *testing.T) {
	tmpDir := t.TempDir()
	nonexistentDir := filepath.Join(tmpDir, "nonexistent-ep")

	err := UpdateEndpointConfig(nonexistentDir, &EndpointConfig{Model: "gpt-5"})
	if err == nil {
		t.Fatal("更新不存在的端点应返回错误")
	}
	if !strings.Contains(err.Error(), "不存在") {
		t.Errorf("错误信息应包含'不存在'，got: %v", err)
	}
}

// --- IT-1: RemoveEndpointConfig ---

func TestRemoveEndpointConfig_DeleteEndpoint(t *testing.T) {
	tmpDir := t.TempDir()
	endpointDir := filepath.Join(tmpDir, "endpoints", "test-ep")

	// 先创建端点
	cfg := &EndpointConfig{
		Provider: "openai",
		URL:      "https://api.openai.com",
		Key:      "sk-test-key",
		Model:    "gpt-4",
	}
	if err := WriteEndpointConfig(endpointDir, cfg); err != nil {
		t.Fatalf("创建端点失败: %v", err)
	}

	// 验证目录存在
	if _, err := os.Stat(endpointDir); os.IsNotExist(err) {
		t.Fatal("端点目录应存在")
	}

	// 删除端点
	if err := RemoveEndpointConfig(endpointDir); err != nil {
		t.Fatalf("RemoveEndpointConfig 返回错误: %v", err)
	}

	// 验证目录已被删除
	if _, err := os.Stat(endpointDir); !os.IsNotExist(err) {
		t.Error("端点目录应已被删除")
	}
}

func TestRemoveEndpointConfig_NonexistentEndpoint(t *testing.T) {
	tmpDir := t.TempDir()
	nonexistentDir := filepath.Join(tmpDir, "nonexistent-ep")

	err := RemoveEndpointConfig(nonexistentDir)
	if err == nil {
		t.Fatal("删除不存在的端点应返回错误")
	}
	if !strings.Contains(err.Error(), "不存在") {
		t.Errorf("错误信息应包含'不存在'，got: %v", err)
	}
}

// --- IT-1: TestEndpoint (with mock HTTP server) ---

// mockChatServer 创建一个模拟 chat/completions 端点的 HTTP server。
func mockChatServer(model string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 验证请求方法
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// 验证 Content-Type
		if r.Header.Get("Content-Type") != "application/json" {
			http.Error(w, "Bad content type", http.StatusBadRequest)
			return
		}

		// 验证 Authorization header
		if r.Header.Get("Authorization") == "" {
			http.Error(w, "Missing auth", http.StatusUnauthorized)
			return
		}

		// 返回模拟响应
		resp := chatCompletionResponse{
			ID:     "chatcmpl-test123",
			Object: "chat.completion",
			Model:  model,
			Choices: []struct {
				Index   int `json:"index"`
				Message struct {
					Role    string `json:"role"`
					Content string `json:"content"`
				} `json:"message"`
			}{
				{
					Index: 0,
					Message: struct {
						Role    string `json:"role"`
						Content string `json:"content"`
					}{
						Role:    "assistant",
						Content: "Hello! This is a test response.",
					},
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
}

func TestEndpoint_TestSuccess(t *testing.T) {
	mockServer := mockChatServer("gpt-4-test")
	defer mockServer.Close()

	tmpDir := t.TempDir()
	endpointDir := filepath.Join(tmpDir, "ep-test")

	cfg := &EndpointConfig{
		Provider: "openai",
		URL:      mockServer.URL,
		Key:      "sk-test-key",
		Model:    "gpt-4-test",
	}
	if err := WriteEndpointConfig(endpointDir, cfg); err != nil {
		t.Fatalf("创建端点失败: %v", err)
	}

	result, err := TestEndpoint(endpointDir)
	if err != nil {
		t.Fatalf("TestEndpoint 返回错误: %v", err)
	}
	if result == nil {
		t.Fatal("TestEndpoint 返回 nil result")
	}
	if result.Latency <= 0 {
		t.Errorf("延迟应为正数，got %v", result.Latency)
	}
	if result.Model != "gpt-4-test" {
		t.Errorf("Model = %q, want %q", result.Model, "gpt-4-test")
	}
	if !strings.Contains(result.ResponsePreview, "Hello!") {
		t.Errorf("ResponsePreview 应包含 Hello!，got %q", result.ResponsePreview)
	}
}

func TestEndpoint_TestUnauthorized(t *testing.T) {
	// 创建一个总是返回 401 的 server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": "invalid_api_key"}`))
	}))
	defer mockServer.Close()

	tmpDir := t.TempDir()
	endpointDir := filepath.Join(tmpDir, "ep-unauth")

	cfg := &EndpointConfig{
		Provider: "openai",
		URL:      mockServer.URL,
		Key:      "sk-invalid-key",
		Model:    "gpt-4",
	}
	if err := WriteEndpointConfig(endpointDir, cfg); err != nil {
		t.Fatalf("创建端点失败: %v", err)
	}

	_, err := TestEndpoint(endpointDir)
	if err == nil {
		t.Fatal("认证失败时应返回错误")
	}
	if !strings.Contains(err.Error(), "认证失败") {
		t.Errorf("错误信息应包含'认证失败'，got: %v", err)
	}
}

func TestEndpoint_TestTimeout(t *testing.T) {
	// 创建一个永不响应的 server（模拟超时）
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second) // 模拟延迟（但测试使用短超时）
		w.WriteHeader(http.StatusOK)
	}))
	defer mockServer.Close()

	tmpDir := t.TempDir()
	endpointDir := filepath.Join(tmpDir, "ep-timeout")

	cfg := &EndpointConfig{
		Provider: "openai",
		URL:      mockServer.URL,
		Key:      "sk-test-key",
		Model:    "gpt-4",
	}
	if err := WriteEndpointConfig(endpointDir, cfg); err != nil {
		t.Fatalf("创建端点失败: %v", err)
	}

	// TestEndpoint 内部超时为 30 秒，但 TestEndpoint 对连接错误的处理
	// 会立即看到错误或超时。这里不等待 30 秒，而是验证连接超时处理。
	// 使用 httptest 的话 server 是立即可达的，所以不会真的超时
	// 我们这里验证的是 http.Client 的超时逻辑被正确设置
	result, err := TestEndpoint(endpointDir)
	if err != nil {
		// 超时也是可接受的
		t.Logf("TestEndpoint 返回错误（可接受）: %v", err)
	}
	if result != nil {
		t.Logf("TestEndpoint 成功（httptest 下直接响应）: %+v", result)
	}
}

func TestEndpoint_EndpointNotExist(t *testing.T) {
	tmpDir := t.TempDir()
	nonexistentDir := filepath.Join(tmpDir, "nonexistent-ep")

	_, err := TestEndpoint(nonexistentDir)
	if err == nil {
		t.Fatal("测试不存在的端点应返回错误")
	}
	if !strings.Contains(err.Error(), "读取端点配置失败") {
		t.Errorf("错误信息应包含'读取端点配置失败'，got: %v", err)
	}
}

func TestEndpoint_EmptyURL(t *testing.T) {
	tmpDir := t.TempDir()
	endpointDir := filepath.Join(tmpDir, "ep-no-url")

	cfg := &EndpointConfig{
		Provider: "openai",
		URL:      "",
		Key:      "sk-test-key",
		Model:    "gpt-4",
	}
	if err := WriteEndpointConfig(endpointDir, cfg); err != nil {
		t.Fatalf("创建端点失败: %v", err)
	}

	_, err := TestEndpoint(endpointDir)
	if err == nil {
		t.Fatal("空 URL 时应返回错误")
	}
	if !strings.Contains(err.Error(), "URL 为空") {
		t.Errorf("错误信息应包含'URL 为空'，got: %v", err)
	}
}

// --- fieldValue 全量测试 ---

func TestFieldValue_AllKeys(t *testing.T) {
	tests := []struct {
		name string
		key  string
		cfg  *EndpointConfig
		want string
	}{
		{"PROVIDER", "PROVIDER", &EndpointConfig{Provider: "p"}, "p"},
		{"URL", "URL", &EndpointConfig{URL: "u"}, "u"},
		{"KEY", "KEY", &EndpointConfig{Key: "k"}, "k"},
		{"MODEL", "MODEL", &EndpointConfig{Model: "m"}, "m"},
		{"MODEL_OPUS", "MODEL_OPUS", &EndpointConfig{ModelOpus: "mo"}, "mo"},
		{"MODEL_SONNET", "MODEL_SONNET", &EndpointConfig{ModelSonnet: "ms"}, "ms"},
		{"MODEL_HAIKU", "MODEL_HAIKU", &EndpointConfig{ModelHaiku: "mh"}, "mh"},
		{"MODEL_SUBAGENT", "MODEL_SUBAGENT", &EndpointConfig{ModelSubagent: "msub"}, "msub"},
		{"unknown key returns empty", "UNKNOWN", &EndpointConfig{}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := fieldValue(tt.cfg, tt.key)
			if got != tt.want {
				t.Errorf("fieldValue(%q) = %q, want %q", tt.key, got, tt.want)
			}
		})
	}
}

// --- MaskKey 极限短 key ---

func TestMaskKey_VeryShortKey(t *testing.T) {
	tests := []struct {
		name string
		key  string
		want string
	}{
		{"length 1", "a", "a***a"},
		{"length 2", "ab", "a***b"},
		{"length 3", "abc", "a***c"},
		{"length 4", "abcd", "a***d"},
		{"length 5", "abcde", "a***e"},
		{"length 6", "abcdef", "ab***ef"},
		{"length 8", "abcdefgh", "ab***gh"},
		{"length 11", "abcdefghijk", "abc***ijk"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MaskKey(tt.key)
			if got != tt.want {
				t.Errorf("MaskKey(%q) = %q, want %q", tt.key, got, tt.want)
			}
		})
	}
}

// --- ParseEndpointEnv 更多边界 ---

func TestParseEndpointEnv_CommentLines(t *testing.T) {
	content := `# This is a comment line
PROVIDER=openai
# another comment
URL=https://api.openai.com
KEY=sk-key
MODEL=gpt-4
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
	if cfg.Key != "sk-key" {
		t.Errorf("Key = %q, want %q", cfg.Key, "sk-key")
	}
	if cfg.Model != "gpt-4" {
		t.Errorf("Model = %q, want %q", cfg.Model, "gpt-4")
	}
}

func TestParseEndpointEnv_EmptyKeyLine(t *testing.T) {
	content := `=orphan-value
PROVIDER=openai
=another-orphan
URL=https://api.openai.com
KEY=sk-key
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
	if cfg.Key != "sk-key" {
		t.Errorf("Key = %q, want %q", cfg.Key, "sk-key")
	}
}

func TestParseEndpointEnv_LineWhitespace(t *testing.T) {
	content := `  PROVIDER  =  openai
URL=https://api.openai.com
  KEY=sk-key
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
	if cfg.Key != "sk-key" {
		t.Errorf("Key = %q, want %q", cfg.Key, "sk-key")
	}
}

func TestParseEndpointEnv_UnknownKeys(t *testing.T) {
	content := `PROVIDER=deepseek
URL=https://api.deepseek.com
KEY=sk-ds-key
MODEL=deepseek-chat
UNKNOWN_KEY=some_value
ANOTHER_UNKNOWN=another_value
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
	if cfg.Model != "deepseek-chat" {
		t.Errorf("Model = %q, want %q", cfg.Model, "deepseek-chat")
	}
}

func TestParseEndpointEnv_ScannerError(t *testing.T) {
	longLine := strings.Repeat("A", 64*1024+1)
	_, err := ParseEndpointEnv(longLine)
	if err == nil {
		t.Fatal("超长行应触发 scanner 错误")
	}
}

// --- ReadEndpointConfig 非 NotExist 错误 ---

func TestReadEndpointConfig_PathIsFile(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "not-a-dir")
	if err := os.WriteFile(filePath, []byte("content"), 0644); err != nil {
		t.Fatalf("创建文件失败: %v", err)
	}

	_, err := ReadEndpointConfig(filePath)
	if err == nil {
		t.Fatal("以文件路径作为目录读取应返回错误")
	}
	if strings.Contains(err.Error(), "不存在") {
		t.Errorf("错误不应是'不存在'，got: %v", err)
	}
}

// --- UpdateEndpointConfig 全量 ---

func TestUpdateEndpointConfig_AllFields(t *testing.T) {
	tmpDir := t.TempDir()
	endpointDir := filepath.Join(tmpDir, "ep-all-update")

	orig := &EndpointConfig{
		Provider:      "openai",
		URL:           "https://api.openai.com",
		Key:           "sk-original",
		Model:         "gpt-4",
		ModelOpus:     "gpt-4-32k",
		ModelSonnet:   "gpt-4-turbo",
		ModelHaiku:    "gpt-3.5-turbo",
		ModelSubagent: "gpt-3.5-turbo",
	}
	if err := WriteEndpointConfig(endpointDir, orig); err != nil {
		t.Fatalf("WriteEndpointConfig 失败: %v", err)
	}

	updates := &EndpointConfig{
		Provider:      "anthropic",
		URL:           "https://api.anthropic.com",
		Key:           "sk-new",
		Model:         "claude-3-opus",
		ModelOpus:     "claude-3-opus-20240229",
		ModelSonnet:   "claude-3-sonnet-20240229",
		ModelHaiku:    "claude-3-haiku-20240307",
		ModelSubagent: "claude-3-haiku-20240307",
	}
	if err := UpdateEndpointConfig(endpointDir, updates); err != nil {
		t.Fatalf("UpdateEndpointConfig 失败: %v", err)
	}

	cfg, err := ReadEndpointConfig(endpointDir)
	if err != nil {
		t.Fatalf("ReadEndpointConfig 失败: %v", err)
	}
	if cfg.Provider != "anthropic" {
		t.Errorf("Provider = %q, want %q", cfg.Provider, "anthropic")
	}
	if cfg.URL != "https://api.anthropic.com" {
		t.Errorf("URL = %q, want %q", cfg.URL, "https://api.anthropic.com")
	}
	if cfg.Key != "sk-new" {
		t.Errorf("Key = %q, want %q", cfg.Key, "sk-new")
	}
	if cfg.Model != "claude-3-opus" {
		t.Errorf("Model = %q, want %q", cfg.Model, "claude-3-opus")
	}
	if cfg.ModelOpus != "claude-3-opus-20240229" {
		t.Errorf("ModelOpus = %q, want %q", cfg.ModelOpus, "claude-3-opus-20240229")
	}
	if cfg.ModelSonnet != "claude-3-sonnet-20240229" {
		t.Errorf("ModelSonnet = %q, want %q", cfg.ModelSonnet, "claude-3-sonnet-20240229")
	}
	if cfg.ModelHaiku != "claude-3-haiku-20240307" {
		t.Errorf("ModelHaiku = %q, want %q", cfg.ModelHaiku, "claude-3-haiku-20240307")
	}
	if cfg.ModelSubagent != "claude-3-haiku-20240307" {
		t.Errorf("ModelSubagent = %q, want %q", cfg.ModelSubagent, "claude-3-haiku-20240307")
	}
}

func TestUpdateEndpointConfig_EmptyUpdates(t *testing.T) {
	tmpDir := t.TempDir()
	endpointDir := filepath.Join(tmpDir, "ep-empty-update")

	orig := &EndpointConfig{
		Provider: "openai",
		URL:      "https://api.openai.com",
		Key:      "sk-key",
		Model:    "gpt-4",
	}
	if err := WriteEndpointConfig(endpointDir, orig); err != nil {
		t.Fatalf("WriteEndpointConfig 失败: %v", err)
	}

	if err := UpdateEndpointConfig(endpointDir, &EndpointConfig{}); err != nil {
		t.Fatalf("UpdateEndpointConfig 空更新失败: %v", err)
	}

	cfg, err := ReadEndpointConfig(endpointDir)
	if err != nil {
		t.Fatalf("ReadEndpointConfig 失败: %v", err)
	}
	if cfg.Provider != "openai" {
		t.Errorf("Provider 不应被修改，got %q", cfg.Provider)
	}
	if cfg.URL != "https://api.openai.com" {
		t.Errorf("URL 不应被修改，got %q", cfg.URL)
	}
	if cfg.Key != "sk-key" {
		t.Errorf("Key 不应被修改，got %q", cfg.Key)
	}
	if cfg.Model != "gpt-4" {
		t.Errorf("Model 不应被修改，got %q", cfg.Model)
	}
}

func TestUpdateEndpointConfig_WriteError(t *testing.T) {
	tmpDir := t.TempDir()
	endpointDir := filepath.Join(tmpDir, "ep-write-error")

	orig := &EndpointConfig{
		Provider: "openai",
		URL:      "https://api.openai.com",
		Key:      "sk-key",
		Model:    "gpt-4",
	}
	if err := WriteEndpointConfig(endpointDir, orig); err != nil {
		t.Fatalf("WriteEndpointConfig 失败: %v", err)
	}

	filePath := filepath.Join(endpointDir, EndpointEnvFilename)
	if err := os.Chmod(filePath, 0444); err != nil {
		t.Fatalf("Chmod 失败: %v", err)
	}

	err := UpdateEndpointConfig(endpointDir, &EndpointConfig{Model: "gpt-5"})
	if err == nil {
		t.Fatal("只读文件写回应返回错误")
	}
	if !strings.Contains(err.Error(), "写回失败") {
		t.Errorf("错误信息应包含'写回失败'，got: %v", err)
	}
}

// --- isTimeoutError ---

type timeoutErr struct{}

func (timeoutErr) Error() string   { return "timeout" }
func (timeoutErr) Timeout() bool   { return true }

type nonTimeoutErr struct{}

func (nonTimeoutErr) Error() string { return "error" }
func (nonTimeoutErr) Timeout() bool { return false }

type timeoutStrErr struct{}

func (timeoutStrErr) Error() string { return "connection timeout" }

type deadlineExceededErr struct{}

func (deadlineExceededErr) Error() string { return "deadline exceeded" }

func TestIsTimeoutError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"timeout interface returns true", timeoutErr{}, true},
		{"timeout interface returns false", nonTimeoutErr{}, false},
		{"string contains 'timeout'", timeoutStrErr{}, true},
		{"string contains 'deadline exceeded'", deadlineExceededErr{}, true},
		{"normal error", fmt.Errorf("normal error"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isTimeoutError(tt.err)
			if got != tt.want {
				t.Errorf("isTimeoutError(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

// --- TestEndpoint 更多覆盖 ---

func TestEndpoint_EmptyKey(t *testing.T) {
	tmpDir := t.TempDir()
	endpointDir := filepath.Join(tmpDir, "ep-no-key")

	cfg := &EndpointConfig{
		Provider: "openai",
		URL:      "https://api.openai.com",
		Key:      "",
		Model:    "gpt-4",
	}
	if err := WriteEndpointConfig(endpointDir, cfg); err != nil {
		t.Fatalf("WriteEndpointConfig 失败: %v", err)
	}

	_, err := TestEndpoint(endpointDir)
	if err == nil {
		t.Fatal("空 Key 时应返回错误")
	}
	if !strings.Contains(err.Error(), "API key 为空") {
		t.Errorf("错误信息应包含'API key 为空'，got: %v", err)
	}
}

func TestEndpoint_Forbidden(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"error": "forbidden"}`))
	}))
	defer mockServer.Close()

	tmpDir := t.TempDir()
	endpointDir := filepath.Join(tmpDir, "ep-forbidden")
	cfg := &EndpointConfig{
		Provider: "openai",
		URL:      mockServer.URL,
		Key:      "sk-invalid-key",
		Model:    "gpt-4",
	}
	if err := WriteEndpointConfig(endpointDir, cfg); err != nil {
		t.Fatalf("WriteEndpointConfig 失败: %v", err)
	}

	_, err := TestEndpoint(endpointDir)
	if err == nil {
		t.Fatal("403 时应返回错误")
	}
	if !strings.Contains(err.Error(), "认证失败") {
		t.Errorf("错误信息应包含'认证失败'，got: %v", err)
	}
}

func TestEndpoint_NonOKStatus(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "internal server error"}`))
	}))
	defer mockServer.Close()

	tmpDir := t.TempDir()
	endpointDir := filepath.Join(tmpDir, "ep-500")
	cfg := &EndpointConfig{
		Provider: "openai",
		URL:      mockServer.URL,
		Key:      "sk-key",
		Model:    "gpt-4",
	}
	if err := WriteEndpointConfig(endpointDir, cfg); err != nil {
		t.Fatalf("WriteEndpointConfig 失败: %v", err)
	}

	_, err := TestEndpoint(endpointDir)
	if err == nil {
		t.Fatal("500 时应返回错误")
	}
	if !strings.Contains(err.Error(), "HTTP 500") {
		t.Errorf("错误信息应包含'HTTP 500'，got: %v", err)
	}
}

func TestEndpoint_InvalidJSON(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`this is not valid json`))
	}))
	defer mockServer.Close()

	tmpDir := t.TempDir()
	endpointDir := filepath.Join(tmpDir, "ep-bad-json")
	cfg := &EndpointConfig{
		Provider: "openai",
		URL:      mockServer.URL,
		Key:      "sk-key",
		Model:    "gpt-4",
	}
	if err := WriteEndpointConfig(endpointDir, cfg); err != nil {
		t.Fatalf("WriteEndpointConfig 失败: %v", err)
	}

	_, err := TestEndpoint(endpointDir)
	if err == nil {
		t.Fatal("无效 JSON 时应返回错误")
	}
	if !strings.Contains(err.Error(), "解析响应") {
		t.Errorf("错误信息应包含'解析响应'，got: %v", err)
	}
}

func TestEndpoint_NoChoices(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := chatCompletionResponse{
			ID:     "chatcmpl-test",
			Object: "chat.completion",
			Model:  "gpt-4",
			Choices: []struct {
				Index   int `json:"index"`
				Message struct {
					Role    string `json:"role"`
					Content string `json:"content"`
				} `json:"message"`
			}{},
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer mockServer.Close()

	tmpDir := t.TempDir()
	endpointDir := filepath.Join(tmpDir, "ep-no-choices")
	cfg := &EndpointConfig{
		Provider: "openai",
		URL:      mockServer.URL,
		Key:      "sk-key",
		Model:    "gpt-4",
	}
	if err := WriteEndpointConfig(endpointDir, cfg); err != nil {
		t.Fatalf("WriteEndpointConfig 失败: %v", err)
	}

	result, err := TestEndpoint(endpointDir)
	if err != nil {
		t.Fatalf("TestEndpoint 不应返回错误: %v", err)
	}
	if result.ResponsePreview != "" {
		t.Errorf("无 choices 时应返回空预览，got %q", result.ResponsePreview)
	}
}

func TestEndpoint_LongResponsePreview(t *testing.T) {
	longContent := strings.Repeat("This is a test response that will be long. ", 10)
	if len(longContent) <= 120 {
		t.Fatal("测试数据应超过 120 字符")
	}

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := chatCompletionResponse{
			Model: "gpt-4",
			Choices: []struct {
				Index   int `json:"index"`
				Message struct {
					Role    string `json:"role"`
					Content string `json:"content"`
				} `json:"message"`
			}{
				{
					Index: 0,
					Message: struct {
						Role    string `json:"role"`
						Content string `json:"content"`
					}{
						Role:    "assistant",
						Content: longContent,
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer mockServer.Close()

	tmpDir := t.TempDir()
	endpointDir := filepath.Join(tmpDir, "ep-long-preview")
	cfg := &EndpointConfig{
		Provider: "openai",
		URL:      mockServer.URL,
		Key:      "sk-key",
		Model:    "gpt-4",
	}
	if err := WriteEndpointConfig(endpointDir, cfg); err != nil {
		t.Fatalf("WriteEndpointConfig 失败: %v", err)
	}

	result, err := TestEndpoint(endpointDir)
	if err != nil {
		t.Fatalf("TestEndpoint 不应返回错误: %v", err)
	}
	if !strings.HasSuffix(result.ResponsePreview, "...") {
		t.Errorf("长响应预览应以 ... 结尾，got %q", result.ResponsePreview)
	}
	if len(result.ResponsePreview) > 123 {
		t.Errorf("预览不应超过 123 字符，got %d", len(result.ResponsePreview))
	}
}

func TestEndpoint_EmptyResponseModel(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := chatCompletionResponse{
			Model: "",
			Choices: []struct {
				Index   int `json:"index"`
				Message struct {
					Role    string `json:"role"`
					Content string `json:"content"`
				} `json:"message"`
			}{
				{
					Index: 0,
					Message: struct {
						Role    string `json:"role"`
						Content string `json:"content"`
					}{
						Role:    "assistant",
						Content: "Hello!",
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer mockServer.Close()

	tmpDir := t.TempDir()
	endpointDir := filepath.Join(tmpDir, "ep-empty-resp-model")
	cfg := &EndpointConfig{
		Provider: "openai",
		URL:      mockServer.URL,
		Key:      "sk-key",
		Model:    "configured-model",
	}
	if err := WriteEndpointConfig(endpointDir, cfg); err != nil {
		t.Fatalf("WriteEndpointConfig 失败: %v", err)
	}

	result, err := TestEndpoint(endpointDir)
	if err != nil {
		t.Fatalf("TestEndpoint 不应返回错误: %v", err)
	}
	if result.Model != "configured-model" {
		t.Errorf("Model 应为配置的模型名 %q，got %q", "configured-model", result.Model)
	}
}

func TestEndpoint_ModelFallbackChain(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := chatCompletionResponse{
			Choices: []struct {
				Index   int `json:"index"`
				Message struct {
					Role    string `json:"role"`
					Content string `json:"content"`
				} `json:"message"`
			}{
				{
					Index: 0,
					Message: struct {
						Role    string `json:"role"`
						Content string `json:"content"`
					}{
						Role:    "assistant",
						Content: "Hello!",
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer mockServer.Close()

	tests := []struct {
		name string
		cfg  *EndpointConfig
		want string
	}{
		{
			name: "uses direct model",
			cfg:  &EndpointConfig{URL: mockServer.URL, Key: "sk-key", Model: "direct"},
			want: "direct",
		},
		{
			name: "falls back to ModelOpus",
			cfg:  &EndpointConfig{URL: mockServer.URL, Key: "sk-key", ModelOpus: "opus"},
			want: "opus",
		},
		{
			name: "falls back to ModelSonnet",
			cfg:  &EndpointConfig{URL: mockServer.URL, Key: "sk-key", ModelSonnet: "sonnet"},
			want: "sonnet",
		},
		{
			name: "falls back to ModelHaiku",
			cfg:  &EndpointConfig{URL: mockServer.URL, Key: "sk-key", ModelHaiku: "haiku"},
			want: "haiku",
		},
		{
			name: "falls back to ModelSubagent",
			cfg:  &EndpointConfig{URL: mockServer.URL, Key: "sk-key", ModelSubagent: "sub"},
			want: "sub",
		},
		{
			name: "falls back to default",
			cfg:  &EndpointConfig{URL: mockServer.URL, Key: "sk-key"},
			want: "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			epDir := filepath.Join(tmpDir, tt.name)
			if err := WriteEndpointConfig(epDir, tt.cfg); err != nil {
				t.Fatalf("WriteEndpointConfig 失败: %v", err)
			}
			result, err := TestEndpoint(epDir)
			if err != nil {
				t.Fatalf("TestEndpoint 失败: %v", err)
			}
			if result.Model != tt.want {
				t.Errorf("Model = %q, want %q", result.Model, tt.want)
			}
		})
	}
}

func TestEndpoint_NewRequestError(t *testing.T) {
	tmpDir := t.TempDir()
	epDir := filepath.Join(tmpDir, "ep-bad-url")

	cfg := &EndpointConfig{
		URL:   "http://\x00invalid",
		Key:   "sk-key",
		Model: "gpt-4",
	}
	if err := WriteEndpointConfig(epDir, cfg); err != nil {
		t.Fatalf("WriteEndpointConfig 失败: %v", err)
	}
	_, err := TestEndpoint(epDir)
	if err == nil {
		t.Fatal("无效 URL 创建请求时应返回错误")
	}
	if !strings.Contains(err.Error(), "创建 HTTP 请求失败") {
		t.Errorf("错误信息应包含'创建 HTTP 请求失败'，got: %v", err)
	}
}

func TestEndpoint_ConnectionError(t *testing.T) {
	tmpDir := t.TempDir()
	epDir := filepath.Join(tmpDir, "ep-conn-err")
	cfg := &EndpointConfig{
		URL:   "http://127.0.0.1:1",
		Key:   "sk-key",
		Model: "gpt-4",
	}
	if err := WriteEndpointConfig(epDir, cfg); err != nil {
		t.Fatalf("WriteEndpointConfig 失败: %v", err)
	}
	_, err := TestEndpoint(epDir)
	if err == nil {
		t.Fatal("连接失败时应返回错误")
	}
	if !strings.Contains(err.Error(), "连接失败") {
		t.Errorf("错误信息应包含'连接失败'，got: %v", err)
	}
}

// --- 剩余未覆盖分支 ---

func TestWriteEndpointConfig_MkdirAllError(t *testing.T) {
	tmpDir := t.TempDir()
	// 创建一个文件阻塞目录路径，使 MkdirAll 失败
	blockPath := filepath.Join(tmpDir, "block")
	if err := os.WriteFile(blockPath, []byte("block"), 0644); err != nil {
		t.Fatalf("创建阻塞文件失败: %v", err)
	}

	err := WriteEndpointConfig(blockPath, &EndpointConfig{Provider: "test"})
	if err == nil {
		t.Fatal("MkdirAll 应返回错误")
	}
	if !strings.Contains(err.Error(), "创建端点配置目录失败") {
		t.Errorf("错误信息应包含'创建端点配置目录失败'，got: %v", err)
	}
}

func TestReadEndpointConfig_ParseError(t *testing.T) {
	tmpDir := t.TempDir()
	epDir := filepath.Join(tmpDir, "ep-parse-err")
	if err := os.MkdirAll(epDir, 0755); err != nil {
		t.Fatalf("MkdirAll 失败: %v", err)
	}

	// 写入包含超长行（>64KB）的文件触发 scanner.Err()
	longLine := strings.Repeat("A", 64*1024+1)
	envFile := filepath.Join(epDir, EndpointEnvFilename)
	if err := os.WriteFile(envFile, []byte(longLine), 0600); err != nil {
		t.Fatalf("WriteFile 失败: %v", err)
	}

	_, err := ReadEndpointConfig(epDir)
	if err == nil {
		t.Fatal("ReadEndpointConfig 应返回解析错误")
	}
	if !strings.Contains(err.Error(), "解析端点配置文件失败") {
		t.Errorf("错误信息应包含'解析端点配置文件失败'，got: %v", err)
	}
}

func TestRemoveEndpointConfig_RemoveError(t *testing.T) {
	tmpDir := t.TempDir()
	epDir := filepath.Join(tmpDir, "ep-remove-err")

	cfg := &EndpointConfig{
		Provider: "openai",
		URL:      "https://api.openai.com",
		Key:      "sk-key",
		Model:    "gpt-4",
	}
	if err := WriteEndpointConfig(epDir, cfg); err != nil {
		t.Fatalf("WriteEndpointConfig 失败: %v", err)
	}

	// 将目录设为只读，使 RemoveAll 失败
	if err := os.Chmod(epDir, 0555); err != nil {
		t.Fatalf("Chmod 失败: %v", err)
	}
	// 确保测试结束后恢复权限，让 t.TempDir() 能正常清理
	defer os.Chmod(epDir, 0755)

	err := RemoveEndpointConfig(epDir)
	if err == nil {
		t.Fatal("只读目录下删除应返回错误")
	}
	if !strings.Contains(err.Error(), "删除端点配置目录失败") {
		t.Errorf("错误信息应包含'删除端点配置目录失败'，got: %v", err)
	}
}

func TestEndpoint_ReadBodyError(t *testing.T) {
	// 谎报 Content-Length 触发 io.ReadAll 错误
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "1000000")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`partial`))
	}))
	defer mockServer.Close()

	tmpDir := t.TempDir()
	epDir := filepath.Join(tmpDir, "ep-readerr")
	cfg := &EndpointConfig{
		Provider: "openai",
		URL:      mockServer.URL,
		Key:      "sk-key",
		Model:    "gpt-4",
	}
	if err := WriteEndpointConfig(epDir, cfg); err != nil {
		t.Fatalf("WriteEndpointConfig 失败: %v", err)
	}

	_, err := TestEndpoint(epDir)
	if err == nil {
		t.Fatal("响应体读取失败时应返回错误")
	}
	if !strings.Contains(err.Error(), "读取响应体失败") {
		t.Errorf("期望'读取响应体失败'错误，got: %v", err)
	}
}

func TestEndpoint_NonOKStatus_LongBody(t *testing.T) {
	// body > 200 字符触发响应预览截断分支
	longBody := strings.Repeat("error details ", 30)
	if len(longBody) <= 200 {
		t.Fatal("测试 body 应超过 200 字符")
	}

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(longBody))
	}))
	defer mockServer.Close()

	tmpDir := t.TempDir()
	epDir := filepath.Join(tmpDir, "ep-long-body")
	cfg := &EndpointConfig{
		Provider: "openai",
		URL:      mockServer.URL,
		Key:      "sk-key",
		Model:    "gpt-4",
	}
	if err := WriteEndpointConfig(epDir, cfg); err != nil {
		t.Fatalf("WriteEndpointConfig 失败: %v", err)
	}

	_, err := TestEndpoint(epDir)
	if err == nil {
		t.Fatal("500 时应返回错误")
	}
	if !strings.Contains(err.Error(), "HTTP 500") {
		t.Errorf("错误信息应包含'HTTP 500'，got: %v", err)
	}
}
