package git

import (
	"os"
	"path/filepath"
	"testing"

	"giverny/internal/cmdutil"
)

// initTestRepo initializes a git repository in the given directory with an initial commit.
// It configures the repo with test user credentials and creates a test.txt file.
// If content is empty, it defaults to "test".
func initTestRepo(t *testing.T, dir string, content ...string) {
	t.Helper()

	// Initialize git repo
	if err := cmdutil.RunCommandInDir(dir, "git", "init"); err != nil {
		t.Fatalf("failed to init git repo: %v", err)
	}

	// Configure git repo
	cmdutil.RunCommand("git", "-C", dir, "config", "init.defaultBranch", "main")
	cmdutil.RunCommand("git", "-C", dir, "config", "user.email", "test@example.com")
	cmdutil.RunCommand("git", "-C", dir, "config", "user.name", "Test User")

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
	cmdutil.RunCommand("git", "-C", dir, "add", ".")
	if err := cmdutil.RunCommand("git", "-C", dir, "commit", "-m", "initial commit"); err != nil {
		t.Fatalf("failed to create initial commit: %v", err)
	}
}
