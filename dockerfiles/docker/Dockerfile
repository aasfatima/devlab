FROM devlab-base:latest

# Since Docker CLI is already installed in base image, we just need to set up Docker-in-Docker
# Add devlab user to docker group (if it exists)
RUN usermod -aG docker devlab 2>/dev/null || echo "Docker group not available"

# Install Docker Compose (if not already installed)
RUN if ! command -v docker-compose &> /dev/null; then \
        curl -L "https://github.com/docker/compose/releases/download/v2.20.0/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose && \
        chmod +x /usr/local/bin/docker-compose; \
    fi

# Install additional Docker tools
RUN curl -L https://github.com/docker/buildx/releases/download/v0.11.0/buildx-v0.11.0.linux-amd64 -o /usr/local/bin/docker-buildx && \
    chmod +x /usr/local/bin/docker-buildx && \
    mkdir -p /home/devlab/.docker/cli-plugins && \
    ln -s /usr/local/bin/docker-buildx /home/devlab/.docker/cli-plugins/docker-buildx

# Set up Docker environment
USER devlab
RUN mkdir -p /home/devlab/.docker && \
    echo 'export DOCKER_BUILDKIT=1' >> /home/devlab/.bashrc && \
    echo 'export COMPOSE_DOCKER_CLI_BUILD=1' >> /home/devlab/.bashrc

USER root

# Set working directory
WORKDIR /home/devlab

# Default command
CMD ["/bin/bash"] 