package shell

import (
	"os"
)

// Detect returns the preferred shell for the current environment.
// It checks for available shells in the following order:
// 1. /bin/zsh
// 2. /bin/bash
// 3. /bin/sh (fallback)
func Detect() string {
	// Try common shells in order of preference
	if _, err := os.Stat("/bin/zsh"); err == nil {
		return "/bin/zsh"
	}
	if _, err := os.Stat("/bin/bash"); err == nil {
		return "/bin/bash"
	}

	// Fallback to sh
	return "/bin/sh"
}
