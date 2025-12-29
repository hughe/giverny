#!/bin/bash
# Teardown test environment for giverny tests

set -e

if [ -z "$TEST_ENV_DIR" ]; then
    echo "Error: TEST_ENV_DIR not set"
    exit 1
fi

if [ ! -d "$TEST_ENV_DIR" ]; then
    echo "Warning: Test environment directory does not exist: $TEST_ENV_DIR"
    exit 0
fi

echo "Tearing down test environment: $TEST_ENV_DIR"
rm -rf "$TEST_ENV_DIR"
echo "Test environment cleaned up"
