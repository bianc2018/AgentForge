package argsparser

import (
	"testing"
)

func TestParseRun_FullParams(t *testing.T) {
	args := []string{"-a", "claude", "-p", "3000:3000", "-p", "8080:8080", "-m", "/data", "-e", "KEY=VAL", "-w", "/work"}
	params, err := ParseRun(args)
	if err != nil {
		t.Fatalf("ParseRun() error = %v", err)
	}
	if params.Agent != "claude" {
		t.Errorf("Agent = %q, want %q", params.Agent, "claude")
	}
	if len(params.Ports) != 2 || params.Ports[0] != "3000:3000" || params.Ports[1] != "8080:8080" {
		t.Errorf("Ports = %v, want [3000:3000 8080:8080]", params.Ports)
	}
	if len(params.Mounts) != 1 || params.Mounts[0] != "/data" {
		t.Errorf("Mounts = %v, want [/data]", params.Mounts)
	}
	if len(params.Envs) != 1 || params.Envs[0] != "KEY=VAL" {
		t.Errorf("Envs = %v, want [KEY=VAL]", params.Envs)
	}
	if params.Workdir != "/work" {
		t.Errorf("Workdir = %q, want %q", params.Workdir, "/work")
	}
}

func TestParseRun_RecallAlias(t *testing.T) {
	r1, err := ParseRun([]string{"-r"})
	if err != nil {
		t.Fatalf("ParseRun(-r) error = %v", err)
	}
	if !r1.Recall {
		t.Error("ParseRun(-r): Recall = false, want true")
	}

	r2, err := ParseRun([]string{"--recall"})
	if err != nil {
		t.Fatalf("ParseRun(--recall) error = %v", err)
	}
	if !r2.Recall {
		t.Error("ParseRun(--recall): Recall = false, want true")
	}
}

func TestParseRun_DockerAlias(t *testing.T) {
	r1, err := ParseRun([]string{"--docker"})
	if err != nil {
		t.Fatalf("ParseRun(--docker) error = %v", err)
	}
	if !r1.Docker {
		t.Error("ParseRun(--docker): Docker = false, want true")
	}

	r2, err := ParseRun([]string{"--dind"})
	if err != nil {
		t.Fatalf("ParseRun(--dind) error = %v", err)
	}
	if !r2.Docker {
		t.Error("ParseRun(--dind): Docker = false, want true")
	}
}

func TestParseRun_DefaultValues(t *testing.T) {
	params, err := ParseRun([]string{})
	if err != nil {
		t.Fatalf("ParseRun() error = %v", err)
	}
	if params.Agent != "" {
		t.Errorf("Agent = %q, want empty", params.Agent)
	}
	if params.Ports != nil {
		t.Errorf("Ports = %v, want nil", params.Ports)
	}
	if params.Mounts != nil {
		t.Errorf("Mounts = %v, want nil", params.Mounts)
	}
	if params.Envs != nil {
		t.Errorf("Envs = %v, want nil", params.Envs)
	}
	if params.Recall {
		t.Error("Recall = true, want false")
	}
	if params.Docker {
		t.Error("Docker = true, want false")
	}
}

func TestParseRun_UnknownFlag(t *testing.T) {
	_, err := ParseRun([]string{"--unknown"})
	if err == nil {
		t.Fatal("ParseRun(--unknown): expected error")
	}
	if _, ok := err.(*ErrUnknownFlag); !ok {
		t.Errorf("ParseRun(--unknown): error type = %T, want *ErrUnknownFlag", err)
	}
}

func TestParseRun_MissingValue(t *testing.T) {
	_, err := ParseRun([]string{"-a"})
	if err == nil {
		t.Fatal("ParseRun(-a): expected error")
	}
	if _, ok := err.(*ErrMissingValue); !ok {
		t.Errorf("ParseRun(-a): error type = %T, want *ErrMissingValue", err)
	}
}

func TestParseRun_CrossCommandFlag(t *testing.T) {
	// build flags should not work in run
	_, err := ParseRun([]string{"-R"})
	if err == nil {
		t.Fatal("ParseRun(-R): expected error (build flag not applicable to run)")
	}

	_, err2 := ParseRun([]string{"--no-cache"})
	if err2 == nil {
		t.Fatal("ParseRun(--no-cache): expected error (build flag not applicable to run)")
	}
}

func TestParseBuild_FullParams(t *testing.T) {
	args := []string{"-d", "all", "--max-retry", "5", "--gh-proxy", "https://proxy.example.com", "--no-cache"}
	params, err := ParseBuild(args)
	if err != nil {
		t.Fatalf("ParseBuild() error = %v", err)
	}
	if params.Deps != "all" {
		t.Errorf("Deps = %q, want %q", params.Deps, "all")
	}
	if params.MaxRetry != 5 {
		t.Errorf("MaxRetry = %d, want %d", params.MaxRetry, 5)
	}
	if params.GHProxy != "https://proxy.example.com" {
		t.Errorf("GHProxy = %q, want %q", params.GHProxy, "https://proxy.example.com")
	}
	if !params.NoCache {
		t.Error("NoCache = false, want true")
	}
}

func TestParseBuild_RebuildAlias(t *testing.T) {
	b1, err := ParseBuild([]string{"-R"})
	if err != nil {
		t.Fatalf("ParseBuild(-R) error = %v", err)
	}
	if !b1.Rebuild {
		t.Error("ParseBuild(-R): Rebuild = false, want true")
	}

	b2, err := ParseBuild([]string{"--rebuild"})
	if err != nil {
		t.Fatalf("ParseBuild(--rebuild) error = %v", err)
	}
	if !b2.Rebuild {
		t.Error("ParseBuild(--rebuild): Rebuild = false, want true")
	}
}

func TestParseBuild_DefaultValues(t *testing.T) {
	params, err := ParseBuild([]string{})
	if err != nil {
		t.Fatalf("ParseBuild() error = %v", err)
	}
	if params.BaseImage != "docker.1ms.run/centos:7" {
		t.Errorf("BaseImage = %q, want %q", params.BaseImage, "docker.1ms.run/centos:7")
	}
	if params.MaxRetry != 3 {
		t.Errorf("MaxRetry = %d, want %d", params.MaxRetry, 3)
	}
	if params.Deps != "" {
		t.Errorf("Deps = %q, want empty", params.Deps)
	}
	if params.Rebuild {
		t.Error("Rebuild = true, want false")
	}
}

func TestParseBuild_UnknownFlag(t *testing.T) {
	_, err := ParseBuild([]string{"--unknown-flag"})
	if err == nil {
		t.Fatal("ParseBuild(--unknown-flag): expected error")
	}
	if _, ok := err.(*ErrUnknownFlag); !ok {
		t.Errorf("ParseBuild(--unknown-flag): error type = %T, want *ErrUnknownFlag", err)
	}
}

func TestParseBuild_MissingValue(t *testing.T) {
	_, err := ParseBuild([]string{"-d"})
	if err == nil {
		t.Fatal("ParseBuild(-d): expected error")
	}
	if _, ok := err.(*ErrMissingValue); !ok {
		t.Errorf("ParseBuild(-d): error type = %T, want *ErrMissingValue", err)
	}
}

func TestParseBuild_InvalidMaxRetry(t *testing.T) {
	_, err := ParseBuild([]string{"--max-retry", "not-a-number"})
	if err == nil {
		t.Fatal("ParseBuild(--max-retry not-a-number): expected error")
	}
	if _, ok := err.(*ErrInvalidValue); !ok {
		t.Errorf("ParseBuild(--max-retry not-a-number): error type = %T, want *ErrInvalidValue", err)
	}
}

func TestParseBuild_NegativeMaxRetry(t *testing.T) {
	_, err := ParseBuild([]string{"--max-retry", "-1"})
	if err == nil {
		t.Fatal("ParseBuild(--max-retry -1): expected error")
	}
}

func TestParseBuild_CrossCommandFlag(t *testing.T) {
	// run flags should not work in build
	_, err := ParseBuild([]string{"-a", "claude"})
	if err == nil {
		t.Fatal("ParseBuild(-a): expected error (run flag not applicable to build)")
	}

	_, err2 := ParseBuild([]string{"--docker"})
	if err2 == nil {
		t.Fatal("ParseBuild(--docker): expected error (run flag not applicable to build)")
	}
}

func TestParseBuild_BaseImageCustom(t *testing.T) {
	params, err := ParseBuild([]string{"-b", "centos:8"})
	if err != nil {
		t.Fatalf("ParseBuild(-b centos:8) error = %v", err)
	}
	if params.BaseImage != "centos:8" {
		t.Errorf("BaseImage = %q, want %q", params.BaseImage, "centos:8")
	}

	params2, err := ParseBuild([]string{"--base-image", "ubuntu:22.04"})
	if err != nil {
		t.Fatalf("ParseBuild(--base-image ubuntu:22.04) error = %v", err)
	}
	if params2.BaseImage != "ubuntu:22.04" {
		t.Errorf("BaseImage = %q, want %q", params2.BaseImage, "ubuntu:22.04")
	}
}

func TestParseRun_MultiPortAndMount(t *testing.T) {
	args := []string{
		"-p", "3000:3000", "-p", "8080:8080", "-p", "9090:9090",
		"-m", "/data1", "-m", "/data2",
	}
	params, err := ParseRun(args)
	if err != nil {
		t.Fatalf("ParseRun() error = %v", err)
	}
	if len(params.Ports) != 3 {
		t.Errorf("len(Ports) = %d, want 3", len(params.Ports))
	}
	if len(params.Mounts) != 2 {
		t.Errorf("len(Mounts) = %d, want 2", len(params.Mounts))
	}
}

func TestParseRun_WithSubcommandToken(t *testing.T) {
	// Simulate "run" as subcommand token being passed
	params, err := ParseRun([]string{"run", "-a", "claude"})
	if err != nil {
		t.Fatalf("ParseRun() error = %v", err)
	}
	if params.Agent != "claude" {
		t.Errorf("Agent = %q, want %q", params.Agent, "claude")
	}
}

func TestParseBuild_WithSubcommandToken(t *testing.T) {
	params, err := ParseBuild([]string{"build", "-d", "mini"})
	if err != nil {
		t.Fatalf("ParseBuild() error = %v", err)
	}
	if params.Deps != "mini" {
		t.Errorf("Deps = %q, want %q", params.Deps, "mini")
	}
}
