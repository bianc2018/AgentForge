//go:build perf

package cmd

import (
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// TestPerf_PT4_EndpointTestTimeout 测量 endpoint test 对不可达端点的超时断开时间。
//
// 测量内容: endpoint test <不可达端点> 的超时断开时间
// 阈值: ≤ 30 秒（NFR-4）
// 测量方法: 创建指向 localhost:1 的端点，通过 time 测量执行时间
// 执行次数: 3 次执行，取最大值
// 可追溯性: NFR-4
func TestPerf_PT4_EndpointTestTimeout(t *testing.T) {
	// 构建 CLI 二进制
	binaryPath := filepath.Join(t.TempDir(), "agent-forge")
	buildCmd := exec.Command("/tmp/go/bin/go", "build", "-o", binaryPath, ".")
	buildCmd.Dir = ".."
	output, err := buildCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("构建二进制失败: %v\nOutput: %s", err, output)
	}

	configDir := t.TempDir()

	// 设置: 创建指向不可达地址的端点
	addCmd := exec.Command(binaryPath, "endpoint", "add", "broken-ep",
		"--provider", "openai",
		"--url", "http://localhost:1",
		"--key", "sk-test-key",
		"--model", "gpt-4",
		"-c", configDir,
	)
	if addOut, addErr := addCmd.CombinedOutput(); addErr != nil {
		t.Fatalf("创建端点失败: %v\n%s", addErr, addOut)
	}

	// 执行: 测试不可达端点，测量超时时间
	var maxDuration time.Duration

	for i := 0; i < 3; i++ {
		start := time.Now()

		testCmd := exec.Command(binaryPath, "endpoint", "test", "broken-ep", "-c", configDir)
		_, _ = testCmd.CombinedOutput()
		duration := time.Since(start)

		t.Logf("第 %d 次执行: %v", i+1, duration)

		if duration > maxDuration {
			maxDuration = duration
		}
	}

	t.Logf("最大超时时间: %v", maxDuration)

	// 阈值: ≤ 30 秒（NFR-4）
	threshold := 30 * time.Second
	if maxDuration > threshold {
		t.Errorf("超时时间 %v 超过阈值 %v (NFR-4)", maxDuration, threshold)
	}

	if maxDuration < 100*time.Millisecond {
		t.Logf("警告: 超时时间 %v 过短，可能端点实际可达", maxDuration)
	}
}

// TestPerf_PT5_NonBuildCommandResponse 测量非构建类命令的响应时间。
//
// 测量内容: version、--help、endpoint list、endpoint providers 的响应时间
// 阈值: ≤ 1 秒（NFR-5）
// 测量方法: 通过 time 命令测量各命令执行时间
// 执行次数: 每个命令 5 次执行，取平均值
// 可追溯性: NFR-5
func TestPerf_PT5_NonBuildCommandResponse(t *testing.T) {
	binaryPath := filepath.Join(t.TempDir(), "agent-forge")
	buildCmd := exec.Command("/tmp/go/bin/go", "build", "-o", binaryPath, ".")
	buildCmd.Dir = ".."
	output, err := buildCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("构建二进制失败: %v\nOutput: %s", err, output)
	}

	configDir := t.TempDir()

	// 创建测试端点
	addCmd := exec.Command(binaryPath, "endpoint", "add", "my-ep",
		"--provider", "openai",
		"--url", "https://api.openai.com",
		"--key", "sk-test-key",
		"--model", "gpt-4",
		"-c", configDir,
	)
	if addOut, addErr := addCmd.CombinedOutput(); addErr != nil {
		t.Fatalf("创建端点失败: %v\n%s", addErr, addOut)
	}

	tests := []struct {
		name string
		args []string
	}{
		{"version", []string{"version"}},
		{"help", []string{"--help"}},
		{"endpoint list", []string{"endpoint", "list", "-c", configDir}},
		{"endpoint providers", []string{"endpoint", "providers"}},
	}

	threshold := 1 * time.Second

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var total time.Duration
			count := 5

			for i := 0; i < count; i++ {
				start := time.Now()
				cmd := exec.Command(binaryPath, tt.args...)
				_ = cmd.Run()
				total += time.Since(start)
			}

			avg := total / time.Duration(count)
			t.Logf("%s 平均响应时间: %v（%d 次执行）", tt.name, avg, count)

			if avg > threshold {
				t.Errorf("%s 平均响应时间 %v 超过阈值 %v (NFR-5)", tt.name, avg, threshold)
			}
		})
	}
}
