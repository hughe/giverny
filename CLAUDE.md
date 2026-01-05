# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Giverny is a containerized system for running Claude Code safely. It creates isolated Docker environments where Claude Code can work on tasks without affecting the host system. Inspired by [Sketch](https://github.com/boldsoftware/sketch).

## Architecture

The system has two components that communicate via git:

- **Outie**: Runs on the host, manages Docker containers, runs a git daemon server
- **Innie**: Runs inside the container, clones repo from Outie, runs Claude Code

### Workflow

1. Outie creates branch `giverny/TASK-ID` and starts git daemon on a random port (2001-9999)
2. Outie builds two Docker images:
   - `giverny-innie`: Contains the giverny binary
   - `giverny-main`: Based on user-specified base image, includes git, node, npm, claude-code, and giverny binary
3. Innie clones repo into `/git`, checks out branch into `/app`
4. Innie runs `claude --dangerously-skip-permissions PROMPT`
5. After Claude exits, Innie prompts user to commit changes, restart Claude, or exit
6. On clean exit, Innie pushes to Outie's git server

### Key Docker Connectivity

- Container connects to host git daemon via Docker's special hostname for the host
- `CLAUDE_CODE_OAUTH_TOKEN` environment variable must be set and is passed to container

## Tech Stack

- Backend: Go
- Frontend: HTML, TypeScript, React

## CLI Usage

```
giverny TASK-ID [PROMPT]
```

- `--base-image BASE-IMAGE`: Docker base image (default: `giverny:latest`)
- `--docker-args DOCKER-ARGS`: Additional docker run arguments
- `--innie`: Flag indicating running inside container
- `--git-server-port XXXX`: Port for git daemon connection
- `--debug`: Enable debug output
- `--show-build-output`: Show docker build output

## Testing

The project includes several Makefile targets for testing:

```bash
make test        # Run tests with environment setup/teardown in /tmp/
make test-binary # Test the giverny binary
```

**IMPORTANT:** *Always* run tests with `make test`.  *Always* test the binary using `make test-binary`.

The `test` and `test-binary` targets automatically:
1. Set up a test environment in `/tmp/giverny-test-env-*`
2. Initialize a git repository for testing
3. Run the tests or binary
4. Clean up the test environment

Use these targets when you need an isolated test environment.

### Test Package Structure

**IMPORTANT:** Each test package must include a `TestMain()` function that:
- Checks for the `GIV_TEST_ENV_DIR` environment variable
- Changes to that directory if set
- Calls `m.Run()` to execute the tests

Example:
```go
func TestMain(m *testing.M) {
	// Check if GIV_TEST_ENV_DIR is set and change to that directory
	if testEnvDir := os.Getenv("GIV_TEST_ENV_DIR"); testEnvDir != "" {
		if err := os.Chdir(testEnvDir); err != nil {
			panic("failed to change to test environment directory: " + err.Error())
		}
	}

	m.Run()
}
```

## Behavior

Don't say "Perfect" all the time.  Try to be direct and professional.

# IMPORTANT: Task Tracking

This project uses **bd** (beads) for issue tracking.  **Do not** use
 markdown files or the todo list.  Run `bd quickstart` to learn how.

## Quick Reference

```bash
bd ready              # Find available work
bd show <id>          # View issue details
bd update <id> --status in_progress  # Claim work
bd close <id>         # Complete work
bd sync               # Sync with git
```

When you commit some changes.  Please add the IDs of the tasks that were closed to the commit message.  E.g., 

```
Closes: bd-123, bd-abc
```

*Always* make sure that the changes to file in `.beads/` are included in the commit.

## Landing the Plane (Session Completion)

**When ending a work session**, you MUST complete ALL steps below. 

**MANDATORY WORKFLOW:**

1. **File issues for remaining work** - Create issues for anything that needs follow-up
2. **Run quality gates** (if code changed) - Tests, linters, builds
3. **Update issue status** - Close finished work, update in-progress items
4. **Hand off** - Provide context for next session

**DO NOT PUSH** wait for the human to do a code review.  The human will push.


