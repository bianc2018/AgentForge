// Package endpointmanager 提供 LLM 端点配置的存储读写和管理。
//
// 端点存储格式为 <config-dir>/endpoints/<名称>/endpoint.env 文件（key=value 格式）。
// 文件权限始终设为 0600（仅文件所有者可读写），遵循 NFR-9 安全要求。
package endpointmanager

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// EndpointConfig 表示一个 LLM 端点的完整配置。
//
// 字段对应 endpoint.env 文件中的 KEY=VALUE 行。
// 可选字段（ModelOpus/ModelSonnet/ModelHaiku/ModelSubagent）可为空字符串。
type EndpointConfig struct {
	Provider      string `json:"provider"`       // LLM 服务商：deepseek/openai/anthropic
	URL           string `json:"url"`            // API 基础 URL，如 https://api.openai.com
	Key           string `json:"key"`            // API key（明文存储于宿主机文件系统）
	Model         string `json:"model"`          // 默认模型名，如 gpt-4
	ModelOpus     string `json:"model_opus"`     // Opus 模型名（可选）
	ModelSonnet   string `json:"model_sonnet"`   // Sonnet 模型名（可选）
	ModelHaiku    string `json:"model_haiku"`    // Haiku 模型名（可选）
	ModelSubagent string `json:"model_subagent"` // Subagent 模型名（可选）
}

// EndpointEnvFilename 是端点配置文件的固定文件名。
// 导出以供外部（如 cmd/ 包）引用该常量。
const EndpointEnvFilename = "endpoint.env"

// fieldKeys 是 endpoint.env 中所有字段的 key 列表（用于写入时的固定顺序）。
var fieldKeys = []string{
	"PROVIDER",
	"URL",
	"KEY",
	"MODEL",
	"MODEL_OPUS",
	"MODEL_SONNET",
	"MODEL_HAIKU",
	"MODEL_SUBAGENT",
}

// ParseEndpointEnv 解析 endpoint.env 文件内容并返回 EndpointConfig。
//
// 文件格式为 key=value 格式，每行一个字段，无引号包裹。
// 解析规则：
//   - 空白行和 # 开头的注释行被跳过
//   - 非 key=value 格式的格式错误行被跳过，不中断解析
//   - 缺少的字段（尤其是可选字段）被设置为空字符串
//   - 完全空的内容返回空配置，不返回错误
//
// 返回的 error 仅在内容扫描过程发生 I/O 错误时非 nil。
func ParseEndpointEnv(content string) (*EndpointConfig, error) {
	cfg := &EndpointConfig{}

	if strings.TrimSpace(content) == "" {
		return cfg, nil
	}

	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// 跳过空白行和注释行
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		// 跳过格式错误行（非 key=value）
		parts := strings.SplitN(trimmed, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		if key == "" {
			continue
		}

		switch key {
		case "PROVIDER":
			cfg.Provider = value
		case "URL":
			cfg.URL = value
		case "KEY":
			cfg.Key = value
		case "MODEL":
			cfg.Model = value
		case "MODEL_OPUS":
			cfg.ModelOpus = value
		case "MODEL_SONNET":
			cfg.ModelSonnet = value
		case "MODEL_HAIKU":
			cfg.ModelHaiku = value
		case "MODEL_SUBAGENT":
			cfg.ModelSubagent = value
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("扫描端点配置内容时出错: %w", err)
	}

	return cfg, nil
}

// MaskKey 对 API key 做掩码处理，用于安全显示。
//
// 掩码规则：
//   - key 长度 >= 12：前 8 字符 + "***" + 后 4 字符
//   - key 长度 < 12 且 > 0：显示首尾各 len/3 字符（至少 1 个）+ "***"
//   - 空 key：返回空字符串
//
// 示例：
//
//	MaskKey("sk-test-key-value") → "sk-test-***alue"
//	MaskKey("abcdefgh")         → "ab***gh"
//	MaskKey("")                 → ""
func MaskKey(key string) string {
	n := len(key)
	if n == 0 {
		return ""
	}

	const mask = "***"

	if n >= 12 {
		return key[:8] + mask + key[n-4:]
	}

	// 短 key（小于 12 字符）：显示首尾各约 1/3
	visible := n / 3
	if visible < 1 {
		visible = 1
	}
	return key[:visible] + mask + key[n-visible:]
}

// WriteEndpointConfig 将端点配置写入指定目录下的 endpoint.env 文件。
//
// 如果目录不存在，自动创建（权限 0755）。
// endpoint.env 文件权限设为 0600（仅文件所有者可读写），遵循 NFR-9。
// 写入内容为按固定顺序排列的 KEY=VALUE 行，每行一个字段。
//
// 返回的 error 包含操作上下文（目录路径、文件路径），
// 供上游遵循 NFR-16 错误信息格式。
func WriteEndpointConfig(endpointDir string, cfg *EndpointConfig) error {
	if err := os.MkdirAll(endpointDir, 0755); err != nil {
		return fmt.Errorf("创建端点配置目录失败 %s: %w", endpointDir, err)
	}

	filePath := filepath.Join(endpointDir, EndpointEnvFilename)

	// 按固定顺序生成 key=value 内容
	var b strings.Builder
	for _, key := range fieldKeys {
		value := fieldValue(cfg, key)
		b.WriteString(fmt.Sprintf("%s=%s\n", key, value))
	}

	if err := os.WriteFile(filePath, []byte(b.String()), 0600); err != nil {
		return fmt.Errorf("写入端点配置文件失败 %s: %w", filePath, err)
	}

	return nil
}

// ReadEndpointConfig 从指定目录读取 endpoint.env 文件并解析为 EndpointConfig。
//
// 文件不存在时返回清晰的错误信息（包含路径），供上游遵循 NFR-16 格式。
// 文件内容通过 ParseEndpointEnv 解析。
func ReadEndpointConfig(endpointDir string) (*EndpointConfig, error) {
	filePath := filepath.Join(endpointDir, EndpointEnvFilename)

	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("端点配置文件不存在: %s", filePath)
		}
		return nil, fmt.Errorf("读取端点配置文件失败 %s: %w", filePath, err)
	}

	cfg, err := ParseEndpointEnv(string(data))
	if err != nil {
		return nil, fmt.Errorf("解析端点配置文件失败 %s: %w", filePath, err)
	}

	return cfg, nil
}

// UpdateEndpointConfig 更新指定端点的部分配置字段。
//
// 只更新 updates 中非空的字段，空字段保持原值不变。
// 先读取现有配置文件，应用更新后写回，文件权限保持 0600（NFR-9）。
// 端点不存在时返回包含路径的错误信息（NFR-16 格式）。
func UpdateEndpointConfig(endpointDir string, updates *EndpointConfig) error {
	// 读取现有配置
	cfg, err := ReadEndpointConfig(endpointDir)
	if err != nil {
		return fmt.Errorf("更新端点配置失败: %w", err)
	}

	// 仅更新提供的非空值
	if updates.Provider != "" {
		cfg.Provider = updates.Provider
	}
	if updates.URL != "" {
		cfg.URL = updates.URL
	}
	if updates.Key != "" {
		cfg.Key = updates.Key
	}
	if updates.Model != "" {
		cfg.Model = updates.Model
	}
	if updates.ModelOpus != "" {
		cfg.ModelOpus = updates.ModelOpus
	}
	if updates.ModelSonnet != "" {
		cfg.ModelSonnet = updates.ModelSonnet
	}
	if updates.ModelHaiku != "" {
		cfg.ModelHaiku = updates.ModelHaiku
	}
	if updates.ModelSubagent != "" {
		cfg.ModelSubagent = updates.ModelSubagent
	}

	// 写回文件，保持 0600 权限
	if err := WriteEndpointConfig(endpointDir, cfg); err != nil {
		return fmt.Errorf("更新端点配置后写回失败: %w", err)
	}

	return nil
}

// RemoveEndpointConfig 递归删除指定端点的整个配置目录。
//
// 端点不存在时返回清晰的错误信息（包含路径），供上游遵循 NFR-16 格式。
func RemoveEndpointConfig(endpointDir string) error {
	// 先检查目录是否存在，不存在时返回特定错误
	if _, err := os.Stat(endpointDir); os.IsNotExist(err) {
		return fmt.Errorf("端点配置目录不存在: %s", endpointDir)
	}

	if err := os.RemoveAll(endpointDir); err != nil {
		return fmt.Errorf("删除端点配置目录失败 %s: %w", endpointDir, err)
	}

	return nil
}

// fieldValue 根据字段 key 从 EndpointConfig 中获取对应的值。
func fieldValue(cfg *EndpointConfig, key string) string {
	switch key {
	case "PROVIDER":
		return cfg.Provider
	case "URL":
		return cfg.URL
	case "KEY":
		return cfg.Key
	case "MODEL":
		return cfg.Model
	case "MODEL_OPUS":
		return cfg.ModelOpus
	case "MODEL_SONNET":
		return cfg.ModelSonnet
	case "MODEL_HAIKU":
		return cfg.ModelHaiku
	case "MODEL_SUBAGENT":
		return cfg.ModelSubagent
	default:
		return ""
	}
}

// chatCompletionRequest 是发送给 LLM API 的 chat/completions 请求体。
type chatCompletionRequest struct {
	Model       string              `json:"model"`
	Messages    []chatMessage       `json:"messages"`
	MaxTokens   int                 `json:"max_tokens,omitempty"`
}

// chatMessage 表示 chat/completions 请求中的一条消息。
type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// chatCompletionResponse 是 LLM API 返回的 chat/completions 响应体。
type chatCompletionResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// TestResult 表示端点连通性测试的结果。
type TestResult struct {
	// Latency 是 HTTP 请求的往返延迟。
	Latency time.Duration
	// Model 是 LLM API 返回的模型名称。
	Model string
	// ResponsePreview 是响应内容的简短摘要（前 120 个字符）。
	ResponsePreview string
}

// TestEndpoint 测试指定端点配置目录中的 LLM 端点连通性。
//
// 读取 endpoint.env 获取 URL 和 KEY，通过 Go net/http 向 {URL}/chat/completions
// 发送 POST 请求（含 Bearer 认证头）。请求超时时间设为 30 秒（NFR-4）。
//
// 请求成功时返回 TestResult，包含延迟和回复摘要。
// 请求失败（连接超时、DNS 解析失败、认证错误、端点不可达）时返回清晰的错误信息。
//
// 本函数仅依赖 Go 标准库的 net/http，无需外部 curl 依赖。
func TestEndpoint(endpointDir string) (*TestResult, error) {
	cfg, err := ReadEndpointConfig(endpointDir)
	if err != nil {
		return nil, fmt.Errorf("原因: 读取端点配置失败\n"+
			"上下文: 正在测试端点连通性，端点配置目录: %s\n"+
			"建议: 请确认端点已存在，使用 endpoint list 查看所有可用端点\n"+
			"错误详情: %s", endpointDir, err.Error())
	}

	if cfg.URL == "" {
		return nil, fmt.Errorf("原因: 端点 URL 为空\n"+
			"上下文: 端点配置文件 %s/endpoint.env 中的 URL 字段为空\n"+
			"建议: 请使用 endpoint set 命令设置有效的 API URL", endpointDir)
	}

	if cfg.Key == "" {
		return nil, fmt.Errorf("原因: 端点 API key 为空\n"+
			"上下文: 端点配置文件 %s/endpoint.env 中的 KEY 字段为空\n"+
			"建议: 请使用 endpoint set 命令设置有效的 API key", endpointDir)
	}

	// 构建请求 URL
	apiURL := strings.TrimRight(cfg.URL, "/") + "/chat/completions"

	// 选择要使用的模型
	model := cfg.Model
	if model == "" {
		model = cfg.ModelOpus
	}
	if model == "" {
		model = cfg.ModelSonnet
	}
	if model == "" {
		model = cfg.ModelHaiku
	}
	if model == "" {
		model = cfg.ModelSubagent
	}
	if model == "" {
		model = "default" // 没有指定模型时使用占位符，让 API 决定默认值
	}

	// 构建请求体
	reqBody := chatCompletionRequest{
		Model: model,
		Messages: []chatMessage{
			{Role: "user", Content: "Hello"},
		},
		MaxTokens: 5,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("原因: 序列化请求体失败\n"+
			"上下文: 正在构建发送到 %s 的 chat/completions 请求\n"+
			"建议: 这是一个内部错误，请报告给开发者\n"+
			"错误详情: %s", apiURL, err.Error())
	}

	// 创建 HTTP 客户端，超时 30 秒（NFR-4）
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// 创建 POST 请求
	req, err := http.NewRequest(http.MethodPost, apiURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("原因: 创建 HTTP 请求失败\n"+
			"上下文: 正在创建到 %s 的 POST 请求\n"+
			"建议: 请检查端点 URL 格式是否正确\n"+
			"错误详情: %s", apiURL, err.Error())
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.Key)

	// 执行请求并计时
	start := time.Now()
	resp, err := client.Do(req)
	latency := time.Since(start)

	if err != nil {
		// 判断错误类型以提供更精确的建议
		if isTimeoutError(err) {
			return nil, fmt.Errorf("原因: 请求超时（30 秒）\n"+
				"上下文: 在 %d 秒内无法连接到 %s\n"+
				"建议: 请检查网络连通性，确认 URL 可达，或增加超时时间",
				30, apiURL)
		}
		return nil, fmt.Errorf("原因: 连接失败\n"+
			"上下文: 无法连接到 %s\n"+
			"建议: 请检查 URL 是否正确，网络是否通畅，以及端点服务是否在运行\n"+
			"错误详情: %s", apiURL, err.Error())
	}
	defer resp.Body.Close()

	// 读取响应体
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("原因: 读取响应体失败\n"+
			"上下文: 从 %s 收到 HTTP %d 响应后读取响应体时出错\n"+
			"建议: 请检查网络连接是否稳定\n"+
			"错误详情: %s", apiURL, resp.StatusCode, err.Error())
	}

	// 检查 HTTP 状态码
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, fmt.Errorf("原因: 认证失败（HTTP %d）\n"+
			"上下文: 向 %s 发送请求后收到 HTTP %d 状态码\n"+
			"建议: 请检查 API key 是否正确，使用 endpoint set 命令更新 key",
			resp.StatusCode, apiURL, resp.StatusCode)
	}

	if resp.StatusCode != http.StatusOK {
		// 尝试解析错误响应体以获取更多上下文
		bodyPreview := string(respBody)
		if len(bodyPreview) > 200 {
			bodyPreview = bodyPreview[:200] + "..."
		}
		return nil, fmt.Errorf("原因: 端点返回 HTTP %d\n"+
			"上下文: 向 %s 发送请求后收到非预期状态码\n"+
			"建议: 请检查 URL 和请求参数是否正确\n"+
			"响应详情: %s", resp.StatusCode, apiURL, bodyPreview)
	}

	// 解析 JSON 响应
	var chatResp chatCompletionResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return nil, fmt.Errorf("原因: 解析响应 JSON 失败\n"+
			"上下文: 从 %s 收到 HTTP 200 响应后解析 JSON 时出错\n"+
			"建议: 端点返回的响应格式不符合 chat/completions 规范\n"+
			"错误详情: %s", apiURL, err.Error())
	}

	// 提取模型名
	resultModel := chatResp.Model
	if resultModel == "" {
		resultModel = model
	}

	// 提取响应预览（前 120 字符）
	preview := ""
	if len(chatResp.Choices) > 0 {
		preview = chatResp.Choices[0].Message.Content
	}
	if len(preview) > 120 {
		preview = preview[:120] + "..."
	}

	return &TestResult{
		Latency:         latency,
		Model:           resultModel,
		ResponsePreview: preview,
	}, nil
}

// isTimeoutError 判断错误是否为超时错误。
func isTimeoutError(err error) bool {
	if netErr, ok := err.(interface{ Timeout() bool }); ok {
		return netErr.Timeout()
	}
	return strings.Contains(err.Error(), "timeout") ||
		strings.Contains(err.Error(), "deadline exceeded")
}
