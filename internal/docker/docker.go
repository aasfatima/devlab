package docker

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

// Custom error types for better error handling
var (
	ErrContainerNotFound       = errors.New("container not found")
	ErrContainerNotRunning     = errors.New("container is not running")
	ErrPortUnavailable         = errors.New("no available ports found")
	ErrTTYDFailedToStart       = errors.New("ttyd failed to start")
	ErrInvalidScenarioType     = errors.New("invalid scenario type")
	ErrDockerDaemonUnavailable = errors.New("docker daemon unavailable")
)

type Client interface {
	StartScenarioContainer(ctx context.Context, scenarioType, script string) (string, int, error)
	GetContainerStatus(ctx context.Context, containerID string) (string, error)
	GetTerminalURL(ctx context.Context, containerID string) (string, error)
	StopContainer(ctx context.Context, containerID string) error
	ContainerExists(ctx context.Context, containerID string) (bool, error)
	ExecuteCommand(ctx context.Context, containerID string, command []string) (string, error)
	ListContainers(ctx context.Context) ([]ContainerInfo, error)
	RemoveContainer(ctx context.Context, containerID string) error
}

// ContainerInfo represents information about a Docker container
type ContainerInfo struct {
	ID     string
	Name   string
	Status string
}

type RealClient struct{}

func (RealClient) StartScenarioContainer(ctx context.Context, scenarioType, script string) (string, int, error) {
	if ctx == nil {
		return "", 0, errors.New("nil context provided")
	}

	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		log.Printf("[docker] failed to create client: %v", err)
		return "", 0, fmt.Errorf("%w: %v", ErrDockerDaemonUnavailable, err)
	}
	defer cli.Close()

	// Validate scenario type
	if scenarioType == "" {
		return "", 0, fmt.Errorf("%w: scenario type cannot be empty", ErrInvalidScenarioType)
	}

	// Select image based on scenarioType
	image := "devlab-go:latest"
	switch scenarioType {
	case "go":
		image = "devlab-go:latest"
	case "docker":
		image = "devlab-docker:latest"
	case "k8s":
		image = "devlab-k8s:latest"
	case "python":
		image = "devlab-python:latest"
	case "go-k8s":
		image = "devlab-go-k8s:latest"
	case "python-k8s":
		image = "devlab-python-k8s:latest"
	default:
		log.Printf("[docker] unknown scenario type: %s, using default devlab-go image", scenarioType)
	}
	log.Printf("[docker] using image: %s for scenario type: %s", image, scenarioType)

	// Find an available port for ttyd
	hostPort, err := findAvailablePort()
	if err != nil {
		log.Printf("[docker] failed to find available port: %v", err)
		return "", 0, fmt.Errorf("%w: %v", ErrPortUnavailable, err)
	}
	log.Printf("[docker] using host port %d for ttyd", hostPort)

	var mounts []mount.Mount

	// Create a startup script that runs ttyd (pre-installed in custom images)
	startupScript := fmt.Sprintf(`#!/bin/sh
set -e

# Set scenario type for k3s initialization
SCENARIO_TYPE="%s"

echo "Starting ttyd on port 3000..."
# Start ttyd in background with error checking
ttyd -p 3000 -c admin:admin --writable -t disableReuse=true bash &
TTYD_PID=$!

# Wait a moment for ttyd to start and check if it's running
sleep 3
if ! kill -0 $TTYD_PID 2>/dev/null; then
    echo "ERROR: ttyd failed to start"
    exit 1
fi

echo "ttyd started successfully on port 3000"

# Initialize k3s for k8s scenarios
if [ "$SCENARIO_TYPE" = "k8s" ] || [ "$SCENARIO_TYPE" = "go-k8s" ] || [ "$SCENARIO_TYPE" = "python-k8s" ]; then
    echo "Initializing k3s for Kubernetes scenario..."
    /usr/local/bin/start-k3s.sh &
    echo "k3s initialization started in background"
fi

# Run the scenario script if provided
%s

# Keep container running
echo "Container ready for terminal access"
sleep infinity
`, scenarioType, script)

	// Create startup script content (will be written inside container)
	startupScriptContent := startupScript

	exposedPorts := nat.PortSet{"3000/tcp": struct{}{}}
	portBindings := nat.PortMap{
		"3000/tcp": []nat.PortBinding{{
			HostIP:   "0.0.0.0",
			HostPort: fmt.Sprintf("%d", hostPort),
		}},
	}

	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image:        image,
		Cmd:          []string{"sh", "-c", "cat > /tmp/startup.sh << 'EOF'\n" + startupScriptContent + "\nEOF\nchmod +x /tmp/startup.sh && sh /tmp/startup.sh"},
		Tty:          true,
		ExposedPorts: exposedPorts,
	}, &container.HostConfig{
		Mounts:       mounts,
		PortBindings: portBindings,
	}, nil, nil, "")
	if err != nil {
		log.Printf("[docker] failed to create container: %v", err)
		return "", 0, fmt.Errorf("failed to create container: %w", err)
	}

	if err := cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		log.Printf("[docker] failed to start container %s: %v", resp.ID, err)
		// Try to clean up the created container
		cli.ContainerRemove(ctx, resp.ID, container.RemoveOptions{})
		return "", 0, fmt.Errorf("failed to start container: %w", err)
	}

	// Wait a bit and check if container is still running
	time.Sleep(5 * time.Second)
	containerInfo, err := cli.ContainerInspect(ctx, resp.ID)
	if err != nil {
		log.Printf("[docker] failed to inspect container %s: %v", resp.ID, err)
		return "", 0, fmt.Errorf("failed to verify container status: %w", err)
	}

	if containerInfo.State.Status != "running" {
		log.Printf("[docker] container %s is not running, status: %s", resp.ID, containerInfo.State.Status)
		// Try to get logs for debugging
		logs, _ := cli.ContainerLogs(ctx, resp.ID, container.LogsOptions{})
		if logs != nil {
			defer logs.Close()
			log.Printf("[docker] container logs for %s:", resp.ID)
			// Read and log the container logs
		}
		return "", 0, fmt.Errorf("%w: container exited unexpectedly", ErrTTYDFailedToStart)
	}

	log.Printf("[docker] started container: %s with ttyd on port %d", resp.ID, hostPort)
	return resp.ID, hostPort, nil
}

func (RealClient) GetContainerStatus(ctx context.Context, containerID string) (string, error) {
	if ctx == nil {
		return "", errors.New("nil context provided")
	}

	if containerID == "" {
		return "", errors.New("container ID cannot be empty")
	}

	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		log.Printf("[docker] failed to create client: %v", err)
		return "", fmt.Errorf("%w: %v", ErrDockerDaemonUnavailable, err)
	}
	defer cli.Close()

	containerInfo, err := cli.ContainerInspect(ctx, containerID)
	if err != nil {
		log.Printf("[docker] failed to inspect container %s: %v", containerID, err)
		return "", fmt.Errorf("%w: %v", ErrContainerNotFound, err)
	}

	status := containerInfo.State.Status
	log.Printf("[docker] container %s status: %s", containerID, status)
	return status, nil
}

func (RealClient) GetTerminalURL(ctx context.Context, containerID string) (string, error) {
	if ctx == nil {
		return "", errors.New("nil context provided")
	}

	if containerID == "" {
		return "", errors.New("container ID cannot be empty")
	}

	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		log.Printf("[docker] failed to create client: %v", err)
		return "", fmt.Errorf("%w: %v", ErrDockerDaemonUnavailable, err)
	}
	defer cli.Close()

	containerInfo, err := cli.ContainerInspect(ctx, containerID)
	if err != nil {
		log.Printf("[docker] failed to inspect container %s: %v", containerID, err)
		return "", fmt.Errorf("%w: %v", ErrContainerNotFound, err)
	}

	// Check if container is running
	if containerInfo.State.Status != "running" {
		return "", fmt.Errorf("%w: container status is %s", ErrContainerNotRunning, containerInfo.State.Status)
	}

	// Find the host port mapping for container port 3000
	networkSettings := containerInfo.NetworkSettings
	if networkSettings == nil || networkSettings.Ports == nil {
		return "", fmt.Errorf("no port mappings found for container %s", containerID)
	}

	portBindings, exists := networkSettings.Ports["3000/tcp"]
	if !exists || len(portBindings) == 0 {
		return "", fmt.Errorf("port 3000 not mapped for container %s", containerID)
	}

	hostPort := portBindings[0].HostPort
	hostIP := portBindings[0].HostIP
	if hostIP == "" {
		hostIP = "localhost"
	}

	terminalURL := fmt.Sprintf("http://%s:%s", hostIP, hostPort)
	log.Printf("[docker] terminal URL for container %s: %s", containerID, terminalURL)
	return terminalURL, nil
}

func (RealClient) StopContainer(ctx context.Context, containerID string) error {
	if ctx == nil {
		return errors.New("nil context provided")
	}

	if containerID == "" {
		return errors.New("container ID cannot be empty")
	}

	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		log.Printf("[docker] failed to create client: %v", err)
		return fmt.Errorf("%w: %v", ErrDockerDaemonUnavailable, err)
	}
	defer cli.Close()

	// Check if container exists and get its status
	containerInfo, err := cli.ContainerInspect(ctx, containerID)
	if err != nil {
		if client.IsErrNotFound(err) {
			return fmt.Errorf("%w: container %s", ErrContainerNotFound, containerID)
		}
		return fmt.Errorf("failed to check container existence: %w", err)
	}

	// If container is already stopped or exited, just remove it
	if containerInfo.State.Status == "exited" || containerInfo.State.Status == "stopped" {
		if err := cli.ContainerRemove(ctx, containerID, container.RemoveOptions{}); err != nil {
			log.Printf("[docker] failed to remove stopped container %s: %v", containerID, err)
			return fmt.Errorf("failed to remove stopped container: %w", err)
		}
		log.Printf("[docker] removed stopped container: %s", containerID)
		return nil
	}

	// Stop the container
	if err := cli.ContainerStop(ctx, containerID, container.StopOptions{}); err != nil {
		log.Printf("[docker] failed to stop container %s: %v", containerID, err)
		return fmt.Errorf("failed to stop container: %w", err)
	}

	// Remove the container
	if err := cli.ContainerRemove(ctx, containerID, container.RemoveOptions{}); err != nil {
		log.Printf("[docker] failed to remove container %s: %v", containerID, err)
		return fmt.Errorf("failed to remove container: %w", err)
	}

	log.Printf("[docker] stopped and removed container: %s", containerID)
	return nil
}

func (RealClient) ContainerExists(ctx context.Context, containerID string) (bool, error) {
	if ctx == nil {
		return false, errors.New("nil context provided")
	}

	if containerID == "" {
		return false, errors.New("container ID cannot be empty")
	}

	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		log.Printf("[docker] failed to create client: %v", err)
		return false, fmt.Errorf("%w: %v", ErrDockerDaemonUnavailable, err)
	}
	defer cli.Close()

	_, err = cli.ContainerInspect(ctx, containerID)
	if err != nil {
		if client.IsErrNotFound(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to inspect container: %w", err)
	}

	return true, nil
}

// findAvailablePort finds an available port starting from 3001
func findAvailablePort() (int, error) {
	for port := 3001; port < 3010; port++ {
		addr := fmt.Sprintf(":%d", port)
		ln, err := net.Listen("tcp", addr)
		if err == nil {
			ln.Close()
			return port, nil
		}
	}
	return 0, fmt.Errorf("%w: no available ports found in range 3001-3009", ErrPortUnavailable)
}

func (RealClient) ExecuteCommand(ctx context.Context, containerID string, command []string) (string, error) {
	if ctx == nil {
		return "", errors.New("nil context provided")
	}

	if containerID == "" {
		return "", errors.New("container ID cannot be empty")
	}

	if len(command) == 0 {
		return "", errors.New("command cannot be empty")
	}

	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		log.Printf("[docker] failed to create client: %v", err)
		return "", fmt.Errorf("%w: %v", ErrDockerDaemonUnavailable, err)
	}
	defer cli.Close()

	// Check if container exists and is running
	containerInfo, err := cli.ContainerInspect(ctx, containerID)
	if err != nil {
		log.Printf("[docker] failed to inspect container %s: %v", containerID, err)
		return "", fmt.Errorf("%w: %v", ErrContainerNotFound, err)
	}

	if containerInfo.State.Status != "running" {
		return "", fmt.Errorf("%w: container status is %s", ErrContainerNotRunning, containerInfo.State.Status)
	}

	// Create exec configuration
	execConfig := types.ExecConfig{
		Cmd:          command,
		AttachStdout: true,
		AttachStderr: true,
	}

	// Create exec instance
	execResp, err := cli.ContainerExecCreate(ctx, containerID, execConfig)
	if err != nil {
		log.Printf("[docker] failed to create exec for container %s: %v", containerID, err)
		return "", fmt.Errorf("failed to create exec: %w", err)
	}

	// Attach to exec instance
	resp, err := cli.ContainerExecAttach(ctx, execResp.ID, types.ExecStartCheck{})
	if err != nil {
		log.Printf("[docker] failed to attach to exec for container %s: %v", containerID, err)
		return "", fmt.Errorf("failed to attach to exec: %w", err)
	}
	defer resp.Close()

	// Read output
	output, err := ioutil.ReadAll(resp.Reader)
	if err != nil {
		log.Printf("[docker] failed to read exec output for container %s: %v", containerID, err)
		return "", fmt.Errorf("failed to read exec output: %w", err)
	}

	// Check exec exit code
	inspectResp, err := cli.ContainerExecInspect(ctx, execResp.ID)
	if err != nil {
		log.Printf("[docker] failed to inspect exec for container %s: %v", containerID, err)
		return "", fmt.Errorf("failed to inspect exec: %w", err)
	}

	if inspectResp.ExitCode != 0 {
		log.Printf("[docker] exec command failed with exit code %d for container %s", inspectResp.ExitCode, containerID)
		return string(output), fmt.Errorf("command failed with exit code %d", inspectResp.ExitCode)
	}

	log.Printf("[docker] executed command successfully in container %s", containerID)
	return string(output), nil
}

func (RealClient) ListContainers(ctx context.Context) ([]ContainerInfo, error) {
	if ctx == nil {
		return nil, errors.New("nil context provided")
	}

	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		log.Printf("[docker] failed to create client: %v", err)
		return nil, fmt.Errorf("%w: %v", ErrDockerDaemonUnavailable, err)
	}
	defer cli.Close()

	containers, err := cli.ContainerList(ctx, types.ContainerListOptions{All: true})
	if err != nil {
		log.Printf("[docker] failed to list containers: %v", err)
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	var containerInfos []ContainerInfo
	for _, container := range containers {
		name := container.ID
		if len(container.Names) > 0 {
			name = container.Names[0]
		}
		containerInfos = append(containerInfos, ContainerInfo{
			ID:     container.ID,
			Name:   name,
			Status: container.Status,
		})
	}

	log.Printf("[docker] found %d containers", len(containerInfos))
	return containerInfos, nil
}

func (RealClient) RemoveContainer(ctx context.Context, containerID string) error {
	if ctx == nil {
		return errors.New("nil context provided")
	}

	if containerID == "" {
		return errors.New("container ID cannot be empty")
	}

	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		log.Printf("[docker] failed to create client: %v", err)
		return fmt.Errorf("%w: %v", ErrDockerDaemonUnavailable, err)
	}
	defer cli.Close()

	// Check if container exists
	containerInfo, err := cli.ContainerInspect(ctx, containerID)
	if err != nil {
		log.Printf("[docker] failed to inspect container %s: %v", containerID, err)
		return fmt.Errorf("%w: %v", ErrContainerNotFound, err)
	}

	// Stop the container if it's running
	if containerInfo.State.Status == "running" {
		log.Printf("[docker] stopping container %s before removal", containerID)
		if err := cli.ContainerStop(ctx, containerID, container.StopOptions{}); err != nil {
			log.Printf("[docker] failed to stop container %s: %v", containerID, err)
			return fmt.Errorf("failed to stop container: %w", err)
		}
	}

	// Remove the container
	if err := cli.ContainerRemove(ctx, containerID, container.RemoveOptions{}); err != nil {
		log.Printf("[docker] failed to remove container %s: %v", containerID, err)
		return fmt.Errorf("failed to remove container: %w", err)
	}

	log.Printf("[docker] successfully removed container %s", containerID)
	return nil
}
