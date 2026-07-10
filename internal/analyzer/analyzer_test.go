package analyzer

import (
	"testing"
)

func TestExtractPrefix_SingleSeparator(t *testing.T) {
	tests := []struct {
		key       string
		separator string
		depth     int
		want      string
	}{
		{"user:profile:123", ":", 1, "user"},
		{"user:profile:123", ":", 2, "user:profile"},
		{"user:profile:123", ":", 3, "user:profile:123"},
		{"user:profile:123", ":", 0, "user:profile:123"},
		{"user_profile-123", ":", 1, "user_profile-123"},
		{"a.b.c", ".", 1, "a"},
		{"a.b.c", ".", 2, "a.b"},
		{"single", ":", 1, "single"},
		{"", ":", 1, ""},
	}
	for _, tt := range tests {
		got := extractPrefix(tt.key, tt.separator, tt.depth)
		if got != tt.want {
			t.Errorf("extractPrefix(%q, %q, %d) = %q, want %q",
				tt.key, tt.separator, tt.depth, got, tt.want)
		}
	}
}

func TestExtractPrefix_AutoSeparator(t *testing.T) {
	tests := []struct {
		key   string
		depth int
		want  string
	}{
		{"user:profile:123", 1, "user"},
		{"user_profile-123", 1, "user"},
		{"user-profile_123", 1, "user"},
		{"user:profile-123", 1, "user"},
		{"user:profile:123", 2, "user:profile"},
		{"user_profile-123", 2, "user_profile-123"},  // depth=2 = all parts
		{"user-profile_123", 2, "user-profile_123"},  // depth=2 = all parts
		{"singlekey", 1, "singlekey"},
		{"", 1, ""},
	}
	for _, tt := range tests {
		got := extractPrefix(tt.key, "auto", tt.depth)
		if got != tt.want {
			t.Errorf("extractPrefix(%q, \"auto\", %d) = %q, want %q",
				tt.key, tt.depth, got, tt.want)
		}
	}
}

func TestExtractPrefix_EmptySeparator(t *testing.T) {
	// Empty separator should behave like "auto"
	got := extractPrefix("user_name", "", 1)
	if got != "user" {
		t.Errorf("expected \"user\", got %q", got)
	}
}

func TestExtractPrefixMulti(t *testing.T) {
	tests := []struct {
		key   string
		depth int
		want  string
	}{
		{"user:profile:123", 1, "user"},
		{"user_profile:123", 1, "user"},
		{"user-profile:123", 1, "user"},
		{"no-separator", 1, "no"},  // splits on "-", depth=1
		{"a:b:c:d", 3, "a:b:c"},
		{"a_b_c", 2, "a_b"},
		{"a-b-c", 1, "a"},
	}
	for _, tt := range tests {
		got := extractPrefixMulti(tt.key, tt.depth)
		if got != tt.want {
			t.Errorf("extractPrefixMulti(%q, %d) = %q, want %q",
				tt.key, tt.depth, got, tt.want)
		}
	}
}
