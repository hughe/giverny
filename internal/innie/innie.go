package innie

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"giverny/internal/git"
)

// Config holds the configuration for the Innie
type Config struct {
	TaskID        string
	Prompt        string
	GitServerPort int
	AgentArgs     string
	Debug         bool
}

// Run executes the Innie workflow
func Run(config Config) error {
	if config.Debug {
		fmt.Printf("Running Innie for task: %s\n", config.TaskID)
		fmt.Printf("Prompt: %s\n", config.Prompt)
		fmt.Printf("Git server port: %d\n", config.GitServerPort)
	}

	// Clone the repository from Outie's git server
	if config.Debug {
		fmt.Printf("Cloning repository from git server...\n")
	}
	if err := git.CloneRepo(config.GitServerPort, config.Debug); err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}
	if config.Debug {
		fmt.Printf("Repository cloned successfully to /git\n")
	}

	// List /git directory contents to verify clone (debug mode only)
	if config.Debug {
		fmt.Printf("\nContents of /git:\n")
		cmd := exec.Command("ls", "-la", "/git")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to list /git directory: %v\n", err)
		}
	}

	// Set up workspace in /app
	branchName := fmt.Sprintf("giverny/%s", config.TaskID)
	if err := git.SetupWorkspace(branchName, config.Debug); err != nil {
		return fmt.Errorf("failed to setup workspace: %w", err)
	}

	// Change to /app directory for all subsequent operations
	if err := os.Chdir("/app"); err != nil {
		return fmt.Errorf("failed to change to /app directory: %w", err)
	}

	// Initialize beads if .beads directory exists and bd is available
	if err := initializeBeads(config.Debug); err != nil {
		// Log warning but don't fail - beads initialization is optional
		fmt.Fprintf(os.Stderr, "Warning: beads initialization failed: %v\n", err)
	}

	// Execute Claude Code with the prompt
	if err := executeClaude(config.Prompt, config.AgentArgs, true); err != nil {
		return fmt.Errorf("failed to execute Claude: %w", err)
	}

	// Post-Claude menu loop
	if err := postClaudeMenu(config.AgentArgs); err != nil {
		return fmt.Errorf("menu error: %w", err)
	}

	// Push branch and exit
	if err := git.PushBranch(branchName, config.GitServerPort); err != nil {
		return fmt.Errorf("failed to push branch: %w", err)
	}

	return nil
}

// initializeBeads initializes the beads database if .beads directory exists and bd is available
func initializeBeads(debug bool) error {
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

// executeClaude runs Claude Code with the given prompt in /app
func executeClaude(prompt, agentArgs string, interactive bool) error {
	if interactive {
		fmt.Printf("Executing Claude Code...\n")
	} else {
		fmt.Printf("Executing Claude Code in non-interactive mode...\n")
	}

	args := []string{"--dangerously-skip-permissions", "--allow-dangerously-skip-permissions"}
	if !interactive {
		args = append(args, "--print")
	}

	// Parse and add agent args if provided
	if agentArgs != "" {
		additionalArgs := strings.Fields(agentArgs)
		args = append(args, additionalArgs...)
	}

	args = append(args, prompt)

	cmd := exec.Command("claude", args...)
	cmd.Dir = "/app"
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Env = append(os.Environ(), "IS_SANDBOX=1")

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Claude exited with error: %w", err)
	}

	fmt.Printf("Claude completed successfully\n")
	return nil
}

// postClaudeMenu shows an interactive menu for committing, restarting, or exiting
func postClaudeMenu(agentArgs string) error {
	reader := os.Stdin

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
			return executeClaude("Commit the changes", agentArgs, false)
		case "d":
			if err := runDiffreviewer(agentArgs); err != nil {
				fmt.Fprintf(os.Stderr, "Error running diffreviewer: %v\n", err)
				continue
			}
		case "s":
			if err := startShell(); err != nil {
				fmt.Fprintf(os.Stderr, "Error starting shell: %v\n", err)
				continue
			}
		case "r":
			// Restart Claude - just return and let the loop continue
			return executeClaude(os.Args[len(os.Args)-1], agentArgs, true)
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
	// Determine which shell to use (prefer zsh, then bash, then sh)
	shell := "/bin/sh"
	if _, err := os.Stat("/bin/zsh"); err == nil {
		shell = "/bin/zsh"
	} else if _, err := os.Stat("/bin/bash"); err == nil {
		shell = "/bin/bash"
	}

	fmt.Printf("Starting %s in /app (type 'exit' to return to menu)...\n", shell)

	cmd := exec.Command(shell)
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
func runDiffreviewer(agentArgs string) error {
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
	return executeClaude("Please fix the issues in @/tmp/diffreviewer-notes.md", agentArgs, true)
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
