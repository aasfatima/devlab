FROM devlab-base:latest

# Install kind (Kubernetes in Docker)
RUN curl -Lo ./kind https://kind.sigs.k8s.io/dl/v0.20.0/kind-linux-amd64 && \
    chmod +x ./kind && \
    mv ./kind /usr/local/bin/kind

# Install k3s with native snapshotter (FIXES the issue!)
RUN curl -sfL https://get.k3s.io | INSTALL_K3S_SKIP_START=true INSTALL_K3S_SKIP_ENABLE=true INSTALL_K3S_EXEC=" --snapshotter=native" sh -

# Install additional Kubernetes tools
RUN curl -Lo /usr/local/bin/kustomize https://github.com/kubernetes-sigs/kustomize/releases/download/kustomize%2Fv5.0.3/kustomize_v5.0.3_linux_amd64 && \
    chmod +x /usr/local/bin/kustomize

# Install kubectx and kubens
RUN git clone https://github.com/ahmetb/kubectx /opt/kubectx && \
    ln -s /opt/kubectx/kubectx /usr/local/bin/kubectx && \
    ln -s /opt/kubectx/kubens /usr/local/bin/kubens

# Create k3s startup script
RUN echo '#!/bin/bash' > /usr/local/bin/start-k3s.sh && \
    echo 'echo "Starting k3s server..."' >> /usr/local/bin/start-k3s.sh && \
    echo 'k3s server --disable-agent --snapshotter=native &' >> /usr/local/bin/start-k3s.sh && \
    echo 'K3S_PID=$!' >> /usr/local/bin/start-k3s.sh && \
    echo 'echo "k3s PID: $K3S_PID"' >> /usr/local/bin/start-k3s.sh && \
    echo 'sleep 10' >> /usr/local/bin/start-k3s.sh && \
    echo 'echo "Waiting for k3s config..."' >> /usr/local/bin/start-k3s.sh && \
    echo 'while [ ! -f /etc/rancher/k3s/k3s.yaml ]; do' >> /usr/local/bin/start-k3s.sh && \
    echo '  sleep 2' >> /usr/local/bin/start-k3s.sh && \
    echo 'done' >> /usr/local/bin/start-k3s.sh && \
    echo 'echo "k3s config ready"' >> /usr/local/bin/start-k3s.sh && \
    echo 'mkdir -p /home/devlab/.kube' >> /usr/local/bin/start-k3s.sh && \
    echo 'cp /etc/rancher/k3s/k3s.yaml /home/devlab/.kube/config' >> /usr/local/bin/start-k3s.sh && \
    echo 'echo "Fixing k3s port configuration..."' >> /usr/local/bin/start-k3s.sh && \
    echo 'sed -i "s/127.0.0.1:6443/127.0.0.1:6444/g" /home/devlab/.kube/config' >> /usr/local/bin/start-k3s.sh && \
    echo 'chown devlab:devlab -R /home/devlab/.kube' >> /usr/local/bin/start-k3s.sh && \
    echo 'echo "Setting up kubectl environment..."' >> /usr/local/bin/start-k3s.sh && \
    echo 'export KUBECONFIG=/home/devlab/.kube/config' >> /usr/local/bin/start-k3s.sh && \
    echo 'echo "Waiting for k3s API to be ready..."' >> /usr/local/bin/start-k3s.sh && \
    echo 'sleep 20' >> /usr/local/bin/start-k3s.sh && \
    echo 'echo "Testing k3s connectivity..."' >> /usr/local/bin/start-k3s.sh && \
    echo 'k3s kubectl get nodes > /dev/null 2>&1 && echo "k3s is ready!" || echo "k3s still starting..."' >> /usr/local/bin/start-k3s.sh && \
    echo 'echo "k3s setup complete"' >> /usr/local/bin/start-k3s.sh && \
    chmod +x /usr/local/bin/start-k3s.sh

# Create entrypoint script that will be called by DevLab
RUN echo '#!/bin/bash' > /usr/local/bin/devlab-entrypoint.sh && \
    echo 'set -e' >> /usr/local/bin/devlab-entrypoint.sh && \
    echo 'echo "DevLab Entrypoint: Starting k3s initialization..."' >> /usr/local/bin/devlab-entrypoint.sh && \
    echo '/usr/local/bin/start-k3s.sh &' >> /usr/local/bin/devlab-entrypoint.sh && \
    echo 'echo "k3s initialization started in background"' >> /usr/local/bin/devlab-entrypoint.sh && \
    echo 'echo "DevLab Entrypoint: Executing provided command..."' >> /usr/local/bin/devlab-entrypoint.sh && \
    echo 'exec "$@"' >> /usr/local/bin/devlab-entrypoint.sh && \
    chmod +x /usr/local/bin/devlab-entrypoint.sh

# Set up Kubernetes environment
USER devlab
RUN mkdir -p /home/devlab/.kube && \
    echo 'export KUBECONFIG=/home/devlab/.kube/config' >> /home/devlab/.bashrc && \
    echo 'alias k="kubectl"' >> /home/devlab/.bashrc && \
    echo 'alias k3sk="k3s kubectl"' >> /home/devlab/.bashrc

USER root

# Set working directory
WORKDIR /home/devlab

# Set entrypoint to handle k3s initialization
ENTRYPOINT ["/usr/local/bin/devlab-entrypoint.sh"]

# Default command - start k3s and then bash
CMD ["/bin/bash", "-c", "/usr/local/bin/start-k3s.sh && /bin/bash"] 