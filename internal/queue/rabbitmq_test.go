package queue

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRabbitMQConnection tests basic connection to RabbitMQ
func TestRabbitMQConnection(t *testing.T) {
	// Skip if RabbitMQ is not available
	if testing.Short() {
		t.Skip("Skipping RabbitMQ tests in short mode")
	}

	manager, err := NewQueueManager("amqp://guest:guest@localhost:5672/")
	require.NoError(t, err, "Should connect to RabbitMQ successfully")
	defer manager.Close()

	assert.NotNil(t, manager, "Manager should be created")
}

// TestRabbitMQPublish tests message publishing
func TestRabbitMQPublish(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping RabbitMQ tests in short mode")
	}

	manager, err := NewQueueManager("amqp://guest:guest@localhost:5672/")
	require.NoError(t, err)
	defer manager.Close()

	// Declare test queue
	queueName := "test-queue"
	err = manager.DeclareQueue(queueName)
	require.NoError(t, err)

	// Publish test message
	message := map[string]string{"test": "message"}
	err = manager.PublishMessage(context.Background(), queueName, message)
	require.NoError(t, err, "Should publish message successfully")
}

// TestRabbitMQConsume tests message consumption
func TestRabbitMQConsume(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping RabbitMQ tests in short mode")
	}

	manager, err := NewQueueManager("amqp://guest:guest@localhost:5672/")
	require.NoError(t, err)
	defer manager.Close()

	// Declare test queue
	queueName := "test-consume-queue"
	err = manager.DeclareQueue(queueName)
	require.NoError(t, err)

	// Publish test message
	testMessage := map[string]string{"test": "consume message"}
	err = manager.PublishMessage(context.Background(), queueName, testMessage)
	require.NoError(t, err)

	// Consume message
	receivedMessages := make(chan []byte, 1)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = manager.ConsumeMessages(ctx, queueName, func(msg []byte) error {
		receivedMessages <- msg
		return nil
	})
	require.NoError(t, err)

	// Wait for message
	select {
	case msg := <-receivedMessages:
		assert.NotEmpty(t, msg, "Should receive a message")
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for message")
	}
}

// TestRabbitMQConnectionFailure tests connection failure handling
func TestRabbitMQConnectionFailure(t *testing.T) {
	_, err := NewQueueManager("amqp://invalid:invalid@localhost:5673/")
	assert.Error(t, err, "Should fail to connect to invalid RabbitMQ")
}

// TestRabbitMQQueueDeclaration tests queue declaration
func TestRabbitMQQueueDeclaration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping RabbitMQ tests in short mode")
	}

	manager, err := NewQueueManager("amqp://guest:guest@localhost:5672/")
	require.NoError(t, err)
	defer manager.Close()

	// Test queue declaration
	queueName := "test-declare-queue"
	err = manager.DeclareQueue(queueName)
	require.NoError(t, err, "Should declare queue successfully")

	// Test declaring the same queue again (should not error)
	err = manager.DeclareQueue(queueName)
	require.NoError(t, err, "Should handle duplicate queue declaration")
}

// TestRabbitMQMessageDelivery tests reliable message delivery
func TestRabbitMQMessageDelivery(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping RabbitMQ tests in short mode")
	}

	manager, err := NewQueueManager("amqp://guest:guest@localhost:5672/")
	require.NoError(t, err)
	defer manager.Close()

	queueName := "test-delivery-queue"
	err = manager.DeclareQueue(queueName)
	require.NoError(t, err)

	// Publish multiple messages
	messages := []map[string]string{
		{"id": "1", "content": "message1"},
		{"id": "2", "content": "message2"},
		{"id": "3", "content": "message3"},
	}

	for _, msg := range messages {
		err = manager.PublishMessage(context.Background(), queueName, msg)
		require.NoError(t, err)
	}

	// Consume messages
	receivedMessages := make(chan []byte, len(messages))
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = manager.ConsumeMessages(ctx, queueName, func(msg []byte) error {
		receivedMessages <- msg
		return nil
	})
	require.NoError(t, err)

	// Wait for messages
	received := make([][]byte, 0)
	for i := 0; i < len(messages); i++ {
		select {
		case msg := <-receivedMessages:
			received = append(received, msg)
		case <-time.After(5 * time.Second):
			t.Fatal("Timeout waiting for messages")
		}
	}

	assert.Len(t, received, len(messages), "Should receive all published messages")
}
