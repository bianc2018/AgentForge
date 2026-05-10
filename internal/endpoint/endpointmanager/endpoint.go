// Package endpointmanager 提供 LLM 端点配置的存储读写和管理。
//
// 端点存储格式为 <config-dir>/endpoints/<名称>/endpoint.env 文件（key=value 格式）。
// 文件权限始终设为 0600（仅文件所有者可读写），遵循 NFR-9 安全要求。
package endpointmanager

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
