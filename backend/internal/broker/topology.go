// Package broker defines the RabbitMQ topology used for feed delivery.
package broker

import (
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	JobsExchange          = "feed.jobs"
	JobsQueue             = "feed.jobs"
	PostCreatedRoutingKey = "post.created"
	NotificationsExchange = "feed.notifications"
)

// Initialize verifies the connection and declares the shared topology.
func Initialize(url string) error {
	conn, err := amqp.Dial(url)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer conn.Close()
	ch, err := conn.Channel()
	if err != nil {
		return fmt.Errorf("open channel: %w", err)
	}
	defer ch.Close()
	return declareTopology(ch)
}

// declareTopology creates the durable shared infrastructure. WebSocket queues
// and their user-specific bindings are created by each WebSocket node.
func declareTopology(ch *amqp.Channel) error {
	if err := ch.ExchangeDeclare(JobsExchange, "direct", true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare jobs exchange: %w", err)
	}
	if _, err := ch.QueueDeclare(JobsQueue, true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare jobs queue: %w", err)
	}
	if err := ch.QueueBind(JobsQueue, PostCreatedRoutingKey, JobsExchange, false, nil); err != nil {
		return fmt.Errorf("bind jobs queue: %w", err)
	}
	if err := ch.ExchangeDeclare(NotificationsExchange, "direct", true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare notifications exchange: %w", err)
	}
	return nil
}
