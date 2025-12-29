package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"giverny/internal/docker"
	"giverny/internal/git"
)

type Config struct {
	TaskID        string
	Prompt        string
	BaseImage     string
	DockerArgs    string
	IsInnie       bool
	GitServerPort int
	Debug         bool
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
	if err := docker.BuildInnieImage(); err != nil {
		return fmt.Errorf("failed to build innie image: %w", err)
	}

	// Build giverny-main Docker image
	if err := docker.BuildMainImage(config.BaseImage); err != nil {
		return fmt.Errorf("failed to build main image: %w", err)
	}

	fmt.Printf("Running Outie for task: %s\n", config.TaskID)
	fmt.Printf("Prompt: %s\n", config.Prompt)
	fmt.Printf("Base image: %s\n", config.BaseImage)
	if config.DockerArgs != "" {
		fmt.Printf("Docker args: %s\n", config.DockerArgs)
	}

	// Run the container with Innie
	exitCode, err := docker.RunContainer(config.TaskID, config.Prompt, gitPort, config.DockerArgs)
	if err != nil {
		return fmt.Errorf("container failed: %w", err)
	}

	if exitCode != 0 {
		return fmt.Errorf("container exited with code %d", exitCode)
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

	return nil
}
