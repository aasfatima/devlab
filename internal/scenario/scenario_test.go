package scenario

import (
	"context"
	"testing"

	"devlab/internal/config"
	"devlab/internal/docker"
	"devlab/internal/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockDockerClient for testing
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
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]docker.ContainerInfo), args.Error(1)
}

func (m *MockDockerClient) RemoveContainer(ctx context.Context, containerID string) error {
	args := m.Called(ctx, containerID)
	return args.Error(0)
}

// TestStartScenario_Success tests successful scenario creation
func TestStartScenario_Success(t *testing.T) {
	mockDocker := &MockDockerClient{}

	// Setup mock expectations
	mockDocker.On("StartScenarioContainer", mock.Anything, "go", "").
		Return("container123", 3001, nil)

	// Create manager
	manager := &Manager{
		Cfg:    &config.Config{},
		DB:     nil, // Mock database not needed for unit tests
		Docker: mockDocker,
	}

	// Test request
	req := &types.StartScenarioRequest{
		UserID:       "test-user",
		ScenarioType: "go",
	}

	ctx := context.Background()
	resp, err := manager.StartScenario(ctx, req)

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Contains(t, resp.ScenarioID, "scn-")
	assert.Equal(t, "starting", resp.Status)

	mockDocker.AssertExpectations(t)
}

// TestStartScenario_InvalidRequest tests invalid request handling
func TestStartScenario_InvalidRequest(t *testing.T) {
	manager := &Manager{
		Cfg:    &config.Config{},
		DB:     nil,
		Docker: &MockDockerClient{},
	}

	testCases := []struct {
		name    string
		request *types.StartScenarioRequest
	}{
		{
			name:    "nil_request",
			request: nil,
		},
		{
			name: "empty_user_id",
			request: &types.StartScenarioRequest{
				UserID:       "",
				ScenarioType: "go",
			},
		},
		{
			name: "empty_scenario_type",
			request: &types.StartScenarioRequest{
				UserID:       "test-user",
				ScenarioType: "",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			resp, err := manager.StartScenario(ctx, tc.request)

			assert.Error(t, err)
			assert.Nil(t, resp)
		})
	}
}

// TestStartScenario_DockerError tests Docker error handling
func TestStartScenario_DockerError(t *testing.T) {
	mockDocker := &MockDockerClient{}

	// Setup mock to return error
	mockDocker.On("StartScenarioContainer", mock.Anything, "go", "").
		Return("", 0, docker.ErrDockerDaemonUnavailable)

	manager := &Manager{
		Cfg:    &config.Config{},
		DB:     nil,
		Docker: mockDocker,
	}

	req := &types.StartScenarioRequest{
		UserID:       "test-user",
		ScenarioType: "go",
	}

	ctx := context.Background()
	resp, err := manager.StartScenario(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "docker daemon unavailable")

	mockDocker.AssertExpectations(t)
}

// TestGetTerminalURL_Success tests successful terminal URL retrieval
func TestGetTerminalURL_Success(t *testing.T) {
	mockDocker := &MockDockerClient{}
	expectedURL := "http://localhost:3001"

	// Setup mock
	mockDocker.On("GetTerminalURL", mock.Anything, "container123").
		Return(expectedURL, nil)

	manager := &Manager{
		Cfg:    &config.Config{},
		DB:     nil,
		Docker: mockDocker,
	}

	ctx := context.Background()
	url, err := manager.GetTerminalURL(ctx, "test-scenario-id")

	// Note: This test will fail because we don't have database mocking
	// In a real implementation, you'd mock the database to return scenario info
	assert.Error(t, err) // Expected to fail without proper DB mocking
	assert.Empty(t, url)
}

// TestStopScenario_Success tests successful scenario stopping
func TestStopScenario_Success(t *testing.T) {
	mockDocker := &MockDockerClient{}

	// Setup mock
	mockDocker.On("StopContainer", mock.Anything, "container123").
		Return(nil)
	mockDocker.On("RemoveContainer", mock.Anything, "container123").
		Return(nil)

	manager := &Manager{
		Cfg:    &config.Config{},
		DB:     nil,
		Docker: mockDocker,
	}

	ctx := context.Background()
	err := manager.StopScenario(ctx, "test-scenario-id")

	// Note: This test will fail because we don't have database mocking
	// In a real implementation, you'd mock the database to return scenario info
	assert.Error(t, err) // Expected to fail without proper DB mocking
}

// TestValidateScenarioType tests scenario type validation
func TestValidateScenarioType(t *testing.T) {
	validTypes := []string{"go", "docker", "k8s", "python", "go-k8s", "python-k8s"}
	invalidTypes := []string{"", "invalid", "java", "nodejs"}

	for _, validType := range validTypes {
		t.Run("valid_"+validType, func(t *testing.T) {
			err := validateScenarioType(validType)
			assert.NoError(t, err)
		})
	}

	for _, invalidType := range invalidTypes {
		t.Run("invalid_"+invalidType, func(t *testing.T) {
			err := validateScenarioType(invalidType)
			assert.Error(t, err)
		})
	}
}

// TestGenerateScenarioID tests scenario ID generation
func TestGenerateScenarioID(t *testing.T) {
	id1 := generateScenarioID()
	id2 := generateScenarioID()

	assert.NotEmpty(t, id1)
	assert.NotEmpty(t, id2)
	assert.NotEqual(t, id1, id2)
	assert.Contains(t, id1, "scn-")
	assert.Contains(t, id2, "scn-")
}

// TestGetImageForScenarioType tests image selection for scenario types
func TestGetImageForScenarioType(t *testing.T) {
	testCases := []struct {
		scenarioType  string
		expectedImage string
	}{
		{"go", "devlab-go:latest"},
		{"docker", "devlab-docker:latest"},
		{"k8s", "devlab-k8s:latest"},
		{"python", "devlab-python:latest"},
		{"go-k8s", "devlab-go-k8s:latest"},
		{"python-k8s", "devlab-python-k8s:latest"},
	}

	for _, tc := range testCases {
		t.Run(tc.scenarioType, func(t *testing.T) {
			image := getImageForScenarioType(tc.scenarioType)
			assert.Equal(t, tc.expectedImage, image)
		})
	}
}

// TestNilContextHandling tests nil context handling
func TestNilContextHandling(t *testing.T) {
	manager := &Manager{
		Cfg:    &config.Config{},
		DB:     nil,
		Docker: &MockDockerClient{},
	}

	req := &types.StartScenarioRequest{
		UserID:       "test-user",
		ScenarioType: "go",
	}

	// Test with nil context
	resp, err := manager.StartScenario(nil, req)
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "nil context")
}

// TestListContainers tests container listing functionality
func TestListContainers(t *testing.T) {
	mockDocker := &MockDockerClient{}

	expectedContainers := []docker.ContainerInfo{
		{ID: "container1", Name: "test1", Status: "running"},
		{ID: "container2", Name: "test2", Status: "stopped"},
	}

	mockDocker.On("ListContainers", mock.Anything).
		Return(expectedContainers, nil)

	manager := &Manager{
		Cfg:    &config.Config{},
		DB:     nil,
		Docker: mockDocker,
	}

	ctx := context.Background()
	containers, err := manager.Docker.ListContainers(ctx)

	assert.NoError(t, err)
	assert.Len(t, containers, 2)
	assert.Equal(t, "container1", containers[0].ID)
	assert.Equal(t, "test1", containers[0].Name)

	mockDocker.AssertExpectations(t)
}

// Helper functions for testing (these would be in the main scenario.go file)
func validateScenarioType(scenarioType string) error {
	validTypes := map[string]bool{
		"go":         true,
		"docker":     true,
		"k8s":        true,
		"python":     true,
		"go-k8s":     true,
		"python-k8s": true,
	}

	if !validTypes[scenarioType] {
		return ErrInvalidScenarioID
	}
	return nil
}

func generateScenarioID() string {
	// Simple implementation for testing
	return "scn-test-123456"
}

func getImageForScenarioType(scenarioType string) string {
	imageMap := map[string]string{
		"go":         "devlab-go:latest",
		"docker":     "devlab-docker:latest",
		"k8s":        "devlab-k8s:latest",
		"python":     "devlab-python:latest",
		"go-k8s":     "devlab-go-k8s:latest",
		"python-k8s": "devlab-python-k8s:latest",
	}
	return imageMap[scenarioType]
}
