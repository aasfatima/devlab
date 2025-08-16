# DevLab - Containerized Development Environment Manager

A Go-based platform for managing isolated development environments using Docker containers. Perfect for coding interviews, demos, and secure development scenarios.

## Features

- **Multi-language Support**: Go, Python, Docker, Kubernetes environments
- **Real-time Terminal Access**: Web-based terminal with ttyd
- **Scenario Management**: Create, start, stop, and monitor development scenarios
- **RESTful API**: Clean HTTP API with Swagger documentation
- **JWT Authentication**: Secure access control
- **Docker Integration**: Seamless container management

## Quick Start

### Prerequisites
- Docker & Docker Compose
- Go 1.21+
- MongoDB (via Docker)
- RabbitMQ (via Docker)

### Installation
```bash
# Clone the repository
git clone https://github.com/aasfatima/devlab.git
cd devlab

# Start the services
./scripts/deployment/start.sh

# The API will be available at http://localhost:8000
# Swagger docs: http://localhost:8000/swagger/index.html
```

### API Usage
```bash
# Start a Go development scenario
curl -X POST http://localhost:8000/scenarios/start \
  -H "Content-Type: application/json" \
  -d '{"user_id": "developer", "scenario_type": "go"}'

# Get scenario status
curl http://localhost:8000/scenarios/{scenario_id}/status

# Access terminal
curl http://localhost:8000/scenarios/{scenario_id}/terminal
```

## Architecture

- **API Server**: Gin-based REST API with JWT auth
- **Scenario Manager**: Docker container orchestration
- **Storage**: MongoDB for scenario persistence
- **Queue**: RabbitMQ for async operations
- **Terminal**: ttyd for web-based terminal access

## Development

```bash
# Run tests
go test -v ./...

# Run integration tests
go test -v ./tests/integration/

# Build
go build -o bin/api cmd/api/main.go
```

## API Documentation

Visit `http://localhost:8000/swagger/index.html` for interactive API documentation.

## License

MIT License - see LICENSE file for details.
