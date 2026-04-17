package dockerops

// MockDockerOps is a mock implementation of DockerOps for testing
type MockDockerOps struct {
	// Function stubs that can be set in tests
	BuildImageFunc      func(baseImage string, showOutput bool, forceRebuild bool, debug bool) error
	RunContainerFunc    func(taskID, slug, prompt, baseImage string, gitPort int, dockerArgs, agentArgs string, debug, useAmp bool) (int, error)
	RemoveContainerFunc func(containerName string) error
}

// NewMockDockerOps creates a new MockDockerOps with default no-op implementations
func NewMockDockerOps() *MockDockerOps {
	return &MockDockerOps{
		BuildImageFunc: func(baseImage string, showOutput bool, forceRebuild bool, debug bool) error {
			return nil
		},
		RunContainerFunc: func(taskID, slug, prompt, baseImage string, gitPort int, dockerArgs, agentArgs string, debug, useAmp bool) (int, error) {
			return 0, nil
		},
		RemoveContainerFunc: func(containerName string) error {
			return nil
		},
	}
}

// BuildImage calls the mock function
func (m *MockDockerOps) BuildImage(baseImage string, showOutput bool, forceRebuild bool, debug bool) error {
	return m.BuildImageFunc(baseImage, showOutput, forceRebuild, debug)
}

// RunContainer calls the mock function
func (m *MockDockerOps) RunContainer(taskID, slug, prompt, baseImage string, gitPort int, dockerArgs, agentArgs string, debug, useAmp bool) (int, error) {
	return m.RunContainerFunc(taskID, slug, prompt, baseImage, gitPort, dockerArgs, agentArgs, debug, useAmp)
}

// RemoveContainer calls the mock function
func (m *MockDockerOps) RemoveContainer(containerName string) error {
	return m.RemoveContainerFunc(containerName)
}
