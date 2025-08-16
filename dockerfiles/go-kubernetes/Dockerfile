FROM devlab-k8s:latest

# Install Go
RUN wget -c https://go.dev/dl/go1.22.4.linux-amd64.tar.gz -O - | tar -xz -C /opt/ && \
    chown -R devlab:devlab /opt/go

USER devlab

# Set up Go environment
RUN printf 'export GOROOT=/opt/go\nexport PATH=${PATH}:${GOROOT}/bin\nexport GOPATH=/home/devlab/go\nexport GOCACHE=/home/devlab/.cache/go-build\n' >> /home/devlab/.bashrc

# Create Go workspace
RUN mkdir -p /home/devlab/go/src /home/devlab/go/bin /home/devlab/go/pkg

USER root

# Install Go tools
RUN apt-get update && apt-get install -y golang-golang-x-tools && \
    rm -rf /var/lib/apt/lists/*

# Create symlink for go command
RUN rm -f /usr/bin/go && ln -s /opt/go/bin/go /usr/bin/go

# Install additional Go tools
USER devlab
RUN go install golang.org/x/tools/cmd/goimports@latest && \
    go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest && \
    go install github.com/go-delve/delve/cmd/dlv@latest

USER root

# Set working directory
WORKDIR /home/devlab/go/src

# Default command - use the entrypoint from base image
CMD ["/bin/bash"] 