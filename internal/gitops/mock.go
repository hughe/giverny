package gitops

import "giverny/internal/git"

// MockGitOps is a mock implementation of GitOps for testing
type MockGitOps struct {
	// Function stubs that can be set in tests
	IsWorkspaceDirtyFunc       func() (bool, error)
	BranchExistsFunc           func(branchName string) (bool, error)
	CreateBranchFunc           func(branchName string) error
	GetBranchCommitRangeFunc   func(branchName string) (firstCommit, lastCommit string, err error)
	GetShortHashFunc           func(hash string) string
	StartServerFunc            func(repoPath string) (*git.ServerCmd, int, error)
	StopServerFunc             func(serverCmd *git.ServerCmd) error
	CloneRepoFunc              func(gitPort int, debug bool) error
	SetupWorkspaceFunc         func(branchName string, debug bool) error
	PushBranchFunc             func(branchName string, gitPort int, debug bool) error
}

// NewMockGitOps creates a new MockGitOps with default no-op implementations
func NewMockGitOps() *MockGitOps {
	return &MockGitOps{
		IsWorkspaceDirtyFunc: func() (bool, error) {
			return false, nil
		},
		BranchExistsFunc: func(branchName string) (bool, error) {
			return true, nil
		},
		CreateBranchFunc: func(branchName string) error {
			return nil
		},
		GetBranchCommitRangeFunc: func(branchName string) (firstCommit, lastCommit string, err error) {
			return "", "", nil
		},
		GetShortHashFunc: func(hash string) string {
			return hash[:7]
		},
		StartServerFunc: func(repoPath string) (*git.ServerCmd, int, error) {
			return &git.ServerCmd{}, 9999, nil
		},
		StopServerFunc: func(serverCmd *git.ServerCmd) error {
			return nil
		},
		CloneRepoFunc: func(gitPort int, debug bool) error {
			return nil
		},
		SetupWorkspaceFunc: func(branchName string, debug bool) error {
			return nil
		},
		PushBranchFunc: func(branchName string, gitPort int, debug bool) error {
			return nil
		},
	}
}

// IsWorkspaceDirty calls the mock function
func (m *MockGitOps) IsWorkspaceDirty() (bool, error) {
	return m.IsWorkspaceDirtyFunc()
}

// BranchExists calls the mock function
func (m *MockGitOps) BranchExists(branchName string) (bool, error) {
	return m.BranchExistsFunc(branchName)
}

// CreateBranch calls the mock function
func (m *MockGitOps) CreateBranch(branchName string) error {
	return m.CreateBranchFunc(branchName)
}

// GetBranchCommitRange calls the mock function
func (m *MockGitOps) GetBranchCommitRange(branchName string) (firstCommit, lastCommit string, err error) {
	return m.GetBranchCommitRangeFunc(branchName)
}

// GetShortHash calls the mock function
func (m *MockGitOps) GetShortHash(hash string) string {
	return m.GetShortHashFunc(hash)
}

// StartServer calls the mock function
func (m *MockGitOps) StartServer(repoPath string) (*git.ServerCmd, int, error) {
	return m.StartServerFunc(repoPath)
}

// StopServer calls the mock function
func (m *MockGitOps) StopServer(serverCmd *git.ServerCmd) error {
	return m.StopServerFunc(serverCmd)
}

// CloneRepo calls the mock function
func (m *MockGitOps) CloneRepo(gitPort int, debug bool) error {
	return m.CloneRepoFunc(gitPort, debug)
}

// SetupWorkspace calls the mock function
func (m *MockGitOps) SetupWorkspace(branchName string, debug bool) error {
	return m.SetupWorkspaceFunc(branchName, debug)
}

// PushBranch calls the mock function
func (m *MockGitOps) PushBranch(branchName string, gitPort int, debug bool) error {
	return m.PushBranchFunc(branchName, gitPort, debug)
}
