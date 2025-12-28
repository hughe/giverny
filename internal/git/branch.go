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
