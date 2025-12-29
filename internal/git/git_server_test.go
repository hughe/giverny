package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
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
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to init git repo: %v", err)
	}

	// Configure git user for the test repo
	exec.Command("git", "-C", tmpDir, "config", "user.email", "test@example.com").Run()
	exec.Command("git", "-C", tmpDir, "config", "user.name", "Test User").Run()

	// Create an initial commit
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	exec.Command("git", "-C", tmpDir, "add", ".").Run()
	exec.Command("git", "-C", tmpDir, "commit", "-m", "initial commit").Run()

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
