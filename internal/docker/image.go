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

// EmbeddedSource holds the embedded source code for building the image.
// This is set by the main package which has access to the module root.
var EmbeddedSource embed.FS

// DiffreviewerVersion specifies the version of diffreviewer to install
const DiffreviewerVersion = "v0.2.1"

const dockerfileDepsTemplate = `# Multi-stage build for Giverny dependencies
# This builds the giverny binary, diffreviewer, and beads

# Stage 1: Build giverny binary
FROM golang:alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

# Set working directory
WORKDIR /build

# Copy source code
COPY . .

# Build the binary
RUN mkdir -p /output && make build && ln ./bin/giverny /output/giverny

# Verify the binary was created
RUN test -f /output/giverny && chmod +x /output/giverny

# Stage 2: Build diffreviewer
FROM golang:alpine AS diffreviewer-builder

# Install build dependencies
RUN apk add --no-cache git curl nodejs npm make

# Set working directory
WORKDIR /build

# Download and extract diffreviewer source
RUN curl -L https://api.github.com/repos/hughe/diffreviewer/tarball/{{.DiffreviewerVersion}} -o diffreviewer.tar.gz && \
    mkdir -p diffreviewer && \
    tar -xzf diffreviewer.tar.gz -C diffreviewer --strip-components=1

# Build diffreviewer using Makefile
WORKDIR /build/diffreviewer
RUN make && \
    mkdir -p /output && \
    ln bin/diffreviewer /output/diffreviewer

# Verify the binary was created
RUN test -f /output/diffreviewer

# Stage 3: Build beads
FROM golang:alpine AS beads-builder

# Install beads
RUN go install github.com/steveyegge/beads/cmd/bd@latest && \
    mkdir -p /output && \
    ln $(go env GOPATH)/bin/bd /output/bd

# Verify the binary was created
RUN test -f /output/bd

# Stage 4: Collect all binaries in a single stage
FROM alpine:latest

# Copy all binaries
COPY --from=builder /output/giverny /output/giverny
COPY --from=diffreviewer-builder /output/diffreviewer /output/diffreviewer
COPY --from=beads-builder /output/bd /output/bd

# Verify all binaries are present
RUN test -f /output/giverny && \
    test -f /output/diffreviewer && \
    test -f /output/bd
`

const dockerfileMainTemplate = `# Final Giverny image with dependencies from giverny-deps
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

# Copy binaries from giverny-deps image
COPY --from=giverny-deps:latest /output/giverny /usr/local/bin/giverny
COPY --from=giverny-deps:latest /output/diffreviewer /usr/local/bin/diffreviewer
COPY --from=giverny-deps:latest /output/bd /usr/local/bin/bd

# Create bd wrapper script in /usr/local/sbin (earlier in PATH)
COPY <<'EOF' /usr/local/sbin/bd
#!/bin/bash
# Wrapper script for bd that automatically adds --sandbox and --no-db flag
# This ensures bd runs in sandbox mode by default in the Giverny environment

# Check if 'sync' command or --db flag is present in arguments
for arg in "$@"; do
    if [[ "$arg" == "sync" ]]; then
        echo "Don't sync in containers" >&2
        exit 1
    fi
    if [[ "$arg" == "--db" ]]; then
        echo "Error: --db flag is not allowed in this environment" >&2
        exit 1
    fi
done

# Call the real bd with --sandbox --no-db prepended to arguments
exec /usr/local/bin/bd --sandbox --no-db "$@"
EOF
RUN chmod +x /usr/local/sbin/bd

# Set working directory
WORKDIR /app
`

type DockerfileData struct {
	BaseImage           string
	DiffreviewerVersion string
}

// BuildImage builds the giverny Docker images using two separate Dockerfiles.
// First it builds giverny-deps with all the dependencies (giverny binary, diffreviewer, beads).
// Then it builds giverny-main which uses the deps image and adds the base image components.
// It creates a temporary directory, extracts embedded source code,
// generates both Dockerfiles, builds both images, optionally streams output
// to stdout based on showOutput, and cleans up.
func BuildImage(baseImage string, showOutput bool, debug bool) error {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "giverny-build-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Extract embedded source code to temp directory
	if err := extractEmbeddedSource(tmpDir); err != nil {
		return fmt.Errorf("failed to extract embedded source: %w", err)
	}

	// Build giverny-deps image first
	if debug {
		fmt.Println("Building giverny-deps image...")
	}

	// Generate Dockerfile.deps
	dockerfileDepsPath := filepath.Join(tmpDir, "Dockerfile.deps")
	depsData := DockerfileData{
		BaseImage:           baseImage,
		DiffreviewerVersion: DiffreviewerVersion,
	}
	if err := generateDockerfile(dockerfileDepsPath, dockerfileDepsTemplate, depsData); err != nil {
		return fmt.Errorf("failed to generate Dockerfile.deps: %w", err)
	}

	// Build giverny-deps image
	depsBuildCmd := exec.Command("docker", "build",
		"-f", dockerfileDepsPath,
		"-t", "giverny-deps:latest",
		tmpDir,
	)

	// Conditionally stream output to stdout/stderr
	if showOutput {
		depsBuildCmd.Stdout = os.Stdout
		depsBuildCmd.Stderr = os.Stderr
	}

	if err := depsBuildCmd.Run(); err != nil {
		return fmt.Errorf("docker build failed for giverny-deps: %w", err)
	}

	if debug {
		fmt.Println("Successfully built giverny-deps:latest")
	}

	// Build giverny-main image
	if debug {
		fmt.Println("Building giverny-main image...")
	}

	// Generate Dockerfile.main
	dockerfileMainPath := filepath.Join(tmpDir, "Dockerfile.main")
	mainData := DockerfileData{
		BaseImage:           baseImage,
		DiffreviewerVersion: DiffreviewerVersion,
	}
	if err := generateDockerfile(dockerfileMainPath, dockerfileMainTemplate, mainData); err != nil {
		return fmt.Errorf("failed to generate Dockerfile.main: %w", err)
	}

	// Build giverny-main image
	mainBuildCmd := exec.Command("docker", "build",
		"-f", dockerfileMainPath,
		"-t", "giverny-main:latest",
		tmpDir,
	)

	// Conditionally stream output to stdout/stderr
	if showOutput {
		mainBuildCmd.Stdout = os.Stdout
		mainBuildCmd.Stderr = os.Stderr
	}

	if err := mainBuildCmd.Run(); err != nil {
		return fmt.Errorf("docker build failed for giverny-main: %w", err)
	}

	if debug {
		fmt.Println("Successfully built giverny-main:latest")
	}
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
