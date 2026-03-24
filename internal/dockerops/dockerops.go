package dockerops

import "giverny/internal/docker"

// DockerOps defines the interface for all Docker operations needed by outie.
// This interface allows for mocking Docker operations in tests.
type DockerOps interface {
	// BuildImage builds the giverny Docker images (deps and main)
	BuildImage(baseImage string, showOutput bool, debug bool) error

	// RunContainer runs the giverny container and returns the exit code
	RunContainer(taskID, slug, prompt string, gitPort int, dockerArgs, agentArgs string, debug, useAmp bool) (int, error)

	// RemoveContainer removes a Docker container by name
	RemoveContainer(containerName string) error
}

// RealDockerOps implements DockerOps using the actual docker package functions
type RealDockerOps struct{}

// NewRealDockerOps creates a new RealDockerOps instance
func NewRealDockerOps() *RealDockerOps {
	return &RealDockerOps{}
}

// BuildImage builds the giverny Docker images
func (d *RealDockerOps) BuildImage(baseImage string, showOutput bool, debug bool) error {
	return docker.BuildImage(baseImage, showOutput, debug)
}

// RunContainer runs the giverny container
func (d *RealDockerOps) RunContainer(taskID, slug, prompt string, gitPort int, dockerArgs, agentArgs string, debug, useAmp bool) (int, error) {
	return docker.RunContainer(taskID, slug, prompt, gitPort, dockerArgs, agentArgs, debug, useAmp)
}

// RemoveContainer removes a Docker container
func (d *RealDockerOps) RemoveContainer(containerName string) error {
	return docker.RemoveContainer(containerName)
}
