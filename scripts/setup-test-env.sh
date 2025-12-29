#!/bin/bash
# Setup test environment for giverny tests

set -e

# Create test environment directory
GIV_TEST_ENV_DIR="${GIV_TEST_ENV_DIR:-/tmp/giverny-test-env-$$}"
export GIV_TEST_ENV_DIR

echo "Setting up test environment in: $GIV_TEST_ENV_DIR"

# Create directory structure
mkdir -p "$GIV_TEST_ENV_DIR"

# Initialize a git repository for testing
cd "$GIV_TEST_ENV_DIR"
git init
git config user.email "test@giverny.test"
git config user.name "Giverny Test"

# Create an initial commit
echo "# Test Repository" > README.md
git add README.md
git commit -m "Initial commit"

echo "Test environment setup complete"
echo "GIV_TEST_ENV_DIR=$GIV_TEST_ENV_DIR"
