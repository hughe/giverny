package docker

import (
	"embed"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateDockerfile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "giverny-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dockerfilePath := filepath.Join(tmpDir, "Dockerfile")
	data := DockerfileData{
		BaseImage:           "ubuntu:22.04",
		DiffreviewerVersion: "v0.1.1",
	}

	err = generateDockerfile(dockerfilePath, dockerfileTemplate, data)
	if err != nil {
		t.Fatalf("generateDockerfile failed: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(dockerfilePath); os.IsNotExist(err) {
		t.Fatal("Dockerfile was not created")
	}

	// Read and verify content contains base image
	content, err := os.ReadFile(dockerfilePath)
	if err != nil {
		t.Fatalf("failed to read Dockerfile: %v", err)
	}

	contentStr := string(content)
	if len(contentStr) == 0 {
		t.Fatal("Dockerfile is empty")
	}

	// Check for expected content
	expectedStrings := []string{
		"FROM golang:alpine AS builder",
		"FROM golang:alpine AS diffreviewer-builder",
		"FROM golang:alpine AS beads-builder",
		"apk add --no-cache git curl nodejs npm make",
		"RUN make",
		"go install github.com/steveyegge/beads/cmd/bd@latest",
		"FROM ubuntu:22.04",
		"COPY --from=builder",
		"COPY --from=diffreviewer-builder",
		"COPY --from=beads-builder",
		"v0.1.1",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(contentStr, expected) {
			t.Errorf("Dockerfile missing expected content: %s", expected)
		}
	}
}

func TestGenerateDockerfileWithInvalidPath(t *testing.T) {
	invalidPath := "/nonexistent/directory/Dockerfile"
	data := DockerfileData{
		BaseImage:           "ubuntu:22.04",
		DiffreviewerVersion: "v0.1.1",
	}
	err := generateDockerfile(invalidPath, dockerfileTemplate, data)
	if err == nil {
		t.Fatal("expected error for invalid path, got nil")
	}
}

func TestGenerateDockerfileWithDifferentBaseImage(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "giverny-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dockerfilePath := filepath.Join(tmpDir, "Dockerfile")
	data := DockerfileData{
		BaseImage:           "alpine:latest",
		DiffreviewerVersion: "v0.1.1",
	}

	err = generateDockerfile(dockerfilePath, dockerfileTemplate, data)
	if err != nil {
		t.Fatalf("generateDockerfile failed: %v", err)
	}

	// Read and verify content contains the custom base image
	content, err := os.ReadFile(dockerfilePath)
	if err != nil {
		t.Fatalf("failed to read Dockerfile: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "FROM alpine:latest") {
		t.Error("Dockerfile does not contain custom base image")
	}
}

//go:embed image.go image_test.go
var testEmbedFS embed.FS

func TestBuildImage_IntegrationTest(t *testing.T) {
	// Skip unless INTEGRATION_TEST=1
	if os.Getenv("INTEGRATION_TEST") != "1" {
		t.Skip("Skipping integration test. Set INTEGRATION_TEST=1 to run.")
	}

	// Set up embedded source (normally done by main package)
	EmbeddedSource = testEmbedFS

	// Build the image
	err := BuildImage("alpine:latest", true)
	if err != nil {
		t.Fatalf("BuildImage failed: %v", err)
	}

	// Verify the image was created
	cmd := exec.Command("docker", "image", "inspect", "giverny-main:latest")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Image giverny-main:latest was not created: %v", err)
	}

	// Verify diffreviewer binary exists in the image
	cmd = exec.Command("docker", "run", "--rm", "giverny-main:latest", "which", "diffreviewer")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("diffreviewer not found in image: %v, output: %s", err, output)
	}

	if !strings.Contains(string(output), "/usr/local/bin/diffreviewer") {
		t.Errorf("diffreviewer not installed in expected location, got: %s", output)
	}

	// Verify beads binary exists in the image
	cmd = exec.Command("docker", "run", "--rm", "giverny-main:latest", "which", "bd")
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("bd not found in image: %v, output: %s", err, output)
	}

	if !strings.Contains(string(output), "/usr/local/bin/bd") {
		t.Errorf("bd not installed in expected location, got: %s", output)
	}

	// Clean up - remove the test image
	exec.Command("docker", "rmi", "giverny-main:latest").Run()
}
