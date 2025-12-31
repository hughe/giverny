package main

import (
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"giverny/internal/git"
)

func TestMain(m *testing.M) {
	// Check if GIV_TEST_ENV_DIR is set and change to that directory
	if testEnvDir := os.Getenv("GIV_TEST_ENV_DIR"); testEnvDir != "" {
		if err := os.Chdir(testEnvDir); err != nil {
			panic("failed to change to test environment directory: " + err.Error())
		}
	}

	m.Run()
}

func TestRunOutie_ValidatesClaudeToken(t *testing.T) {
	// Skip this test for now - it requires interactive input with the new menu system
	t.Skip("Skipping integration test - requires interactive input")

	// Clean up test resources after all tests complete
	defer func() {
		// Clean up branch
		cmd := exec.Command("git", "branch", "-D", "giverny/test-task")
		cmd.Run() // Ignore errors - branch may not exist

		// Clean up container
		cmd = exec.Command("docker", "rm", "-f", "giverny-test-task")
		cmd.Run() // Ignore errors - container may not exist
	}()

	tests := []struct{
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

func TestIsWorkspaceDirty_CleanWorkspace(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "giverny-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Initialize a git repository
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to init git repo: %v", err)
	}

	// Configure git
	exec.Command("git", "-C", tmpDir, "config", "user.email", "test@example.com").Run()
	exec.Command("git", "-C", tmpDir, "config", "user.name", "Test User").Run()

	// Create an initial commit
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	exec.Command("git", "-C", tmpDir, "add", ".").Run()
	exec.Command("git", "-C", tmpDir, "commit", "-m", "initial commit").Run()

	// Test isWorkspaceDirty by changing to the temp directory
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	os.Chdir(tmpDir)

	dirty, err := git.IsWorkspaceDirty()
	if err != nil {
		t.Errorf("IsWorkspaceDirty failed: %v", err)
	}

	if dirty {
		t.Error("expected workspace to be clean, but it was dirty")
	}
}

func TestValidateTaskID(t *testing.T) {
	tests := []struct {
		name    string
		taskID  string
		wantErr bool
		errMsg  string
	}{
		// Valid task IDs
		{name: "valid simple", taskID: "abc", wantErr: false},
		{name: "valid with numbers", taskID: "task-123", wantErr: false},
		{name: "valid with underscores", taskID: "my_task", wantErr: false},
		{name: "valid with dots", taskID: "task.1.2", wantErr: false},
		{name: "valid mixed", taskID: "giv-4z1", wantErr: false},

		// Invalid - empty
		{name: "empty", taskID: "", wantErr: true, errMsg: "cannot be empty"},

		// Invalid - forward slash
		{name: "contains slash", taskID: "task/123", wantErr: true, errMsg: "forward slash"},

		// Invalid - starts with dot
		{name: "starts with dot", taskID: ".task", wantErr: true, errMsg: "start with a dot"},

		// Invalid - ends with .lock
		{name: "ends with .lock", taskID: "task.lock", wantErr: true, errMsg: "end with .lock"},

		// Invalid - double dots
		{name: "contains double dots", taskID: "task..123", wantErr: true, errMsg: "double dots"},

		// Invalid - @{
		{name: "contains @{", taskID: "task@{123", wantErr: true, errMsg: "@{"},

		// Invalid - special characters
		{name: "contains backslash", taskID: "task\\123", wantErr: true, errMsg: "backslash"},
		{name: "contains space", taskID: "task 123", wantErr: true, errMsg: "space"},
		{name: "contains tilde", taskID: "task~123", wantErr: true, errMsg: "~"},
		{name: "contains caret", taskID: "task^123", wantErr: true, errMsg: "^"},
		{name: "contains colon", taskID: "task:123", wantErr: true, errMsg: ":"},
		{name: "contains question mark", taskID: "task?123", wantErr: true, errMsg: "?"},
		{name: "contains asterisk", taskID: "task*123", wantErr: true, errMsg: "*"},
		{name: "contains square bracket", taskID: "task[123", wantErr: true, errMsg: "["},

		// Invalid - control characters
		{name: "contains newline", taskID: "task\n123", wantErr: true, errMsg: "control"},
		{name: "contains tab", taskID: "task\t123", wantErr: true, errMsg: "control"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTaskID(tt.taskID)

			if tt.wantErr {
				if err == nil {
					t.Errorf("validateTaskID(%q) expected error containing %q, got nil", tt.taskID, tt.errMsg)
				} else if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("validateTaskID(%q) expected error containing %q, got %q", tt.taskID, tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("validateTaskID(%q) expected no error, got %v", tt.taskID, err)
				}
			}
		})
	}
}

func TestIsWorkspaceDirty_DirtyWorkspace(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "giverny-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Initialize a git repository
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to init git repo: %v", err)
	}

	// Configure git
	exec.Command("git", "-C", tmpDir, "config", "user.email", "test@example.com").Run()
	exec.Command("git", "-C", tmpDir, "config", "user.name", "Test User").Run()

	// Create an initial commit
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	exec.Command("git", "-C", tmpDir, "add", ".").Run()
	exec.Command("git", "-C", tmpDir, "commit", "-m", "initial commit").Run()

	// Make a change without committing
	if err := os.WriteFile(testFile, []byte("modified"), 0644); err != nil {
		t.Fatalf("failed to modify test file: %v", err)
	}

	// Test isWorkspaceDirty by changing to the temp directory
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	os.Chdir(tmpDir)

	dirty, err := git.IsWorkspaceDirty()
	if err != nil {
		t.Errorf("IsWorkspaceDirty failed: %v", err)
	}

	if !dirty {
		t.Error("expected workspace to be dirty, but it was clean")
	}
}
