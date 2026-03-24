package docker

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"giverny/internal/cmdutil"
	"giverny/internal/terminal"
)

// RunContainer starts the giverny-main container with Innie
// Returns the exit code of the container
func RunContainer(taskID, slug, prompt string, gitPort int, dockerArgs, agentArgs string, debug, useAmp bool) (int, error) {
	// Generate a container name based on task ID and slug
	var containerName string
	if slug != "" {
		containerName = fmt.Sprintf("giverny-%s-%s", taskID, slug)
	} else {
		containerName = fmt.Sprintf("giverny-%s", taskID)
	}

	// Get home directory for mounting config
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return 0, fmt.Errorf("failed to get home directory: %w", err)
	}

	// Build the docker run command
	args := []string{
		"run",
		"-it",
		"--name", containerName,
	}

	if useAmp {
		// Validate AMP_API_KEY
		if os.Getenv("AMP_API_KEY") == "" {
			return 0, fmt.Errorf("AMP_API_KEY not set")
		}
		args = append(args, "--env", "AMP_API_KEY")

		// Mount Amp config directory
		ampConfigDir := filepath.Join(homeDir, ".config", "amp")
		if _, err := os.Stat(ampConfigDir); err == nil {
			args = append(args, "-v", fmt.Sprintf("%s:/root/.config/amp", ampConfigDir))
		}
	} else {
		// Validate CLAUDE_CODE_OAUTH_TOKEN
		if os.Getenv("CLAUDE_CODE_OAUTH_TOKEN") == "" {
			return 0, fmt.Errorf("CLAUDE_CODE_OAUTH_TOKEN not set")
		}
		args = append(args,
			"--env", "CLAUDE_CODE_OAUTH_TOKEN",
			"-v", fmt.Sprintf("%s/.claude:/root/.claude", homeDir),
			"-v", fmt.Sprintf("%s/.claude.json:/root/.claude.json", homeDir),
		)
	}

	// Add any additional docker args
	if dockerArgs != "" {
		// Split dockerArgs and add them
		additionalArgs := strings.Fields(dockerArgs)
		args = append(args, additionalArgs...)
	}

	// Specify the image
	args = append(args, "giverny-main:latest")

	// Specify the command to run inside the container
	args = append(args, "giverny", "--innie", fmt.Sprintf("--git-server-port=%d", gitPort))

	// Add --amp flag if using Amp
	if useAmp {
		args = append(args, "--amp")
	}

	// Add debug flag if enabled
	if debug {
		args = append(args, "--debug")
	}

	// Add agent args if provided
	if agentArgs != "" {
		args = append(args, fmt.Sprintf("--agent-args=%s", agentArgs))
	}

	// Add positional arguments: taskID, slug, prompt
	// Always pass all three to avoid the prompt being parsed as a slug
	args = append(args, taskID, slug, prompt)

	cmd := exec.Command("docker", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	fmt.Printf("Starting container %s for task %s...\n", containerName, taskID)
	fmt.Printf("To start a shell in the container, run:\n")
	fmt.Printf("  %s\n\n", terminal.Blue(fmt.Sprintf("docker exec -it %s /bin/sh", containerName)))

	exitCode := 0
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return 0, fmt.Errorf("failed to run container: %w", err)
		}
	}

	return exitCode, nil
}

// RemoveContainer removes a Docker container by name
func RemoveContainer(containerName string) error {
	if err := cmdutil.RunCommand("docker", "rm", containerName); err != nil {
		return fmt.Errorf("failed to remove container %s: %w", containerName, err)
	}
	fmt.Printf("✓ Container removed\n")
	return nil
}
