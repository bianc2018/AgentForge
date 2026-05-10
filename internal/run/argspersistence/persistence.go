// Package argspersistence 提供 run 命令运行参数的持久化和恢复功能。
//
// 每次 run 命令执行成功后自动将所有运行参数持久化到配置目录下的 .last_args 文件，
// 在 -r/--recall 时从该文件读取并还原参数集。
//
// Args Persistence 位于运行层（Run Layer），依赖 Config Resolver 提供的配置目录路径。
package argspersistence

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/agent-forge/cli/internal/shared/argsparser"
)

// LastArgsFileName 是持久化参数文件的名称，位于配置目录下。
const LastArgsFileName = ".last_args"

// ErrFileNotFound 表示 .last_args 文件不存在。
// 当 run -r 但不存在历史参数文件时返回此错误，不会启动容器。
var ErrFileNotFound = errors.New(".last_args 文件不存在，无法恢复上次运行参数")

// Persistence 负责将 run 命令的运行参数持久化到 .last_args 文件，
// 以及在 -r/--recall 时从 .last_args 文件还原参数集。
//
// .last_args 文件位于配置目录（由 Config Resolver 解析）下，
// 格式为 key=value 键值对，每行一个字段。
type Persistence struct {
	configDir string // 配置目录的绝对路径
}

// New 创建一个新的 Args Persistence 实例。
//
// configDir 是配置目录的绝对路径（由 Config Resolver 解析），
// .last_args 文件将存储在该目录下。
func New(configDir string) *Persistence {
	return &Persistence{configDir: configDir}
}

// filePath 返回 .last_args 文件的完整路径。
func (p *Persistence) filePath() string {
	return filepath.Join(p.configDir, LastArgsFileName)
}

// Save 将运行参数持久化到 .last_args 文件。
//
// 保存的字段：
//   - AGENT: AI agent 名称，空表示 bash 模式
//   - PORTS: 端口映射列表，空格分隔多个映射
//   - MOUNTS: 只读目录挂载列表，空格分隔多个路径
//   - WORKDIR: 容器内工作目录
//   - ENVS: 环境变量列表，空格分隔多个 KEY=VALUE
//   - MODE: 运行模式（normal/docker/run）
//   - RUN_CMD: --run 指定的后台命令
//   - DIND: 是否 Docker-in-Docker 特权模式（true/false）
//
// 文件格式为 key=value 键值对，每行一个字段。
// 文件权限设为 0600，仅文件所有者可读写。
func (p *Persistence) Save(params argsparser.RunParams) error {
	// 确保配置目录存在
	if err := os.MkdirAll(p.configDir, 0755); err != nil {
		return fmt.Errorf("创建配置目录失败 %s: %w", p.configDir, err)
	}

	// 确定运行模式
	mode := "normal"
	if params.Docker {
		mode = "docker"
	}
	if params.RunCmd != "" {
		mode = "run"
	}

	// 构建文件内容
	dind := "false"
	if params.Docker {
		dind = "true"
	}

	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("AGENT=%s\n", params.Agent))
	builder.WriteString(fmt.Sprintf("PORTS=%s\n", strings.Join(params.Ports, " ")))
	builder.WriteString(fmt.Sprintf("MOUNTS=%s\n", strings.Join(params.Mounts, " ")))
	builder.WriteString(fmt.Sprintf("WORKDIR=%s\n", params.Workdir))
	builder.WriteString(fmt.Sprintf("ENVS=%s\n", strings.Join(params.Envs, " ")))
	builder.WriteString(fmt.Sprintf("MODE=%s\n", mode))
	builder.WriteString(fmt.Sprintf("RUN_CMD=%s\n", params.RunCmd))
	builder.WriteString(fmt.Sprintf("DIND=%s\n", dind))

	content := builder.String()

	// 写入文件，权限 0600
	if err := os.WriteFile(p.filePath(), []byte(content), 0600); err != nil {
		return fmt.Errorf("写入 .last_args 文件失败 %s: %w", p.filePath(), err)
	}

	return nil
}

// Load 从 .last_args 文件读取并还原运行参数集。
//
// 返回的 RunParams 包含除 Recall 和 Config 外的所有字段，
// 因为 Recall 是命令行模式而非配置参数，Config 已由配置目录决定。
//
// 如果 .last_args 文件不存在，返回 ErrFileNotFound。
// 如果文件格式错误（非关键字段缺失或格式异常），
// 缺失字段以空值填充，不返回错误。
func (p *Persistence) Load() (*argsparser.RunParams, error) {
	f, err := os.Open(p.filePath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrFileNotFound
		}
		return nil, fmt.Errorf("打开 .last_args 文件失败 %s: %w", p.filePath(), err)
	}
	defer f.Close()

	params := argsparser.DefaultRunParams()
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// 跳过空行和注释行
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// 按第一个 = 分割 key 和 value
		eqIdx := strings.Index(line, "=")
		if eqIdx == -1 {
			// 格式错误行，跳过不崩溃
			continue
		}

		key := strings.TrimSpace(line[:eqIdx])
		value := strings.TrimSpace(line[eqIdx+1:])

		switch key {
		case "AGENT":
			params.Agent = value
		case "PORTS":
			if value != "" {
				params.Ports = strings.Fields(value)
			}
		case "MOUNTS":
			if value != "" {
				params.Mounts = strings.Fields(value)
			}
		case "WORKDIR":
			params.Workdir = value
		case "ENVS":
			if value != "" {
				params.Envs = strings.Fields(value)
			}
		case "MODE":
			// MODE 是描述性字段，不映射到 RunParams 的独立字段
			// 运行模式的状态从 DIND 和 RUN_CMD 字段派生
		case "RUN_CMD":
			params.RunCmd = value
		case "DIND":
			params.Docker = value == "true"
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("读取 .last_args 文件失败 %s: %w", p.filePath(), err)
	}

	return &params, nil
}
