package terminal

import (
	"fmt"
)

// ANSI color codes
const (
	ColorReset  = "\033[0m"
	ColorBlue   = "\033[34m"
	ColorBright = "\033[1m"
)

// Blue returns a string wrapped in blue ANSI color codes
func Blue(text string) string {
	if !supportsColor() {
		return text
	}
	return fmt.Sprintf("%s%s%s", ColorBlue, text, ColorReset)
}

// BrightBlue returns a string wrapped in bright blue ANSI color codes
func BrightBlue(text string) string {
	if !supportsColor() {
		return text
	}
	return fmt.Sprintf("%s%s%s%s", ColorBright, ColorBlue, text, ColorReset)
}

// supportsColor checks if the terminal supports ANSI colors
func supportsColor() bool {
	// Similar check to isXterm, but for color support
	return isXterm()
}
