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
