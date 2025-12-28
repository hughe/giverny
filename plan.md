# Giverny Implementation Plan

## Phase 1: Project Setup and CLI Foundation

### 1.1 Initialize Go Project
- Create `go.mod` with module name `giverny`
- Set up directory structure:
  ```
  giverny/
  ├── cmd/
  │   └── giverny/
  │       └── main.go
  ├── internal/
  │   ├── outie/
  │   ├── innie/
  │   ├── docker/
  │   └── git/
  ├── go.mod
  └── go.sum
  ```

### 1.2 CLI Argument Parsing
- Parse command line arguments:
  - `TASK-ID` (required positional arg)
  - `PROMPT` (optional positional arg, defaults to "Please work on TASK-ID.")
  - `--base-image BASE-IMAGE` (default: `giverny:latest`)
  - `--docker-args DOCKER-ARGS`
  - `--innie` (flag to indicate running inside container)
  - `--git-server-port XXXX`
- Route to Outie or Innie based on `--innie` flag

## Phase 2: Outie - Pre-Container Setup

### 2.1 Environment Validation
- Check that `CLAUDE_CODE_OAUTH_TOKEN` is set
- Exit with clear error message if not

### 2.2 Git Branch Creation
- Create branch `giverny/TASK-ID` at current HEAD
- Do not check it out
- Handle error if branch already exists

### 2.3 Dockerfile.innie Generation
- Create temporary directory
- Generate Dockerfile that:
  - Uses a Go build image
  - Copies giverny source code
  - Builds the binary
  - Hard links to `/output/giverny`
- Run `docker build -f Dockerfile.innie -t giverny-innie .`
- Stream build output to terminal
- Remove temporary directory

### 2.4 Dockerfile.main Generation
- Create temporary directory
- Generate Dockerfile that:
  - `FROM giverny-innie:latest AS innie`
  - `FROM BASE-IMAGE`
  - Ensures git is installed
  - Ensures node and npm are installed
  - Runs `npm install -g @anthropic-ai/claude-code`
  - Copies `/output/giverny` from innie to `/usr/local/bin/giverny`
- Run `docker build -t giverny-main .`
- Stream build output to terminal
- Remove temporary directory

### 2.5 Git Server Startup
- Start `git daemon --base-path=. --export-all --enable=receive-pack --reuseaddr --listen=127.0.0.1 --port XXXX`
- Try random ports in range 2001-9999
- Retry on port conflict (exit code 128)
- Store process handle for later cleanup

### 2.6 Container Startup
- Build docker run command:
  ```
  docker run -it --env CLAUDE_CODE_OAUTH_TOKEN DOCKER-ARGS giverny-main:latest \
    /usr/local/bin/giverny --innie --git-server-port XXXX TASK-ID [PROMPT]
  ```
- Run container and wait for exit
- Capture exit code

### 2.7 Post-Container Cleanup
- If exit code 0:
  - Remove the container
  - Print success message with merge/delete commands
- If exit code non-zero:
  - Keep container for recovery
  - Print error to stderr
  - Exit with error code
- Terminate git server child process

## Phase 3: Innie - Container Operations

### 3.1 Git Setup
- Create `/git` directory
- Clone from `http://host.docker.internal:XXXX/` with `--no-checkout`
- Handle clone failure with useful error message

### 3.2 Workspace Setup
- Create `/app` directory
- Checkout `giverny/TASK-ID` branch into `/app`
- Handle checkout failure with useful error message
- Create `giverny/START` branch at current tip (as a label)

### 3.3 Claude Code Execution
- Change to `/app` directory
- Run `claude --dangerously-skip-permissions PROMPT`
- Wait for Claude to exit

### 3.4 Post-Claude Menu Loop
- Check if working directory is dirty (`git status`)
- Display appropriate menu:
  - If dirty: show [c], [s], [r], [x] options with git status
  - If clean: show [s], [r], [x] options
- Handle user input:
  - `c`: Run Claude non-interactively to commit, then re-check and show menu
  - `s`: Start interactive bash, then re-check and show menu
  - `r`: Restart Claude interactively, return to menu on exit
  - `x`: If clean, proceed to push; if dirty, warn and show menu again

### 3.5 Push and Exit
- Push `giverny/TASK-ID` branch to git server
- Exit with code 0

## Phase 4: Polish and Error Handling

### 4.1 Error Messages
- Ensure all error paths have clear, actionable messages
- Include context (what failed, why, what to do)

### 4.2 Signal Handling
- Handle SIGINT/SIGTERM gracefully in Outie
- Clean up git server on interrupt

### 4.3 Testing
- Manual testing of full workflow
- Test error scenarios (missing token, git failures, etc.)

## Implementation Order

1. Phase 1.1, 1.2 - Get basic CLI working
2. Phase 2.1 - Environment validation
3. Phase 2.2 - Git branch creation
4. Phase 2.5 - Git server (test independently)
5. Phase 2.3 - Dockerfile.innie
6. Phase 2.4 - Dockerfile.main
7. Phase 2.6 - Container startup
8. Phase 3.1, 3.2 - Innie git/workspace setup
9. Phase 3.3 - Claude execution
10. Phase 3.4 - Post-Claude menu
11. Phase 3.5 - Push and exit
12. Phase 2.7 - Post-container cleanup
13. Phase 4 - Polish

## Dependencies

- Go 1.21+
- Docker
- git

## Notes

- Use `os/exec` for running external commands
- Use `text/template` for Dockerfile generation
- Consider using `cobra` or `flag` for CLI parsing
- Stream docker build output using `cmd.Stdout = os.Stdout`
