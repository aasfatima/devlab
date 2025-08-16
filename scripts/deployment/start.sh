#!/bin/bash

# DevLab Auto-Start Script
# Automatically detects user and sets environment variables

set -e

echo "🚀 DevLab Auto-Start Script"
echo "============================"

# Auto-detect user information
CURRENT_USER=$(whoami)
CURRENT_UID=$(id -u)
CURRENT_GID=$(id -g)

echo "👤 Detected User: $CURRENT_USER"
echo "🆔 UID: $CURRENT_UID"
echo "🆔 GID: $CURRENT_GID"

# Auto-set environment variables (using different names to avoid readonly UID)
export CONTAINER_UID=$CURRENT_UID
export CONTAINER_GID=$CURRENT_GID

echo "✅ Environment variables set:"
echo "   CONTAINER_UID=$CONTAINER_UID"
echo "   CONTAINER_GID=$CONTAINER_GID"

# Check Docker
if ! command -v docker &> /dev/null; then
    echo "❌ Docker not found. Please install Docker first."
    exit 1
fi

if ! docker info &> /dev/null; then
    echo "❌ Docker daemon not running. Please start Docker."
    exit 1
fi

# Check Docker socket permissions
if [ -S /var/run/docker.sock ]; then
    SOCKET_PERMS=$(ls -la /var/run/docker.sock | awk '{print $1}')
    SOCKET_OWNER=$(ls -la /var/run/docker.sock | awk '{print $3":"$4}')
    echo "🔌 Docker Socket: $SOCKET_PERMS ($SOCKET_OWNER)"
    
    if [ ! -r /var/run/docker.sock ]; then
        echo "⚠️  Docker socket not readable. You may need to:"
        echo "   - Add your user to the docker group: sudo usermod -aG docker $USER"
        echo "   - Or restart Docker Desktop"
        echo "   - Or run with sudo (not recommended for production)"
    fi
else
    echo "❌ Docker socket not found. Is Docker running?"
    exit 1
fi

# Stop existing containers
echo "🛑 Stopping existing containers..."
docker-compose down 2>/dev/null || true

# Start DevLab with auto-detected user
echo "🚀 Starting DevLab with user $CURRENT_USER ($CONTAINER_UID:$CONTAINER_GID)..."
CONTAINER_UID=$CONTAINER_UID CONTAINER_GID=$CONTAINER_GID docker-compose up -d

# Wait for services to be ready
echo "⏳ Waiting for services to be ready..."
sleep 10

# Check service health
echo "🏥 Checking service health..."
docker-compose ps

# Test API
echo "🧪 Testing API..."
if curl -s http://localhost:8000/healthz > /dev/null; then
    echo "✅ API is healthy"
else
    echo "❌ API health check failed"
    echo "📋 Checking logs..."
    docker-compose logs devlab-api
fi

# Test Docker access from container
echo "🔍 Testing Docker access from container..."
if docker exec devlab-api docker ps > /dev/null 2>&1; then
    echo "✅ Docker access working from container"
else
    echo "❌ Docker access failed from container"
    echo "⚠️  This is expected on macOS - the container can't access host Docker socket"
    echo "💡 For production, use cloud platforms that support Docker-in-Docker"
fi

echo ""
echo "🎉 DevLab started successfully!"
echo "========================================"
echo "🌐 API: http://localhost:8000"
echo "📖 Swagger: http://localhost:8000/swagger/index.html"
echo "📊 RabbitMQ: http://localhost:15672 (guest/guest)"
echo "🗄️  MongoDB: localhost:27017"
echo ""
echo "📋 Useful commands:"
echo "   View logs: docker-compose logs -f"
echo "   Stop services: docker-compose down"
echo "   Restart: docker-compose restart"
echo "   Status: docker-compose ps" 