package scenario

import (
	"context"
	"devlab/internal/config"
	"devlab/internal/docker"
	"devlab/internal/storage"
	"devlab/internal/types"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
)

// IntegrationTestSuite runs tests with real Docker and MongoDB
type IntegrationTestSuite struct {
	cfg    *config.Config
	db     *mongo.Database
	client *mongo.Client
	docker docker.Client
}

func NewIntegrationTestSuite(t *testing.T) *IntegrationTestSuite {
	// Use test database
	cfg := &config.Config{
		MongoURI: "mongodb://localhost:27017",
		DBName:   "devlab_test",
	}

	client, err := storage.GetMongoClient(context.Background(), cfg.MongoURI)
	if err != nil {
		t.Skipf("MongoDB not available for integration test: %v", err)
	}

	db := client.Database(cfg.DBName)

	// Clean up test database before each test
	err = db.Drop(context.Background())
	if err != nil {
		t.Logf("Warning: could not drop test database: %v", err)
	}

	return &IntegrationTestSuite{
		cfg:    cfg,
		db:     db,
		client: client,
		docker: docker.RealClient{},
	}
}

func (suite *IntegrationTestSuite) Cleanup() {
	if suite.client != nil {
		suite.client.Disconnect(context.Background())
	}
}

func TestScenarioIntegration_FullWorkflow(t *testing.T) {
	suite := NewIntegrationTestSuite(t)
	defer suite.Cleanup()

	mgr := NewManager(suite.cfg, suite.db, suite.docker)

	// Test successful scenario creation
	t.Run("successful_scenario_creation", func(t *testing.T) {
		req := &types.StartScenarioRequest{
			UserID:       "test-user-1",
			ScenarioType: "go",
			Script:       "echo 'Hello from integration test'",
		}

		resp, err := mgr.StartScenario(context.Background(), req)
		if err != nil {
			t.Fatalf("Failed to start scenario: %v", err)
		}

		if resp.ScenarioID == "" {
			t.Error("Expected scenario ID, got empty string")
		}

		if resp.Status != "provisioning" {
			t.Errorf("Expected status 'provisioning', got '%s'", resp.Status)
		}

		// Verify scenario was stored in database
		var scenario storage.Scenario
		err = suite.db.Collection("scenarios").FindOne(context.Background(), map[string]string{
			"scenario_id": resp.ScenarioID,
		}).Decode(&scenario)

		if err != nil {
			t.Fatalf("Failed to find scenario in database: %v", err)
		}

		if scenario.UserID != req.UserID {
			t.Errorf("Expected user ID %s, got %s", req.UserID, scenario.UserID)
		}

		if scenario.ScenarioType != req.ScenarioType {
			t.Errorf("Expected scenario type %s, got %s", req.ScenarioType, scenario.ScenarioType)
		}
	})

	// Test concurrent scenario creation
	t.Run("concurrent_scenario_creation", func(t *testing.T) {
		const numScenarios = 5
		results := make(chan error, numScenarios)

		for i := 0; i < numScenarios; i++ {
			go func(id int) {
				req := &types.StartScenarioRequest{
					UserID:       "test-user-concurrent",
					ScenarioType: "go",
					Script:       "echo 'Concurrent test'",
				}

				_, err := mgr.StartScenario(context.Background(), req)
				results <- err
			}(i)
		}

		// Wait for all scenarios to complete
		for i := 0; i < numScenarios; i++ {
			if err := <-results; err != nil {
				t.Errorf("Concurrent scenario %d failed: %v", i, err)
			}
		}

		// Verify all scenarios were created
		count, err := suite.db.Collection("scenarios").CountDocuments(context.Background(), map[string]string{
			"user_id": "test-user-concurrent",
		})

		if err != nil {
			t.Fatalf("Failed to count scenarios: %v", err)
		}

		if count != int64(numScenarios) {
			t.Errorf("Expected %d scenarios, got %d", numScenarios, count)
		}
	})
}

func TestScenarioIntegration_TimeoutHandling(t *testing.T) {
	suite := NewIntegrationTestSuite(t)
	defer suite.Cleanup()

	mgr := NewManager(suite.cfg, suite.db, suite.docker)

	t.Run("context_timeout", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer cancel()

		req := &types.StartScenarioRequest{
			UserID:       "test-user-timeout",
			ScenarioType: "go",
			Script:       "sleep 10",
		}

		_, err := mgr.StartScenario(ctx, req)
		if err == nil {
			t.Error("Expected timeout error, got nil")
		}
	})
}

func TestScenarioIntegration_ErrorScenarios(t *testing.T) {
	suite := NewIntegrationTestSuite(t)
	defer suite.Cleanup()

	mgr := NewManager(suite.cfg, suite.db, suite.docker)

	t.Run("invalid_scenario_type", func(t *testing.T) {
		req := &types.StartScenarioRequest{
			UserID:       "test-user-invalid",
			ScenarioType: "invalid-type",
			Script:       "echo 'test'",
		}

		resp, err := mgr.StartScenario(context.Background(), req)
		// Should still work but use default image
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if resp.ScenarioID == "" {
			t.Error("Expected scenario ID even for invalid type")
		}
	})

	t.Run("empty_user_id", func(t *testing.T) {
		req := &types.StartScenarioRequest{
			UserID:       "",
			ScenarioType: "go",
			Script:       "echo 'test'",
		}

		resp, err := mgr.StartScenario(context.Background(), req)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if resp.ScenarioID == "" {
			t.Error("Expected scenario ID even for empty user ID")
		}
	})
}
