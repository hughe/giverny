# Builder a base container for Giverny to work on Giverny in.

# Start with Go in Debian Trixie.
FROM golang:1.25.5-trixie

RUN apt update -y

################################################################################
# Install the Docker CLI

# Add Docker's official GPG key:
RUN apt install -y ca-certificates curl && \
    install -m 0755 -d /etc/apt/keyrings && \
    curl -fsSL https://download.docker.com/linux/debian/gpg -o /etc/apt/keyrings/docker.asc && \
    chmod a+r /etc/apt/keyrings/docker.asc

# Add the repository to Apt sources.
RUN <<EOF tee /etc/apt/sources.list.d/docker.sources
Types: deb
URIs: https://download.docker.com/linux/debian
Suites: $(. /etc/os-release && echo "$VERSION_CODENAME")
Components: stable
Signed-By: /etc/apt/keyrings/docker.asc
EOF

# RUN cat /etc/apt/sources.list.d/docker.sources

# Install the docker CLI
RUN apt update -y
RUN apt install -y docker-ce-cli

