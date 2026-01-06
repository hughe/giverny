package git

import (
	"fmt"
	"os"
	"os/exec"

	"giverny/internal/cmdutil"
)

// SetupWorkspace creates /app, checks out the branch, and creates a START label
func SetupWorkspace(branchName string, debug bool) error {
	// Create /app directory
	if err := os.MkdirAll("/app", 0755); err != nil {
		return fmt.Errorf("failed to create /app directory: %w", err)
	}

	// Checkout the branch to /app using git worktree
	if err := cmdutil.RunCommandWithDebug(debug, "git", "-C", "/git", "worktree", "add", "/app", branchName); err != nil {
		return fmt.Errorf("failed to checkout branch %s to /app: %w", branchName, err)
	}
	if debug {
		fmt.Printf("Checked out branch %s to /app\n", branchName)
	}

	// Configure git user for commits
	if err := cmdutil.RunCommand("git", "-C", "/app", "config", "user.email", "noreply@anthropic.com"); err != nil {
		return fmt.Errorf("failed to set git user.email: %w", err)
	}

	if err := cmdutil.RunCommand("git", "-C", "/app", "config", "user.name", "Claude Code"); err != nil {
		return fmt.Errorf("failed to set git user.name: %w", err)
	}

	// Create START label branch to mark where we started
	startLabel := branchName + "-START"
	if err := cmdutil.RunCommand("git", "-C", "/app", "branch", startLabel); err != nil {
		return fmt.Errorf("failed to create START label branch %s: %w", startLabel, err)
	}
	if debug {
		fmt.Printf("Created START label: %s\n", startLabel)
	}

	return nil
}

// IsWorkspaceDirty checks if there are uncommitted changes in the current git repository
func IsWorkspaceDirty() (bool, error) {
	cmd := exec.Command("git", "status", "--porcelain")
	output, err := cmd.Output()
	if err != nil {
		return false, err
	}
	return len(output) > 0, nil
}

// PushBranch pushes the branch to the git server
func PushBranch(branchName string, gitServerPort int, debug bool) error {
	fmt.Printf("Pushing %s to git server...\n", branchName)

	// Construct the git server URL
	// When git daemon serves with --base-path pointing to a repo,
	// we reference it with / (empty path after host:port)
	gitServerURL := fmt.Sprintf("git://host.docker.internal:%d/", gitServerPort)

	// Push the branch
	if err := cmdutil.RunCommandInDirWithDebug("/app", debug, "git", "push", gitServerURL, branchName); err != nil {
		return fmt.Errorf("git push failed: %w", err)
	}

	fmt.Printf("✓ Successfully pushed %s\n", branchName)
	return nil
}
