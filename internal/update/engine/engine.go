// Package engine 提供 CLI 自更新功能。
//
// 从 UPDATE_URL 或默认 URL 下载最新版本二进制，
// 备份当前版本，更新失败时自动回滚。
// 遵循 REQ-36 和 NFR-13 规范。
package engine

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

// defaultUpdateURL 是默认的更新 URL。
// 可通过 UPDATE_URL 环境变量覆盖。
const defaultUpdateURL = "https://github.com/agent-forge/cli/releases/latest/download/agent-forge-linux-amd64"

// SelfUpdateEngine 是自更新引擎。
type SelfUpdateEngine struct {
	// currentPath 是当前运行的可执行文件路径。
	currentPath string
	// updateURL 是下载最新版本的 URL。
	updateURL string
	// httpClient 是可 mock 的 HTTP 客户端。
	httpClient HTTPClient
	// osRename 是可 mock 的 os.Rename。
	osRename func(oldpath, newpath string) error
	// osStat 是可 mock 的 os.Stat。
	osStat func(name string) (os.FileInfo, error)
	// osRemove 是可 mock 的 os.Remove。
	osRemove func(name string) error
}

// HTTPClient 是可 mock 的 HTTP 客户端接口。
type HTTPClient interface {
	Get(url string) (*http.Response, error)
}

// Option 是 SelfUpdateEngine 的配置选项。
type Option func(*SelfUpdateEngine)

// WithHTTPClient 设置 HTTP 客户端。
func WithHTTPClient(client HTTPClient) Option {
	return func(e *SelfUpdateEngine) {
		e.httpClient = client
	}
}

// WithUpdateURL 设置更新 URL。
func WithUpdateURL(url string) Option {
	return func(e *SelfUpdateEngine) {
		e.updateURL = url
	}
}

// WithCurrentPath 设置当前可执行文件路径。
func WithCurrentPath(path string) Option {
	return func(e *SelfUpdateEngine) {
		e.currentPath = path
	}
}

// WithRename 设置可 mock 的 os.Rename。
func WithRename(fn func(oldpath, newpath string) error) Option {
	return func(e *SelfUpdateEngine) {
		e.osRename = fn
	}
}

// New 创建自更新引擎。
//
// opts 是可选配置参数，用于测试时 mock 依赖。
func New(opts ...Option) *SelfUpdateEngine {
	e := &SelfUpdateEngine{
		updateURL: defaultUpdateURL,
		httpClient: &http.Client{},
		osRename: os.Rename,
		osStat: os.Stat,
		osRemove: os.Remove,
	}

	for _, opt := range opts {
		opt(e)
	}

	// 如果未显式设置 currentPath，尝试自动获取
	if e.currentPath == "" {
		if path, err := os.Executable(); err == nil {
			e.currentPath = path
		}
	}

	// 优先使用环境变量 UPDATE_URL
	if envURL := os.Getenv("UPDATE_URL"); envURL != "" {
		e.updateURL = envURL
	}

	return e
}

// Update 执行完整的自更新流程。
//
// 步骤：
//  1. 备份当前二进制到临时文件
//  2. 下载最新版本到临时文件
//  3. 验证下载完整性
//  4. 用新版本替换当前二进制
//  5. 更新成功，删除备份
//
// 任一步骤失败时自动回滚。
func (e *SelfUpdateEngine) Update() error {
	if e.currentPath == "" {
		return fmt.Errorf("无法确定当前可执行文件路径")
	}

	backupPath := e.currentPath + ".bak"

	// 1. 备份当前二进制
	if err := e.copyFile(e.currentPath, backupPath); err != nil {
		return fmt.Errorf("备份当前版本失败: %w", err)
	}

	// 确保任何失败时回滚
	rollback := true
	defer func() {
		if rollback {
			// 从备份恢复
			_ = e.osRename(backupPath, e.currentPath)
		}
	}()

	// 2. 下载最新版本到临时目录
	tmpDir, err := os.MkdirTemp("", "agent-forge-update-*")
	if err != nil {
		return fmt.Errorf("创建临时目录失败: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	downloadPath := filepath.Join(tmpDir, "agent-forge-new")
	if err := e.download(e.updateURL, downloadPath); err != nil {
		return fmt.Errorf("下载最新版本失败: %w", err)
	}

	// 3. 验证下载文件非空
	info, err := e.osStat(downloadPath)
	if err != nil {
		return fmt.Errorf("验证下载文件失败: %w", err)
	}
	if info.Size() == 0 {
		return fmt.Errorf("下载文件为空")
	}

	// 4. 替换当前二进制
	if err := e.osRename(downloadPath, e.currentPath); err != nil {
		return fmt.Errorf("替换二进制文件失败: %w", err)
	}

	// 5. 设置可执行权限
	if err := os.Chmod(e.currentPath, 0755); err != nil {
		// 权限设置失败但文件已替换，回滚
		return fmt.Errorf("设置可执行权限失败: %w", err)
	}

	// 成功，禁用回滚，删除备份
	rollback = false
	_ = e.osRemove(backupPath)

	return nil
}

// download 下载指定 URL 的内容到目标路径。
func (e *SelfUpdateEngine) download(url, destPath string) error {
	resp, err := e.httpClient.Get(url)
	if err != nil {
		return fmt.Errorf("HTTP 请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP 请求返回状态码 %d", resp.StatusCode)
	}

	outFile, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("创建下载文件失败: %w", err)
	}
	defer outFile.Close()

	written, err := io.Copy(outFile, resp.Body)
	if err != nil {
		return fmt.Errorf("写入下载文件失败: %w", err)
	}
	if written == 0 {
		return fmt.Errorf("下载内容为空")
	}

	return nil
}

// copyFile 复制文件从 src 到 dst。
func (e *SelfUpdateEngine) copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	return destFile.Sync()
}
