package cleanup

import (
	"context"
	"devlab/internal/config"
	"devlab/internal/docker"
	"devlab/internal/storage"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// CleanupManager handles cleanup operations for scenarios
type CleanupManager struct {
	cfg    *config.Config
	db     *mongo.Database
	docker docker.Client
}

// NewCleanupManager creates a new cleanup manager
func NewCleanupManager(cfg *config.Config, db *mongo.Database, dockerClient docker.Client) *CleanupManager {
	return &CleanupManager{
		cfg:    cfg,
		db:     db,
		docker: dockerClient,
	}
}

// CleanupExpiredScenarios removes scenarios that have exceeded their lifetime
func (cm *CleanupManager) CleanupExpiredScenarios(ctx context.Context) error {
	log.Println("[cleanup] starting expired scenario cleanup")

	// Get cleanup configuration
	maxAge := cm.cfg.Cleanup.MaxScenarioAge
	if maxAge == 0 {
		maxAge = 24 * time.Hour // Default to 24 hours
	}

	// Find expired scenarios
	expiredScenarios, err := cm.findExpiredScenarios(ctx, maxAge)
	if err != nil {
		return fmt.Errorf("failed to find expired scenarios: %w", err)
	}

	log.Printf("[cleanup] found %d expired scenarios", len(expiredScenarios))

	// Clean up each expired scenario
	for _, scenario := range expiredScenarios {
		if err := cm.cleanupScenario(ctx, scenario); err != nil {
			log.Printf("[cleanup] failed to cleanup scenario %s: %v", scenario.ScenarioID, err)
			continue
		}
		log.Printf("[cleanup] successfully cleaned up scenario %s", scenario.ScenarioID)
	}

	return nil
}

// CleanupOrphanedContainers removes containers that are not associated with any scenario
func (cm *CleanupManager) CleanupOrphanedContainers(ctx context.Context) error {
	log.Println("[cleanup] starting orphaned container cleanup")

	// Get all running containers
	containers, err := cm.docker.ListContainers(ctx)
	if err != nil {
		return fmt.Errorf("failed to list containers: %w", err)
	}

	// Get all scenario container IDs from database
	scenarioContainers, err := cm.getScenarioContainerIDs(ctx)
	if err != nil {
		return fmt.Errorf("failed to get scenario container IDs: %w", err)
	}

	// Find orphaned containers
	var orphanedCount int
	for _, container := range containers {
		if !cm.isScenarioContainer(container.ID, scenarioContainers) {
			log.Printf("[cleanup] found orphaned container: %s", container.ID)

			// Stop and remove the orphaned container
			if err := cm.docker.StopContainer(ctx, container.ID); err != nil {
				log.Printf("[cleanup] failed to stop orphaned container %s: %v", container.ID, err)
				continue
			}

			if err := cm.docker.RemoveContainer(ctx, container.ID); err != nil {
				log.Printf("[cleanup] failed to remove orphaned container %s: %v", container.ID, err)
				continue
			}

			orphanedCount++
			log.Printf("[cleanup] successfully cleaned up orphaned container %s", container.ID)
		}
	}

	log.Printf("[cleanup] cleaned up %d orphaned containers", orphanedCount)
	return nil
}

// RunPeriodicCleanup runs cleanup operations periodically
func (cm *CleanupManager) RunPeriodicCleanup(ctx context.Context, interval time.Duration) {
	log.Printf("[cleanup] starting periodic cleanup with interval: %v", interval)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("[cleanup] stopping periodic cleanup")
			return
		case <-ticker.C:
			log.Println("[cleanup] running cleanup cycle")

			if err := cm.CleanupExpiredScenarios(ctx); err != nil {
				log.Printf("[cleanup] error cleaning up expired scenarios: %v", err)
			}

			if err := cm.CleanupOrphanedContainers(ctx); err != nil {
				log.Printf("[cleanup] error cleaning up orphaned containers: %v", err)
			}
		}
	}
}

// findExpiredScenarios finds scenarios that have exceeded the maximum age
func (cm *CleanupManager) findExpiredScenarios(ctx context.Context, maxAge time.Duration) ([]*storage.Scenario, error) {
	cutoffTime := time.Now().Add(-maxAge)

	filter := bson.M{
		"created_at": bson.M{"$lt": cutoffTime},
		"status":     bson.M{"$in": []string{"running", "provisioning"}},
	}

	cursor, err := cm.db.Collection("scenarios").Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to query expired scenarios: %w", err)
	}
	defer cursor.Close(ctx)

	var scenarios []*storage.Scenario
	if err = cursor.All(ctx, &scenarios); err != nil {
		return nil, fmt.Errorf("failed to decode expired scenarios: %w", err)
	}

	return scenarios, nil
}

// cleanupScenario stops and removes a scenario and its container
func (cm *CleanupManager) cleanupScenario(ctx context.Context, scenario *storage.Scenario) error {
	log.Printf("[cleanup] cleaning up scenario %s (container: %s)", scenario.ScenarioID, scenario.ContainerID)

	// Stop the container if it exists and is running
	if scenario.ContainerID != "" {
		containerExists, err := cm.docker.ContainerExists(ctx, scenario.ContainerID)
		if err != nil {
			log.Printf("[cleanup] failed to check container existence for %s: %v", scenario.ContainerID, err)
		} else if containerExists {
			// Get container status
			status, err := cm.docker.GetContainerStatus(ctx, scenario.ContainerID)
			if err != nil {
				log.Printf("[cleanup] failed to get container status for %s: %v", scenario.ContainerID, err)
			} else if status == "running" {
				// Stop the container
				if err := cm.docker.StopContainer(ctx, scenario.ContainerID); err != nil {
					log.Printf("[cleanup] failed to stop container %s: %v", scenario.ContainerID, err)
				}
			}

			// Remove the container
			if err := cm.docker.RemoveContainer(ctx, scenario.ContainerID); err != nil {
				log.Printf("[cleanup] failed to remove container %s: %v", scenario.ContainerID, err)
			}
		}
	}

	// Update scenario status to cleaned up
	scenario.Status = "cleaned_up"
	scenario.UpdatedAt = time.Now()

	if err := storage.UpdateScenario(ctx, cm.db, scenario); err != nil {
		return fmt.Errorf("failed to update scenario status: %w", err)
	}

	return nil
}

// getScenarioContainerIDs gets all container IDs associated with scenarios
func (cm *CleanupManager) getScenarioContainerIDs(ctx context.Context) (map[string]bool, error) {
	filter := bson.M{"container_id": bson.M{"$ne": ""}}

	cursor, err := cm.db.Collection("scenarios").Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to query scenario container IDs: %w", err)
	}
	defer cursor.Close(ctx)

	containerIDs := make(map[string]bool)
	for cursor.Next(ctx) {
		var scenario storage.Scenario
		if err := cursor.Decode(&scenario); err != nil {
			log.Printf("[cleanup] failed to decode scenario: %v", err)
			continue
		}
		if scenario.ContainerID != "" {
			containerIDs[scenario.ContainerID] = true
		}
	}

	return containerIDs, nil
}

// isScenarioContainer checks if a container ID is associated with a scenario
func (cm *CleanupManager) isScenarioContainer(containerID string, scenarioContainers map[string]bool) bool {
	return scenarioContainers[containerID]
}
