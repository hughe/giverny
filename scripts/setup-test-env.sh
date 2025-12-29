#!/bin/bash
# Setup test environment for giverny tests

set -e

# Create test environment directory
GIV_TEST_ENV_DIR="${GIV_TEST_ENV_DIR:-/tmp/giverny-test-env-$$}"
export GIV_TEST_ENV_DIR

echo "Setting up test environment in: $GIV_TEST_ENV_DIR"

# Save the current directory (should be the project root)
PROJECT_ROOT=$(pwd)

# Create directory structure
mkdir -p "$GIV_TEST_ENV_DIR"

# Copy source code into test environment
# This is necessary because some tests (like TestRunOutie_ValidatesClaudeToken)
# need to build Docker images, which require access to the source code and Dockerfiles.
# By copying the source into the test environment, tests can run in an isolated
# directory while still having access to all necessary files.
echo "Copying source code to test environment..."
rsync -a \
  --exclude='.git' \
  --exclude='build' \
  --exclude='*.test' \
  --exclude='/tmp' \
  "$PROJECT_ROOT/" "$GIV_TEST_ENV_DIR/"

# Initialize a git repository for testing
cd "$GIV_TEST_ENV_DIR"
git init
git config user.email "test@giverny.test"
git config user.name "Giverny Test"

# Add all source files and create initial commit
git add .
git commit -m "Initial commit with source code"

echo "Test environment setup complete"
echo "GIV_TEST_ENV_DIR=$GIV_TEST_ENV_DIR"
