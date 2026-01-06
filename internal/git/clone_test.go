package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"giverny/internal/testutil"
)

// TestCloneRepo tests cloning a repository from a git daemon server
func TestCloneRepo(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") == "" {
		t.Skip("Skipping integration test. Set INTEGRATION_TEST=1 to run.")
	}

	// Create a temporary git repository to serve
	sourceRepo, err := os.MkdirTemp("", "giverny-clone-source-*")
	if err != nil {
		t.Fatalf("failed to create source repo dir: %v", err)
	}
	defer os.RemoveAll(sourceRepo)

	// Initialize git repo
	testutil.InitTestRepo(t, sourceRepo, "test content")

	// Start git server on the source repository
	serverCmd, port, err := StartServer(sourceRepo)
	if err != nil {
		t.Fatalf("failed to start git server: %v", err)
	}
	defer StopServer(serverCmd)

	// Create a temporary directory for the clone
	gitDir := t.TempDir()

	// Clone from the local git server using localhost
	err = CloneRepoFromHost(port, gitDir, "localhost", false)
	if err != nil {
		t.Errorf("CloneRepoFromHost failed: %v", err)
	}

	// Verify the clone was successful by checking the .git directory exists
	gitConfigFile := filepath.Join(gitDir, ".git", "config")
	if _, err := os.Stat(gitConfigFile); os.IsNotExist(err) {
		t.Error("cloned repository does not contain .git/config")
	}

	// Since CloneRepoFromHost uses --no-checkout, we need to checkout the files
	// to verify the clone worked correctly
	checkoutCmd := exec.Command("git", "checkout", "HEAD")
	checkoutCmd.Dir = gitDir
	if err := checkoutCmd.Run(); err != nil {
		t.Fatalf("failed to checkout files: %v", err)
	}

	// Verify the test file was checked out
	clonedFile := filepath.Join(gitDir, "test.txt")
	if _, err := os.Stat(clonedFile); os.IsNotExist(err) {
		t.Error("checked out repository does not contain expected test file")
	}

	// Verify file content
	content, err := os.ReadFile(clonedFile)
	if err != nil {
		t.Errorf("failed to read cloned file: %v", err)
	}
	if string(content) != "test content" {
		t.Errorf("cloned file content = %q, want %q", string(content), "test content")
	}
}
