package git

import (
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"

	"giverny/internal/testutil"
)

func TestMain(m *testing.M) {
	// Check if GIV_TEST_ENV_DIR is set and change to that directory
	if testEnvDir := os.Getenv("GIV_TEST_ENV_DIR"); testEnvDir != "" {
		if err := os.Chdir(testEnvDir); err != nil {
			panic("failed to change to test environment directory: " + err.Error())
		}
	}

	m.Run()
}

func TestStartServer(t *testing.T) {
	// Create a temporary git repository for testing
	tmpDir, err := os.MkdirTemp("", "giverny-git-server-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Initialize git repo
	testutil.InitTestRepo(t, tmpDir)

	t.Run("starts server successfully", func(t *testing.T) {
		serverCmd, port, err := StartServer(tmpDir)
		if err != nil {
			t.Fatalf("failed to start server: %v", err)
		}
		t.Cleanup(func() {
			if err := StopServer(serverCmd); err != nil {
				t.Errorf("failed to stop server: %v", err)
			}
		})

		// Verify port is in valid range
		if port < minPort || port > maxPort {
			t.Errorf("port %d is outside valid range %d-%d", port, minPort, maxPort)
		}

		// Verify actual process is running
		if serverCmd.ActualPid <= 0 {
			t.Error("server actual PID is invalid")
		}

		// Give it a moment to ensure it stays running
		time.Sleep(200 * time.Millisecond)

		// Check if process is still alive using ps command
		cmd := exec.Command("ps", "-p", fmt.Sprintf("%d", serverCmd.ActualPid))
		if err := cmd.Run(); err != nil {
			t.Errorf("server process is not running (pid %d)", serverCmd.ActualPid)
		}
	})

	t.Run("stops server successfully", func(t *testing.T) {
		serverCmd, _, err := StartServer(tmpDir)
		if err != nil {
			t.Fatalf("failed to start server: %v", err)
		}

		actualPid := serverCmd.ActualPid
		err = StopServer(serverCmd)
		if err != nil {
			t.Errorf("failed to stop server: %v", err)
		}

		// Give it a moment to shut down
		time.Sleep(100 * time.Millisecond)

		// Verify process is stopped using ps command
		cmd := exec.Command("ps", "-p", fmt.Sprintf("%d", actualPid))
		if err := cmd.Run(); err == nil {
			t.Error("server process is still running after stop")
		}
	})

	t.Run("stopping nil server is safe", func(t *testing.T) {
		err := StopServer(nil)
		if err != nil {
			t.Errorf("StopServer(nil) returned error: %v", err)
		}
	})
}

func TestRandomPort(t *testing.T) {
	// Test that randomPort generates valid ports
	for i := 0; i < 100; i++ {
		port := randomPort()
		if port < minPort || port > maxPort {
			t.Errorf("randomPort() = %d, want value between %d and %d", port, minPort, maxPort)
		}
	}
}

func TestPollForPidFile(t *testing.T) {
	t.Run("reads valid PID immediately", func(t *testing.T) {
		expectedPid := 12345
		mockReader := func(path string) ([]byte, error) {
			return []byte(fmt.Sprintf("%d\n", expectedPid)), nil
		}

		pid, err := pollForPidFile("test.pid", 100*time.Millisecond, mockReader)
		if err != nil {
			t.Errorf("pollForPidFile() error = %v, want nil", err)
		}
		if pid != expectedPid {
			t.Errorf("pollForPidFile() = %d, want %d", pid, expectedPid)
		}
	})

	t.Run("times out when file never appears", func(t *testing.T) {
		callCount := 0
		mockReader := func(path string) ([]byte, error) {
			callCount++
			return nil, os.ErrNotExist
		}

		start := time.Now()
		pid, err := pollForPidFile("test.pid", 50*time.Millisecond, mockReader)
		elapsed := time.Since(start)

		if err == nil {
			t.Error("pollForPidFile() error = nil, want timeout error")
		}
		if pid != 0 {
			t.Errorf("pollForPidFile() = %d, want 0", pid)
		}
		if elapsed < 50*time.Millisecond {
			t.Errorf("pollForPidFile() returned too quickly: %v", elapsed)
		}
		if callCount < 2 {
			t.Errorf("mockReader called %d times, want at least 2", callCount)
		}
	})

	t.Run("waits for empty file to be filled", func(t *testing.T) {
		callCount := 0
		expectedPid := 54321
		mockReader := func(path string) ([]byte, error) {
			callCount++
			if callCount <= 3 {
				// First few calls return empty
				return []byte{}, nil
			}
			// Later calls return valid PID
			return []byte(fmt.Sprintf("%d\n", expectedPid)), nil
		}

		pid, err := pollForPidFile("test.pid", 200*time.Millisecond, mockReader)
		if err != nil {
			t.Errorf("pollForPidFile() error = %v, want nil", err)
		}
		if pid != expectedPid {
			t.Errorf("pollForPidFile() = %d, want %d", pid, expectedPid)
		}
		if callCount <= 3 {
			t.Errorf("expected multiple polls, got %d", callCount)
		}
	})

	t.Run("waits for file to appear", func(t *testing.T) {
		callCount := 0
		expectedPid := 99999
		mockReader := func(path string) ([]byte, error) {
			callCount++
			if callCount <= 3 {
				// First few calls file doesn't exist
				return nil, os.ErrNotExist
			}
			// Later calls return valid PID
			return []byte(fmt.Sprintf("%d\n", expectedPid)), nil
		}

		pid, err := pollForPidFile("test.pid", 200*time.Millisecond, mockReader)
		if err != nil {
			t.Errorf("pollForPidFile() error = %v, want nil", err)
		}
		if pid != expectedPid {
			t.Errorf("pollForPidFile() = %d, want %d", pid, expectedPid)
		}
		if callCount <= 3 {
			t.Errorf("expected multiple polls, got %d", callCount)
		}
	})

	t.Run("handles invalid content then valid content", func(t *testing.T) {
		callCount := 0
		expectedPid := 77777
		mockReader := func(path string) ([]byte, error) {
			callCount++
			if callCount <= 3 {
				// First few calls return invalid content
				return []byte("invalid\n"), nil
			}
			// Later calls return valid PID
			return []byte(fmt.Sprintf("%d\n", expectedPid)), nil
		}

		pid, err := pollForPidFile("test.pid", 200*time.Millisecond, mockReader)
		if err != nil {
			t.Errorf("pollForPidFile() error = %v, want nil", err)
		}
		if pid != expectedPid {
			t.Errorf("pollForPidFile() = %d, want %d", pid, expectedPid)
		}
		if callCount <= 3 {
			t.Errorf("expected multiple polls, got %d", callCount)
		}
	})

	t.Run("respects timeout with invalid content", func(t *testing.T) {
		mockReader := func(path string) ([]byte, error) {
			return []byte("always-invalid\n"), nil
		}

		start := time.Now()
		pid, err := pollForPidFile("test.pid", 50*time.Millisecond, mockReader)
		elapsed := time.Since(start)

		if err == nil {
			t.Error("pollForPidFile() error = nil, want timeout error")
		}
		if pid != 0 {
			t.Errorf("pollForPidFile() = %d, want 0", pid)
		}
		if elapsed < 50*time.Millisecond {
			t.Errorf("pollForPidFile() returned too quickly: %v", elapsed)
		}
	})

	t.Run("handles read errors gracefully", func(t *testing.T) {
		callCount := 0
		expectedPid := 11111
		mockReader := func(path string) ([]byte, error) {
			callCount++
			if callCount <= 2 {
				// First few calls return different errors
				if callCount == 1 {
					return nil, os.ErrPermission
				}
				return nil, fmt.Errorf("temporary read error")
			}
			// Later calls succeed
			return []byte(fmt.Sprintf("%d\n", expectedPid)), nil
		}

		pid, err := pollForPidFile("test.pid", 200*time.Millisecond, mockReader)
		if err != nil {
			t.Errorf("pollForPidFile() error = %v, want nil", err)
		}
		if pid != expectedPid {
			t.Errorf("pollForPidFile() = %d, want %d", pid, expectedPid)
		}
	})
}
