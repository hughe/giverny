package git

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestCreateBranch(t *testing.T) {
	// Create a temporary git repository for testing
	tmpDir, err := os.MkdirTemp("", "giverny-git-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Initialize git repo
	initTestRepo(t, tmpDir)

	// Change to temp directory for tests
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	defer os.Chdir(origDir)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp dir: %v", err)
	}

	t.Run("creates new branch successfully", func(t *testing.T) {
		err := CreateBranch("giverny/test-task-1")
		if err != nil {
			t.Errorf("expected no error, got: %v", err)
		}

		// Verify branch was created
		cmd := exec.Command("git", "branch", "--list", "giverny/test-task-1")
		output, err := cmd.Output()
		if err != nil {
			t.Fatalf("failed to list branches: %v", err)
		}

		if len(output) == 0 {
			t.Error("branch was not created")
		}
	})

	t.Run("returns error when branch already exists", func(t *testing.T) {
		branchName := "giverny/test-task-2"

		// Create the branch first time
		err := CreateBranch(branchName)
		if err != nil {
			t.Fatalf("first creation failed: %v", err)
		}

		// Try to create it again
		err = CreateBranch(branchName)
		if err == nil {
			t.Error("expected error for duplicate branch, got nil")
		}

		if err != nil && !strings.Contains(err.Error(), "already exists") {
			t.Errorf("expected 'already exists' error, got: %v", err)
		}
	})

	t.Run("does not check out the branch", func(t *testing.T) {
		branchName := "giverny/test-task-3"

		err := CreateBranch(branchName)
		if err != nil {
			t.Fatalf("branch creation failed: %v", err)
		}

		// Check current branch
		cmd := exec.Command("git", "branch", "--show-current")
		output, err := cmd.Output()
		if err != nil {
			t.Fatalf("failed to get current branch: %v", err)
		}

		currentBranch := strings.TrimSpace(string(output))
		if strings.Contains(currentBranch, branchName) {
			t.Errorf("branch was checked out but should not have been")
		}
	})
}
