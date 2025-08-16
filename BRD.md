# 📄 Business Requirements Document (BRD)

## 🏷️ Project Title:
**DevLab – Cloud-Based Coding Environment Provisioner**

---

## 📌 Executive Summary
DevLab is a cloud-native servive designed to provision isolated development environments (containers) for users. It enables learners, developers, or engineers to launch fully-configured coding workspaces that include language-specific tools, Docker, Kubernetes CLIs, and direct terminal access through a browser. It is Golang project.

The project is inspired by real-world experience with cloud platforms like Katacoda, DevLab focuses on infrastructure automation, container orchestration, and developer usability. It will be implemented in **Go**, with supporting tools like **RabbitMQ**, **MongoDB**,**Kubernetes**, **Docker**, **gRPC**, and **ttyd** for web terminal access.

The scope is limited to an MVP deliverable within **1-2 weeks**, targeting a feature-rich but lean architecture.

---

## 🎯 Goals & Objectives

- Provision Docker-based coding environments via REST API or gRPC
- Allow users to run interactive terminal sessions in a browser
- Handle background cleanup of idle or stopped environments using queues
- Store scenario metadata and session details in a database
- Simulate real developer workflow: launching, running scripts, managing containers
- Showcase production-aligned backend architecture using microservice patterns

---

## 🧩 Core Features (Functional Requirements)

| Feature                  | Description                                                                 |
|--------------------------|-----------------------------------------------------------------------------|
| Start Scenario API       | Launches a pre-defined container environment (e.g. Go/K8s/Docker) with optional script injection |
| Stop Scenario API        | Stops and cleans up a specific running container                            |
| Terminal Access          | Launch a live terminal using ttyd, embedded via web browser, allowing users to interact with their environment |
| Directory Structure JSON | Returns a file tree-like JSON structure representing the container's home directory |
| Status API               | Returns the current status of a scenario (e.g. running, stopped, provisioning) |
| Async Cleanup Worker     | RabbitMQ consumer to stop and clean up containers after timeout/inactivity  |

---

## 🧰 Non-Functional Requirements

- Built in Golang using idiomatic project structure
- REST APIs via Gin
- Internal service communication via gRPC
- MongoDB for persistent storage of scenario metadata
- RabbitMQ for background job queues
- Docker for isolated container environments
- Embedded terminal via ttyd
- OpenTelemetry + Zerolog for observability
- JWT-based token auth (optional for MVP)

---

## 🔧 Technical Architecture Overview

```
User
 ↓
[ Gin API Server ]
 ↓
[ Scenario Controller ]
 ↓
[ Docker Container Provisioner ]
 ↓
[ MongoDB ] ←→ [ RabbitMQ Publisher ] → [ Cleanup Worker (Goroutine) ]
 ↓
[ ttyd terminal service exposed at port 3000 ]
 ↓
[ HTML frontend embeds terminal in <iframe> ]
```

---

## 💡 Terminal Access (ttyd) – Detailed Breakdown

| Item            | Detail                                                                 |
|-----------------|------------------------------------------------------------------------|
| Tool            | ttyd (Terminal over WebSocket)                                         |
| Port            | 3000 inside each container                                              |
| Embed Method    | `<iframe src="http://localhost:3000">` or custom UI                    |
| Shell Access    | Bash terminal inside container with Go/Docker/K8s tools                |
| Purpose         | Enables users to run live commands (e.g. `kubectl`, `touch`, etc.)     |
| Installed Tools | go, docker, kubectl, vim, nano, git, etc.                              |
| Docker Setup    | Installed and configured in container using Dockerfile                 |
| Use Case        | Demonstrates cloud-based development scenarios in interview-ready way  |

---

## 📦 Scenario Types (Optional Scope)

| Type             | Tools Pre-installed        | Use Case                          |
|------------------|----------------------------|-----------------------------------|
| Go Environment   | go, vim, Git               | Write and run Go code             |
| Docker-in-Docker | Docker CLI, Compose        | Test Dockerfiles, build images    |
| Kubernetes Lab   | kubectl, kind              | Simulate `kubectl` commands       |
| Python Environment| python3, pip, Flask        | Write and run Python code         |
| Go-Kubernetes    | go, kubectl, kind          | Go development with K8s tools     |
| Python-Kubernetes| python3, kubectl, kind     | Python development with K8s tools |

---

## 🧪 MVP Deliverables (1-Week Scope)

| Deliverable                       | Status |
|----------------------------------|--------|
| REST API to start/stop scenario  | ✅     |
| Docker container provisioning    | ✅     |
| Web terminal integration (ttyd)  | ✅     |
| MongoDB metadata storage         | ✅     |
| Async cleanup job (RabbitMQ)     | ✅     |
| Directory structure endpoint     | ✅     |
| gRPC interface (optional)        | ✅     |
| Swagger/OpenAPI docs             | ✅     |

---

## 🛠 Dev Tools & Libraries

- Gin – REST framework
- gRPC – internal communication
- MongoDB – NoSQL DB
- RabbitMQ – messaging broker
- Docker – container engine
- ttyd – web terminal
- OpenTelemetry + Zerolog – tracing/logging

---

## ⏱ Timeline (1 Week Plan)

| Day | Focus Area                                  |
|-----|----------------------------------------------|
| 1   | Setup base project, Dockerfiles, MongoDB     |
| 2   | Implement start/stop scenario API            |
| 3   | Integrate ttyd terminal in container         |
| 4   | Build directory structure API                |
| 5   | Implement cleanup logic with RabbitMQ        |
| 6   | Testing, Docker Compose, Swagger setup       |
| 7   | Final polish, README, push to GitHub         |

---

## 📎 Future Scope (Optional)

- Add file explorer UI with editable files
- Add custom Go-based editor in the browser (Monaco, xterm.js)
- Per-user container isolation with access tokens
- Full user authentication and login flow
- Allow loading scenarios from YAML configs
- Resource usage monitoring (CPU, memory)

---

## 📘 Appendix

- Example image: Terminal executing `kubectl get deployments`
- Folder structure (monorepo layout)
- HTML snippet to embed terminal
- Docker Compose file with terminal port
- Screenshot of successful provisioning flow
