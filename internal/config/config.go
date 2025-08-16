package config

import (
	"os"
	"time"
)

type Config struct {
	MongoURI    string
	DBName      string
	DockerImage string
	Cleanup     CleanupConfig
}

type CleanupConfig struct {
	MaxScenarioAge  time.Duration
	CleanupInterval time.Duration
	EnableCleanup   bool
}

func Load() *Config {
	return &Config{
		MongoURI:    getEnv("MONGODB_URI", "mongodb://localhost:27017"),
		DBName:      getEnv("DB_NAME", "devlab"),
		DockerImage: getEnv("DOCKER_IMAGE", "golang:1.21"),
		Cleanup: CleanupConfig{
			MaxScenarioAge:  getDurationEnv("CLEANUP_MAX_SCENARIO_AGE", 24*time.Hour),
			CleanupInterval: getDurationEnv("CLEANUP_INTERVAL", 15*time.Minute),
			EnableCleanup:   getBoolEnv("CLEANUP_ENABLED", true),
		},
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getDurationEnv(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if duration, err := time.ParseDuration(v); err == nil {
			return duration
		}
	}
	return fallback
}

func getBoolEnv(key string, fallback bool) bool {
	if v := os.Getenv(key); v != "" {
		return v == "true" || v == "1" || v == "yes"
	}
	return fallback
}
