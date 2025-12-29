#!/bin/bash
# Teardown test environment for giverny tests

set -e

if [ -z "$GIV_TEST_ENV_DIR" ]; then
    echo "Error: GIV_TEST_ENV_DIR not set"
    exit 1
fi

if [ ! -d "$GIV_TEST_ENV_DIR" ]; then
    echo "Warning: Test environment directory does not exist: $GIV_TEST_ENV_DIR"
    exit 0
fi

echo "Tearing down test environment: $GIV_TEST_ENV_DIR"
rm -rf "$GIV_TEST_ENV_DIR"
echo "Test environment cleaned up"
