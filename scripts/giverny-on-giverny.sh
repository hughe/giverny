#!/bin/bash
# Run giverny to work on the giverny codebase itself.
#
# This script:
# 1. Builds the giverny-builder Docker image (Go + Docker CLI)
# 2. Builds the giverny binary
# 3. Runs giverny with --base-image giverny-builder
#
# Usage: ./scripts/giverny-on-giverny.sh TASK-ID [SLUG] [PROMPT]
#
# All arguments are passed through to giverny.

set -e

# Get the directory where this script lives
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

cd "$PROJECT_ROOT"

# Build the base image for giverny development
echo "Building giverny-builder Docker image..."
make image

# Build the giverny binary
echo "Building giverny binary..."
make build

# Run giverny with the giverny-builder base image
echo "Starting giverny..."
exec ./bin/giverny --base-image giverny-builder "$@"
