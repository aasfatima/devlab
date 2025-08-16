package scenario

import (
	"context"
	"devlab/internal/config"
	"devlab/internal/docker"
	"devlab/internal/storage"
	"devlab/internal/types"
	"errors"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
)

// Custom error types for scenario management
var (
	ErrScenarioNotFound       = errors.New("scenario not found")
	ErrScenarioNotRunning     = errors.New("scenario is not running")
	ErrScenarioAlreadyStopped = errors.New("scenario is already stopped")
	ErrInvalidScenarioID      = errors.New("invalid scenario ID")
	ErrDatabaseUnavailable    = errors.New("database unavailable")
)

type Manager struct {
	Cfg    *config.Config
	DB     *mongo.Database
	Docker docker.Client
}

func NewManager(cfg *config.Config, db *mongo.Database, dockerClient docker.Client) *Manager {
	return &Manager{Cfg: cfg, DB: db, Docker: dockerClient}
}

func (m *Manager) StartScenario(ctx context.Context, req *types.StartScenarioRequest) (*types.StartScenarioResponse, error) {
	if ctx == nil {
		return nil, errors.New("nil context provided")
	}

	if req == nil {
		return nil, errors.New("request cannot be nil")
	}

	if req.UserID == "" {
		return nil, errors.New("user ID cannot be empty")
	}

	if req.ScenarioType == "" {
		return nil, errors.New("scenario type cannot be empty")
	}

	log.Printf("[scenario] starting scenario for user: %s, type: %s", req.UserID, req.ScenarioType)

	containerID, terminalPort, err := m.Docker.StartScenarioContainer(ctx, req.ScenarioType, req.Script)
	if err != nil {
		log.Printf("[scenario] docker error: %v", err)
		return nil, fmt.Errorf("failed to provision container: %w", err)
	}

	scenarioID := fmt.Sprintf("scn-%d", time.Now().UnixNano())
	s := &storage.Scenario{
		ScenarioID:   scenarioID,
		UserID:       req.UserID,
		ScenarioType: req.ScenarioType,
		ContainerID:  containerID,
		Status:       "provisioning",
		TerminalPort: terminalPort,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := storage.StoreScenario(ctx, m.DB, s); err != nil {
		log.Printf("[scenario] mongo error: %v", err)
		// Try to clean up the container if database storage fails
		m.Docker.StopContainer(ctx, containerID)
		return nil, fmt.Errorf("failed to store scenario metadata: %w", err)
	}

	log.Printf("[scenario] scenario created: %s (container: %s, terminal port: %d)", scenarioID, containerID, terminalPort)
	return &types.StartScenarioResponse{
		ScenarioID: scenarioID,
		Status:     "provisioning",
	}, nil
}

func (m *Manager) GetScenarioStatus(ctx context.Context, scenarioID string) (*types.ScenarioStatusResponse, error) {
	if ctx == nil {
		return nil, errors.New("nil context provided")
	}

	if scenarioID == "" {
		return nil, fmt.Errorf("%w: scenario ID cannot be empty", ErrInvalidScenarioID)
	}

	log.Printf("[scenario] getting status for scenario: %s", scenarioID)

	// Get scenario from database
	scenario, err := storage.GetScenario(ctx, m.DB, scenarioID)
	if err != nil {
		log.Printf("[scenario] failed to get scenario from DB: %v", err)
		if errors.Is(err, storage.ErrScenarioNotFound) {
			return nil, fmt.Errorf("%w: %s", ErrScenarioNotFound, scenarioID)
		}
		return nil, fmt.Errorf("failed to get scenario: %w", err)
	}

	// Check if container exists and get its status
	containerExists, err := m.Docker.ContainerExists(ctx, scenario.ContainerID)
	if err != nil {
		log.Printf("[scenario] failed to check container existence: %v", err)
		// Return database status if we can't check container
		return &types.ScenarioStatusResponse{
			ScenarioID:   scenario.ScenarioID,
			UserID:       scenario.UserID,
			ScenarioType: scenario.ScenarioType,
			ContainerID:  scenario.ContainerID,
			Status:       scenario.Status,
			Message:      "Container status unavailable",
		}, nil
	}

	if !containerExists {
		// Container doesn't exist, update status to stopped
		scenario.Status = "stopped"
		scenario.UpdatedAt = time.Now()
		if err := storage.UpdateScenario(ctx, m.DB, scenario); err != nil {
			log.Printf("[scenario] failed to update scenario status: %v", err)
		}

		return &types.ScenarioStatusResponse{
			ScenarioID:      scenario.ScenarioID,
			UserID:          scenario.UserID,
			ScenarioType:    scenario.ScenarioType,
			ContainerID:     scenario.ContainerID,
			Status:          "stopped",
			ContainerStatus: "not_found",
			Message:         "Container no longer exists",
		}, nil
	}

	// Get container status from Docker
	containerStatus, err := m.Docker.GetContainerStatus(ctx, scenario.ContainerID)
	if err != nil {
		log.Printf("[scenario] failed to get container status: %v", err)
		// Return database status if we can't get container status
		return &types.ScenarioStatusResponse{
			ScenarioID:      scenario.ScenarioID,
			UserID:          scenario.UserID,
			ScenarioType:    scenario.ScenarioType,
			ContainerID:     scenario.ContainerID,
			Status:          scenario.Status,
			ContainerStatus: "unknown",
			Message:         "Container status unavailable",
		}, nil
	}

	// Update status based on container state
	status := scenario.Status
	if containerStatus == "running" && scenario.Status == "provisioning" {
		status = "running"
		scenario.Status = "running"
		scenario.UpdatedAt = time.Now()
		if err := storage.UpdateScenario(ctx, m.DB, scenario); err != nil {
			log.Printf("[scenario] failed to update scenario status: %v", err)
		}
	} else if containerStatus == "exited" || containerStatus == "stopped" {
		status = "stopped"
		scenario.Status = "stopped"
		scenario.UpdatedAt = time.Now()
		if err := storage.UpdateScenario(ctx, m.DB, scenario); err != nil {
			log.Printf("[scenario] failed to update scenario status: %v", err)
		}
	}

	log.Printf("[scenario] scenario %s status: %s (container: %s)", scenarioID, status, containerStatus)

	return &types.ScenarioStatusResponse{
		ScenarioID:      scenario.ScenarioID,
		UserID:          scenario.UserID,
		ScenarioType:    scenario.ScenarioType,
		ContainerID:     scenario.ContainerID,
		Status:          status,
		ContainerStatus: containerStatus,
		Message:         "Scenario status retrieved successfully",
	}, nil
}

func (m *Manager) GetTerminalURL(ctx context.Context, scenarioID string) (string, error) {
	if ctx == nil {
		return "", errors.New("nil context provided")
	}

	if scenarioID == "" {
		return "", fmt.Errorf("%w: scenario ID cannot be empty", ErrInvalidScenarioID)
	}

	log.Printf("[scenario] getting terminal URL for scenario: %s", scenarioID)

	// Get scenario from database
	scenario, err := storage.GetScenario(ctx, m.DB, scenarioID)
	if err != nil {
		log.Printf("[scenario] failed to get scenario from DB: %v", err)
		if errors.Is(err, storage.ErrScenarioNotFound) {
			return "", fmt.Errorf("%w: %s", ErrScenarioNotFound, scenarioID)
		}
		return "", fmt.Errorf("failed to get scenario: %w", err)
	}

	// Check if scenario is running
	if scenario.Status != "running" {
		return "", fmt.Errorf("%w: scenario status is %s", ErrScenarioNotRunning, scenario.Status)
	}

	// Check if container exists
	containerExists, err := m.Docker.ContainerExists(ctx, scenario.ContainerID)
	if err != nil {
		log.Printf("[scenario] failed to check container existence: %v", err)
		return "", fmt.Errorf("failed to verify container: %w", err)
	}

	if !containerExists {
		return "", fmt.Errorf("%w: container %s not found", ErrScenarioNotRunning, scenario.ContainerID)
	}

	// Get terminal URL from Docker
	terminalURL, err := m.Docker.GetTerminalURL(ctx, scenario.ContainerID)
	if err != nil {
		log.Printf("[scenario] failed to get terminal URL: %v", err)
		return "", fmt.Errorf("failed to get terminal URL: %w", err)
	}

	log.Printf("[scenario] terminal URL for scenario %s: %s", scenarioID, terminalURL)
	return terminalURL, nil
}

func (m *Manager) StopScenario(ctx context.Context, scenarioID string) error {
	if ctx == nil {
		return errors.New("nil context provided")
	}

	if scenarioID == "" {
		return fmt.Errorf("%w: scenario ID cannot be empty", ErrInvalidScenarioID)
	}

	log.Printf("[scenario] stopping scenario: %s", scenarioID)

	// Get scenario from database
	scenario, err := storage.GetScenario(ctx, m.DB, scenarioID)
	if err != nil {
		log.Printf("[scenario] failed to get scenario from DB: %v", err)
		if errors.Is(err, storage.ErrScenarioNotFound) {
			return fmt.Errorf("%w: %s", ErrScenarioNotFound, scenarioID)
		}
		return fmt.Errorf("failed to get scenario: %w", err)
	}

	// Stop the container
	if err := m.Docker.StopContainer(ctx, scenario.ContainerID); err != nil {
		log.Printf("[scenario] failed to stop container %s: %v", scenario.ContainerID, err)
		// Don't return error if container is already stopped
		if !errors.Is(err, docker.ErrContainerNotFound) {
			return fmt.Errorf("failed to stop container: %w", err)
		}
	}

	// Update scenario status
	scenario.Status = "stopped"
	scenario.UpdatedAt = time.Now()
	if err := storage.UpdateScenario(ctx, m.DB, scenario); err != nil {
		log.Printf("[scenario] failed to update scenario status: %v", err)
		return fmt.Errorf("failed to update scenario status: %w", err)
	}

	log.Printf("[scenario] scenario %s stopped successfully", scenarioID)
	return nil
}

func (m *Manager) GetDirectoryStructure(ctx context.Context, scenarioID string) (*types.DirectoryStructureResponse, error) {
	if ctx == nil {
		return nil, errors.New("nil context provided")
	}

	if scenarioID == "" {
		return nil, fmt.Errorf("%w: scenario ID cannot be empty", ErrInvalidScenarioID)
	}

	log.Printf("[scenario] getting directory structure for scenario: %s", scenarioID)

	// Get scenario from database
	scenario, err := storage.GetScenario(ctx, m.DB, scenarioID)
	if err != nil {
		log.Printf("[scenario] failed to get scenario from DB: %v", err)
		if errors.Is(err, storage.ErrScenarioNotFound) {
			return nil, fmt.Errorf("%w: %s", ErrScenarioNotFound, scenarioID)
		}
		return nil, fmt.Errorf("failed to get scenario: %w", err)
	}

	// Check if container exists and is running
	containerExists, err := m.Docker.ContainerExists(ctx, scenario.ContainerID)
	if err != nil {
		log.Printf("[scenario] failed to check container existence: %v", err)
		return nil, fmt.Errorf("failed to check container existence: %w", err)
	}

	if !containerExists {
		return nil, fmt.Errorf("%w: container %s", ErrScenarioNotRunning, scenario.ContainerID)
	}

	// Execute command to get directory structure
	// We'll use a simple find command to get the file tree
	command := []string{"find", "/home/devlab", "-type", "f", "-o", "-type", "d", "-printf", "%p %y\n"}
	output, err := m.Docker.ExecuteCommand(ctx, scenario.ContainerID, command)
	if err != nil {
		log.Printf("[scenario] failed to execute directory structure command: %v", err)
		return nil, fmt.Errorf("failed to get directory structure: %w", err)
	}

	// Parse the output and build the file tree structure
	structure, err := parseDirectoryStructure(output)
	if err != nil {
		log.Printf("[scenario] failed to parse directory structure: %v", err)
		return nil, fmt.Errorf("failed to parse directory structure: %w", err)
	}

	log.Printf("[scenario] successfully retrieved directory structure for scenario %s", scenarioID)

	return &types.DirectoryStructureResponse{
		ScenarioID: scenarioID,
		Path:       "/home/devlab",
		Structure:  structure,
		Message:    "Directory structure retrieved successfully",
	}, nil
}

// parseDirectoryStructure parses the output of the find command and builds a file tree
func parseDirectoryStructure(output string) ([]types.FileNode, error) {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) == 0 {
		return []types.FileNode{}, nil
	}

	var structure []types.FileNode
	pathMap := make(map[string]*types.FileNode)

	// First pass: create all nodes
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) != 2 {
			continue
		}

		path := parts[0]
		fileType := parts[1]

		// Skip if not under /home/devlab
		if !strings.HasPrefix(path, "/home/devlab") {
			continue
		}

		// Skip cache directories to reduce response size
		if shouldSkipPath(path) {
			continue
		}

		node := &types.FileNode{
			Path:     path,
			Type:     getNodeType(fileType),
			IsRoot:   path == "/home/devlab",
			Children: []string{},
			IsOpen:   false,
			IsSaved:  true,
		}

		pathMap[path] = node
		structure = append(structure, *node)
	}

	// Second pass: build parent-child relationships
	for path := range pathMap {
		if path == "/home/devlab" {
			continue // Root node
		}

		parentPath := getParentPath(path)
		if parent, exists := pathMap[parentPath]; exists {
			parent.Children = append(parent.Children, path)
		}
	}

	return structure, nil
}

// getNodeType converts the find command type to our type
func getNodeType(findType string) string {
	switch findType {
	case "d":
		return "folder"
	case "f":
		return "file"
	default:
		return "file"
	}
}

// getParentPath returns the parent directory path
func getParentPath(path string) string {
	dir := filepath.Dir(path)
	if dir == "." {
		return "/home/devlab"
	}
	return dir
}

// shouldSkipPath determines if a path should be excluded from directory structure
func shouldSkipPath(path string) bool {
	// Skip cache directories to reduce response size
	cachePatterns := []string{
		"/home/devlab/.cache",
		"/home/devlab/go/pkg/mod",
		"/home/devlab/.config",
		"/home/devlab/.local",
		"/home/devlab/.npm",
		"/home/devlab/.pip",
		"/home/devlab/.conda",
		"/home/devlab/.m2",
		"/home/devlab/.gradle",
		"/home/devlab/.ivy2",
		"/home/devlab/.sbt",
		"/home/devlab/.cargo",
		"/home/devlab/.rustup",
		"/home/devlab/.node_modules",
		"/home/devlab/.yarn",
		"/home/devlab/.bundle",
		"/home/devlab/.gem",
		"/home/devlab/.pub-cache",
		"/home/devlab/.dart",
		"/home/devlab/.flutter",
	}

	for _, pattern := range cachePatterns {
		if strings.HasPrefix(path, pattern) {
			return true
		}
	}

	return false
}
