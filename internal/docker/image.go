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
const DiffreviewerVersion = "v0.1.1"

const dockerfileTemplate = `# Stage 1: Build giverny binary
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

# Stage 4: Final image with dependencies
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

# Copy giverny binary from builder stage
COPY --from=builder /output/giverny /usr/local/bin/giverny

# Copy diffreviewer binary from diffreviewer-builder stage
COPY --from=diffreviewer-builder /output/diffreviewer /usr/local/bin/diffreviewer

# Copy beads binary from beads-builder stage
COPY --from=beads-builder /output/bd /usr/local/bin/bd

# Create bd wrapper script in /usr/local/sbin (earlier in PATH)
COPY <<'EOF' /usr/local/sbin/bd
#!/bin/bash
# Wrapper script for bd that automatically adds --sandbox and --no-db flag
# This ensures bd runs in sandbox mode by default in the Giverny environment

# Check if --db flag is present in arguments
for arg in "$@"; do
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

// BuildImage builds the giverny Docker image using a multistage Dockerfile.
// It creates a temporary directory, extracts embedded source code,
// generates the Dockerfile, builds the image, optionally streams output
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

	// Generate Dockerfile
	dockerfilePath := filepath.Join(tmpDir, "Dockerfile")
	data := DockerfileData{
		BaseImage:           baseImage,
		DiffreviewerVersion: DiffreviewerVersion,
	}
	if err := generateDockerfile(dockerfilePath, dockerfileTemplate, data); err != nil {
		return fmt.Errorf("failed to generate Dockerfile: %w", err)
	}

	// Build Docker image using temp directory as build context
	if debug {
		fmt.Println("Building giverny image...")
	}
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
