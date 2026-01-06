package outie

import (
	"os"
	"path/filepath"
	"testing"

	"giverny/internal/cmdutil"
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

// initTestRepo initializes a git repository in the given directory with an initial commit.
// It configures the repo with test user credentials and creates a test.txt file.
func initTestRepo(t *testing.T, dir string) {
	t.Helper()

	// Initialize git repo with 'main' as the default branch
	if err := cmdutil.RunCommandInDir(dir, "git", "init", "--initial-branch=main"); err != nil {
		t.Fatalf("failed to init git repo: %v", err)
	}

	// Configure git repo
	cmdutil.RunCommand("git", "-C", dir, "config", "user.email", "test@example.com")
	cmdutil.RunCommand("git", "-C", dir, "config", "user.name", "Test User")

	// Create initial commit
	testFile := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	if err := cmdutil.RunCommandInDir(dir, "git", "add", "test.txt"); err != nil {
		t.Fatalf("failed to git add: %v", err)
	}

	if err := cmdutil.RunCommandInDir(dir, "git", "commit", "-m", "Initial commit"); err != nil {
		t.Fatalf("failed to git commit: %v", err)
	}
}

func TestDirtyWorkspaceCheck(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "giverny-outie-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Initialize a git repository
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

	// Set required environment variable
	os.Setenv("CLAUDE_CODE_OAUTH_TOKEN", "test-token")
	defer os.Unsetenv("CLAUDE_CODE_OAUTH_TOKEN")

	t.Run("rejects dirty workspace by default", func(t *testing.T) {
		// Make a change without committing
		testFile := filepath.Join(tmpDir, "test.txt")
		if err := os.WriteFile(testFile, []byte("modified"), 0644); err != nil {
			t.Fatalf("failed to modify test file: %v", err)
		}

		// Create config without AllowDirty flag
		config := Config{
			TaskID:     "test-dirty",
			Prompt:     "test prompt",
			AllowDirty: false,
		}

		// Run should fail due to dirty workspace
		err := Run(config)
		if err == nil {
			t.Error("expected error for dirty workspace, got nil")
		}

		// Check error message mentions uncommitted changes
		if err != nil && err.Error() != "working directory has uncommitted changes. Commit or stash them first, or use --allow-dirty flag" {
			t.Errorf("unexpected error message: %v", err)
		}

		// Clean up the change
		if err := cmdutil.RunCommand("git", "checkout", "test.txt"); err != nil {
			t.Fatalf("failed to checkout test file: %v", err)
		}
	})

	t.Run("allows dirty workspace with --allow-dirty flag", func(t *testing.T) {
		// Make a change without committing
		testFile := filepath.Join(tmpDir, "test.txt")
		if err := os.WriteFile(testFile, []byte("modified"), 0644); err != nil {
			t.Fatalf("failed to modify test file: %v", err)
		}

		// Create config with AllowDirty flag
		config := Config{
			TaskID:     "test-allow-dirty",
			Prompt:     "test prompt",
			AllowDirty: true,
		}

		// Run should fail for a different reason (git server, docker, etc.)
		// but not because of dirty workspace
		err := Run(config)

		// We expect an error, but it should NOT be about uncommitted changes
		if err != nil && err.Error() == "working directory has uncommitted changes. Commit or stash them first, or use --allow-dirty flag" {
			t.Error("--allow-dirty flag did not bypass dirty workspace check")
		}

		// Clean up the change
		if err := cmdutil.RunCommand("git", "checkout", "test.txt"); err != nil {
			t.Fatalf("failed to checkout test file: %v", err)
		}
	})

	t.Run("skips dirty check with --existing-branch flag", func(t *testing.T) {
		// First create a branch
		branchName := "giverny/test-existing"
		if err := cmdutil.RunCommand("git", "branch", branchName); err != nil {
			t.Fatalf("failed to create branch: %v", err)
		}

		// Make a change without committing
		testFile := filepath.Join(tmpDir, "test.txt")
		if err := os.WriteFile(testFile, []byte("modified"), 0644); err != nil {
			t.Fatalf("failed to modify test file: %v", err)
		}

		// Create config with ExistingBranch flag
		config := Config{
			TaskID:         "test-existing",
			Prompt:         "test prompt",
			ExistingBranch: true,
		}

		// Run should fail for a different reason (git server, docker, etc.)
		// but not because of dirty workspace
		err := Run(config)

		// We expect an error, but it should NOT be about uncommitted changes
		if err != nil && err.Error() == "working directory has uncommitted changes. Commit or stash them first, or use --allow-dirty flag" {
			t.Error("--existing-branch flag did not bypass dirty workspace check")
		}

		// Clean up the change
		if err := cmdutil.RunCommand("git", "checkout", "test.txt"); err != nil {
			t.Fatalf("failed to checkout test file: %v", err)
		}
	})

	t.Run("allows clean workspace", func(t *testing.T) {
		// Ensure workspace is clean
		if err := cmdutil.RunCommand("git", "checkout", "."); err != nil {
			t.Fatalf("failed to clean workspace: %v", err)
		}

		// Create config without AllowDirty flag
		config := Config{
			TaskID: "test-clean",
			Prompt: "test prompt",
		}

		// Run should fail for a different reason (git server, docker, etc.)
		// but not because of dirty workspace
		err := Run(config)

		// We expect an error, but it should NOT be about uncommitted changes
		if err != nil && err.Error() == "working directory has uncommitted changes. Commit or stash them first, or use --allow-dirty flag" {
			t.Error("clean workspace was rejected as dirty")
		}
	})
}
