package cleanup

import (
	"context"
	"devlab/internal/config"
	"devlab/internal/docker"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockDockerClient is a mock implementation of the docker.Client interface
type MockDockerClient struct {
	mock.Mock
}

func (m *MockDockerClient) StartScenarioContainer(ctx context.Context, scenarioType, script string) (string, int, error) {
	args := m.Called(ctx, scenarioType, script)
	return args.String(0), args.Int(1), args.Error(2)
}

func (m *MockDockerClient) GetContainerStatus(ctx context.Context, containerID string) (string, error) {
	args := m.Called(ctx, containerID)
	return args.String(0), args.Error(1)
}

func (m *MockDockerClient) GetTerminalURL(ctx context.Context, containerID string) (string, error) {
	args := m.Called(ctx, containerID)
	return args.String(0), args.Error(1)
}

func (m *MockDockerClient) StopContainer(ctx context.Context, containerID string) error {
	args := m.Called(ctx, containerID)
	return args.Error(0)
}

func (m *MockDockerClient) ContainerExists(ctx context.Context, containerID string) (bool, error) {
	args := m.Called(ctx, containerID)
	return args.Bool(0), args.Error(1)
}

func (m *MockDockerClient) ExecuteCommand(ctx context.Context, containerID string, command []string) (string, error) {
	args := m.Called(ctx, containerID, command)
	return args.String(0), args.Error(1)
}

func (m *MockDockerClient) ListContainers(ctx context.Context) ([]docker.ContainerInfo, error) {
	args := m.Called(ctx)
	return args.Get(0).([]docker.ContainerInfo), args.Error(1)
}

func (m *MockDockerClient) RemoveContainer(ctx context.Context, containerID string) error {
	args := m.Called(ctx, containerID)
	return args.Error(0)
}

func TestCleanupManager_isScenarioContainer(t *testing.T) {
	// Setup
	cfg := &config.Config{}
	mockDocker := &MockDockerClient{}
	cleanupManager := NewCleanupManager(cfg, nil, mockDocker)

	scenarioContainers := map[string]bool{
		"container-1": true,
		"container-2": true,
		"container-3": false,
	}

	// Test
	assert.True(t, cleanupManager.isScenarioContainer("container-1", scenarioContainers))
	assert.True(t, cleanupManager.isScenarioContainer("container-2", scenarioContainers))
	assert.False(t, cleanupManager.isScenarioContainer("container-3", scenarioContainers))
	assert.False(t, cleanupManager.isScenarioContainer("container-4", scenarioContainers))
}

func TestCleanupManager_Configuration(t *testing.T) {
	// Test configuration loading
	cfg := &config.Config{
		Cleanup: config.CleanupConfig{
			MaxScenarioAge:  2 * time.Hour,
			CleanupInterval: 30 * time.Minute,
			EnableCleanup:   true,
		},
	}

	assert.Equal(t, 2*time.Hour, cfg.Cleanup.MaxScenarioAge)
	assert.Equal(t, 30*time.Minute, cfg.Cleanup.CleanupInterval)
	assert.True(t, cfg.Cleanup.EnableCleanup)
}

func TestCleanupManager_NewCleanupManager(t *testing.T) {
	// Test cleanup manager creation
	cfg := &config.Config{
		Cleanup: config.CleanupConfig{
			MaxScenarioAge: 1 * time.Hour,
		},
	}

	mockDocker := &MockDockerClient{}

	// This should not panic even with nil database
	cleanupManager := NewCleanupManager(cfg, nil, mockDocker)
	assert.NotNil(t, cleanupManager)
	assert.Equal(t, cfg, cleanupManager.cfg)
	assert.Equal(t, mockDocker, cleanupManager.docker)
}

func TestCleanupManager_OrphanedContainerDetection(t *testing.T) {
	// Test orphaned container detection logic
	scenarioContainers := map[string]bool{
		"container-1": true,
		"container-2": true,
	}

	// Test containers
	testCases := []struct {
		containerID string
		isOrphaned  bool
	}{
		{"container-1", false}, // Associated with scenario
		{"container-2", false}, // Associated with scenario
		{"orphaned-1", true},   // Not associated
		{"orphaned-2", true},   // Not associated
	}

	cfg := &config.Config{}
	mockDocker := &MockDockerClient{}
	cleanupManager := NewCleanupManager(cfg, nil, mockDocker)

	for _, tc := range testCases {
		isOrphaned := !cleanupManager.isScenarioContainer(tc.containerID, scenarioContainers)
		assert.Equal(t, tc.isOrphaned, isOrphaned,
			"Container %s should be orphaned: %v", tc.containerID, tc.isOrphaned)
	}
}
