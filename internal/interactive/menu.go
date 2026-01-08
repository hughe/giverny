package interactive

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"giverny/internal/git"
	"giverny/internal/shell"
)

// PostClaudeMenu shows an interactive menu for committing, restarting, or exiting.
// It returns nil when the user chooses to exit with a clean workspace.
// The executeClaude parameter is a function that executes Claude Code with a given prompt.
func PostClaudeMenu(executeClaude func(prompt string, interactive bool) error, reader io.Reader) error {
	if reader == nil {
		reader = os.Stdin
	}

	for {
		// Check if there are uncommitted changes
		dirty, err := git.IsWorkspaceDirty()
		if err != nil {
			return fmt.Errorf("failed to check workspace status: %w", err)
		}

		// Show menu
		fmt.Println("\nWhat would you like to do?")
		fmt.Println("  [c] Ask Claude to Commit the changes")
		fmt.Println("  [d] Start diffreviewer")
		fmt.Println("  [s] Start a shell")
		fmt.Println("  [r] Restart Claude")
		fmt.Println("  [x] Exit")
		if dirty {
			fmt.Println("⚠️  You have uncommitted changes")
		}
		fmt.Print("Choice: ")

		// Read user input
		var choice string
		fmt.Fscanln(reader, &choice)

		switch choice {
		case "c":
			return executeClaude("Commit the changes", false)
		case "d":
			if err := runDiffreviewer(executeClaude); err != nil {
				fmt.Fprintf(os.Stderr, "Error running diffreviewer: %v\n", err)
				continue
			}
		case "s":
			if err := startShell(); err != nil {
				fmt.Fprintf(os.Stderr, "Error starting shell: %v\n", err)
				continue
			}
		case "r":
			// Restart Claude - use the last argument as the prompt
			return executeClaude(os.Args[len(os.Args)-1], true)
		case "x":
			// Only allow exit if workspace is clean
			if dirty {
				fmt.Println("⚠️  Cannot exit with uncommitted changes. Please commit or discard them first.")
				continue
			}
			return nil
		default:
			fmt.Println("Invalid choice. Please enter c, d, s, r, or x.")
		}
	}
}

// startShell starts an interactive shell in /app
func startShell() error {
	// Determine which shell to use
	shellPath := shell.Detect()

	fmt.Printf("Starting %s in /app (type 'exit' to return to menu)...\n", shellPath)

	cmd := exec.Command(shellPath)
	cmd.Dir = "/app"
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("shell exited with error: %w", err)
	}

	return nil
}

// runDiffreviewer runs diffreviewer and if notes are found, asks Claude to fix them
func runDiffreviewer(executeClaude func(prompt string, interactive bool) error) error {
	fmt.Println("Starting diffreviewer...")

	// Run diffreviewer and capture output
	cmd := exec.Command("diffreviewer")
	cmd.Dir = "/app"
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("diffreviewer exited with error: %w", err)
	}

	// Parse the notes from the output
	// The output format has notes between the separator lines
	notes := parseNotesFromOutput(string(output))

	// If notes are empty, just return
	if notes == "" {
		fmt.Println("No review notes found.")
		return nil
	}

	// Write notes to file
	notesPath := "/tmp/diffreviewer-notes.md"
	if err := os.WriteFile(notesPath, []byte(notes), 0644); err != nil {
		return fmt.Errorf("failed to write notes file: %w", err)
	}
	defer os.Remove(notesPath) // Clean up notes file after Claude runs

	fmt.Printf("Review notes written to %s\n", notesPath)
	fmt.Println("Starting Claude to fix the issues...")

	// Start Claude with the notes
	return executeClaude("Please fix the issues in @/tmp/diffreviewer-notes.md", true)
}

// parseNotesFromOutput extracts notes from diffreviewer output
func parseNotesFromOutput(output string) string {
	// Find the notes section between the separator lines
	lines := strings.Split(output, "\n")
	inNotes := false
	var noteLines []string

	for _, line := range lines {
		if strings.Contains(line, "================================================================================") {
			if inNotes {
				// End of notes section
				break
			}
			// Start of notes section
			inNotes = true
			continue
		}
		if inNotes {
			noteLines = append(noteLines, line)
		}
	}

	notes := strings.TrimSpace(strings.Join(noteLines, "\n"))

	// Check if notes section only contains header
	if notes == "# Review Notes" || notes == "" {
		return ""
	}

	return notes
}
