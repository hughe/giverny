package docker

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"
)

// EmbeddedSource holds the embedded source code for building the innie image.
// This is set by the main package which has access to the module root.
var EmbeddedSource embed.FS

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
// It creates a temporary directory, extracts embedded source code,
// generates the Dockerfile, builds the image, optionally streams output
// to stdout based on showOutput, and cleans up.
func BuildInnieImage(showOutput bool) error {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "giverny-innie-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Extract embedded source code to temp directory
	if err := extractEmbeddedSource(tmpDir); err != nil {
		return fmt.Errorf("failed to extract embedded source: %w", err)
	}

	// Generate Dockerfile
	dockerfilePath := filepath.Join(tmpDir, "Dockerfile.innie")
	if err := generateDockerfile(dockerfilePath, dockerfileInnieTemplate, nil); err != nil {
		return fmt.Errorf("failed to generate Dockerfile: %w", err)
	}

	// Build Docker image using temp directory as build context
	fmt.Println("Building giverny-innie image...")
	cmd := exec.Command("docker", "build",
		"--no-cache",
		"-f", dockerfilePath,
		"-t", "giverny-innie:latest",
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

	fmt.Println("Successfully built giverny-innie:latest")
	return nil
}

// extractEmbeddedSource extracts all embedded source files to the target directory.
func extractEmbeddedSource(targetDir string) error {
	return fs.WalkDir(EmbeddedSource, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip the root "." directory
		if path == "." {
			return nil
		}

		// Construct target path
		targetPath := filepath.Join(targetDir, path)

		if d.IsDir() {
			// Create directory
			return os.MkdirAll(targetPath, 0755)
		}

		// Read embedded file
		content, err := EmbeddedSource.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read embedded file %s: %w", path, err)
		}

		// Write to target
		if err := os.WriteFile(targetPath, content, 0644); err != nil {
			return fmt.Errorf("failed to write file %s: %w", targetPath, err)
		}

		return nil
	})
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
