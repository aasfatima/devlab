package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestConfigLoading tests basic config loading
func TestConfigLoading(t *testing.T) {
	// Test default config loading
	cfg := Load()

	assert.NotNil(t, cfg)
	assert.NotEmpty(t, cfg.MongoURI)
	assert.NotEmpty(t, cfg.DBName)
	assert.NotEmpty(t, cfg.Cleanup.CleanupInterval)
	assert.NotEmpty(t, cfg.Cleanup.MaxScenarioAge)
}

// TestEnvironmentVariables tests environment variable overrides
func TestEnvironmentVariables(t *testing.T) {
	// Set test environment variables
	os.Setenv("MONGODB_URI", "mongodb://test-host:27017")
	os.Setenv("DB_NAME", "test_db")
	os.Setenv("CLEANUP_INTERVAL", "30s")
	os.Setenv("CLEANUP_MAX_SCENARIO_AGE", "2h")
	os.Setenv("ENABLE_CLEANUP", "true")

	defer func() {
		os.Unsetenv("MONGODB_URI")
		os.Unsetenv("DB_NAME")
		os.Unsetenv("CLEANUP_INTERVAL")
		os.Unsetenv("CLEANUP_MAX_SCENARIO_AGE")
		os.Unsetenv("CLEANUP_ENABLED")
	}()

	cfg := Load()

	assert.Equal(t, "mongodb://test-host:27017", cfg.MongoURI)
	assert.Equal(t, "test_db", cfg.DBName)
	assert.Equal(t, 30*time.Second, cfg.Cleanup.CleanupInterval)
	assert.Equal(t, 2*time.Hour, cfg.Cleanup.MaxScenarioAge)
	assert.True(t, cfg.Cleanup.EnableCleanup)
}

// TestConfigValidation tests config validation
func TestConfigValidation(t *testing.T) {
	// Test with invalid MongoDB URI
	os.Setenv("MONGODB_URI", "invalid-uri")
	defer os.Unsetenv("MONGODB_URI")

	cfg := Load()

	// Should still load but with default values
	assert.NotNil(t, cfg)
	assert.Equal(t, "invalid-uri", cfg.MongoURI)
}

// TestDefaultValues tests default configuration values
func TestDefaultValues(t *testing.T) {
	// Clear all environment variables
	os.Unsetenv("MONGODB_URI")
	os.Unsetenv("DB_NAME")
	os.Unsetenv("CLEANUP_INTERVAL")
	os.Unsetenv("CLEANUP_MAX_SCENARIO_AGE")
	os.Unsetenv("ENABLE_CLEANUP")

	cfg := Load()

	// Check default values
	assert.Equal(t, "mongodb://localhost:27017", cfg.MongoURI)
	assert.Equal(t, "devlab", cfg.DBName)
	assert.Equal(t, 15*time.Minute, cfg.Cleanup.CleanupInterval)
	assert.Equal(t, 24*time.Hour, cfg.Cleanup.MaxScenarioAge)
	assert.True(t, cfg.Cleanup.EnableCleanup)
}

// TestDockerImageConfig tests Docker image configuration
func TestDockerImageConfig(t *testing.T) {
	// Test default Docker image
	cfg := Load()

	assert.Equal(t, "golang:1.21", cfg.DockerImage)

	// Test custom Docker image
	os.Setenv("DOCKER_IMAGE", "golang:1.22")
	defer os.Unsetenv("DOCKER_IMAGE")

	cfg = Load()
	assert.Equal(t, "golang:1.22", cfg.DockerImage)
}

// TestCleanupConfig tests cleanup configuration
func TestCleanupConfig(t *testing.T) {
	// Test default cleanup settings
	cfg := Load()

	assert.True(t, cfg.Cleanup.EnableCleanup)
	assert.Equal(t, 15*time.Minute, cfg.Cleanup.CleanupInterval)
	assert.Equal(t, 24*time.Hour, cfg.Cleanup.MaxScenarioAge)

	// Test custom cleanup settings
	os.Setenv("CLEANUP_ENABLED", "false")
	os.Setenv("CLEANUP_INTERVAL", "30m")
	os.Setenv("CLEANUP_MAX_SCENARIO_AGE", "48h")
	defer func() {
		os.Unsetenv("CLEANUP_ENABLED")
		os.Unsetenv("CLEANUP_INTERVAL")
		os.Unsetenv("CLEANUP_MAX_SCENARIO_AGE")
	}()

	cfg = Load()
	assert.False(t, cfg.Cleanup.EnableCleanup)
	assert.Equal(t, 30*time.Minute, cfg.Cleanup.CleanupInterval)
	assert.Equal(t, 48*time.Hour, cfg.Cleanup.MaxScenarioAge)
}

// TestInvalidConfigValues tests handling of invalid config values
func TestInvalidConfigValues(t *testing.T) {
	// Test invalid duration values
	os.Setenv("CLEANUP_INTERVAL", "invalid-duration")
	os.Setenv("CLEANUP_MAX_SCENARIO_AGE", "invalid-age")
	defer func() {
		os.Unsetenv("CLEANUP_INTERVAL")
		os.Unsetenv("CLEANUP_MAX_SCENARIO_AGE")
	}()

	cfg := Load()

	// Should fall back to default values
	assert.Equal(t, 15*time.Minute, cfg.Cleanup.CleanupInterval)
	assert.Equal(t, 24*time.Hour, cfg.Cleanup.MaxScenarioAge)
}

// TestConfigReload tests config reloading
func TestConfigReload(t *testing.T) {
	// Initial config
	cfg1 := Load()

	// Change environment variable
	os.Setenv("DB_NAME", "new_db_name")
	defer os.Unsetenv("DB_NAME")

	// Reload config
	cfg2 := Load()

	assert.NotEqual(t, cfg1.DBName, cfg2.DBName)
	assert.Equal(t, "new_db_name", cfg2.DBName)
}

// TestConfigConsistency tests config consistency across loads
func TestConfigConsistency(t *testing.T) {
	// Load config multiple times
	cfg1 := Load()
	cfg2 := Load()
	cfg3 := Load()

	// All configs should be identical
	assert.Equal(t, cfg1.MongoURI, cfg2.MongoURI)
	assert.Equal(t, cfg2.MongoURI, cfg3.MongoURI)
	assert.Equal(t, cfg1.DBName, cfg2.DBName)
	assert.Equal(t, cfg2.DBName, cfg3.DBName)
	assert.Equal(t, cfg1.Cleanup.EnableCleanup, cfg2.Cleanup.EnableCleanup)
	assert.Equal(t, cfg2.Cleanup.EnableCleanup, cfg3.Cleanup.EnableCleanup)
}
