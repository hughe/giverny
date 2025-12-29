package git

import (
	"os"
	"testing"
)

// TestCloneRepo is skipped in unit tests because it requires:
// 1. Running inside a Docker container (for host.docker.internal to resolve)
// 2. Having a git server running on the host
// 3. Write permissions to /git directory
//
// Full integration testing is done in the end-to-end tests.
func TestCloneRepo(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") == "" {
		t.Skip("Skipping integration test. Set INTEGRATION_TEST=1 to run.")
	}

	// This would only run in the actual container integration tests
	port := 9418 // Default git daemon port
	err := CloneRepo(port)
	if err != nil {
		t.Errorf("CloneRepo failed: %v", err)
	}
}
