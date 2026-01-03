package git

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// CloneRepo clones a repository from the git server into /git directory.
// Uses --no-checkout to create a bare-like clone that can be checked out later.
// Returns an error if the clone fails.
func CloneRepo(gitServerPort int, debug bool) error {
	return CloneRepoToDir(gitServerPort, "/git", debug)
}

// CloneRepoToDir clones a repository from the git server into the specified directory.
// Uses --no-checkout to create a bare-like clone that can be checked out later.
// Returns an error if the clone fails.
func CloneRepoToDir(gitServerPort int, gitDir string, debug bool) error {
	return CloneRepoFromHost(gitServerPort, gitDir, "host.docker.internal", debug)
}

// CloneRepoFromHost clones a repository from the specified host and port into the specified directory.
// Uses --no-checkout to create a bare-like clone that can be checked out later.
// Returns an error if the clone fails.
func CloneRepoFromHost(gitServerPort int, gitDir string, host string, debug bool) error {
	// Create directory
	if err := os.MkdirAll(gitDir, 0755); err != nil {
		return fmt.Errorf("failed to create %s directory: %w", gitDir, err)
	}

	// Clone from the specified host
	// Docker provides host.docker.internal as a special DNS name that resolves to the host
	repoURL := fmt.Sprintf("git://%s:%d/", host, gitServerPort)

	// Run git clone with --no-checkout
	args := []string{"clone", "--no-checkout"}
	if !debug {
		args = append(args, "--quiet")
	}
	args = append(args, repoURL, gitDir)

	cmd := exec.Command("git", args...)
	output, err := cmd.CombinedOutput()

	if err != nil {
		// Provide useful error message
		outputStr := strings.TrimSpace(string(output))
		if strings.Contains(outputStr, "Connection refused") {
			return fmt.Errorf("failed to connect to git server at %s\nIs the git server running on the host?\nError: %s", repoURL, outputStr)
		}
		if strings.Contains(outputStr, "does not appear to be a git repository") {
			return fmt.Errorf("git server at %s does not appear to be serving a valid repository\nError: %s", repoURL, outputStr)
		}
		return fmt.Errorf("failed to clone repository from %s: %s", repoURL, outputStr)
	}

	return nil
}
