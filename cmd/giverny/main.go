package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
	"giverny"
	"giverny/internal/docker"
	"giverny/internal/git"
)

func init() {
	// Initialize the embedded source for the docker package
	docker.EmbeddedSource = giverny.Source
}

type Config struct {
	TaskID          string
	Prompt          string
	BaseImage       string
	DockerArgs      string
	IsInnie         bool
	GitServerPort   int
	Debug           bool
	ShowBuildOutput bool
}

var (
	config Config
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "giverny [OPTIONS] TASK-ID [PROMPT]",
		Short: "Containerized system for running Claude Code safely",
		Long:  "Giverny creates isolated Docker environments where Claude Code can work on tasks without affecting the host system.",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			config.TaskID = args[0]

			// Validate TASK-ID
			if err := validateTaskID(config.TaskID); err != nil {
				return fmt.Errorf("invalid TASK-ID: %w", err)
			}

			// Set prompt - default or from argument
			if len(args) >= 2 {
				config.Prompt = args[1]
			} else {
				config.Prompt = fmt.Sprintf("Please work on %s.", config.TaskID)
			}

			// Validate innie-specific requirements
			if config.IsInnie && config.GitServerPort == 0 {
				return fmt.Errorf("--git-server-port is required when --innie is set")
			}

			// Execute appropriate mode
			if config.IsInnie {
				return runInnie(config)
			}
			return runOutie(config)
		},
	}

	// Define flags
	rootCmd.Flags().StringVar(&config.BaseImage, "base-image", "giverny:latest", "Docker base image")
	rootCmd.Flags().StringVar(&config.DockerArgs, "docker-args", "", "Additional docker run arguments")
	rootCmd.Flags().BoolVar(&config.Debug, "debug", false, "Enable debug output")
	rootCmd.Flags().BoolVar(&config.ShowBuildOutput, "show-build-output", false, "Show docker build output")

	// Hidden flags (for internal use only)
	rootCmd.Flags().BoolVar(&config.IsInnie, "innie", false, "Internal flag for running inside container")
	rootCmd.Flags().IntVar(&config.GitServerPort, "git-server-port", 0, "Internal flag for git server port")
	rootCmd.Flags().MarkHidden("innie")
	rootCmd.Flags().MarkHidden("git-server-port")

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// validateTaskID ensures TASK-ID contains only characters valid in git branch names.
// Since we use the format "giverny/TASK-ID", the TASK-ID must not contain '/' and
// must follow git branch naming rules.
func validateTaskID(taskID string) error {
	if taskID == "" {
		return fmt.Errorf("TASK-ID cannot be empty")
	}

	// Regex for invalid characters in git branch names:
	// - No forward slash, backslash, space, control chars (0-31, 127)
	// - No ~, ^, :, ?, *, [
	invalidCharsPattern := regexp.MustCompile(`[/\\\s\x00-\x1f\x7f~^:?*\[]`)
	if match := invalidCharsPattern.FindString(taskID); match != "" {
		if match == "/" {
			return fmt.Errorf("TASK-ID cannot contain forward slash (/)")
		} else if match == "\\" {
			return fmt.Errorf("TASK-ID cannot contain backslash (\\)")
		} else if match == " " {
			return fmt.Errorf("TASK-ID cannot contain spaces")
		} else if match[0] < 32 || match[0] == 127 {
			return fmt.Errorf("TASK-ID cannot contain control characters")
		}
		return fmt.Errorf("TASK-ID cannot contain '%s'", match)
	}

	// Check for special invalid patterns
	if strings.Contains(taskID, "..") {
		return fmt.Errorf("TASK-ID cannot contain double dots (..)")
	}
	if strings.Contains(taskID, "@{") {
		return fmt.Errorf("TASK-ID cannot contain @{")
	}

	// Check if starts with dot
	if strings.HasPrefix(taskID, ".") {
		return fmt.Errorf("TASK-ID cannot start with a dot")
	}

	// Check if ends with .lock
	if strings.HasSuffix(taskID, ".lock") {
		return fmt.Errorf("TASK-ID cannot end with .lock")
	}

	return nil
}

// findProjectRoot finds the project root by looking for .git directory
func findProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// Walk up the directory tree looking for .git
	for {
		gitPath := filepath.Join(dir, ".git")
		if info, err := os.Stat(gitPath); err == nil && info.IsDir() {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached root without finding .git
			return "", fmt.Errorf("could not find .git directory in any parent directory")
		}
		dir = parent
	}
}

func runOutie(config Config) error {
	// Find project root and change to it
	projectRoot, err := findProjectRoot()
	if err != nil {
		return fmt.Errorf("failed to find project root: %w", err)
	}
	if err := os.Chdir(projectRoot); err != nil {
		return fmt.Errorf("failed to change to project root: %w", err)
	}

	// Validate CLAUDE_CODE_OAUTH_TOKEN is set
	if os.Getenv("CLAUDE_CODE_OAUTH_TOKEN") == "" {
		return fmt.Errorf("CLAUDE_CODE_OAUTH_TOKEN environment variable is not set.\nPlease set it with: export CLAUDE_CODE_OAUTH_TOKEN=your-token")
	}

	// Create git branch for this task
	branchName := fmt.Sprintf("giverny/%s", config.TaskID)
	if err := git.CreateBranch(branchName); err != nil {
		return fmt.Errorf("failed to create branch: %w", err)
	}
	fmt.Printf("Created branch: %s\n", branchName)

	// Start git server
	serverCmd, gitPort, err := git.StartServer(projectRoot)
	if err != nil {
		return fmt.Errorf("failed to start git server: %w", err)
	}
	// Ensure server is stopped on exit
	defer func() {
		if err := git.StopServer(serverCmd); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to stop git server: %v\n", err)
		}
	}()
	fmt.Printf("Started git server on port: %d\n", gitPort)

	// Build giverny Docker image
	if err := docker.BuildImage(config.BaseImage, config.ShowBuildOutput); err != nil {
		return fmt.Errorf("failed to build image: %w", err)
	}

	fmt.Printf("Running Outie for task: %s\n", config.TaskID)
	fmt.Printf("Prompt: %s\n", config.Prompt)
	fmt.Printf("Base image: %s\n", config.BaseImage)
	if config.DockerArgs != "" {
		fmt.Printf("Docker args: %s\n", config.DockerArgs)
	}

	// Run the container with Innie
	exitCode, err := docker.RunContainer(config.TaskID, config.Prompt, gitPort, config.DockerArgs, config.Debug)

	// Post-container cleanup
	containerName := fmt.Sprintf("giverny-%s", config.TaskID)

	if err != nil || exitCode != 0 {
		// On failure: keep container for debugging, print error
		fmt.Fprintf(os.Stderr, "\n❌ Task failed\n")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		} else {
			fmt.Fprintf(os.Stderr, "Container exited with code %d\n", exitCode)
		}
		fmt.Fprintf(os.Stderr, "Container '%s' has been kept for debugging\n", containerName)
		fmt.Fprintf(os.Stderr, "To inspect: docker logs %s\n", containerName)
		fmt.Fprintf(os.Stderr, "To remove: docker rm %s\n", containerName)

		if err != nil {
			return fmt.Errorf("container failed: %w", err)
		}
		return fmt.Errorf("container exited with code %d", exitCode)
	}

	// On success: remove container, print success
	fmt.Printf("\n✓ Task completed successfully\n")
	fmt.Printf("Removing container...\n")
	if err := docker.RemoveContainer(containerName); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to remove container: %v\n", err)
	}

	return nil
}

func runInnie(config Config) error {
	fmt.Printf("Running Innie for task: %s\n", config.TaskID)
	fmt.Printf("Prompt: %s\n", config.Prompt)
	fmt.Printf("Git server port: %d\n", config.GitServerPort)

	// Clone the repository from Outie's git server
	fmt.Printf("Cloning repository from git server...\n")
	if err := git.CloneRepo(config.GitServerPort); err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}
	fmt.Printf("Repository cloned successfully to /git\n")

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
	if err := git.SetupWorkspace(branchName); err != nil {
		return fmt.Errorf("failed to setup workspace: %w", err)
	}

	// Change to /app directory for all subsequent operations
	if err := os.Chdir("/app"); err != nil {
		return fmt.Errorf("failed to change to /app directory: %w", err)
	}

	// Execute Claude Code with the prompt
	if err := executeClaude(config.Prompt, true); err != nil {
		return fmt.Errorf("failed to execute Claude: %w", err)
	}

	// Post-Claude menu loop
	if err := postClaudeMenu(); err != nil {
		return fmt.Errorf("menu error: %w", err)
	}

	// Push branch and exit
	if err := git.PushBranch(branchName, config.GitServerPort); err != nil {
		return fmt.Errorf("failed to push branch: %w", err)
	}

	return nil
}

// executeClaude runs Claude Code with the given prompt in /app
func executeClaude(prompt string, interactive bool) error {
	if interactive {
		fmt.Printf("Executing Claude Code...\n")
	} else {
		fmt.Printf("Executing Claude Code in non-interactive mode...\n")
	}

	args := []string{"--dangerously-skip-permissions", "--allow-dangerously-skip-permissions"}
	if !interactive {
		args = append(args, "--print")
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
func postClaudeMenu() error {
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
			return executeClaude("Commit the changes", false)
		case "d":
			if err := runDiffreviewer(); err != nil {
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
	fmt.Println("Starting shell in /app (type 'exit' to return to menu)...")

	cmd := exec.Command("/bin/sh")
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
func runDiffreviewer() error {
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
