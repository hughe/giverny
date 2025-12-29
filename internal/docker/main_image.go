package docker

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

const dockerfileMainTemplate = `# Get giverny binary from innie image
FROM giverny-innie:latest AS innie

# Start from base image
FROM {{.BaseImage}}

# Install git if not present
RUN command -v git >/dev/null 2>&1 || \
    (apt-get update && apt-get install -y git) || \
    (apk add --no-cache git) || \
    (yum install -y git)

# Install node and npm if not present
RUN command -v node >/dev/null 2>&1 || \
    (apt-get update && apt-get install -y nodejs npm) || \
    (apk add --no-cache nodejs npm) || \
    (yum install -y nodejs npm)

# Install Claude Code
RUN npm install -g @anthropic-ai/claude-code

# Copy giverny binary from innie image
COPY --from=innie /output/giverny /usr/local/bin/giverny
RUN chmod +x /usr/local/bin/giverny

# Set working directory
WORKDIR /app
`

type MainDockerfileData struct {
	BaseImage string
}

// BuildMainImage builds the giverny-main Docker image.
// It creates a temporary directory, generates the Dockerfile, builds the image,
// optionally streams output to stdout based on showOutput, and cleans up.
func BuildMainImage(baseImage string, showOutput bool) error {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "giverny-main-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Generate Dockerfile
	dockerfilePath := filepath.Join(tmpDir, "Dockerfile.main")
	data := MainDockerfileData{
		BaseImage: baseImage,
	}
	if err := generateDockerfile(dockerfilePath, dockerfileMainTemplate, data); err != nil {
		return fmt.Errorf("failed to generate Dockerfile: %w", err)
	}

	// Build Docker image
	fmt.Println("Building giverny-main image...")
	cmd := exec.Command("docker", "build",
		"-f", dockerfilePath,
		"-t", "giverny-main:latest",
		tmpDir,
	)

	// Conditionally stream output to stdout/stderr
	if showOutput {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker build failed: %w", err)
	}

	fmt.Println("Successfully built giverny-main:latest")
	return nil
}
