// MongoDB initialization script for DevLab
// This script runs when the MongoDB container starts for the first time

// Switch to devlab database
db = db.getSiblingDB('devlab');

// Create scenarios collection with indexes
db.createCollection('scenarios');

// Create indexes for better query performance
db.scenarios.createIndex({ "scenario_id": 1 }, { unique: true });
db.scenarios.createIndex({ "user_id": 1 });
db.scenarios.createIndex({ "status": 1 });
db.scenarios.createIndex({ "scenario_type": 1 });
db.scenarios.createIndex({ "created_at": 1 });
db.scenarios.createIndex({ "container_id": 1 });

// Create compound indexes for common queries
db.scenarios.createIndex({ "user_id": 1, "status": 1 });
db.scenarios.createIndex({ "user_id": 1, "created_at": 1 });
db.scenarios.createIndex({ "status": 1, "created_at": 1 });

// Create TTL index for automatic cleanup of old scenarios (optional)
// This will automatically delete scenarios older than 30 days
db.scenarios.createIndex({ "created_at": 1 }, { expireAfterSeconds: 2592000 });

// Create users collection (for future use)
db.createCollection('users');
db.users.createIndex({ "user_id": 1 }, { unique: true });

// Create logs collection (for future use)
db.createCollection('logs');
db.logs.createIndex({ "timestamp": 1 });
db.logs.createIndex({ "level": 1 });
db.logs.createIndex({ "service": 1 });

// Insert some sample data for testing (optional)
db.scenarios.insertOne({
    scenario_id: "sample-scn-001",
    user_id: "demo-user",
    scenario_type: "go",
    container_id: "sample-container-001",
    status: "running",
    terminal_port: 3001,
    created_at: new Date(),
    updated_at: new Date()
});

print("DevLab database initialized successfully!");
print("Collections created: scenarios, users, logs");
print("Indexes created for optimal query performance"); 