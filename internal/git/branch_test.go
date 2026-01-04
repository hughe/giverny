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

func TestGetBranchCommitRange(t *testing.T) {
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

	t.Run("returns empty when branch has no commits", func(t *testing.T) {
		branchName := "giverny/test-empty"
		if err := CreateBranch(branchName); err != nil {
			t.Fatalf("failed to create branch: %v", err)
		}

		first, last, err := GetBranchCommitRange(branchName)
		if err != nil {
			t.Errorf("expected no error, got: %v", err)
		}
		if first != "" || last != "" {
			t.Errorf("expected empty commits for branch with no new commits, got first=%s, last=%s", first, last)
		}
	})

	t.Run("returns commit range with START label", func(t *testing.T) {
		branchName := "giverny/test-with-commits"
		if err := CreateBranch(branchName); err != nil {
			t.Fatalf("failed to create branch: %v", err)
		}

		// Create START label
		startLabel := branchName + "-START"
		cmd := exec.Command("git", "branch", startLabel)
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to create START label: %v", err)
		}

		// Checkout the branch
		cmd = exec.Command("git", "checkout", branchName)
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to checkout branch: %v", err)
		}

		// Make a commit
		cmd = exec.Command("sh", "-c", "echo 'test1' > test1.txt && git add test1.txt && git commit -m 'First commit'")
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to make first commit: %v", err)
		}

		// Get the first commit hash
		cmd = exec.Command("git", "rev-parse", "HEAD")
		output, err := cmd.Output()
		if err != nil {
			t.Fatalf("failed to get first commit hash: %v", err)
		}
		expectedFirst := strings.TrimSpace(string(output))

		// Make another commit
		cmd = exec.Command("sh", "-c", "echo 'test2' > test2.txt && git add test2.txt && git commit -m 'Second commit'")
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to make second commit: %v", err)
		}

		// Get the second commit hash
		cmd = exec.Command("git", "rev-parse", "HEAD")
		output, err = cmd.Output()
		if err != nil {
			t.Fatalf("failed to get second commit hash: %v", err)
		}
		expectedLast := strings.TrimSpace(string(output))

		// Now test GetBranchCommitRange
		first, last, err := GetBranchCommitRange(branchName)
		if err != nil {
			t.Errorf("expected no error, got: %v", err)
		}
		if first != expectedFirst {
			t.Errorf("expected first commit %s, got %s", expectedFirst, first)
		}
		if last != expectedLast {
			t.Errorf("expected last commit %s, got %s", expectedLast, last)
		}
	})

	t.Run("handles single commit", func(t *testing.T) {
		branchName := "giverny/test-single-commit"
		if err := CreateBranch(branchName); err != nil {
			t.Fatalf("failed to create branch: %v", err)
		}

		// Create START label
		startLabel := branchName + "-START"
		cmd := exec.Command("git", "branch", startLabel)
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to create START label: %v", err)
		}

		// Checkout the branch
		cmd = exec.Command("git", "checkout", branchName)
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to checkout branch: %v", err)
		}

		// Make a single commit
		cmd = exec.Command("sh", "-c", "echo 'single' > single.txt && git add single.txt && git commit -m 'Single commit'")
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to make commit: %v", err)
		}

		// Get the commit hash
		cmd = exec.Command("git", "rev-parse", "HEAD")
		output, err := cmd.Output()
		if err != nil {
			t.Fatalf("failed to get commit hash: %v", err)
		}
		expectedCommit := strings.TrimSpace(string(output))

		// Test GetBranchCommitRange
		first, last, err := GetBranchCommitRange(branchName)
		if err != nil {
			t.Errorf("expected no error, got: %v", err)
		}
		if first != expectedCommit {
			t.Errorf("expected first commit %s, got %s", expectedCommit, first)
		}
		if last != expectedCommit {
			t.Errorf("expected last commit %s, got %s", expectedCommit, last)
		}
	})
}
