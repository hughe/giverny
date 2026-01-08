package outie

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	"giverny/internal/dockerops"
	"giverny/internal/git"
	"giverny/internal/gitops"
	"giverny/internal/testutil"
)

// setupTestDir creates a temporary directory with a git repo for testing
func setupTestDir(t *testing.T) (tmpDir string, cleanup func()) {
	tmpDir, err := os.MkdirTemp("", "giverny-outie-unit-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	// Initialize git repo
	testutil.InitTestRepo(t, tmpDir)

	// Change to temp directory
	origDir, err := os.Getwd()
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to get working directory: %v", err)
	}

	if err := os.Chdir(tmpDir); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to change to temp dir: %v", err)
	}

	cleanup = func() {
		os.Chdir(origDir)
		os.RemoveAll(tmpDir)
	}

	return tmpDir, cleanup
}

// TestRunWithDeps_ValidatesClaudeToken verifies that Run checks for the OAuth token
func TestRunWithDeps_ValidatesClaudeToken(t *testing.T) {
	_, cleanup := setupTestDir(t)
	defer cleanup()

	// Save original token and restore after test
	originalToken := os.Getenv("CLAUDE_CODE_OAUTH_TOKEN")
	defer func() {
		if originalToken != "" {
			os.Setenv("CLAUDE_CODE_OAUTH_TOKEN", originalToken)
		} else {
			os.Unsetenv("CLAUDE_CODE_OAUTH_TOKEN")
		}
	}()

	// Clear token for test
	os.Unsetenv("CLAUDE_CODE_OAUTH_TOKEN")

	mockGit := gitops.NewMockGitOps()
	mockDocker := dockerops.NewMockDockerOps()

	config := Config{
		TaskID:     "test-task",
		Prompt:     "test prompt",
		BaseImage:  "alpine:latest",
		AllowDirty: true,
	}

	err := RunWithDeps(config, mockGit, mockDocker)

	if err == nil {
		t.Fatal("Expected error when CLAUDE_CODE_OAUTH_TOKEN is not set")
	}

	expectedMsg := "CLAUDE_CODE_OAUTH_TOKEN environment variable is not set"
	if !strings.Contains(err.Error(), expectedMsg) {
		t.Errorf("Expected error message to contain %q, got: %v", expectedMsg, err)
	}
}

// TestRunWithDeps_ChecksDirtyWorkspace verifies workspace dirty check behavior
func TestRunWithDeps_ChecksDirtyWorkspace(t *testing.T) {
	_, cleanup := setupTestDir(t)
	defer cleanup()

	// Set token for test
	originalToken := os.Getenv("CLAUDE_CODE_OAUTH_TOKEN")
	os.Setenv("CLAUDE_CODE_OAUTH_TOKEN", "test-token")
	defer func() {
		if originalToken != "" {
			os.Setenv("CLAUDE_CODE_OAUTH_TOKEN", originalToken)
		} else {
			os.Unsetenv("CLAUDE_CODE_OAUTH_TOKEN")
		}
	}()

	t.Run("rejects dirty workspace by default", func(t *testing.T) {
		mockGit := gitops.NewMockGitOps()
		mockGit.IsWorkspaceDirtyFunc = func() (bool, error) {
			return true, nil // Workspace is dirty
		}

		mockDocker := dockerops.NewMockDockerOps()

		config := Config{
			TaskID:     "test-task",
			Prompt:     "test prompt",
			BaseImage:  "alpine:latest",
			AllowDirty: false,
		}

		err := RunWithDeps(config, mockGit, mockDocker)

		if err == nil {
			t.Fatal("Expected error when workspace is dirty")
		}

		expectedMsg := "working directory has uncommitted changes"
		if err.Error() != expectedMsg+". Commit or stash them first, or use --allow-dirty flag" {
			t.Errorf("Expected error about dirty workspace, got: %v", err)
		}
	})

	t.Run("allows dirty workspace with AllowDirty flag", func(t *testing.T) {
		branchCreated := false
		serverStarted := false
		imageBuilt := false
		containerRan := false

		mockGit := gitops.NewMockGitOps()
		mockGit.IsWorkspaceDirtyFunc = func() (bool, error) {
			return true, nil // Workspace is dirty
		}
		mockGit.CreateBranchFunc = func(branchName string) error {
			branchCreated = true
			return nil
		}
		mockGit.StartServerFunc = func(repoPath string) (*git.ServerCmd, int, error) {
			serverStarted = true
			return &git.ServerCmd{}, 9999, nil
		}
		mockGit.StopServerFunc = func(serverCmd *git.ServerCmd) error {
			return nil
		}
		mockGit.GetBranchCommitRangeFunc = func(branchName string) (string, string, error) {
			return "", "", nil
		}

		mockDocker := dockerops.NewMockDockerOps()
		mockDocker.BuildImageFunc = func(baseImage string, showOutput bool, debug bool) error {
			imageBuilt = true
			return nil
		}
		mockDocker.RunContainerFunc = func(taskID, prompt string, gitPort int, dockerArgs, agentArgs string, debug bool) (int, error) {
			containerRan = true
			return 0, nil // Success
		}
		mockDocker.RemoveContainerFunc = func(containerName string) error {
			return nil
		}

		config := Config{
			TaskID:     "test-task",
			Prompt:     "test prompt",
			BaseImage:  "alpine:latest",
			AllowDirty: true,
		}

		err := RunWithDeps(config, mockGit, mockDocker)

		if err != nil {
			t.Fatalf("Unexpected error with AllowDirty flag: %v", err)
		}

		if !branchCreated {
			t.Error("Expected branch to be created")
		}
		if !serverStarted {
			t.Error("Expected git server to be started")
		}
		if !imageBuilt {
			t.Error("Expected Docker image to be built")
		}
		if !containerRan {
			t.Error("Expected container to run")
		}
	})

	t.Run("skips dirty check with ExistingBranch flag", func(t *testing.T) {
		dirtyCheckCalled := false

		mockGit := gitops.NewMockGitOps()
		mockGit.IsWorkspaceDirtyFunc = func() (bool, error) {
			dirtyCheckCalled = true
			return true, nil
		}
		mockGit.BranchExistsFunc = func(branchName string) (bool, error) {
			return true, nil
		}
		mockGit.StartServerFunc = func(repoPath string) (*git.ServerCmd, int, error) {
			return &git.ServerCmd{}, 9999, nil
		}
		mockGit.StopServerFunc = func(serverCmd *git.ServerCmd) error {
			return nil
		}
		mockGit.GetBranchCommitRangeFunc = func(branchName string) (string, string, error) {
			return "", "", nil
		}

		mockDocker := dockerops.NewMockDockerOps()
		mockDocker.BuildImageFunc = func(baseImage string, showOutput bool, debug bool) error {
			return nil
		}
		mockDocker.RunContainerFunc = func(taskID, prompt string, gitPort int, dockerArgs, agentArgs string, debug bool) (int, error) {
			return 0, nil
		}
		mockDocker.RemoveContainerFunc = func(containerName string) error {
			return nil
		}

		config := Config{
			TaskID:         "test-task",
			Prompt:         "test prompt",
			BaseImage:      "alpine:latest",
			ExistingBranch: true,
		}

		err := RunWithDeps(config, mockGit, mockDocker)

		if err != nil {
			t.Fatalf("Unexpected error with ExistingBranch flag: %v", err)
		}

		if dirtyCheckCalled {
			t.Error("Dirty check should not be called when ExistingBranch is true")
		}
	})
}

// TestRunWithDeps_HandlesGitErrors verifies error handling for git operations
func TestRunWithDeps_HandlesGitErrors(t *testing.T) {
	_, cleanup := setupTestDir(t)
	defer cleanup()

	// Set token for test
	originalToken := os.Getenv("CLAUDE_CODE_OAUTH_TOKEN")
	os.Setenv("CLAUDE_CODE_OAUTH_TOKEN", "test-token")
	defer func() {
		if originalToken != "" {
			os.Setenv("CLAUDE_CODE_OAUTH_TOKEN", originalToken)
		} else {
			os.Unsetenv("CLAUDE_CODE_OAUTH_TOKEN")
		}
	}()

	t.Run("handles branch creation failure", func(t *testing.T) {
		mockGit := gitops.NewMockGitOps()
		mockGit.CreateBranchFunc = func(branchName string) error {
			return errors.New("branch already exists")
		}

		mockDocker := dockerops.NewMockDockerOps()

		config := Config{
			TaskID:     "test-task",
			Prompt:     "test prompt",
			BaseImage:  "alpine:latest",
			AllowDirty: true,
		}

		err := RunWithDeps(config, mockGit, mockDocker)

		if err == nil {
			t.Fatal("Expected error when branch creation fails")
		}

		expectedMsg := "failed to create branch"
		if err.Error() != expectedMsg+": branch already exists" {
			t.Errorf("Expected error about branch creation, got: %v", err)
		}
	})

	t.Run("handles server start failure", func(t *testing.T) {
		mockGit := gitops.NewMockGitOps()
		mockGit.CreateBranchFunc = func(branchName string) error {
			return nil
		}
		mockGit.StartServerFunc = func(repoPath string) (*git.ServerCmd, int, error) {
			return nil, 0, errors.New("port already in use")
		}

		mockDocker := dockerops.NewMockDockerOps()

		config := Config{
			TaskID:     "test-task",
			Prompt:     "test prompt",
			BaseImage:  "alpine:latest",
			AllowDirty: true,
		}

		err := RunWithDeps(config, mockGit, mockDocker)

		if err == nil {
			t.Fatal("Expected error when server start fails")
		}

		expectedMsg := "failed to start git server"
		if err.Error() != expectedMsg+": port already in use" {
			t.Errorf("Expected error about server start, got: %v", err)
		}
	})
}

// TestRunWithDeps_HandlesDockerErrors verifies error handling for Docker operations
func TestRunWithDeps_HandlesDockerErrors(t *testing.T) {
	_, cleanup := setupTestDir(t)
	defer cleanup()

	// Set token for test
	originalToken := os.Getenv("CLAUDE_CODE_OAUTH_TOKEN")
	os.Setenv("CLAUDE_CODE_OAUTH_TOKEN", "test-token")
	defer func() {
		if originalToken != "" {
			os.Setenv("CLAUDE_CODE_OAUTH_TOKEN", originalToken)
		} else {
			os.Unsetenv("CLAUDE_CODE_OAUTH_TOKEN")
		}
	}()

	t.Run("handles build failure", func(t *testing.T) {
		mockGit := gitops.NewMockGitOps()
		mockGit.StartServerFunc = func(repoPath string) (*git.ServerCmd, int, error) {
			return &git.ServerCmd{}, 9999, nil
		}
		mockGit.StopServerFunc = func(serverCmd *git.ServerCmd) error {
			return nil
		}

		mockDocker := dockerops.NewMockDockerOps()
		mockDocker.BuildImageFunc = func(baseImage string, showOutput bool, debug bool) error {
			return errors.New("docker build failed")
		}

		config := Config{
			TaskID:     "test-task",
			Prompt:     "test prompt",
			BaseImage:  "alpine:latest",
			AllowDirty: true,
		}

		err := RunWithDeps(config, mockGit, mockDocker)

		if err == nil {
			t.Fatal("Expected error when docker build fails")
		}

		expectedMsg := "failed to build image"
		if err.Error() != expectedMsg+": docker build failed" {
			t.Errorf("Expected error about build failure, got: %v", err)
		}
	})

	t.Run("handles container run failure", func(t *testing.T) {
		mockGit := gitops.NewMockGitOps()
		mockGit.StartServerFunc = func(repoPath string) (*git.ServerCmd, int, error) {
			return &git.ServerCmd{}, 9999, nil
		}
		mockGit.StopServerFunc = func(serverCmd *git.ServerCmd) error {
			return nil
		}

		mockDocker := dockerops.NewMockDockerOps()
		mockDocker.BuildImageFunc = func(baseImage string, showOutput bool, debug bool) error {
			return nil
		}
		mockDocker.RunContainerFunc = func(taskID, prompt string, gitPort int, dockerArgs, agentArgs string, debug bool) (int, error) {
			return 1, nil // Non-zero exit code
		}

		config := Config{
			TaskID:     "test-task",
			Prompt:     "test prompt",
			BaseImage:  "alpine:latest",
			AllowDirty: true,
		}

		err := RunWithDeps(config, mockGit, mockDocker)

		if err == nil {
			t.Fatal("Expected error when container exits with non-zero code")
		}

		expectedMsg := "container exited with code 1"
		if err.Error() != expectedMsg {
			t.Errorf("Expected error about exit code, got: %v", err)
		}
	})
}

// TestRunWithDeps_SuccessfulFlow verifies the complete successful workflow
func TestRunWithDeps_SuccessfulFlow(t *testing.T) {
	_, cleanup := setupTestDir(t)
	defer cleanup()

	// Set token for test
	originalToken := os.Getenv("CLAUDE_CODE_OAUTH_TOKEN")
	os.Setenv("CLAUDE_CODE_OAUTH_TOKEN", "test-token")
	defer func() {
		if originalToken != "" {
			os.Setenv("CLAUDE_CODE_OAUTH_TOKEN", originalToken)
		} else {
			os.Unsetenv("CLAUDE_CODE_OAUTH_TOKEN")
		}
	}()

	// Track call sequence
	var callSequence []string

	mockGit := gitops.NewMockGitOps()
	mockGit.IsWorkspaceDirtyFunc = func() (bool, error) {
		callSequence = append(callSequence, "IsWorkspaceDirty")
		return false, nil
	}
	mockGit.CreateBranchFunc = func(branchName string) error {
		callSequence = append(callSequence, "CreateBranch")
		if branchName != "giverny/test-task" {
			return fmt.Errorf("unexpected branch name: %s", branchName)
		}
		return nil
	}
	mockGit.StartServerFunc = func(repoPath string) (*git.ServerCmd, int, error) {
		callSequence = append(callSequence, "StartServer")
		return &git.ServerCmd{}, 9999, nil
	}
	mockGit.StopServerFunc = func(serverCmd *git.ServerCmd) error {
		callSequence = append(callSequence, "StopServer")
		return nil
	}
	mockGit.GetBranchCommitRangeFunc = func(branchName string) (string, string, error) {
		callSequence = append(callSequence, "GetBranchCommitRange")
		return "abc123", "def456", nil
	}
	mockGit.GetShortHashFunc = func(hash string) string {
		callSequence = append(callSequence, fmt.Sprintf("GetShortHash(%s)", hash))
		return hash[:6]
	}

	mockDocker := dockerops.NewMockDockerOps()
	mockDocker.BuildImageFunc = func(baseImage string, showOutput bool, debug bool) error {
		callSequence = append(callSequence, "BuildImage")
		if baseImage != "alpine:latest" {
			return fmt.Errorf("unexpected base image: %s", baseImage)
		}
		return nil
	}
	mockDocker.RunContainerFunc = func(taskID, prompt string, gitPort int, dockerArgs, agentArgs string, debug bool) (int, error) {
		callSequence = append(callSequence, "RunContainer")
		if taskID != "test-task" {
			return 1, fmt.Errorf("unexpected task ID: %s", taskID)
		}
		if prompt != "test prompt" {
			return 1, fmt.Errorf("unexpected prompt: %s", prompt)
		}
		if gitPort != 9999 {
			return 1, fmt.Errorf("unexpected git port: %d", gitPort)
		}
		return 0, nil
	}
	mockDocker.RemoveContainerFunc = func(containerName string) error {
		callSequence = append(callSequence, "RemoveContainer")
		if containerName != "giverny-test-task" {
			return fmt.Errorf("unexpected container name: %s", containerName)
		}
		return nil
	}

	config := Config{
		TaskID:     "test-task",
		Prompt:     "test prompt",
		BaseImage:  "alpine:latest",
		AllowDirty: false,
	}

	err := RunWithDeps(config, mockGit, mockDocker)

	if err != nil {
		t.Fatalf("Unexpected error in successful flow: %v", err)
	}

	// Verify call sequence
	// Note: StopServer is called via defer, so it runs after GetBranchCommitRange/GetShortHash
	expectedSequence := []string{
		"IsWorkspaceDirty",
		"CreateBranch",
		"StartServer",
		"BuildImage",
		"RunContainer",
		"RemoveContainer",
		"GetBranchCommitRange",
		"GetShortHash(abc123)",
		"GetShortHash(def456)",
		"StopServer",
	}

	if len(callSequence) != len(expectedSequence) {
		t.Fatalf("Expected %d calls, got %d: %v", len(expectedSequence), len(callSequence), callSequence)
	}

	for i, expected := range expectedSequence {
		if callSequence[i] != expected {
			t.Errorf("Call %d: expected %q, got %q", i, expected, callSequence[i])
		}
	}
}
