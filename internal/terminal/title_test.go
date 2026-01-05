package terminal

import (
	"os"
	"testing"
)

func TestIsXterm(t *testing.T) {
	tests := []struct {
		name     string
		term     string
		expected bool
	}{
		{"xterm", "xterm", true},
		{"xterm-256color", "xterm-256color", true},
		{"screen", "screen", true},
		{"tmux", "tmux-256color", true},
		{"rxvt", "rxvt-unicode", true},
		{"linux", "linux", true},
		{"dumb", "dumb", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original TERM value
			originalTerm := os.Getenv("TERM")
			defer os.Setenv("TERM", originalTerm)

			// Set TERM to test value
			os.Setenv("TERM", tt.term)

			result := isXterm()
			if result != tt.expected {
				t.Errorf("isXterm() with TERM=%q: got %v, want %v", tt.term, result, tt.expected)
			}
		})
	}
}

func TestSetTitle(t *testing.T) {
	// Save original TERM value
	originalTerm := os.Getenv("TERM")
	defer os.Setenv("TERM", originalTerm)

	// Test with xterm
	os.Setenv("TERM", "xterm")
	// Just verify it doesn't panic
	SetTitle("Test Title")

	// Test with non-xterm terminal
	os.Setenv("TERM", "dumb")
	// Should not print anything but shouldn't panic
	SetTitle("Test Title")
}

func TestGetTitle(t *testing.T) {
	// Save original TERM value
	originalTerm := os.Getenv("TERM")
	defer os.Setenv("TERM", originalTerm)

	// Test with xterm (may or may not have xdotool)
	os.Setenv("TERM", "xterm")
	title := GetTitle()
	// We can't assert a specific value since xdotool may not be available
	// Just verify it doesn't panic and returns a string (possibly empty)
	_ = title

	// Test with non-xterm terminal
	os.Setenv("TERM", "dumb")
	title = GetTitle()
	if title != "" {
		t.Errorf("GetTitle() with TERM=dumb: got %q, want empty string", title)
	}
}
