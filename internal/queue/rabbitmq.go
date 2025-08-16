package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

// QueueManager handles RabbitMQ operations
type QueueManager struct {
	conn    *amqp.Connection
	channel *amqp.Channel
}

// NewQueueManager creates a new queue manager
func NewQueueManager(url string) (*QueueManager, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	return &QueueManager{
		conn:    conn,
		channel: ch,
	}, nil
}

// Close closes the RabbitMQ connection
func (qm *QueueManager) Close() error {
	if qm.channel != nil {
		qm.channel.Close()
	}
	if qm.conn != nil {
		return qm.conn.Close()
	}
	return nil
}

// PublishMessage publishes a message to a queue
func (qm *QueueManager) PublishMessage(ctx context.Context, queueName string, message interface{}) error {
	body, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	err = qm.channel.PublishWithContext(ctx,
		"",        // exchange
		queueName, // routing key
		false,     // mandatory
		false,     // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		})

	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	log.Printf("[queue] published message to queue: %s", queueName)
	return nil
}

// ConsumeMessages consumes messages from a queue
func (qm *QueueManager) ConsumeMessages(ctx context.Context, queueName string, handler func([]byte) error) error {
	msgs, err := qm.channel.Consume(
		queueName, // queue
		"",        // consumer
		true,      // auto-ack
		false,     // exclusive
		false,     // no-local
		false,     // no-wait
		nil,       // args
	)
	if err != nil {
		return fmt.Errorf("failed to start consuming: %w", err)
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				log.Printf("[queue] stopping consumer for queue: %s", queueName)
				return
			case msg := <-msgs:
				if err := handler(msg.Body); err != nil {
					log.Printf("[queue] error handling message: %v", err)
				}
			}
		}
	}()

	log.Printf("[queue] started consuming from queue: %s", queueName)
	return nil
}

// DeclareQueue declares a queue if it doesn't exist
func (qm *QueueManager) DeclareQueue(queueName string) error {
	_, err := qm.channel.QueueDeclare(
		queueName, // name
		true,      // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare queue: %w", err)
	}

	log.Printf("[queue] declared queue: %s", queueName)
	return nil
}
