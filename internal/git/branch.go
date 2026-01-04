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

// GetBranchCommitRange returns the first and last commit hashes for a branch.
// Returns empty strings if the branch has no commits beyond its START label.
// The START label is expected to be named "branchName-START" and marks the beginning of work.
func GetBranchCommitRange(branchName string) (firstCommit, lastCommit string, err error) {
	// Get the last commit (HEAD of the branch)
	cmd := exec.Command("git", "rev-parse", branchName)
	output, err := cmd.Output()
	if err != nil {
		return "", "", fmt.Errorf("failed to get last commit for branch '%s': %w", branchName, err)
	}
	lastCommit = strings.TrimSpace(string(output))

	// Check if START label exists
	startLabel := branchName + "-START"
	cmd = exec.Command("git", "rev-parse", "--verify", startLabel)
	output, err = cmd.Output()
	if err != nil {
		// START label doesn't exist, fall back to finding commits unique to this branch
		cmd = exec.Command("git", "log", "--reverse", "--format=%H", branchName, "--not", "--all", "--not", branchName)
		output, err = cmd.Output()
		if err != nil {
			return "", "", fmt.Errorf("failed to get first commit for branch '%s': %w", branchName, err)
		}

		commits := strings.TrimSpace(string(output))
		if commits == "" {
			// No commits unique to this branch
			return "", "", nil
		}

		// First line is the first commit
		lines := strings.Split(commits, "\n")
		firstCommit = lines[0]
		return firstCommit, lastCommit, nil
	}

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
