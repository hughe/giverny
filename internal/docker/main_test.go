package docker

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMain(m *testing.M) {
	// Check if GIV_TEST_ENV_DIR is set and change to that directory
	if testEnvDir := os.Getenv("GIV_TEST_ENV_DIR"); testEnvDir != "" {
		if err := os.Chdir(testEnvDir); err != nil {
			panic("failed to change to test environment directory: " + err.Error())
		}
	}

	m.Run()
}

func TestGenerateDockerfileMain(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "giverny-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dockerfilePath := filepath.Join(tmpDir, "Dockerfile.test")
	data := MainDockerfileData{
		BaseImage: "ubuntu:22.04",
	}

	err = generateDockerfile(dockerfilePath, dockerfileMainTemplate, data)
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
		"FROM giverny-innie:latest AS innie",
		"FROM ubuntu:22.04",
		"RUN command -v git",
		"RUN command -v node",
		"RUN npm install -g @anthropic-ai/claude-code",
		"COPY --from=innie /output/giverny /usr/local/bin/giverny",
		"RUN chmod +x /usr/local/bin/giverny",
		"WORKDIR /app",
	}

	for _, element := range requiredElements {
		if !strings.Contains(contentStr, element) {
			t.Errorf("Dockerfile missing expected element: %s", element)
		}
	}

	// Verify base image substitution worked
	if !strings.Contains(contentStr, "FROM ubuntu:22.04") {
		t.Error("Base image substitution did not work")
	}
}

func TestGenerateDockerfileMainWithDifferentBaseImage(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "giverny-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dockerfilePath := filepath.Join(tmpDir, "Dockerfile.test")
	data := MainDockerfileData{
		BaseImage: "alpine:latest",
	}

	err = generateDockerfile(dockerfilePath, dockerfileMainTemplate, data)
	if err != nil {
		t.Fatalf("generateDockerfile failed: %v", err)
	}

	// Read the generated file
	content, err := os.ReadFile(dockerfilePath)
	if err != nil {
		t.Fatalf("failed to read generated Dockerfile: %v", err)
	}

	contentStr := string(content)

	if !strings.Contains(contentStr, "FROM alpine:latest") {
		t.Error("Expected alpine:latest as base image")
	}
}
