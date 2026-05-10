// Package configresolver 提供配置目录路径解析功能。
//
// 根据 -c 参数或默认值解析配置父目录路径，
// 并基于配置父目录导出 endpoint 存储路径、agent 配置路径等衍生路径。
// 所有相对路径会被转换为绝对路径。
package configresolver

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// DefaultConfigDirName 是未指定 -c 参数时的默认配置目录名称。
const DefaultConfigDirName = "coding-config"

// Resolver 负责解析和统一管理配置目录路径。
type Resolver struct {
	configDir string // 解析后的绝对路径
}

// New 创建一个新的路径解析器。
//
// 如果 configDir 为空字符串，则使用当前工作目录下的 coding-config 作为默认值。
// 如果 configDir 是相对路径，则将其转换为当前工作目录下的绝对路径。
// 该方法不会在文件系统上创建目录。
//
// 返回的 error 仅在获取当前工作目录失败时非 nil。
func New(configDir string) (*Resolver, error) {
	if configDir == "" {
		wd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("获取当前工作目录失败: %w", err)
		}
		configDir = filepath.Join(wd, DefaultConfigDirName)
	}

	absPath, err := filepath.Abs(configDir)
	if err != nil {
		return nil, fmt.Errorf("解析配置目录绝对路径失败: %w", err)
	}

	return &Resolver{configDir: absPath}, nil
}

// ConfigDir 返回解析后的配置目录绝对路径。
func (r *Resolver) ConfigDir() string {
	return r.configDir
}

// EndpointsDir 返回端点配置存储目录的绝对路径。
// 格式: <config-dir>/endpoints/
func (r *Resolver) EndpointsDir() string {
	return filepath.Join(r.configDir, "endpoints")
}

// AgentConfigDir 返回指定 agent 的配置目录绝对路径。
// 格式: <config-dir>/agents/<agent-name>/
func (r *Resolver) AgentConfigDir(agentName string) string {
	if agentName == "" {
		return filepath.Join(r.configDir, "agents")
	}
	return filepath.Join(r.configDir, "agents", agentName)
}

// EnsureEndpointsDir 确保端点配置存储目录存在，并返回其绝对路径。
// 该方法会按需创建目录，权限为 0755。
func (r *Resolver) EnsureEndpointsDir() (string, error) {
	dir := r.EndpointsDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("创建端点配置目录失败 %s: %w", dir, err)
	}
	return dir, nil
}

// EnsureConfigDir 确保配置目录存在，并返回其绝对路径。
// 该方法会按需创建目录，权限为 0755。
func (r *Resolver) EnsureConfigDir() (string, error) {
	if err := os.MkdirAll(r.configDir, 0755); err != nil {
		return "", fmt.Errorf("创建配置目录失败 %s: %w", r.configDir, err)
	}
	return r.configDir, nil
}

// IsDefaultConfigDir 判断当前配置目录是否为默认路径。
func (r *Resolver) IsDefaultConfigDir() bool {
	wd, err := os.Getwd()
	if err != nil {
		return false
	}
	defaultPath := filepath.Join(wd, DefaultConfigDirName)
	return r.configDir == defaultPath
}

// Resolve 是便捷方法，直接返回解析后的配置目录绝对路径。
// 等价于调用 New(configDir) 后获取 ConfigDir()。
func Resolve(configDir string) (string, error) {
	r, err := New(configDir)
	if err != nil {
		return "", err
	}
	return r.ConfigDir(), nil
}

// StandardErrors 定义 ConfigResolver 相关的标准错误。
var (
	// ErrGetwdFailed 表示无法获取当前工作目录。
	ErrGetwdFailed = errors.New("无法获取当前工作目录")
	// ErrAbsPathFailed 表示无法将路径转换为绝对路径。
	ErrAbsPathFailed = errors.New("无法将路径转换为绝对路径")
)
