package outie

import (
	"fmt"
	"os"
	"path/filepath"

	"giverny/internal/docker"
	"giverny/internal/git"
	"giverny/internal/terminal"
)

// Config holds the configuration for the Outie
type Config struct {
	TaskID          string
	Prompt          string
	BaseImage       string
	DockerArgs      string
	AgentArgs       string
	Debug           bool
	ShowBuildOutput bool
	ExistingBranch  bool
	AllowDirty      bool
}

// Run executes the Outie workflow
func Run(config Config) error {
	// Save the current terminal title and set it to "Giverny: TASK-ID"
	originalTitle := terminal.GetTitle()
	terminal.SetTitle(fmt.Sprintf("Giverny: %s", config.TaskID))

	// Restore the original title on exit
	defer func() {
		if originalTitle != "" {
			terminal.SetTitle(originalTitle)
		}
	}()

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

	// Check for uncommitted changes before creating branch (unless --allow-dirty is set)
	if !config.AllowDirty && !config.ExistingBranch {
		isDirty, err := git.IsWorkspaceDirty()
		if err != nil {
			return fmt.Errorf("failed to check workspace status: %w", err)
		}
		if isDirty {
			return fmt.Errorf("working directory has uncommitted changes. Commit or stash them first, or use --allow-dirty flag")
		}
	}

	// Create or validate git branch for this task
	branchName := fmt.Sprintf("giverny/%s", config.TaskID)
	if config.ExistingBranch {
		// Validate that the branch exists
		exists, err := git.BranchExists(branchName)
		if err != nil {
			return fmt.Errorf("failed to check if branch exists: %w", err)
		}
		if !exists {
			return fmt.Errorf("branch '%s' does not exist", branchName)
		}
		fmt.Printf("Using existing branch: %s\n", branchName)
	} else {
		// Create new branch
		if err := git.CreateBranch(branchName); err != nil {
			return fmt.Errorf("failed to create branch: %w", err)
		}
		fmt.Printf("Created branch: %s\n", branchName)
	}

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
	if config.Debug {
		fmt.Printf("Started git server on port: %d\n", gitPort)
	}

	// Build giverny Docker image
	if err := docker.BuildImage(config.BaseImage, config.ShowBuildOutput, config.Debug); err != nil {
		return fmt.Errorf("failed to build image: %w", err)
	}

	if config.Debug {
		fmt.Printf("Running Outie for task: %s\n", config.TaskID)
		fmt.Printf("Prompt: %s\n", config.Prompt)
		fmt.Printf("Base image: %s\n", config.BaseImage)
		if config.DockerArgs != "" {
			fmt.Printf("Docker args: %s\n", config.DockerArgs)
		}
	}

	// Run the container with Innie
	exitCode, err := docker.RunContainer(config.TaskID, config.Prompt, gitPort, config.DockerArgs, config.AgentArgs, config.Debug)

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
	if config.Debug {
		fmt.Printf("Removing container...\n")
	}
	if err := docker.RemoveContainer(containerName); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to remove container: %v\n", err)
	}

	// Get commit range for merge/cherry-pick instructions
	firstCommit, lastCommit, err := git.GetBranchCommitRange(branchName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to get commit range: %v\n", err)
	} else if firstCommit != "" && lastCommit != "" {
		// Only show merge instructions if branch has commits
		fmt.Printf("\nTo merge the changes into your main branch:\n")
		fmt.Printf("  %s\n", terminal.Blue(fmt.Sprintf("git merge --ff-only %s", branchName)))

		// Convert to short hashes for display
		firstShort := git.GetShortHash(firstCommit)
		lastShort := git.GetShortHash(lastCommit)

		fmt.Printf("\nOr to cherry-pick the changes:\n")
		if firstCommit == lastCommit {
			// Only one commit
			fmt.Printf("  %s\n", terminal.Blue(fmt.Sprintf("git cherry-pick %s", firstShort)))
		} else {
			// Multiple commits
			fmt.Printf("  %s\n", terminal.Blue(fmt.Sprintf("git cherry-pick %s^..%s", firstShort, lastShort)))
		}

		fmt.Printf("\nTo delete the branch:\n")
		fmt.Printf("  %s\n", terminal.Blue(fmt.Sprintf("git branch -D %s", branchName)))
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
