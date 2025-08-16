package storage

import (
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/bson"
	"errors"
	"time"
)

// Custom error types for storage operations
var (
	ErrScenarioNotFound = errors.New("scenario not found")
	ErrDatabaseNil      = errors.New("database is nil")
	ErrInvalidScenario  = errors.New("invalid scenario data")
)

type Scenario struct {
	ScenarioID   string    `bson:"scenario_id"`
	UserID       string    `bson:"user_id"`
	ScenarioType string    `bson:"scenario_type"`
	ContainerID  string    `bson:"container_id"`
	Status       string    `bson:"status"`
	TerminalPort int       `bson:"terminal_port,omitempty"`
	CreatedAt    time.Time `bson:"created_at,omitempty"`
	UpdatedAt    time.Time `bson:"updated_at,omitempty"`
}

func GetMongoClient(ctx context.Context, uri string) (*mongo.Client, error) {
	return mongo.Connect(ctx, options.Client().ApplyURI(uri))
}

func StoreScenario(ctx context.Context, db *mongo.Database, s *Scenario) error {
	if db == nil {
		return fmt.Errorf("%w", ErrDatabaseNil)
	}
	
	if s == nil {
		return fmt.Errorf("%w: scenario cannot be nil", ErrInvalidScenario)
	}
	
	if s.ScenarioID == "" {
		return fmt.Errorf("%w: scenario ID cannot be empty", ErrInvalidScenario)
	}
	
	_, err := db.Collection("scenarios").InsertOne(ctx, s)
	if err != nil {
		return fmt.Errorf("failed to store scenario: %w", err)
	}
	
	return nil
}

func GetScenario(ctx context.Context, db *mongo.Database, scenarioID string) (*Scenario, error) {
	if db == nil {
		return nil, fmt.Errorf("%w", ErrDatabaseNil)
	}
	
	if scenarioID == "" {
		return nil, fmt.Errorf("%w: scenario ID cannot be empty", ErrInvalidScenario)
	}
	
	var scenario Scenario
	err := db.Collection("scenarios").FindOne(ctx, bson.M{"scenario_id": scenarioID}).Decode(&scenario)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("%w: %s", ErrScenarioNotFound, scenarioID)
		}
		return nil, fmt.Errorf("failed to get scenario: %w", err)
	}
	
	return &scenario, nil
}

func UpdateScenario(ctx context.Context, db *mongo.Database, s *Scenario) error {
	if db == nil {
		return fmt.Errorf("%w", ErrDatabaseNil)
	}
	
	if s == nil {
		return fmt.Errorf("%w: scenario cannot be nil", ErrInvalidScenario)
	}
	
	if s.ScenarioID == "" {
		return fmt.Errorf("%w: scenario ID cannot be empty", ErrInvalidScenario)
	}
	
	// Update the scenario with current timestamp
	s.UpdatedAt = time.Now()
	
	_, err := db.Collection("scenarios").UpdateOne(
		ctx,
		bson.M{"scenario_id": s.ScenarioID},
		bson.M{"$set": s},
	)
	if err != nil {
		return fmt.Errorf("failed to update scenario: %w", err)
	}
	
	return nil
}

func DeleteScenario(ctx context.Context, db *mongo.Database, scenarioID string) error {
	if db == nil {
		return fmt.Errorf("%w", ErrDatabaseNil)
	}
	
	if scenarioID == "" {
		return fmt.Errorf("%w: scenario ID cannot be empty", ErrInvalidScenario)
	}
	
	_, err := db.Collection("scenarios").DeleteOne(ctx, bson.M{"scenario_id": scenarioID})
	if err != nil {
		return fmt.Errorf("failed to delete scenario: %w", err)
	}
	
	return nil
}

func ListScenarios(ctx context.Context, db *mongo.Database, userID string) ([]*Scenario, error) {
	if db == nil {
		return nil, fmt.Errorf("%w", ErrDatabaseNil)
	}
	
	filter := bson.M{}
	if userID != "" {
		filter["user_id"] = userID
	}
	
	cursor, err := db.Collection("scenarios").Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list scenarios: %w", err)
	}
	defer cursor.Close(ctx)
	
	var scenarios []*Scenario
	if err = cursor.All(ctx, &scenarios); err != nil {
		return nil, fmt.Errorf("failed to decode scenarios: %w", err)
	}
	
	return scenarios, nil
}
