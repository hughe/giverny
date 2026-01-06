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

func TestBranchExists(t *testing.T) {
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

	t.Run("returns true for existing branch", func(t *testing.T) {
		branchName := "giverny/test-exists"
		if err := CreateBranch(branchName); err != nil {
			t.Fatalf("failed to create branch: %v", err)
		}

		exists, err := BranchExists(branchName)
		if err != nil {
			t.Errorf("expected no error, got: %v", err)
		}
		if !exists {
			t.Error("expected branch to exist, but it doesn't")
		}
	})

	t.Run("returns false for non-existing branch", func(t *testing.T) {
		exists, err := BranchExists("giverny/non-existing-branch")
		if err != nil {
			t.Errorf("expected no error, got: %v", err)
		}
		if exists {
			t.Error("expected branch to not exist, but it does")
		}
	})

	t.Run("returns true for current branch", func(t *testing.T) {
		// Get current branch
		cmd := exec.Command("git", "branch", "--show-current")
		output, err := cmd.Output()
		if err != nil {
			t.Fatalf("failed to get current branch: %v", err)
		}
		currentBranch := strings.TrimSpace(string(output))

		exists, err := BranchExists(currentBranch)
		if err != nil {
			t.Errorf("expected no error, got: %v", err)
		}
		if !exists {
			t.Error("expected current branch to exist, but it doesn't")
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

	t.Run("finds divergence point without START label (outie scenario)", func(t *testing.T) {
		// This test simulates the scenario in outie where:
		// 1. A branch is created from the default branch
		// 2. Commits are made to the branch (inside container)
		// 3. We need to find the commit range without the START label
		//    (which only exists inside the container)

		// Get the current branch name (could be 'main' or 'master')
		cmd := exec.Command("git", "branch", "--show-current")
		output, err := cmd.Output()
		if err != nil {
			t.Fatalf("failed to get current branch: %v", err)
		}
		defaultBranch := strings.TrimSpace(string(output))

		// First, rename the default branch to 'main' for consistency
		cmd = exec.Command("git", "branch", "-m", defaultBranch, "main")
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to rename branch to main: %v", err)
		}

		// Make a commit on main
		cmd = exec.Command("sh", "-c", "echo 'divergence-test-main' > divergence-main.txt && git add divergence-main.txt && git commit -m 'Commit on main'")
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to make commit on main: %v", err)
		}

		// Create a branch from main
		branchName := "giverny/test-without-label"
		if err := CreateBranch(branchName); err != nil {
			t.Fatalf("failed to create branch: %v", err)
		}

		// Checkout the branch
		cmd = exec.Command("git", "checkout", branchName)
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to checkout branch: %v", err)
		}

		// Make commits on the branch (simulating work done in container)
		cmd = exec.Command("sh", "-c", "echo 'divergence-test1' > divergence-test1.txt && git add divergence-test1.txt && git commit -m 'First commit'")
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to make first commit: %v", err)
		}

		// Get the first commit hash
		cmd = exec.Command("git", "rev-parse", "HEAD")
		output, err = cmd.Output()
		if err != nil {
			t.Fatalf("failed to get first commit hash: %v", err)
		}
		expectedFirst := strings.TrimSpace(string(output))

		// Make another commit
		cmd = exec.Command("sh", "-c", "echo 'divergence-test2' > divergence-test2.txt && git add divergence-test2.txt && git commit -m 'Second commit'")
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

		// Go back to main (simulating outie checking the branch)
		cmd = exec.Command("git", "checkout", "main")
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to checkout main: %v", err)
		}

		// Now test GetBranchCommitRange from main (no START label exists)
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

	t.Run("finds divergence point with upstream tracking branch set", func(t *testing.T) {
		// This test ensures that even when a branch has an upstream tracking branch,
		// GetBranchCommitRange still returns the commits relative to 'main', not
		// relative to the upstream. This is important for giverny's cherry-pick
		// instructions which should always be relative to the main branch.

		// Get the current branch name (could be 'main' or 'master')
		cmd := exec.Command("git", "branch", "--show-current")
		output, err := cmd.Output()
		if err != nil {
			t.Fatalf("failed to get current branch: %v", err)
		}
		defaultBranch := strings.TrimSpace(string(output))

		// First, rename the default branch to 'main' for consistency
		if defaultBranch != "main" {
			cmd = exec.Command("git", "branch", "-m", defaultBranch, "main")
			if err := cmd.Run(); err != nil {
				t.Fatalf("failed to rename branch to main: %v", err)
			}
		}

		// Make a commit on main to establish a divergence point
		cmd = exec.Command("sh", "-c", "echo 'upstream-test-main' > upstream-main.txt && git add upstream-main.txt && git commit -m 'Commit on main'")
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to make commit on main: %v", err)
		}

		// Create a branch from main
		branchName := "giverny/test-with-upstream"
		if err := CreateBranch(branchName); err != nil {
			t.Fatalf("failed to create branch: %v", err)
		}

		// Checkout the branch
		cmd = exec.Command("git", "checkout", branchName)
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to checkout branch: %v", err)
		}

		// Make commits on the branch
		cmd = exec.Command("sh", "-c", "echo 'upstream-test1' > upstream-test1.txt && git add upstream-test1.txt && git commit -m 'First commit'")
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to make first commit: %v", err)
		}

		// Get the first commit hash
		cmd = exec.Command("git", "rev-parse", "HEAD")
		output, err = cmd.Output()
		if err != nil {
			t.Fatalf("failed to get first commit hash: %v", err)
		}
		expectedFirst := strings.TrimSpace(string(output))

		// Make another commit
		cmd = exec.Command("sh", "-c", "echo 'upstream-test2' > upstream-test2.txt && git add upstream-test2.txt && git commit -m 'Second commit'")
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

		// Set up a fake upstream tracking branch
		// First, add a fake remote
		cmd = exec.Command("git", "remote", "add", "origin", "fake-url")
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to add remote: %v", err)
		}

		// Create a fake remote branch by creating a ref
		cmd = exec.Command("git", "update-ref", "refs/remotes/origin/"+branchName, expectedLast)
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to create fake remote ref: %v", err)
		}

		// Set the upstream tracking
		cmd = exec.Command("git", "branch", "--set-upstream-to=origin/"+branchName, branchName)
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to set upstream: %v", err)
		}

		// Verify upstream is set
		cmd = exec.Command("git", "rev-parse", "--abbrev-ref", branchName+"@{upstream}")
		output, err = cmd.Output()
		if err != nil {
			t.Fatalf("upstream should be set but got error: %v", err)
		}
		upstream := strings.TrimSpace(string(output))
		if upstream != "origin/"+branchName {
			t.Fatalf("expected upstream to be origin/%s, got %s", branchName, upstream)
		}

		// Now test GetBranchCommitRange - it should return commits relative to main,
		// not relative to the upstream (which would return no commits since they're synced)
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

		// Clean up: go back to main
		cmd = exec.Command("git", "checkout", "main")
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to checkout main: %v", err)
		}
	})
}

func TestGetShortHash(t *testing.T) {
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

	// Get the full hash of HEAD
	cmd := exec.Command("git", "rev-parse", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("failed to get HEAD hash: %v", err)
	}
	fullHash := strings.TrimSpace(string(output))

	// Get the short hash
	shortHash := GetShortHash(fullHash)

	// Verify the short hash is actually shorter
	if len(shortHash) >= len(fullHash) {
		t.Errorf("expected short hash to be shorter than full hash, got %d vs %d chars", len(shortHash), len(fullHash))
	}

	// Verify the short hash is typically 7 characters (can be more if needed for uniqueness)
	if len(shortHash) < 7 {
		t.Errorf("expected short hash to be at least 7 characters, got %d", len(shortHash))
	}

	// Verify the full hash starts with the short hash
	if !strings.HasPrefix(fullHash, shortHash) {
		t.Errorf("expected full hash %s to start with short hash %s", fullHash, shortHash)
	}

	// Test with an invalid hash - should return the original
	invalidHash := "invalid-hash-xyz"
	result := GetShortHash(invalidHash)
	if result != invalidHash {
		t.Errorf("expected GetShortHash to return original hash on error, got %s", result)
	}
}
