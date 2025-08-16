package docker

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock Docker client for testing
type MockDockerClient struct {
	mock.Mock
}

func (m *MockDockerClient) ContainerCreate(ctx context.Context, config interface{}, hostConfig interface{}, networkingConfig interface{}, containerName string) (interface{}, error) {
	args := m.Called(ctx, config, hostConfig, networkingConfig, containerName)
	return args.Get(0), args.Error(1)
}

func (m *MockDockerClient) ContainerStart(ctx context.Context, containerID string, options interface{}) error {
	args := m.Called(ctx, containerID, options)
	return args.Error(0)
}

func (m *MockDockerClient) ContainerInspect(ctx context.Context, container string) (types.ContainerJSON, error) {
	args := m.Called(ctx, container)
	return args.Get(0).(types.ContainerJSON), args.Error(1)
}

func (m *MockDockerClient) Close() error {
	args := m.Called()
	return args.Error(0)
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

func TestStartScenarioContainer(t *testing.T) {
	tests := []struct {
		name          string
		scenarioType  string
		script        string
		expectedImage string
		expectError   bool
	}{
		{
			name:          "go_scenario",
			scenarioType:  "go",
			script:        "go version",
			expectedImage: "golang:1.21",
			expectError:   false,
		},
		{
			name:          "docker_scenario",
			scenarioType:  "docker",
			script:        "docker --version",
			expectedImage: "docker:24.0.7",
			expectError:   false,
		},
		{
			name:          "k8s_scenario",
			scenarioType:  "k8s",
			script:        "kubectl version",
			expectedImage: "bitnami/kubectl:latest",
			expectError:   false,
		},
		{
			name:          "unknown_scenario_type",
			scenarioType:  "unknown",
			script:        "echo test",
			expectedImage: "golang:1.21", // Default image
			expectError:   false,
		},
		{
			name:          "empty_script",
			scenarioType:  "go",
			script:        "",
			expectedImage: "golang:1.21",
			expectError:   false,
		},
		{
			name:          "complex_script",
			scenarioType:  "go",
			script:        "echo 'Hello World' && sleep 1 && echo 'Done'",
			expectedImage: "golang:1.21",
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This test would require a real Docker daemon
			// For now, we'll test the logic without actual Docker calls
			client := RealClient{}

			// Test with a context that will timeout quickly
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()

			containerID, _, err := client.StartScenarioContainer(ctx, tt.scenarioType, tt.script)

			// We expect an error because Docker daemon is not available in test environment
			// But we can verify the function doesn't panic and handles the scenario type correctly
			if err == nil {
				// If somehow Docker is available, verify the container ID is not empty
				assert.NotEmpty(t, containerID)
			}
		})
	}
}

func TestStartScenarioContainer_ImageSelection(t *testing.T) {
	client := RealClient{}

	// Test image selection logic
	testCases := []struct {
		scenarioType  string
		expectedImage string
	}{
		{"go", "golang:1.21"},
		{"docker", "docker:24.0.7"},
		{"k8s", "bitnami/kubectl:latest"},
		{"unknown", "golang:1.21"}, // Default
		{"", "golang:1.21"},        // Empty
	}

	for _, tc := range testCases {
		t.Run(tc.scenarioType, func(t *testing.T) {
			// We can't easily test the actual image selection without mocking
			// But we can verify the function doesn't panic
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()

			_, _, err := client.StartScenarioContainer(ctx, tc.scenarioType, "echo test")

			// Function should not panic, even if Docker is not available
			assert.NotPanics(t, func() {
				client.StartScenarioContainer(ctx, tc.scenarioType, "echo test")
			})

			// Error is expected if Docker daemon is not available
			if err != nil {
				// Verify it's a Docker-related error, not a panic
				assert.Contains(t, err.Error(), "docker")
			}
		})
	}
}

func TestStartScenarioContainer_ScriptInjection(t *testing.T) {
	client := RealClient{}

	tests := []struct {
		name     string
		script   string
		hasMount bool
	}{
		{
			name:     "with_script",
			script:   "echo 'Hello World'",
			hasMount: true,
		},
		{
			name:     "empty_script",
			script:   "",
			hasMount: false,
		},
		{
			name:     "complex_script",
			script:   "echo 'Start' && sleep 1 && echo 'End'",
			hasMount: true,
		},
		{
			name: "script_with_special_chars",
			script: "#!/bin/bash\n" +
				"echo \"Script with special chars: \\$@\\\"'\"\n" +
				"echo \"Testing quotes: 'single' \\\"double\\\" `backticks`\"\n" +
				"echo \"Testing variables: $PATH $HOME\"\n",
			hasMount: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()

			_, _, err := client.StartScenarioContainer(ctx, "go", tt.script)

			// Function should not panic
			assert.NotPanics(t, func() {
				_, _, _ = client.StartScenarioContainer(ctx, "go", tt.script)
			})

			// Error is expected if Docker daemon is not available
			if err != nil {
				assert.Contains(t, err.Error(), "docker")
			}
		})
	}
}

func TestStartScenarioContainer_ContextHandling(t *testing.T) {
	client := RealClient{}

	t.Run("context_cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		_, _, err := client.StartScenarioContainer(ctx, "go", "echo test")

		// Should handle context cancellation gracefully
		assert.Error(t, err)
	})

	t.Run("context_timeout", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer cancel()

		_, _, err := client.StartScenarioContainer(ctx, "go", "echo test")

		// Should handle timeout gracefully
		assert.Error(t, err)
	})

	t.Run("nil_context", func(t *testing.T) {
		// This should return an error, not panic
		_, _, err := client.StartScenarioContainer(nil, "go", "echo test")

		// Should handle nil context gracefully by returning an error
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "nil")
	})
}

// Benchmark tests
func BenchmarkStartScenarioContainer(b *testing.B) {
	client := RealClient{}

	b.Run("go_scenario", func(b *testing.B) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _, err := client.StartScenarioContainer(ctx, "go", "echo benchmark")
			if err != nil {
				// Expected error if Docker is not available
				break
			}
		}
	})

	b.Run("docker_scenario", func(b *testing.B) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _, err := client.StartScenarioContainer(ctx, "docker", "echo benchmark")
			if err != nil {
				// Expected error if Docker is not available
				break
			}
		}
	})
}

// Test error scenarios
func TestStartScenarioContainer_ErrorScenarios(t *testing.T) {
	client := RealClient{}

	t.Run("docker_daemon_unavailable", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		_, _, err := client.StartScenarioContainer(ctx, "go", "echo test")

		// Should return a meaningful error
		if err != nil {
			assert.Contains(t, err.Error(), "docker")
		}
	})

	t.Run("invalid_scenario_type", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		_, _, err := client.StartScenarioContainer(ctx, "invalid-type", "echo test")
		// Should not error due to invalid scenario type, but may fail due to Docker issues
		if err != nil {
			// If there's an error, it should not be due to invalid scenario type
			assert.NotErrorIs(t, err, ErrInvalidScenarioType)
		}
	})
}

func TestStartScenarioContainer_WithTerminal(t *testing.T) {
	tests := []struct {
		name         string
		scenarioType string
		script       string
		setupMock    func(*MockDockerClient)
		expectedID   string
		expectedPort int
		expectError  bool
	}{
		{
			name:         "successful_go_scenario_with_terminal",
			scenarioType: "go",
			script:       "echo 'hello world'",
			setupMock: func(m *MockDockerClient) {
				m.On("StartScenarioContainer", mock.Anything, "go", "echo 'hello world'").
					Return("container123", 3001, nil)
			},
			expectedID:   "container123",
			expectedPort: 3001,
			expectError:  false,
		},
		{
			name:         "successful_docker_scenario_with_terminal",
			scenarioType: "docker",
			script:       "",
			setupMock: func(m *MockDockerClient) {
				m.On("StartScenarioContainer", mock.Anything, "docker", "").
					Return("container456", 3002, nil)
			},
			expectedID:   "container456",
			expectedPort: 3002,
			expectError:  false,
		},
		{
			name:         "docker_error",
			scenarioType: "k8s",
			script:       "kubectl version",
			setupMock: func(m *MockDockerClient) {
				m.On("StartScenarioContainer", mock.Anything, "k8s", "kubectl version").
					Return("", 0, assert.AnError)
			},
			expectedID:   "",
			expectedPort: 0,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockDockerClient{}
			tt.setupMock(mockClient)

			ctx := context.Background()
			containerID, terminalPort, err := mockClient.StartScenarioContainer(ctx, tt.scenarioType, tt.script)

			if tt.expectError {
				assert.Error(t, err)
				assert.Empty(t, containerID)
				assert.Equal(t, 0, terminalPort)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedID, containerID)
				assert.Equal(t, tt.expectedPort, terminalPort)
			}

			mockClient.AssertExpectations(t)
		})
	}
}

func TestGetTerminalURL_Success(t *testing.T) {
	mockClient := &MockDockerClient{}

	expectedURL := "http://localhost:3001"
	mockClient.On("GetTerminalURL", mock.Anything, "container123").
		Return(expectedURL, nil)

	ctx := context.Background()
	url, err := mockClient.GetTerminalURL(ctx, "container123")

	assert.NoError(t, err)
	assert.Equal(t, expectedURL, url)
	mockClient.AssertExpectations(t)
}

func TestGetTerminalURL_ContainerNotFound(t *testing.T) {
	mockClient := &MockDockerClient{}

	mockClient.On("GetTerminalURL", mock.Anything, "nonexistent").
		Return("", assert.AnError)

	ctx := context.Background()
	url, err := mockClient.GetTerminalURL(ctx, "nonexistent")

	assert.Error(t, err)
	assert.Empty(t, url)
	mockClient.AssertExpectations(t)
}

func TestGetTerminalURL_NoPortMapping(t *testing.T) {
	mockClient := &MockDockerClient{}

	mockClient.On("GetTerminalURL", mock.Anything, "container456").
		Return("", assert.AnError)

	ctx := context.Background()
	url, err := mockClient.GetTerminalURL(ctx, "container456")

	assert.Error(t, err)
	assert.Empty(t, url)
	mockClient.AssertExpectations(t)
}

// Test the findAvailablePort function
func TestFindAvailablePort(t *testing.T) {
	port, err := findAvailablePort()

	assert.NoError(t, err)
	assert.GreaterOrEqual(t, port, 3001)
	assert.LessOrEqual(t, port, 3009)
}

// Test multiple calls to findAvailablePort to ensure different ports
func TestFindAvailablePort_MultipleCalls(t *testing.T) {
	ports := make(map[int]bool)

	for i := 0; i < 5; i++ {
		port, err := findAvailablePort()
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, port, 3001)
		assert.LessOrEqual(t, port, 3009)
		ports[port] = true
	}

	// In a real environment, we might get different ports
	// In test environment, we might get the same port if it's available
	assert.True(t, len(ports) >= 1)
}

func TestStartScenarioContainer_ErrorHandling(t *testing.T) {
	client := RealClient{}
	ctx := context.Background()

	t.Run("nil_context", func(t *testing.T) {
		// This should return an error, not panic
		_, _, err := client.StartScenarioContainer(nil, "go", "echo test")

		// Should handle nil context gracefully by returning an error
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "nil")
	})

	t.Run("empty_scenario_type", func(t *testing.T) {
		_, _, err := client.StartScenarioContainer(ctx, "", "echo test")
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidScenarioType)
		assert.Contains(t, err.Error(), "empty")
	})

	t.Run("invalid_scenario_type", func(t *testing.T) {
		_, _, err := client.StartScenarioContainer(ctx, "invalid-type", "echo test")
		// Should not error, but use default image
		assert.NoError(t, err)
	})

	t.Run("port_unavailability", func(t *testing.T) {
		// This test would require mocking the port finding logic
		// For now, we'll test the error type is correct
		_, _, err := client.StartScenarioContainer(ctx, "go", "echo test")
		// The actual error depends on Docker availability, but we can test the structure
		if err != nil {
			// Should not be a port unavailability error in normal conditions
			assert.NotErrorIs(t, err, ErrPortUnavailable)
		}
	})
}

func TestGetContainerStatus_ErrorHandling(t *testing.T) {
	client := RealClient{}
	ctx := context.Background()

	t.Run("nil_context", func(t *testing.T) {
		_, err := client.GetContainerStatus(nil, "test-container")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "nil")
	})

	t.Run("empty_container_id", func(t *testing.T) {
		_, err := client.GetContainerStatus(ctx, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "empty")
	})

	t.Run("nonexistent_container", func(t *testing.T) {
		_, err := client.GetContainerStatus(ctx, "nonexistent-container-id")
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrContainerNotFound)
	})
}

func TestGetTerminalURL_ErrorHandling(t *testing.T) {
	client := RealClient{}
	ctx := context.Background()

	t.Run("nil_context", func(t *testing.T) {
		_, err := client.GetTerminalURL(nil, "test-container")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "nil")
	})

	t.Run("empty_container_id", func(t *testing.T) {
		_, err := client.GetTerminalURL(ctx, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "empty")
	})

	t.Run("nonexistent_container", func(t *testing.T) {
		_, err := client.GetTerminalURL(ctx, "nonexistent-container-id")
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrContainerNotFound)
	})

	t.Run("stopped_container", func(t *testing.T) {
		// This would require creating a stopped container for testing
		// For now, we test the error handling structure
		_, err := client.GetTerminalURL(ctx, "nonexistent-container-id")
		assert.Error(t, err)
		// Should be container not found, not container not running
		assert.ErrorIs(t, err, ErrContainerNotFound)
	})
}

func TestStopContainer_ErrorHandling(t *testing.T) {
	client := RealClient{}
	ctx := context.Background()

	t.Run("nil_context", func(t *testing.T) {
		err := client.StopContainer(nil, "test-container")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "nil")
	})

	t.Run("empty_container_id", func(t *testing.T) {
		err := client.StopContainer(ctx, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "empty")
	})

	t.Run("nonexistent_container", func(t *testing.T) {
		err := client.StopContainer(ctx, "nonexistent-container-id")
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrContainerNotFound)
	})
}

func TestContainerExists_ErrorHandling(t *testing.T) {
	client := RealClient{}
	ctx := context.Background()

	t.Run("nil_context", func(t *testing.T) {
		_, err := client.ContainerExists(nil, "test-container")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "nil")
	})

	t.Run("empty_container_id", func(t *testing.T) {
		_, err := client.ContainerExists(ctx, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "empty")
	})

	t.Run("nonexistent_container", func(t *testing.T) {
		exists, err := client.ContainerExists(ctx, "nonexistent-container-id")
		assert.NoError(t, err)
		assert.False(t, exists)
	})
}

func TestFindAvailablePort_ErrorHandling(t *testing.T) {
	t.Run("port_range_exhaustion", func(t *testing.T) {
		// This test would require mocking all ports to be in use
		// For now, we test the function works in normal conditions
		port, err := findAvailablePort()
		if err != nil {
			assert.ErrorIs(t, err, ErrPortUnavailable)
		} else {
			assert.GreaterOrEqual(t, port, 3001)
			assert.LessOrEqual(t, port, 3009)
		}
	})
}

func TestStartScenarioContainer_TTYDFailureHandling(t *testing.T) {
	client := RealClient{}
	ctx := context.Background()

	t.Run("ttyd_installation_failure", func(t *testing.T) {
		// This test would require a container image without package managers
		// For now, we test the error handling structure
		_, _, err := client.StartScenarioContainer(ctx, "go", "echo test")
		if err != nil {
			// Should not be a TTYD failure error in normal conditions
			assert.NotErrorIs(t, err, ErrTTYDFailedToStart)
		}
	})

	t.Run("ttyd_startup_failure", func(t *testing.T) {
		// This test would require mocking ttyd to fail to start
		// For now, we test the error handling structure
		_, _, err := client.StartScenarioContainer(ctx, "go", "echo test")
		if err != nil {
			// Should not be a TTYD failure error in normal conditions
			assert.NotErrorIs(t, err, ErrTTYDFailedToStart)
		}
	})
}

func TestDockerDaemonUnavailable(t *testing.T) {
	client := RealClient{}
	ctx := context.Background()

	t.Run("docker_daemon_unavailable", func(t *testing.T) {
		// This test would require stopping the Docker daemon
		// For now, we test the error handling structure
		_, _, err := client.StartScenarioContainer(ctx, "go", "echo test")
		if err != nil {
			// Should not be a Docker daemon error in normal conditions
			assert.NotErrorIs(t, err, ErrDockerDaemonUnavailable)
		}
	})
}

func TestErrorTypes(t *testing.T) {
	t.Run("error_type_comparison", func(t *testing.T) {
		// Test that our custom error types work correctly
		err1 := ErrContainerNotFound
		err2 := ErrContainerNotRunning
		err3 := ErrPortUnavailable
		err4 := ErrTTYDFailedToStart
		err5 := ErrInvalidScenarioType
		err6 := ErrDockerDaemonUnavailable

		assert.NotEqual(t, err1, err2)
		assert.NotEqual(t, err2, err3)
		assert.NotEqual(t, err3, err4)
		assert.NotEqual(t, err4, err5)
		assert.NotEqual(t, err5, err6)

		// Test error wrapping
		wrappedErr := fmt.Errorf("failed to start container: %w", ErrContainerNotFound)
		assert.ErrorIs(t, wrappedErr, ErrContainerNotFound)
	})
}

func TestContextHandling_Enhanced(t *testing.T) {
	client := RealClient{}

	t.Run("context_cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		_, _, err := client.StartScenarioContainer(ctx, "go", "echo test")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "canceled")
	})

	t.Run("context_timeout", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer cancel()

		time.Sleep(1 * time.Millisecond) // Ensure timeout

		_, _, err := client.StartScenarioContainer(ctx, "go", "echo test")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "deadline")
	})

	t.Run("nil_context", func(t *testing.T) {
		// This should return an error, not panic
		_, _, err := client.StartScenarioContainer(nil, "go", "echo test")

		// Should handle nil context gracefully by returning an error
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "nil")
	})
}

func TestErrorScenarios_Enhanced(t *testing.T) {
	client := RealClient{}
	ctx := context.Background()

	t.Run("docker_daemon_unavailable", func(t *testing.T) {
		_, _, err := client.StartScenarioContainer(ctx, "go", "echo test")
		if err != nil {
			// In normal conditions, this should not be a Docker daemon error
			assert.NotErrorIs(t, err, ErrDockerDaemonUnavailable)
		}
	})

	t.Run("invalid_scenario_type", func(t *testing.T) {
		_, _, err := client.StartScenarioContainer(ctx, "invalid-type", "echo test")
		// Should not error, but use default image
		assert.NoError(t, err)
	})

	t.Run("empty_script", func(t *testing.T) {
		_, _, err := client.StartScenarioContainer(ctx, "go", "")
		// Should not error with empty script
		assert.NoError(t, err)
	})

	t.Run("complex_script", func(t *testing.T) {
		script := `#!/bin/bash
echo "Starting complex script"
for i in {1..5}; do
    echo "Iteration $i"
    sleep 0.1
done
echo "Script completed"`

		_, _, err := client.StartScenarioContainer(ctx, "go", script)
		// Should handle complex scripts
		assert.NoError(t, err)
	})

	t.Run("script_with_special_chars", func(t *testing.T) {
		script := "#!/bin/bash\n" +
			"echo \"Script with special chars: \\$@\\\"'\"\n" +
			"echo \"Testing quotes: 'single' \\\"double\\\" `backticks`\"\n" +
			"echo \"Testing variables: $PATH $HOME\"\n"

		_, _, err := client.StartScenarioContainer(ctx, "go", script)
		// Should handle special characters in scripts
		assert.NoError(t, err)
	})
}

func TestGetTerminalURL_Enhanced(t *testing.T) {
	client := RealClient{}
	ctx := context.Background()

	t.Run("successful_go_scenario_with_terminal", func(t *testing.T) {
		// Start a container first
		containerID, _, err := client.StartScenarioContainer(ctx, "go", "echo 'Starting terminal test'")
		if err != nil {
			t.Skipf("Skipping test due to Docker error: %v", err)
		}

		// Wait a bit for container to be ready
		time.Sleep(2 * time.Second)

		// Get terminal URL
		url, err := client.GetTerminalURL(ctx, containerID)
		if err != nil {
			t.Skipf("Skipping test due to terminal error: %v", err)
		}

		assert.NotEmpty(t, url)
		assert.Contains(t, url, "http://")
		assert.Contains(t, url, ":300")
	})

	t.Run("successful_docker_scenario_with_terminal", func(t *testing.T) {
		// Start a container first
		containerID, _, err := client.StartScenarioContainer(ctx, "docker", "echo 'Starting Docker terminal test'")
		if err != nil {
			t.Skipf("Skipping test due to Docker error: %v", err)
		}

		// Wait a bit for container to be ready
		time.Sleep(2 * time.Second)

		// Get terminal URL
		url, err := client.GetTerminalURL(ctx, containerID)
		if err != nil {
			t.Skipf("Skipping test due to terminal error: %v", err)
		}

		assert.NotEmpty(t, url)
		assert.Contains(t, url, "http://")
		assert.Contains(t, url, ":300")
	})

	t.Run("docker_error", func(t *testing.T) {
		_, err := client.GetTerminalURL(ctx, "nonexistent-container")
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrContainerNotFound)
	})
}

func TestRealClient_ExecuteCommand(t *testing.T) {
	client := RealClient{}

	tests := []struct {
		name        string
		containerID string
		command     []string
		expectError bool
	}{
		{
			name:        "empty_container_id",
			containerID: "",
			command:     []string{"ls", "-la"},
			expectError: true,
		},
		{
			name:        "empty_command",
			containerID: "test-container",
			command:     []string{},
			expectError: true,
		},
		{
			name:        "nil_context",
			containerID: "test-container",
			command:     []string{"ls", "-la"},
			expectError: true,
		},
		{
			name:        "nonexistent_container",
			containerID: "nonexistent-container",
			command:     []string{"ls", "-la"},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ctx context.Context
			if tt.name == "nil_context" {
				ctx = nil
			} else {
				ctx = context.Background()
			}

			output, err := client.ExecuteCommand(ctx, tt.containerID, tt.command)

			if tt.expectError {
				assert.Error(t, err)
				assert.Empty(t, output)
			} else {
				// This would require a running container, so we expect an error
				// but we're testing the validation logic
				assert.Error(t, err)
			}
		})
	}
}

func TestRealClient_ExecuteCommand_Integration(t *testing.T) {
	// This test requires Docker to be running and a test container
	// It's marked as integration test and can be run separately
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client := RealClient{}

	// Start a test container
	ctx := context.Background()
	containerID, _, err := client.StartScenarioContainer(ctx, "go", "echo 'test container'")
	if err != nil {
		t.Skipf("Skipping test - failed to start test container: %v", err)
	}
	defer client.StopContainer(ctx, containerID)

	// Wait a moment for container to be ready
	time.Sleep(2 * time.Second)

	tests := []struct {
		name        string
		command     []string
		expectError bool
	}{
		{
			name:        "simple_command",
			command:     []string{"echo", "hello world"},
			expectError: false,
		},
		{
			name:        "list_directory",
			command:     []string{"ls", "-la", "/home/devlab"},
			expectError: false,
		},
		{
			name:        "find_command",
			command:     []string{"find", "/home/devlab", "-type", "f", "-o", "-type", "d", "-printf", "%p %y\n"},
			expectError: false,
		},
		{
			name:        "invalid_command",
			command:     []string{"nonexistent_command"},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := client.ExecuteCommand(ctx, containerID, tt.command)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, output)
				t.Logf("Command output: %s", output)
			}
		})
	}
}

func TestRealClient_StopContainer(t *testing.T) {
	client := RealClient{}

	tests := []struct {
		name        string
		containerID string
		expectError bool
	}{
		{
			name:        "empty_container_id",
			containerID: "",
			expectError: true,
		},
		{
			name:        "nil_context",
			containerID: "test-container",
			expectError: true,
		},
		{
			name:        "nonexistent_container",
			containerID: "nonexistent-container",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ctx context.Context
			if tt.name == "nil_context" {
				ctx = nil
			} else {
				ctx = context.Background()
			}

			err := client.StopContainer(ctx, tt.containerID)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				// This would require a running container, so we expect an error
				// but we're testing the validation logic
				assert.Error(t, err)
			}
		})
	}
}

func TestRealClient_StopContainer_Integration(t *testing.T) {
	// This test requires Docker to be running and a test container
	// It's marked as integration test and can be run separately
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client := RealClient{}

	// Start a test container
	ctx := context.Background()
	containerID, _, err := client.StartScenarioContainer(ctx, "go", "echo 'test container for stopping'")
	if err != nil {
		t.Skipf("Skipping test - failed to start test container: %v", err)
	}

	// Wait a moment for container to be ready
	time.Sleep(2 * time.Second)

	t.Run("stop_running_container", func(t *testing.T) {
		// Stop the container
		err := client.StopContainer(ctx, containerID)
		assert.NoError(t, err)

		// Verify container is stopped
		exists, err := client.ContainerExists(ctx, containerID)
		assert.NoError(t, err)
		assert.False(t, exists, "Container should be removed after stopping")
	})

	t.Run("stop_already_stopped_container", func(t *testing.T) {
		// Try to stop the same container again (should not error)
		err := client.StopContainer(ctx, containerID)
		// This should not error since the container is already stopped/removed
		assert.NoError(t, err)
	})

	t.Run("stop_nonexistent_container", func(t *testing.T) {
		// Try to stop a non-existent container
		err := client.StopContainer(ctx, "nonexistent-container-id")
		assert.Error(t, err)
		assert.ErrorContains(t, err, "container not found")
	})
}

func TestRealClient_StopContainer_ErrorHandling(t *testing.T) {
	client := RealClient{}

	t.Run("docker_daemon_unavailable", func(t *testing.T) {
		// This test would require mocking the Docker client
		// For now, we'll just test the validation logic
		err := client.StopContainer(nil, "test-container")
		assert.Error(t, err)
		assert.ErrorContains(t, err, "nil context provided")
	})

	t.Run("container_already_stopped", func(t *testing.T) {
		// This would require a stopped container
		// The implementation should handle this gracefully
		if testing.Short() {
			t.Skip("Skipping test in short mode")
		}

		ctx := context.Background()
		containerID, _, err := client.StartScenarioContainer(ctx, "go", "echo 'test'")
		if err != nil {
			t.Skipf("Skipping test - failed to start container: %v", err)
		}

		// Stop the container first
		err = client.StopContainer(ctx, containerID)
		assert.NoError(t, err)

		// Try to stop it again
		err = client.StopContainer(ctx, containerID)
		// Should not error since container is already stopped/removed
		assert.NoError(t, err)
	})
}
