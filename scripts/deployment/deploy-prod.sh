#!/bin/bash

# DevLab Production Deployment Script
# Uses environment variables for dynamic user mapping

set -e

echo "🚀 DevLab Production Deployment"
echo "================================"

# Check if running as root
if [ "$EUID" -eq 0 ]; then
    echo "❌ Please don't run as root. Use a regular user."
    exit 1
fi

# Detect user information
CURRENT_USER=$(whoami)
CURRENT_UID=$(id -u)
CURRENT_GID=$(id -g)

echo "👤 Current User: $CURRENT_USER"
echo "🆔 UID: $CURRENT_UID"
echo "🆔 GID: $CURRENT_GID"

# Set environment variables
export UID=$CURRENT_UID
export GID=$CURRENT_GID

# Set production environment variables
export JWT_SECRET=${JWT_SECRET:-$(openssl rand -hex 32)}
export REDIS_PASSWORD=${REDIS_PASSWORD:-$(openssl rand -hex 16)}

echo "🔐 JWT Secret: ${JWT_SECRET:0:16}..."
echo "🔐 Redis Password: ${REDIS_PASSWORD:0:16}..."

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

# Create necessary directories
echo "📁 Creating directories..."
mkdir -p logs configs

# Build images
echo "🔨 Building DevLab images..."
docker-compose -f docker-compose.prod.yml build

# Start services
echo "🚀 Starting DevLab services..."
docker-compose -f docker-compose.prod.yml up -d

# Wait for services to be ready
echo "⏳ Waiting for services to be ready..."
sleep 10

# Check service health
echo "🏥 Checking service health..."
docker-compose -f docker-compose.prod.yml ps

# Test API
echo "🧪 Testing API..."
if curl -s http://localhost:8000/healthz > /dev/null; then
    echo "✅ API is healthy"
else
    echo "❌ API health check failed"
    echo "📋 Checking logs..."
    docker-compose -f docker-compose.prod.yml logs devlab-api
fi

echo ""
echo "🎉 DevLab Production Deployment Complete!"
echo "========================================"
echo "🌐 API: http://localhost:8000"
echo "📖 Swagger: http://localhost:8000/swagger/index.html"
echo "📊 RabbitMQ: http://localhost:15672 (guest/guest)"
echo "🗄️  MongoDB: localhost:27017"
echo "🔴 Redis: localhost:6379"
echo ""
echo "📋 Useful commands:"
echo "   View logs: docker-compose -f docker-compose.prod.yml logs -f"
echo "   Stop services: docker-compose -f docker-compose.prod.yml down"
echo "   Restart: docker-compose -f docker-compose.prod.yml restart"
echo "   Status: docker-compose -f docker-compose.prod.yml ps" 