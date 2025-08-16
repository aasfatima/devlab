FROM devlab-base:latest

# Install Python 3.9 and related tools
RUN apt-get update && apt-get install -y \
    python3 \
    python3.9 \
    python3-pip \
    python3.9-distutils \
    python3-pygments \
    pkg-config \
    software-properties-common \
    && rm -rf /var/lib/apt/lists/*

# Add deadsnakes PPA for Python 3.9
RUN add-apt-repository ppa:deadsnakes/ppa && \
    apt-get update && \
    apt-get install -y python3.9 && \
    rm -rf /var/lib/apt/lists/*

# Create symlinks for Python
RUN ln -s /usr/bin/pip3 /usr/local/bin/pip && \
    rm -f /usr/bin/python && \
    ln -s /usr/bin/python3 /usr/bin/python

USER devlab

# Set up Python environment
RUN printf 'export PYTHONPATH=/home/devlab/python\n' >> /home/devlab/.bashrc && \
    printf 'export LC_ALL=C.UTF-8\nexport LANG=C.UTF-8\n' >> /home/devlab/.bashrc

# Install common Python packages
RUN pip install Flask==2.0.1 requests pytest

USER root

# Create Python workspace
RUN mkdir -p /home/devlab/python/src && \
    chown -R devlab:devlab /home/devlab/python

# Set working directory
WORKDIR /home/devlab/python/src

# Default command
CMD ["/bin/bash"] 