package testutil

import (
	"os"
	"path/filepath"
	"testing"

	"giverny/internal/cmdutil"
)

// initTestRepo initializes a git repository in the given directory with an initial commit.
// It configures the repo with test user credentials and creates a test.txt file.
// If content is empty, it defaults to "test".
func InitTestRepo(t *testing.T, dir string, content ...string) {
	t.Helper()

	// Initialize git repo with 'main' as the default branch
	if err := cmdutil.RunCommandInDir(dir, "git", "init", "--initial-branch=main"); err != nil {
		t.Fatalf("failed to init git repo: %v", err)
	}

	// Configure git repo with proper error checking
	if err := cmdutil.RunCommand("git", "-C", dir, "config", "user.email", "test@example.com"); err != nil {
		t.Fatalf("failed to set user.email: %v", err)
	}
	if err := cmdutil.RunCommand("git", "-C", dir, "config", "user.name", "Test User"); err != nil {
		t.Fatalf("failed to set user.name: %v", err)
	}

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
	if err := cmdutil.RunCommand("git", "-C", dir, "add", "."); err != nil {
		t.Fatalf("failed to add files: %v", err)
	}
	if err := cmdutil.RunCommand("git", "-C", dir, "commit", "-m", "initial commit"); err != nil {
		t.Fatalf("failed to create initial commit: %v", err)
	}
}
