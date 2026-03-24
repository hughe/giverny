package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
	"giverny"
	"giverny/internal/ctrlsock"
	"giverny/internal/docker"
	"giverny/internal/innie"
	"giverny/internal/outie"
)

// Version information - injected at build time via -ldflags
var (
	versionTag     string
	versionTagHash string
	versionHash    string
	versionBranch  string
)

func init() {
	// Initialize the embedded source for the docker package
	docker.EmbeddedSource = giverny.Source
}

type Config struct {
	TaskID          string
	Slug            string
	Prompt          string
	BaseImage       string
	DockerArgs      string
	AgentArgs       string
	IsInnie         bool
	GitServerPort   int
	Debug           bool
	ShowBuildOutput bool
	ExistingBranch  bool
	AllowDirty      bool
	UseAmp          bool
	CtrlSend        string
}

var (
	config      Config
	showVersion bool
)

// getVersion returns the formatted version string
func getVersion() string {
	if versionTag == "" {
		versionTag = "v0.0.0"
	}
	if versionHash == "" {
		versionHash = "unknown"
	}
	if versionTagHash == "" {
		versionTagHash = "unknown"
	}
	if versionBranch == "" {
		versionBranch = "unknown"
	}

	// If current commit matches the tagged commit, omit the hash
	onTaggedCommit := versionHash == versionTagHash

	// Don't print branch name if it's "main"
	onMainBranch := versionBranch == "main"

	if onTaggedCommit {
		return versionTag
	}

	if onMainBranch {
		return fmt.Sprintf("%s.%s", versionTag, versionHash)
	}
	return fmt.Sprintf("%s.%s %s", versionTag, versionHash, versionBranch)
}

func main() {
	rootCmd := &cobra.Command{
		Use:   "giverny [OPTIONS] TASK-ID [SLUG] [PROMPT]",
		Short: "Containerized system for running Claude Code safely",
		Long:  "Giverny creates isolated Docker environments where Claude Code can work on tasks without affecting the host system.",
		Args:  cobra.RangeArgs(0, 3),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Handle --version flag
			if showVersion {
				fmt.Println(getVersion())
				return nil
			}

			// Handle --ctrl-send: send a message on the control socket and exit
			if config.CtrlSend != "" {
				addr := ctrlsock.ContainerAddr()
				if addr == "" {
					return fmt.Errorf("%s environment variable is not set", ctrlsock.EnvVar)
				}
				return ctrlsock.Send(addr, config.CtrlSend)
			}

			// Require TASK-ID if not showing version
			if len(args) < 1 {
				return fmt.Errorf("TASK-ID is required")
			}
			config.TaskID = args[0]

			// Validate TASK-ID
			if err := validateTaskID(config.TaskID); err != nil {
				return fmt.Errorf("invalid TASK-ID: %w", err)
			}

			// Set slug and prompt based on number of arguments
			switch len(args) {
			case 1:
				// Only TASK-ID provided
				config.Slug = ""
				config.Prompt = fmt.Sprintf("Please work on %s.", config.TaskID)
			case 2:
				// TASK-ID and SLUG provided
				config.Slug = sanitizeSlug(args[1])
				config.Prompt = fmt.Sprintf("Please work on %s.", config.TaskID)
			case 3:
				// TASK-ID, SLUG, and PROMPT provided
				config.Slug = sanitizeSlug(args[1])
				config.Prompt = args[2]
			}

			// Validate innie-specific requirements
			if config.IsInnie && config.GitServerPort == 0 {
				return fmt.Errorf("--git-server-port is required when --innie is set")
			}

			// Execute appropriate mode
			if config.IsInnie {
				innieConfig := innie.Config{
					TaskID:        config.TaskID,
					Slug:          config.Slug,
					Prompt:        config.Prompt,
					GitServerPort: config.GitServerPort,
					AgentArgs:     config.AgentArgs,
					Debug:         config.Debug,
					UseAmp:        config.UseAmp,
				}
				return innie.Run(innieConfig)
			}
			outieConfig := outie.Config{
				TaskID:          config.TaskID,
				Slug:            config.Slug,
				Prompt:          config.Prompt,
				BaseImage:       config.BaseImage,
				DockerArgs:      config.DockerArgs,
				AgentArgs:       config.AgentArgs,
				Debug:           config.Debug,
				ShowBuildOutput: config.ShowBuildOutput,
				ExistingBranch:  config.ExistingBranch,
				AllowDirty:      config.AllowDirty,
				UseAmp:          config.UseAmp,
			}
			return outie.Run(outieConfig)
		},
	}

	// Define flags
	rootCmd.Flags().BoolVar(&showVersion, "version", false, "Show version information")
	rootCmd.Flags().StringVar(&config.BaseImage, "base-image", "giverny:latest", "Docker base image")
	rootCmd.Flags().StringVar(&config.DockerArgs, "docker-args", "", "Additional docker run arguments")
	rootCmd.Flags().StringVar(&config.AgentArgs, "agent-args", "", "Additional arguments to pass to the agent (claude code)")
	rootCmd.Flags().BoolVar(&config.Debug, "debug", false, "Enable debug output")
	rootCmd.Flags().BoolVar(&config.ShowBuildOutput, "show-build-output", false, "Show docker build output")
	rootCmd.Flags().BoolVar(&config.ExistingBranch, "existing-branch", false, "Use existing branch instead of creating a new one")
	rootCmd.Flags().BoolVar(&config.AllowDirty, "allow-dirty", false, "Allow creating branch even if working directory has uncommitted changes")
	rootCmd.Flags().BoolVarP(&config.UseAmp, "amp", "a", false, "Use Amp instead of Claude Code as the agent")

	// Hidden flags (for internal use only)
	rootCmd.Flags().BoolVar(&config.IsInnie, "innie", false, "Internal flag for running inside container")
	rootCmd.Flags().IntVar(&config.GitServerPort, "git-server-port", 0, "Internal flag for git server port")
	rootCmd.Flags().StringVar(&config.CtrlSend, "ctrl-send", "", "Send a message on the control socket and exit")
	rootCmd.Flags().MarkHidden("innie")
	rootCmd.Flags().MarkHidden("git-server-port")
	rootCmd.Flags().MarkHidden("ctrl-send")

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// sanitizeSlug replaces any characters that are not safe for git branch names
// or docker container names with hyphens. Also collapses multiple consecutive
// hyphens into a single hyphen and trims leading/trailing hyphens.
func sanitizeSlug(slug string) string {
	// Replace any character that's not alphanumeric, hyphen, or underscore with a hyphen
	invalidCharsPattern := regexp.MustCompile(`[^a-zA-Z0-9_-]+`)
	result := invalidCharsPattern.ReplaceAllString(slug, "-")

	// Collapse multiple consecutive hyphens into one
	multipleHyphens := regexp.MustCompile(`-+`)
	result = multipleHyphens.ReplaceAllString(result, "-")

	// Trim leading and trailing hyphens
	result = strings.Trim(result, "-")

	return result
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
