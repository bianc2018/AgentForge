package depsmodule

import (
	"testing"
)

// --- ExpandDeps tests ---

func TestExpandDeps_All(t *testing.T) {
	result := ExpandDeps("all")
	if len(result) == 0 {
		t.Fatal("ExpandDeps(\"all\") returned empty list")
	}
		// Verify all known deps are included (except those intentionally excluded)
		known := ListAllKnownDeps()
		// gitnexus needs C++14 for native module compilation, excluded from all
		// until CentOS 7 base image supports it
		excluded := map[string]bool{"gitnexus": true}
		unknowMap := make(map[string]bool)
		for _, dep := range result {
			baseName, _ := splitNameVersion(dep)
			unknowMap[baseName] = true
		}
		for _, name := range known {
			if excluded[name] {
				continue
			}
			if !unknowMap[name] {
				t.Errorf("ExpandDeps(\"all\") missing dep: %s", name)
			}
		}
}

func TestExpandDeps_Mini(t *testing.T) {
	result := ExpandDeps("mini")
	if len(result) == 0 {
		t.Fatal("ExpandDeps(\"mini\") returned empty list")
	}
	if len(result) >= len(ExpandDeps("all")) {
		t.Errorf("ExpandDeps(\"mini\") returned %d items, expected fewer than all (%d)",
			len(result), len(ExpandDeps("all")))
	}
	// mini should include claude
	hasClaude := false
	for _, dep := range result {
		if dep == "claude" {
			hasClaude = true
			break
		}
	}
	if !hasClaude {
		t.Error("ExpandDeps(\"mini\") should include claude")
	}
}

func TestExpandDeps_SingleDeps(t *testing.T) {
	result := ExpandDeps("claude,golang@1.21,node@20")
	if len(result) != 3 {
		t.Fatalf("ExpandDeps() returned %d items, want 3: %v", len(result), result)
	}
	expected := []string{"claude", "golang@1.21", "node@20"}
	for i, exp := range expected {
		if result[i] != exp {
			t.Errorf("result[%d] = %q, want %q", i, result[i], exp)
		}
	}
}

func TestExpandDeps_MixedWithMeta(t *testing.T) {
	result := ExpandDeps("all,claude")
	// Should have all deps, with claude deduplicated
	allLen := len(ExpandDeps("all"))
	if len(result) != allLen {
		t.Errorf("ExpandDeps(\"all,claude\") returned %d items, want %d (all with dedup)", len(result), allLen)
	}
}

func TestExpandDeps_UnknownName(t *testing.T) {
	result := ExpandDeps("my-custom-pkg")
	if len(result) != 1 || result[0] != "my-custom-pkg" {
		t.Errorf("ExpandDeps(\"my-custom-pkg\") = %v, want [my-custom-pkg]", result)
	}
}

func TestExpandDeps_EmptyInput(t *testing.T) {
	result := ExpandDeps("")
	if len(result) != 0 {
		t.Errorf("ExpandDeps(\"\") = %v, want empty slice", result)
	}
}

func TestExpandDeps_Whitespace(t *testing.T) {
	result := ExpandDeps("  claude ,  golang@1.21  ")
	if len(result) != 2 {
		t.Fatalf("ExpandDeps() returned %d items, want 2: %v", len(result), result)
	}
	if result[0] != "claude" {
		t.Errorf("result[0] = %q, want %q", result[0], "claude")
	}
}

func TestExpandDeps_WithLeadingTrailingSpaces(t *testing.T) {
	result := ExpandDeps("  mini  ")
	if len(result) == 0 {
		t.Fatal("ExpandDeps(\"  mini  \") returned empty")
	}
	hasClaude := false
	for _, dep := range result {
		if dep == "claude" {
			hasClaude = true
			break
		}
	}
	if !hasClaude {
		t.Error("ExpandDeps(\"  mini  \") should include claude")
	}
}

// --- ResolveInstallMethod tests ---

func TestResolveInstallMethod_Agent(t *testing.T) {
	method, err := ResolveInstallMethod("claude")
	if err != nil {
		t.Fatalf("ResolveInstallMethod(\"claude\") error = %v", err)
	}
	if method.Type != DepAgent {
		t.Errorf("Type = %v, want DepAgent", method.Type)
	}
	if len(method.Commands) == 0 {
		t.Error("Commands is empty")
	}
	if method.Version != "" {
		t.Errorf("Version = %q, want empty", method.Version)
	}
}

func TestResolveInstallMethod_RuntimeWithVersion(t *testing.T) {
	method, err := ResolveInstallMethod("golang@1.21")
	if err != nil {
		t.Fatalf("ResolveInstallMethod(\"golang@1.21\") error = %v", err)
	}
	if method.Type != DepRuntime {
		t.Errorf("Type = %v, want DepRuntime", method.Type)
	}
	if method.Version != "1.21" {
		t.Errorf("Version = %q, want \"1.21\"", method.Version)
	}
	if len(method.Commands) == 0 {
		t.Error("Commands is empty")
	}
}

func TestResolveInstallMethod_RuntimeWithoutVersion(t *testing.T) {
	method, err := ResolveInstallMethod("node")
	if err != nil {
		t.Fatalf("ResolveInstallMethod(\"node\") error = %v", err)
	}
	if method.Type != DepRuntime {
		t.Errorf("Type = %v, want DepRuntime", method.Type)
	}
	if method.Version != "" {
		t.Errorf("Version = %q, want empty", method.Version)
	}
}

func TestResolveInstallMethod_Tool(t *testing.T) {
	method, err := ResolveInstallMethod("gitnexus")
	if err != nil {
		t.Fatalf("ResolveInstallMethod(\"speckit\") error = %v", err)
	}
	if method.Type != DepTool {
		t.Errorf("Type = %v, want DepTool", method.Type)
	}
	if len(method.Commands) == 0 {
		t.Error("Commands is empty")
	}
	if method.Version != "" {
		t.Errorf("Version = %q, want empty", method.Version)
	}
}

func TestResolveInstallMethod_UnknownName(t *testing.T) {
	method, err := ResolveInstallMethod("my-pkg")
	if err != nil {
		t.Fatalf("ResolveInstallMethod(\"my-pkg\") error = %v", err)
	}
	if method.Type != DepSystemPkg {
		t.Errorf("Type = %v, want DepSystemPkg", method.Type)
	}
	if len(method.Commands) != 1 {
		t.Fatalf("Commands = %v, want 1 command", method.Commands)
	}
	if method.Commands[0] != "yum install -y my-pkg" {
		t.Errorf("Command = %q, want \"yum install -y my-pkg\"", method.Commands[0])
	}
}

func TestResolveInstallMethod_InvalidName(t *testing.T) {
	_, err := ResolveInstallMethod("@1.21")
	if err == nil {
		t.Fatal("ResolveInstallMethod(\"@1.21\"): expected error")
	}
}

func TestResolveInstallMethod_EmptyName(t *testing.T) {
	_, err := ResolveInstallMethod("")
	if err == nil {
		t.Fatal("ResolveInstallMethod(\"\"): expected error")
	}
}

func TestResolveInstallMethod_AllTypes(t *testing.T) {
	tests := []struct {
		name     string
		wantType DepType
	}{
		{"claude", DepAgent},
		{"opencode", DepAgent},
		{"kimi", DepAgent},
		{"deepseek-tui", DepAgent},
		{"golang", DepRuntime},
		{"node", DepRuntime},
		{"gitnexus", DepTool},
		{"docker", DepTool},
		{"rtk", DepTool},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			method, err := ResolveInstallMethod(tt.name)
			if err != nil {
				t.Fatalf("ResolveInstallMethod(%q) error = %v", tt.name, err)
			}
			if method.Type != tt.wantType {
				t.Errorf("Type = %v, want %v", method.Type, tt.wantType)
			}
			if len(method.Commands) == 0 {
				t.Error("Commands is empty")
			}
		})
	}
}

func TestResolveInstallMethod_DockerUsesStaticBinary(t *testing.T) {
	method, err := ResolveInstallMethod("docker")
	if err != nil {
		t.Fatalf("ResolveInstallMethod(\"docker\") error = %v", err)
	}
	if method.Type != DepTool {
		t.Errorf("Type = %v, want DepTool", method.Type)
	}
	if len(method.Commands) != 3 {
		t.Fatalf("Commands count = %d, want 3 (curl + tar + cleanup)", len(method.Commands))
	}
	// Must use curl to download static binary, not yum
	if contains(method.Commands[0], "yum install") {
		t.Error("docker install should use static binary, not yum")
	}
	if !contains(method.Commands[0], "download.docker.com/linux/static/stable") {
		t.Error("docker install should download from Docker official static binary URL")
	}
	if !contains(method.Commands[0], "docker-24.0.7.tgz") {
		t.Error("docker install should use default version 24.0.7")
	}
	if !contains(method.Commands[1], "tar -C /usr/local/bin") {
		t.Error("docker install should extract to /usr/local/bin")
	}
}

func TestResolveInstallMethod_DockerCustomVersion(t *testing.T) {
	method, err := ResolveInstallMethod("docker@26.1.0")
	if err != nil {
		t.Fatalf("ResolveInstallMethod(\"docker@26.1.0\") error = %v", err)
	}
	if method.Version != "26.1.0" {
		t.Errorf("Version = %q, want \"26.1.0\"", method.Version)
	}
	if !contains(method.Commands[0], "docker-26.1.0.tgz") {
		t.Error("docker install should use custom version 26.1.0")
	}
}

func TestResolveInstallMethod_GolangCommandsContainVersion(t *testing.T) {
	method, err := ResolveInstallMethod("golang@1.21")
	if err != nil {
		t.Fatalf("ResolveInstallMethod(\"golang@1.21\") error = %v", err)
	}
	found := false
	for _, cmd := range method.Commands {
		if contains(cmd, "go1.21") {
			found = true
			break
		}
	}
	if !found {
		t.Error("golang@1.21 commands should contain version reference \"go1.21\"")
	}
}

// --- Helper Tests ---

func TestSplitNameVersion(t *testing.T) {
	tests := []struct {
		input      string
		wantBase   string
		wantVersion string
	}{
		{"claude", "claude", ""},
		{"golang@1.21", "golang", "1.21"},
		{"node@20", "node", "20"},
		{"@1.21", "@1.21", ""}, // @ at start not treated as version separator
	}
	for _, tt := range tests {
		base, version := splitNameVersion(tt.input)
		if base != tt.wantBase {
			t.Errorf("splitNameVersion(%q) base = %q, want %q", tt.input, base, tt.wantBase)
		}
		if version != tt.wantVersion {
			t.Errorf("splitNameVersion(%q) version = %q, want %q", tt.input, version, tt.wantVersion)
		}
	}
}

func TestIsKnownDep(t *testing.T) {
	if !IsKnownDep("claude") {
		t.Error("IsKnownDep(\"claude\") = false, want true")
	}
	if !IsKnownDep("golang@1.21") {
		t.Error("IsKnownDep(\"golang@1.21\") = false, want true")
	}
	if IsKnownDep("unknown-pkg") {
		t.Error("IsKnownDep(\"unknown-pkg\") = true, want false")
	}
}

// --- DepType.String() tests ---

func TestDepType_String(t *testing.T) {
	tests := []struct {
		dt   DepType
		want string
	}{
		{DepAgent, "agent"},
		{DepRuntime, "runtime"},
		{DepTool, "tool"},
		{DepSystemPkg, "system"},
		{DepType(99), "unknown"},  // default branch
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.dt.String()
			if got != tt.want {
				t.Errorf("DepType(%d).String() = %q, want %q", tt.dt, got, tt.want)
			}
		})
	}
}

// --- ExpandDeps edge case tests ---

func TestExpandDeps_EmptyEntries(t *testing.T) {
	result := ExpandDeps("claude,,golang@1.21")
	if len(result) != 2 {
		t.Fatalf("ExpandDeps(\"claude,,golang@1.21\") = %v (len=%d), want [claude golang@1.21]", result, len(result))
	}
	if result[0] != "claude" {
		t.Errorf("result[0] = %q, want \"claude\"", result[0])
	}
	if result[1] != "golang@1.21" {
		t.Errorf("result[1] = %q, want \"golang@1.21\"", result[1])
	}
}

func TestExpandDeps_AllDedupWithinList(t *testing.T) {
	all := ExpandDeps("all")
	allDup := ExpandDeps("all,all")
	if len(allDup) != len(all) {
		t.Errorf("ExpandDeps(\"all,all\") = %d items, want %d (no dupes)", len(allDup), len(all))
	}
}

func TestExpandDeps_MultipleMetaTags(t *testing.T) {
	result := ExpandDeps("mini,all")
	allLen := len(ExpandDeps("all"))
	if len(result) != allLen {
		t.Errorf("ExpandDeps(\"mini,all\") = %d items, want %d (all superset of mini)", len(result), allLen)
	}
}

func TestExpandDeps_OnlyCommas(t *testing.T) {
	result := ExpandDeps(",,,")
	if len(result) != 0 {
		t.Errorf("ExpandDeps(\",,,\") = %v (len=%d), want empty slice", result, len(result))
	}
}

// --- ResolveInstallMethod additional tests ---

func TestResolveInstallMethod_OpenspecAndSpeckit(t *testing.T) {
	t.Run("openspec", func(t *testing.T) {
		method, err := ResolveInstallMethod("openspec")
		if err != nil {
			t.Fatalf("ResolveInstallMethod(\"openspec\") error = %v", err)
		}
		if method.Type != DepAgent {
			t.Errorf("Type = %v, want DepAgent", method.Type)
		}
		if len(method.Commands) == 0 {
			t.Error("Commands is empty")
		}
	})
	t.Run("speckit", func(t *testing.T) {
		method, err := ResolveInstallMethod("speckit")
		if err != nil {
			t.Fatalf("ResolveInstallMethod(\"speckit\") error = %v", err)
		}
		if method.Type != DepAgent {
			t.Errorf("Type = %v, want DepAgent", method.Type)
		}
		if len(method.Commands) == 0 {
			t.Error("Commands is empty")
		}
	})
}

func TestResolveInstallMethod_InvalidNameFormats(t *testing.T) {
	tests := []struct {
		name string
	}{
		{"-foo"},         // starts with dash
		{".foo"},         // starts with dot
		{"_foo"},         // starts with underscore
		{"foo@"},         // empty version after @
		{"foo@@1.0"},     // double @
		{"foo@bar@baz"},  // multiple @
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ResolveInstallMethod(tt.name)
			if err == nil {
				t.Errorf("ResolveInstallMethod(%q) expected error, got nil", tt.name)
			}
		})
	}
}

func TestResolveInstallMethod_UnknownWithVersion(t *testing.T) {
	method, err := ResolveInstallMethod("my-pkg@1.0")
	if err != nil {
		t.Fatalf("ResolveInstallMethod(\"my-pkg@1.0\") error = %v", err)
	}
	if method.Type != DepSystemPkg {
		t.Errorf("Type = %v, want DepSystemPkg", method.Type)
	}
	if method.Version != "1.0" {
		t.Errorf("Version = %q, want \"1.0\"", method.Version)
	}
	if len(method.Commands) != 1 {
		t.Fatalf("Commands = %v, want 1 command", method.Commands)
	}
	if method.Commands[0] != "yum install -y my-pkg" {
		t.Errorf("Command = %q, want \"yum install -y my-pkg\"", method.Commands[0])
	}
}

func TestResolveInstallMethod_WhitespaceTrimmed(t *testing.T) {
	method, err := ResolveInstallMethod("  claude  ")
	if err != nil {
		t.Fatalf("ResolveInstallMethod(\"  claude  \") error = %v", err)
	}
	if method.Type != DepAgent {
		t.Errorf("Type = %v, want DepAgent", method.Type)
	}
}

func TestResolveInstallMethod_NodeVersionAppendsXSuffix(t *testing.T) {
	method, err := ResolveInstallMethod("node@20")
	if err != nil {
		t.Fatalf("ResolveInstallMethod(\"node@20\") error = %v", err)
	}
	if method.Type != DepRuntime {
		t.Errorf("Type = %v, want DepRuntime", method.Type)
	}
	if method.Version != "20" {
		t.Errorf("Version = %q, want \"20\"", method.Version)
	}
	// The commands should reference setup_20.x (appended .x suffix)
	found := false
	for _, cmd := range method.Commands {
		if contains(cmd, "setup_20.x") {
			found = true
			break
		}
	}
	if !found {
		t.Error("node@20 commands should contain \"setup_20.x\" (appended .x suffix)")
	}
}

func TestResolveInstallMethod_GolangShortVersionPadZero(t *testing.T) {
	// golang@1.22 has 2 parts -> should become 1.22.0
	method, err := ResolveInstallMethod("golang@1.22")
	if err != nil {
		t.Fatalf("ResolveInstallMethod(\"golang@1.22\") error = %v", err)
	}
	if method.Version != "1.22" {
		t.Errorf("Version = %q, want \"1.22\"", method.Version)
	}
	found := false
	for _, cmd := range method.Commands {
		if contains(cmd, "go1.22.0") {
			found = true
			break
		}
	}
	if !found {
		t.Error("golang@1.22 commands should contain \"go1.22.0\" (padded with .0)")
	}
}

func TestResolveInstallMethod_GolangFullVersionNoPad(t *testing.T) {
	// golang@1.22.4 has 3 parts -> should stay as-is
	method, err := ResolveInstallMethod("golang@1.22.4")
	if err != nil {
		t.Fatalf("ResolveInstallMethod(\"golang@1.22.4\") error = %v", err)
	}
	if method.Version != "1.22.4" {
		t.Errorf("Version = %q, want \"1.22.4\"", method.Version)
	}
	found := false
	for _, cmd := range method.Commands {
		if contains(cmd, "go1.22.4") {
			found = true
			break
		}
	}
	if !found {
		t.Error("golang@1.22.4 commands should contain \"go1.22.4\" (no padding needed)")
	}
}

// --- splitNameVersion edge case tests ---

func TestSplitNameVersion_EdgeCases(t *testing.T) {
	tests := []struct {
		input      string
		wantBase   string
		wantVersion string
	}{
		{"name@@1.0", "name@", "1.0"},   // double @, split at last
		{"a@b", "a", "b"},
		{"simple", "simple", ""},
		{"@atstart", "@atstart", ""},    // @ at position 0
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			base, version := splitNameVersion(tt.input)
			if base != tt.wantBase {
				t.Errorf("splitNameVersion(%q) base = %q, want %q", tt.input, base, tt.wantBase)
			}
			if version != tt.wantVersion {
				t.Errorf("splitNameVersion(%q) version = %q, want %q", tt.input, version, tt.wantVersion)
			}
		})
	}
}

// --- IsKnownDep whitespace and edge-case tests ---

func TestIsKnownDep_Whitespace(t *testing.T) {
	if !IsKnownDep("  claude  ") {
		t.Error("IsKnownDep(\"  claude  \") = false, want true")
	}
	if !IsKnownDep("  golang@1.21  ") {
		t.Error("IsKnownDep(\"  golang@1.21  \") = false, want true")
	}
}

func TestIsKnownDep_AllKnownDeps(t *testing.T) {
	known := ListAllKnownDeps()
	for _, name := range known {
		if !IsKnownDep(name) {
			t.Errorf("IsKnownDep(%q) = false, want true", name)
		}
	}
	if IsKnownDep("") {
		t.Error("IsKnownDep(\"\") = true, want false")
	}
	if IsKnownDep("  ") {
		t.Error("IsKnownDep(\"  \") = true, want false")
	}
}

// --- ListAllKnownDeps tests ---

func TestListAllKnownDeps(t *testing.T) {
	deps := ListAllKnownDeps()
	if len(deps) == 0 {
		t.Fatal("ListAllKnownDeps() returned empty slice")
	}
	// Should contain all required deps
	expected := map[string]bool{
		"claude": true, "opencode": true, "kimi": true,
		"deepseek-tui": true, "openspec": true, "speckit": true,
		"golang": true, "node": true,
		"gitnexus": true, "docker": true, "rtk": true,
	}
	depMap := make(map[string]bool)
	for _, d := range deps {
		if depMap[d] {
			t.Errorf("ListAllKnownDeps() contains duplicate: %q", d)
		}
		depMap[d] = true
	}
	for name := range expected {
		if !depMap[name] {
			t.Errorf("ListAllKnownDeps() missing %q", name)
		}
	}
}

// --- Helpers ---

func contains(s, substr string) bool {
	return len(s) >= len(substr) && containsSubstr(s, substr)
}

func containsSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
