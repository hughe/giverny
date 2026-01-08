package git

import (
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"strings"
	"time"
)

const (
	minPort        = 2001
	maxPort        = 9999
	maxRetries     = 10
	startupTimeout = 2 * time.Second
	pidPollInterval = 10 * time.Millisecond
)

// StartServer starts a git daemon server on a random port between 2001-9999.
// It enables receive-pack to allow pushing and retries on port conflicts.
// Returns the process command, the port number, and any error.
func StartServer(repoPath string) (*ServerCmd, int, error) {
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

// ServerCmd wraps exec.Cmd and tracks the actual daemon PID
type ServerCmd struct {
	*exec.Cmd
	ActualPid int
}

// tryStartServer attempts to start git daemon on the specified port
func tryStartServer(repoPath string, port int) (*ServerCmd, error) {
	// Create a temporary PID file
	pidFile, err := os.CreateTemp("", "giverny-git-daemon-*.pid")
	if err != nil {
		return nil, fmt.Errorf("failed to create PID file: %w", err)
	}
	pidFilePath := pidFile.Name()
	pidFile.Close()
	defer os.Remove(pidFilePath)

	cmd := exec.Command("git", "daemon",
		"--base-path="+repoPath,
		"--enable=receive-pack",
		"--reuseaddr",
		fmt.Sprintf("--port=%d", port),
		"--export-all",
		"--verbose",
		"--pid-file="+pidFilePath,
	)

	// Start the server
	if err := cmd.Start(); err != nil {
		// Check if it's a port conflict
		if strings.Contains(err.Error(), "address already in use") {
			return nil, fmt.Errorf("port %d already in use", port)
		}
		return nil, fmt.Errorf("failed to start git server on port %d: %w", port, err)
	}

	// Poll for the PID file to be created with valid content
	actualPid, err := pollForPidFile(pidFilePath, startupTimeout, os.ReadFile)
	if err != nil {
		cmd.Process.Kill()
		cmd.Wait()
		return nil, fmt.Errorf("failed to start git server on port %d: %w", port, err)
	}

	return &ServerCmd{Cmd: cmd, ActualPid: actualPid}, nil
}

// fileReader is a function type for reading file contents
type fileReader func(string) ([]byte, error)

// pollForPidFile polls for the PID file to be created and contain a valid PID.
// It polls at regular intervals until the timeout is reached.
// The readFile parameter allows dependency injection for testing.
func pollForPidFile(pidFilePath string, timeout time.Duration, readFile fileReader) (int, error) {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		// Check if file exists and has content
		pidData, err := readFile(pidFilePath)
		if err != nil {
			// File doesn't exist yet or can't be read, keep polling
			time.Sleep(pidPollInterval)
			continue
		}

		// File exists, check if it has content
		if len(pidData) == 0 {
			// File exists but is empty, keep polling
			time.Sleep(pidPollInterval)
			continue
		}

		// Parse the PID
		var actualPid int
		if _, err := fmt.Sscanf(string(pidData), "%d", &actualPid); err != nil {
			// File has content but can't be parsed, keep polling
			// (git daemon might be in the middle of writing)
			time.Sleep(pidPollInterval)
			continue
		}

		// Successfully read a valid PID
		return actualPid, nil
	}

	return 0, fmt.Errorf("timeout waiting for PID file")
}

// StopServer stops a running git server process
func StopServer(serverCmd *ServerCmd) error {
	if serverCmd == nil {
		return nil
	}

	// Kill the actual daemon process (not the wrapper process)
	// Git daemon forks itself, so we need to kill the child process
	if serverCmd.ActualPid > 0 {
		process, err := os.FindProcess(serverCmd.ActualPid)
		if err != nil {
			return fmt.Errorf("failed to find process %d: %w", serverCmd.ActualPid, err)
		}

		if err := process.Kill(); err != nil {
			// Process might have already exited
			if !strings.Contains(err.Error(), "process already finished") &&
				!strings.Contains(err.Error(), "no such process") {
				return fmt.Errorf("failed to kill git server (PID %d): %w", serverCmd.ActualPid, err)
			}
		}
	}

	// Also wait on the wrapper process to prevent zombies
	if serverCmd.Cmd != nil {
		_ = serverCmd.Wait()
	}

	return nil
}
