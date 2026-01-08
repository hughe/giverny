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
	t.Run("reads valid PID from existing file", func(t *testing.T) {
		pidFile, err := os.CreateTemp("", "test-pid-*")
		if err != nil {
			t.Fatalf("failed to create temp file: %v", err)
		}
		defer os.Remove(pidFile.Name())

		// Write a valid PID
		expectedPid := 12345
		fmt.Fprintf(pidFile, "%d\n", expectedPid)
		pidFile.Close()

		pid, err := pollForPidFile(pidFile.Name(), 100*time.Millisecond)
		if err != nil {
			t.Errorf("pollForPidFile() error = %v, want nil", err)
		}
		if pid != expectedPid {
			t.Errorf("pollForPidFile() = %d, want %d", pid, expectedPid)
		}
	})

	t.Run("times out when file never appears", func(t *testing.T) {
		nonExistentFile := "/tmp/nonexistent-pid-file-" + fmt.Sprintf("%d", time.Now().UnixNano())

		start := time.Now()
		pid, err := pollForPidFile(nonExistentFile, 100*time.Millisecond)
		elapsed := time.Since(start)

		if err == nil {
			t.Error("pollForPidFile() error = nil, want timeout error")
		}
		if pid != 0 {
			t.Errorf("pollForPidFile() = %d, want 0", pid)
		}
		if elapsed < 100*time.Millisecond {
			t.Errorf("pollForPidFile() returned too quickly: %v", elapsed)
		}
	})

	t.Run("waits for empty file to be filled", func(t *testing.T) {
		pidFile, err := os.CreateTemp("", "test-pid-*")
		if err != nil {
			t.Fatalf("failed to create temp file: %v", err)
		}
		defer os.Remove(pidFile.Name())
		pidFile.Close()

		expectedPid := 54321

		// Simulate delayed write
		go func() {
			time.Sleep(50 * time.Millisecond)
			os.WriteFile(pidFile.Name(), []byte(fmt.Sprintf("%d\n", expectedPid)), 0644)
		}()

		pid, err := pollForPidFile(pidFile.Name(), 200*time.Millisecond)
		if err != nil {
			t.Errorf("pollForPidFile() error = %v, want nil", err)
		}
		if pid != expectedPid {
			t.Errorf("pollForPidFile() = %d, want %d", pid, expectedPid)
		}
	})

	t.Run("waits for file to appear", func(t *testing.T) {
		pidFilePath := "/tmp/delayed-pid-file-" + fmt.Sprintf("%d", time.Now().UnixNano())
		defer os.Remove(pidFilePath)

		expectedPid := 99999

		// Simulate delayed file creation
		go func() {
			time.Sleep(50 * time.Millisecond)
			os.WriteFile(pidFilePath, []byte(fmt.Sprintf("%d\n", expectedPid)), 0644)
		}()

		pid, err := pollForPidFile(pidFilePath, 200*time.Millisecond)
		if err != nil {
			t.Errorf("pollForPidFile() error = %v, want nil", err)
		}
		if pid != expectedPid {
			t.Errorf("pollForPidFile() = %d, want %d", pid, expectedPid)
		}
	})

	t.Run("handles invalid content then valid content", func(t *testing.T) {
		pidFile, err := os.CreateTemp("", "test-pid-*")
		if err != nil {
			t.Fatalf("failed to create temp file: %v", err)
		}
		defer os.Remove(pidFile.Name())

		// Write invalid content initially
		pidFile.WriteString("invalid\n")
		pidFile.Close()

		expectedPid := 77777

		// Simulate fixing the content
		go func() {
			time.Sleep(50 * time.Millisecond)
			os.WriteFile(pidFile.Name(), []byte(fmt.Sprintf("%d\n", expectedPid)), 0644)
		}()

		pid, err := pollForPidFile(pidFile.Name(), 200*time.Millisecond)
		if err != nil {
			t.Errorf("pollForPidFile() error = %v, want nil", err)
		}
		if pid != expectedPid {
			t.Errorf("pollForPidFile() = %d, want %d", pid, expectedPid)
		}
	})

	t.Run("respects timeout with invalid content", func(t *testing.T) {
		pidFile, err := os.CreateTemp("", "test-pid-*")
		if err != nil {
			t.Fatalf("failed to create temp file: %v", err)
		}
		defer os.Remove(pidFile.Name())

		// Write invalid content that never gets fixed
		pidFile.WriteString("always-invalid\n")
		pidFile.Close()

		start := time.Now()
		pid, err := pollForPidFile(pidFile.Name(), 100*time.Millisecond)
		elapsed := time.Since(start)

		if err == nil {
			t.Error("pollForPidFile() error = nil, want timeout error")
		}
		if pid != 0 {
			t.Errorf("pollForPidFile() = %d, want 0", pid)
		}
		if elapsed < 100*time.Millisecond {
			t.Errorf("pollForPidFile() returned too quickly: %v", elapsed)
		}
	})
}
