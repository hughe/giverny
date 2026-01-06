package cmdutil

import (
	"os"
	"path/filepath"
	"strings"
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

func TestRunCommand(t *testing.T) {
	tests := []struct {
		name    string
		command string
		args    []string
		wantErr bool
	}{
		{
			name:    "successful command",
			command: "echo",
			args:    []string{"hello"},
			wantErr: false,
		},
		{
			name:    "failing command",
			command: "false",
			args:    []string{},
			wantErr: true,
		},
		{
			name:    "nonexistent command",
			command: "nonexistent-command-12345",
			args:    []string{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := RunCommand(tt.command, tt.args...)
			if (err != nil) != tt.wantErr {
				t.Errorf("RunCommand() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRunCommandInDir(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	tests := []struct {
		name    string
		dir     string
		command string
		args    []string
		setup   func()
		verify  func() error
		wantErr bool
	}{
		{
			name:    "successful command in directory",
			dir:     tmpDir,
			command: "touch",
			args:    []string{testFile},
			setup:   func() {},
			verify: func() error {
				if _, err := os.Stat(testFile); err != nil {
					return err
				}
				return nil
			},
			wantErr: false,
		},
		{
			name:    "command fails in directory",
			dir:     tmpDir,
			command: "false",
			args:    []string{},
			setup:   func() {},
			verify:  func() error { return nil },
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			err := RunCommandInDir(tt.dir, tt.command, tt.args...)
			if (err != nil) != tt.wantErr {
				t.Errorf("RunCommandInDir() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil {
				if verifyErr := tt.verify(); verifyErr != nil {
					t.Errorf("Verification failed: %v", verifyErr)
				}
			}
		})
	}
}

func TestRunCommandWithOutput(t *testing.T) {
	tests := []struct {
		name       string
		command    string
		args       []string
		wantOutput string
		wantErr    bool
	}{
		{
			name:       "successful command with output",
			command:    "echo",
			args:       []string{"hello world"},
			wantOutput: "hello world",
			wantErr:    false,
		},
		{
			name:       "command with no output",
			command:    "true",
			args:       []string{},
			wantOutput: "",
			wantErr:    false,
		},
		{
			name:       "failing command",
			command:    "false",
			args:       []string{},
			wantOutput: "",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := RunCommandWithOutput(tt.command, tt.args...)
			if (err != nil) != tt.wantErr {
				t.Errorf("RunCommandWithOutput() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && output != tt.wantOutput {
				t.Errorf("RunCommandWithOutput() output = %v, want %v", output, tt.wantOutput)
			}
		})
	}
}

func TestRunCommandInDirWithOutput(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name       string
		dir        string
		command    string
		args       []string
		wantOutput string
		wantErr    bool
	}{
		{
			name:       "successful command in directory with output",
			dir:        tmpDir,
			command:    "pwd",
			args:       []string{},
			wantOutput: tmpDir,
			wantErr:    false,
		},
		{
			name:       "failing command in directory",
			dir:        tmpDir,
			command:    "false",
			args:       []string{},
			wantOutput: "",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := RunCommandInDirWithOutput(tt.dir, tt.command, tt.args...)
			if (err != nil) != tt.wantErr {
				t.Errorf("RunCommandInDirWithOutput() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && output != tt.wantOutput {
				t.Errorf("RunCommandInDirWithOutput() output = %v, want %v", output, tt.wantOutput)
			}
		})
	}
}

func TestRunCommandWithDebug(t *testing.T) {
	tests := []struct {
		name    string
		debug   bool
		command string
		args    []string
		wantErr bool
	}{
		{
			name:    "successful command with debug true",
			debug:   true,
			command: "echo",
			args:    []string{"debug output"},
			wantErr: false,
		},
		{
			name:    "successful command with debug false",
			debug:   false,
			command: "echo",
			args:    []string{"no debug output"},
			wantErr: false,
		},
		{
			name:    "failing command with debug true",
			debug:   true,
			command: "false",
			args:    []string{},
			wantErr: true,
		},
		{
			name:    "failing command with debug false",
			debug:   false,
			command: "false",
			args:    []string{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := RunCommandWithDebug(tt.debug, tt.command, tt.args...)
			if (err != nil) != tt.wantErr {
				t.Errorf("RunCommandWithDebug() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRunCommandInDirWithDebug(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name    string
		dir     string
		debug   bool
		command string
		args    []string
		wantErr bool
	}{
		{
			name:    "successful command in dir with debug true",
			dir:     tmpDir,
			debug:   true,
			command: "pwd",
			args:    []string{},
			wantErr: false,
		},
		{
			name:    "successful command in dir with debug false",
			dir:     tmpDir,
			debug:   false,
			command: "pwd",
			args:    []string{},
			wantErr: false,
		},
		{
			name:    "failing command in dir with debug true",
			dir:     tmpDir,
			debug:   true,
			command: "false",
			args:    []string{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := RunCommandInDirWithDebug(tt.dir, tt.debug, tt.command, tt.args...)
			if (err != nil) != tt.wantErr {
				t.Errorf("RunCommandInDirWithDebug() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestErrorMessages(t *testing.T) {
	tests := []struct {
		name           string
		fn             func() error
		wantErrContain string
	}{
		{
			name: "RunCommand error message",
			fn: func() error {
				return RunCommand("false")
			},
			wantErrContain: "failed to run false",
		},
		{
			name: "RunCommandInDir error message",
			fn: func() error {
				return RunCommandInDir("/tmp", "false")
			},
			wantErrContain: "failed to run false in /tmp",
		},
		{
			name: "RunCommandWithOutput error message",
			fn: func() error {
				_, err := RunCommandWithOutput("false")
				return err
			},
			wantErrContain: "failed to run false",
		},
		{
			name: "RunCommandInDirWithOutput error message",
			fn: func() error {
				_, err := RunCommandInDirWithOutput("/tmp", "false")
				return err
			},
			wantErrContain: "failed to run false in /tmp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			if err == nil {
				t.Error("expected error but got nil")
				return
			}
			if !strings.Contains(err.Error(), tt.wantErrContain) {
				t.Errorf("error message %q does not contain %q", err.Error(), tt.wantErrContain)
			}
		})
	}
}
