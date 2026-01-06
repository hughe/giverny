package beads

import (
	"fmt"
	"os"
	"os/exec"
)

// Initialize initializes the beads database if .beads directory exists and bd is available
func Initialize(debug bool) error {
	// Check if .beads directory exists
	beadsPath := "/app/.beads"
	if _, err := os.Stat(beadsPath); os.IsNotExist(err) {
		// .beads directory doesn't exist, skip initialization
		return nil
	}

	// Check if bd command is available
	if _, err := exec.LookPath("bd"); err != nil {
		// bd is not available, skip initialization
		return nil
	}

	// Check if AGENTS.md exists before running bd init
	agentsPath := "/app/AGENTS.md"
	_, err := os.Stat(agentsPath)
	agentsExistedBefore := err == nil

	if debug {
		fmt.Println("Initializing beads database...")
	}

	// Run bd init
	cmd := exec.Command("bd", "init")
	cmd.Dir = "/app"
	if debug {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("bd init failed: %w", err)
	}

	// Handle AGENTS.md based on whether it existed before
	if _, err := os.Stat(agentsPath); err == nil {
		// AGENTS.md exists after bd init
		if agentsExistedBefore {
			// It existed before, restore it from git
			if debug {
				fmt.Println("Restoring AGENTS.md from git...")
			}
			cmd := exec.Command("git", "restore", "AGENTS.md")
			cmd.Dir = "/app"
			if debug {
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
			}
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("failed to restore AGENTS.md: %w", err)
			}
		} else {
			// It didn't exist before, delete what bd init created
			if debug {
				fmt.Println("Deleting AGENTS.md created by bd init...")
			}
			if err := os.Remove(agentsPath); err != nil {
				return fmt.Errorf("failed to delete AGENTS.md: %w", err)
			}
		}
	}

	if debug {
		fmt.Println("Beads initialization completed")
	}
	return nil
}
