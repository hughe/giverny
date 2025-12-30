package git

import (
	"fmt"
	"os"
	"os/exec"
)

// SetupWorkspace creates /app, checks out the branch, and creates a START label
func SetupWorkspace(branchName string) error {
	// Create /app directory
	if err := os.MkdirAll("/app", 0755); err != nil {
		return fmt.Errorf("failed to create /app directory: %w", err)
	}

	// Checkout the branch to /app using git worktree
	cmd := exec.Command("git", "-C", "/git", "worktree", "add", "/app", branchName)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to checkout branch %s to /app: %w", branchName, err)
	}
	fmt.Printf("Checked out branch %s to /app\n", branchName)

	// Create giverny/START label branch to mark where we started
	startLabel := branchName + "/START"
	cmd = exec.Command("git", "-C", "/app", "branch", startLabel)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create START label branch %s: %w", startLabel, err)
	}
	fmt.Printf("Created START label: %s\n", startLabel)

	return nil
}

// IsWorkspaceDirty checks if there are uncommitted changes in /app
func IsWorkspaceDirty() (bool, error) {
	cmd := exec.Command("git", "status", "--porcelain")
	// Use /app if it exists, otherwise use current directory (for testing)
	if _, err := os.Stat("/app"); err == nil {
		cmd.Dir = "/app"
	}
	output, err := cmd.Output()
	if err != nil {
		return false, err
	}
	return len(output) > 0, nil
}

// PushBranch pushes the branch to the git server
func PushBranch(branchName string, gitServerPort int) error {
	fmt.Printf("Pushing %s to git server...\n", branchName)

	// Construct the git server URL
	gitServerURL := fmt.Sprintf("git://host.docker.internal:%d/git", gitServerPort)

	// Push the branch
	cmd := exec.Command("git", "push", gitServerURL, branchName)
	cmd.Dir = "/app"
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git push failed: %w", err)
	}

	fmt.Printf("✓ Successfully pushed %s\n", branchName)
	return nil
}
