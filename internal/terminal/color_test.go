package terminal

import (
	"os"
	"testing"
)

func TestBlue(t *testing.T) {
	tests := []struct {
		name     string
		termEnv  string
		input    string
		expected string
	}{
		{
			name:     "with xterm",
			termEnv:  "xterm-256color",
			input:    "test text",
			expected: "\033[34mtest text\033[0m",
		},
		{
			name:     "with dumb terminal",
			termEnv:  "dumb",
			input:    "test text",
			expected: "test text",
		},
		{
			name:     "empty string",
			termEnv:  "xterm",
			input:    "",
			expected: "\033[34m\033[0m",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set terminal type
			oldTerm := os.Getenv("TERM")
			os.Setenv("TERM", tt.termEnv)
			defer os.Setenv("TERM", oldTerm)

			result := Blue(tt.input)
			if result != tt.expected {
				t.Errorf("Blue(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestBrightBlue(t *testing.T) {
	tests := []struct {
		name     string
		termEnv  string
		input    string
		expected string
	}{
		{
			name:     "with xterm",
			termEnv:  "xterm-256color",
			input:    "test text",
			expected: "\033[1m\033[34mtest text\033[0m",
		},
		{
			name:     "with dumb terminal",
			termEnv:  "dumb",
			input:    "test text",
			expected: "test text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set terminal type
			oldTerm := os.Getenv("TERM")
			os.Setenv("TERM", tt.termEnv)
			defer os.Setenv("TERM", oldTerm)

			result := BrightBlue(tt.input)
			if result != tt.expected {
				t.Errorf("BrightBlue(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}
