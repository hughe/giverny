package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

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

func main() {
	config := parseArgs(flag.CommandLine, os.Args[1:])

	var err error
	if config.IsInnie {
		err = runInnie(config)
	} else {
		err = runOutie(config)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func parseArgs(flags *flag.FlagSet, args []string) Config {
	var config Config

	// Define flags
	flags.StringVar(&config.BaseImage, "base-image", "giverny:latest", "Docker base image")
	flags.StringVar(&config.DockerArgs, "docker-args", "", "Additional docker run arguments")
	flags.BoolVar(&config.IsInnie, "innie", false, "Flag indicating running inside container")
	flags.IntVar(&config.GitServerPort, "git-server-port", 0, "Port for git daemon connection")
	flags.BoolVar(&config.Debug, "debug", false, "Enable debug output")
	flags.BoolVar(&config.ShowBuildOutput, "show-build-output", false, "Show docker build output")

	// Custom usage message
	flags.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: giverny [OPTIONS] TASK-ID [PROMPT]\n\n")
		fmt.Fprintf(os.Stderr, "Giverny - Containerized system for running Claude Code safely\n\n")
		fmt.Fprintf(os.Stderr, "Arguments:\n")
		fmt.Fprintf(os.Stderr, "  TASK-ID    Task identifier (required)\n")
		fmt.Fprintf(os.Stderr, "  PROMPT     Prompt for Claude Code (optional, defaults to 'Please work on TASK-ID.')\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flags.PrintDefaults()
	}

	flags.Parse(args)

	// Get positional arguments
	positionalArgs := flags.Args()
	if len(positionalArgs) < 1 {
		fmt.Fprintf(os.Stderr, "Error: TASK-ID is required\n\n")
		flags.Usage()
		os.Exit(1)
	}

	config.TaskID = positionalArgs[0]

	// Set prompt - default or from argument
	if len(positionalArgs) >= 2 {
		config.Prompt = positionalArgs[1]
	} else {
		config.Prompt = fmt.Sprintf("Please work on %s.", config.TaskID)
	}

	// Validate innie-specific requirements
	if config.IsInnie && config.GitServerPort == 0 {
		fmt.Fprintf(os.Stderr, "Error: --git-server-port is required when --innie is set\n")
		os.Exit(1)
	}

	return config
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

	// Build giverny-innie Docker image
	if err := docker.BuildInnieImage(config.ShowBuildOutput); err != nil {
		return fmt.Errorf("failed to build innie image: %w", err)
	}

	// Build giverny-main Docker image
	if err := docker.BuildMainImage(config.BaseImage, config.ShowBuildOutput); err != nil {
		return fmt.Errorf("failed to build main image: %w", err)
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
	if err := setupWorkspace(branchName); err != nil {
		return fmt.Errorf("failed to setup workspace: %w", err)
	}

	// Execute Claude Code with the prompt
	if err := executeClaude(config.Prompt); err != nil {
		return fmt.Errorf("failed to execute Claude: %w", err)
	}

	// Post-Claude menu loop
	if err := postClaudeMenu(); err != nil {
		return fmt.Errorf("menu error: %w", err)
	}

	// Push branch and exit
	if err := pushBranchAndExit(branchName, config.GitServerPort); err != nil {
		return fmt.Errorf("failed to push branch: %w", err)
	}

	return nil
}

// setupWorkspace creates /app, checks out the branch, and creates a START label
func setupWorkspace(branchName string) error {
	// Create /app directory
	if err := os.MkdirAll("/app", 0755); err != nil {
		return fmt.Errorf("failed to create /app directory: %w", err)
	}

	// Checkout the branch to /app using git worktree
	cmd := exec.Command("git", "-C", "/git", "worktree", "add", "/app", branchName)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to checkout branch %s to /app: %w", branchName, err)
	}
	fmt.Printf("Checked out branch %s to /app\n", branchName)

	// Create giverny/START label branch to mark where we started
	startLabel := branchName + "/START"
	cmd = exec.Command("git", "-C", "/app", "branch", startLabel)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create START label branch %s: %w", startLabel, err)
	}
	fmt.Printf("Created START label: %s\n", startLabel)

	return nil
}

// executeClaude runs Claude Code with the given prompt in /app
func executeClaude(prompt string) error {
	fmt.Printf("Executing Claude Code...\n")

	cmd := exec.Command("claude", "--dangerously-skip-permissions", prompt)
	cmd.Dir = "/app"
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

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
		dirty, err := isWorkspaceDirty()
		if err != nil {
			return fmt.Errorf("failed to check workspace status: %w", err)
		}

		// Show menu
		fmt.Println("\nWhat would you like to do?")
		fmt.Println("  [c] Commit changes")
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
			if err := commitChanges(); err != nil {
				fmt.Fprintf(os.Stderr, "Error committing: %v\n", err)
				continue
			}
		case "s":
			if err := startShell(); err != nil {
				fmt.Fprintf(os.Stderr, "Error starting shell: %v\n", err)
				continue
			}
		case "r":
			// Restart Claude - just return and let the loop continue
			return executeClaude(os.Args[len(os.Args)-1])
		case "x":
			// Only allow exit if workspace is clean
			if dirty {
				fmt.Println("⚠️  Cannot exit with uncommitted changes. Please commit or discard them first.")
				continue
			}
			return nil
		default:
			fmt.Println("Invalid choice. Please enter c, s, r, or x.")
		}
	}
}

// isWorkspaceDirty checks if there are uncommitted changes in /app
func isWorkspaceDirty() (bool, error) {
	cmd := exec.Command("git", "status", "--porcelain")
	// Use /app if it exists, otherwise use current directory (for testing)
	if _, err := os.Stat("/app"); err == nil {
		cmd.Dir = "/app"
	}
	output, err := cmd.Output()
	if err != nil {
		return false, err
	}
	return len(output) > 0, nil
}

// commitChanges commits all changes in /app
func commitChanges() error {
	fmt.Println("Committing changes...")

	// Add all changes
	cmd := exec.Command("git", "add", "-A")
	cmd.Dir = "/app"
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git add failed: %w", err)
	}

	// Commit with a prompt for message
	fmt.Print("Commit message: ")
	var message string
	fmt.Scanln(&message)
	if message == "" {
		message = "Work in progress"
	}

	cmd = exec.Command("git", "commit", "-m", message)
	cmd.Dir = "/app"
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git commit failed: %w", err)
	}

	fmt.Println("✓ Changes committed")
	return nil
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

// pushBranchAndExit pushes the branch to the git server and exits cleanly
func pushBranchAndExit(branchName string, gitServerPort int) error {
	fmt.Printf("Pushing %s to git server...\n", branchName)

	// Construct the git server URL
	gitServerURL := fmt.Sprintf("git://host.docker.internal:%d/git", gitServerPort)

	// Push the branch
	cmd := exec.Command("git", "push", gitServerURL, branchName)
	cmd.Dir = "/app"
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git push failed: %w", err)
	}

	fmt.Printf("✓ Successfully pushed %s\n", branchName)
	return nil
}
