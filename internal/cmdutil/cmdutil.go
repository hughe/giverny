package cmdutil

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// RunCommand runs a command and returns an error if it fails.
// The command runs in the current working directory.
func RunCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run %s: %w", name, err)
	}
	return nil
}

// RunCommandInDir runs a command in the specified directory and returns an error if it fails.
func RunCommandInDir(dir, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run %s in %s: %w", name, dir, err)
	}
	return nil
}

// RunCommandWithOutput runs a command and returns its combined stdout/stderr output.
// Returns the output as a string and any error that occurred.
func RunCommandWithOutput(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to run %s: %w", name, err)
	}
	return strings.TrimSpace(string(output)), nil
}

// RunCommandInDirWithOutput runs a command in the specified directory and returns its combined stdout/stderr output.
func RunCommandInDirWithOutput(dir, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to run %s in %s: %w", name, dir, err)
	}
	return strings.TrimSpace(string(output)), nil
}

// RunCommandWithDebug runs a command with optional debug output.
// If debug is true, stdout and stderr are connected to os.Stdout and os.Stderr.
func RunCommandWithDebug(debug bool, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	if debug {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run %s: %w", name, err)
	}
	return nil
}

// RunCommandInDirWithDebug runs a command in the specified directory with optional debug output.
// If debug is true, stdout and stderr are connected to os.Stdout and os.Stderr.
func RunCommandInDirWithDebug(dir string, debug bool, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	if debug {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run %s in %s: %w", name, dir, err)
	}
	return nil
}
