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

	dockerfilePath := filepath.Join(tmpDir, "Dockerfile.test")

	err = generateDockerfile(dockerfilePath, dockerfileInnieTemplate, nil)
	if err != nil {
		t.Fatalf("generateDockerfile failed: %v", err)
	}

	// Read the generated file
	content, err := os.ReadFile(dockerfilePath)
	if err != nil {
		t.Fatalf("failed to read generated Dockerfile: %v", err)
	}

	contentStr := string(content)

	// Verify key elements are present
	requiredElements := []string{
		"FROM golang:alpine AS builder",
		"WORKDIR /build",
		"COPY . .",
		"RUN go build -o /output/giverny ./cmd/giverny",
		"RUN test -f /output/giverny",
	}

	for _, element := range requiredElements {
		if !strings.Contains(contentStr, element) {
			t.Errorf("Dockerfile missing expected element: %s", element)
		}
	}
}

func TestGenerateDockerfileWithInvalidPath(t *testing.T) {
	invalidPath := "/nonexistent/directory/Dockerfile"
	err := generateDockerfile(invalidPath, dockerfileInnieTemplate, nil)

	if err == nil {
		t.Error("expected error for invalid path, got nil")
	}
}
