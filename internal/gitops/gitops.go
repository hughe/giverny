package gitops

import "giverny/internal/git"

// GitOps defines the interface for all git operations needed by outie and innie.
// This interface allows for mocking git operations in tests.
type GitOps interface {
	// Branch operations
	IsWorkspaceDirty() (bool, error)
	BranchExists(branchName string) (bool, error)
	CreateBranch(branchName string) error
	GetBranchCommitRange(branchName string) (firstCommit, lastCommit string, err error)
	GetShortHash(hash string) string

	// Server operations
	StartServer(repoPath string) (*git.ServerCmd, int, error)
	StopServer(serverCmd *git.ServerCmd) error

	// Repository operations (for innie)
	CloneRepo(gitPort int, debug bool) error
	SetupWorkspace(branchName string, debug bool) error
	PushBranch(branchName string, gitPort int, debug bool) error
}

// RealGitOps implements GitOps using the actual git package functions
type RealGitOps struct{}

// NewRealGitOps creates a new RealGitOps instance
func NewRealGitOps() *RealGitOps {
	return &RealGitOps{}
}

// IsWorkspaceDirty checks if the workspace has uncommitted changes
func (g *RealGitOps) IsWorkspaceDirty() (bool, error) {
	return git.IsWorkspaceDirty()
}

// BranchExists checks if a branch exists
func (g *RealGitOps) BranchExists(branchName string) (bool, error) {
	return git.BranchExists(branchName)
}

// CreateBranch creates a new git branch
func (g *RealGitOps) CreateBranch(branchName string) error {
	return git.CreateBranch(branchName)
}

// GetBranchCommitRange gets the first and last commit of a branch
func (g *RealGitOps) GetBranchCommitRange(branchName string) (firstCommit, lastCommit string, err error) {
	return git.GetBranchCommitRange(branchName)
}

// GetShortHash converts a full hash to short form
func (g *RealGitOps) GetShortHash(hash string) string {
	return git.GetShortHash(hash)
}

// StartServer starts a git daemon server
func (g *RealGitOps) StartServer(repoPath string) (*git.ServerCmd, int, error) {
	return git.StartServer(repoPath)
}

// StopServer stops a running git server
func (g *RealGitOps) StopServer(serverCmd *git.ServerCmd) error {
	return git.StopServer(serverCmd)
}

// CloneRepo clones the repository from the git server
func (g *RealGitOps) CloneRepo(gitPort int, debug bool) error {
	return git.CloneRepo(gitPort, debug)
}

// SetupWorkspace sets up the workspace in /app
func (g *RealGitOps) SetupWorkspace(branchName string, debug bool) error {
	return git.SetupWorkspace(branchName, debug)
}

// PushBranch pushes the branch to the git server
func (g *RealGitOps) PushBranch(branchName string, gitPort int, debug bool) error {
	return git.PushBranch(branchName, gitPort, debug)
}
