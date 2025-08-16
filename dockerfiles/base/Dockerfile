FROM ubuntu:20.04

# Set environment variables
ENV DEBIAN_FRONTEND=noninteractive
ENV TZ=UTC

# Create devlab user
RUN useradd -ms /bin/bash devlab
RUN mkdir -p /opt/devlab && chown -R devlab /opt/devlab

# Install system dependencies
RUN apt-get update && apt-get install -y \
    git \
    build-essential \
    g++ \
    gcc \
    make \
    pkg-config \
    vim \
    nano \
    curl \
    wget \
    unzip \
    jq \
    cmake \
    libjson-c-dev \
    libwebsockets-dev \
    python3 \
    python3-pip \
    software-properties-common \
    ca-certificates \
    gnupg \
    lsb-release \
    && rm -rf /var/lib/apt/lists/*

# Install ttyd from source (more reliable than package managers)
RUN git clone https://github.com/tsl0922/ttyd.git /tmp/ttyd && \
    cd /tmp/ttyd && \
    mkdir build && cd build && \
    cmake .. && make && make install && \
    cd / && rm -rf /tmp/ttyd

# Install Docker CLI (simplified approach)
RUN curl -fsSL https://download.docker.com/linux/ubuntu/gpg | gpg --dearmor -o /usr/share/keyrings/docker-archive-keyring.gpg && \
    echo "deb [arch=amd64 signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable" | tee /etc/apt/sources.list.d/docker.list > /dev/null && \
    apt-get update && \
    apt-get install -y docker-ce-cli containerd.io && \
    rm -rf /var/lib/apt/lists/*

# Install Docker Compose
RUN curl -L "https://github.com/docker/compose/releases/download/v2.20.0/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose && \
    chmod +x /usr/local/bin/docker-compose

# Install kubectl
RUN curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl" && \
    chmod +x kubectl && \
    mv kubectl /usr/local/bin/

# Set up environment for devlab user
RUN printf 'export LC_ALL=C.UTF-8\nexport LANG=C.UTF-8\n' >> /home/devlab/.bashrc

# Create startup script directory
RUN mkdir -p /opt/devlab/scripts && chown -R devlab /opt/devlab

WORKDIR /opt/devlab

# Switch to devlab user for final setup
USER devlab

# Set up bash profile
RUN echo 'alias ll="ls -la"' >> /home/devlab/.bashrc && \
    echo 'export PATH=$PATH:/usr/local/bin' >> /home/devlab/.bashrc

USER root

# Create systemd service for ttyd (optional, for systemd-enabled containers)
RUN echo '[Unit]' > /etc/systemd/system/ttyd.service && \
    echo 'Description=ttyd terminal service' >> /etc/systemd/system/ttyd.service && \
    echo 'After=network.target' >> /etc/systemd/system/ttyd.service && \
    echo '' >> /etc/systemd/system/ttyd.service && \
    echo '[Service]' >> /etc/systemd/system/ttyd.service && \
    echo 'Type=simple' >> /etc/systemd/system/ttyd.service && \
    echo 'User=devlab' >> /etc/systemd/system/ttyd.service && \
    echo 'ExecStart=/usr/local/bin/ttyd -p 3000 -c admin:admin bash' >> /etc/systemd/system/ttyd.service && \
    echo 'Restart=always' >> /etc/systemd/system/ttyd.service && \
    echo '' >> /etc/systemd/system/ttyd.service && \
    echo '[Install]' >> /etc/systemd/system/ttyd.service && \
    echo 'WantedBy=multi-user.target' >> /etc/systemd/system/ttyd.service

# Default command
CMD ["/bin/bash"] 