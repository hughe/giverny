package innie

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"giverny/internal/beads"
	"giverny/internal/git"
	"giverny/internal/interactive"
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
	if err := beads.Initialize(config.Debug); err != nil {
		// Log warning but don't fail - beads initialization is optional
		fmt.Fprintf(os.Stderr, "Warning: beads initialization failed: %v\n", err)
	}

	// Execute Claude Code with the prompt
	if err := executeClaude(config.Prompt, config.AgentArgs, true); err != nil {
		return fmt.Errorf("failed to execute Claude: %w", err)
	}

	// Post-Claude menu loop
	// Create a wrapper function that captures agentArgs
	executeClaudeWrapper := func(prompt string, isInteractive bool) error {
		return executeClaude(prompt, config.AgentArgs, isInteractive)
	}
	if err := interactive.PostClaudeMenu(executeClaudeWrapper, nil); err != nil {
		return fmt.Errorf("menu error: %w", err)
	}

	// Push branch and exit
	if err := git.PushBranch(branchName, config.GitServerPort, config.Debug); err != nil {
		return fmt.Errorf("failed to push branch: %w", err)
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

