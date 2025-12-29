package git

import (
	"fmt"
	"math/rand"
	"os/exec"
	"strings"
	"time"
)

const (
	minPort     = 2001
	maxPort     = 9999
	maxRetries  = 10
	startupWait = 100 * time.Millisecond
)

// StartServer starts a git daemon server on a random port between 2001-9999.
// It enables receive-pack to allow pushing and retries on port conflicts.
// Returns the process command, the port number, and any error.
func StartServer(repoPath string) (*exec.Cmd, int, error) {
	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		port := randomPort()
		cmd, err := tryStartServer(repoPath, port)
		if err == nil {
			return cmd, port, nil
		}
		lastErr = err
	}

	return nil, 0, fmt.Errorf("failed to start git server after %d attempts: %w", maxRetries, lastErr)
}

// randomPort generates a random port number in the valid range
func randomPort() int {
	return minPort + rand.Intn(maxPort-minPort+1)
}

// tryStartServer attempts to start git daemon on the specified port
func tryStartServer(repoPath string, port int) (*exec.Cmd, error) {
	cmd := exec.Command("git", "daemon",
		"--base-path="+repoPath,
		"--enable=receive-pack",
		"--reuseaddr",
		fmt.Sprintf("--port=%d", port),
		"--export-all",
	)

	// Start the server
	if err := cmd.Start(); err != nil {
		// Check if it's a port conflict
		if strings.Contains(err.Error(), "address already in use") {
			return nil, fmt.Errorf("port %d already in use", port)
		}
		return nil, fmt.Errorf("failed to start git server on port %d: %w", port, err)
	}

	// Give it a moment to initialize and potentially fail on port conflict
	time.Sleep(startupWait)

	// Use a channel to check if process exits early
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	// Check if process exited immediately
	select {
	case <-done:
		return nil, fmt.Errorf("git server exited immediately on port %d", port)
	case <-time.After(10 * time.Millisecond):
		// Process is still running
		return cmd, nil
	}
}

// StopServer stops a running git server process
func StopServer(cmd *exec.Cmd) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}

	if err := cmd.Process.Kill(); err != nil {
		if strings.Contains(err.Error(), "process already finished") {
			return nil
		}
		return fmt.Errorf("failed to kill git server: %w", err)
	}

	// Wait for the process to exit
	cmd.Wait()
	return nil
}
