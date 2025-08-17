#!/bin/bash

# Build script for DevLab scenario images
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}Building DevLab scenario images...${NC}"

# Build base image first
echo -e "${YELLOW}Building base image...${NC}"
docker build -f dockerfiles/base/Dockerfile -t devlab-base:latest .

# Build Go scenario image
echo -e "${YELLOW}Building Go scenario image...${NC}"
docker build -f dockerfiles/go/Dockerfile -t devlab-go:latest .

# Build Docker scenario image
echo -e "${YELLOW}Building Docker scenario image...${NC}"
docker build -f dockerfiles/docker/Dockerfile -t devlab-docker:latest .

# Build Kubernetes scenario image
echo -e "${YELLOW}Building Kubernetes scenario image...${NC}"
docker build -f dockerfiles/kubernetes/Dockerfile -t devlab-k8s:latest .

# Build Python scenario image
echo -e "${YELLOW}Building Python scenario image...${NC}"
docker build -f dockerfiles/python/Dockerfile -t devlab-python:latest .

# Build Go-Kubernetes scenario image
echo -e "${YELLOW}Building Go-Kubernetes scenario image...${NC}"
docker build -f dockerfiles/go-kubernetes/Dockerfile -t devlab-go-k8s:latest .

# Build Python-Kubernetes scenario image
echo -e "${YELLOW}Building Python-Kubernetes scenario image...${NC}"
docker build -f dockerfiles/python-kubernetes/Dockerfile -t devlab-python-k8s:latest .

echo -e "${GREEN}All images built successfully!${NC}"

# List built images
echo -e "${YELLOW}Built images:${NC}"
docker images | grep devlab 