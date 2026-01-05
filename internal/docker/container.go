package docker

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"giverny/internal/terminal"
)

// RunContainer starts the giverny-main container with Innie
// Returns the exit code of the container
func RunContainer(taskID, prompt string, gitPort int, dockerArgs, agentArgs string, debug bool) (int, error) {
	// Get the OAuth token
	token := os.Getenv("CLAUDE_CODE_OAUTH_TOKEN")
	if token == "" {
		return 0, fmt.Errorf("CLAUDE_CODE_OAUTH_TOKEN not set")
	}

	// Generate a container name based on task ID
	containerName := fmt.Sprintf("giverny-%s", taskID)

	// Get home directory for mounting Claude config
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return 0, fmt.Errorf("failed to get home directory: %w", err)
	}

	// Build the docker run command
	args := []string{
		"run",
		"-it",
		"--name", containerName,
		"--env", "CLAUDE_CODE_OAUTH_TOKEN",
		"-v", fmt.Sprintf("%s/.claude:/root/.claude", homeDir),
		"-v", fmt.Sprintf("%s/.claude.json:/root/.claude.json", homeDir),
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

	// Add debug flag if enabled
	if debug {
		args = append(args, "--debug")
	}

	// Add agent args if provided
	if agentArgs != "" {
		args = append(args, fmt.Sprintf("--agent-args=%s", agentArgs))
	}

	args = append(args, taskID, prompt)

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
	cmd := exec.Command("docker", "rm", containerName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to remove container %s: %w", containerName, err)
	}
	fmt.Printf("✓ Container removed\n")
	return nil
}
