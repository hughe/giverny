package docker

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// RunContainer starts the giverny-main container with Innie
// Returns the exit code of the container
func RunContainer(taskID, prompt string, gitPort int, dockerArgs string, debug bool) (int, error) {
	// Get the OAuth token
	token := os.Getenv("CLAUDE_CODE_OAUTH_TOKEN")
	if token == "" {
		return 0, fmt.Errorf("CLAUDE_CODE_OAUTH_TOKEN not set")
	}

	// Generate a container name based on task ID
	containerName := fmt.Sprintf("giverny-%s", taskID)

	// Build the docker run command
	args := []string{
		"run",
		"--name", containerName,
		"--env", "CLAUDE_CODE_OAUTH_TOKEN",
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

	args = append(args, taskID, prompt)

	cmd := exec.Command("docker", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	fmt.Printf("Starting container %s for task %s...\n", containerName, taskID)
	fmt.Printf("To start a shell in the container, run:\n")
	fmt.Printf("  docker exec -it %s /bin/sh\n\n", containerName)

	exitCode := 0
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return 0, fmt.Errorf("failed to run container: %w", err)
		}
	}

	// Only remove container if it exited successfully
	if exitCode == 0 {
		fmt.Printf("Container exited successfully, removing...\n")
		rmCmd := exec.Command("docker", "rm", containerName)
		if err := rmCmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to remove container %s: %v\n", containerName, err)
		}
	} else {
		fmt.Printf("Container exited with code %d, leaving container for inspection\n", exitCode)
		fmt.Printf("\nTo restart the container, run:\n")
		fmt.Printf("  docker start -ai %s\n", containerName)
	}

	return exitCode, nil
}
