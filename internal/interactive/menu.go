package interactive

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"giverny/internal/ctrlsock"
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

// runDiffreviewer starts diffreviewer as a server, notifies outie to open a
// browser, and waits for diffreviewer to exit. If review notes are produced,
// asks the agent to fix them.
func runDiffreviewer(executeClaude func(prompt string, interactive bool) error) error {
	fmt.Println("Starting diffreviewer...")

	notesPath := "/tmp/diffreviewer-notes.md"

	cmd := exec.Command("diffreviewer", "-notes", notesPath)
	cmd.Dir = "/app"
	cmd.Stdin = os.Stdin

	// Capture stderr to detect the startup message with the port.
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}
	cmd.Stdout = os.Stdout

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start diffreviewer: %w", err)
	}

	// Read stderr line by line; forward to os.Stderr and watch for the
	// startup message so we can notify outie.
	scanner := bufio.NewScanner(stderrPipe)
	notified := false
	for scanner.Scan() {
		line := scanner.Text()
		fmt.Fprintln(os.Stderr, line)

		if !notified && strings.Contains(line, "DiffReviewer starting on") {
			// Extract the URL from: "DiffReviewer starting on http://localhost:PORT"
			if idx := strings.Index(line, "http"); idx >= 0 {
				url := strings.TrimSpace(line[idx:])
				if addr := ctrlsock.ContainerAddr(); addr != "" {
					if err := ctrlsock.Send(addr, "OPEN-DIFFR "+url); err != nil {
						fmt.Fprintf(os.Stderr, "Warning: failed to notify outie to open browser: %v\n", err)
					}
				}
			}
			notified = true
		}
	}

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("diffreviewer exited with error: %w", err)
	}

	// Check if notes file was produced
	notesData, err := os.ReadFile(notesPath)
	if err != nil {
		// No notes file means no review notes
		fmt.Println("No review notes found.")
		return nil
	}
	defer os.Remove(notesPath)

	notes := strings.TrimSpace(string(notesData))
	if notes == "" || notes == "# Review Notes" {
		fmt.Println("No review notes found.")
		return nil
	}

	fmt.Printf("Review notes written to %s\n", notesPath)
	fmt.Println("Starting agent to fix the issues...")

	return executeClaude("Please fix the issues in @/tmp/diffreviewer-notes.md", true)
}


