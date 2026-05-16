package engine

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/docker/docker/api"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"

	"github.com/agent-forge/cli/internal/shared/dockerhelper"
)

// testAPIVersion 是与 Docker SDK api.DefaultVersion 一致的版本。
//
// Docker SDK 的版本协商策略是"仅降级不升级"：当 Ping 返回的
// API-Version >= SDK 默认版本时，SDK 仍使用默认版本。因此 mock
// server 必须注册与 SDK 默认版本一致的版本前缀路径。
var testAPIVersion = api.DefaultVersion

// setupMockDockerDaemon 创建模拟 Docker daemon 的测试服务器，
// 并返回已连接到该 mock 的 DistributionEngine 与清理函数。
func setupMockDockerDaemon(t *testing.T, handlers map[string]http.HandlerFunc) (*DistributionEngine, func()) {
	t.Helper()

	mux := http.NewServeMux()

	// /_ping — SDK 版本协商必需的端点（不限制 HTTP method）
	mux.HandleFunc("/_ping", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("API-Version", testAPIVersion)
		w.WriteHeader(http.StatusOK)
	})

	for pattern, handler := range handlers {
		mux.HandleFunc(pattern, handler)
	}

	server := httptest.NewServer(mux)
	host := "tcp://" + strings.TrimPrefix(server.URL, "http://")

	dClient, err := dockerhelper.NewClientWithOpts(
		client.WithHost(host),
		client.WithHTTPClient(server.Client()),
	)
	if err != nil {
		server.Close()
		t.Fatalf("创建测试 Docker 客户端失败: %v", err)
	}

	return New(dClient), func() {
		dClient.Close()
		server.Close()
	}
}

// =============================================================
//  New — 构造函数
// =============================================================

func TestNew(t *testing.T) {
	dClient, err := dockerhelper.NewClientWithOpts(
		client.WithHost("tcp://127.0.0.1:1"),
	)
	if err != nil {
		t.Fatalf("创建 Docker 客户端失败: %v", err)
	}
	defer dClient.Close()

	engine := New(dClient)
	if engine == nil {
		t.Fatal("New() 返回 nil")
	}
}

// =============================================================
//  Export — 导出
// =============================================================

func TestExport_Success(t *testing.T) {
	handlers := map[string]http.HandlerFunc{
		"GET /v" + testAPIVersion + "/images/json": func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]types.ImageSummary{
				{ID: "sha256:abc", RepoTags: []string{"test-img:latest"}, Size: 12345},
			})
		},
		"GET /v" + testAPIVersion + "/images/get": func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/x-tar")
			w.Write([]byte("mock-tar-data"))
		},
	}

	engine, cleanup := setupMockDockerDaemon(t, handlers)
	defer cleanup()

	outputPath := filepath.Join(t.TempDir(), "export-test.tar")
	if err := engine.Export(context.Background(), "test-img:latest", outputPath); err != nil {
		t.Fatalf("Export() 返回错误: %v", err)
	}

	info, err := os.Stat(outputPath)
	if err != nil {
		t.Fatalf("导出文件不存在: %v", err)
	}
	if info.Size() == 0 {
		t.Error("导出文件不应为空")
	}
}

func TestExport_NonexistentImage(t *testing.T) {
	handlers := map[string]http.HandlerFunc{
		"GET /v" + testAPIVersion + "/images/json": func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte("[]"))
		},
	}
	engine, cleanup := setupMockDockerDaemon(t, handlers)
	defer cleanup()

	err := engine.Export(context.Background(), "ghost-img:latest", filepath.Join(t.TempDir(), "out.tar"))
	if err == nil {
		t.Fatal("Export() 应返回错误")
	}
	if !strings.Contains(err.Error(), "不存在") {
		t.Errorf("错误信息应包含'不存在', 实际: %v", err)
	}
}

func TestExport_ImageExistsAPIError(t *testing.T) {
	handlers := map[string]http.HandlerFunc{
		"GET /v" + testAPIVersion + "/images/json": func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"message": "daemon error"})
		},
	}
	engine, cleanup := setupMockDockerDaemon(t, handlers)
	defer cleanup()

	err := engine.Export(context.Background(), "test-img:latest", filepath.Join(t.TempDir(), "out.tar"))
	if err == nil {
		t.Fatal("Export() 应返回错误")
	}
	if !strings.Contains(err.Error(), "检查镜像失败") {
		t.Errorf("错误信息应包含'检查镜像失败', 实际: %v", err)
	}
}

func TestExport_ImageSaveAPIError(t *testing.T) {
	handlers := map[string]http.HandlerFunc{
		"GET /v" + testAPIVersion + "/images/json": func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]types.ImageSummary{
				{ID: "sha256:abc", RepoTags: []string{"test-img:latest"}, Size: 12345},
			})
		},
		"GET /v" + testAPIVersion + "/images/get": func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"message": "save failed"})
		},
	}
	engine, cleanup := setupMockDockerDaemon(t, handlers)
	defer cleanup()

	err := engine.Export(context.Background(), "test-img:latest", filepath.Join(t.TempDir(), "out.tar"))
	if err == nil {
		t.Fatal("Export() 应返回错误")
	}
	if !strings.Contains(err.Error(), "导出镜像失败") {
		t.Errorf("错误信息应包含'导出镜像失败', 实际: %v", err)
	}
}

func TestExport_EmptyTar(t *testing.T) {
	// io.Copy 返回 (0, nil) → written == 0
	handlers := map[string]http.HandlerFunc{
		"GET /v" + testAPIVersion + "/images/json": func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]types.ImageSummary{
				{ID: "sha256:abc", RepoTags: []string{"test-img:latest"}, Size: 12345},
			})
		},
		"GET /v" + testAPIVersion + "/images/get": func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/x-tar")
			w.WriteHeader(http.StatusOK)
			// 不写入 body → response body 为 0 字节
		},
	}
	engine, cleanup := setupMockDockerDaemon(t, handlers)
	defer cleanup()

	err := engine.Export(context.Background(), "test-img:latest", filepath.Join(t.TempDir(), "out.tar"))
	if err == nil {
		t.Fatal("Export() 应返回错误")
	}
	if !strings.Contains(err.Error(), "导出文件为空") {
		t.Errorf("错误信息应包含'导出文件为空', 实际: %v", err)
	}
}

func TestExport_InvalidOutputPath(t *testing.T) {
	// os.Create 失败 → "创建输出文件失败"
	handlers := map[string]http.HandlerFunc{
		"GET /v" + testAPIVersion + "/images/json": func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]types.ImageSummary{
				{ID: "sha256:abc", RepoTags: []string{"test-img:latest"}, Size: 12345},
			})
		},
		"GET /v" + testAPIVersion + "/images/get": func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/x-tar")
			w.Write([]byte("mock-tar-data"))
		},
	}
	engine, cleanup := setupMockDockerDaemon(t, handlers)
	defer cleanup()

	err := engine.Export(context.Background(), "test-img:latest", "/nonexistent-dir-99999/out.tar")
	if err == nil {
		t.Fatal("Export() 应返回错误")
	}
	if !strings.Contains(err.Error(), "创建输出文件失败") {
		t.Errorf("错误信息应包含'创建输出文件失败', 实际: %v", err)
	}
}

func TestExport_IOCopyError(t *testing.T) {
	// io.Copy 期间 reader 返回错误
	handlers := map[string]http.HandlerFunc{
		"GET /v" + testAPIVersion + "/images/json": func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]types.ImageSummary{
				{ID: "sha256:abc", RepoTags: []string{"test-img:latest"}, Size: 12345},
			})
		},
		"GET /v" + testAPIVersion + "/images/get": func(w http.ResponseWriter, r *http.Request) {
			// ImageSave API 成功返回，但只写部分数据然后关闭连接
			// 使用 Hijack 模拟中途断连
			hj, ok := w.(http.Hijacker)
			if !ok {
				t.Fatal("server does not support hijacking")
			}
			conn, buf, err := hj.Hijack()
			if err != nil {
				t.Fatalf("hijack failed: %v", err)
			}
			defer conn.Close()
			buf.WriteString("HTTP/1.1 200 OK\r\nContent-Type: application/x-tar\r\nContent-Length: 30\r\n\r\n")
			buf.WriteString("partial-tar-data")
			buf.Flush()
			// 不写满 30 字节就关闭连接 → client 读到不完整响应
		},
	}
	engine, cleanup := setupMockDockerDaemon(t, handlers)
	defer cleanup()

	err := engine.Export(context.Background(), "test-img:latest", filepath.Join(t.TempDir(), "out.tar"))
	if err == nil {
		t.Fatal("Export() 应返回错误")
	}
	if !strings.Contains(err.Error(), "写入导出文件失败") {
		t.Errorf("错误信息应包含'写入导出文件失败', 实际: %v", err)
	}
}

// =============================================================
//  Import — 导入
// =============================================================

func TestImport_Success(t *testing.T) {
	handlers := map[string]http.HandlerFunc{
		"POST /v" + testAPIVersion + "/images/load": func(w http.ResponseWriter, r *http.Request) {
			// 读取并丢弃请求 body
			io.Copy(io.Discard, r.Body)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"message": "Loaded image"})
		},
	}
	engine, cleanup := setupMockDockerDaemon(t, handlers)
	defer cleanup()

	inputPath := filepath.Join(t.TempDir(), "import-test.tar")
	if err := os.WriteFile(inputPath, []byte("mock-tar-data"), 0644); err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}

	if err := engine.Import(context.Background(), inputPath); err != nil {
		t.Fatalf("Import() 返回错误: %v", err)
	}
}

func TestImport_NonexistentFile(t *testing.T) {
	// 无 Docker 调用，os.Stat 直接失败
	engine, cleanup := setupMockDockerDaemon(t, nil)
	defer cleanup()

	err := engine.Import(context.Background(), "/tmp/nonexistent-file-99999.tar")
	if err == nil {
		t.Fatal("Import() 应返回错误")
	}
	if !strings.Contains(err.Error(), "不存在") {
		t.Errorf("错误信息应包含'不存在', 实际: %v", err)
	}
}

func TestImport_OpenFileError(t *testing.T) {
	// os.Open 失败分支：文件存在但不可读（如权限不足）
	inputPath := filepath.Join(t.TempDir(), "restricted.tar")
	if err := os.WriteFile(inputPath, []byte("mock-tar-data"), 0000); err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}
	// 先验证本地 os.Open 确实会失败；若以 root 运行则跳过
	if f, err := os.Open(inputPath); err == nil {
		f.Close()
		t.Skip("当前用户为 root，无法测试无权限场景")
	}

	// 只需要能创建 engine 即可，os.Open 失败后不会调用 Docker
	dClient, err := dockerhelper.NewClientWithOpts(
		client.WithHost("tcp://127.0.0.1:1"),
	)
	if err != nil {
		t.Fatalf("创建 Docker 客户端失败: %v", err)
	}
	defer dClient.Close()

	engine := New(dClient)
	err = engine.Import(context.Background(), inputPath)
	if err == nil {
		t.Fatal("Import() 应返回错误")
	}
	if !strings.Contains(err.Error(), "打开导入文件失败") {
		t.Errorf("错误信息应包含'打开导入文件失败', 实际: %v", err)
	}
}

func TestImport_ImageLoadAPIError(t *testing.T) {
	handlers := map[string]http.HandlerFunc{
		"POST /v" + testAPIVersion + "/images/load": func(w http.ResponseWriter, r *http.Request) {
			io.Copy(io.Discard, r.Body)
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"message": "load failed"})
		},
	}
	engine, cleanup := setupMockDockerDaemon(t, handlers)
	defer cleanup()

	inputPath := filepath.Join(t.TempDir(), "import-test.tar")
	if err := os.WriteFile(inputPath, []byte("mock-tar-data"), 0644); err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}

	err := engine.Import(context.Background(), inputPath)
	if err == nil {
		t.Fatal("Import() 应返回错误")
	}
	if !strings.Contains(err.Error(), "导入镜像失败") {
		t.Errorf("错误信息应包含'导入镜像失败', 实际: %v", err)
	}
}
