package generation

import "testing"

func TestNormalizeLanguage(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"supported english", "en", "en"},
		{"supported italian", "it", "it"},
		{"unsupported french falls back to english", "fr", "en"},
		{"unsupported german falls back to english", "de", "en"},
		{"empty falls back to english", "", "en"},
		{"case-insensitive supported code", "IT", "it"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NormalizeLanguage(tt.in); got != tt.want {
				t.Errorf("NormalizeLanguage(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
