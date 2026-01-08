package shell

import (
	"os"
	"testing"
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

func TestDetect(t *testing.T) {
	// Test that Detect returns one of the expected shells
	result := Detect()

	// Result should be one of the valid shells
	validShells := []string{"/bin/zsh", "/bin/bash", "/bin/sh"}
	valid := false
	for _, shell := range validShells {
		if result == shell {
			valid = true
			break
		}
	}

	if !valid {
		t.Errorf("Detect() returned unexpected shell: %v, expected one of %v", result, validShells)
	}

	// Verify the returned shell actually exists
	if _, err := os.Stat(result); err != nil {
		t.Errorf("Detect() returned shell %v that does not exist: %v", result, err)
	}
}

func TestDetect_PreferenceOrder(t *testing.T) {
	// This test documents the preference order
	// We can't easily mock os.Stat, so we just verify the behavior
	result := Detect()

	// The result should always be a valid shell path
	validShells := []string{"/bin/zsh", "/bin/bash", "/bin/sh"}
	valid := false
	for _, shell := range validShells {
		if result == shell {
			valid = true
			break
		}
	}

	if !valid {
		t.Errorf("Detect() returned unexpected shell: %v, expected one of %v", result, validShells)
	}

	// Verify the returned shell actually exists
	if _, err := os.Stat(result); err != nil {
		t.Errorf("Detect() returned shell %v that does not exist: %v", result, err)
	}

	// Log the preference for documentation purposes
	t.Logf("Detected shell: %s (preference order: zsh > bash > sh)", result)
}
