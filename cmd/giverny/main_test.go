package main

import (
	"flag"
	"os"
	"os/exec"
	"testing"
)

func TestRunOutie_ValidatesClaudeToken(t *testing.T) {
	// Clean up test branch after all tests complete
	defer func() {
		cmd := exec.Command("git", "branch", "-D", "giverny/test-task")
		cmd.Run() // Ignore errors - branch may not exist
	}()

	tests := []struct {
		name        string
		tokenValue  string
		shouldError bool
	}{
		{
			name:        "token is set",
			tokenValue:  "test-token-123",
			shouldError: false,
		},
		{
			name:        "token is empty",
			tokenValue:  "",
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original token value
			originalToken := os.Getenv("CLAUDE_CODE_OAUTH_TOKEN")
			defer os.Setenv("CLAUDE_CODE_OAUTH_TOKEN", originalToken)

			// Set test token value
			if tt.tokenValue != "" {
				os.Setenv("CLAUDE_CODE_OAUTH_TOKEN", tt.tokenValue)
			} else {
				os.Unsetenv("CLAUDE_CODE_OAUTH_TOKEN")
			}

			// Test runOutie
			config := Config{
				TaskID:    "test-task",
				Prompt:    "test prompt",
				BaseImage: "debian:stable",
			}

			err := runOutie(config)

			if tt.shouldError && err == nil {
				t.Error("expected error but got nil")
			}

			if !tt.shouldError && err != nil {
				t.Errorf("expected no error but got: %v", err)
			}
		})
	}
}

func TestParseArgs_DefaultPrompt(t *testing.T) {
	flags := flag.NewFlagSet("test", flag.ContinueOnError)
	args := []string{"task-123"}

	config := parseArgs(flags, args)

	if config.TaskID != "task-123" {
		t.Errorf("expected TaskID 'task-123', got '%s'", config.TaskID)
	}

	expectedPrompt := "Please work on task-123."
	if config.Prompt != expectedPrompt {
		t.Errorf("expected Prompt '%s', got '%s'", expectedPrompt, config.Prompt)
	}

	if config.BaseImage != "giverny:latest" {
		t.Errorf("expected default BaseImage 'giverny:latest', got '%s'", config.BaseImage)
	}
}

func TestParseArgs_CustomPrompt(t *testing.T) {
	flags := flag.NewFlagSet("test", flag.ContinueOnError)
	args := []string{"task-456", "Custom prompt here"}

	config := parseArgs(flags, args)

	if config.TaskID != "task-456" {
		t.Errorf("expected TaskID 'task-456', got '%s'", config.TaskID)
	}

	if config.Prompt != "Custom prompt here" {
		t.Errorf("expected Prompt 'Custom prompt here', got '%s'", config.Prompt)
	}
}

func TestParseArgs_WithFlags(t *testing.T) {
	flags := flag.NewFlagSet("test", flag.ContinueOnError)
	args := []string{
		"--base-image", "ubuntu:22.04",
		"--docker-args", "-v /tmp:/tmp",
		"task-789",
	}

	config := parseArgs(flags, args)

	if config.TaskID != "task-789" {
		t.Errorf("expected TaskID 'task-789', got '%s'", config.TaskID)
	}

	if config.BaseImage != "ubuntu:22.04" {
		t.Errorf("expected BaseImage 'ubuntu:22.04', got '%s'", config.BaseImage)
	}

	if config.DockerArgs != "-v /tmp:/tmp" {
		t.Errorf("expected DockerArgs '-v /tmp:/tmp', got '%s'", config.DockerArgs)
	}
}

func TestParseArgs_InnieMode(t *testing.T) {
	flags := flag.NewFlagSet("test", flag.ContinueOnError)
	args := []string{
		"--innie",
		"--git-server-port", "3000",
		"task-001",
	}

	config := parseArgs(flags, args)

	if !config.IsInnie {
		t.Error("expected IsInnie to be true")
	}

	if config.GitServerPort != 3000 {
		t.Errorf("expected GitServerPort 3000, got %d", config.GitServerPort)
	}
}
