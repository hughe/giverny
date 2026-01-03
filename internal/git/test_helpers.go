package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// initTestRepo initializes a git repository in the given directory with an initial commit.
// It configures the repo with test user credentials and creates a test.txt file.
// If content is empty, it defaults to "test".
func initTestRepo(t *testing.T, dir string, content ...string) {
	t.Helper()

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to init git repo: %v", err)
	}

	// Configure git repo
	exec.Command("git", "-C", dir, "config", "init.defaultBranch", "main").Run()
	exec.Command("git", "-C", dir, "config", "user.email", "test@example.com").Run()
	exec.Command("git", "-C", dir, "config", "user.name", "Test User").Run()

	// Determine content to write
	fileContent := "test"
	if len(content) > 0 && content[0] != "" {
		fileContent = content[0]
	}

	// Create an initial commit
	testFile := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(testFile, []byte(fileContent), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	exec.Command("git", "-C", dir, "add", ".").Run()
	if err := exec.Command("git", "-C", dir, "commit", "-m", "initial commit").Run(); err != nil {
		t.Fatalf("failed to create initial commit: %v", err)
	}
}
