package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
	"giverny"
	"giverny/internal/docker"
	"giverny/internal/innie"
	"giverny/internal/outie"
)

func init() {
	// Initialize the embedded source for the docker package
	docker.EmbeddedSource = giverny.Source
}

type Config struct {
	TaskID          string
	Prompt          string
	BaseImage       string
	DockerArgs      string
	IsInnie         bool
	GitServerPort   int
	Debug           bool
	ShowBuildOutput bool
}

var (
	config Config
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "giverny [OPTIONS] TASK-ID [PROMPT]",
		Short: "Containerized system for running Claude Code safely",
		Long:  "Giverny creates isolated Docker environments where Claude Code can work on tasks without affecting the host system.",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			config.TaskID = args[0]

			// Validate TASK-ID
			if err := validateTaskID(config.TaskID); err != nil {
				return fmt.Errorf("invalid TASK-ID: %w", err)
			}

			// Set prompt - default or from argument
			if len(args) >= 2 {
				config.Prompt = args[1]
			} else {
				config.Prompt = fmt.Sprintf("Please work on %s.", config.TaskID)
			}

			// Validate innie-specific requirements
			if config.IsInnie && config.GitServerPort == 0 {
				return fmt.Errorf("--git-server-port is required when --innie is set")
			}

			// Execute appropriate mode
			if config.IsInnie {
				innieConfig := innie.Config{
					TaskID:        config.TaskID,
					Prompt:        config.Prompt,
					GitServerPort: config.GitServerPort,
					Debug:         config.Debug,
				}
				return innie.Run(innieConfig)
			}
			outieConfig := outie.Config{
				TaskID:          config.TaskID,
				Prompt:          config.Prompt,
				BaseImage:       config.BaseImage,
				DockerArgs:      config.DockerArgs,
				Debug:           config.Debug,
				ShowBuildOutput: config.ShowBuildOutput,
			}
			return outie.Run(outieConfig)
		},
	}

	// Define flags
	rootCmd.Flags().StringVar(&config.BaseImage, "base-image", "giverny:latest", "Docker base image")
	rootCmd.Flags().StringVar(&config.DockerArgs, "docker-args", "", "Additional docker run arguments")
	rootCmd.Flags().BoolVar(&config.Debug, "debug", false, "Enable debug output")
	rootCmd.Flags().BoolVar(&config.ShowBuildOutput, "show-build-output", false, "Show docker build output")

	// Hidden flags (for internal use only)
	rootCmd.Flags().BoolVar(&config.IsInnie, "innie", false, "Internal flag for running inside container")
	rootCmd.Flags().IntVar(&config.GitServerPort, "git-server-port", 0, "Internal flag for git server port")
	rootCmd.Flags().MarkHidden("innie")
	rootCmd.Flags().MarkHidden("git-server-port")

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// validateTaskID ensures TASK-ID contains only characters valid in git branch names.
// Since we use the format "giverny/TASK-ID", the TASK-ID must not contain '/' and
// must follow git branch naming rules.
func validateTaskID(taskID string) error {
	if taskID == "" {
		return fmt.Errorf("TASK-ID cannot be empty")
	}

	// Regex for invalid characters in git branch names:
	// - No forward slash, backslash, space, control chars (0-31, 127)
	// - No ~, ^, :, ?, *, [
	invalidCharsPattern := regexp.MustCompile(`[/\\\s\x00-\x1f\x7f~^:?*\[]`)
	if match := invalidCharsPattern.FindString(taskID); match != "" {
		if match == "/" {
			return fmt.Errorf("TASK-ID cannot contain forward slash (/)")
		} else if match == "\\" {
			return fmt.Errorf("TASK-ID cannot contain backslash (\\)")
		} else if match == " " {
			return fmt.Errorf("TASK-ID cannot contain spaces")
		} else if match[0] < 32 || match[0] == 127 {
			return fmt.Errorf("TASK-ID cannot contain control characters")
		}
		return fmt.Errorf("TASK-ID cannot contain '%s'", match)
	}

	// Check for special invalid patterns
	if strings.Contains(taskID, "..") {
		return fmt.Errorf("TASK-ID cannot contain double dots (..)")
	}
	if strings.Contains(taskID, "@{") {
		return fmt.Errorf("TASK-ID cannot contain @{")
	}

	// Check if starts with dot
	if strings.HasPrefix(taskID, ".") {
		return fmt.Errorf("TASK-ID cannot start with a dot")
	}

	// Check if ends with .lock
	if strings.HasSuffix(taskID, ".lock") {
		return fmt.Errorf("TASK-ID cannot end with .lock")
	}

	return nil
}
