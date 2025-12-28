package docker

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"
)

const dockerfileInnieTemplate = `# Build stage for giverny binary
FROM golang:alpine AS builder

# Install build dependencies
RUN apk add --no-cache git

# Set working directory
WORKDIR /build

# Copy source code
COPY . .

# Build the binary
RUN go build -o /output/giverny ./cmd/giverny

# Verify the binary was created
RUN test -f /output/giverny && chmod +x /output/giverny
`

// BuildInnieImage builds the giverny-innie Docker image.
// It creates a temporary directory, generates the Dockerfile, builds the image,
// streams output to stdout, and cleans up.
func BuildInnieImage() error {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "giverny-innie-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Get current working directory (should be project root)
	projectRoot, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Generate Dockerfile
	dockerfilePath := filepath.Join(tmpDir, "Dockerfile.innie")
	if err := generateDockerfile(dockerfilePath, dockerfileInnieTemplate, nil); err != nil {
		return fmt.Errorf("failed to generate Dockerfile: %w", err)
	}

	// Build Docker image
	fmt.Println("Building giverny-innie image...")
	cmd := exec.Command("docker", "build",
		"--no-cache",
		"-f", dockerfilePath,
		"-t", "giverny-innie:latest",
		projectRoot,
	)

	// Stream output to stdout/stderr
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker build failed: %w", err)
	}

	fmt.Println("Successfully built giverny-innie:latest")
	return nil
}

// generateDockerfile creates a Dockerfile from a template
func generateDockerfile(path string, templateStr string, data interface{}) error {
	tmpl, err := template.New("dockerfile").Parse(templateStr)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	if err := tmpl.Execute(file, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return nil
}
