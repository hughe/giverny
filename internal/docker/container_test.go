package docker

import (
	"os"
	"testing"
)

func TestRunContainer_RequiresToken(t *testing.T) {
	// Save and clear the token
	originalToken := os.Getenv("CLAUDE_CODE_OAUTH_TOKEN")
	os.Unsetenv("CLAUDE_CODE_OAUTH_TOKEN")
	defer func() {
		if originalToken != "" {
			os.Setenv("CLAUDE_CODE_OAUTH_TOKEN", originalToken)
		}
	}()

	// Should fail without token
	_, err := RunContainer("test-task", "test prompt", 9999, "")
	if err == nil {
		t.Error("expected error when CLAUDE_CODE_OAUTH_TOKEN is not set")
	}
	if err != nil && err.Error() != "CLAUDE_CODE_OAUTH_TOKEN not set" {
		t.Errorf("unexpected error message: %v", err)
	}
}
