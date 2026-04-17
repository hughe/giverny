package docker

import (
	"os"
	"testing"
)

func TestRunContainer_RequiresClaudeToken(t *testing.T) {
	// Save and clear the token
	originalToken := os.Getenv("CLAUDE_CODE_OAUTH_TOKEN")
	os.Unsetenv("CLAUDE_CODE_OAUTH_TOKEN")
	defer func() {
		if originalToken != "" {
			os.Setenv("CLAUDE_CODE_OAUTH_TOKEN", originalToken)
		}
	}()

	// Should fail without token (useAmp=false)
	_, err := RunContainer("test-task", "", "test prompt", "alpine:latest", 9999, "", "", false, false)
	if err == nil {
		t.Error("expected error when CLAUDE_CODE_OAUTH_TOKEN is not set")
	}
	if err != nil && err.Error() != "CLAUDE_CODE_OAUTH_TOKEN not set" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestRunContainer_RequiresAmpToken(t *testing.T) {
	// Save and clear the token
	originalToken := os.Getenv("AMP_API_KEY")
	os.Unsetenv("AMP_API_KEY")
	defer func() {
		if originalToken != "" {
			os.Setenv("AMP_API_KEY", originalToken)
		}
	}()

	// Should fail without token (useAmp=true)
	_, err := RunContainer("test-task", "", "test prompt", "alpine:latest", 9999, "", "", false, true)
	if err == nil {
		t.Error("expected error when AMP_API_KEY is not set")
	}
	if err != nil && err.Error() != "AMP_API_KEY not set" {
		t.Errorf("unexpected error message: %v", err)
	}
}
