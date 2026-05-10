package dockerhelper

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

func TestNewClient_Default(t *testing.T) {
	c, err := NewClient()
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer c.Close()
}

func TestNewClientWithOpts_CustomHost(t *testing.T) {
	_, err := NewClientWithOpts(
		client.WithHost("unix:///var/run/nonexistent.sock"),
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		t.Fatalf("NewClientWithOpts() should not fail on creation: %v", err)
	}
}

func TestPing_WithRealDaemon(t *testing.T) {
	c, err := NewClient()
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = c.Ping(ctx)
	if err != nil {
		t.Fatalf("Ping() error = %v", err)
	}
}

func TestPing_UnreachableDaemon(t *testing.T) {
	c, err := NewClientWithOpts(
		client.WithHost("tcp://192.0.2.1:2375"),
	)
	if err != nil {
		t.Fatalf("NewClientWithOpts() error = %v", err)
	}
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err = c.Ping(ctx)
	if err == nil {
		t.Fatal("Ping() expected error for unreachable daemon, got nil")
	}
	t.Logf("Ping() error for unreachable daemon (expected): %v", err)
}

func TestInfo_WithRealDaemon(t *testing.T) {
	c, err := NewClient()
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	info, err := c.Info(ctx)
	if err != nil {
		t.Fatalf("Info() error = %v", err)
	}

	if info.ID == "" {
		t.Error("Info() returned empty Docker daemon ID")
	}
	t.Logf("Docker version: %s, OS: %s", info.ServerVersion, info.OperatingSystem)
}

func TestImageList_WithRealDaemon(t *testing.T) {
	c, err := NewClient()
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	images, err := c.ImageList(ctx, types.ImageListOptions{})
	if err != nil {
		t.Fatalf("ImageList() error = %v", err)
	}

	t.Logf("Found %d local images", len(images))
	for _, img := range images {
		for _, tag := range img.RepoTags {
			t.Logf("  - %s", tag)
		}
	}
}

func TestIsBuildKitEnabled(t *testing.T) {
	c, err := NewClient()
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer c.Close()

	// Save and restore env var
	oldVal := os.Getenv("DOCKER_BUILDKIT")
	defer os.Setenv("DOCKER_BUILDKIT", oldVal)

	os.Setenv("DOCKER_BUILDKIT", "1")
	if !c.IsBuildKitEnabled() {
		t.Error("IsBuildKitEnabled() = false, want true when DOCKER_BUILDKIT=1")
	}

	os.Setenv("DOCKER_BUILDKIT", "")
	if c.IsBuildKitEnabled() {
		t.Error("IsBuildKitEnabled() = true, want false when DOCKER_BUILDKIT unset")
	}

	os.Setenv("DOCKER_BUILDKIT", "0")
	if c.IsBuildKitEnabled() {
		t.Error("IsBuildKitEnabled() = true, want false when DOCKER_BUILDKIT=0")
	}
}

func TestImageExists(t *testing.T) {
	c, err := NewClient()
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	images, err := c.ImageList(ctx, types.ImageListOptions{})
	if err != nil {
		t.Fatalf("ImageList() error = %v", err)
	}

	if len(images) > 0 {
		for _, tag := range images[0].RepoTags {
			exists, err := c.ImageExists(ctx, tag)
			if err != nil {
				t.Fatalf("ImageExists(%q) error = %v", tag, err)
			}
			if !exists {
				t.Errorf("ImageExists(%q) = false, want true", tag)
			}
			break
		}
	} else {
		t.Log("No images found on this system, skipping ImageExists test")
	}
}

func TestPingWithInfo(t *testing.T) {
	c, err := NewClient()
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	info, err := c.PingWithInfo(ctx)
	if err != nil {
		t.Fatalf("PingWithInfo() error = %v", err)
	}
	if info.ServerVersion == "" {
		t.Error("PingWithInfo() returned empty ServerVersion")
	}
	t.Logf("Docker Server Version: %s", info.ServerVersion)
}

func TestClassifyError_Nil(t *testing.T) {
	if err := ClassifyError(nil); err != nil {
		t.Errorf("ClassifyError(nil) = %v, want nil", err)
	}
}
