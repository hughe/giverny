package git

import (
	"fmt"
	"os/exec"
	"strings"
)

// CreateBranch creates a new git branch at the current HEAD without checking it out.
// Returns an error if the branch already exists or if git command fails.
func CreateBranch(branchName string) error {
	// Create the branch without checking it out
	cmd := exec.Command("git", "branch", branchName)
	output, err := cmd.CombinedOutput()

	if err != nil {
		// Check if branch already exists
		if strings.Contains(string(output), "already exists") {
			return fmt.Errorf("branch '%s' already exists", branchName)
		}
		return fmt.Errorf("failed to create branch '%s': %s", branchName, strings.TrimSpace(string(output)))
	}

	return nil
}

// BranchExists checks if a git branch exists.
// Returns true if the branch exists, false otherwise.
func BranchExists(branchName string) (bool, error) {
	cmd := exec.Command("git", "rev-parse", "--verify", branchName)
	err := cmd.Run()

	if err != nil {
		// If exit status is not 0, the branch does not exist
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 128 {
			return false, nil
		}
		return false, fmt.Errorf("failed to check if branch '%s' exists: %w", branchName, err)
	}

	return true, nil
}

// GetBranchCommitRange returns the first and last commit hashes for a branch.
// Returns empty strings if the branch has no commits beyond its divergence point.
//
// The function tries multiple strategies to find the commit range:
// 1. If a START label exists (branchName-START), use commits after that label
// 2. Otherwise, find where the branch diverged from 'main' using merge-base
//
// This always compares against 'main' regardless of upstream tracking settings,
// ensuring cherry-pick instructions are relative to the main branch.
func GetBranchCommitRange(branchName string) (firstCommit, lastCommit string, err error) {
	// Get the last commit (HEAD of the branch)
	cmd := exec.Command("git", "rev-parse", branchName)
	output, err := cmd.Output()
	if err != nil {
		return "", "", fmt.Errorf("failed to get last commit for branch '%s': %w", branchName, err)
	}
	lastCommit = strings.TrimSpace(string(output))

	// Strategy 1: Check if START label exists (used inside containers)
	startLabel := branchName + "-START"
	cmd = exec.Command("git", "rev-parse", "--verify", startLabel)
	output, err = cmd.Output()
	if err == nil {
		// START label exists, get the first commit after it
		startCommit := strings.TrimSpace(string(output))

		// Check if there are any commits between START and the branch HEAD
		cmd = exec.Command("git", "rev-list", "--reverse", startCommit+".."+branchName)
		output, err = cmd.Output()
		if err != nil {
			return "", "", fmt.Errorf("failed to get commits after START label: %w", err)
		}

		commits := strings.TrimSpace(string(output))
		if commits == "" {
			// No commits after START label
			return "", "", nil
		}

		// First line is the first commit
		lines := strings.Split(commits, "\n")
		firstCommit = lines[0]

		return firstCommit, lastCommit, nil
	}

	// Strategy 2: Find divergence point using merge-base with parent branch
	// Always use 'main' as the parent branch for cherry-pick instructions.
	// This ensures users get instructions to cherry-pick commits from the task
	// branch into their main branch, regardless of upstream tracking settings.
	parentBranch := "main"

	// Find the merge-base (common ancestor) between the branch and its parent
	cmd = exec.Command("git", "merge-base", parentBranch, branchName)
	output, err = cmd.Output()
	if err != nil {
		// If merge-base fails, the branches may not share history
		// Fall back to returning empty (no commits to cherry-pick)
		return "", "", nil
	}

	mergeBase := strings.TrimSpace(string(output))

	// Check if the branch has diverged at all
	if mergeBase == lastCommit {
		// No commits beyond the merge-base
		return "", "", nil
	}

	// Get all commits from merge-base to branch HEAD
	cmd = exec.Command("git", "rev-list", "--reverse", mergeBase+".."+branchName)
	output, err = cmd.Output()
	if err != nil {
		return "", "", fmt.Errorf("failed to get commits after merge-base: %w", err)
	}

	commits := strings.TrimSpace(string(output))
	if commits == "" {
		// No commits after merge-base
		return "", "", nil
	}

	// First line is the first commit
	lines := strings.Split(commits, "\n")
	firstCommit = lines[0]

	return firstCommit, lastCommit, nil
}

// GetShortHash converts a full git commit hash to its short form.
// Returns the short hash (typically 7 characters) or the original hash if conversion fails.
func GetShortHash(fullHash string) string {
	cmd := exec.Command("git", "rev-parse", "--short", fullHash)
	output, err := cmd.Output()
	if err != nil {
		// If we can't get the short hash, return the full hash
		return fullHash
	}
	return strings.TrimSpace(string(output))
}
