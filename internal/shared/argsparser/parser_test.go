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

// ---------------------------------------------------------------------------
// Error type formatting
// ---------------------------------------------------------------------------

func TestErrUnknownFlag_Error(t *testing.T) {
	e := &ErrUnknownFlag{Flag: "--bogus"}
	want := "未知参数: --bogus"
	if got := e.Error(); got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestErrMissingValue_Error(t *testing.T) {
	e := &ErrMissingValue{Flag: "-x"}
	want := "参数 -x 缺少值"
	if got := e.Error(); got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestErrInvalidValue_Error(t *testing.T) {
	e := &ErrInvalidValue{Flag: "--port", Value: "abc", Reason: "必须为整数"}
	want := `参数 --port 的值 "abc" 无效: 必须为整数`
	if got := e.Error(); got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

// ---------------------------------------------------------------------------
// ParseBuild — table-driven edge cases
// ---------------------------------------------------------------------------

func TestParseBuild_MissingValueForEachFlag(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "deps short -d", args: []string{"-d"}},
		{name: "base-image short -b", args: []string{"-b"}},
		{name: "config short -c", args: []string{"-c"}},
		{name: "max-retry without value", args: []string{"--max-retry"}},
		{name: "gh-proxy without value", args: []string{"--gh-proxy"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseBuild(tt.args)
			if err == nil {
				t.Fatal("expected ErrMissingValue")
			}
			if _, ok := err.(*ErrMissingValue); !ok {
				t.Errorf("error type = %T, want *ErrMissingValue", err)
			}
		})
	}
}

func TestParseBuild_ConfigFlagShortAndLong(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "short -c", args: []string{"-c", "/tmp/build-cfg"}, want: "/tmp/build-cfg"},
		{name: "long --config", args: []string{"--config", "/tmp/build-cfg"}, want: "/tmp/build-cfg"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params, err := ParseBuild(tt.args)
			if err != nil {
				t.Fatalf("ParseBuild(%v) error = %v", tt.args, err)
			}
			if params.Config != tt.want {
				t.Errorf("Config = %q, want %q", params.Config, tt.want)
			}
		})
	}
}

func TestParseBuild_LongFormFlags(t *testing.T) {
	params, err := ParseBuild([]string{
		"--deps", "golang,node",
		"--base-image", "ubuntu:22.04",
		"--no-cache",
		"--rebuild",
		"--max-retry", "10",
		"--gh-proxy", "https://ghproxy.example.com",
	})
	if err != nil {
		t.Fatalf("ParseBuild() error = %v", err)
	}
	if params.Deps != "golang,node" {
		t.Errorf("Deps = %q, want %q", params.Deps, "golang,node")
	}
	if params.BaseImage != "ubuntu:22.04" {
		t.Errorf("BaseImage = %q, want %q", params.BaseImage, "ubuntu:22.04")
	}
	if !params.NoCache {
		t.Error("NoCache = false, want true")
	}
	if !params.Rebuild {
		t.Error("Rebuild = false, want true")
	}
	if params.MaxRetry != 10 {
		t.Errorf("MaxRetry = %d, want %d", params.MaxRetry, 10)
	}
	if params.GHProxy != "https://ghproxy.example.com" {
		t.Errorf("GHProxy = %q, want %q", params.GHProxy, "https://ghproxy.example.com")
	}
}

func TestParseBuild_NilAndEmptyArgs(t *testing.T) {
	// nil slice
	params, err := ParseBuild(nil)
	if err != nil {
		t.Fatalf("ParseBuild(nil) error = %v", err)
	}
	if params.BaseImage != "docker.1ms.run/centos:7" {
		t.Errorf("BaseImage = %q, want default", params.BaseImage)
	}
	// empty slice
	params2, err := ParseBuild([]string{})
	if err != nil {
		t.Fatalf("ParseBuild([]) error = %v", err)
	}
	if params2.MaxRetry != 3 {
		t.Errorf("MaxRetry = %d, want 3", params2.MaxRetry)
	}
}

func TestParseBuild_MaxRetryZero(t *testing.T) {
	params, err := ParseBuild([]string{"--max-retry", "0"})
	if err != nil {
		t.Fatalf("ParseBuild(--max-retry 0) error = %v", err)
	}
	if params.MaxRetry != 0 {
		t.Errorf("MaxRetry = %d, want 0", params.MaxRetry)
	}
}

// ---------------------------------------------------------------------------
// ParseRun — table-driven edge cases
// ---------------------------------------------------------------------------

func TestParseRun_MissingValueForEachFlag(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "agent short -a", args: []string{"-a"}},
		{name: "port short -p", args: []string{"-p"}},
		{name: "mount short -m", args: []string{"-m"}},
		{name: "env short -e", args: []string{"-e"}},
		{name: "workdir short -w", args: []string{"-w"}},
		{name: "run flag --run", args: []string{"--run"}},
		{name: "config short -c", args: []string{"-c"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseRun(tt.args)
			if err == nil {
				t.Fatal("expected ErrMissingValue")
			}
			if _, ok := err.(*ErrMissingValue); !ok {
				t.Errorf("error type = %T, want *ErrMissingValue", err)
			}
		})
	}
}

func TestParseRun_CommandFlag(t *testing.T) {
	params, err := ParseRun([]string{"--run", "npm test"})
	if err != nil {
		t.Fatalf("ParseRun(--run) error = %v", err)
	}
	if params.RunCmd != "npm test" {
		t.Errorf("RunCmd = %q, want %q", params.RunCmd, "npm test")
	}
}

func TestParseRun_ConfigFlagShortAndLong(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "short -c", args: []string{"-c", "/tmp/run-cfg"}, want: "/tmp/run-cfg"},
		{name: "long --config", args: []string{"--config", "/tmp/run-cfg"}, want: "/tmp/run-cfg"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params, err := ParseRun(tt.args)
			if err != nil {
				t.Fatalf("ParseRun(%v) error = %v", tt.args, err)
			}
			if params.Config != tt.want {
				t.Errorf("Config = %q, want %q", params.Config, tt.want)
			}
		})
	}
}

func TestParseRun_LongFormFlags(t *testing.T) {
	params, err := ParseRun([]string{
		"--agent", "claude",
		"--port", "3000:3000",
		"--mount", "/data",
		"--env", "KEY=VAL",
		"--workdir", "/app",
		"--recall",
		"--docker",
	})
	if err != nil {
		t.Fatalf("ParseRun() error = %v", err)
	}
	if params.Agent != "claude" {
		t.Errorf("Agent = %q, want %q", params.Agent, "claude")
	}
	if len(params.Ports) != 1 || params.Ports[0] != "3000:3000" {
		t.Errorf("Ports = %v, want [3000:3000]", params.Ports)
	}
	if len(params.Mounts) != 1 || params.Mounts[0] != "/data" {
		t.Errorf("Mounts = %v, want [/data]", params.Mounts)
	}
	if len(params.Envs) != 1 || params.Envs[0] != "KEY=VAL" {
		t.Errorf("Envs = %v, want [KEY=VAL]", params.Envs)
	}
	if params.Workdir != "/app" {
		t.Errorf("Workdir = %q, want %q", params.Workdir, "/app")
	}
	if !params.Recall {
		t.Error("Recall = false, want true")
	}
	if !params.Docker {
		t.Error("Docker = false, want true")
	}
}

func TestParseRun_MultiEnv(t *testing.T) {
	params, err := ParseRun([]string{"-e", "A=1", "-e", "B=2", "-e", "C=3"})
	if err != nil {
		t.Fatalf("ParseRun() error = %v", err)
	}
	if len(params.Envs) != 3 {
		t.Fatalf("len(Envs) = %d, want 3", len(params.Envs))
	}
	if params.Envs[0] != "A=1" || params.Envs[1] != "B=2" || params.Envs[2] != "C=3" {
		t.Errorf("Envs = %v, want [A=1 B=2 C=3]", params.Envs)
	}
}

func TestParseRun_NilAndEmptyArgs(t *testing.T) {
	// nil slice
	params, err := ParseRun(nil)
	if err != nil {
		t.Fatalf("ParseRun(nil) error = %v", err)
	}
	if params.Agent != "" {
		t.Errorf("Agent = %q, want empty", params.Agent)
	}
	// empty slice
	params2, err := ParseRun([]string{})
	if err != nil {
		t.Fatalf("ParseRun([]) error = %v", err)
	}
	if params2.Docker {
		t.Error("Docker = true, want false")
	}
}

func TestParseRun_DindAlias(t *testing.T) {
	// --dind is already tested in TestParseRun_DockerAlias, but ensure
	// the short form of recall doesn't interfere with --dind
	params, err := ParseRun([]string{"--dind"})
	if err != nil {
		t.Fatalf("ParseRun(--dind) error = %v", err)
	}
	if !params.Docker {
		t.Error("Docker = false, want true")
	}
}
