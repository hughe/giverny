# Giverny Desgin

## Introduction

Giverny is a containerized system for fearlessly running Claude Code
in.  It's a poor imitation of
[sketch](https://github.com/boldsoftware/sketch "Sketch is an agentic
coding tool"), but with Claude Code or possibly other CLI Coding
Agents.

# User Story

* Start Giverny from the command line, like:

```Giverny TASK-ID [PROMPT]```

* `TASK-ID` is the id of a task to perform.  It might be a identifier
  of a task in an issue tracker like
  [beads](https://github.com/steveyegge/beads "Beads is a distributed,
  git-backed graph issue tracker for AI agents.") (e.g., `giv-0f9`), or
  it could be a identifier like `create-hello-world`.
  
* `PROMPT` is a string prompt telling Claude Code what to do.  If it
  is not present then it should default to "Please work on TASK-ID."
  
* Giverny creates a git branch called `giverny/TASK-ID` to work in, it
  does not check it out.

* Giverny creates a Dockerfile in a temporary directory.  We'll call this file `Dockerfile.innie`.
  
  + `Dockerfile.innie` installs the tools necessary to build `giverny`.
  
  + `Dockerfile.innie` copies the source code for `giverny`.
  
  + `Dockerfile.innie` builds the `giverny` binary.

  + It hard links the `giverny` binary to `/output/giverny`.
  
  + Giverny runs `docker build -f Dockerfile.innie -t
    giverny-innie` in the temporary directory.
	
  + The output of the docker build is visible in the terminal.
  
  + The temporary directory is removed.
  
* Giverny starts a Dockerfile in a temporary directory.  We'll call this file `Dockerfile.main`.

  + The docker file starts with the `FROM giverny-innie:latest AS innie`.
  
  + Then `FROM BASE-IMAGE`.
  
  + The BASE-IMAGE is specified on the Giverny command line using
    `--base-image BASE-IMAGE`.  It defaults to `giverny:latest`.
	 
  + `Dockerfile.main` ensures that `git` is installed. Installing them if necessary.
  
  + `Dockerfile.main` ensures that `node` and `npm` are installed.  Installing them if necessary.
  
  + `Dockerfile.main` installs claude code. `RUN npm install -g @anthropic-ai/claude-code`
  
  + `Dockerfile.main` copies the `giverny` binary from `/output/` in `innie` into `/usr/local/bin`.
  
  + The output of the docker build is visible in the terminal.
  
  + Giverny builds `Dockerfile.main` and tags it `giverny-main`.
  
* Giverny starts a git server in the root of the checkout.
   `git daemon --base-path=. --export-all --reuseaddr --verbose --listen=127.0.0.1 --port XXXX`
   
   + `XXXX` is a random port in the range 2001-9999. If `git` exits
     with a code `128` and the message `fatal: unable to allocate any
     listen sockets on port XXXX` then try another port.
	 
     - See if there is a list of `git` error codes and if we can
       remove the bit above where we check for the message.
	   
* The `giverny-main:latest` container is started. With
  `/usr/local/bin/giverny` as the entry point. From now on we refer to
  the instance of Giverny running in the container as Innie.  The
  instance running on the outside is referred to as the Outie.
  
  
  + We should be able to specify arguments for creating the container
    using `--docker-args DOCKER-ARGS`.

  + The port number for git `XXXX` is paseed to Innie with a flag. `--git-server-port`.
  
  + Innie has a command line flag so it knows it is the Innie.  Maybe `--Innie`
  
  + The task id is passed to the Innie. `TASK-ID` 
  
  + The prompt is prompt to the Innie. `PROMPT`.
  
  + Innie command line `INNIE-COMMAND-LINE` looks like `/usr/local/bin/giverny --innie --git-server-port XXXX TASK-ID [PROMPT]`.
  
  + The docker command will look something like `docker run -it -env CLAUDE_CODE_OAUTH_TOKEN  DOCKKER-ARGS CONTAINER-ID INNIE-COMMAND-LINE`
  
  + Outie should check that `CLAUDE_CODE_OAUTH_TOKEN` is configured and exit with an error if it is not.
  
* Innie creates a directory for git `/git`.

* Innie clones the git repo from outside.  `git clone --no-checkout http://OUTER-HOST:XXXX/ /git`.

  + Where outer host is the name of the host running Outie.  There
    is a special docker hostname for this.

* Innie makes a directory `/app`.

* Innie checks out the `giverny/TASK-ID` branch into `/app`.

* Innie creates, but does not checkout, another branch at the tip of
  `giverny/TASK-ID` that is called `giverny/START`.  This is just to
  serve as a label for where we start work.
  
* Innie starts Claude Code with the prompt.  `claude --dangerously-skip-permissions PROMPT`.

* After the Claude Code process exits, Innie prompts the user
  with these choices.

  1. if the working directory is dirty, ask claude to commit it, non-interactively. 
  2. if the working directory is dirty, ask the user if they would like to commit manually.
  3. would the user like to restart claude.
  4. would the user like to giverny.

  The user might be assked to hit:
  
  + `c` to ask claude to commit.
  + `s` to start a shell and commit 
  + `r` to rstart claude code.
  + `x` to exit.
  
  Options 1 and 2 would only be shown if the working directory is dirty.
  
  The prompt might look like:
  
  ```
  The working directory is dirty.
  
  [INSERT git status OUTPUT HERE]
  
  Would you like to:
  
  [c] Ask Claude Code to commit (non-interactive)
  [s] Manually commit in a Shell
  [r] Restart Claude (interactively)
  [x] Exit Giverny
  
  >
  ```
  
  or, if the working directory is clean
  
  ```
  Would you like to:

  [s] Start a shell
  [r] Restart Claude (interactively)
  [x] Exit Giverny
  
  >
  ```

* When the claude process exits.  Innie checks to see if `/app` is
  dirty (if there are any files that are not committed.  If it is it
  prints something like "Working directory is dirty" to `stderr`. Then starts an
  interactive `bash` in `/app`.
  
  + The user cleans up the files, commits if necessary.
  
  + The user exits bash.
  
  + Innie checks that directory is clean.  If it is not, it prints a the message again and starts `bash`.
  
* When the dirctory is clean, Innie pushs the branch to the git server.

* Innie exits with code 0.  This causes the container to stop.

* Outie detects that the container has stopped because the `docker run ...` command exits.

* Outie prints to `stdout` a message that reads something like. 

  ```
  Giverny has closed. 

  Merge changes:     git merge --ff-only giverny/TASK-ID
  Delete the branch: git -d -f giverny/TASK-ID
  ```

  Maybe a command to cherry pick, like sketch.

## Tech 

Giverny is written in go.

## The Docker Image

Must include: 
* git
* diffreviewer
* giverny


## Future work
  
* Check for commits and print something when a new commit is made.
  Maybe start diffreviewer and print the URL of diffreviewer.

* Open port for diffreviewer if we need them with `docker`.  Currently
  using OrbStack so don't need to open ports.
  
* Start a API server in Outie, pass the port to Innie.  Use this to
  send comannds from Innie to Outie.
  
* When the working directory is dirty Innie, after claude exits, could
  run `claude` in non-interactive mode, with a prompt asking claude to
  tidy up the working directory.
  
* Build diffreviewer in it's own image and copy it into the
  `giverny-main` image.
  
* Install the `diffreviewer` skill in the `giverny-main` container.


  
  
