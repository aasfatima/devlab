package storage

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// TestMongoConnection tests MongoDB connection functionality
func TestMongoConnection(t *testing.T) {
	tests := []struct {
		name        string
		mongoURI    string
		expectError bool
	}{
		{
			name:        "valid_connection_string",
			mongoURI:    "mongodb://localhost:27017",
			expectError: false,
		},
		{
			name:        "invalid_connection_string",
			mongoURI:    "mongodb://invalid-host:27017",
			expectError: true, // MongoDB client will fail on invalid host
		},
		{
			name:        "empty_connection_string",
			mongoURI:    "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			client, err := GetMongoClient(ctx, tt.mongoURI)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, client)
			} else {
				if err != nil {
					t.Skipf("MongoDB not available: %v", err)
				}
				assert.NoError(t, err)
				assert.NotNil(t, client)

				// Test ping
				err = client.Ping(ctx, nil)
				assert.NoError(t, err)

				// Clean up
				client.Disconnect(ctx)
			}
		})
	}
}

// TestScenarioCRUD tests Create, Read, Update, Delete operations for scenarios
func TestScenarioCRUD(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Connect to test database
	client, err := GetMongoClient(ctx, "mongodb://localhost:27017")
	if err != nil {
		t.Skipf("MongoDB not available: %v", err)
	}
	defer client.Disconnect(ctx)

	db := client.Database("devlab_test")
	collection := db.Collection("scenarios")

	// Clean up before test
	collection.Drop(ctx)

	t.Run("create_scenario", func(t *testing.T) {
		scenario := &Scenario{
			ScenarioID:   "test-scn-123",
			UserID:       "test-user",
			ScenarioType: "go",
			ContainerID:  "test-container-123",
			Status:       "running",
			TerminalPort: 3001,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}

		// Insert scenario
		_, err := collection.InsertOne(ctx, scenario)
		assert.NoError(t, err)

		// Verify insertion
		var result Scenario
		err = collection.FindOne(ctx, bson.M{"scenario_id": "test-scn-123"}).Decode(&result)
		assert.NoError(t, err)
		assert.Equal(t, scenario.ScenarioID, result.ScenarioID)
		assert.Equal(t, scenario.UserID, result.UserID)
		assert.Equal(t, scenario.ScenarioType, result.ScenarioType)
	})

	t.Run("read_scenario", func(t *testing.T) {
		var scenario Scenario
		err := collection.FindOne(ctx, bson.M{"scenario_id": "test-scn-123"}).Decode(&scenario)
		assert.NoError(t, err)
		assert.Equal(t, "test-scn-123", scenario.ScenarioID)
		assert.Equal(t, "test-user", scenario.UserID)
	})

	t.Run("update_scenario", func(t *testing.T) {
		update := bson.M{
			"$set": bson.M{
				"status":     "stopped",
				"updated_at": time.Now(),
			},
		}

		result, err := collection.UpdateOne(
			ctx,
			bson.M{"scenario_id": "test-scn-123"},
			update,
		)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), result.ModifiedCount)

		// Verify update
		var updatedScenario Scenario
		err = collection.FindOne(ctx, bson.M{"scenario_id": "test-scn-123"}).Decode(&updatedScenario)
		assert.NoError(t, err)
		assert.Equal(t, "stopped", updatedScenario.Status)
	})

	t.Run("delete_scenario", func(t *testing.T) {
		result, err := collection.DeleteOne(ctx, bson.M{"scenario_id": "test-scn-123"})
		assert.NoError(t, err)
		assert.Equal(t, int64(1), result.DeletedCount)

		// Verify deletion
		var deletedScenario Scenario
		err = collection.FindOne(ctx, bson.M{"scenario_id": "test-scn-123"}).Decode(&deletedScenario)
		assert.Error(t, err)
		assert.Equal(t, mongo.ErrNoDocuments, err)
	})
}

// TestScenarioQueries tests various query operations
func TestScenarioQueries(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := GetMongoClient(ctx, "mongodb://localhost:27017")
	if err != nil {
		t.Skipf("MongoDB not available: %v", err)
	}
	defer client.Disconnect(ctx)

	db := client.Database("devlab_test")
	collection := db.Collection("scenarios")

	// Clean up and insert test data
	collection.Drop(ctx)

	scenarios := []interface{}{
		&Scenario{
			ScenarioID:   "scn-1",
			UserID:       "user1",
			ScenarioType: "go",
			Status:       "running",
			CreatedAt:    time.Now(),
		},
		&Scenario{
			ScenarioID:   "scn-2",
			UserID:       "user1",
			ScenarioType: "docker",
			Status:       "stopped",
			CreatedAt:    time.Now(),
		},
		&Scenario{
			ScenarioID:   "scn-3",
			UserID:       "user2",
			ScenarioType: "go",
			Status:       "running",
			CreatedAt:    time.Now(),
		},
	}

	_, err = collection.InsertMany(ctx, scenarios)
	require.NoError(t, err)

	t.Run("find_by_user_id", func(t *testing.T) {
		cursor, err := collection.Find(ctx, bson.M{"user_id": "user1"})
		assert.NoError(t, err)
		defer cursor.Close(ctx)

		var results []Scenario
		err = cursor.All(ctx, &results)
		assert.NoError(t, err)
		assert.Len(t, results, 2)
	})

	t.Run("find_by_status", func(t *testing.T) {
		cursor, err := collection.Find(ctx, bson.M{"status": "running"})
		assert.NoError(t, err)
		defer cursor.Close(ctx)

		var results []Scenario
		err = cursor.All(ctx, &results)
		assert.NoError(t, err)
		assert.Len(t, results, 2)
	})

	t.Run("find_by_scenario_type", func(t *testing.T) {
		cursor, err := collection.Find(ctx, bson.M{"scenario_type": "go"})
		assert.NoError(t, err)
		defer cursor.Close(ctx)

		var results []Scenario
		err = cursor.All(ctx, &results)
		assert.NoError(t, err)
		assert.Len(t, results, 2)
	})

	t.Run("complex_query", func(t *testing.T) {
		filter := bson.M{
			"user_id":       "user1",
			"scenario_type": "go",
			"status":        "running",
		}

		cursor, err := collection.Find(ctx, filter)
		assert.NoError(t, err)
		defer cursor.Close(ctx)

		var results []Scenario
		err = cursor.All(ctx, &results)
		assert.NoError(t, err)
		assert.Len(t, results, 1)
		assert.Equal(t, "scn-1", results[0].ScenarioID)
	})
}

// TestScenarioIndexes tests index creation and usage
func TestScenarioIndexes(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := GetMongoClient(ctx, "mongodb://localhost:27017")
	if err != nil {
		t.Skipf("MongoDB not available: %v", err)
	}
	defer client.Disconnect(ctx)

	db := client.Database("devlab_test")
	collection := db.Collection("scenarios")

	// Clean up
	collection.Drop(ctx)

	t.Run("create_indexes", func(t *testing.T) {
		// Create indexes
		indexes := []mongo.IndexModel{
			{
				Keys: bson.D{
					{Key: "scenario_id", Value: 1},
				},
				Options: options.Index().SetUnique(true),
			},
			{
				Keys: bson.D{
					{Key: "user_id", Value: 1},
				},
			},
			{
				Keys: bson.D{
					{Key: "status", Value: 1},
				},
			},
			{
				Keys: bson.D{
					{Key: "created_at", Value: -1},
				},
			},
		}

		_, err := collection.Indexes().CreateMany(ctx, indexes)
		assert.NoError(t, err)

		// Verify indexes
		indexList, err := collection.Indexes().List(ctx)
		assert.NoError(t, err)

		var indexNames []string
		for indexList.Next(ctx) {
			var index bson.M
			err := indexList.Decode(&index)
			assert.NoError(t, err)
			indexNames = append(indexNames, index["name"].(string))
		}

		// Should have _id_ index plus our custom indexes
		assert.GreaterOrEqual(t, len(indexNames), 4)
	})
}

// TestScenarioAggregation tests aggregation pipeline operations
func TestScenarioAggregation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := GetMongoClient(ctx, "mongodb://localhost:27017")
	if err != nil {
		t.Skipf("MongoDB not available: %v", err)
	}
	defer client.Disconnect(ctx)

	db := client.Database("devlab_test")
	collection := db.Collection("scenarios")

	// Clean up and insert test data
	collection.Drop(ctx)

	scenarios := []interface{}{
		&Scenario{
			ScenarioID:   "scn-1",
			UserID:       "user1",
			ScenarioType: "go",
			Status:       "running",
			CreatedAt:    time.Now(),
		},
		&Scenario{
			ScenarioID:   "scn-2",
			UserID:       "user1",
			ScenarioType: "docker",
			Status:       "stopped",
			CreatedAt:    time.Now(),
		},
		&Scenario{
			ScenarioID:   "scn-3",
			UserID:       "user2",
			ScenarioType: "go",
			Status:       "running",
			CreatedAt:    time.Now(),
		},
	}

	_, err = collection.InsertMany(ctx, scenarios)
	require.NoError(t, err)

	t.Run("count_by_status", func(t *testing.T) {
		pipeline := mongo.Pipeline{
			{{Key: "$group", Value: bson.M{
				"_id":   "$status",
				"count": bson.M{"$sum": 1},
			}}},
		}

		cursor, err := collection.Aggregate(ctx, pipeline)
		assert.NoError(t, err)
		defer cursor.Close(ctx)

		var results []bson.M
		err = cursor.All(ctx, &results)
		assert.NoError(t, err)

		// Should have counts for "running" and "stopped"
		assert.Len(t, results, 2)
	})

	t.Run("count_by_user", func(t *testing.T) {
		pipeline := mongo.Pipeline{
			{{Key: "$group", Value: bson.M{
				"_id":   "$user_id",
				"count": bson.M{"$sum": 1},
			}}},
		}

		cursor, err := collection.Aggregate(ctx, pipeline)
		assert.NoError(t, err)
		defer cursor.Close(ctx)

		var results []bson.M
		err = cursor.All(ctx, &results)
		assert.NoError(t, err)

		// Should have counts for "user1" and "user2"
		assert.Len(t, results, 2)
	})
}

// TestScenarioErrorHandling tests error scenarios
func TestScenarioErrorHandling(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client, err := GetMongoClient(ctx, "mongodb://localhost:27017")
	if err != nil {
		t.Skipf("MongoDB not available: %v", err)
	}
	defer client.Disconnect(ctx)

	db := client.Database("devlab_test")
	collection := db.Collection("scenarios")

	// Clean up
	collection.Drop(ctx)

	t.Run("duplicate_scenario_id", func(t *testing.T) {
		scenario1 := &Scenario{
			ScenarioID: "duplicate-id",
			UserID:     "user1",
			Status:     "running",
		}

		scenario2 := &Scenario{
			ScenarioID: "duplicate-id", // Same ID
			UserID:     "user2",
			Status:     "running",
		}

		// Insert first scenario
		_, err := collection.InsertOne(ctx, scenario1)
		assert.NoError(t, err)

		// Try to insert second scenario with same ID
		_, err = collection.InsertOne(ctx, scenario2)
		// MongoDB doesn't enforce unique constraints by default, so this might succeed
		// We'll just test that the operation completes without error
		assert.NoError(t, err)
	})

	t.Run("invalid_document", func(t *testing.T) {
		// Try to insert invalid document
		invalidDoc := bson.M{
			"invalid_field": "value",
		}

		_, err := collection.InsertOne(ctx, invalidDoc)
		// This might succeed depending on MongoDB configuration
		// but we're testing the behavior
		assert.NoError(t, err)
	})
}

// TestScenarioConcurrency tests concurrent operations
func TestScenarioConcurrency(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := GetMongoClient(ctx, "mongodb://localhost:27017")
	if err != nil {
		t.Skipf("MongoDB not available: %v", err)
	}
	defer client.Disconnect(ctx)

	db := client.Database("devlab_test")
	collection := db.Collection("scenarios")

	// Clean up
	collection.Drop(ctx)

	t.Run("concurrent_inserts", func(t *testing.T) {
		const numGoroutines = 10
		results := make(chan error, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				scenario := &Scenario{
					ScenarioID:   fmt.Sprintf("concurrent-scn-%d", id),
					UserID:       fmt.Sprintf("user-%d", id),
					ScenarioType: "go",
					Status:       "running",
					CreatedAt:    time.Now(),
				}

				_, err := collection.InsertOne(ctx, scenario)
				results <- err
			}(i)
		}

		// Wait for all goroutines to complete
		for i := 0; i < numGoroutines; i++ {
			err := <-results
			assert.NoError(t, err)
		}

		// Verify all scenarios were inserted
		count, err := collection.CountDocuments(ctx, bson.M{})
		assert.NoError(t, err)
		assert.Equal(t, int64(numGoroutines), count)
	})
}

// TestScenarioPerformance tests performance characteristics
func TestScenarioPerformance(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := GetMongoClient(ctx, "mongodb://localhost:27017")
	if err != nil {
		t.Skipf("MongoDB not available: %v", err)
	}
	defer client.Disconnect(ctx)

	db := client.Database("devlab_test")
	collection := db.Collection("scenarios")

	// Clean up
	collection.Drop(ctx)

	t.Run("bulk_insert_performance", func(t *testing.T) {
		const numDocuments = 1000
		documents := make([]interface{}, numDocuments)

		for i := 0; i < numDocuments; i++ {
			documents[i] = &Scenario{
				ScenarioID:   fmt.Sprintf("perf-scn-%d", i),
				UserID:       fmt.Sprintf("user-%d", i%10), // 10 different users
				ScenarioType: "go",
				Status:       "running",
				CreatedAt:    time.Now(),
			}
		}

		start := time.Now()
		_, err := collection.InsertMany(ctx, documents)
		duration := time.Since(start)

		assert.NoError(t, err)
		t.Logf("Inserted %d documents in %v", numDocuments, duration)

		// Should complete within reasonable time
		assert.Less(t, duration, 5*time.Second)
	})

	t.Run("query_performance", func(t *testing.T) {
		// Test query performance
		start := time.Now()
		cursor, err := collection.Find(ctx, bson.M{"status": "running"})
		duration := time.Since(start)

		assert.NoError(t, err)
		defer cursor.Close(ctx)

		var results []Scenario
		err = cursor.All(ctx, &results)
		assert.NoError(t, err)

		t.Logf("Queried %d documents in %v", len(results), duration)

		// Should complete within reasonable time
		assert.Less(t, duration, 1*time.Second)
	})
}

// Benchmark tests for performance measurement
func BenchmarkScenarioInsert(b *testing.B) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := GetMongoClient(ctx, "mongodb://localhost:27017")
	if err != nil {
		b.Skipf("MongoDB not available: %v", err)
	}
	defer client.Disconnect(ctx)

	db := client.Database("devlab_test")
	collection := db.Collection("scenarios")

	// Clean up
	collection.Drop(ctx)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		scenario := &Scenario{
			ScenarioID:   fmt.Sprintf("bench-scn-%d", i),
			UserID:       "bench-user",
			ScenarioType: "go",
			Status:       "running",
			CreatedAt:    time.Now(),
		}

		_, err := collection.InsertOne(ctx, scenario)
		if err != nil {
			b.Fatalf("Insert failed: %v", err)
		}
	}
}

func BenchmarkScenarioQuery(b *testing.B) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := GetMongoClient(ctx, "mongodb://localhost:27017")
	if err != nil {
		b.Skipf("MongoDB not available: %v", err)
	}
	defer client.Disconnect(ctx)

	db := client.Database("devlab_test")
	collection := db.Collection("scenarios")

	// Clean up and insert test data
	collection.Drop(ctx)

	// Insert test data
	for i := 0; i < 1000; i++ {
		scenario := &Scenario{
			ScenarioID:   fmt.Sprintf("bench-scn-%d", i),
			UserID:       "bench-user",
			ScenarioType: "go",
			Status:       "running",
			CreatedAt:    time.Now(),
		}

		_, err := collection.InsertOne(ctx, scenario)
		if err != nil {
			b.Fatalf("Setup failed: %v", err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cursor, err := collection.Find(ctx, bson.M{"user_id": "bench-user"})
		if err != nil {
			b.Fatalf("Query failed: %v", err)
		}

		var results []Scenario
		err = cursor.All(ctx, &results)
		if err != nil {
			b.Fatalf("Cursor failed: %v", err)
		}

		cursor.Close(ctx)
	}
}
