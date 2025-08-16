package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	// JWT token for integration tests (generated with scripts/generate_token.go)
	testJWTToken = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3NTQ3OTY1NDMsImlhdCI6MTc1NDcxMDE0MywiaXNzIjoiZGV2bGFiIiwidXNlcl9pZCI6ImRlbW8tdXNlciJ9.VhAaUdUZCBIP6Tdm0L2KqN10FzYJbwOU1mD8egkuasw"
	apiBaseURL   = "http://localhost:8000"
)

// TestCompleteWorkflow tests the complete DevLab workflow from start to finish
func TestCompleteWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	// Step 1: Start DevLab stack
	t.Log("Starting DevLab stack...")
	err := startDevLabStack()
	require.NoError(t, err, "Failed to start DevLab stack")
	defer stopDevLabStack()

	// Step 2: Wait for services to be ready
	t.Log("Waiting for services to be ready...")
	err = waitForServices()
	require.NoError(t, err, "Services not ready")

	// Step 3: Test API health
	t.Log("Testing API health...")
	err = testAPIHealth()
	require.NoError(t, err, "API health check failed")

	// Step 4: Create a scenario
	t.Log("Creating a test scenario...")
	scenarioID, err := createTestScenario()
	require.NoError(t, err, "Failed to create test scenario")
	require.NotEmpty(t, scenarioID, "Scenario ID should not be empty")

	// Step 5: Wait for scenario to be running
	t.Log("Waiting for scenario to be running...")
	err = waitForScenarioRunning(scenarioID)
	require.NoError(t, err, "Scenario failed to start")

	// Step 6: Get scenario status
	t.Log("Getting scenario status...")
	status, err := getScenarioStatus(scenarioID)
	require.NoError(t, err, "Failed to get scenario status")
	assert.Equal(t, "running", status, "Scenario should be running")

	// Step 7: Get terminal URL
	t.Log("Getting terminal URL...")
	terminalURL, err := getTerminalURL(scenarioID)
	require.NoError(t, err, "Failed to get terminal URL")
	assert.NotEmpty(t, terminalURL, "Terminal URL should not be empty")

	// Step 8: Get directory structure
	t.Log("Getting directory structure...")
	structure, err := getDirectoryStructure(scenarioID)
	require.NoError(t, err, "Failed to get directory structure")
	assert.NotEmpty(t, structure, "Directory structure should not be empty")

	// Step 9: Stop scenario
	t.Log("Stopping scenario...")
	err = stopScenario(scenarioID)
	require.NoError(t, err, "Failed to stop scenario")

	// Step 10: Verify scenario is stopped
	t.Log("Verifying scenario is stopped...")
	status, err = getScenarioStatus(scenarioID)
	require.NoError(t, err, "Failed to get scenario status")
	assert.Equal(t, "stopped", status, "Scenario should be stopped")

	t.Log("Complete workflow test passed!")
}

// TestDockerComposeIntegration tests the Docker Compose setup
func TestDockerComposeIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	// Test Docker Compose up
	t.Log("Testing Docker Compose up...")
	err := testDockerComposeUp()
	require.NoError(t, err, "Docker Compose up failed")

	// Test all services are running
	t.Log("Testing all services are running...")
	services := []string{"devlab-api", "devlab-worker", "devlab-mongo", "devlab-rabbitmq"}
	for _, service := range services {
		running, err := isServiceRunning(service)
		require.NoError(t, err, "Failed to check service: %s", service)
		assert.True(t, running, "Service %s should be running", service)
	}

	// Test service connectivity
	t.Log("Testing service connectivity...")
	err = testServiceConnectivity()
	require.NoError(t, err, "Service connectivity test failed")

	// Test Docker Compose down
	t.Log("Testing Docker Compose down...")
	err = testDockerComposeDown()
	require.NoError(t, err, "Docker Compose down failed")

	t.Log("Docker Compose integration test passed!")
}

// TestEndToEndScenario tests a complete scenario lifecycle
func TestEndToEndScenario(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	// Start stack
	err := startDevLabStack()
	require.NoError(t, err, "Failed to start DevLab stack")
	defer stopDevLabStack()

	// Wait for services
	err = waitForServices()
	require.NoError(t, err, "Services not ready")

	// Test different scenario types
	scenarioTypes := []string{"go", "docker", "k8s", "python"}
	for _, scenarioType := range scenarioTypes {
		t.Run("scenario_"+scenarioType, func(t *testing.T) {
			t.Logf("Testing %s scenario...", scenarioType)

			// Create scenario
			scenarioID, err := createScenarioWithType(scenarioType)
			require.NoError(t, err, "Failed to create %s scenario", scenarioType)
			require.NotEmpty(t, scenarioID, "Scenario ID should not be empty")

			// Wait for running
			err = waitForScenarioRunning(scenarioID)
			require.NoError(t, err, "%s scenario failed to start", scenarioType)

			// Verify status
			status, err := getScenarioStatus(scenarioID)
			require.NoError(t, err, "Failed to get %s scenario status", scenarioType)
			assert.Equal(t, "running", status, "%s scenario should be running", scenarioType)

			// Get terminal URL
			terminalURL, err := getTerminalURL(scenarioID)
			require.NoError(t, err, "Failed to get %s scenario terminal URL", scenarioType)
			assert.NotEmpty(t, terminalURL, "Terminal URL should not be empty")

			// Stop scenario
			err = stopScenario(scenarioID)
			require.NoError(t, err, "Failed to stop %s scenario", scenarioType)

			// Verify stopped
			status, err = getScenarioStatus(scenarioID)
			require.NoError(t, err, "Failed to get %s scenario status", scenarioType)
			assert.Equal(t, "stopped", status, "%s scenario should be stopped", scenarioType)

			t.Logf("%s scenario test passed!", scenarioType)
		})
	}
}

// TestServiceDependencies tests service dependency health
func TestServiceDependencies(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	// Start stack
	err := startDevLabStack()
	require.NoError(t, err, "Failed to start DevLab stack")
	defer stopDevLabStack()

	// Test MongoDB connectivity
	t.Log("Testing MongoDB connectivity...")
	err = testMongoDBConnectivity()
	require.NoError(t, err, "MongoDB connectivity test failed")

	// Test RabbitMQ connectivity
	t.Log("Testing RabbitMQ connectivity...")
	err = testRabbitMQConnectivity()
	require.NoError(t, err, "RabbitMQ connectivity test failed")

	// Test Docker connectivity
	t.Log("Testing Docker connectivity...")
	err = testDockerConnectivity()
	require.NoError(t, err, "Docker connectivity test failed")

	// Test API service health
	t.Log("Testing API service health...")
	err = testAPIServiceHealth()
	require.NoError(t, err, "API service health test failed")

	// Test Worker service health
	t.Log("Testing Worker service health...")
	err = testWorkerServiceHealth()
	require.NoError(t, err, "Worker service health test failed")

	t.Log("Service dependencies test passed!")
}

// Helper functions

func startDevLabStack() error {
	cmd := exec.Command("docker-compose", "up", "-d")
	cmd.Dir = "../../"
	return cmd.Run()
}

func stopDevLabStack() error {
	cmd := exec.Command("docker-compose", "down")
	cmd.Dir = "../../"
	return cmd.Run()
}

func waitForServices() error {
	// Wait for services to be ready
	time.Sleep(10 * time.Second)
	return nil
}

func testAPIHealth() error {
	resp, err := http.Get(apiBaseURL + "/healthz")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API health check failed with status: %d", resp.StatusCode)
	}
	return nil
}

func createTestScenario() (string, error) {
	requestBody := map[string]interface{}{
		"user_id":       "integration-test-user",
		"scenario_type": "go",
		"script":        "echo 'Hello from integration test'",
	}

	body, _ := json.Marshal(requestBody)
	req, err := http.NewRequest("POST", apiBaseURL+"/scenarios/start", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+testJWTToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to create scenario, status: %d", resp.StatusCode)
	}

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return "", err
	}

	scenarioID, ok := response["scenario_id"].(string)
	if !ok {
		return "", fmt.Errorf("scenario_id not found in response")
	}

	return scenarioID, nil
}

func createScenarioWithType(scenarioType string) (string, error) {
	requestBody := map[string]interface{}{
		"user_id":       "integration-test-user",
		"scenario_type": scenarioType,
	}

	body, _ := json.Marshal(requestBody)
	req, err := http.NewRequest("POST", apiBaseURL+"/scenarios/start", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+testJWTToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to create %s scenario, status: %d", scenarioType, resp.StatusCode)
	}

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return "", err
	}

	scenarioID, ok := response["scenario_id"].(string)
	if !ok {
		return "", fmt.Errorf("scenario_id not found in response")
	}

	return scenarioID, nil
}

func waitForScenarioRunning(scenarioID string) error {
	// Wait up to 60 seconds for scenario to be running
	for i := 0; i < 60; i++ {
		status, err := getScenarioStatus(scenarioID)
		if err != nil {
			return err
		}
		if status == "running" {
			return nil
		}
		time.Sleep(1 * time.Second)
	}
	return fmt.Errorf("scenario %s did not reach running state", scenarioID)
}

func getScenarioStatus(scenarioID string) (string, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf(apiBaseURL+"/scenarios/%s/status", scenarioID), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+testJWTToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to get scenario status, status: %d", resp.StatusCode)
	}

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return "", err
	}

	status, ok := response["status"].(string)
	if !ok {
		return "", fmt.Errorf("status not found in response")
	}

	return status, nil
}

func getTerminalURL(scenarioID string) (string, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf(apiBaseURL+"/scenarios/%s/terminal", scenarioID), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+testJWTToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to get terminal URL, status: %d", resp.StatusCode)
	}

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return "", err
	}

	url, ok := response["url"].(string)
	if !ok {
		return "", fmt.Errorf("url not found in response")
	}

	return url, nil
}

func getDirectoryStructure(scenarioID string) (string, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf(apiBaseURL+"/scenarios/%s/directory", scenarioID), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+testJWTToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to get directory structure, status: %d", resp.StatusCode)
	}

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return "", err
	}

	structure, ok := response["structure"].([]interface{})
	if !ok {
		return "", fmt.Errorf("structure not found in response")
	}

	return fmt.Sprintf("%v", structure), nil
}

func stopScenario(scenarioID string) error {
	req, err := http.NewRequest("DELETE", fmt.Sprintf(apiBaseURL+"/scenarios/%s", scenarioID), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+testJWTToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to stop scenario, status: %d", resp.StatusCode)
	}

	return nil
}

func testDockerComposeUp() error {
	cmd := exec.Command("docker-compose", "up", "-d")
	cmd.Dir = "../../"
	return cmd.Run()
}

func testDockerComposeDown() error {
	cmd := exec.Command("docker-compose", "down")
	cmd.Dir = "../../"
	return cmd.Run()
}

func isServiceRunning(serviceName string) (bool, error) {
	cmd := exec.Command("docker-compose", "ps", "-q", serviceName)
	cmd.Dir = "../../"
	output, err := cmd.Output()
	if err != nil {
		return false, err
	}
	return len(output) > 0, nil
}

func testServiceConnectivity() error {
	// Test API connectivity
	resp, err := http.Get(apiBaseURL + "/healthz")
	if err != nil {
		return fmt.Errorf("API connectivity failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API health check failed with status: %d", resp.StatusCode)
	}

	return nil
}

func testMongoDBConnectivity() error {
	// Test MongoDB connection
	cmd := exec.Command("docker", "exec", "devlab-mongo", "mongosh", "--eval", "db.runCommand('ping')")
	return cmd.Run()
}

func testRabbitMQConnectivity() error {
	// Test RabbitMQ connection
	cmd := exec.Command("docker", "exec", "devlab-rabbitmq", "rabbitmq-diagnostics", "ping")
	return cmd.Run()
}

func testDockerConnectivity() error {
	// Test Docker daemon connectivity
	cmd := exec.Command("docker", "version")
	return cmd.Run()
}

func testAPIServiceHealth() error {
	// Test API service is responding
	resp, err := http.Get(apiBaseURL + "/healthz")
	if err != nil {
		return fmt.Errorf("API service health check failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API service health check failed with status: %d", resp.StatusCode)
	}

	return nil
}

func testWorkerServiceHealth() error {
	// Test worker service is running
	cmd := exec.Command("docker", "exec", "devlab-worker", "ps", "aux")
	return cmd.Run()
}
