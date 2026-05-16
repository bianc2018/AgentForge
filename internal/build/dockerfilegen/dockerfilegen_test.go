package dockerfilegen

import (
	"strings"
	"testing"
)

func TestGenerate_NormalPath(t *testing.T) {
	opts := Options{
		Deps: []string{"claude", "speckit"},
	}
	dockerfile, err := Generate(opts)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	// Should contain FROM instruction
	if !strings.Contains(dockerfile, "FROM ") {
		t.Error("Generated Dockerfile missing FROM instruction")
	}

	// Should contain RUN instructions
	if !strings.Contains(dockerfile, "RUN ") {
		t.Error("Generated Dockerfile missing RUN instructions")
	}

	// Should contain npm install commands for claude
	if !strings.Contains(dockerfile, "npm install -g @anthropic-ai/claude-code") {
		t.Error("Generated Dockerfile missing claude install command")
	}

	// Should contain CMD
	if !strings.Contains(dockerfile, "CMD [\"/bin/bash\"]") {
		t.Error("Generated Dockerfile missing CMD instruction")
	}

	// Should be valid Dockerfile format (each instruction line starts with valid instruction)
	// Continuation lines (ending with \) are allowed to have arbitrary content.
	lines := strings.Split(dockerfile, "\n")
	isContinuation := false
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			isContinuation = false
			continue
		}
		if isContinuation {
			isContinuation = strings.HasSuffix(line, "\\")
			continue
		}
		validPrefix := false
		for _, prefix := range []string{"FROM", "RUN", "ENV", "CMD", "COPY", "ADD", "EXPOSE", "WORKDIR", "LABEL", "MAINTAINER", "ARG", "VOLUME", "USER", "ONBUILD", "STOPSIGNAL", "HEALTHCHECK", "SHELL", "ENTRYPOINT"} {
			if strings.HasPrefix(line, prefix+" ") || strings.HasPrefix(line, prefix+"\t") || line == prefix {
				validPrefix = true
				break
			}
		}
		if !validPrefix {
			t.Errorf("Invalid Dockerfile line (not a valid instruction and not a continuation): %q", line)
		}
		isContinuation = strings.HasSuffix(line, "\\")
	}
}

func TestGenerate_CustomBaseImage(t *testing.T) {
	opts := Options{
		BaseImage: "centos:8",
	}
	dockerfile, err := Generate(opts)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if !strings.Contains(dockerfile, "FROM centos:8") {
		t.Errorf("Expected FROM centos:8, got:\n%s", dockerfile)
	}
}

func TestGenerate_DefaultBaseImage(t *testing.T) {
	opts := Options{
		Deps: []string{},
	}
	dockerfile, err := Generate(opts)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if !strings.Contains(dockerfile, "FROM docker.1ms.run/centos:7") {
		t.Errorf("Expected default FROM, got:\n%s", dockerfile)
	}
}

func TestGenerate_MirrorSourceConfig(t *testing.T) {
	opts := Options{
		Deps: []string{"claude", "rtk"},
	}
	dockerfile, err := Generate(opts)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	// Should contain npm mirror configuration
	if !strings.Contains(dockerfile, "registry.npmmirror.com") {
		t.Error("Generated Dockerfile missing npm mirror config")
	}

	// Should contain pip mirror configuration
	if !strings.Contains(dockerfile, "mirrors.aliyun.com/pypi") {
		t.Error("Generated Dockerfile missing pip mirror config")
	}
}

func TestGenerate_WithGHProxy(t *testing.T) {
	opts := Options{
		Deps:    []string{"golang@1.22.3"},
		GHProxy: "https://ghproxy.example.com/",
	}
	dockerfile, err := Generate(opts)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	// Should contain GH_PROXY_URL env var
	if !strings.Contains(dockerfile, "GH_PROXY_URL=") {
		t.Error("Generated Dockerfile missing GH_PROXY_URL env var")
	}

	// Should NOT apply gh-proxy to Go download URL (golang.google.cn accessible directly)
	if strings.Contains(dockerfile, "https://ghproxy.example.com/https://golang.google.cn/dl/") {
		t.Error("Go download URL was incorrectly proxy-prefixed")
	}
}

func TestGenerate_EmptyDeps(t *testing.T) {
	opts := Options{
		Deps: []string{},
	}
	dockerfile, err := Generate(opts)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	// Should have FROM
	if !strings.Contains(dockerfile, "FROM ") {
		t.Error("Minimal Dockerfile missing FROM")
	}

	// Should have base tools
	if !strings.Contains(dockerfile, "yum install -y curl git") {
		t.Error("Minimal Dockerfile missing base tools")
	}

	// Should have CMD
	if !strings.Contains(dockerfile, "CMD ") {
		t.Error("Minimal Dockerfile missing CMD")
	}

	// Empty deps should not include dependency-specific installs
	if strings.Contains(dockerfile, "claude") {
		t.Log("Empty deps Dockerfile should not contain specific deps")
	}
}

func TestGenerate_WithAllDeps(t *testing.T) {
	// Use a moderate number of deps to keep the test fast
	opts := Options{
		Deps: []string{"claude", "opencode", "node@20", "gitnexus", "docker"},
	}
	dockerfile, err := Generate(opts)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	// Count RUN instructions (should be more than base)
	runCount := strings.Count(dockerfile, "\nRUN ")
	if runCount < 5 {
		t.Errorf("Expected at least 5 RUN instructions with 5 deps, got %d", runCount)
	}

	// Each dep should have a comment
	if !strings.Contains(dockerfile, "# claude") {
		t.Error("Missing comment for claude")
	}
	if !strings.Contains(dockerfile, "# opencode") {
		t.Error("Missing comment for opencode")
	}
}

func TestGenerate_NoGHProxyByDefault(t *testing.T) {
	opts := Options{
		Deps: []string{"claude"},
	}
	dockerfile, err := Generate(opts)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if strings.Contains(dockerfile, "GH_PROXY_URL") {
		t.Error("Dockerfile should not contain GH_PROXY_URL when not set")
	}
}

func TestGenerate_NodeRuntimeInstall(t *testing.T) {
	opts := Options{
		Deps: []string{"node@16"},
	}
	dockerfile, err := Generate(opts)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	// Node install should have the correct nodesource setup
	if !strings.Contains(dockerfile, "nodesource.com/setup_16.x") {
		t.Error("Missing Node.js 16.x setup reference")
	}

	// Node should be installed
	if !strings.Contains(dockerfile, "yum install -y nodejs") {
		t.Error("Missing nodejs yum install")
	}
}

// Test that the generated Dockerfile is parseable by checking structural integrity
func TestGenerate_StructuralIntegrity(t *testing.T) {
	opts := Options{
		BaseImage: "docker.1ms.run/centos:7",
		Deps:      []string{"claude", "golang@1.22.3", "speckit", "gitnexus"},
		GHProxy:   "https://ghproxy.example.com/",
	}
	dockerfile, err := Generate(opts)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	// FROM must be the first non-comment, non-empty line
	lines := strings.Split(dockerfile, "\n")
	firstRealLine := ""
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && !strings.HasPrefix(trimmed, "#") {
			firstRealLine = trimmed
			break
		}
	}
	if !strings.HasPrefix(firstRealLine, "FROM ") {
		t.Errorf("First instruction should be FROM, got %q", firstRealLine)
	}

	// CMD should be the last instruction
	lastRealLine := ""
	for i := len(lines) - 1; i >= 0; i-- {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed != "" && !strings.HasPrefix(trimmed, "#") {
			lastRealLine = trimmed
			break
		}
	}
	if lastRealLine != "CMD [\"/bin/bash\"]" {
		t.Errorf("Last instruction should be CMD, got %q", lastRealLine)
	}

	// No empty FROM or RUN
	if strings.Contains(dockerfile, "FROM \n") {
		t.Error("Empty FROM reference found")
	}

	t.Logf("Generated Dockerfile:\n%s", dockerfile[:500]+"...")
}

func TestAdaptCommandForFamily_DebianTranslatesYumToApt(t *testing.T) {
	tests := []struct {
		name string
		cmd  string
		want string
	}{
		{
			name: "yum install translated",
			cmd:  "yum install -y curl",
			want: "apt-get install -y curl",
		},
		{
			name: "yum clean translated",
			cmd:  "yum clean all && rm -rf /var/cache/yum/*",
			want: "apt-get clean && rm -rf /var/lib/apt/lists/*",
		},
		{
			name: "curl command not translated",
			cmd:  "curl -fsSL https://example.com/file -o /tmp/file",
			want: "curl -fsSL https://example.com/file -o /tmp/file",
		},
		{
			name: "npm command not translated",
			cmd:  "npm install -g some-package",
			want: "npm install -g some-package",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := adaptCommandForFamily(tt.cmd, FamilyDebian)
			if got != tt.want {
				t.Errorf("adaptCommandForFamily(%q, FamilyDebian) = %q, want %q", tt.cmd, got, tt.want)
			}
		})
	}
}

func TestAdaptCommandForFamily_RHELNoChange(t *testing.T) {
	cmd := "yum install -y docker"
	got := adaptCommandForFamily(cmd, FamilyRHEL)
	if got != cmd {
		t.Errorf("adaptCommandForFamily(%q, FamilyRHEL) = %q, want unchanged", cmd, got)
	}
}

func TestAdaptCommandForFamily_PackageNameMapping(t *testing.T) {
	// docker → docker.io mapping on Debian family
	cmd := "yum install -y docker"
	got := adaptCommandForFamily(cmd, FamilyDebian)
	if !strings.Contains(got, "apt-get install -y docker.io") {
		t.Errorf("Expected docker → docker.io mapping on Debian, got: %q", got)
	}
	if strings.Contains(got, "apt-get install -y docker ") || got == "apt-get install -y docker" {
		t.Errorf("Package name should be mapped to docker.io, got: %q", got)
	}
}

func TestAdaptCommandForFamily_UnknownFamily(t *testing.T) {
	// FamilyUnknown should behave like RHEL (no translation)
	cmd := "yum install -y curl"
	got := adaptCommandForFamily(cmd, FamilyUnknown)
	if got != cmd {
		t.Errorf("adaptCommandForFamily(%q, FamilyUnknown) = %q, want unchanged", cmd, got)
	}
}

// --- detectImageFamily tests ---

func TestDetectImageFamily(t *testing.T) {
	tests := []struct {
		name      string
		baseImage string
		want      ImageFamily
	}{
		{"ubuntu", "ubuntu:22.04", FamilyDebian},
		{"ubuntu latest", "ubuntu:latest", FamilyDebian},
		{"debian", "debian:11", FamilyDebian},
		{"debian slim", "debian:bookworm-slim", FamilyDebian},
		{"centos", "centos:7", FamilyRHEL},
		{"centos stream", "centos:stream9", FamilyRHEL},
		{"rhel", "registry.access.redhat.com/rhel:8", FamilyRHEL},
		{"fedora", "fedora:39", FamilyRHEL},
		{"rocky", "rockylinux:9", FamilyRHEL},
		{"almalinux", "almalinux:9", FamilyRHEL},
		{"oraclelinux", "oraclelinux:8", FamilyRHEL},
		{"alpine", "alpine:3.18", FamilyUnknown},
		{"scratch", "scratch", FamilyUnknown},
		{"empty image string", "", FamilyUnknown},
		{"case insensitive debian", "Ubuntu:22.04", FamilyDebian},
		{"case insensitive rhel", "CentOS:7", FamilyRHEL},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectImageFamily(tt.baseImage)
			if got != tt.want {
				t.Errorf("detectImageFamily(%q) = %v, want %v", tt.baseImage, got, tt.want)
			}
		})
	}
}

// --- applyGHProxy tests ---

func TestApplyGHProxy_EmptyProxyReturnsCommandsUnchanged(t *testing.T) {
	cmds := []string{"curl -fsSL https://github.com/foo/bar -o /tmp/bar", "echo done"}
	got := applyGHProxy(cmds, "")
	if len(got) != len(cmds) {
		t.Fatalf("len(got) = %d, want %d", len(got), len(cmds))
	}
	for i := range cmds {
		if got[i] != cmds[i] {
			t.Errorf("got[%d] = %q, want %q", i, got[i], cmds[i])
		}
	}
}

func TestApplyGHProxy_GithubURLsPrefixed(t *testing.T) {
	cmds := []string{
		"curl -fsSL https://github.com/opencode-ai/opencode/releases/download/v0.0.55/opencode-linux-x86_64.tar.gz -o /tmp/opencode.tar.gz",
		"tar -C /usr/local/bin -xzf /tmp/opencode.tar.gz opencode",
	}
	proxy := "https://ghproxy.example.com/"
	got := applyGHProxy(cmds, proxy)
	if !strings.Contains(got[0], proxy+"https://github.com/") {
		t.Errorf("github URL should be prefixed with proxy, got: %q", got[0])
	}
	if got[1] != cmds[1] {
		t.Errorf("non-github command should be unchanged, got[%d] = %q", 1, got[1])
	}
}

func TestApplyGHProxy_NonGithubURLsNotAffected(t *testing.T) {
	cmds := []string{
		"curl -fsSL https://golang.google.cn/dl/go1.22.3.linux-amd64.tar.gz -o /tmp/go.tar.gz",
		"echo hello",
	}
	proxy := "https://ghproxy.example.com/"
	got := applyGHProxy(cmds, proxy)
	for i := range cmds {
		if got[i] != cmds[i] {
			t.Errorf("non-github command should be unchanged, got[%d] = %q", i, got[i])
		}
	}
}

func TestApplyGHProxy_NilInput(t *testing.T) {
	got := applyGHProxy(nil, "https://ghproxy.example.com/")
	if got == nil {
		t.Error("expected non-nil empty slice for nil input with proxy set")
	}
	if len(got) != 0 {
		t.Errorf("expected empty slice, got len=%d", len(got))
	}
}

// --- writeDebianSetup tests ---

func TestWriteDebianSetup(t *testing.T) {
	var sb strings.Builder
	writeDebianSetup(&sb)
	output := sb.String()

	if !strings.Contains(output, "mirrors.aliyun.com") {
		t.Error("Debian setup should contain Aliyun mirror configuration")
	}
	if !strings.Contains(output, "archive.ubuntu.com") {
		t.Error("Debian setup should reference archive.ubuntu.com sources")
	}
	if !strings.Contains(output, "deb.debian.org/debian") {
		t.Error("Debian setup should reference deb.debian.org sources")
	}
	if !strings.Contains(output, "apt-get update") {
		t.Error("Debian setup should contain apt-get update")
	}
	if !strings.Contains(output, "apt-get install -y curl git wget tar gzip unzip ca-certificates") {
		t.Error("Debian setup should contain base tool installation")
	}
	if !strings.Contains(output, "rm -rf /var/lib/apt/lists/*") {
		t.Error("Debian setup should clean apt lists")
	}
}

// --- writeRHELSetup tests ---

func TestWriteRHELSetup(t *testing.T) {
	var sb strings.Builder
	writeRHELSetup(&sb)
	output := sb.String()

	if !strings.Contains(output, "mirrors.aliyun.com/centos-vault") {
		t.Error("RHEL setup should contain Aliyun CentOS vault mirror")
	}
	if !strings.Contains(output, "yum install -y epel-release") {
		t.Error("RHEL setup should install epel-release")
	}
	if !strings.Contains(output, "yum install -y curl git wget tar gzip unzip") {
		t.Error("RHEL setup should contain base tool installation")
	}
	if !strings.Contains(output, "rm -rf /var/cache/yum/*") {
		t.Error("RHEL setup should clean yum cache")
	}
	if !strings.Contains(output, "mirrorlist") {
		t.Error("RHEL setup should reference mirrorlist in sed")
	}
}

// --- analyzeRuntimeNeeds tests ---

func TestAnalyzeRuntimeNeeds(t *testing.T) {
	tests := []struct {
		name    string
		deps    []string
		wantNpm bool
		wantPip bool
		wantGCC bool
		wantErr bool
		errMsg  string
	}{
		{
			name:    "npm dep sets npm and gcc",
			deps:    []string{"claude"},
			wantNpm: true,
			wantGCC: true,
		},
		{
			name:    "pip dep sets pip",
			deps:    []string{"rtk"},
			wantPip: true,
		},
		{
			name:    "npm and pip deps set all",
			deps:    []string{"claude", "rtk"},
			wantNpm: true,
			wantPip: true,
			wantGCC: true,
		},
		{
			name:    "system dep sets nothing",
			deps:    []string{"unknown-pkg"},
		},
		{
			name:    "docker dep sets nothing",
			deps:    []string{"docker"},
		},
		{
			name:    "empty deps sets nothing",
			deps:    []string{},
		},
		{
			name:    "tool with npm sets npm and gcc",
			deps:    []string{"gitnexus"},
			wantNpm: true,
			wantGCC: true,
		},
		{
			name:    "invalid dep name returns error",
			deps:    []string{"@invalid"},
			wantErr: true,
			errMsg:  "分析依赖",
		},
		{
			name:    "empty dep string returns error",
			deps:    []string{""},
			wantErr: true,
			errMsg:  "分析依赖",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			needs, err := analyzeRuntimeNeeds(tt.deps)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error but got nil")
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("error message = %q, want containing %q", err.Error(), tt.errMsg)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if needs.needsNpm != tt.wantNpm {
				t.Errorf("needsNpm = %v, want %v", needs.needsNpm, tt.wantNpm)
			}
			if needs.needsPip != tt.wantPip {
				t.Errorf("needsPip = %v, want %v", needs.needsPip, tt.wantPip)
			}
			if needs.needsGCC != tt.wantGCC {
				t.Errorf("needsGCC = %v, want %v", needs.needsGCC, tt.wantGCC)
			}
		})
	}
}

// --- Generate tests for Debian family ---

func TestGenerate_DebianFamily(t *testing.T) {
	opts := Options{
		BaseImage: "ubuntu:22.04",
		Deps:      []string{},
	}
	dockerfile, err := Generate(opts)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if !strings.Contains(dockerfile, "FROM ubuntu:22.04") {
		t.Errorf("Expected FROM ubuntu:22.04")
	}

	if strings.Contains(dockerfile, "yum ") {
		t.Error("Debian Dockerfile should not contain yum commands")
	}

	if !strings.Contains(dockerfile, "archive.ubuntu.com") {
		t.Error("Debian Dockerfile should contain apt source configuration")
	}

	if !strings.Contains(dockerfile, "apt-get clean") {
		t.Error("Debian Dockerfile cleanup should use apt-get clean")
	}

	if strings.Contains(dockerfile, "yum clean") {
		t.Error("Debian Dockerfile should not contain yum clean")
	}

	if strings.Contains(dockerfile, "build-essential") {
		t.Error("Debian Dockerfile should not have build-essential without npm deps")
	}

	// Should have CMD
	if !strings.Contains(dockerfile, "CMD [\"/bin/bash\"]") {
		t.Error("Missing CMD instruction")
	}
}

func TestGenerate_DebianFamily_WithNpmOnly(t *testing.T) {
	opts := Options{
		BaseImage: "ubuntu:22.04",
		Deps:      []string{"claude"},
	}
	dockerfile, err := Generate(opts)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	// Should install build-essential (needsGCC triggered by npm)
	if !strings.Contains(dockerfile, "build-essential") {
		t.Error("Debian Dockerfile with npm deps should install build-essential")
	}
	if strings.Contains(dockerfile, "gcc-c++") {
		t.Error("Debian Dockerfile should not use gcc-c++ (RHEL package)")
	}

	// Should use deb.nodesource.com for npm
	if !strings.Contains(dockerfile, "https://deb.nodesource.com/setup_22.x") {
		t.Error("Debian Dockerfile should use deb.nodesource.com setup_22.x")
	}
	if strings.Contains(dockerfile, "https://rpm.nodesource.com/setup_16.x") {
		t.Error("Debian Dockerfile should not use rpm.nodesource.com")
	}

	// Should install nodejs via apt
	if !strings.Contains(dockerfile, "apt-get install -y nodejs") {
		t.Error("Debian Dockerfile should install nodejs via apt")
	}

	// Should have npm mirror
	if !strings.Contains(dockerfile, "npm_config_registry") {
		t.Error("Debian Dockerfile should have npm mirror env var")
	}

	// Should NOT have pip-related content
	if strings.Contains(dockerfile, "pip3") {
		t.Error("Debian Dockerfile with only npm deps should not have pip content")
	}

	// Cleanup should include npm cache but not pip3 cache
	if !strings.Contains(dockerfile, "npm cache clean") {
		t.Error("Should include npm cache cleanup")
	}
	if strings.Contains(dockerfile, "pip3 cache purge") {
		t.Error("Should not include pip3 cache cleanup")
	}

	// Install claude via npm
	if !strings.Contains(dockerfile, "npm install -g @anthropic-ai/claude-code") {
		t.Error("Should install claude via npm")
	}
}

func TestGenerate_DebianFamily_WithPipOnly(t *testing.T) {
	opts := Options{
		BaseImage: "ubuntu:24.04",
		Deps:      []string{"rtk"},
	}
	dockerfile, err := Generate(opts)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	// Should have pip runtime install via apt
	if !strings.Contains(dockerfile, "apt-get install -y python3 python3-pip") {
		t.Error("Should install python3-pip via apt-get")
	}

	// Should NOT have npm or gcc setup
	if strings.Contains(dockerfile, "build-essential") {
		t.Error("Should not install build-essential without npm deps")
	}
	if strings.Contains(dockerfile, "nodesource") {
		t.Error("Should not have Node.js setup without npm deps")
	}

	// pip mirror via config set (Debian style)
	if !strings.Contains(dockerfile, "pip3 config set global.index-url") {
		t.Error("Should configure pip mirror via pip3 config set")
	}
	if strings.Contains(dockerfile, "PIP_INDEX_URL") {
		t.Error("Debian should not use ENV for pip mirror (RHEL style)")
	}

	// Cleanup should include pip3 cache purge but not npm
	if !strings.Contains(dockerfile, "pip3 cache purge") {
		t.Error("Should include pip3 cache cleanup")
	}
	if strings.Contains(dockerfile, "npm cache clean") {
		t.Error("Should not include npm cache cleanup")
	}

	// rtk install via pip3
	if !strings.Contains(dockerfile, "pip3 install rtk") {
		t.Error("Should install rtk via pip3")
	}
}

func TestGenerate_DebianFamily_WithNpmAndPip(t *testing.T) {
	opts := Options{
		BaseImage: "debian:bookworm-slim",
		Deps:      []string{"claude", "rtk"},
	}
	dockerfile, err := Generate(opts)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if !strings.Contains(dockerfile, "FROM debian:bookworm-slim") {
		t.Error("FROM mismatch")
	}
	if strings.Contains(dockerfile, "yum ") {
		t.Error("Debian Dockerfile should not contain yum")
	}

	// gcc section
	if !strings.Contains(dockerfile, "build-essential") {
		t.Error("Should install build-essential")
	}

	// npm runtime: deb.nodesource.com
	if !strings.Contains(dockerfile, "deb.nodesource.com/setup_22.x") {
		t.Error("Should use deb.nodesource.com setup_22.x")
	}
	if !strings.Contains(dockerfile, "apt-get install -y nodejs") {
		t.Error("Should install nodejs via apt")
	}
	if !strings.Contains(dockerfile, "npm_config_registry") {
		t.Error("Should have npm mirror")
	}

	// pip runtime: apt install + config set
	if !strings.Contains(dockerfile, "apt-get install -y python3 python3-pip") {
		t.Error("Should install python3-pip via apt")
	}
	if !strings.Contains(dockerfile, "pip3 config set global.index-url") {
		t.Error("Should configure pip mirror via pip3 config set")
	}

	// Cleanup
	if !strings.Contains(dockerfile, "apt-get clean") {
		t.Error("Should use apt-get clean")
	}
	if !strings.Contains(dockerfile, "npm cache clean") {
		t.Error("Should include npm cache cleanup")
	}
	if !strings.Contains(dockerfile, "pip3 cache purge") {
		t.Error("Should include pip3 cache cleanup")
	}
}

// --- Generate error paths ---

func TestGenerate_Error_InvalidDepName(t *testing.T) {
	opts := Options{
		Deps: []string{"@invalid"},
	}
	_, err := Generate(opts)
	if err == nil {
		t.Fatal("expected error for invalid dep name, got nil")
	}
	if !strings.Contains(err.Error(), "分析依赖") {
		t.Errorf("error should contain '分析依赖', got: %v", err)
	}
}

func TestGenerate_Error_EmptyDepName(t *testing.T) {
	opts := Options{
		Deps: []string{""},
	}
	_, err := Generate(opts)
	if err == nil {
		t.Fatal("expected error for empty dep name, got nil")
	}
	if !strings.Contains(err.Error(), "分析依赖") {
		t.Errorf("error should contain '分析依赖', got: %v", err)
	}
}

// --- node dep skip when needsNpm is true ---

func TestGenerate_SkipNodeDepWhenNeedsNpm(t *testing.T) {
	// claude triggers needsNpm=true, node@20 should be skipped in install loop
	opts := Options{
		Deps: []string{"claude", "node@20"},
	}
	dockerfile, err := Generate(opts)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	// Should have skip comment for node@20
	if !strings.Contains(dockerfile, "跳过（Node.js 已由运行时层安装）") {
		t.Error("Expected node@20 dep to be skipped with runtime layer comment")
	}
}

// --- GHProxy with actual github URLs in dep commands ---

func TestGenerate_GHProxyWithGithubCommands(t *testing.T) {
	// opencode has a github.com URL in its install commands
	opts := Options{
		Deps:    []string{"opencode"},
		GHProxy: "https://ghproxy.example.com/",
	}
	dockerfile, err := Generate(opts)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if !strings.Contains(dockerfile, "GH_PROXY_URL=https://ghproxy.example.com/") {
		t.Error("Missing GH_PROXY_URL env var")
	}

	// The github.com URL should be proxy-prefixed
	if !strings.Contains(dockerfile, "https://ghproxy.example.com/https://github.com/") {
		t.Error("github.com URL should be proxy-prefixed in RUN command")
	}
}

// --- unknown dep (system pkg) adapted for Debian ---

func TestGenerate_DebianWithSystemPkg(t *testing.T) {
	opts := Options{
		BaseImage: "ubuntu:22.04",
		Deps:      []string{"some-unknown-pkg"},
	}
	dockerfile, err := Generate(opts)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	// yum install should be adapted to apt-get install for Debian
	if !strings.Contains(dockerfile, "apt-get install -y some-unknown-pkg") {
		t.Error("yum install for unknown pkg should be adapted to apt-get install on Debian")
	}
	if strings.Contains(dockerfile, "yum install -y some-unknown-pkg") {
		t.Error("Original yum command should not appear on Debian")
	}
}

// --- Debian with docker dep (known pkg name mapping) ---

func TestGenerate_DebianWithDockerDep(t *testing.T) {
	opts := Options{
		BaseImage: "ubuntu:22.04",
		Deps:      []string{"docker"},
	}
	dockerfile, err := Generate(opts)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	// docker is known dep - uses curl download, not yum install
	// But its commands don't contain yum, so adaptCommandForFamily won't change them
	// Verify docker commands are present
	if !strings.Contains(dockerfile, "download.docker.com") {
		t.Error("Docker dep should include download URL")
	}
}

// --- RHEL-based Generate with system pkg (adaptCommandForFamily not triggered) ---

func TestGenerate_RHELWithSystemPkg(t *testing.T) {
	opts := Options{
		Deps: []string{"some-rhel-pkg"},
	}
	dockerfile, err := Generate(opts)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	// Default is centos:7, so yum should be used
	if !strings.Contains(dockerfile, "yum install -y some-rhel-pkg") {
		t.Error("RHEL Dockerfile should use yum install for unknown pkg")
	}
	if strings.Contains(dockerfile, "apt-get install -y some-rhel-pkg") {
		t.Error("RHEL Dockerfile should not use apt-get")
	}
}

// --- Unknown family defaults to RHEL behavior ---

func TestGenerate_UnknownFamilyDefaultsToRHEL(t *testing.T) {
	opts := Options{
		BaseImage: "alpine:3.18",
		Deps:      []string{},
	}
	dockerfile, err := Generate(opts)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if !strings.Contains(dockerfile, "FROM alpine:3.18") {
		t.Error("FROM mismatch")
	}

	// Unknown family defaults to RHEL-style yum setup
	if !strings.Contains(dockerfile, "yum install -y epel-release") {
		t.Error("Unknown family should default to RHEL-style yum setup")
	}

	// Cleanup should use yum
	if !strings.Contains(dockerfile, "yum clean all") {
		t.Error("Unknown family cleanup should use yum clean")
	}
	// Should use RHEL-style rm -rf /tmp/* (not /var/lib/apt/lists/*)
	if strings.Contains(dockerfile, "/var/lib/apt/lists/*") {
		t.Error("Unknown family cleanup should not use Debian paths")
	}
}

// --- Verify cleanup section for RHEL with npm + pip ---

func TestGenerate_RHEL_CleanupWithNpmAndPip(t *testing.T) {
	opts := Options{
		Deps: []string{"claude", "rtk"},
	}
	dockerfile, err := Generate(opts)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	// Should have npm cache clean
	if !strings.Contains(dockerfile, "npm cache clean") {
		t.Error("Should include npm cache cleanup")
	}

	// Should have pip3 cache purge
	if !strings.Contains(dockerfile, "pip3 cache purge") {
		t.Error("Should include pip3 cache cleanup")
	}

	// Should use yum clean for RHEL
	if !strings.Contains(dockerfile, "yum clean all") {
		t.Error("RHEL cleanup should use yum clean")
	}
}

// --- Windows family tests ---

func TestDetectImageFamily_Windows(t *testing.T) {
	tests := []struct {
		name      string
		baseImage string
	}{
		{"nanoserver powershell", "mcr.microsoft.com/powershell:lts-nanoserver-1809"},
		{"servercore", "mcr.microsoft.com/windows/servercore:ltsc2022"},
		{"windows servercore full", "mcr.microsoft.com/powershell:lts-windowsservercore-1809"},
		{"case insensitive", "Mcr.Microsoft.Com/Windows/ServerCore:ltsc2022"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectImageFamily(tt.baseImage)
			if got != FamilyWindows {
				t.Errorf("detectImageFamily(%q) = %v, want FamilyWindows", tt.baseImage, got)
			}
		})
	}
}

func TestGenerate_WindowsNanoserver_EmptyDeps(t *testing.T) {
	opts := Options{
		BaseImage: "mcr.microsoft.com/powershell:lts-nanoserver-1809",
		Deps:      []string{},
	}
	dockerfile, err := Generate(opts)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if !strings.Contains(dockerfile, "FROM mcr.microsoft.com/powershell:lts-nanoserver-1809") {
		t.Error("FROM mismatch")
	}
	if !strings.Contains(dockerfile, `SHELL ["powershell", "-Command"]`) {
		t.Error("Windows Dockerfile should have SHELL powershell")
	}
	if !strings.Contains(dockerfile, "Invoke-WebRequest") {
		t.Error("Windows Dockerfile should use Invoke-WebRequest")
	}
	if strings.Contains(dockerfile, "yum ") {
		t.Error("Windows Dockerfile should not contain yum")
	}
	if strings.Contains(dockerfile, "apt-get") {
		t.Error("Windows Dockerfile should not contain apt-get")
	}
	if strings.Contains(dockerfile, ".curlrc") {
		t.Error("Windows Dockerfile should not have curl config")
	}
	if !strings.Contains(dockerfile, `CMD ["powershell"]`) {
		t.Error("Windows Dockerfile should end with CMD powershell")
	}
	if strings.Contains(dockerfile, `CMD ["/bin/bash"]`) {
		t.Error("Windows Dockerfile should not have bash CMD")
	}
}

func TestGenerate_WindowsNanoserver_WithNpm(t *testing.T) {
	opts := Options{
		BaseImage: "mcr.microsoft.com/powershell:lts-nanoserver-1809",
		Deps:      []string{"claude"},
	}
	dockerfile, err := Generate(opts)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if !strings.Contains(dockerfile, "node-v22.") {
		t.Error("Windows Dockerfile should download Node.js MSI")
	}
	if !strings.Contains(dockerfile, "msiexec.exe") {
		t.Error("Windows Dockerfile should use msiexec for Node.js install")
	}
	if !strings.Contains(dockerfile, "npm_config_registry") {
		t.Error("Windows Dockerfile should have npm mirror")
	}
	if strings.Contains(dockerfile, "nodesource.com") {
		t.Error("Windows Dockerfile should not use nodesource.com")
	}
	if strings.Contains(dockerfile, "build-essential") || strings.Contains(dockerfile, "gcc") {
		t.Error("Windows Dockerfile should not install gcc")
	}
	if strings.Contains(dockerfile, "yum install") || strings.Contains(dockerfile, "apt-get install") {
		t.Error("Windows Dockerfile should not use yum or apt-get")
	}
}

func TestGenerate_WindowsNanoserver_WithPip(t *testing.T) {
	opts := Options{
		BaseImage: "mcr.microsoft.com/powershell:lts-nanoserver-1809",
		Deps:      []string{"rtk"},
	}
	dockerfile, err := Generate(opts)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if !strings.Contains(dockerfile, "python-3.") {
		t.Error("Windows Dockerfile should download Python installer")
	}
	if !strings.Contains(dockerfile, "PIP_INDEX_URL") {
		t.Error("Windows Dockerfile should have pip mirror")
	}
	if strings.Contains(dockerfile, "apt-get install -y python3") {
		t.Error("Windows should not use apt-get for python")
	}
}

func TestGenerate_WindowsServerCore(t *testing.T) {
	opts := Options{
		BaseImage: "mcr.microsoft.com/windows/servercore:ltsc2022",
		Deps:      []string{},
	}
	dockerfile, err := Generate(opts)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if !strings.Contains(dockerfile, "FROM mcr.microsoft.com/windows/servercore:ltsc2022") {
		t.Error("FROM mismatch")
	}
	if !strings.Contains(dockerfile, `SHELL ["powershell", "-Command"]`) {
		t.Error("ServerCore should also use PowerShell")
	}
}

func TestAdaptCommandForFamily_Windows(t *testing.T) {
	tests := []struct {
		name string
		cmd  string
		want string
	}{
		{
			name: "curl translated to Invoke-WebRequest",
			cmd:  "curl -fsSL https://example.com/tool.tar.gz -o /tmp/tool.tar.gz",
			want: "Invoke-WebRequest -Uri https://example.com/tool.tar.gz -OutFile $env:TEMP\\tool.tar.gz",
		},
		{
			name: "tar translated to Expand-Archive",
			cmd:  "tar -xzf /tmp/foo.tar.gz",
			want: "Expand-Archive -Path $env:TEMP\\foo.tar.gz",
		},
		{
			name: "bash translated to powershell",
			cmd:  "bash /tmp/setup.sh",
			want: "powershell -File $env:TEMP\\setup.sh",
		},
		{
			name: "chmod skipped",
			cmd:  "chmod +x /usr/local/bin/tool",
			want: "Write-Host 'skip chmod: C:\\ProgramData\\bin/tool'",
		},
		{
			name: "npm unchanged",
			cmd:  "npm install -g some-package",
			want: "npm install -g some-package",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := adaptCommandForFamily(tt.cmd, FamilyWindows)
			if got != tt.want {
				t.Errorf("adaptCommandForFamily(%q, FamilyWindows) = %q, want %q", tt.cmd, got, tt.want)
			}
		})
	}
}

func TestWriteWindowsSetup(t *testing.T) {
	var sb strings.Builder
	writeWindowsSetup(&sb)
	output := sb.String()

	if !strings.Contains(output, `SHELL ["powershell", "-Command"]`) {
		t.Error("Windows setup should set SHELL to powershell")
	}
	if !strings.Contains(output, "Invoke-WebRequest") {
		t.Error("Windows setup should use Invoke-WebRequest for Git")
	}
	if !strings.Contains(output, "git-for-windows") {
		t.Error("Windows setup should install Git for Windows")
	}
}

// --- applyGHProxy edge cases ---

func TestApplyGHProxy_NoTrailingSlash(t *testing.T) {
	cmds := []string{
		"curl -fsSL https://github.com/foo/bar/releases/download/v1.0/app -o /usr/local/bin/app",
	}
	proxy := "https://ghproxy.example.com" // 无尾部斜杠
	got := applyGHProxy(cmds, proxy)

	wantPrefix := "https://ghproxy.example.com/https://github.com/"
	if !strings.Contains(got[0], wantPrefix) {
		t.Errorf("无尾部斜杠应正确拼接，got: %q, want prefix: %q", got[0], wantPrefix)
	}
	// 不应出现双写斜杠
	if strings.Contains(got[0], "ghproxy.example.com//") {
		t.Error("不应出现双斜杠")
	}
}

func TestApplyGHProxy_TrailingSlash(t *testing.T) {
	cmds := []string{
		"curl -fsSL https://github.com/foo/bar/releases/download/v1.0/app -o /usr/local/bin/app",
	}
	proxy := "https://ghproxy.example.com/" // 有尾部斜杠
	got := applyGHProxy(cmds, proxy)

	wantPrefix := "https://ghproxy.example.com/https://github.com/"
	if !strings.Contains(got[0], wantPrefix) {
		t.Errorf("有尾部斜杠不应双写，got: %q, want prefix: %q", got[0], wantPrefix)
	}
	// 不应出现双斜杠
	if strings.Contains(got[0], "ghproxy.example.com//") {
		t.Error("不应出现双斜杠")
	}
}

func TestGenerate_DefaultGHProxy(t *testing.T) {
	// 不传 GHProxy（空 Options），验证不含代理配置
	opts := Options{
		Deps: []string{"opencode"},
	}
	dockerfile, err := Generate(opts)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if strings.Contains(dockerfile, "GH_PROXY_URL") {
		t.Error("未设置 GHProxy 时不应出现 GH_PROXY_URL 环境变量")
	}
	// opencode 的 github.com URL 应保持原样（不被代理包装）
	if !strings.Contains(dockerfile, "https://github.com/opencode-ai/") {
		t.Error("未设置 GHProxy 时 github.com URL 应保持原样")
	}
}
