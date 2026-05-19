package metrics

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

const TopicMetricEvents = "prism.metric.events"

// Publisher publishes metric events to Kafka.
type Publisher struct {
	producer *kafka.Producer
}

// NewPublisher creates a new Publisher connected to the given Kafka broker.
func NewPublisher(brokerAddr string) (*Publisher, error) {
	producer, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": brokerAddr,
	})
	if err != nil {
		return nil, fmt.Errorf("creating kafka producer: %w", err)
	}
	return &Publisher{producer: producer}, nil
}

// Publish sends a MetricEvent to Kafka.
// The event is keyed by experiment_id so all events for the same experiment
// land in the same partition, preserving the order
func (p *Publisher) Publish(ctx context.Context, event MetricEvent) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshalling event: %w", err)
	}

	key := []byte(event.ExperimentID)
	topic := TopicMetricEvents

	deliveryChan := make(chan kafka.Event, 1)
	err = p.producer.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{
			Topic:     &topic,
			Partition: kafka.PartitionAny,
		},
		Key:   key,
		Value: payload,
	}, deliveryChan)
	if err != nil {
		return fmt.Errorf("producing message: %w", err)
	}

	// Wait for delivery confirmation or error.
	e := <-deliveryChan
	msg, ok := e.(*kafka.Message)
	if !ok {
		return fmt.Errorf("unexpected delivery event type")
	}
	if msg.TopicPartition.Error != nil {
		return fmt.Errorf("delivering message: %w", msg.TopicPartition.Error)
	}

	return nil
}

// Close flushes any pending messages and shuts down the producer.
func (p *Publisher) Close() {
	p.producer.Flush(5000)
	p.producer.Close()
}
