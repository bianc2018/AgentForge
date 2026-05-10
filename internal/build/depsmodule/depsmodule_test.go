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
