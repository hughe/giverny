package terminal

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// SetTitle sets the terminal title using xterm escape sequences
func SetTitle(title string) {
	if !isXterm() {
		return
	}
	fmt.Printf("\033]0;%s\007", title)
}

// GetTitle retrieves the current terminal title
// Returns empty string if not in an xterm-compatible terminal or if retrieval fails
func GetTitle() string {
	if !isXterm() {
		return ""
	}

	// Try to get the title using xdotool as a fallback for some terminals
	// This is a best-effort approach as not all terminals support title retrieval
	cmd := exec.Command("xdotool", "getactivewindow", "getwindowname")
	output, err := cmd.Output()
	if err == nil {
		return strings.TrimSpace(string(output))
	}

	// If xdotool is not available or fails, return empty string
	// We can't reliably read the title in all terminals
	return ""
}

// isXterm checks if the terminal supports xterm escape sequences
func isXterm() bool {
	term := os.Getenv("TERM")
	// Check for common xterm-compatible terminals
	return strings.Contains(term, "xterm") ||
		strings.Contains(term, "screen") ||
		strings.Contains(term, "tmux") ||
		strings.Contains(term, "rxvt") ||
		term == "linux"
}
