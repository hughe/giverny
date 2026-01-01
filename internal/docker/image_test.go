package docker

import (
	"os"
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
		BaseImage: "ubuntu:22.04",
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
		"FROM ubuntu:22.04",
		"COPY --from=builder",
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
		BaseImage: "ubuntu:22.04",
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
		BaseImage: "alpine:latest",
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
