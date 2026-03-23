package beads

import (
	"fmt"
	"os"
	"os/exec"
)

// Initialize initializes the beads database if .beads directory exists and br is available
func Initialize(debug bool) error {
	// Check if .beads directory exists
	beadsPath := "/app/.beads"
	if _, err := os.Stat(beadsPath); os.IsNotExist(err) {
		// .beads directory doesn't exist, skip initialization
		return nil
	}

	// Check if br command is available
	if _, err := exec.LookPath("br"); err != nil {
		// br is not available, skip initialization
		return nil
	}

	if debug {
		fmt.Println("Initializing beads database...")
	}

	// Run br init
	cmd := exec.Command("br", "init")
	cmd.Dir = "/app"
	if debug {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("br init failed: %w", err)
	}

	if debug {
		fmt.Println("Beads initialization completed")
	}
	return nil
}
