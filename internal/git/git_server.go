package git

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

const (
	minPort        = 2001
	maxPort        = 9999
	maxRetries     = 10
	startupTimeout = 2 * time.Second
)

var readyPattern = regexp.MustCompile(`Ready to rumble`)

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

	// Capture stderr to monitor for "Ready to rumble" message
	//
	// TODO: Rather than capturing stderr, and searching for "Ready to
	// rumble" could we poll for the existance of the pid file that
	// has length > 0?
	//
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the server
	if err := cmd.Start(); err != nil {
		// Check if it's a port conflict
		if strings.Contains(err.Error(), "address already in use") {
			return nil, fmt.Errorf("port %d already in use", port)
		}
		return nil, fmt.Errorf("failed to start git server on port %d: %w", port, err)
	}

	// Channel to signal when server is ready
	ready := make(chan bool, 1)
	errChan := make(chan error, 1)

	// Read stderr in a goroutine to detect when server is ready
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			if readyPattern.MatchString(line) {
				ready <- true
				return
			}
		}
		if err := scanner.Err(); err != nil {
			errChan <- fmt.Errorf("error reading stderr: %w", err)
		}
	}()

	// Wait for ready message or timeout
	select {
	case <-ready:
		// Read the PID from the PID file
		pidData, err := os.ReadFile(pidFilePath)
		if err != nil {
			cmd.Process.Kill()
			cmd.Wait()
			return nil, fmt.Errorf("failed to read PID file: %w", err)
		}
		var actualPid int
		if _, err := fmt.Sscanf(string(pidData), "%d", &actualPid); err != nil {
			cmd.Process.Kill()
			cmd.Wait()
			return nil, fmt.Errorf("failed to parse PID from file: %w", err)
		}
		return &ServerCmd{Cmd: cmd, ActualPid: actualPid}, nil
	case err := <-errChan:
		cmd.Process.Kill()
		cmd.Wait()
		return nil, err
	case <-time.After(startupTimeout):
		cmd.Process.Kill()
		cmd.Wait()
		return nil, fmt.Errorf("git server startup timeout on port %d", port)
	}
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
