package main

import (
	"os"
	"path/filepath"
	"testing"

	"giverny/internal/cmdutil"
)

// initTestRepo initializes a git repository in the given directory with an initial commit.
// It configures the repo with test user credentials and creates a test.txt file.
func initTestRepo(t *testing.T, dir string) {
	t.Helper()

	// Initialize a git repository
	if err := cmdutil.RunCommandInDir(dir, "git", "init"); err != nil {
		t.Fatalf("failed to init git repo: %v", err)
	}

	// Configure git
	cmdutil.RunCommand("git", "-C", dir, "config", "init.defaultBranch", "main")
	cmdutil.RunCommand("git", "-C", dir, "config", "user.email", "test@example.com")
	cmdutil.RunCommand("git", "-C", dir, "config", "user.name", "Test User")

	// Create an initial commit
	testFile := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	cmdutil.RunCommand("git", "-C", dir, "add", ".")
	cmdutil.RunCommand("git", "-C", dir, "commit", "-m", "initial commit")
}
