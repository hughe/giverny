# Giverny - A House for Claude Code

Giverny is a containerized system for running [Claude Code](https://claude.ai/code) safely. It creates isolated Docker environments where Claude Code can work on tasks without affecting the host system.

Inspired by [Sketch](https://github.com/boldsoftware/sketch).

## What is Giverny?

Giverny provides a sandboxed environment for Claude Code by:

- Creating isolated Docker containers for each task
- Using git branches to isolate changes
- Running a local git daemon for communication between host and container
- Allowing Claude Code to work safely without affecting your host system

## Running

Basic usage:

```bash
giverny TASK-ID [PROMPT]
```

Where:
- `TASK-ID` is the id of a task to perform. It might be an identifier from an issue tracker like [beads](https://github.com/jlewallen/beads) (e.g., `giv-0f9`), or it could be an identifier like `create-hello-world`.
- `PROMPT` is an optional string prompt telling Claude Code what to do. If not specified, it defaults to "Please work on TASK-ID."

### Options

- `--base-image BASE-IMAGE`: Docker base image (default: `giverny:latest`)
- `--docker-args DOCKER-ARGS`: Additional docker run arguments
- `--debug`: Enable debug output
- `--show-build-output`: Show docker build output
- `--existing-branch`: Use existing branch instead of creating a new one
- `--version`: Show version information

### Examples

```bash
# Run Claude Code on a task with a specific prompt
giverny my-feature "Implement user authentication"

# Use a different base image
giverny --base-image ubuntu:22.04 my-feature "Fix the login bug"

# Enable debug output
giverny --debug my-feature "Add unit tests"
```

## Architecture

The system consists of two components that communicate via git:

- **Outie**: Runs on the host, manages Docker containers, runs a git daemon server
- **Innie**: Runs inside the container, clones the repo from Outie, runs Claude Code

### How It Works

1. Outie creates a branch `giverny/TASK-ID` and starts a git daemon on a random port (2001-9999)
2. Outie builds two Docker images:
   - `giverny-innie`: Contains the giverny binary
   - `giverny-main`: Based on user-specified base image, includes git, node, npm, claude-code, and giverny binary
3. Innie clones the repo into `/git`, checks out the branch into `/app`
4. Innie runs `claude --dangerously-skip-permissions PROMPT`
5. After Claude exits, Innie prompts the user to commit changes, restart Claude, or exit
6. On clean exit, Innie pushes to Outie's git server

## Prerequisites

- Docker installed and running
- Go 1.21 or later
- Git
- `CLAUDE_CODE_OAUTH_TOKEN` environment variable set (obtain from [claude.ai/code](https://claude.ai/code))

## Building

Build the giverny binary:

```bash
make build
```

This creates the binary at `bin/giverny`.

To install to `$GOPATH/bin`:

```bash
make install
```

### Other Build Targets

```bash
make clean           # Remove build artifacts
make fmt             # Format code
make lint            # Run linter
make image           # Build Docker image for Giverny to work on Giverny
```

## Testing

Run tests with automatic environment setup/teardown:

```bash
make test              # Run all tests
make test-binary       # Test the giverny binary
make integration-test  # Run integration tests
```

**Note:** Always use `make test` instead of `go test` directly. The Makefile target sets up an isolated test environment in `/tmp/giverny-test-env-*`.

## Development

The project structure:

```
giverny/
├── cmd/giverny/      # Main entry point
├── docker/           # Dockerfile for giverny-builder image
├── scripts/          # Test environment setup/teardown scripts
├── Makefile          # Build and test targets
└── README.md         # This file
```

## License

See LICENSE file for details.
