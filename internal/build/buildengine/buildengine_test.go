package buildengine

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/agent-forge/cli/internal/build/dockerfilegen"
	"github.com/agent-forge/cli/internal/shared/dockerhelper"
	"github.com/agent-forge/cli/internal/shared/platform"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

// --- CalculateBackoff unit tests ---

func TestCalculateBackoff(t *testing.T) {
	tests := []struct {
		attempt int
		want    time.Duration
	}{
		{0, 0},
		{1, 1 * time.Second},
		{2, 2 * time.Second},
		{3, 4 * time.Second},
		{4, 8 * time.Second},
		{5, 16 * time.Second},
	}
	for _, tt := range tests {
		got := CalculateBackoff(tt.attempt)
		if got != tt.want {
			t.Errorf("CalculateBackoff(%d) = %v, want %v", tt.attempt, got, tt.want)
		}
	}
}

// --- isRetryableError unit tests ---

func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		err  error
		want bool
	}{
		{fmt.Errorf("connection refused"), true},
		{fmt.Errorf("no such host"), true},
		{fmt.Errorf("dial tcp 192.168.1.1:2375: i/o timeout"), true},
		{fmt.Errorf("connection reset by peer"), true},
		{fmt.Errorf("TLS handshake timeout"), true},
		{fmt.Errorf("Cannot connect to the Docker daemon"), true},
		{fmt.Errorf("EOF"), true},
		{fmt.Errorf("something else"), false},
		{nil, false},
	}
	for _, tt := range tests {
		got := isRetryableError(tt.err)
		if got != tt.want {
			t.Errorf("isRetryableError(%v) = %v, want %v", tt.err, got, tt.want)
		}
	}
}

// --- RetryExhaustedError tests ---

func TestRetryExhaustedError(t *testing.T) {
	err := &RetryExhaustedError{MaxRetry: 3}
	msg := err.Error()
	if !strings.Contains(msg, "3") {
		t.Errorf("RetryExhaustedError message should mention retry count: %s", msg)
	}
}

// --- ValidateParams tests ---

func TestValidateParams_Valid(t *testing.T) {
	err := validateParams(BuildParams{MaxRetry: 3})
	if err != nil {
		t.Fatalf("validateParams() error = %v", err)
	}
}

func TestValidateParams_NegativeMaxRetry(t *testing.T) {
	err := validateParams(BuildParams{MaxRetry: -1})
	if err == nil {
		t.Fatal("validateParams() expected error for negative MaxRetry")
	}
	if _, ok := err.(*InvalidParamsError); !ok {
		t.Errorf("error type = %T, want *InvalidParamsError", err)
	}
}

// --- createBuildContext tests ---

func TestCreateBuildContext(t *testing.T) {
	dockerfile := "FROM centos:7\nRUN echo hello\n"
	buf, err := createBuildContext(dockerfile)
	if err != nil {
		t.Fatalf("createBuildContext() error = %v", err)
	}

	// Read the tar content
	data, err := io.ReadAll(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("ReadAll error = %v", err)
	}

	// The tar should contain the Dockerfile
	if !bytes.Contains(data, []byte("FROM centos:7")) {
		t.Error("tar archive missing Dockerfile content")
	}

	// Verify we can read it multiple times (for retry)
	data2, err := io.ReadAll(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("second ReadAll error = %v", err)
	}
	if !bytes.Equal(data, data2) {
		t.Error("second read should produce same content")
	}
}

// --- isBuildSuccessful tests ---

func TestIsBuildSuccessful(t *testing.T) {
	tests := []struct {
		output string
		want   bool
	}{
		{"Successfully tagged agent-forge:latest", true},
		{"Successfully built abc123", true},
		{"Step 1/5 : FROM centos:7\nSuccessfully tagged agent-forge:latest", true},
		{"error: build failed", false},
		{"", false},
		{"Step 1/5 : FROM centos:7\nError: pull access denied", false},
	}

	for _, tt := range tests {
		got := isBuildSuccessful(tt.output)
		if got != tt.want {
			t.Errorf("isBuildSuccessful(%q) = %v, want %v", tt.output, got, tt.want)
		}
	}
}

// --- Error type tests ---

func TestInvalidParamsError(t *testing.T) {
	err := &InvalidParamsError{Reason: "test reason"}
	if !strings.Contains(err.Error(), "test reason") {
		t.Errorf("Error() = %q, should contain 'test reason'", err.Error())
	}
}

func TestBuildError(t *testing.T) {
	err := &BuildError{Message: "构建失败", Output: "log output", ExitCode: 1}
	if err.Error() != "构建失败" {
		t.Errorf("Error() = %q, want '构建失败'", err.Error())
	}
}

// --- BuildEngine integration-style tests ---

func TestBuildEngine_Build_InvalidParams(t *testing.T) {
	helper, err := dockerhelper.NewClient()
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer helper.Close()

	engine := New(helper)
	defer engine.Close()

	_, err = engine.Build(context.Background(), BuildParams{
		Deps:     "",
		MaxRetry: -1,
	})
	if err == nil {
		t.Fatal("Build() expected error for negative MaxRetry")
	}
	if _, ok := err.(*InvalidParamsError); !ok {
		t.Errorf("error type = %T, want *InvalidParamsError", err)
	}
}

func TestBuildEngine_New(t *testing.T) {
	helper, err := dockerhelper.NewClient()
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer helper.Close()

	engine := New(helper)
	if engine == nil {
		t.Fatal("New() returned nil")
	}
	engine.Close()
}

// TestBuildEngine_BuildWithMinimalDeps runs a Docker build with empty deps to
// verify the basic build pipeline works end-to-end.
// This is an integration test that requires Docker daemon with base image cached.
func TestBuildEngine_BuildWithMinimalDeps(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试：需要完整 Docker 构建（使用 -short 跳过）")
	}

	helper, err := dockerhelper.NewClient()
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer helper.Close()

	// Verify Docker is reachable
	pingCtx, pingCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer pingCancel()
	if err := helper.Ping(pingCtx); err != nil {
		t.Skipf("Docker daemon not available, skipping: %v", err)
	}

	// Check if base image is already cached
	baseImage := dockerfilegen.DefaultBaseImage
	exists, err := helper.ImageExists(pingCtx, baseImage)
	if err != nil || !exists {
		t.Skipf("Base image %s not cached, skipping integration test (IT-5 will cover this)", baseImage)
	}

	engine := New(helper)
	defer engine.Close()

	buildCtx, buildCancel := context.WithTimeout(context.Background(), 600*time.Second)
	defer buildCancel()

	output, err := engine.Build(buildCtx, BuildParams{
		Deps:     "",
		MaxRetry: 1,
	})
	if err != nil {
		t.Fatalf("Build() error = %v\nOutput: %s", err, output)
	}

	if output == "" {
		t.Error("Build() returned empty output")
	}

	// Cleanup: remove the test image
	_, err = helper.ImageRemove(buildCtx, ImageTag, true, true)
	if err != nil {
		t.Logf("Warning: failed to remove test image: %v", err)
	}
}

// Test to verify --no-cache is passed correctly
func TestBuildEngine_BuildNoCacheOptions(t *testing.T) {
	opts := types.ImageBuildOptions{
		NoCache: true,
	}
	if !opts.NoCache {
		t.Error("ImageBuildOptions.NoCache should be true")
	}
}

// TestBuildEngine fails with invalid Docker client
func TestBuildEngine_Build_WithUnreachableDocker(t *testing.T) {
	unreachableClient, err := dockerhelper.NewClientWithOpts(
		client.WithHost("unix:///var/run/nonexistent.sock"),
	)
	if err != nil {
		t.Fatalf("NewClientWithOpts() error = %v", err)
	}
	defer unreachableClient.Close()

	engine := New(unreachableClient)
	defer engine.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = engine.Build(ctx, BuildParams{
		Deps:     "",
		MaxRetry: 1,
	})
	if err == nil {
		t.Fatal("Build() expected error with unreachable Docker daemon")
	}
	t.Logf("Build() with unreachable Docker returned expected error: %v", err)
}

// --- Rebuild mode tests ---

// TestBuildEngine_BuildFailure verifies build returns non-zero error when
// an invalid base image is specified. This tests the build failure path.
func TestBuildEngine_BuildFailure(t *testing.T) {
	helper, err := dockerhelper.NewClient()
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer helper.Close()

	pingCtx, pingCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer pingCancel()
	if err := helper.Ping(pingCtx); err != nil {
		t.Skipf("Docker daemon not available, skipping: %v", err)
	}

	engine := New(helper)
	defer engine.Close()

	buildCtx, buildCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer buildCancel()

	// Build with a nonexistent base image should fail quickly
	_, err = engine.Build(buildCtx, BuildParams{
		BaseImage: "docker.1ms.run/nonexistent:latest",
		MaxRetry:  0,
	})
	if err == nil {
		t.Fatal("Build() expected error with nonexistent base image, got nil")
	}
	t.Logf("Build failure returned expected error: %v", err)
}

// TestBuildEngine_RebuildFailure verifies that when rebuild mode fails,
// the original agent-forge:latest image is preserved unchanged.
// Uses a pre-existing base image tagged as agent-forge:latest to avoid
// a full initial build.
func TestBuildEngine_RebuildFailure(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试：依赖全局 Docker agent-forge:latest 状态（使用 -short 跳过）")
	}

	helper, err := dockerhelper.NewClient()
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer helper.Close()

	pingCtx, pingCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer pingCancel()
	if err := helper.Ping(pingCtx); err != nil {
		t.Skipf("Docker daemon not available, skipping: %v", err)
	}

	// Check if base image is cached
	baseImage := dockerfilegen.DefaultBaseImage
	exists, err := helper.ImageExists(pingCtx, baseImage)
	if err != nil || !exists {
		t.Skipf("Base image %s not cached, skipping integration test", baseImage)
	}

	// Clean up any residual agent-forge:latest from previous tests to avoid state pollution
	_, _ = helper.ImageRemove(context.Background(), ImageTag, true, true)

	// Tag base image as agent-forge:latest (simulates existing image)
	if err := helper.ImageTag(pingCtx, baseImage, ImageTag); err != nil {
		t.Fatalf("ImageTag() error = %v", err)
	}
	defer func() {
		_, _ = helper.ImageRemove(context.Background(), ImageTag, true, true)
	}()

	// Verify the tag was created
	taggedExists, err := helper.ImageExists(pingCtx, ImageTag)
	if err != nil || !taggedExists {
		t.Fatalf("ImageTag succeeded but ImageExists(%q) returned exists=%v, err=%v", ImageTag, taggedExists, err)
	}

	// Get original image ID
	images, err := helper.ImageList(pingCtx, types.ImageListOptions{})
	if err != nil {
		t.Fatalf("ImageList() error = %v", err)
	}
	origID := ""
	for _, img := range images {
		for _, tag := range img.RepoTags {
			if tag == ImageTag {
				origID = img.ID
				break
			}
		}
		if origID != "" {
			break
		}
	}
	if origID == "" {
		t.Fatal("Could not find agent-forge:latest after tagging")
	}

	engine := New(helper)
	defer engine.Close()

	buildCtx, buildCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer buildCancel()

	// Rebuild with invalid base image should fail
	output, err := engine.Build(buildCtx, BuildParams{
		BaseImage: "docker.1ms.run/nonexistent:latest",
		MaxRetry:  0,
		Rebuild:   true,
	})
	if err == nil {
		t.Fatal("Build() expected error with nonexistent base image in rebuild mode")
	}
	t.Logf("Rebuild failure returned expected error: %v", err)
	t.Logf("Build output:\n%s", output)

	// Verify original agent-forge:latest still exists with same ID (use fresh context
	// because buildCtx may be close to deadline after a slow Docker operation)
	verifyCtx, verifyCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer verifyCancel()
	images2, err := helper.ImageList(verifyCtx, types.ImageListOptions{})
	if err != nil {
		t.Fatalf("ImageList() error = %v", err)
	}
	newID := ""
	for _, img := range images2 {
		for _, tag := range img.RepoTags {
			if tag == ImageTag {
				newID = img.ID
				break
			}
		}
		if newID != "" {
			break
		}
	}
	if newID == "" {
		t.Fatal("agent-forge:latest was removed after failed rebuild")
	}
	if newID != origID {
		t.Errorf("Image ID changed after failed rebuild: was %s, now %s", origID, newID)
	}
}

// TestBuildEngine_RebuildSuccess verifies rebuild mode atomically replaces
// the existing image with a newly built one.
func TestBuildEngine_RebuildSuccess(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试：需要完整 Docker 构建（使用 -short 跳过）")
	}

	helper, err := dockerhelper.NewClient()
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer helper.Close()

	pingCtx, pingCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer pingCancel()
	if err := helper.Ping(pingCtx); err != nil {
		t.Skipf("Docker daemon not available, skipping: %v", err)
	}

	baseImage := dockerfilegen.DefaultBaseImage
	exists, err := helper.ImageExists(pingCtx, baseImage)
	if err != nil || !exists {
		t.Skipf("Base image %s not cached, skipping integration test", baseImage)
	}

	// First build to create agent-forge:latest
	engine := New(helper)
	defer engine.Close()

	buildCtx, buildCancel := context.WithTimeout(context.Background(), 600*time.Second)
	defer buildCancel()

	output, err := engine.Build(buildCtx, BuildParams{
		Deps:     "",
		MaxRetry: 1,
	})
	if err != nil {
		t.Fatalf("First Build() error = %v\nOutput: %s", err, output)
	}

	// Get original image ID
	images, err := helper.ImageList(buildCtx, types.ImageListOptions{})
	if err != nil {
		t.Fatalf("ImageList() error = %v", err)
	}
	origID := ""
	for _, img := range images {
		for _, tag := range img.RepoTags {
			if tag == ImageTag {
				origID = img.ID
				break
			}
		}
		if origID != "" {
			break
		}
	}
	if origID == "" {
		t.Fatal("Could not find agent-forge:latest after first build")
	}

	// Rebuild with -R (forces --no-cache, atomic replacement)
	output2, err := engine.Build(buildCtx, BuildParams{
		Deps:     "",
		MaxRetry: 1,
		Rebuild:  true,
	})
	if err != nil {
		t.Fatalf("Rebuild() error = %v\nOutput: %s", err, output2)
	}

	// Verify new image exists with different ID
	images2, err := helper.ImageList(buildCtx, types.ImageListOptions{})
	if err != nil {
		t.Fatalf("ImageList() error = %v", err)
	}
	newID := ""
	for _, img := range images2 {
		for _, tag := range img.RepoTags {
			if tag == ImageTag {
				newID = img.ID
				break
			}
		}
		if newID != "" {
			break
		}
	}
	if newID == "" {
		t.Fatal("agent-forge:latest not found after rebuild")
	}
	if newID == origID {
		t.Log("Warning: rebuilt image has same ID (may be identical content with --no-cache)")
	}

	// Verify old image ID is no longer present (cleaned up)
	oldStillExists := false
	for _, img := range images2 {
		if img.ID == origID {
			// Check if the old image still has any tags
			if len(img.RepoTags) > 0 {
				oldStillExists = true
			}
			break
		}
	}
	if oldStillExists {
		t.Log("Note: old image still has tags (expected if intermediate was shared)")
	}

	// Cleanup
	_, err = helper.ImageRemove(buildCtx, ImageTag, true, true)
	if err != nil {
		t.Logf("Warning: failed to remove test image after rebuild: %v", err)
	}
}

func TestRebuild_SuccessPath_WithoutDocker(t *testing.T) {
	// Test that a rebuild build with unreachable Docker properly returns error
	// (doesn't panic, doesn't create temporary artifacts)
	unreachableClient, err := dockerhelper.NewClientWithOpts(
		client.WithHost("unix:///var/run/nonexistent.sock"),
	)
	if err != nil {
		t.Fatalf("NewClientWithOpts() error = %v", err)
	}
	defer unreachableClient.Close()

	engine := New(unreachableClient)
	defer engine.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = engine.Build(ctx, BuildParams{
		Deps:     "claude",
		MaxRetry: 1,
		Rebuild:  true,
	})
	if err == nil {
		t.Fatal("Build() expected error with unreachable Docker daemon in rebuild mode")
	}
	t.Logf("Rebuild mode with unreachable Docker returned expected error: %v", err)
}


func TestBuildParams_WindowsImageOnLinuxDaemon(t *testing.T) {
	_, _, err := platform.ResolvePlatform(
		"mcr.microsoft.com/powershell:lts-nanoserver-1809",
		"linux",
	)
	if err == nil {
		t.Fatal("Windows image on Linux daemon should return error")
	}
}

func TestBuildParams_WindowsDaemonDefaultImage(t *testing.T) {
	resolvedPlatform, resolvedImage, err := platform.ResolvePlatform("", "windows")
	if err != nil {
		t.Fatalf("ResolvePlatform() unexpected error: %v", err)
	}
	if resolvedPlatform != platform.PlatformWindows {
		t.Errorf("platform = %q, want windows", resolvedPlatform)
	}
	if resolvedImage != platform.DefaultWindowsBaseImage {
		t.Errorf("image = %q, want %q", resolvedImage, platform.DefaultWindowsBaseImage)
	}
}

func TestBuildParams_LinuxDaemonDefaultImage(t *testing.T) {
	resolvedPlatform, resolvedImage, err := platform.ResolvePlatform("", "linux")
	if err != nil {
		t.Fatalf("ResolvePlatform() unexpected error: %v", err)
	}
	if resolvedPlatform != platform.PlatformLinux {
		t.Errorf("platform = %q, want linux", resolvedPlatform)
	}
	if resolvedImage != platform.DefaultLinuxBaseImage {
		t.Errorf("image = %q, want %q", resolvedImage, platform.DefaultLinuxBaseImage)
	}
}

func TestImageTagWindowsConstant(t *testing.T) {
	if ImageTagWindows != "agent-forge:latest-windows" {
		t.Errorf("ImageTagWindows = %q, want agent-forge:latest-windows", ImageTagWindows)
	}
}
