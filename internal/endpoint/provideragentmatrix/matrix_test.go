package provideragentmatrix

import (
	"reflect"
	"testing"
)

// --- UT-11: GetProviders ---

func TestGetProviders_NormalPath(t *testing.T) {
	got := GetProviders()
	want := []string{"deepseek", "openai", "anthropic"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("GetProviders() = %v, want %v", got, want)
	}
}

func TestGetProviders_Immutability(t *testing.T) {
	first := GetProviders()
	second := GetProviders()

	// 两次调用应返回完全相同的值
	if !reflect.DeepEqual(first, second) {
		t.Error("多次调用 GetProviders() 应返回相同的列表")
	}

	// 修改 first 不应影响 second（确保每次返回新切片）
	if len(first) > 0 {
		first[0] = "mutated"
	}
	// second 不应受影响
	if second[0] == "mutated" {
		t.Error("GetProviders() 返回的切片应不可变，修改副本不应影响原始数据")
	}
}

// --- UT-12: GetAgentsForProvider ---

func TestGetAgentsForProvider_DeepSeek(t *testing.T) {
	got := GetAgentsForProvider("deepseek")
	want := []string{"claude", "opencode", "kimi", "deepseek-tui"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("GetAgentsForProvider(\"deepseek\") = %v, want %v", got, want)
	}
}

func TestGetAgentsForProvider_OpenAI(t *testing.T) {
	got := GetAgentsForProvider("openai")
	want := []string{"claude", "opencode"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("GetAgentsForProvider(\"openai\") = %v, want %v", got, want)
	}
}

func TestGetAgentsForProvider_Anthropic(t *testing.T) {
	got := GetAgentsForProvider("anthropic")
	want := []string{"claude"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("GetAgentsForProvider(\"anthropic\") = %v, want %v", got, want)
	}
}

func TestGetAgentsForProvider_UnknownProvider(t *testing.T) {
	got := GetAgentsForProvider("unknown")
	if got != nil {
		t.Errorf("GetAgentsForProvider(\"unknown\") = %v, want nil", got)
	}
}

// TestGetAgentsForProvider_Immutability 验证返回的切片不被调用方修改影响
func TestGetAgentsForProvider_Immutability(t *testing.T) {
	agents := GetAgentsForProvider("deepseek")
	// 修改返回的切片
	if len(agents) > 0 {
		agents[0] = "mutated"
	}

	// 再次查询应不受影响
	got := GetAgentsForProvider("deepseek")
	if got[0] == "mutated" {
		t.Error("GetAgentsForProvider() 返回的切片应不可变")
	}
}
