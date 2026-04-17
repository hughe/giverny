package docker

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
	"time"
)

// MainImageName returns the tag for the giverny-main image derived from the
// given base image. We embed the base image name so that runs against
// different base images don't collide on a single shared "giverny-main:latest"
// tag. e.g. "alpine:latest" -> "alpine-giverny-main:latest",
// "gcr.io/foo/bar:dev" -> "gcr.io-foo-bar-giverny-main:dev".
func MainImageName(baseImage string) string {
	name, tag := baseImage, "latest"
	// Split on the last colon to separate tag (avoid splitting registry ports
	// like "registry:5000/foo"; if there's a slash after the colon it's a
	// port, not a tag).
	if i := strings.LastIndex(baseImage, ":"); i != -1 && !strings.Contains(baseImage[i:], "/") {
		name, tag = baseImage[:i], baseImage[i+1:]
	}
	name = strings.ReplaceAll(name, "/", "-")
	return fmt.Sprintf("%s-giverny-main:%s", name, tag)
}

// EmbeddedSource holds the embedded source code for building the image.
// This is set by the main package which has access to the module root.
var EmbeddedSource embed.FS

// DiffreviewerVersion specifies the version of diffreviewer to install
const DiffreviewerVersion = "v0.2.3"

// BeadsRustVersion specifies the version of beads_rust to install
const BeadsRustVersion = "v0.1.14"

// ImageMaxAge is the maximum age of a Docker image before it should be rebuilt
const ImageMaxAge = 24 * time.Hour

const dockerfileDepsTemplate = `# Multi-stage build for Giverny dependencies
# This builds the giverny binary, diffreviewer, and beads_rust

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

# Stage 3: Build beads_rust (br)
FROM rust:alpine AS beads-builder

# Install build dependencies
RUN apk add --no-cache git musl-dev

# Install beads_rust
RUN cargo install --git https://github.com/Dicklesworthstone/beads_rust.git --tag {{.BeadsRustVersion}} && \
    mkdir -p /output && \
    cp $(which br) /output/br

# Verify the binary was created
RUN test -f /output/br

# Stage 4: Collect all binaries in a single stage
FROM alpine:latest

# Copy all binaries
COPY --from=builder /output/giverny /output/giverny
COPY --from=diffreviewer-builder /output/diffreviewer /output/diffreviewer
COPY --from=beads-builder /output/br /output/br

# Verify all binaries are present
RUN test -f /output/giverny && \
    test -f /output/diffreviewer && \
    test -f /output/br
`

const dockerfileMainTemplate = `# Final Giverny image with dependencies from giverny-deps
FROM {{.BaseImage}}

# Install git and curl if not present
RUN command -v git >/dev/null 2>&1 || \
    (apt-get update && apt-get install -y git) || \
    (apk add --no-cache git) || \
    (yum install -y git)

RUN command -v curl >/dev/null 2>&1 || \
    (apt-get update && apt-get install -y curl) || \
    (apk add --no-cache curl) || \
    (yum install -y curl)

# Install ripgrep if not present
RUN command -v rg >/dev/null 2>&1 || \
    (apt-get update && apt-get install -y ripgrep) || \
    (apk add --no-cache ripgrep) || \
    (yum install -y ripgrep) || \
    echo "Warning: ripgrep not available in package manager"

# Install node and npm if not present (still needed for Amp)
RUN command -v node >/dev/null 2>&1 || \
    (apt-get update && apt-get install -y nodejs npm) || \
    (apk add --no-cache nodejs npm) || \
    (yum install -y nodejs npm)

# Install Claude Code using official installer.
# The installer drops the binary in ~/.local/bin, which isn't in $PATH when
# giverny --innie execs claude, so symlink it into /usr/local/bin.
RUN curl -fsSL https://claude.ai/install.sh | bash && \
    ln -s /root/.local/bin/claude /usr/local/bin/claude && \
    claude --version

# Install Amp
RUN npm install -g @sourcegraph/amp@latest

# Copy binaries from giverny-deps image
COPY --from=giverny-deps:latest /output/giverny /usr/local/bin/giverny
COPY --from=giverny-deps:latest /output/br /usr/local/bin/br

# Install diffreviewer: real binary in /usr/local/lib/giverny, wrapper in PATH
RUN mkdir -p /usr/local/lib/giverny
COPY --from=giverny-deps:latest /output/diffreviewer /usr/local/lib/giverny/diffreviewer
COPY scripts/diffreviewer-wrapper.sh /usr/local/bin/diffreviewer
RUN chmod +x /usr/local/bin/diffreviewer

# Set working directory
WORKDIR /app
`

type DockerfileData struct {
	BaseImage           string
	DiffreviewerVersion string
	BeadsRustVersion    string
}

// getImageAge returns the age of a Docker image, or an error if the image doesn't exist
func getImageAge(imageName string) (time.Duration, error) {
	cmd := exec.Command("docker", "inspect", "--format", "{{json .Created}}", imageName)
	output, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("image not found: %w", err)
	}

	var createdStr string
	if err := json.Unmarshal(output, &createdStr); err != nil {
		return 0, fmt.Errorf("failed to parse image creation time: %w", err)
	}

	created, err := time.Parse(time.RFC3339Nano, createdStr)
	if err != nil {
		return 0, fmt.Errorf("failed to parse timestamp: %w", err)
	}

	return time.Since(created), nil
}

// BuildImage builds the giverny Docker images using two separate Dockerfiles.
// First it builds giverny-deps with all the dependencies (giverny binary, diffreviewer, beads_rust).
// Then it builds giverny-main which uses the deps image and adds the base image components.
// It creates a temporary directory, extracts embedded source code,
// generates both Dockerfiles, builds both images, optionally streams output
// to stdout based on showOutput, and cleans up.
//
// If giverny-main:latest exists and is less than 24 hours old, the build is skipped
// unless forceRebuild is true.
func BuildImage(baseImage string, showOutput bool, forceRebuild bool, debug bool) error {
	mainImage := MainImageName(baseImage)
	// Check if giverny-main image exists and is fresh enough
	if !forceRebuild {
		if age, err := getImageAge(mainImage); err == nil {
			if age < ImageMaxAge {
				if debug {
					fmt.Printf("Using existing %s image (age: %s)\n", mainImage, age.Round(time.Minute))
				}
				return nil
			}
			if debug {
				fmt.Printf("Rebuilding %s image (age: %s, max: %s)\n", mainImage, age.Round(time.Minute), ImageMaxAge)
			}
		} else if debug {
			fmt.Printf("Building %s image (no existing image found)\n", mainImage)
		}
	} else if debug {
		fmt.Printf("Force rebuilding %s image\n", mainImage)
	}
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
		BeadsRustVersion:    BeadsRustVersion,
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
		BeadsRustVersion:    BeadsRustVersion,
	}
	if err := generateDockerfile(dockerfileMainPath, dockerfileMainTemplate, mainData); err != nil {
		return fmt.Errorf("failed to generate Dockerfile.main: %w", err)
	}

	// Build giverny-main image
	mainBuildCmd := exec.Command("docker", "build",
		"-f", dockerfileMainPath,
		"-t", mainImage,
		tmpDir,
	)

	// Conditionally stream output to stdout/stderr
	if showOutput {
		mainBuildCmd.Stdout = os.Stdout
		mainBuildCmd.Stderr = os.Stderr
	}

	if err := mainBuildCmd.Run(); err != nil {
		return fmt.Errorf("docker build failed for %s: %w", mainImage, err)
	}

	if debug {
		fmt.Printf("Successfully built %s\n", mainImage)
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
